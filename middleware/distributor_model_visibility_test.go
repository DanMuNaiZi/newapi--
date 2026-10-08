package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetModelRequestKeepsClientModelForResponsesCompact(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/responses/compact",
		strings.NewReader(`{"model":"gpt-5.6-sol","input":[]}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	request, shouldSelectChannel, err := getModelRequest(ctx)

	require.NoError(t, err)
	require.True(t, shouldSelectChannel)
	require.NotNil(t, request)
	require.Equal(t, "gpt-5.6-sol", request.RequestModelName)
	// Compact routing no longer mutates the model name. The client model is
	// retained as both the routing and request identity; any provider mapping
	// is applied later by the relay adapter.
	require.Equal(t, "gpt-5.6-sol", request.Model)
}
