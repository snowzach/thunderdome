package postgres

import (
	"time"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (suite *DBTestSuite) TestProcessLedgerRecordInternal() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 0)
	a2 := suite.newTestAccount("testuser2", 10)

	expiresAt := time.Now().Add(time.Hour)

	lr1 := &tdrpc.LedgerRecord{
		Id:        "tr1",
		AccountId: a1.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     0,
		Memo:      "memo-tr1",
		Request:   "request-tr1",
	}

	// Setup the pending in record
	err := suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Process the payment, will fail for missing pending record
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id)
	suite.NotNil(err)

	// Create the pending record with wrong state
	lr2 := &tdrpc.LedgerRecord{
		Id:        "tr1" + tdrpc.InternalIdSuffix, // MUST INCLUDE INTERNAL SUFFIX FOR OUTBOUND
		AccountId: a2.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.FAILED,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     7,
		Memo:      "memo-tr1",
		Request:   "request-tr1",
	}

	// Setup as failed
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Will fail for wrong state
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id)
	suite.NotNil(err)

	// Set for right state, not enough funds
	lr2.Status = tdrpc.PENDING
	lr2.Value = 20

	// Update record - not enough funds
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.NotNil(err)

	// Set for right state with enough funds
	lr2.Status = tdrpc.PENDING
	lr2.Value = 7

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Should succeed
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id)
	suite.Nil(err)

	// Check the balance, should show lesser balance
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a1.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a1.Balance, int64(7))    // Make sure Balance = 7

	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a2.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a2.Balance, int64(3))    // Make sure Balance = 3

}
