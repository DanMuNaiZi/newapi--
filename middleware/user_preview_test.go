package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserPreviewPathAllowlistExcludesSensitiveAndAdminEndpoints(t *testing.T) {
	for _, path := range []string{
		"/api/user/self",
		"/api/subscription/self",
		"/api/log/self",
		"/api/lottery/self",
		"/api/lottery/plans/:id",
		"/api/lottery/plans/:id/participants/page",
		"/api/lottery/plans/:id/results/page",
		"/api/public-pool/sites",
		"/api/referral-campaign/self",
	} {
		assert.True(t, isUserPreviewPathAllowed(path), path)
	}
	for _, path := range []string{
		"/api/user/token",
		"/api/user/passkey",
		"/api/user/oauth/bindings",
		"/api/user/topup/info",
		"/api/user/topup/self",
		"/api/user/pay",
		"/api/log/",
		"/api/public-pool/admin/sites",
		"/api/lottery/admin/plans",
		"/api/lottery/internal/debug",
		"/api/lottery/:id/join",
	} {
		assert.False(t, isUserPreviewPathAllowed(path), path)
	}
}
