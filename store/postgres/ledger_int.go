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

	if sender.ExpiresAt == nil || sender.ExpiresAt.IsZero() || receiver.ExpiresAt == nil || receiver.ExpiresAt.IsZero() {
		return nil, fmt.Errorf("Invalid expiration time")
	}

	// Since this is an internal payment, we will need to create a new IN record in case someone pays the payment request anyhow
	// This will also prevent double payment of the request internally
	receiver.Id = internalID
	origReceiverValue := receiver.Value
	receiver.Value = 0 // This avoids screwing up the pending_in balance
	err = c.processLedgerRecord(ctx, tx, &receiver)
	if err != nil {
		return nil, err
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

	// The sender funds have already been placed into pending_out by the pending request setting it up
	// Update the sender pending_out and remove them
	_, err = tx.ExecContext(ctx, `UPDATE account SET pending_out = pending_out - $1 WHERE id = $2`, sender.ValueTotal(), sender.AccountId)
	if err != nil {
		return nil, fmt.Errorf("Could not process sender pending_out: %v", err)
	}

	// Update the receiver balance
	_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1, pending_in = pending_in - $2 WHERE id = $3`, sender.ValueTotal(), origReceiverValue, receiver.AccountId)
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

	// Update the real ledger record to hidden unless it's already completed (which is technically impossible)
	// It also updates the value = 0 as the pending_in value has already been adjusted above to no longer show the balance
	_, err = tx.ExecContext(ctx, `
		UPDATE ledger SET
		hidden = true,
		value = 0
		WHERE id = $1 AND status != $2
	`, id, tdrpc.COMPLETED)
	if err != nil {
		return nil, err
	}

	// Get the updated sender record
	err = tx.GetContext(ctx, &sender, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, internalID, tdrpc.OUT)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find post sender request")
	} else if err != nil {
		return nil, err
	}

	return &sender, nil

}
