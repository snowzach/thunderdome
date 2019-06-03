package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"git.coinninja.net/backend/thunderdome/conf"
)

func TestVersionGet(t *testing.T) {

	// Mock Store and server
	s, err := New()
	assert.Nil(t, err)

	response, err := s.Version(context.Background(), nil)
	assert.Nil(t, err)
	assert.Equal(t, conf.GitVersion, response.Version)

}
