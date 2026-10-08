package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserPreviewReadOnlySessionBoundaries(t *testing.T) {
	actor, _ := setupSecurityEnrollmentTest(t)
	require.NoError(t, model.DB.Model(actor).Update("aff_code", "preview-administrator").Error)
	require.NoError(t, model.DB.Model(actor).Update("role", common.RoleRootUser).Error)
	require.NoError(t, model.PublishUserAuthCache(actor.Id))
	bundle, err := service.CreateLoginSession(actor.Id, "password", "127.0.0.1", "preview-test")
	require.NoError(t, err)
	target := &model.User{Username: "preview-target", Password: "placeholder", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AuthVersion: 1, Group: "default", Email: "private@example.com", GitHubId: "private-github", AffCode: "private-invite", Quota: 123}
	require.NoError(t, model.DB.Create(target).Error)
	require.NoError(t, model.PublishUserAuthCache(target.Id))
	other := &model.User{Username: "other-root", Password: "placeholder", Role: common.RoleRootUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "other-root"}
	require.NoError(t, model.DB.Create(other).Error)
	otherBundle, err := service.CreateLoginSession(other.Id, "password", "127.0.0.1", "preview-test")
	require.NoError(t, err)
	require.NoError(t, model.UpdateUserAccessToken(actor.Id, "preview-root-pat"))

	router := gin.New()
	router.POST("/api/user/:id/preview", middleware.RootAuth(), StartUserPreview)
	router.GET("/api/user/self", middleware.UserAuth(), GetSelf)
	mutations := 0
	router.POST("/api/user/self", middleware.UserAuth(), func(c *gin.Context) { mutations++; c.Status(http.StatusOK) })
	router.GET("/api/user/token/status", middleware.UserAuth(), GetAccessTokenStatus)
	router.GET("/api/channel/", middleware.RootAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, path := range []string{
		"/api/lottery/self", "/api/lottery/plans/:id/participants/page",
		"/api/referral-campaign/self", "/api/public-pool/contributions/self", "/api/usage-rankings",
		"/api/user/self/sessions", "/api/user/self/security", "/api/user/sessions",
		"/api/user/topup/info", "/api/user/manual-wechat/info",
	} {
		router.GET(path, middleware.UserAuth(), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"user_id": c.GetInt("id")}) })
	}
	router.GET("/api/user/aff", middleware.UserAuth(), GetAffCode)
	request := func(method, path, authorization, preview string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("Authorization", "Bearer "+authorization)
		r.Header.Set("New-Api-User-Preview", preview)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	start := request("POST", fmt.Sprintf("/api/user/%d/preview", target.Id), bundle.AccessToken, "")
	var issued struct {
		Success bool `json:"success"`
		Data    struct {
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(start.Body.Bytes(), &issued))
	require.True(t, issued.Success, start.Body.String())
	require.NotEmpty(t, issued.Data.Token)
	assert.InDelta(t, time.Now().Add(15*time.Minute).Unix(), issued.Data.ExpiresAt, 2)

	self := request("GET", "/api/user/self", bundle.AccessToken, issued.Data.Token)
	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(self.Body.Bytes(), &response))
	require.True(t, response.Success, self.Body.String())
	assert.EqualValues(t, target.Id, response.Data["id"])
	assert.EqualValues(t, 123, response.Data["quota"])
	for _, field := range []string{"email", "github_id", "access_token", "password", "setting", "aff_code"} {
		assert.NotContains(t, response.Data, field)
	}
	for _, path := range []string{
		"/api/lottery/self", "/api/lottery/plans/123/participants/page", "/api/referral-campaign/self",
		"/api/public-pool/contributions/self", "/api/usage-rankings",
	} {
		preview := request("GET", path, bundle.AccessToken, issued.Data.Token)
		assert.Equal(t, http.StatusOK, preview.Code, path)
		assert.JSONEq(t, fmt.Sprintf(`{"user_id":%d}`, target.Id), preview.Body.String(), path)
	}
	for _, path := range []string{"/api/user/self/sessions", "/api/user/self/security", "/api/user/sessions", "/api/user/topup/info", "/api/user/manual-wechat/info"} {
		assert.Equal(t, http.StatusForbidden, request("GET", path, bundle.AccessToken, issued.Data.Token).Code, path)
	}
	// Reading a preview must not create an invitation code for the target.
	require.NoError(t, model.DB.Model(target).Update("aff_code", "").Error)
	assert.Equal(t, http.StatusOK, request("GET", "/api/user/aff", bundle.AccessToken, issued.Data.Token).Code)
	var unchanged model.User
	require.NoError(t, model.DB.First(&unchanged, target.Id).Error)
	assert.Empty(t, unchanged.AffCode)

	expired, _, err := service.IssueUserPreviewToken(actor.Id, target.Id, time.Now().Add(-16*time.Minute))
	require.NoError(t, err)
	for _, tc := range []struct {
		name, method, path, auth, token string
		status                          int
	}{
		{"mutation", "POST", "/api/user/self", bundle.AccessToken, issued.Data.Token, 403},
		{"credentials", "GET", "/api/user/token/status", bundle.AccessToken, issued.Data.Token, 403},
		{"admin route", "GET", "/api/channel/", bundle.AccessToken, issued.Data.Token, 403},
		{"tampered", "GET", "/api/user/self", bundle.AccessToken, "x" + issued.Data.Token, 401},
		{"expired", "GET", "/api/user/self", bundle.AccessToken, expired, 401},
		{"other actor", "GET", "/api/user/self", otherBundle.AccessToken, issued.Data.Token, 401},
		{"PAT", "GET", "/api/user/self", "preview-root-pat", issued.Data.Token, 401},
	} {
		t.Run(tc.name, func(t *testing.T) { assert.Equal(t, tc.status, request(tc.method, tc.path, tc.auth, tc.token).Code) })
	}
	assert.Zero(t, mutations)
	for _, role := range []int{common.RoleAuthorizedAdmin, common.RoleAdminUser, common.RoleRootUser} {
		require.NoError(t, model.DB.Model(target).Update("role", role).Error)
		require.NoError(t, model.PublishUserAuthCache(target.Id))
		assert.Equal(t, 403, request("GET", "/api/user/self", bundle.AccessToken, issued.Data.Token).Code)
	}
	require.NoError(t, model.DB.Model(target).Updates(map[string]any{"role": common.RoleCommonUser, "status": common.UserStatusDisabled}).Error)
	require.NoError(t, model.PublishUserAuthCache(target.Id))
	assert.Equal(t, 403, request("GET", "/api/user/self", bundle.AccessToken, issued.Data.Token).Code)
	ordinary := request("GET", "/api/user/self", bundle.AccessToken, "")
	require.NoError(t, common.Unmarshal(ordinary.Body.Bytes(), &response))
	assert.EqualValues(t, actor.Id, response.Data["id"], "actor session remains unchanged")
	var events []model.AuditLog
	require.NoError(t, model.LOG_DB.Where("action LIKE ?", "user.preview_%").Find(&events).Error)
	assert.GreaterOrEqual(t, len(events), 10)
	for _, event := range events {
		encoded, err := common.Marshal(event.Other)
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), issued.Data.Token)
		assert.NotContains(t, string(encoded), bundle.AccessToken)
	}

	// Issuing a preview is a browser-session action, never a PAT capability.
	require.NoError(t, model.DB.Model(target).Update("status", common.UserStatusEnabled).Error)
	patStart := request("POST", fmt.Sprintf("/api/user/%d/preview", target.Id), "preview-root-pat", "")
	assert.Equal(t, http.StatusUnauthorized, patStart.Code)
	assert.NotContains(t, patStart.Body.String(), `"token"`)
	revoked, err := model.RevokeUserSession(actor.Id, bundle.Session.SID, "preview test revoked")
	require.NoError(t, err)
	require.True(t, revoked)
	assert.Equal(t, http.StatusUnauthorized, request("GET", "/api/user/self", bundle.AccessToken, issued.Data.Token).Code)
}
