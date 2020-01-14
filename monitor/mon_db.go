package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/statsd"

	"git.coinninja.net/backend/thunderdome/conf"
)

// MonitorDB will chgeck t
func (m *Monitor) MonitorDB() {

	// Handle shutting down
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-conf.Stop.Chan()
		cancel()
	}()

	firstRun := make(chan struct{}, 1)
	firstRun <- struct{}{}

monLoop:
	for !conf.Stop.Bool() {

		// Sleep 1 minute
		select {
		case <-firstRun:
		case <-conf.Stop.Chan():
			break monLoop
		case <-time.After(15 * time.Minute):
		}

		err := m.store.CheckDatabaseConsistency(ctx)
		if err != nil {
			m.logger.Errorw("Database Validation Error", "error", err)

			if m.ddclient != nil {
				_ = m.ddclient.Event(&statsd.Event{
					Title:     "Thunderdome Database Consistency Issue",
					Text:      fmt.Sprintf(`Thunderdome Database Consistency Issue: %v`, err),
					Priority:  statsd.Normal,
					AlertType: statsd.Error,
				})
			}
		} else {
			m.logger.Info("Database validation complete.")
		}

	}

}
