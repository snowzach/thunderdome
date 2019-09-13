package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// GetLedger returns the ledger for a user
func (c *Client) GetLedger(ctx context.Context, filter map[string]string, after time.Time, offset int, limit int) ([]*tdrpc.LedgerRecord, error) {

	var queryClause string
	var queryParams = []interface{}{}

	// Validate the filters
	for filter, value := range filter {
		switch filter {
		case "account_id":
			if value == "" {
				return nil, fmt.Errorf("Invalid value for account_id")
			}
			queryParams = append(queryParams, value)
			queryClause += fmt.Sprintf(" AND account_id = $%d", len(queryParams))
		case "status":
			// Validate
			if err := new(tdrpc.LedgerRecord_Status).Scan(value); err != nil {
				return nil, err
			}
			queryParams = append(queryParams, value)
			queryClause += fmt.Sprintf(" AND status = $%d", len(queryParams))
		case "type":
			// Validate
			if err := new(tdrpc.LedgerRecord_Type).Scan(value); err != nil {
				return nil, err
			}
			queryParams = append(queryParams, value)
			queryClause += fmt.Sprintf(" AND type = $%d", len(queryParams))
		case "direction":
			// Validate
			if err := new(tdrpc.LedgerRecord_Direction).Scan(value); err != nil {
				return nil, err
			}
			queryParams = append(queryParams, value)
			queryClause += fmt.Sprintf(" AND direction = $%d", len(queryParams))
		case "generated":
			value = strings.ToLower(value)
			if value == "true" {
				queryParams = append(queryParams, true)
			} else if value == "false" {
				queryParams = append(queryParams, false)
			} else {
				return nil, fmt.Errorf("Invalid value for generated")
			}
			queryClause += fmt.Sprintf(" AND generated = $%d", len(queryParams))
		case "request":
			if value == "" {
				return nil, fmt.Errorf("Invalid value for request")
			}
			queryParams = append(queryParams, value)
			queryClause += fmt.Sprintf(" AND request = $%d", len(queryParams))
		case "hidden":
			value = strings.ToLower(value)
			if value == "true" {
				queryParams = append(queryParams, true)
			} else if value == "false" {
				queryParams = append(queryParams, false)
			} else if value == "*" {
				// Don't filter at all
				break
			} else {
				return nil, fmt.Errorf("Invalid value for hidden")
			}
			queryClause += fmt.Sprintf(" AND hidden = $%d", len(queryParams))
		default:
			return nil, fmt.Errorf("Unsupported filter %s", filter)

		}
	}

	// Handle the after field
	if !after.IsZero() {
		queryParams = append(queryParams, after)
		queryClause += fmt.Sprintf(" AND created_at >= $%d", len(queryParams))
	}

	// Order By
	queryClause += " ORDER BY created_at DESC"

	if limit > 0 {
		queryClause += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		queryClause += fmt.Sprintf(" OFFSET %d", offset)
	}

	var lrs = make([]*tdrpc.LedgerRecord, 0)
	err := c.db.SelectContext(ctx, &lrs, `SELECT * FROM ledger WHERE 1=1`+queryClause, queryParams...)
	if err != nil {
		return lrs, err
	}
	return lrs, nil

}

// GetLedgerRecord returns the LedgerRecord
func (c *Client) GetLedgerRecord(ctx context.Context, id string, direction tdrpc.LedgerRecord_Direction) (*tdrpc.LedgerRecord, error) {

	var lr = new(tdrpc.LedgerRecord)
	err := c.db.GetContext(ctx, lr, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, id, direction)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return lr, nil

}

// GetActiveGeneratedLightningLedgerRequest returns a lightning request that is not paid, active and with the generic flag
func (c *Client) GetActiveGeneratedLightningLedgerRequest(ctx context.Context, accountID string) (*tdrpc.LedgerRecord, error) {

	var lr = new(tdrpc.LedgerRecord)
	// Find the newest pending ledger record for lightning inbound where generated = true that will expire in more than an hour
	// But that also does not have an internal payment made to it (in case someone pays it externally)
	err := c.db.GetContext(ctx, lr, `
		SELECT * FROM ledger WHERE
		account_id = $1 AND
		status = $2 AND
		type = $3 AND
		direction = $4 AND
		generated = true AND
		expires_at > NOW() + INTERVAL '1 HOUR' AND
		NOT EXISTS (SELECT 1 FROM ledger AS li WHERE li.id = CONCAT(ledger.id, '`+tdrpc.InternalIdSuffix+`'))
		ORDER BY expires_at DESC
		LIMIT 1
	`, accountID, tdrpc.PENDING, tdrpc.LIGHTNING, tdrpc.IN)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return lr, nil

}

