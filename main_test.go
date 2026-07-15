package main

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestSessionCookieOptionsAllowOAuthCallbacks(t *testing.T) {
	previousSecure := common.SessionCookieSecure
	common.SessionCookieSecure = true
	t.Cleanup(func() {
		common.SessionCookieSecure = previousSecure
	})

	options := sessionCookieOptions()

	assert.Equal(t, http.SameSiteLaxMode, options.SameSite)
	assert.True(t, options.HttpOnly)
	assert.True(t, options.Secure)
	assert.Equal(t, 2592000, options.MaxAge)
}
