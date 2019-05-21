package postgres

import (
	"context"
	"database/sql"

	"git.coinninja.net/backend/thunderdome/store"
)

// AddInvoice adds an invoice
func (c *Client) AddInvoice(ctx context.Context, userID string, paymentHash string) error {

	_, err := c.db.QueryxContext(ctx, `
		INSERT INTO invoice (user_id, payment_hash)
		VALUES($1, $2)
	`, userID, paymentHash)
	if err != nil {
		return err
	}
	return nil

}

//  GetUserIDByPaymentHash gets userID from paymentHash
func (c *Client) GetUserIDByPaymentHash(ctx context.Context, paymentHash string) (string, error) {

	var userID string
	err := c.db.GetContext(ctx, userID, `SELECT user_id FROM invoice WHERE payment_hash = $1`, paymentHash)
	if err == sql.ErrNoRows {
		return "", store.ErrNotFound
	} else if err != nil {
		return "", err
	}

	return userID, nil
}
