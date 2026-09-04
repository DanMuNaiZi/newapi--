package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUsageRankingsValidatesPeriodAndLimit(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "non numeric limit", url: "/api/usage-rankings?limit=abc"},
		{name: "invalid period", url: "/api/usage-rankings?period=quarter"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, testCase.url, nil)

			GetUsageRankings(ctx)

			var response struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.Success)
		})
	}
}

func TestGetUsageRankingsNormalizesNonPositiveLimit(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{
		Username: "usage-ranking-controller-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		AffCode:  "usage-ranking-controller-user",
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId:           user.Id,
		Username:         user.Username,
		Type:             model.LogTypeConsume,
		Quota:            100,
		PromptTokens:     10,
		CompletionTokens: 5,
		CreatedAt:        common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/usage-rankings?period=today&limit=0", nil)
	ctx.Set("id", user.Id)

	GetUsageRankings(ctx)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			DisplayCount int `json:"display_count"`
			TotalUsers   int `json:"total_users"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 10, response.Data.DisplayCount)
	assert.Equal(t, 1, response.Data.TotalUsers)
}
