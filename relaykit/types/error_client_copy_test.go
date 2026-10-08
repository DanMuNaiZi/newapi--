package types

import (
	"net/http"
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
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

func TestNewAPIErrorClientCopySeparatesNestedErrorDetails(t *testing.T) {
	code := map[string]any{
		"details": []any{map[string]any{"model": "upstream-model"}},
		"typed":   map[string][]string{"keys": {"upstream-key"}},
	}
	original := WithOpenAIError(OpenAIError{Message: "rejected", Type: "upstream_error", Code: code}, http.StatusBadRequest)
	before, err := kitutil.Marshal(original.ToOpenAIError())
	require.NoError(t, err)

	clientCopy := original.ClientCopy()
	clientCode, ok := clientCopy.ToOpenAIError().Code.(map[string]any)
	require.True(t, ok)
	clientCode["details"].([]any)[0].(map[string]any)["model"] = "requested-model"
	clientCode["typed"].(map[string][]string)["keys"][0] = "masked-key"
	clientCode["extra"] = "client-only"

	after, err := kitutil.Marshal(original.ToOpenAIError())
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
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
