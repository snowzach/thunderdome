package monitor

import (
	"context"
	"time"

	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type Stats struct {
	UserCount                int64 `json:"user_count"`
	UserBalance              int64 `json:"user_balance"`
	UserPendingIn            int64 `json:"user_pending_in"`
	UserPendingOut           int64 `json:"user_pending_out"`
	TopupPendingCount        int64 `json:"topup_pending_count"`
	TopupPendingValue        int64 `json:"topup_pending_value"`
	TopupInstantPendingCount int64 `json:"topup_instant_pending_count"`
	TopupInstantPendingValue int64 `json:"topup_instant_pending_value"`
}

// MonitorStats will log stats from the local system
func (m *Monitor) MonitorStats() {

	// Handle shutting down
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-conf.Stop.Chan()
		cancel()
	}()

	firstRun := make(chan struct{}, 1)
	firstRun <- struct{}{}

	// Monitor the channels and balance of the LND node
monLoop:
	for !conf.Stop.Bool() {

		// Sleep 1 minute
		select {
		case <-firstRun:
		case <-conf.Stop.Chan():
			break monLoop
		case <-time.After(1 * time.Minute):
		}

		var stats Stats

		// Get regular pending stats
		lrStats, err := m.store.GetLedgerRecordStats(ctx, map[string]string{
			"type":      tdrpc.BTC.String(),
			"direction": tdrpc.IN.String(),
			"status":    tdrpc.PENDING.String(),
		}, time.Time{})
		if err != nil {
			m.logger.Errorw("Could not get system ledger record stats", "error", err)
			continue
		}
		stats.TopupPendingCount = lrStats.Count
		stats.TopupPendingValue = lrStats.Value

		// Get regular pending stats
		lrStats, err = m.store.GetLedgerRecordStats(ctx, map[string]string{
			"type":      tdrpc.BTC.String(),
			"direction": tdrpc.IN.String(),
			"status":    tdrpc.COMPLETED.String(),
			"request":   tdrpc.RequestInstantPending,
		}, time.Time{})
		if err != nil {
			m.logger.Errorw("Could not get system instant ledger record stats", "error", err)
			continue
		}
		stats.TopupInstantPendingCount = lrStats.Count
		stats.TopupInstantPendingValue = lrStats.Value

		aStats, err := m.store.GetAccountStats(ctx)
		if err != nil {
			m.logger.Errorw("Could not get account stats", "error", err)
			continue
		}
		stats.UserCount = aStats.Count
		stats.UserBalance = aStats.Balance
		stats.UserPendingIn = aStats.PendingIn
		stats.UserPendingOut = aStats.PendingOut

		m.logger.Debugw("Stats", zap.Any("stats", stats))

		if m.ddclient != nil {
			m.ddclient.Gauge("user_count", float64(stats.UserCount), []string{}, 1)
			m.ddclient.Gauge("user_balance", float64(stats.UserBalance), []string{}, 1)
			m.ddclient.Gauge("user_pending_in", float64(stats.UserPendingIn), []string{}, 1)
			m.ddclient.Gauge("user_pending_out", float64(stats.UserPendingOut), []string{}, 1)
			m.ddclient.Gauge("topup_pending_count", float64(stats.TopupPendingCount), []string{}, 1)
			m.ddclient.Gauge("topup_pending_value", float64(stats.TopupPendingValue), []string{}, 1)
			m.ddclient.Gauge("topup_instant_pending_count", float64(stats.TopupInstantPendingCount), []string{}, 1)
			m.ddclient.Gauge("topup_instant_pending_value", float64(stats.TopupInstantPendingValue), []string{}, 1)
		}
	}

}
