package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// GetLastChanBackup gets the last channel backup from the database
func (c *Client) GetLastChanBackup(ctx context.Context) (*tdrpc.ChanBackup, error) {

	var cb tdrpc.ChanBackup
	err := c.db.GetContext(ctx, &cb, `SELECT * FROM chan_backup ORDER BY TIMESTAMP DESC LIMIT 1`)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &cb, nil

}

// StoreChanBackup stores a channel backup in the database
func (c *Client) StoreChanBackup(ctx context.Context, fundingTXIDs string, data tdrpc.ChanBackupData) (*tdrpc.ChanBackup, error) {

	var cb tdrpc.ChanBackup
	err := c.db.GetContext(ctx, &cb, `INSERT INTO chan_backup (funding_txids, data) VALUES($1, $2) RETURNING *`, fundingTXIDs, data)
	if err != nil {
		return nil, fmt.Errorf("Could not store backup: %v", err)
	}
	return &cb, nil

}
