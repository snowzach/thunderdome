package postgres

import (
	"time"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

func (suite *DBTestSuite) TestProcessLedgerRecordIn() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 0)

	// Check basic storage of the record
	lr1 := &tdrpc.LedgerRecord{
		Id:        "tr1",
		AccountId: a1.Id,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     10,
		Memo:      "memo-tr1",
		Request:   "request-tr1",
	}

	err := suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Fetch the record back out to ensure it matches
	lr2, err := suite.client.GetLedgerRecord(suite.ctx, lr1.Id, lr1.Direction)
	suite.Nil(err)
	suite.Equal(lr1, lr2)

	// Check the current balance
	a1, err = suite.client.AccountGetByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(0))    // No Balance
	suite.Equal(a1.BalanceIn, int64(10)) // Make sure BalanceIn = 10

	// Complete the inbound transaction
	lr1.Status = tdrpc.COMPLETED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Check the current balance -
	a1, err = suite.client.AccountGetByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(10))  // No Balance
	suite.Equal(a1.BalanceIn, int64(0)) // Make sure BalanceIn = 10

	// Atempt to set it back to pending should fail
	lr1.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.NotNil(err)

	a2 := suite.newTestAccount("testuser2", 0)

	// Check basic storage of the record
	lr2 = &tdrpc.LedgerRecord{
		Id:        "tr2",
		AccountId: a2.Id,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     10,
		Memo:      "memo-tr2",
		Request:   "request-tr2",
	}

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.BalanceIn, int64(10)) // Make sure BalanceIn = 10
	suite.Equal(a2.Balance, int64(0))    // Make sure Balance = 0

	// Fail the request - this essentially makes it no longer exist
	lr2.Status = tdrpc.FAILED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.Balance, int64(0))   // Make sure Balance = 0
	suite.Equal(a2.BalanceIn, int64(0)) // Make sure BalanceIn = 0 // request removed

	// Set it back to pending
	lr2.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance again, should show pending once more
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.BalanceIn, int64(10)) // Make sure BalanceIn = 10
	suite.Equal(a2.Balance, int64(0))    // Make sure Balance = 0

	// This time complete the transaction but with a lesser value
	lr2.Status = tdrpc.COMPLETED
	lr2.Value = 5
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance, should show lesser balance
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.BalanceIn, int64(0)) // Make sure BalanceIn = 0
	suite.Equal(a2.Balance, int64(5))   // Make sure Balance = 5

}

func (suite *DBTestSuite) TestProcessLedgerRecordOut() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 10)

	// Check basic storage of the record
	lr1 := &tdrpc.LedgerRecord{
		Id:        "tr1",
		AccountId: a1.Id,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     20,
		Memo:      "memo-tr1",
		Request:   "request-tr1",
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
	a1, err = suite.client.AccountGetByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(5))    // = 5
	suite.Equal(a1.BalanceOut, int64(5)) // Make sure BalanceOut = 5

	// Complete the outbound transaction
	lr1.Status = tdrpc.COMPLETED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.Nil(err)

	// Check the current balance -
	a1, err = suite.client.AccountGetByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.Balance, int64(5))   // = 5
	suite.Equal(a1.BalanceIn, int64(0)) // back to 0

	// Atempt to set it back to pending should fail
	lr1.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr1)
	suite.NotNil(err)

	a2 := suite.newTestAccount("testuser2", 12)

	// Check basic storage of the record
	lr2 = &tdrpc.LedgerRecord{
		Id:        "tr2",
		AccountId: a2.Id,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     10,
		Memo:      "memo-tr2",
		Request:   "request-tr2",
	}

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.BalanceOut, int64(10)) // Make sure BalanceOut = 10
	suite.Equal(a2.Balance, int64(2))     // Make sure Balance = 2

	// Fail the request - this essentially makes it no longer exist
	lr2.Status = tdrpc.FAILED
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the current balance -
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.Balance, int64(12))   // Make sure Balance = 12
	suite.Equal(a2.BalanceOut, int64(0)) // Make sure BalanceOut = 0 // request removed

	// Set it back to pending
	lr2.Status = tdrpc.PENDING
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance again, should show pending once more
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.BalanceOut, int64(10)) // Make sure BalanceOut = 12
	suite.Equal(a2.Balance, int64(2))     // Make sure Balance = 2

	// This time complete the transaction but with a lesser value
	lr2.Status = tdrpc.COMPLETED
	lr2.Value = 5
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr2)
	suite.Nil(err)

	// Check the balance, should show lesser balance
	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.BalanceOut, int64(0)) // Make sure BalanceOut = 0
	suite.Equal(a2.Balance, int64(7))    // Make sure Balance = 7

}

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
		Id:        "tr1" + thunderdome.InternalIdSuffix, // MUST INCLUDE INTERNAL SUFFIX FOR OUTBOUND
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
	a1, err = suite.client.AccountGetByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1.BalanceIn, int64(0))  // Make sure BalanceIn = 0
	suite.Equal(a1.BalanceOut, int64(0)) // Make sure BalanceOut = 0
	suite.Equal(a1.Balance, int64(7))    // Make sure Balance = 7

	a2, err = suite.client.AccountGetByID(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.Equal(a2.BalanceIn, int64(0))  // Make sure BalanceIn = 0
	suite.Equal(a2.BalanceOut, int64(0)) // Make sure BalanceOut = 0
	suite.Equal(a2.Balance, int64(3))    // Make sure Balance = 3

}

func (suite *DBTestSuite) TestGetLedger() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 10)
	a2 := suite.newTestAccount("testuser2", 10)

	lr11 := &tdrpc.LedgerRecord{
		Id:        "tr1.1",
		AccountId: a1.Id,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     7,
		Memo:      "memo-tr1.1",
		Request:   "request-tr1.2",
	}

	// Update record - not enough funds
	err := suite.client.ProcessLedgerRecord(suite.ctx, lr11)
	suite.Nil(err)

	lr12 := &tdrpc.LedgerRecord{
		Id:        "tr1.2",
		AccountId: a1.Id,
		Status:    tdrpc.COMPLETED,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     3,
		Memo:      "memo-tr1.2",
		Request:   "request-tr1.2",
	}

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr12)
	suite.Nil(err)

	lr1list := []*tdrpc.LedgerRecord{lr11, lr12}

	// Create the pending record with wrong state
	lr21 := &tdrpc.LedgerRecord{
		Id:        "tr2.1",
		AccountId: a2.Id,
		Status:    tdrpc.FAILED,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     4,
		Memo:      "memo-tr21",
		Request:   "request-tr21",
	}

	// Update record - not enough funds
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr21)
	suite.Nil(err)

	lr22 := &tdrpc.LedgerRecord{
		Id:        "tr2.2",
		AccountId: a2.Id,
		Status:    tdrpc.EXPIRED,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     4,
		Memo:      "memo-tr22",
		Request:   "request-tr22",
	}
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr22)
	suite.Nil(err)

	lr2list := []*tdrpc.LedgerRecord{lr21, lr22}

	l, err := suite.client.GetLedger(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.ElementsMatch(l, lr1list)

	l, err = suite.client.GetLedger(suite.ctx, a2.Id)
	suite.Nil(err)
	suite.ElementsMatch(l, lr2list)

	lrtest, err := suite.client.GetLedgerRecord(suite.ctx, lr22.Id, lr22.Direction)
	suite.Nil(err)
	suite.Equal(lr22, lrtest)

}