// UpdateLedgerRecordID returns the LedgerRecord
func (c *Client) UpdateLedgerRecordID(ctx context.Context, oldID string, newID string) error {

	for retries := 5; retries > 0; retries-- {

		// Start a transaction
		tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
		})
		if err != nil {
			return fmt.Errorf("Could not start transaction: %v", err)
		}

		// If we panic, roll the transaction back
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback()
				c.logger.Panic(string(debug.Stack()))

			}
		}()

		err = c.updateLedgerRecordID(ctx, tx, oldID, newID)
		if err != nil {
			_ = tx.Rollback()
			if IsTransactionError(err) {
				c.logger.Warnf("UpdateLedgerRecordID TX Fail: %v - Retries Left %d", err, retries)
				time.Sleep(time.Duration(retries) * time.Second)
				continue
			}
			return err
		}

		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			if IsTransactionError(err) {
				c.logger.Warnf("UpdateLedgerRecordID TX Fail: %v - Retries Left %d", err, retries)
				time.Sleep(time.Duration(retries) * time.Second)
				continue
			}
			return fmt.Errorf("Commit Error: %v", err)
		}

		return nil

	}

	return fmt.Errorf("Transaction failed, out of retries")

}

func (c *Client) updateLedgerRecordID(ctx context.Context, tx *sqlx.Tx, oldID string, newID string) error {

	var temp int

	// Make sure the ID exists
	err := tx.GetContext(ctx, &temp, `SELECT 1 FROM ledger WHERE id = $1 LIMIT 1`, oldID)
	if err == sql.ErrNoRows {
		return store.ErrNotFound
	} else if err != nil {
		return err
	}

	// Make sure the newID doesn't exist
	err = tx.GetContext(ctx, &temp, `SELECT 1 FROM ledger WHERE id = $1 LIMIT 1`, newID)
	if err == nil {
		return fmt.Errorf("Cannot rename record. Record already exists.")
	} else if err == sql.ErrNoRows {
		// We're good
	} else {
		return err
	}

	// Rename the records
	_, err = tx.ExecContext(ctx, `UPDATE ledger SET id = $1 WHERE id = $2`, newID, oldID)
	if err != nil {
		return err
	}

	return nil

}

// ExpireLedgerRequests finds any LedgerRequests that have expired and expires them
func (c *Client) ExpireLedgerRequests(ctx context.Context) error {

	var lrs = make([]*tdrpc.LedgerRecord, 0)
	err := c.db.SelectContext(ctx, &lrs, `SELECT * FROM ledger WHERE status = $1 AND expires_at < NOW()`, tdrpc.PENDING)
	if err != nil {
		return err
	}

	var lastError error

	for _, lr := range lrs {
		lr.Status = tdrpc.EXPIRED

		c.logger.Infow("Expiring LedgerRequest", "lr", lr)

		err := c.ProcessLedgerRecord(ctx, lr)
		if err != nil {
			lastError = err
			c.logger.Errorw("Could not expire LedgerRecord", "error", err)
			continue
		}
	}

	return lastError

}

// LedgerDeltaLog logs the difference between 2 ledger records
func (c *Client) LedgerDeltaLog(lr1, lr2 *tdrpc.LedgerRecord) {

	ret := make([]interface{}, 0)

	if lr1 == nil || lr2 == nil {
		return
	}

	if lr1.AccountId != lr2.AccountId {
		ret = append(ret, "account_id", fmt.Sprintf("%v->%v", lr1.AccountId, lr2.AccountId))
	}
	if lr1.Status != lr2.Status {
		ret = append(ret, "status", fmt.Sprintf("%v->%v", lr1.Status, lr2.Status))
	}
	if lr1.Type != lr2.Type {
		ret = append(ret, "type", fmt.Sprintf("%v->%v", lr1.Type, lr2.Type))
	}
	if lr1.Generated != lr2.Generated {
		ret = append(ret, "generated", fmt.Sprintf("%v->%v", lr1.Generated, lr2.Generated))
	}
	if lr1.Direction != lr2.Direction {
		ret = append(ret, "direction", fmt.Sprintf("%v->%v", lr1.Direction, lr2.Direction))
	}
	if lr1.ValueTotal() != lr2.ValueTotal() {
		ret = append(ret, "value", fmt.Sprintf("%v->%v", lr1.Value, lr2.Value))
	}
	if lr1.NetworkFee != lr2.NetworkFee {
		ret = append(ret, "value", fmt.Sprintf("%v->%v", lr1.NetworkFee, lr2.NetworkFee))
	}
	if lr1.ProcessingFee != lr2.ProcessingFee {
		ret = append(ret, "value", fmt.Sprintf("%v->%v", lr1.ProcessingFee, lr2.ProcessingFee))
	}
	if lr1.AddIndex != lr2.AddIndex {
		ret = append(ret, "add_index", fmt.Sprintf("%v->%v", lr1.AddIndex, lr2.AddIndex))
	}
	if lr1.Memo != lr2.Memo {
		ret = append(ret, "memo", fmt.Sprintf("%v->%v", lr1.Memo, lr2.Memo))
	}
	if lr1.Request != lr2.Request {
		ret = append(ret, "request", fmt.Sprintf("%v->%v", lr1.Request, lr2.Request))
	}
	if lr1.Error != lr2.Error {
		ret = append(ret, "error", fmt.Sprintf("%v->%v", lr1.Error, lr2.Error))
	}

	if len(ret) > 0 {
		c.logger.Debugw("ProcessLedgerRecord Delta", ret...)
	}

}
