package postgres

import (
	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (suite *DBTestSuite) TestProcessLedgerRecordIn() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 0)

	// Check basic storage of the record
	lr1 := &tdrpc.LedgerRecord{
		Id:            "tr1",
		AccountId:     a1.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.LIGHTNING,
		Direction:     tdrpc.IN,
		Value:         10,
		NetworkFee:    2,
		ProcessingFee: 3,
		Memo:          "memo-tr1",
		Request:       "request-tr1",
	}

	err := suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Fetch the record back out to ensure it matches
	lr2, err := suite.client.GetLedgerRecord(suite.ctx, lr1.Id, lr1.Direction)
	suite.Nil(err)
	suite.Equal(lr1, lr2)

	// Check the current balance
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(0))   // No Balance
	suite.Equal(a1.PendingIn, int64(0)) // Make sure PendingIn = 0 - Not adjusted

	// Complete the inbound transaction
	lr1.Status = tdrpc.COMPLETED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Check the current balance -
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(10))  // No Balance
	suite.Equal(a1.PendingIn, int64(0)) // Make sure PendingIn = 0

	// Atempt to set it back to pending should fail
	lr1.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.NotNil(err)

	a2 := suite.newTestAccount("testuser2", 0)

	// Check basic storage of the record
	lr2 = &tdrpc.LedgerRecord{
		Id:            "tr2",
		AccountId:     a2.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.LIGHTNING,
		Direction:     tdrpc.IN,
		Value:         10,
		NetworkFee:    2,
		ProcessingFee: 3,
		Memo:          "memo-tr2",
		Request:       "request-tr2",
	}

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0)) // Make sure PendingIn = 0 - Not adjusted
	suite.Equal(a2.Balance, int64(0))   // Make sure Balance = 0

	// Fail the request - this essentially makes it no longer exist
	lr2.Status = tdrpc.FAILED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.Balance, int64(0))   // Make sure Balance = 0
	suite.Equal(a2.PendingIn, int64(0)) // Make sure PendingIn = 0 // request removed

	// Set it back to pending
	lr2.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance again, should show pending once more
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0)) // Make sure PendingIn = 0 - Not adjusted
	suite.Equal(a2.Balance, int64(0))   // Make sure Balance = 0

	// This time complete the transaction but with a lesser value
	lr2.Status = tdrpc.COMPLETED
	lr2.Value = 5
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance, should show lesser balance
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingIn, int64(0)) // Make sure PendingIn = 0
	suite.Equal(a2.Balance, int64(5))   // Make sure Balance = 5

}

func (suite *DBTestSuite) TestProcessLedgerRecordOut() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 10)

	// Check basic storage of the record
	lr1 := &tdrpc.LedgerRecord{
		Id:            "tr1",
		AccountId:     a1.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.LIGHTNING,
		Direction:     tdrpc.OUT,
		Value:         20,
		NetworkFee:    1,
		ProcessingFee: 2,
		Memo:          "memo-tr1",
		Request:       "request-tr1",
	}

	// Should be not enough funds
	err := suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.NotNil(err)

	_, err = suite.client.GetLedgerRecord(suite.ctx, lr1.Id, lr1.Direction)
	suite.Equal(err, store.ErrNotFound)

	// Set the value to 5, an amount we have
	lr1.Value = 5
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Fetch the record back out to ensure it matches
	lr2, err := suite.client.GetLedgerRecord(suite.ctx, lr1.Id, lr1.Direction)
	suite.Nil(err)
	suite.Equal(lr1, lr2)

	// Check the current balance
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(2))    // = 5 - 1 - 2
	suite.Equal(a1.PendingOut, int64(8)) // Make sure PendingOut = 5 + 1 + 2

	// Complete the outbound transaction
	lr1.Status = tdrpc.COMPLETED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Check the current balance -
	a1, err = suite.client.GetAccountByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(2))   // = 5 - 1 - 2
	suite.Equal(a1.PendingIn, int64(0)) // back to 0

	// Atempt to set it back to pending should fail
	lr1.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.NotNil(err)

	a2 := suite.newTestAccount("testuser2", 15)

	// Check basic storage of the record
	lr2 = &tdrpc.LedgerRecord{
		Id:            "tr2",
		AccountId:     a2.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.LIGHTNING,
		Direction:     tdrpc.OUT,
		Value:         10,
		NetworkFee:    1,
		ProcessingFee: 2,
		Memo:          "memo-tr2",
		Request:       "request-tr2",
	}

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingOut, int64(13)) // Make sure PendingOut = 10 - 1 - 2
	suite.Equal(a2.Balance, int64(2))     // Make sure Balance = 2

	// Fail the request - this essentially makes it no longer exist
	lr2.Status = tdrpc.FAILED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.Balance, int64(15))   // Make sure Balance = 12
	suite.Equal(a2.PendingOut, int64(0)) // Make sure PendingOut = 0 // request removed

	// Set it back to pending
	lr2.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance again, should show pending once more
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingOut, int64(13)) // Make sure PendingOut = 12
	suite.Equal(a2.Balance, int64(2))     // Make sure Balance = 2

	// This time complete the transaction but with a lesser value
	lr2.Status = tdrpc.COMPLETED
	lr2.Value = 5
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance, should show lesser balance
	a2, err = suite.client.GetAccountByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.PendingOut, int64(0)) // Make sure PendingOut = 0
	suite.Equal(a2.Balance, int64(7))    // Make sure Balance = 7

}
