package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// UpsertLedgerRecord opens a ledger request and handles any balance transfers etc
func (c *Client) UpsertLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error {

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
	if err == sql.ErrNoRows {
		prevlr = nil
	} else if err != nil {
		return fmt.Errorf("Could not fetch prevStatus: %v", err)
	} else if prevlr.Status == tdrpc.FAILED {
		// Failed is basically like there is no previous status
		prevlr = nil
	} else {
		// If the status hasn't changed, there's nothing to update
		if prevlr.Status == lr.Status {
			tx.Rollback()
			return nil
		}

		// The previous transaction exists, validate that nothing of note has changed in the request
		if prevlr.Type != lr.Type || prevlr.Direction != lr.Direction || prevlr.AccountId != lr.AccountId || prevlr.Request != lr.Request {
			return fmt.Errorf("Existing Ledger Entry Mismatch %v:%v", prevlr, lr)
		}

		// Invalid state transitions
		if ((prevlr.Status == tdrpc.EXPIRED || prevlr.Status == tdrpc.COMPLETED) && lr.Status == tdrpc.PENDING) ||
			(prevlr.Status == tdrpc.COMPLETED && lr.Status == tdrpc.EXPIRED) ||
			(prevlr.Status == tdrpc.EXPIRED && lr.Status == tdrpc.COMPLETED) {
			return fmt.Errorf("Invalid State Transition %v->%v", prevlr.Status, lr.Status)
		}
	}

	// Handle outbound requests
	if lr.Direction == tdrpc.OUT {

		// There is a previous LedgerRecord
		if prevlr != nil {

			if prevlr.Status == tdrpc.PENDING { // It was previously pending, pull the reserved funds from balance_out
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance_out = balance_out - $1 WHERE id = $2`, prevlr.Value, prevlr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
			}

			if lr.Status == tdrpc.EXPIRED { // Put the money back in balance - it expired
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, lr.Value, lr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
			}

		} else { // No previous record

			// Get the balance
			var balance int64
			err = tx.GetContext(ctx, &balance, `SELECT balance FROM account WHERE id = ?`, lr.Id)
			if err != nil {
				tx.Rollback()
				return err
			}

			if lr.Status == tdrpc.PENDING {

				// Check to make sure we have enough funds
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

			} else if lr.Status == tdrpc.COMPLETED { // Pull the money out of balance

				// Check to make sure we have enough funds
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
			if prevlr.Status == tdrpc.PENDING { // It was previously pending, pull the reserved funds from balance_in
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance_in = balance_in - $1 WHERE id = $2`, prevlr.Value, prevlr.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}

		if lr.Status == tdrpc.COMPLETED { // Put the money back in balance - it expired
			_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, lr.Value, lr.Id)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

	} else {
		tx.Rollback()
		return fmt.Errorf("Unknown direction: %v", lr.Direction)
	}

	_, err = tx.QueryxContext(ctx, `
		INSERT INTO ledger (id, account_id, created_at, updated_at, expires_at, status, type, direction, value, memo, request)
		VALUES($1, $2, NOW(), NOW(), $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE
		SET
		updated_at = NOW(),
		expires_at = $3,
		status = $4,
		type = $5,
		direction = $6,
		value = $7,
		memo = $8,
		request = $9
	`, lr.Id, lr.AccountId, lr.ExpiresAt, lr.Status, lr.Type, lr.Direction, lr.Value, lr.Memo, lr.Request)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil

}
