package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeUpstreamErrorMessageRemovesCredentialsAndRawBody(t *testing.T) {
	message := `request failed: Authorization: Bearer sk-live-secret, api_key=AIza-secret, access_token="access-secret", url=https://api.example.com/v1/chat?key=query-secret, body: {"token":"body-secret","prompt":"private"}`

	sanitized := SanitizeUpstreamErrorMessage(message, "sk-live-secret", "access-secret")

	assert.Contains(t, sanitized, "request failed")
	assert.Contains(t, sanitized, "body: [omitted]")
	assert.NotContains(t, sanitized, "sk-live-secret")
	assert.NotContains(t, sanitized, "AIza-secret")
	assert.NotContains(t, sanitized, "access-secret")
	assert.NotContains(t, sanitized, "query-secret")
	assert.NotContains(t, sanitized, "body-secret")
	assert.NotContains(t, sanitized, "private")
}

func TestSanitizeUpstreamErrorMessageRemovesRawBodyWithEqualsDelimiter(t *testing.T) {
	message := `upstream failed, response_body={"token":"raw-secret","detail":"private"}`

	sanitized := SanitizeUpstreamErrorMessage(message)

	assert.Contains(t, sanitized, "body: [omitted]")
	assert.NotContains(t, sanitized, "raw-secret")
	assert.NotContains(t, sanitized, "private")
}

func TestSanitizeUpstreamErrorMessageKeepsDiagnosticMessage(t *testing.T) {
	message := "The 'gpt-5.6-sol' model is not supported when using Codex with a ChatGPT account."

	assert.Equal(t, message, SanitizeUpstreamErrorMessage(message))
}

func TestSanitizeUpstreamErrorMessageOmitsTrailingRawMetadata(t *testing.T) {
	message := `upstream failure ({"token":"raw-secret","detail":"private body"})`

	sanitized := SanitizeUpstreamErrorMessage(message)

	assert.Equal(t, "upstream failure [metadata omitted]", sanitized)
}

func TestSanitizeUpstreamErrorMessageMasksQuotedCredentialFields(t *testing.T) {
	message := `upstream said {"token":"unknown-secret","authorization":"Bearer unknown-bearer"} then failed`

	sanitized := SanitizeUpstreamErrorMessage(message)

	assert.NotContains(t, sanitized, "unknown-secret")
	assert.NotContains(t, sanitized, "unknown-bearer")
	assert.Contains(t, sanitized, `"token":***`)
}

func TestNormalizeChannelErrorMessageCollapsesVolatileRequestIds(t *testing.T) {
	first := NormalizeChannelErrorMessage(" upstream failed   request_id=req-123 ")
	second := NormalizeChannelErrorMessage("upstream failed request_id=req-999")

	require.NotEmpty(t, first)
	assert.Equal(t, first, second)
	assert.False(t, strings.Contains(first, "req-123"))
}
