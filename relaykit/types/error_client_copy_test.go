package types

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIErrorClientCopyDoesNotMutateOriginal(t *testing.T) {
	original := WithOpenAIError(OpenAIError{
		Message:  "upstream rejected model",
		Type:     "invalid_request_error",
		Code:     "unsupported_model",
		Metadata: []byte(`{"secret":"upstream-metadata"}`),
	}, http.StatusBadRequest)

	clientCopy := original.ClientCopy()
	require.NotSame(t, original, clientCopy)
	clientCopy.SetOpenAIClientError(
		"upstream unavailable",
		"upstream_error",
		ErrorCodeUpstreamServiceUnavailable,
		http.StatusServiceUnavailable,
	)
	clientCopy.ClearMetadata()

	assert.Equal(t, http.StatusBadRequest, original.StatusCode)
	assert.Equal(t, http.StatusBadRequest, original.GetOriginalStatusCode())
	assert.Equal(t, "upstream rejected model ({\"secret\":\"upstream-metadata\"})", original.Error())
	assert.NotEmpty(t, original.Metadata)

	assert.Equal(t, http.StatusServiceUnavailable, clientCopy.StatusCode)
	assert.Equal(t, http.StatusBadRequest, clientCopy.GetOriginalStatusCode())
	assert.Equal(t, "upstream unavailable", clientCopy.Error())
	assert.Empty(t, clientCopy.Metadata)
	assert.Equal(t, OpenAIError{
		Message: "upstream unavailable",
		Type:    "upstream_error",
		Code:    ErrorCodeUpstreamServiceUnavailable,
	}, clientCopy.ToOpenAIError())
}

func TestNewAPIErrorSetMessageUpdatesStructuredRelayError(t *testing.T) {
	apiErr := WithClaudeError(ClaudeError{
		Type:    "upstream_error",
		Message: "before",
	}, http.StatusBadGateway)

	apiErr.SetMessage("after")

	assert.Equal(t, "after", apiErr.Error())
	assert.Equal(t, ClaudeError{Type: "upstream_error", Message: "after"}, apiErr.ToClaudeError())
}
