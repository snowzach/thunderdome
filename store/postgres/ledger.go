package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/jmoiron/sqlx"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

// ProcessLedgerRecord handles any balance transfer and changes to the ledger based on the status of the LedgerRecord
func (c *Client) ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error {

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

		err = c.processLedgerRecord(ctx, tx, lr)
		if err != nil {
			_ = tx.Rollback()
			if IsTransactionError(err) {
				c.logger.Warnf("TX Fail: %v - Retries Left %d", err, retries)
				continue
			}
			return err
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("Commit Error: %v", err)
		}

		return nil
	}

	return fmt.Errorf("Transaction failed, out of retries")

}

// This does the actual work but it allows you to nest it inside of another transaction
func (c *Client) processLedgerRecord(ctx context.Context, tx *sqlx.Tx, lr *tdrpc.LedgerRecord) error {

	// See if the ledger entry already exists
	prevlr := new(tdrpc.LedgerRecord)
	err := tx.GetContext(ctx, prevlr, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, lr.Id, lr.Direction)
	if err == sql.ErrNoRows || (err == nil && prevlr.Status == tdrpc.FAILED) {
		// There is no previous record or the status is failed.
		// We don't need to consider anything with the previous record
		prevlr = nil
	} else if err != nil {
		return fmt.Errorf("Could not fetch existing LedgerRecord: %v", err)
	}

	if prevlr != nil {
		c.logger.Debugw("ProcessLedgerRecord Delta", getLedgerDeltaLog(prevlr, lr)...)

		// The previous transaction exists, validate that nothing of note has changed in the request
		if prevlr.Type != lr.Type || prevlr.Direction != lr.Direction || prevlr.AccountId != lr.AccountId || prevlr.Request != lr.Request {
			return fmt.Errorf("Existing Ledger Entry Mismatch %v:%v", prevlr, lr)
		}

		// Invalid status transitions
		if ((prevlr.Status == tdrpc.EXPIRED || prevlr.Status == tdrpc.COMPLETED) && (lr.Status == tdrpc.PENDING || lr.Status == tdrpc.FAILED)) ||
			// Completed is a final state always
			(prevlr.Status == tdrpc.COMPLETED && lr.Status != tdrpc.COMPLETED) {
			if prevlr.Status == tdrpc.COMPLETED {
				return fmt.Errorf("already processed/paid")
			}
			return fmt.Errorf("Invalid Status Transition %v->%v", prevlr.Status, lr.Status)
		}

		// If the status hasn't changed
		if prevlr.Status == lr.Status {

			// Update only the fields we are allowed to update
			_, err = tx.ExecContext(ctx, `
				UPDATE ledger SET
				updated_at = NOW(),
				expires_at = $1,
				memo = $2,
				request = $3,
				error = $4
				WHERE id = $5 AND direction = $6
			`, lr.ExpiresAt, lr.Memo, lr.Request, lr.Error, lr.Id, lr.Direction)
			if err != nil {
				return err
			}

			return nil
		}
	}

	// Handle outbound requests
	if lr.Direction == tdrpc.OUT {

		// There is a previous LedgerRecord and the status has changed
		if prevlr != nil {

			// It was previously pending, pull the reserved funds from pending_out
			if prevlr.Status == tdrpc.PENDING {
				_, err = tx.ExecContext(ctx, `UPDATE account SET pending_out = pending_out - $1 WHERE id = $2`, prevlr.ValueTotal(), prevlr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process out existing pending pending_out: %v", err)
				}
			}

			// It failed or expired - put the money back in balance
			if lr.Status == tdrpc.EXPIRED || lr.Status == tdrpc.FAILED {
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, prevlr.ValueTotal(), prevlr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process out existing failed/expired balance: %v", err)
				}
			} else if lr.Status == tdrpc.COMPLETED && prevlr.ValueTotal() != lr.ValueTotal() {
				// If for some reason the settled balance was different from the pending balance, adjust to the completed value
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 - $2 WHERE id = $3`, prevlr.ValueTotal(), lr.ValueTotal(), prevlr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process out existing failed/expired balance: %v", err)
				}
			}

		} else { // No previous record/status

			// Get the current balance
			var balance int64
			err = tx.GetContext(ctx, &balance, `SELECT balance FROM account WHERE id = $1`, lr.AccountId)
			if err != nil {
				return fmt.Errorf("Could not get out new balance: %v", err)
			}

			// We've started a new transaction
			if lr.Status == tdrpc.PENDING {

				// Check to make sure we have enough funds to start this transaction
				if balance < lr.ValueTotal() {
					return store.ErrInsufficientFunds
				}

				// Put it in pending_out
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance - $1, pending_out = pending_out + $1 WHERE id = $2`, lr.ValueTotal(), lr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process out new pending pending_out: %v", err)
				}

				// The transaction is completed
			} else if lr.Status == tdrpc.COMPLETED {

				// Check to make sure we have enough funds - this should never really happen unless somehow made out of band
				if balance < lr.ValueTotal() {
					return store.ErrInsufficientFunds
				}

				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance - $1 WHERE id = $2`, lr.ValueTotal(), lr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process out new completed balance: %v", err)
				}
			}
		}

	} else if lr.Direction == tdrpc.IN {

		// There is a previous LedgerRecord
		if prevlr != nil {

			// It was previously pending, pull the reserved funds from pending_in
			if prevlr.Status == tdrpc.PENDING {
				_, err = tx.ExecContext(ctx, `UPDATE account SET pending_in = pending_in - $1 WHERE id = $2`, prevlr.ValueTotal(), prevlr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process in new pending balance: %v", err)
				}
			}
		}

		// Pending incoming transactions
		if lr.Status == tdrpc.PENDING {
			_, err = tx.ExecContext(ctx, `UPDATE account SET pending_in = pending_in + $1 WHERE id = $2`, lr.ValueTotal(), lr.AccountId)
			if err != nil {
				return fmt.Errorf("Could not process in new completed balance: %v", err)
			}

			// It completed, put the value into the balance
		} else if lr.Status == tdrpc.COMPLETED {
			_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, lr.ValueTotal(), lr.AccountId)
			if err != nil {
				return fmt.Errorf("Could not process in new completed balance: %v", err)
			}
		}

	} else {
		// Not possible
		return fmt.Errorf("Unknown direction: %v", lr.Direction)
	}

	// Upsert the data, capture the result
	var ret tdrpc.LedgerRecord
	err = tx.GetContext(ctx, &ret, `
		INSERT INTO ledger (id, account_id, created_at, updated_at, expires_at, status, type, direction, generated, value, add_index, memo, request, error)
		VALUES($1, $2, NOW(), NOW(), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id, direction) DO UPDATE
		SET
		updated_at = NOW(),
		expires_at = $3,
		status = $4,
		value = $8,
		memo = $10,
		request = $11,
		error = $12
		RETURNING *
	`, lr.Id, lr.AccountId, lr.ExpiresAt, lr.Status, lr.Type, lr.Direction, lr.Generated, lr.ValueTotal(), lr.AddIndex, lr.Memo, lr.Request, lr.Error)
	if err != nil {
		return fmt.Errorf("Could not process ledger: %v", err)
	}

	// Replace the existing value with the modified one
	*lr = ret

	return nil

}

// ProcessInternal will process an payment between two accounts on this system
func (c *Client) ProcessInternal(ctx context.Context, id string) (*tdrpc.LedgerRecord, error) {

	for retries := 5; retries > 0; retries-- {

		// Start a transaction
		tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
		})
		if err != nil {
			return nil, fmt.Errorf("Could not start transaction: %v", err)
		}

		// If we panic, roll the transaction back
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback()
				c.logger.Panic(string(debug.Stack()))

			}
		}()

		lr, err := c.processInternal(ctx, tx, id)
		if err != nil {
			_ = tx.Rollback()
			if IsTransactionError(err) {
				c.logger.Warnf("TX Fail: %v - Retries Left %d", err, retries)
				continue
			}
			return nil, err
		}

		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("Commit Error: %v", err)
		}

		return lr, nil

	}

	return nil, fmt.Errorf("Transaction failed, out of retries")

}

func (c *Client) processInternal(ctx context.Context, tx *sqlx.Tx, id string) (*tdrpc.LedgerRecord, error) {

	var sender, receiver tdrpc.LedgerRecord
	internalID := id + thunderdome.InternalIdSuffix

	// Get the receiver record
	err := tx.GetContext(ctx, &receiver, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, id, tdrpc.IN)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find receiver request")
	} else if err != nil {
		return nil, err
	}

	// Get the sender record with the internal suffix that was added by the pay endpoint
	err = tx.GetContext(ctx, &sender, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, internalID, tdrpc.OUT)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find sender request")
	} else if err != nil {
		return nil, err
	}

	// Make sure the status is right
	if sender.Status != tdrpc.PENDING || receiver.Status != tdrpc.PENDING {
		return nil, fmt.Errorf("Invalid status sender:%s receiver:%s", sender.Status, receiver.Status)
	}

	// Make sure the status is right
	if sender.Type != tdrpc.LIGHTNING || receiver.Type != tdrpc.LIGHTNING {
		return nil, fmt.Errorf("Invalid type sender:%s receiver:%s", sender.Type, receiver.Type)
	}

	// Since this is an internal payment, we will need to create a new IN record in case someone pays the payment request anyhow
	receiver.Id = internalID
	err = c.processLedgerRecord(ctx, tx, &receiver)
	if err != nil {
		return nil, err
	}

	if sender.ExpiresAt == nil || sender.ExpiresAt.IsZero() || receiver.ExpiresAt == nil || receiver.ExpiresAt.IsZero() {
		return nil, fmt.Errorf("Invalid expiration time")
	}

	// Set the expired time if it's not
	if time.Now().UTC().After(*sender.ExpiresAt) || time.Now().UTC().After(*receiver.ExpiresAt) {
		_, err = tx.ExecContext(ctx, `UPDATE ledger SET status = $1 WHERE id = $2 OR id = $3`, tdrpc.EXPIRED, id, internalID)
		if err != nil {
			return nil, err
		}

		// Commit the transaction and return the error
		err = tx.Commit()
		if err != nil {
			return nil, err
		}

		return nil, store.ErrRequestExpired
	}

	// The funds have already been placed into balance out by the pending request setting it up

	// Update the sender pending_out
	_, err = tx.ExecContext(ctx, `UPDATE account SET pending_out = pending_out - $1 WHERE id = $2`, sender.ValueTotal(), sender.AccountId)
	if err != nil {
		return nil, fmt.Errorf("Could not process sender pending_out: %v", err)
	}

	// Update the receiver balance
	_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, sender.ValueTotal(), receiver.AccountId)
	if err != nil {
		return nil, fmt.Errorf("Could not process receiver balance: %v", err)
	}

	// Update both LedgerRecords
	_, err = tx.ExecContext(ctx, `
		UPDATE ledger SET
		updated_at = NOW(),
		status = $1,
		value = $2
		WHERE id = $3
	`, tdrpc.COMPLETED, sender.ValueTotal(), internalID)
	if err != nil {
		return nil, err
	}

	// Get the updated sender record
	err = c.db.GetContext(ctx, &sender, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, internalID, tdrpc.OUT)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find post sender request")
	} else if err != nil {
		return nil, err
	}

	return &sender, nil

}

// GetLedger returns the ledger for a user
func (c *Client) GetLedger(ctx context.Context, accountID string) ([]*tdrpc.LedgerRecord, error) {

	var lrs = make([]*tdrpc.LedgerRecord, 0)
	err := c.db.SelectContext(ctx, &lrs, `SELECT * FROM ledger WHERE account_id = $1`, accountID)
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
		NOT EXISTS (SELECT 1 FROM ledger AS li WHERE li.id = CONCAT(ledger.id, '`+thunderdome.InternalIdSuffix+`'))
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
				c.logger.Warnf("TX Fail: %v - Retries Left %d", err, retries)
				continue
			}
			return err
		}

		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
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

// Returns the difference in two ledger records suitable for logging
func getLedgerDeltaLog(lr1, lr2 *tdrpc.LedgerRecord) []interface{} {

	ret := make([]interface{}, 0)

	if lr1 == nil || lr2 == nil {
		return ret
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

	return ret

}
