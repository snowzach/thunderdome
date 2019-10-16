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

// ProcessLedgerRecord handles any balance transfer and changes to the ledger based on the status of the LedgerRecord
func (c *Client) ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error {

	for retries := 10; retries > 0; retries-- {

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
				c.logger.Warnf("ProcessLedgerRecord TX Fail: %v - Retries Left %d", err, retries)
				time.Sleep(time.Duration(retries) * time.Second)
				continue
			}
			return err
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			if IsTransactionError(err) {
				c.logger.Warnf("ProcessLedgerRecord TX Fail: %v - Retries Left %d", err, retries)
				time.Sleep(time.Duration(retries) * time.Second)
				continue
			}
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
		// Create a log entry for deltas (if any)
		c.LedgerDeltaLog(prevlr, lr)

		// The previous transaction exists, validate that nothing of note has changed in the request
		if prevlr.Type != lr.Type || prevlr.Direction != lr.Direction || prevlr.AccountId != lr.AccountId {
			return fmt.Errorf("Existing Ledger Entry Mismatch %v:%v", prevlr, lr)
		}

		// Invalid status transitions
		if ((prevlr.Status == tdrpc.EXPIRED || prevlr.Status == tdrpc.COMPLETED) && (lr.Status == tdrpc.PENDING || lr.Status == tdrpc.FAILED)) ||
			// Completed is a final state always
			(prevlr.Status == tdrpc.COMPLETED && lr.Status != tdrpc.COMPLETED) {
			if prevlr.Status == tdrpc.COMPLETED {
				return tdrpc.ErrRequestAlreadyPaid
			}
			return fmt.Errorf("Invalid Status Transition %v->%v", prevlr.Status, lr.Status)
		}

		// If the status hasn't changed
		if prevlr.Status == lr.Status {

			// Update only the fields we are allowed to update, and only set updated_at if something changed
			_, err = tx.ExecContext(ctx, `
				UPDATE ledger SET
				updated_at = NOW(),
				expires_at = $1,
				memo = $2,
				request = $3,
				error = $4
				WHERE id = $5 AND direction = $6 AND (
					expires_at <> $1 OR
					memo <> $2 OR
					request <> $3 OR
					error <> $4
				)
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
					return tdrpc.ErrInsufficientFunds
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
					return tdrpc.ErrInsufficientFunds
				}

				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance - $1 WHERE id = $2`, lr.ValueTotal(), lr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process out new completed balance: %v", err)
				}
			}
		}

	} else if lr.Direction == tdrpc.IN {

		// When inbound, only the values are taken into account. Network Fee and Processing Fee are ignored

		// There is a previous LedgerRecord
		if prevlr != nil {

			// It was previously pending, pull the reserved funds from pending_in
			if prevlr.Status == tdrpc.PENDING && lr.Type == tdrpc.BTC {
				_, err = tx.ExecContext(ctx, `UPDATE account SET pending_in = pending_in - $1 WHERE id = $2`, prevlr.ValueTotal(), prevlr.AccountId)
				if err != nil {
					return fmt.Errorf("Could not process in new pending balance: %v", err)
				}
			}
		}

		// Pending incoming transactions (but only for BTC) add balance to pending
		if lr.Status == tdrpc.PENDING && lr.Type == tdrpc.BTC {
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

	// If the status is completed, ensure that it is not hidden on update
	// Internal payments can lead to a case where this is payable but hidden
	if lr.Status == tdrpc.COMPLETED {
		lr.Hidden = false
	}

	// Upsert the data, capture the result
	var ret tdrpc.LedgerRecord
	err = tx.GetContext(ctx, &ret, `
		INSERT INTO ledger (id, account_id, created_at, updated_at, expires_at, status, type, direction, generated, value, network_fee, processing_fee, add_index, memo, request, error, hidden)
		VALUES($1, $2, NOW(), NOW(), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id, direction) DO UPDATE
		SET
		updated_at = NOW(),
		expires_at = $3,
		status = $4,
		value = $8,
		network_fee = $9,
		processing_fee = $10,
		memo = $12,
		request = $13,
		error = $14,
		hidden = $15
		RETURNING *
	`, lr.Id, lr.AccountId, lr.ExpiresAt, lr.Status, lr.Type, lr.Direction, lr.Generated, lr.Value, lr.NetworkFee, lr.ProcessingFee, lr.AddIndex, lr.Memo, lr.Request, lr.Error, lr.Hidden)
	if err != nil {
		return fmt.Errorf("Could not process ledger: %v", err)
	}

	// Replace the existing value with the modified one
	*lr = ret

	return nil

}
