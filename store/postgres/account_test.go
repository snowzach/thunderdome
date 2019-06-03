package postgres

import (
	"github.com/stretchr/testify/assert"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (suite *DBTestSuite) newTestAccount(id string, balance int64) *tdrpc.Account {
	account, err := suite.client.AccountSave(suite.ctx, &tdrpc.Account{
		Id:         id,
		Address:    "address:" + id,
		Balance:    balance,
		BalanceIn:  0,
		BalanceOut: 0,
	})
	assert.Nil(suite.T(), err, "Could not create test account")

	return account
}

func (suite *DBTestSuite) TestAccount() {

	// Create a test account
	a1 := suite.newTestAccount("testuser1", 0)

	// Fetch it from the database and compare
	a2, err := suite.client.AccountGetByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1, a2)

	// Update the balance
	a1.Balance = 1000
	a1.BalanceIn = 2000
	a1.BalanceOut = 2000
	_, err = suite.client.AccountSave(suite.ctx, a1)
	suite.Nil(err)

	// Fetch and compare again
	a2, err = suite.client.AccountGetByID(suite.ctx, a1.Id)
	suite.Nil(err)
	suite.Equal(a1, a2)

	// Find it by address
	a3, err := suite.client.AccountGetByAddress(suite.ctx, a1.Address)
	suite.Nil(err)
	suite.Equal(a1, a3)

	// Check for missing ID
	_, err = suite.client.AccountGetByID(suite.ctx, "missingid")
	suite.Equal(store.ErrNotFound, err)

	// Check for missing address
	_, err = suite.client.AccountGetByAddress(suite.ctx, "missingaddress")
	suite.Equal(store.ErrNotFound, err)

}
