package monitor

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/store"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (m *Monitor) MonitorLNDChan() {

	// Handle shutting down
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-conf.Stop.Chan()
		cancel()
	}()

	// Fetch the current backup
	snapshot, err := m.lclient.ExportAllChannelBackups(ctx, &lnrpc.ChanBackupExportRequest{})
	if err != nil {
		m.logger.Fatalw("LND ExportAllChannelBackups Error", "monitor", "lnd_chan", "error", err)
	}

	cb, err := m.cbstore.GetLastChanBackup(ctx)
	if err == store.ErrNotFound {
		// No backup, always store
		_, err = m.cbstore.StoreChanBackup(ctx, getChanBackupSnapshotFundingTXIDs(snapshot), snapshot.MultiChanBackup.MultiChanBackup)
		if err != nil {
			m.logger.Fatalw("Could not StoreChanBackup", "monitor", "lnd_chan", "error", err)
		}
	} else if err != nil {
		m.logger.Fatalw("LND GetLastChanBackup Error", "monitor", "lnd_chan", "error", err)
	} else {
		// We fetched an existing channel backup, validate it
		_, err := m.lclient.VerifyChanBackup(ctx, &lnrpc.ChanBackupSnapshot{
			MultiChanBackup: &lnrpc.MultiChanBackup{
				MultiChanBackup: cb.Data,
			},
		})
		// Failed validation, perform backup
		if err != nil {
			m.logger.Infow("Channel Backup fails validation, performing backup", "error", err)
			_, err = m.cbstore.StoreChanBackup(ctx, getChanBackupSnapshotFundingTXIDs(snapshot), snapshot.MultiChanBackup.MultiChanBackup)
			if err != nil {
				m.logger.Fatalw("Could not StoreChanBackup", "monitor", "lnd_chan", "error", err)
			}
		} else {
			m.logger.Debugw("Channel backup passes validation", "monitor", "lnd_chan")

			// Check to make sure the funding TXIDs list matches
			if cb.FundingTXIDs != getChanBackupSnapshotFundingTXIDs(snapshot) {
				m.logger.Infow("Channel Backup funding TXID mismatch, performing backup")
				_, err = m.cbstore.StoreChanBackup(ctx, getChanBackupSnapshotFundingTXIDs(snapshot), snapshot.MultiChanBackup.MultiChanBackup)
				if err != nil {
					m.logger.Fatalw("Could not StoreChanBackup", "monitor", "lnd_chan", "error", err)
				}
			} else {
				m.logger.Infow("Channel Backup Current", "monitor", "lnd_chan")
			}
		}
	}

	// Connect to the channel stream
	conf.Stop.Add(1)
	chanBackupClient, err := m.lclient.SubscribeChannelBackups(ctx, &lnrpc.ChannelBackupSubscription{})
	if err != nil {
		m.logger.Fatalw("Could not SubscribeChannelBackups", "monitor", "lnd_chan", "error", err)
	}

	m.logger.Infow("Listening for channel backups...", "monitor", "lnd_chan")

	for !conf.Stop.Bool() {

		// Get the next message
		snapshot, err = chanBackupClient.Recv()
		if err == io.EOF {
			m.logger.Fatalw("LND Chan Backup EOF", "monitor", "lnd_chan")
			continue
		} else if status.Code(err) == codes.Canceled {
			m.logger.Info("LND Chan Backup Shutting Down")
			break
		} else if err != nil {
			m.logger.Fatalw("LND Chan Backup Failure", "monitor", "lnd_chan", "error", err)
		}

		// Store the channel backup
		_, err = m.cbstore.StoreChanBackup(ctx, getChanBackupSnapshotFundingTXIDs(snapshot), snapshot.MultiChanBackup.MultiChanBackup)
		if err != nil {
			m.logger.Fatalw("Could not StoreChanBackup", "monitor", "lnd_chan", "error", err)
		}

		m.logger.Infow("Backed Up Channel Snapshot Change", "monitor", "lnd_chan")

	}

	_ = chanBackupClient.CloseSend()
	conf.Stop.Done()

}

// Calculate a repeatable hash of channel points so we can determine if it has changed
func getChanBackupSnapshotFundingTXIDs(snapshot *lnrpc.ChanBackupSnapshot) string {

	fundingTXIDs := make([]string, 0)
	for _, cp := range snapshot.MultiChanBackup.ChanPoints {
		switch txid := cp.FundingTxid.(type) {
		case *lnrpc.ChannelPoint_FundingTxidBytes:
			hash, _ := chainhash.NewHash(txid.FundingTxidBytes)
			fundingTXIDs = append(fundingTXIDs, fmt.Sprintf("%s:%d", hash.String(), cp.OutputIndex))
		case *lnrpc.ChannelPoint_FundingTxidStr:
			fundingTXIDs = append(fundingTXIDs, fmt.Sprintf("%s:%d", txid, cp.OutputIndex))
		}
	}

	// Sort them to ensure it's repeatable
	sort.Strings(fundingTXIDs)

	return strings.Join(fundingTXIDs, ",")

}
