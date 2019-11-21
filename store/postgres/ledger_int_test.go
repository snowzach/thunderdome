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
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id, lr1)
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
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id, lr1)
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

	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a2.PendingOut, int64(7)) // Make sure PendingOut = 7
	suite.Equal(a2.Balance, int64(3))    // Make sure Balance = 3

	// Should succeed but for different amount
	lr2.Value = 6
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id, lr2)
	suite.Nil(err)

	// Check the balance, should show lesser balance
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a1.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a1.Balance, int64(6))    // Make sure Balance = 6

	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a2.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a2.Balance, int64(4))    // Make sure Balance = 4

}

func (suite *DBTestSuite) TestProcessLedgerRecordInternalFees() {

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
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id, lr1)
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
		Memo:      "memo-lr2",
		Request:   "request-lr2",
	}

	// Setup as failed
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Will fail for wrong state
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id, lr1)
	suite.NotNil(err)

	// Set for right state, not enough funds
	lr2.Status = tdrpc.PENDING
	lr2.NetworkFee = 3
	lr2.ProcessingFee = 4
	lr2.Value = 7

	// Update record - not enough funds
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.NotNil(err)

	// Set for right state with enough funds
	lr2.Status = tdrpc.PENDING
	lr2.NetworkFee = 1
	lr2.ProcessingFee = 1
	lr2.Value = 7

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a2.PendingOut, int64(9)) // Make sure PendingOut = 9
	suite.Equal(a2.Balance, int64(1))    // Make sure Balance = 1

	// Should succeed but for different amount
	lr2.Value = 6
	_, err = suite.client.ProcessInternal(suite.ctx, lr1.Id, lr2)
	suite.Nil(err)

	// Check the balance, should show lesser balance
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a1.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a1.Balance, int64(6))    // Make sure Balance = 6

	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0))  // Make sure PendingIn = 0
	suite.Equal(a2.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a2.Balance, int64(2))    // Make sure Balance = 2

}

func (suite *DBTestSuite) TestProcessLedgerPreAuthInt() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 10)
	a2 := suite.newTestAccount("testuser2", 10)

	// Create the incoming request
	receiver := tdrpc.LedgerRecord{
		Id:            "TRINTID",
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
		AccountId:     a2.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.LIGHTNING,
		Direction:     tdrpc.IN,
		Value:         7,
		NetworkFee:    0,
		ProcessingFee: 0,
		Memo:          "memo-tr1",
		Request:       "Some Request ID",
	}

	err := suite.client.ProcessLedgerRecord(suite.ctx, &receiver)
	suite.Nil(err)

	// Create the pre-auth request
	palr := &tdrpc.LedgerRecord{
		Id:            "PreAuthIntRandomeId1",
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
		AccountId:     a1.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.LIGHTNING,
		Direction:     tdrpc.OUT,
		Value:         20,
		NetworkFee:    0,
		ProcessingFee: 0,
		Memo:          "memo-preauth",
		Request:       tdrpc.PreAuthRequest,
	}

	// Fail insufficient funds
	err = suite.client.ProcessLedgerRecord(suite.ctx, palr)
	suite.Equal(tdrpc.ErrInsufficientFunds, err)

	// Complete pre-auth for 7 sats
	palr.Value = 7
	err = suite.client.ProcessLedgerRecord(suite.ctx, palr)
	suite.Nil(err)

	// Make a copy, change the fields
	sender := receiver
	sender.Id += tdrpc.InternalIdSuffix
	sender.AccountId = a1.Id
	sender.Direction = tdrpc.OUT
	sender.Status = tdrpc.PENDING
	sender.Value = 12

	// Prepare to pay the internal record, update the preauth to the sender.Id
	err = suite.client.UpdateLedgerRecordID(suite.ctx, palr.Id, sender.Id)
	suite.Nil(err)

	// Create the record, will notr change anything so it will succeed
	err = suite.client.ProcessLedgerRecord(suite.ctx, &sender)
	suite.Nil(err)

	// Fail for insufficient funds
	_, err = suite.client.ProcessInternal(suite.ctx, receiver.Id, &sender)
	suite.NotNil(err)

	sender.Value = 5
	sender.NetworkFee = 1
	sender.ProcessingFee = 1
	lr, err := suite.client.ProcessInternal(suite.ctx, receiver.Id, &sender)
	suite.Nil(err)

	// This is what it should look like
	lr.CreatedAt = sender.CreatedAt
	lr.UpdatedAt = sender.UpdatedAt
	sender.Status = tdrpc.COMPLETED // Now completed
	suite.Equal(sender, *lr)

	// Try to pay it again, it should fail
	_, err = suite.client.ProcessInternal(suite.ctx, receiver.Id, &sender)
	suite.NotNil(err)

	// Check the balance, should show lesser balance
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a1.Balance, int64(3))    // Make sure Balance = 5

	// Check the balance, should show lesser balance
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a2.Balance, int64(15))   // Make sure Balance = 15

	// Check the LR for the right fields on the sender side
	lr, err = suite.client.GetLedgerRecord(suite.ctx, sender.Id, tdrpc.OUT)
	suite.Nil(err)
	// These fields are set by the database
	sender.CreatedAt = lr.CreatedAt
	sender.UpdatedAt = lr.UpdatedAt
	sender.Status = tdrpc.COMPLETED
	suite.Equal(sender, *lr)

	// Check the LR for the Pre Auth
	_, err = suite.client.GetLedgerRecord(suite.ctx, "PreAuthIntRandomeId1", tdrpc.OUT)
	suite.NotNil(err)
}
