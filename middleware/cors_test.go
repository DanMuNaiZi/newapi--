package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSAllowsBearerClientsWithoutAdvertisingCredentialedWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())
	router.GET("/v1/models", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.Header.Set("Origin", "https://client.example")
	request.Header.Set("Authorization", "Bearer test-token")
	router.ServeHTTP(recorder, request)

	assert.Equal(t, "*", recorder.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Credentials"))
}
