package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/jmoiron/sqlx"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// ProcessInternal will process an payment between two accounts on this system
func (c *Client) ProcessInternal(ctx context.Context, id string, lr *tdrpc.LedgerRecord) (*tdrpc.LedgerRecord, error) {

	for retries := 10; retries > 0; retries-- {

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

		lr, err := c.processInternal(ctx, tx, id, lr)
		if err != nil {
			_ = tx.Rollback()
			if IsTransactionError(err) {
				c.logger.Warnf("ProcessInternal TX Fail: %v - Retries Left %d", err, retries)
				time.Sleep(time.Duration(retries) * time.Second)
				continue
			}
			return nil, err
		}

		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			if IsTransactionError(err) {
				c.logger.Warnf("ProcessInternal TX Fail: %v - Retries Left %d", err, retries)
				time.Sleep(time.Duration(retries) * time.Second)
				continue
			}
			return nil, fmt.Errorf("Commit Error: %v", err)
		}

		return lr, nil

	}

	return nil, fmt.Errorf("Transaction failed, out of retries")

}

func (c *Client) processInternal(ctx context.Context, tx *sqlx.Tx, id string, paylr *tdrpc.LedgerRecord) (*tdrpc.LedgerRecord, error) {

	var prevlr, receiver tdrpc.LedgerRecord
	internalID := id + tdrpc.InternalIdSuffix

	// Get the receiver record
	err := tx.GetContext(ctx, &receiver, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, id, tdrpc.IN)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find receiver request. Was this a manually created payment request?")
	} else if err != nil {
		return nil, err
	}

	// Get the prevlr record which could either be a current payment or possibly a pre-authorized payment
	err = tx.GetContext(ctx, &prevlr, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, internalID, tdrpc.OUT)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find sender request")
	} else if err != nil {
		return nil, err
	}

	// Make sure the status is right
	if prevlr.Status != tdrpc.PENDING || receiver.Status != tdrpc.PENDING {
		return nil, fmt.Errorf("Invalid status sender:%s receiver:%s", prevlr.Status, receiver.Status)
	}

	// Make sure the status is right
	if prevlr.Type != tdrpc.LIGHTNING || receiver.Type != tdrpc.LIGHTNING {
		return nil, fmt.Errorf("Invalid type sender:%s receiver:%s", prevlr.Type, receiver.Type)
	}

	if prevlr.ExpiresAt == nil || prevlr.ExpiresAt.IsZero() || receiver.ExpiresAt == nil || receiver.ExpiresAt.IsZero() || paylr.ExpiresAt == nil || paylr.ExpiresAt.IsZero() {
		return nil, fmt.Errorf("Invalid expiration time")
	}

	// The prevlr (possibly the pre-auth) has expired
	if time.Now().UTC().After(*prevlr.ExpiresAt) {
		_, err = tx.ExecContext(ctx, `UPDATE ledger SET status = $1 WHERE id = $2`, tdrpc.EXPIRED, internalID)
		if err != nil {
			return nil, err
		}
		// Commit the transaction and return the error
		err = tx.Commit()
		if err != nil {
			return nil, err
		}
		return nil, tdrpc.ErrRequestExpired
	}

	// The actual payment request has expired
	if time.Now().UTC().After(*receiver.ExpiresAt) {
		_, err = tx.ExecContext(ctx, `UPDATE ledger SET status = $1 WHERE id = $2 OR id = $3`, tdrpc.EXPIRED, id, internalID)
		if err != nil {
			return nil, err
		}
		// Commit the transaction and return the error
		err = tx.Commit()
		if err != nil {
			return nil, err
		}
		return nil, tdrpc.ErrRequestExpired
	}

	// If the payment value is larger than the prevlr value, then this is a pre-authorized transaction and it exceeds the pre-auth amount
	if paylr.ValueTotal() > prevlr.ValueTotal() {
		return nil, fmt.Errorf("Request value exceeds pre-authorized amount")
	}

	// Since this is an internal payment, we will need to create a new IN record in case someone pays the payment request anyhow
	// This will also prevent double payment of the request internally
	receiver.Id = internalID
	err = c.processLedgerRecord(ctx, tx, &receiver)
	if err != nil {
		return nil, err
	}

	// The sender funds have already been placed into pending_out by the pending request setting it up
	// Update the sender pending_out and remove them
	_, err = tx.ExecContext(ctx, `UPDATE account SET pending_out = pending_out - $1 WHERE id = $2`, prevlr.ValueTotal(), prevlr.AccountId)
	if err != nil {
		return nil, fmt.Errorf("Could not process sender pending_out: %v", err)
	}

	// Update the sender balance just in case the pending amount is not the same as the actual amount with any unspent funds
	// This could be the case if a transaction was pre-authorized with a larger amount than was actually sent
	// This should not be typical
	if prevlr.ValueTotal()-paylr.ValueTotal() != 0 {
		_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, prevlr.ValueTotal()-paylr.ValueTotal(), prevlr.AccountId)
		if err != nil {
			return nil, fmt.Errorf("Could not process sender balance: %v", err)
		}
	}

	// Update the receiver balance, unlock the account if it is
	_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1, locked = false WHERE id = $2`, paylr.Value, receiver.AccountId)
	if err != nil {
		return nil, fmt.Errorf("Could not process receiver balance: %v", err)
	}

	// Update both sender/receiver LedgerRecords with all the proper fields
	_, err = tx.ExecContext(ctx, `
		UPDATE ledger SET
		updated_at = NOW(),
		status = $1,
		value = $2,
		network_fee = $3,
		processing_fee = $4,
		memo = $5,
		request = $6,
		hidden = false
		WHERE id = $7
	`, tdrpc.COMPLETED, paylr.Value, paylr.NetworkFee, paylr.ProcessingFee, paylr.Memo, paylr.Request, internalID)
	if err != nil {
		return nil, err
	}

	// Update the real ledger record/payment request to hidden unless it's already completed (which is technically impossible)
	_, err = tx.ExecContext(ctx, `
		UPDATE ledger SET
		hidden = true
		WHERE id = $1 AND status != $2
	`, id, tdrpc.COMPLETED)
	if err != nil {
		return nil, err
	}

	// Get the updated record, use the prevlr variable to hold the return value
	err = tx.GetContext(ctx, &prevlr, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, internalID, tdrpc.OUT)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not get post processed request")
	} else if err != nil {
		return nil, err
	}

	return &prevlr, nil

}
