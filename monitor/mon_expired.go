package monitor

import (
	"context"
	"time"

	"git.coinninja.net/backend/thunderdome/conf"
)

// MonitorExpired will mark records as expired once every 2 minutes
// This will restore handle balance processing as well
func (m *Monitor) MonitorExpired() {

	for !conf.Stop.Bool() {

		err := m.store.ExpireLedgerRequests(context.Background())
		if err != nil {
			m.logger.Fatalw("Could not ExpireLedgerRequests", "error", err)
		}

		select {
		case <-time.After(2 * time.Minute):
		case <-conf.Stop.Chan():
		}
	}

}
