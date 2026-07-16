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

func TestUpdateRedemptionSnapshotsSubscriptionReward(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Redemption{}, &model.SubscriptionPlan{}))
	admin := &model.User{Username: "redemption-update-admin", Password: "password", Status: common.UserStatusEnabled, AffCode: "redemption-update-admin"}
	require.NoError(t, db.Create(admin).Error)
	plan := &model.SubscriptionPlan{
		Title:              "Redemption subscription",
		DurationUnit:       "month",
		DurationValue:      1,
		Enabled:            true,
		MaxPurchasePerUser: 1,
		TotalAmount:        100,
	}
	plan.NormalizeDefaults()
	require.NoError(t, db.Create(plan).Error)
	redemption := &model.Redemption{
		UserId:      admin.Id,
		Name:        "Quota code",
		Key:         "10000000000000000000000000000001",
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       100,
		RewardType:  model.RedemptionRewardQuota,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, db.Create(redemption).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body := fmt.Sprintf(`{"id":%d,"name":"Subscription code","quota":0,"reward_type":"subscription","subscription_plan_id":%d,"batch":"campaign-1","source_ref":"lottery","remark":"winner"}`, redemption.Id, plan.Id)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/redemption/", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", admin.Id)

	UpdateRedemption(ctx)
	require.Contains(t, recorder.Body.String(), `"success":true`)

	var stored model.Redemption
	require.NoError(t, db.First(&stored, redemption.Id).Error)
	assert.Equal(t, model.RedemptionRewardSubscription, stored.RewardType)
	assert.Equal(t, plan.Id, stored.SubscriptionPlanId)
	assert.NotEmpty(t, stored.SubscriptionSnapshot)
	assert.Equal(t, "campaign-1", stored.Batch)
	assert.Equal(t, "lottery", stored.SourceRef)
}

func TestAddRedemptionCreatesSubscriptionSnapshot(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Redemption{}, &model.SubscriptionPlan{}))
	admin := &model.User{Username: "redemption-create-admin", Password: "password", Status: common.UserStatusEnabled, AffCode: "redemption-create-admin"}
	require.NoError(t, db.Create(admin).Error)
	plan := &model.SubscriptionPlan{
		Title:              "Created redemption subscription",
		DurationUnit:       "month",
		DurationValue:      1,
		Enabled:            true,
		MaxPurchasePerUser: 1,
		TotalAmount:        200,
	}
	plan.NormalizeDefaults()
	require.NoError(t, db.Create(plan).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body := fmt.Sprintf(`{"name":"Subscription batch","count":2,"reward_type":"subscription","subscription_plan_id":%d,"batch":"campaign-2","source_ref":"admin","remark":"subscription reward"}`, plan.Id)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/redemption/", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", admin.Id)

	AddRedemption(ctx)
	require.Contains(t, recorder.Body.String(), `"success":true`)

	var redemptions []model.Redemption
	require.NoError(t, db.Where("batch = ?", "campaign-2").Find(&redemptions).Error)
	require.Len(t, redemptions, 2)
	for _, redemption := range redemptions {
		assert.Equal(t, model.RedemptionRewardSubscription, redemption.RewardType)
		assert.Equal(t, plan.Id, redemption.SubscriptionPlanId)
		assert.NotEmpty(t, redemption.SubscriptionSnapshot)
		assert.Equal(t, "admin", redemption.SourceRef)
	}
}

func TestTopUpSubscriptionReturnsSubscriptionOutcome(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.Redemption{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
	))
	user := &model.User{
		Username: "subscription-topup-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		AffCode:  "subscription-topup-user",
	}
	require.NoError(t, db.Create(user).Error)
	plan := &model.SubscriptionPlan{
		Title:         "Subscription topup plan",
		DurationUnit:  "month",
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   500,
	}
	plan.NormalizeDefaults()
	require.NoError(t, db.Create(plan).Error)
	snapshot, err := common.Marshal(plan)
	require.NoError(t, err)
	redemption := &model.Redemption{
		Name:                 "Subscription topup code",
		Key:                  "30000000000000000000000000000001",
		Status:               common.RedemptionCodeStatusEnabled,
		RewardType:           model.RedemptionRewardSubscription,
		SubscriptionPlanId:   plan.Id,
		SubscriptionSnapshot: string(snapshot),
		CreatedTime:          common.GetTimestamp(),
	}
	require.NoError(t, db.Create(redemption).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/user/topup",
		bytes.NewBufferString(fmt.Sprintf(`{"key":%q}`, redemption.Key)),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", user.Id)

	TopUp(ctx)

	response := struct {
		Success            bool   `json:"success"`
		Data               int    `json:"data"`
		RewardType         string `json:"reward_type"`
		SubscriptionPlanId int    `json:"subscription_plan_id"`
		SubscriptionId     int    `json:"subscription_id"`
	}{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Zero(t, response.Data)
	assert.Equal(t, model.RedemptionRewardSubscription, response.RewardType)
	assert.Equal(t, plan.Id, response.SubscriptionPlanId)
	assert.NotZero(t, response.SubscriptionId)

	var subscription model.UserSubscription
	require.NoError(t, db.First(&subscription, response.SubscriptionId).Error)
	assert.Equal(t, user.Id, subscription.UserId)
	assert.Equal(t, plan.Id, subscription.PlanId)
	assert.Equal(t, "redemption", subscription.Source)
}
