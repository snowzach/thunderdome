package postgres

import (
	"context"
	"fmt"
)

// CheckDatabaseConsistency will balance all deposits, withdraws, sent and received invoices with the current balance to ensure no discrepancies
func (c *Client) CheckDatabaseConsistency(ctx context.Context) error {

	var delta int
	err := c.db.GetContext(ctx, &delta, `
	SELECT
		(SELECT COALESCE(SUM(value), 0) AS total FROM ledger WHERE direction = 'in' AND status = 'completed')
	-
		(SELECT COALESCE(SUM(value), 0) + COALESCE(SUM(network_fee), 0) + COALESCE(SUM(processing_fee), 0) AS total FROM ledger WHERE direction = 'out' AND (status = 'completed' OR status = 'pending'))
	-
		(SELECT COALESCE(SUM(balance), 0) FROM account)
	AS delta
	`)
	if err != nil {
		return fmt.Errorf("Could not run validation query: %v", err)
	}

	if delta != 0 {
		return fmt.Errorf("Validation delta is not 0 = %d", delta)
	}

	return nil
}
