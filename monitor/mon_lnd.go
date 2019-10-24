package monitor

import (
	"context"
	"time"

	"git.coinninja.net/backend/thunderdome/conf"
	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"
)

type LNDStats struct {
	TotalBalance          int64 `json:"total_balance"`
	ConfirmedBalance      int64 `json:"confirmed_balance"`
	UnconfirmedBalance    int64 `json:"uncomfirmed_balance"`
	ChannelBalance        int64 `json:"channel_balance"`
	ChannelPendingBalance int64 `json:"channel_pending_balance"`
	LocalBalance          int64 `json:"local_balance"`
	RemoteBalance         int64 `json:"remote_balance"`
	ChannelCount          int   `json:"channel_count"`
}

func (m *Monitor) MonitorLND() {

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
		stats.TotalBalance = walletBalanceResponse.TotalBalance
		stats.ConfirmedBalance = walletBalanceResponse.ConfirmedBalance
		stats.UnconfirmedBalance = walletBalanceResponse.UnconfirmedBalance

		channelBalanceResponse, err := m.lclient.ChannelBalance(ctx, &lnrpc.ChannelBalanceRequest{})
		if err != nil {
			m.logger.Errorw("Could not get channel balance", "error", err)
			continue
		}
		stats.ChannelBalance = channelBalanceResponse.Balance
		stats.ChannelPendingBalance = channelBalanceResponse.PendingOpenBalance

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
		m.logger.Debugw("LND Stats", zap.Any("lnd_stats", stats))

		if m.ddclient != nil {
			m.ddclient.Gauge("lnd.total_balance", float64(stats.TotalBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.confirmed_balance", float64(stats.ConfirmedBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.unconfirmed_balance", float64(stats.UnconfirmedBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.channel_balance", float64(stats.ChannelBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.channel_pending_balance", float64(stats.ChannelPendingBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.local_balance", float64(stats.LocalBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.remote_balance", float64(stats.RemoteBalance), []string{}, 1)
			m.ddclient.Gauge("lnd.channel_count", float64(stats.ChannelCount), []string{}, 1)
		}
	}

}
