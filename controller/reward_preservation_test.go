package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUpdateReferralCampaignPreservesStoredRewardSnapshot(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ReferralCampaign{}, &model.ReferralCampaignScheduleLock{}))
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })
	now := common.GetTimestamp()
	const originalSnapshot = `{"usd_exchange_rate":"7.3","quota_per_unit":"500000","quota":500000,"unit":"usd","amount":"1","type":"quota"}`
	campaign := &model.ReferralCampaign{
		Title:                   "Original campaign",
		Enabled:                 false,
		StartTime:               now + 60,
		EndTime:                 now + 3600,
		ActivationWindowSeconds: 3600,
		RewardSnapshotJSON:      originalSnapshot,
	}
	require.NoError(t, model.CreateReferralCampaign(campaign))

	common.QuotaPerUnit = 600_000
	payload := []byte(fmt.Sprintf(`{"title":"Preserved campaign","enabled":false,"start_time":%d,"end_time":%d,"activation_window_seconds":3600,"max_rewards_per_inviter":0,"total_reward_limit":0,"preserve_reward":true,"reward":{"type":"quota","amount":"999","unit":"usd"}}`, now+60, now+3600))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/referral-campaign/admin/campaigns/1", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(campaign.Id)}}
	AdminUpdateReferralCampaign(ctx)

	response := struct {
		Success bool                   `json:"success"`
		Data    model.ReferralCampaign `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.NotNil(t, response.Data.Reward)
	assert.Equal(t, "1", response.Data.Reward.InputAmount)
	assert.Equal(t, 500_000, response.Data.Reward.Quota)
	assert.Equal(t, "500000", response.Data.Reward.QuotaPerUnit)

	var stored model.ReferralCampaign
	require.NoError(t, db.First(&stored, campaign.Id).Error)
	assert.Equal(t, originalSnapshot, stored.RewardSnapshotJSON)

	var audit model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeManage).Last(&audit).Error)
	auditOther := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(audit.Other, &auditOther))
	op := auditOther["op"].(map[string]interface{})
	params := op["params"].(map[string]interface{})
	assert.Equal(t, true, params["preserve_reward"])
	assert.Equal(t, "1", params["reward_amount"])
	assert.Equal(t, float64(500_000), params["reward_quota"])

	repeatPayload := []byte(fmt.Sprintf(`{"title":"Preserved again","enabled":false,"start_time":%d,"end_time":%d,"activation_window_seconds":3600,"max_rewards_per_inviter":0,"total_reward_limit":0,"preserve_reward":true,"reward":{"type":"quota","amount":"123","unit":"usd"}}`, now+60, now+3600))
	repeatRecorder := httptest.NewRecorder()
	repeatCtx, _ := gin.CreateTestContext(repeatRecorder)
	repeatCtx.Request = httptest.NewRequest(http.MethodPut, "/api/referral-campaign/admin/campaigns/1", bytes.NewReader(repeatPayload))
	repeatCtx.Request.Header.Set("Content-Type", "application/json")
	repeatCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(campaign.Id)}}
	AdminUpdateReferralCampaign(repeatCtx)
	require.NoError(t, db.First(&stored, campaign.Id).Error)
	assert.Equal(t, originalSnapshot, stored.RewardSnapshotJSON)

	updatePayload := []byte(fmt.Sprintf(`{"title":"Recalculated campaign","enabled":false,"start_time":%d,"end_time":%d,"activation_window_seconds":3600,"max_rewards_per_inviter":0,"total_reward_limit":0,"preserve_reward":false,"reward":{"type":"quota","amount":"1","unit":"usd"}}`, now+60, now+3600))
	updateRecorder := httptest.NewRecorder()
	updateCtx, _ := gin.CreateTestContext(updateRecorder)
	updateCtx.Request = httptest.NewRequest(http.MethodPut, "/api/referral-campaign/admin/campaigns/1", bytes.NewReader(updatePayload))
	updateCtx.Request.Header.Set("Content-Type", "application/json")
	updateCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(campaign.Id)}}
	AdminUpdateReferralCampaign(updateCtx)
	require.NoError(t, db.First(&stored, campaign.Id).Error)
	recalculated, err := model.DecodeRewardSnapshot(stored.RewardSnapshotJSON)
	require.NoError(t, err)
	assert.Equal(t, 600_000, recalculated.Quota)
	assert.Equal(t, "600000", recalculated.QuotaPerUnit)

	omittedPayload := []byte(fmt.Sprintf(`{"title":"Omitted preserve campaign","enabled":false,"start_time":%d,"end_time":%d,"activation_window_seconds":3600,"max_rewards_per_inviter":0,"total_reward_limit":0,"reward":{"type":"quota","amount":"1","unit":"usd"}}`, now+60, now+3600))
	omittedRecorder := httptest.NewRecorder()
	omittedCtx, _ := gin.CreateTestContext(omittedRecorder)
	omittedCtx.Request = httptest.NewRequest(http.MethodPut, "/api/referral-campaign/admin/campaigns/1", bytes.NewReader(omittedPayload))
	omittedCtx.Request.Header.Set("Content-Type", "application/json")
	omittedCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(campaign.Id)}}
	AdminUpdateReferralCampaign(omittedCtx)
	require.NoError(t, db.First(&stored, campaign.Id).Error)
	recalculated, err = model.DecodeRewardSnapshot(stored.RewardSnapshotJSON)
	require.NoError(t, err)
	assert.Equal(t, 600_000, recalculated.Quota)
}

func TestAdminUpdateReferralCampaignPreservesLegacyAndSubscriptionSnapshots(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ReferralCampaign{}, &model.ReferralCampaignScheduleLock{}))
	now := common.GetTimestamp()
	cases := []struct {
		name     string
		snapshot string
	}{
		{name: "legacy cny", snapshot: `{"type":"quota","amount":"7.3","unit":"cny","quota":500000,"quota_per_unit":"500000","usd_exchange_rate":"7.3"}`},
		{name: "subscription", snapshot: `{"type":"subscription","subscription_plan_id":123,"subscription_plan_title":"Historic","subscription_plan_snapshot":"{\"id\":123,\"title\":\"Historic\"}"}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			campaign := &model.ReferralCampaign{Title: testCase.name, StartTime: now + 60, EndTime: now + 3600, ActivationWindowSeconds: 3600, RewardSnapshotJSON: testCase.snapshot}
			require.NoError(t, model.CreateReferralCampaign(campaign))
			payload := []byte(fmt.Sprintf(`{"title":"%s updated","enabled":false,"start_time":%d,"end_time":%d,"activation_window_seconds":3600,"max_rewards_per_inviter":0,"total_reward_limit":0,"preserve_reward":true,"reward":{"type":"quota","amount":"999","unit":"usd"}}`, testCase.name, now+60, now+3600))
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPut, "/api/referral-campaign/admin/campaigns/1", bytes.NewReader(payload))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(campaign.Id)}}
			AdminUpdateReferralCampaign(ctx)
			assert.Contains(t, recorder.Body.String(), `"success":true`)
			var stored model.ReferralCampaign
			require.NoError(t, db.First(&stored, campaign.Id).Error)
			assert.Equal(t, testCase.snapshot, stored.RewardSnapshotJSON)
		})
	}
}

