package postgres

import (
	"time"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

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

	l, err := suite.client.GetLedger(suite.ctx, map[string]string{"account_id": a1.Id}, time.Time{}, 0, -1)
	suite.Nil(err)
	suite.ElementsMatch(l, lr1list)

	l, err = suite.client.GetLedger(suite.ctx, map[string]string{"account_id": a2.Id}, time.Time{}, 0, -1)
	suite.Nil(err)
	suite.ElementsMatch(l, lr2list)

	lrtest, err := suite.client.GetLedgerRecord(suite.ctx, lr22.Id, lr22.Direction)
	suite.Nil(err)
	suite.Equal(lr22, lrtest)

}

func (suite *DBTestSuite) TestUpdateLedgerRecordID() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 10)

	err := suite.client.UpdateLedgerRecordID(suite.ctx, "abc", "123")
	suite.NotNil(err)

	lr := &tdrpc.LedgerRecord{
		Id:        "tr1",
		AccountId: a1.Id,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     2,
		Memo:      "memo-tr1",
		Request:   "request-tr1",
	}

	err = suite.client.ProcessLedgerRecord(suite.ctx, lr)
	suite.Nil(err)

	// Sucess
	err = suite.client.UpdateLedgerRecordID(suite.ctx, "tr1", "tr2")
	suite.Nil(err)

	// Check to find it
	lrAfter, err := suite.client.GetLedgerRecord(suite.ctx, "tr2", lr.Direction)
	suite.Nil(err)
	lr.Id = "tr2"
	suite.Equal(lr, lrAfter)

	// Create a third
	lr.Id = "tr3"
	err = suite.client.ProcessLedgerRecord(suite.ctx, lr)
	suite.Nil(err)

	// Already exists
	err = suite.client.UpdateLedgerRecordID(suite.ctx, "tr3", "tr2")
	suite.NotNil(err)

}
