package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// ProcessLedgerRecord opens a ledger request and handles any balance transfers etc
func (c *Client) ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error {

	// Start a transaction
	tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return fmt.Errorf("Could not start transaction: %v", err)
	}

	// See if the ledger entry already exists
	prevlr := new(tdrpc.LedgerRecord)
	err = tx.GetContext(ctx, &prevlr, `SELECT * FROM ledger WHERE id = $1`, lr.Id)
	if err == sql.ErrNoRows || (prevlr != nil && prevlr.Status == tdrpc.FAILED) {
		// There is no previous record or the status is failed.
		// We don't need to consider anything with the previous record
		prevlr = nil
	} else if err != nil {
		return fmt.Errorf("Could not fetch existing LedgerRecord: %v", err)
	} else {
		// The previous transaction exists, validate that nothing of note has changed in the request
		if prevlr.Type != lr.Type || prevlr.Direction != lr.Direction || prevlr.AccountId != lr.AccountId || prevlr.Request != lr.Request {
			return fmt.Errorf("Existing Ledger Entry Mismatch %v:%v", prevlr, lr)
		}

		// Invalid status transitions
		if ((prevlr.Status == tdrpc.EXPIRED || prevlr.Status == tdrpc.COMPLETED) && (lr.Status == tdrpc.PENDING || lr.Status == tdrpc.FAILED)) ||
			(prevlr.Status == tdrpc.COMPLETED && lr.Status == tdrpc.EXPIRED) ||
			(prevlr.Status == tdrpc.EXPIRED && lr.Status == tdrpc.COMPLETED) {
			return fmt.Errorf("Invalid Status Transition %v->%v", prevlr.Status, lr.Status)
		}

		// If the status hasn't changed
		if prevlr.Status == lr.Status {

			// Update only the fields we are allowed to update
			_, err = tx.ExecContext(ctx, `
				UPDATE ledger SET
				updated_at = NOW(),
				expires_at = $1,
				value = $2,
				memo = $3,
				request = $4
			`, lr.ExpiresAt, lr.Value, lr.Memo, lr.Request)
			if err != nil {
				tx.Rollback()
				return err
			}

			// Commit the transaction
			err = tx.Commit()
			if err != nil {
				tx.Rollback()
				return err
			}

			return nil
		}
	}

	// Handle outbound requests
	if lr.Direction == tdrpc.OUT {

		// There is a previous LedgerRecord and the status has changed
		if prevlr != nil {

			// It was previously pending, pull the reserved funds from balance_out
			if prevlr.Status == tdrpc.PENDING {
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance_out = balance_out - $1 WHERE id = $2`, prevlr.Value, prevlr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
			}

			// It failed or expired - put the money back in balance
			if lr.Status == tdrpc.EXPIRED || lr.Status == tdrpc.FAILED {
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, prevlr.Value, prevlr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
			}

		} else { // No previous record/status

			// Get the current balance
			var balance int64
			err = tx.GetContext(ctx, &balance, `SELECT balance FROM account WHERE id = ?`, lr.Id)
			if err != nil {
				tx.Rollback()
				return err
			}

			// We've started a new transaction
			if lr.Status == tdrpc.PENDING {

				// Check to make sure we have enough funds to start this transaction
				if balance < lr.Value {
					tx.Rollback()
					return store.ErrInsufficientFunds
				}

				// Put it in balance_out
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance - $1, balance_out = balance_out + $1 WHERE id = $2`, lr.Value, lr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}

				// The transaction is completed
			} else if lr.Status == tdrpc.COMPLETED {

				// Check to make sure we have enough funds - this should never really happen unless somehow made out of band
				if balance < lr.Value {
					tx.Rollback()
					return store.ErrInsufficientFunds
				}

				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance - $1 WHERE id = $2`, lr.Value, lr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}

	} else if lr.Direction == tdrpc.IN {

		// There is a previous LedgerRecord
		if prevlr != nil {

			// It was previously pending, pull the reserved funds from balance_in
			if prevlr.Status == tdrpc.PENDING {
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance_in = balance_in - $1 WHERE id = $2`, prevlr.Value, prevlr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}

		// It completed, put the value into the balance
		if lr.Status == tdrpc.COMPLETED {
			_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, lr.Value, lr.Id)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

	} else {
		// Not possible
		tx.Rollback()
		return fmt.Errorf("Unknown direction: %v", lr.Direction)
	}

	// Upsert the data
	_, err = tx.ExecContext(ctx, `
		INSERT INTO ledger (id, account_id, created_at, updated_at, expires_at, status, type, direction, value, memo, request)
		VALUES($1, $2, NOW(), NOW(), $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE
		SET
		updated_at = NOW(),
		expires_at = $3,
		status = $4,
		value = $7,
		memo = $8,
		request = $9
	`, lr.Id, lr.AccountId, lr.ExpiresAt, lr.Status, lr.Type, lr.Direction, lr.Value, lr.Memo, lr.Request)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil

}

// GetLedger returns the ledger for a user
func (c *Client) GetLedger(ctx context.Context, accountID string) ([]*tdrpc.LedgerRecord, error) {

	var lrs = make([]*tdrpc.LedgerRecord, 0, 0)
	err := c.db.SelectContext(ctx, &lrs, `SELECT * FROM ledger WHERE account_id = $1`, accountID)
	if err != nil {
		return lrs, err
	}
	return lrs, nil

}
