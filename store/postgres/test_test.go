package postgres

import (
	"context"
	"testing"

	config "github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/conf" // Will initialize the config and loggers if you wish to use them
)

type DBTestSuite struct {
	suite.Suite

	ctx    context.Context
	client *Client
	logger *zap.SugaredLogger
}

// The suite ensures the database is in a sane state
func (suite *DBTestSuite) SetupSuite() {
	// Override the database name to whatever the database_test variable is
	config.SetDefault("storage.database", config.GetString("storage.database_test"))
	// Wipe the database upon startup
	config.SetDefault("storage.wipe_confirm", true)

	conf.InitLogger()

	suite.logger = zap.S().With("package", "server")
	suite.ctx = context.Background()

	// Create the database connection
	var err error
	suite.client, err = New()
	assert.Nil(suite.T(), err)
}

// Make sure the database is empty and up to date prior to the test
func (suite *DBTestSuite) SetupTest() {

	_, err := suite.client.db.Exec(`DELETE FROM ledger`)
	assert.Nil(suite.T(), err)

	_, err = suite.client.db.Exec(`DELETE FROM account`)
	assert.Nil(suite.T(), err)

}

// Run the test suite
func TestDBTestSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
