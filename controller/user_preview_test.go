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

func TestGetSelfRedactsSensitiveIdentityAndPaymentFieldsDuringPreview(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{
		Username:       "preview-safe-user",
		Password:       "password",
		Status:         common.UserStatusEnabled,
		Role:           common.RoleCommonUser,
		Group:          "vip",
		Quota:          123,
		Email:          "private@example.com",
		GitHubId:       "github-secret",
		OidcId:         "oidc-secret",
		StripeCustomer: "stripe-secret",
		Setting:        `{"private":"setting"}`,
		AffCode:        "preview-safe-user",
	}
	require.NoError(t, db.Create(user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	ctx.Set("id", user.Id)
	ctx.Set("role", user.Role)
	ctx.Set("preview_mode", true)

	GetSelf(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"username":"preview-safe-user"`)
	assert.Contains(t, recorder.Body.String(), `"quota":123`)
	assert.NotContains(t, recorder.Body.String(), "private@example.com")
	assert.NotContains(t, recorder.Body.String(), "github-secret")
	assert.NotContains(t, recorder.Body.String(), "oidc-secret")
	assert.NotContains(t, recorder.Body.String(), "stripe-secret")
	assert.NotContains(t, recorder.Body.String(), `"setting"`)
}

func TestGetAffCodeDoesNotCreateCodeDuringPreview(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{
		Username: "preview-aff-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Group:    "default",
	}
	require.NoError(t, db.Create(user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/aff", nil)
	ctx.Set("id", user.Id)
	ctx.Set("preview_mode", true)

	GetAffCode(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"data":""`)
	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Empty(t, stored.AffCode)
}