func TestAdminUpdatePublicPoolSitePreservesStoredRewardSnapshotAndKeepsNullSemantics(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PublicPoolSite{}))
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })
	const originalSnapshot = `{"usd_exchange_rate":"7.3","quota_per_unit":"500000","quota":500000,"unit":"usd","amount":"1","type":"quota"}`
	site := &model.PublicPoolSite{Name: "Original site", URL: "https://example.com", Status: model.PublicPoolSiteStatusEnabled, RewardSnapshotJSON: originalSnapshot}
	require.NoError(t, model.CreatePublicPoolSite(site))

	common.QuotaPerUnit = 600_000
	preservePayload := []byte(`{"name":"Preserved site","url":"https://example.com","status":"enabled","sort_order":0,"preserve_reward":true,"reward":{"type":"quota","amount":"999","unit":"usd"}}`)
	preserveRecorder := httptest.NewRecorder()
	preserveCtx, _ := gin.CreateTestContext(preserveRecorder)
	preserveCtx.Request = httptest.NewRequest(http.MethodPut, "/api/public-pool/admin/sites/1", bytes.NewReader(preservePayload))
	preserveCtx.Request.Header.Set("Content-Type", "application/json")
	preserveCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(site.Id)}}
	AdminUpdatePublicPoolSite(preserveCtx)

	response := struct {
		Success bool                 `json:"success"`
		Data    model.PublicPoolSite `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(preserveRecorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.NotNil(t, response.Data.Reward)
	assert.Equal(t, 500_000, response.Data.Reward.Quota)
	assert.Equal(t, "1", response.Data.Reward.InputAmount)
	var stored model.PublicPoolSite
	require.NoError(t, db.First(&stored, site.Id).Error)
	assert.Equal(t, originalSnapshot, stored.RewardSnapshotJSON)

	recalculatePayload := []byte(`{"name":"Recalculated site","url":"https://example.com","status":"enabled","sort_order":0,"preserve_reward":false,"reward":{"type":"quota","amount":"1","unit":"usd"}}`)
	recalculateRecorder := httptest.NewRecorder()
	recalculateCtx, _ := gin.CreateTestContext(recalculateRecorder)
	recalculateCtx.Request = httptest.NewRequest(http.MethodPut, "/api/public-pool/admin/sites/1", bytes.NewReader(recalculatePayload))
	recalculateCtx.Request.Header.Set("Content-Type", "application/json")
	recalculateCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(site.Id)}}
	AdminUpdatePublicPoolSite(recalculateCtx)
	require.NoError(t, db.First(&stored, site.Id).Error)
	recalculated, err := model.DecodeRewardSnapshot(stored.RewardSnapshotJSON)
	require.NoError(t, err)
	assert.Equal(t, 600_000, recalculated.Quota)

	resetPayload := []byte(`{"name":"Reset site","url":"https://example.com","status":"enabled","sort_order":0,"reward":{"type":"quota","amount":"1","unit":"usd"}}`)
	resetRecorder := httptest.NewRecorder()
	resetCtx, _ := gin.CreateTestContext(resetRecorder)
	resetCtx.Request = httptest.NewRequest(http.MethodPut, "/api/public-pool/admin/sites/1", bytes.NewReader(resetPayload))
	resetCtx.Request.Header.Set("Content-Type", "application/json")
	resetCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(site.Id)}}
	AdminUpdatePublicPoolSite(resetCtx)
	require.NoError(t, db.First(&stored, site.Id).Error)
	recalculated, err = model.DecodeRewardSnapshot(stored.RewardSnapshotJSON)
	require.NoError(t, err)
	assert.Equal(t, 600_000, recalculated.Quota)
	snapshotBeforeNull := stored.RewardSnapshotJSON

	preserveNullPayload := []byte(`{"name":"Preserved null site","url":"https://example.com","status":"enabled","sort_order":0,"preserve_reward":true,"reward":null}`)
	preserveNullRecorder := httptest.NewRecorder()
	preserveNullCtx, _ := gin.CreateTestContext(preserveNullRecorder)
	preserveNullCtx.Request = httptest.NewRequest(http.MethodPut, "/api/public-pool/admin/sites/1", bytes.NewReader(preserveNullPayload))
	preserveNullCtx.Request.Header.Set("Content-Type", "application/json")
	preserveNullCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(site.Id)}}
	AdminUpdatePublicPoolSite(preserveNullCtx)
	require.NoError(t, db.First(&stored, site.Id).Error)
	assert.Equal(t, snapshotBeforeNull, stored.RewardSnapshotJSON)

	clearPayload := []byte(`{"name":"Cleared site","url":"https://example.com","status":"enabled","sort_order":0,"reward":null}`)
	clearRecorder := httptest.NewRecorder()
	clearCtx, _ := gin.CreateTestContext(clearRecorder)
	clearCtx.Request = httptest.NewRequest(http.MethodPut, "/api/public-pool/admin/sites/1", bytes.NewReader(clearPayload))
	clearCtx.Request.Header.Set("Content-Type", "application/json")
	clearCtx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(site.Id)}}
	AdminUpdatePublicPoolSite(clearCtx)
	require.NoError(t, db.First(&stored, site.Id).Error)
	assert.Empty(t, stored.RewardSnapshotJSON)
}

func TestAdminCreateReferralCampaignCannotUsePreserveRewardToSkipRewardValidation(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ReferralCampaign{}, &model.ReferralCampaignScheduleLock{}))
	now := common.GetTimestamp()
	payload := []byte(fmt.Sprintf(`{"title":"Invalid preserved create","enabled":false,"start_time":%d,"end_time":%d,"activation_window_seconds":3600,"max_rewards_per_inviter":0,"total_reward_limit":0,"preserve_reward":true,"reward":{"type":"quota","amount":"invalid","unit":"usd"}}`, now+60, now+3600))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/referral-campaign/admin/campaigns", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	AdminCreateReferralCampaign(ctx)

	assert.Contains(t, recorder.Body.String(), `"success":false`)
	var count int64
	require.NoError(t, db.Model(&model.ReferralCampaign{}).Count(&count).Error)
	assert.Zero(t, count)
}
