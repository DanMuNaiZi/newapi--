package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserPreviewTokenRejectsTamperingAndExpiry(t *testing.T) {
	previous := common.SessionSecret
	common.SessionSecret = "user-preview-test-secret"
	t.Cleanup(func() { common.SessionSecret = previous })

	now := time.Unix(1_800_000_000, 0)
	token, claims, err := IssueUserPreviewToken(10, 20, now)
	require.NoError(t, err)
	verified, err := VerifyUserPreviewToken(token, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, claims, verified)

	replacement := "A"
	if strings.HasSuffix(token, "A") {
		replacement = "B"
	}
	tampered := token[:len(token)-1] + replacement
	_, err = VerifyUserPreviewToken(tampered, now)
	assert.ErrorIs(t, err, ErrUserPreviewInvalid)

	_, err = VerifyUserPreviewToken(token, now.Add(UserPreviewTTL))
	assert.ErrorIs(t, err, ErrUserPreviewExpired)

	futureToken, _, err := IssueUserPreviewToken(10, 20, now.Add(time.Minute))
	require.NoError(t, err)
	_, err = VerifyUserPreviewToken(futureToken, now)
	assert.ErrorIs(t, err, ErrUserPreviewExpired)
}

func TestUserPreviewTokenRejectsMalformedClaims(t *testing.T) {
	previous := common.SessionSecret
	common.SessionSecret = "user-preview-test-secret"
	t.Cleanup(func() { common.SessionSecret = previous })

	_, err := VerifyUserPreviewToken("not-a-token", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUserPreviewInvalid))
}
