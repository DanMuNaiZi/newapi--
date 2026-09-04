package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetModelRequestKeepsClientModelWhenResponsesCompactAddsSuffix(t *testing.T) {
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
	require.Equal(t, "gpt-5.6-sol-openai-compact", request.Model)
}
