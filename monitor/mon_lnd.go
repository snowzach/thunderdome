package monitor

import (
	"context"
	"time"

	"git.coinninja.net/backend/thunderdome/conf"
	"github.com/lightningnetwork/lnd/lnrpc"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
)

type LNDStats struct {
	ConfirmedBalance int64 `json:"confirmed_balance"`
	LocalBalance     int64 `json:"local_balance"`
	RemoteBalance    int64 `json:"remote_balance"`
	ChannelCount     int   `json:"channel_count"`
}

func (m *Monitor) MonitorLND() {

	confirmedBalance := stats.Int64("lnd/confirmed_balance", "The confirmed LND wallet balance", stats.UnitDimensionless)
	localBalance := stats.Int64("lnd/local_balance", "The total local balance of the LND wallet on all channels", stats.UnitDimensionless)
	remoteBalance := stats.Int64("lnd/remote_balance", "The total remote balance of the LND wallet on all channels", stats.UnitDimensionless)
	openChannels := stats.Int64("lnd/open_channels", "The count of open channels", stats.UnitDimensionless)

	// Register the views
	if err := view.Register(
		&view.View{Name: confirmedBalance.Name(), Measure: confirmedBalance, Description: confirmedBalance.Description(), Aggregation: view.LastValue()},
		&view.View{Name: localBalance.Name(), Measure: localBalance, Description: localBalance.Description(), Aggregation: view.LastValue()},
		&view.View{Name: remoteBalance.Name(), Measure: remoteBalance, Description: remoteBalance.Description(), Aggregation: view.LastValue()},
		&view.View{Name: openChannels.Name(), Measure: openChannels, Description: openChannels.Description(), Aggregation: view.LastValue()},
	); err != nil {
		m.logger.Errorf("Could not register monitor views: %v", err)
	}

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

		var stats LNDStats

		walletBalanceResponse, err := m.lclient.WalletBalance(ctx, &lnrpc.WalletBalanceRequest{})
		if err != nil {
			m.logger.Errorw("Could not get wallet balance", "error", err)
			continue
		}
		stats.ConfirmedBalance = walletBalanceResponse.ConfirmedBalance

		// List our channels
		listChannelsResponse, err := m.lclient.ListChannels(ctx, &lnrpc.ListChannelsRequest{ActiveOnly: true, PublicOnly: true})
		if err != nil {
			m.logger.Errorw("Could not list channels", "error", err)
			continue
		}

		stats.ChannelCount = len(listChannelsResponse.Channels)

		for _, c := range listChannelsResponse.Channels {
			stats.LocalBalance += c.LocalBalance
			stats.RemoteBalance += c.RemoteBalance
		}
		m.logger.Infow("Stats", zap.Any("lnd_stats", stats))

		if m.ddclient != nil {
			m.ddclient.Gauge("lnd.confirmed_balance", float64(stats.ConfirmedBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.local_balance", float64(stats.LocalBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.remote_balance", float64(stats.RemoteBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.channel_count", float64(stats.ChannelCount), []string{}, 1)
		}
	}

}
