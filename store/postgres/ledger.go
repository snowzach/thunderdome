package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// ProcessLedgerRecord handles any balance transfer and changes to the ledger based on the status of the LedgerRecord
func (c *Client) ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error {

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
			tx.Rollback()
			c.logger.Panic(r)
		}
	}()

	// See if the ledger entry already exists
	prevlr := new(tdrpc.LedgerRecord)
	err = tx.GetContext(ctx, prevlr, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, lr.Id, lr.Direction)
	if err == sql.ErrNoRows || (prevlr != nil && prevlr.Status == tdrpc.FAILED) {
		// There is no previous record or the status is failed.
		// We don't need to consider anything with the previous record
		prevlr = nil
	} else if err != nil {
		return fmt.Errorf("Could not fetch existing LedgerRecord: %v", err)
	}

	c.logger.Debugw("ProcessLedgerRecord Delta", getLedgerDeltaLog(prevlr, lr)...)

	if prevlr != nil {
		// The previous transaction exists, validate that nothing of note has changed in the request
		if prevlr.Type != lr.Type || prevlr.Direction != lr.Direction || prevlr.AccountId != lr.AccountId || prevlr.Request != lr.Request {
			tx.Rollback()
			return fmt.Errorf("Existing Ledger Entry Mismatch %v:%v", prevlr, lr)
		}

		// Invalid status transitions
		if ((prevlr.Status == tdrpc.EXPIRED || prevlr.Status == tdrpc.COMPLETED) && (lr.Status == tdrpc.PENDING || lr.Status == tdrpc.FAILED)) ||
			(prevlr.Status == tdrpc.COMPLETED && lr.Status != tdrpc.COMPLETED) ||
			(prevlr.Status == tdrpc.EXPIRED && lr.Status != tdrpc.EXPIRED) ||
			(prevlr.Status == tdrpc.PENDING && lr.Status == tdrpc.PENDING) {
			tx.Rollback()
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
				request = $4,
				error = $5
				WHERE id = $6 AND direction = $7
			`, lr.ExpiresAt, lr.Value, lr.Memo, lr.Request, lr.Error, lr.Id, lr.Direction)
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

				_, err = tx.ExecContext(ctx, `UPDATE account SET balance_out = balance_out - $1 WHERE id = $2`, prevlr.Value, prevlr.AccountId)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("Could not process out existing pending balance_out: %v", err)
				}
			}

			// It failed or expired - put the money back in balance
			if lr.Status == tdrpc.EXPIRED || lr.Status == tdrpc.FAILED {
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, prevlr.Value, prevlr.AccountId)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("Could not process out existing failed/expired balance: %v", err)
				}
			}

		} else { // No previous record/status

			// Get the current balance
			var balance int64
			err = tx.GetContext(ctx, &balance, `SELECT balance FROM account WHERE id = $1`, lr.AccountId)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("Could not get out new balance: %v", err)
			}

			// We've started a new transaction
			if lr.Status == tdrpc.PENDING {

				// Check to make sure we have enough funds to start this transaction
				if balance < lr.Value {
					tx.Rollback()
					return store.ErrInsufficientFunds
				}

				// Put it in balance_out
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance - $1, balance_out = balance_out + $1 WHERE id = $2`, lr.Value, lr.AccountId)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("Could not process out new pending balance_out: %v", err)
				}

				// The transaction is completed
			} else if lr.Status == tdrpc.COMPLETED {

				// Check to make sure we have enough funds - this should never really happen unless somehow made out of band
				if balance < lr.Value {
					tx.Rollback()
					return store.ErrInsufficientFunds
				}

				_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance - $1 WHERE id = $2`, lr.Value, lr.AccountId)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("Could not process out new completed balance: %v", err)
				}
			}
		}

	} else if lr.Direction == tdrpc.IN {

		// There is a previous LedgerRecord
		if prevlr != nil {

			// It was previously pending, pull the reserved funds from balance_in
			if prevlr.Status == tdrpc.PENDING {
				_, err = tx.ExecContext(ctx, `UPDATE account SET balance_in = balance_in - $1 WHERE id = $2`, prevlr.Value, prevlr.AccountId)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("Could not process in new pending balance: %v", err)
				}
			}
		}

		// Pending incoming transactions
		if lr.Status == tdrpc.PENDING {
			_, err = tx.ExecContext(ctx, `UPDATE account SET balance_in = balance_in + $1 WHERE id = $2`, lr.Value, lr.AccountId)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("Could not process in new completed balance: %v", err)
			}

			// It completed, put the value into the balance
		} else if lr.Status == tdrpc.COMPLETED {
			_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, lr.Value, lr.AccountId)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("Could not process in new completed balance: %v", err)
			}
		}

	} else {
		// Not possible
		tx.Rollback()
		return fmt.Errorf("Unknown direction: %v", lr.Direction)
	}

	// Upsert the data, capture the result
	var ret tdrpc.LedgerRecord
	err = tx.GetContext(ctx, &ret, `
		INSERT INTO ledger (id, account_id, created_at, updated_at, expires_at, status, type, direction, value, add_index, memo, request, error)
		VALUES($1, $2, NOW(), NOW(), $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id, direction) DO UPDATE
		SET
		updated_at = NOW(),
		expires_at = $3,
		status = $4,
		value = $7,
		memo = $9,
		request = $10,
		error = $11
		RETURNING *
	`, lr.Id, lr.AccountId, lr.ExpiresAt, lr.Status, lr.Type, lr.Direction, lr.Value, lr.AddIndex, lr.Memo, lr.Request, lr.Error)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Could not process ledger: %v", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Commit Error: %v", err)
	}

	// Replace the existing value with the modified one
	*lr = ret

	return nil

}

// ProcessInternal will process an payment between two accounts on this system
func (c *Client) ProcessInternal(ctx context.Context, id string) (*tdrpc.LedgerRecord, error) {

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
			tx.Rollback()
			c.logger.Panic(r)
		}
	}()

	var sender, receiver tdrpc.LedgerRecord

	// Get the receiver record
	err = tx.GetContext(ctx, &receiver, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, id, tdrpc.IN)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find receiver request")
	} else if err != nil {
		return nil, err
	}

	// Get the sender record
	err = tx.GetContext(ctx, &sender, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, id, tdrpc.OUT)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find sender request")
	} else if err != nil {
		return nil, err
	}

	// Make sure the status is right
	if sender.Status != tdrpc.PENDING || receiver.Status != tdrpc.PENDING {
		return nil, fmt.Errorf("Invalid status sender:%s receiver:%s", sender.Status, receiver.Status)
	}

	// Set the expired time if it's not
	if time.Now().UTC().After(*sender.ExpiresAt) || time.Now().UTC().After(*receiver.ExpiresAt) {
		_, err = tx.ExecContext(ctx, `UPDATE ledger SET status = $1 WHERE id = $2`, tdrpc.EXPIRED, id)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		// Commit the transaction and return the error
		err = tx.Commit()
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		return nil, store.ErrRequestExpired
	}

	// Get the current balance from the sender
	var balance int64
	err = tx.GetContext(ctx, &balance, `SELECT balance FROM account WHERE id = $1`, sender.AccountId)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("Could not get out new balance: %v", err)
	}

	// Check the balance is sufficient
	if balance < sender.Value {
		tx.Rollback()
		return nil, store.ErrInsufficientFunds
	}

	// Update the sender balance_out
	_, err = tx.ExecContext(ctx, `UPDATE account SET balance_out = balance_out - $1 WHERE id = $2`, sender.Value, sender.AccountId)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("Could not process sender balance_out: %v", err)
	}

	// Update the receiver balance
	_, err = tx.ExecContext(ctx, `UPDATE account SET balance = balance + $1 WHERE id = $2`, sender.Value, receiver.AccountId)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("Could not process receiver balance: %v", err)
	}

	// Update both LedgerRecords
	_, err = tx.ExecContext(ctx, `
		UPDATE ledger SET
		updated_at = NOW(),
		status = $1,
		value = $2
		WHERE id = $3
	`, tdrpc.COMPLETED, sender.Value, id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Get the updated sender record
	err = c.db.GetContext(ctx, &sender, `SELECT * FROM ledger WHERE id = $1 AND direction = $2`, id, tdrpc.OUT)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Could not find post sender request")
	} else if err != nil {
		return nil, err
	}

	return &sender, nil

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
	if lr1.Direction != lr2.Direction {
		ret = append(ret, "direction", fmt.Sprintf("%v->%v", lr1.Direction, lr2.Direction))
	}
	if lr1.Value != lr2.Value {
		ret = append(ret, "value", fmt.Sprintf("%v->%v", lr1.Value, lr2.Value))
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
