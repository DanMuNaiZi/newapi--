package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReferralReviewRoutesSeparateReadAndOperatePermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB, previousLog := model.DB, model.LOG_DB
	previousRedis := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open("file:referral_route_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLog
		common.RedisEnabled = previousRedis
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.CasbinRule{}, &model.AuthzRole{}, &model.ReferralCampaign{}, &model.ReferralCampaignEvent{}, &model.RewardGrant{}, &model.Log{}))
	require.NoError(t, authz.Init(db))
	for _, account := range []struct {
		id, role    int
		name, token string
	}{
		{201, common.RoleCommonUser, "referral-reader-user", "referral-user-test-token"},
		{202, common.RoleAuthorizedAdmin, "referral-readonly-admin", "referral-read-test-token"},
		{203, common.RoleAuthorizedAdmin, "referral-reviewer", "referral-review-test-token"},
	} {
		token := account.token
		require.NoError(t, db.Create(&model.User{Id: account.id, Username: account.name, Role: account.role, Status: common.UserStatusEnabled, AccessToken: &token, AffCode: account.name}).Error)
	}
	require.NoError(t, authz.SetUserPermissionsForRole(202, common.RoleAuthorizedAdmin, authz.PermissionsMap{"referral_campaign": {"read": true}}))
	require.NoError(t, authz.SetUserPermissionsForRole(203, common.RoleAuthorizedAdmin, authz.PermissionsMap{"referral_campaign": {"read": true, "operate": true}}))
	campaign := &model.ReferralCampaign{Id: 1, Title: "Review"}
	require.NoError(t, db.Create(campaign).Error)
	event := &model.ReferralCampaignEvent{Id: 1, CampaignId: 1, InviterUserId: 201, InviteeUserId: 203, Status: model.ReferralEventPendingReview, RegisteredAt: common.GetTimestamp() - 60, ActivationDeadline: common.GetTimestamp() + 3600, RewardSnapshotJSON: `{ "type":"quota", "quota": 100 }`}
	require.NoError(t, db.Create(event).Error)
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("referral-review-test-session-key"))))
	registerReferralCampaignRoutes(r.Group("/api"))
	for _, tc := range []struct {
		name, method, token string
		userId, status      int
	}{
		{"anonymous read", http.MethodGet, "", 0, http.StatusUnauthorized},
		{"user cannot read", http.MethodGet, "referral-user-test-token", 201, http.StatusOK},
		{"admin can read", http.MethodGet, "referral-read-test-token", 202, http.StatusOK},
		{"read is not operate", http.MethodPost, "referral-read-test-token", 202, http.StatusForbidden},
		{"reviewer can reject", http.MethodPost, "referral-review-test-token", 203, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/api/referral-campaign/admin/events/1/review", strings.NewReader(`{"decision":"reject","remark":"checked"}`))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
				req.Header.Set("New-Api-User", fmt.Sprint(tc.userId))
			}
			r.ServeHTTP(recorder, req)
			assert.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			if tc.userId == 201 {
				assert.Contains(t, recorder.Body.String(), `"success":false`)
				assert.NotContains(t, recorder.Body.String(), `"event"`)
			} else if tc.status == http.StatusOK {
				assert.Contains(t, recorder.Body.String(), `"success":true`)
			}
		})
	}
	var stored model.ReferralCampaignEvent
	require.NoError(t, db.First(&stored, 1).Error)
	assert.Equal(t, "rejected", stored.ReviewDecision)
	assert.Equal(t, 203, stored.ReviewedBy)
}
