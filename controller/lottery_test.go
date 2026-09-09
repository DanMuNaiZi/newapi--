package controller

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLotteryControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.LotteryPlan{},
		&model.LotteryPlanGroup{},
		&model.LotteryPlanUser{},
		&model.LotteryPrize{},
		&model.LotteryParticipant{},
		&model.LotteryDrawRun{},
		&model.LotteryResult{},
		&model.LotteryNotification{},
	))
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestGetLotteryPlansForSelfReturnsOnlyVisiblePlans(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	allowedUser := &model.User{Username: "lottery-controller-allowed", Password: "password", Status: common.UserStatusEnabled, Group: "vip", AffCode: "lottery-controller-a"}
	otherUser := &model.User{Username: "lottery-controller-other", Password: "password", Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-controller-b"}
	require.NoError(t, db.Create([]*model.User{allowedUser, otherUser}).Error)
	now := common.GetTimestamp()
	publicPlan := &model.LotteryPlan{Title: "Public", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 5, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	privatePlan := &model.LotteryPlan{Title: "Private", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityUsers, MaxParticipants: 5, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	prize := []*model.LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}
	require.NoError(t, model.CreateLotteryPlan(publicPlan, nil, nil, prize))
	require.NoError(t, model.CreateLotteryPlan(privatePlan, []int{allowedUser.Id}, nil, []*model.LotteryPrize{{Name: "Private prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lottery/self", nil)
	ctx.Set("id", otherUser.Id)

	GetLotteryPlansForSelf(ctx)

	response := struct {
		Success bool                `json:"success"`
		Data    []model.LotteryPlan `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, publicPlan.Id, response.Data[0].Id)
}

func TestUserPreviewNeverReturnsLotteryRedemptionCodes(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-controller-preview", Password: "password", Status: common.UserStatusEnabled, Group: "vip", AffCode: "lottery-controller-preview"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&model.LotteryResult{
		PlanId:            100,
		UserId:            user.Id,
		PrizeId:           1,
		FulfillmentMode:   model.LotteryFulfillmentRedemptionCode,
		FulfillmentStatus: "fulfilled",
		RedemptionCode:    "secret-code",
		CreatedAt:         common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lottery/results/self", nil)
	ctx.Set("id", user.Id)
	ctx.Set("preview_mode", true)

	GetLotteryResultsForSelf(ctx)

	response := struct {
		Success bool                          `json:"success"`
		Data    []model.LotterySelfResultView `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Empty(t, response.Data[0].RedemptionCode)
}

func TestLotterySelfActionsJoinLeaveAndClaim(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-controller-user", Password: "password", Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-controller-self"}
	require.NoError(t, db.Create(user).Error)
	now := common.GetTimestamp()
	plan := &model.LotteryPlan{Title: "Self actions", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(plan, nil, nil, []*model.LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentSelfClaim, ClaimExpireSeconds: 3600}}))

	joinRecorder := httptest.NewRecorder()
	joinContext, _ := gin.CreateTestContext(joinRecorder)
	joinContext.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/1/join", nil)
	joinContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	joinContext.Set("id", user.Id)
	JoinLotteryPlanForSelf(joinContext)
	require.Contains(t, joinRecorder.Body.String(), `"success":true`)

	leaveRecorder := httptest.NewRecorder()
	leaveContext, _ := gin.CreateTestContext(leaveRecorder)
	leaveContext.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/1/leave", nil)
	leaveContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	leaveContext.Set("id", user.Id)
	LeaveLotteryPlanForSelf(leaveContext)
	require.Contains(t, leaveRecorder.Body.String(), `"success":true`)

	require.NoError(t, model.JoinLotteryPlan(plan.Id, user.Id))
	_, err := model.DrawLotteryPlan(plan.Id, model.LotteryDrawTriggerManual, "test")
	require.NoError(t, err)
	var result model.LotteryResult
	require.NoError(t, db.Where("plan_id = ? AND user_id = ?", plan.Id, user.Id).First(&result).Error)

	claimRecorder := httptest.NewRecorder()
	claimContext, _ := gin.CreateTestContext(claimRecorder)
	claimContext.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/results/1/claim", nil)
	claimContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(result.Id)}}
	claimContext.Set("id", user.Id)
	ClaimLotteryResultForSelf(claimContext)
	require.Contains(t, claimRecorder.Body.String(), `"success":true`)

	var storedUser model.User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Equal(t, 100, storedUser.Quota)
}

func TestLotteryPublicDetailsEndpointsMaskParticipantAndWinnerNames(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-public-user", DisplayName: "Public Winner", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-public-user"}
	require.NoError(t, db.Create(user).Error)
	now := common.GetTimestamp()
	plan := &model.LotteryPlan{Title: "Public details", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(plan, nil, nil, []*model.LotteryPrize{{Name: "Public prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))
	require.NoError(t, model.JoinLotteryPlan(plan.Id, user.Id))
	_, err := model.DrawLotteryPlan(plan.Id, model.LotteryDrawTriggerManual, "verify public endpoints")
	require.NoError(t, err)

	participantsRecorder := httptest.NewRecorder()
	participantsContext, _ := gin.CreateTestContext(participantsRecorder)
	participantsContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	participantsContext.Set("id", user.Id)
	GetLotteryParticipantsForSelf(participantsContext)
	require.Contains(t, participantsRecorder.Body.String(), `"username":"l***r"`)
	require.Contains(t, participantsRecorder.Body.String(), `"is_self":true`)
	require.NotContains(t, participantsRecorder.Body.String(), `lottery-public-user`)
	require.NotContains(t, participantsRecorder.Body.String(), `"weight"`)
	require.NotContains(t, participantsRecorder.Body.String(), `"preset_prize_id"`)

	resultsRecorder := httptest.NewRecorder()
	resultsContext, _ := gin.CreateTestContext(resultsRecorder)
	resultsContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	resultsContext.Set("id", user.Id)
	GetLotteryPlanResultsForSelf(resultsContext)
	require.Contains(t, resultsRecorder.Body.String(), `"username":"l***r"`)
	require.Contains(t, resultsRecorder.Body.String(), `"is_self":true`)
	require.NotContains(t, resultsRecorder.Body.String(), `lottery-public-user`)
	require.Contains(t, resultsRecorder.Body.String(), `"prize_name":"Public prize"`)
	require.NotContains(t, resultsRecorder.Body.String(), `"redemption_code"`)
}

func TestGetLotteryPlanForSelfUsesHTTPStatusForInvalidInvisibleAndMissingPlans(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-detail-user", Password: "password", Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-detail-user"}
	allowedUser := &model.User{Username: "lottery-detail-allowed", Password: "password", Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-detail-allowed"}
	require.NoError(t, db.Create([]*model.User{user, allowedUser}).Error)
	now := common.GetTimestamp()
	privatePlan := &model.LotteryPlan{Title: "Private", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityUsers, MaxParticipants: 2, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(privatePlan, []int{allowedUser.Id}, nil, []*model.LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))

	tests := []struct {
		name       string
		pathID     string
		wantStatus int
	}{
		{name: "invalid", pathID: "bad", wantStatus: http.StatusBadRequest},
		{name: "invisible", pathID: fmt.Sprint(privatePlan.Id), wantStatus: http.StatusForbidden},
		{name: "missing", pathID: "999999", wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lottery/plans/"+test.pathID, nil)
			ctx.Params = gin.Params{{Key: "id", Value: test.pathID}}
			ctx.Set("id", user.Id)
			ctx.Set(common.RequestIdKey, "lottery-request-id")

			GetLotteryPlanForSelf(ctx)

			assert.Equal(t, test.wantStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), `"request_id":"lottery-request-id"`)
		})
	}
}

func TestLotteryUserAPIErrorHidesInternalFailureDetails(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lottery/plans/1", nil)
	ctx.Set(common.RequestIdKey, "lottery-internal-error")

	lotteryUserAPIError(ctx, 0, errors.New("database password leaked in driver error"))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"message":"internal server error"`)
	assert.Contains(t, recorder.Body.String(), `"request_id":"lottery-internal-error"`)
	assert.NotContains(t, recorder.Body.String(), "database password")
}

func TestAdminCreateLotteryPlanPersistsPrizeAndAllowList(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	admin := &model.User{Username: "lottery-controller-admin", Password: "password", Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-controller-admin"}
	allowedUser := &model.User{Username: "lottery-controller-allowed-user", Password: "password", Status: common.UserStatusEnabled, Group: "vip", AffCode: "lottery-controller-allowed"}
	require.NoError(t, db.Create([]*model.User{admin, allowedUser}).Error)
	now := common.GetTimestamp()
	body := fmt.Sprintf(`{"title":"Admin plan","icon":"https://cdn.example.com/lottery.png","status":"scheduled","eligibility_mode":"users","max_participants":5,"registration_start_time":%d,"draw_time":%d,"user_ids":[%d],"prizes":[{"name":"Prize","quantity":1,"reward_type":"quota","quota":100,"fulfillment_mode":"auto"}]}`, now+60, now+3600, allowedUser.Id)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/admin/plans", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", admin.Id)

	AdminCreateLotteryPlan(ctx)

	require.Contains(t, recorder.Body.String(), `"success":true`)
	var plan model.LotteryPlan
	require.NoError(t, db.Where("title = ?", "Admin plan").First(&plan).Error)
	assert.Equal(t, model.LotteryPlanStatusScheduled, plan.Status)
	assert.Equal(t, "https://cdn.example.com/lottery.png", plan.Icon)
	var allowListCount int64
	require.NoError(t, db.Model(&model.LotteryPlanUser{}).Where("plan_id = ? AND user_id = ?", plan.Id, allowedUser.Id).Count(&allowListCount).Error)
	assert.Equal(t, int64(1), allowListCount)
}

func TestAdminDrawLotteryPlanRequiresReasonAndFinishesPlan(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	admin := &model.User{Username: "lottery-draw-admin", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-draw-admin"}
	participant := &model.User{Username: "lottery-draw-participant", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-draw-participant"}
	require.NoError(t, db.Create([]*model.User{admin, participant}).Error)
	now := common.GetTimestamp()
	plan := &model.LotteryPlan{Title: "Manual draw", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(plan, nil, nil, []*model.LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))
	require.NoError(t, model.JoinLotteryPlan(plan.Id, participant.Id))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/admin/plans/1/draw", bytes.NewBufferString(`{"reason":"manual verification"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	ctx.Set("id", admin.Id)
	AdminDrawLotteryPlan(ctx)
	require.Contains(t, recorder.Body.String(), `"success":true`)

	var stored model.LotteryPlan
	require.NoError(t, db.First(&stored, plan.Id).Error)
	assert.Equal(t, model.LotteryPlanStatusFinished, stored.Status)
}

func TestAdminListLotteryResultsReturnsWinnerDetails(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	participant := &model.User{Username: "lottery-result-participant", DisplayName: "Result Winner", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-result-participant"}
	require.NoError(t, db.Create(participant).Error)
	now := common.GetTimestamp()
	plan := &model.LotteryPlan{Title: "Result details", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(plan, nil, nil, []*model.LotteryPrize{{Name: "Result prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))
	require.NoError(t, model.JoinLotteryPlan(plan.Id, participant.Id))
	_, err := model.DrawLotteryPlan(plan.Id, model.LotteryDrawTriggerManual, "verify result endpoint")
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lottery/admin/plans/1/results", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	AdminListLotteryResults(ctx)

	response := struct {
		Success bool                      `json:"success"`
		Data    []model.LotteryResultView `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, participant.Id, response.Data[0].UserId)
	assert.Equal(t, participant.Username, response.Data[0].Username)
	assert.Equal(t, participant.DisplayName, response.Data[0].DisplayName)
	assert.Equal(t, "Result prize", response.Data[0].PrizeName)
}

func TestAdminUpdateLotteryParticipantUpdatesWeight(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	admin := &model.User{Username: "lottery-weight-admin", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-weight-admin"}
	participant := &model.User{Username: "lottery-weight-participant", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-weight-participant"}
	require.NoError(t, db.Create([]*model.User{admin, participant}).Error)
	now := common.GetTimestamp()
	plan := &model.LotteryPlan{Title: "Participant management", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(plan, nil, nil, []*model.LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))
	require.NoError(t, model.JoinLotteryPlan(plan.Id, participant.Id))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/lottery/admin/plans/1/participants", bytes.NewBufferString(fmt.Sprintf(`{"user_id":%d,"weight":500}`, participant.Id)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	ctx.Set("id", admin.Id)
	AdminUpdateLotteryParticipant(ctx)
	require.Contains(t, recorder.Body.String(), `"success":true`)

	var stored model.LotteryParticipant
	require.NoError(t, db.Where("plan_id = ? AND user_id = ?", plan.Id, participant.Id).First(&stored).Error)
	assert.Equal(t, 500, stored.Weight)
}

func TestAdminUpdatesAndCancelsPublishedLotteryPlan(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	admin := &model.User{Username: "lottery-plan-admin", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-plan-admin"}
	require.NoError(t, db.Create(admin).Error)
	now := common.GetTimestamp()
	plan := &model.LotteryPlan{Title: "Published plan", Status: model.LotteryPlanStatusScheduled, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: now + 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(plan, nil, nil, []*model.LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))

	updateRecorder := httptest.NewRecorder()
	updateContext, _ := gin.CreateTestContext(updateRecorder)
	updateContext.Request = httptest.NewRequest(http.MethodPatch, "/api/lottery/admin/plans/1", bytes.NewBufferString(fmt.Sprintf(`{"title":"Updated plan","description":"Updated copy","draw_time":%d}`, now+7200)))
	updateContext.Request.Header.Set("Content-Type", "application/json")
	updateContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	updateContext.Set("id", admin.Id)
	AdminUpdateLotteryPlan(updateContext)
	require.Contains(t, updateRecorder.Body.String(), `"success":true`)

	cancelRecorder := httptest.NewRecorder()
	cancelContext, _ := gin.CreateTestContext(cancelRecorder)
	cancelContext.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/admin/plans/1/cancel", nil)
	cancelContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	cancelContext.Set("id", admin.Id)
	AdminCancelLotteryPlan(cancelContext)
	require.Contains(t, cancelRecorder.Body.String(), `"success":true`)

	var stored model.LotteryPlan
	require.NoError(t, db.First(&stored, plan.Id).Error)
	assert.Equal(t, "Updated plan", stored.Title)
	assert.Equal(t, now+7200, stored.DrawTime)
	assert.Equal(t, model.LotteryPlanStatusCancelled, stored.Status)
}

func TestGetLotteryResultsPageForSelfReturnsClaimStateAndCursor(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-results-page-user", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-results-page-user"}
	require.NoError(t, db.Create(user).Error)
	now := common.GetTimestamp()
	require.NoError(t, db.Create([]*model.LotteryResult{
		{
			PlanId:            1,
			UserId:            user.Id,
			FulfillmentMode:   model.LotteryFulfillmentSelfClaim,
			FulfillmentStatus: "pending",
			ClaimExpiresAt:    now - 1,
			CreatedAt:         now + 1,
		},
		{
			PlanId:            2,
			UserId:            user.Id,
			FulfillmentMode:   model.LotteryFulfillmentSelfClaim,
			FulfillmentStatus: "pending",
			CreatedAt:         now,
		},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lottery/results/self/page?limit=1", nil)
	ctx.Set("id", user.Id)
	GetLotteryResultsPageForSelf(ctx)

	response := struct {
		Success bool                        `json:"success"`
		Data    model.LotterySelfResultPage `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	require.Len(t, response.Data.Items, 1)
	assert.True(t, response.Data.HasMore)
	assert.NotEmpty(t, response.Data.NextCursor)
	assert.Equal(t, "expired", response.Data.Items[0].ClaimStatus)
	assert.False(t, response.Data.Items[0].Claimable)
}

func TestMarkLotteryNotificationsReadForSelfOnlyUpdatesSubmittedIDs(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-notification-user", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-notification-user"}
	otherUser := &model.User{Username: "lottery-notification-other", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-notification-other"}
	require.NoError(t, db.Create([]*model.User{user, otherUser}).Error)
	now := common.GetTimestamp()
	notifications := []*model.LotteryNotification{
		{UserId: user.Id, PlanId: 1, Type: "lottery_result", CreatedAt: now},
		{UserId: user.Id, PlanId: 2, Type: "lottery_result", CreatedAt: now + 1},
		{UserId: user.Id, PlanId: 3, Type: "lottery_result", CreatedAt: now + 2},
		{UserId: otherUser.Id, PlanId: 4, Type: "lottery_result", CreatedAt: now + 3},
	}
	for _, notification := range notifications {
		require.NoError(t, db.Create(notification).Error)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/lottery/notifications/self/read",
		bytes.NewBufferString(fmt.Sprintf(`{"ids":[%d,%d]}`, notifications[0].Id, notifications[1].Id)),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", user.Id)
	MarkLotteryNotificationsReadForSelf(ctx)
	require.Contains(t, recorder.Body.String(), `"success":true`)

	for _, notification := range notifications[:2] {
		var stored model.LotteryNotification
		require.NoError(t, db.First(&stored, notification.Id).Error)
		assert.NotZero(t, stored.ReadAt)
	}
	for _, notification := range notifications[2:] {
		var stored model.LotteryNotification
		require.NoError(t, db.First(&stored, notification.Id).Error)
		assert.Zero(t, stored.ReadAt)
	}
}

func TestLotterySelfResultsExposeOnlyOwnRedemptionCode(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-code-owner", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-code-owner"}
	otherUser := &model.User{Username: "lottery-code-other", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-code-other"}
	require.NoError(t, db.Create([]*model.User{user, otherUser}).Error)
	ownResult := &model.LotteryResult{
		PlanId:            1,
		UserId:            user.Id,
		FulfillmentMode:   model.LotteryFulfillmentRedemptionCode,
		FulfillmentStatus: "issued",
		RedemptionCode:    "OWN-CODE",
		CreatedAt:         common.GetTimestamp(),
	}
	otherResult := &model.LotteryResult{
		PlanId:            2,
		UserId:            otherUser.Id,
		FulfillmentMode:   model.LotteryFulfillmentRedemptionCode,
		FulfillmentStatus: "issued",
		RedemptionCode:    "OTHER-CODE",
		CreatedAt:         common.GetTimestamp(),
	}
	require.NoError(t, db.Create([]*model.LotteryResult{ownResult, otherResult}).Error)

	callResults := func(userID int) (string, []model.LotterySelfResultView) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lottery/results/self", nil)
		ctx.Set("id", userID)
		GetLotteryResultsForSelf(ctx)
		response := struct {
			Success bool                          `json:"success"`
			Data    []model.LotterySelfResultView `json:"data"`
		}{}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success)
		return recorder.Body.String(), response.Data
	}

	ownerBody, ownerResults := callResults(user.Id)
	require.Len(t, ownerResults, 1)
	assert.Equal(t, "OWN-CODE", ownerResults[0].RedemptionCode)
	assert.Contains(t, ownerBody, "OWN-CODE")
	assert.NotContains(t, ownerBody, "OTHER-CODE")

	otherBody, otherResults := callResults(otherUser.Id)
	require.Len(t, otherResults, 1)
	assert.Equal(t, "OTHER-CODE", otherResults[0].RedemptionCode)
	assert.Contains(t, otherBody, "OTHER-CODE")
	assert.NotContains(t, otherBody, "OWN-CODE")

	claimRecorder := httptest.NewRecorder()
	claimContext, _ := gin.CreateTestContext(claimRecorder)
	claimContext.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/results/1/claim", nil)
	claimContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(ownResult.Id)}}
	claimContext.Set("id", otherUser.Id)
	ClaimLotteryResultForSelf(claimContext)
	assert.Contains(t, claimRecorder.Body.String(), `"success":false`)
	var storedOwn model.LotteryResult
	require.NoError(t, db.First(&storedOwn, ownResult.Id).Error)
	assert.Equal(t, "issued", storedOwn.FulfillmentStatus)
}

func TestMarkLotteryNotificationsReadForSelfValidatesBatchAndDeduplicatesIDs(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{Username: "lottery-read-owner", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-read-owner"}
	otherUser := &model.User{Username: "lottery-read-other", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-read-other"}
	require.NoError(t, db.Create([]*model.User{user, otherUser}).Error)
	ownNotification := &model.LotteryNotification{UserId: user.Id, PlanId: 1, Type: "lottery_result", CreatedAt: common.GetTimestamp()}
	otherNotification := &model.LotteryNotification{UserId: otherUser.Id, PlanId: 2, Type: "lottery_result", CreatedAt: common.GetTimestamp()}
	require.NoError(t, db.Create([]*model.LotteryNotification{ownNotification, otherNotification}).Error)

	tooManyIDs := make([]int, lotteryNotificationReadMaxIds+1)
	for index := range tooManyIDs {
		tooManyIDs[index] = index + 1
	}
	tooManyBody, err := common.Marshal(map[string]interface{}{"ids": tooManyIDs})
	require.NoError(t, err)
	tooManyRecorder := httptest.NewRecorder()
	tooManyContext, _ := gin.CreateTestContext(tooManyRecorder)
	tooManyContext.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/notifications/self/read", bytes.NewReader(tooManyBody))
	tooManyContext.Request.Header.Set("Content-Type", "application/json")
	tooManyContext.Set("id", user.Id)
	MarkLotteryNotificationsReadForSelf(tooManyContext)
	assert.Contains(t, tooManyRecorder.Body.String(), `"success":false`)

	validBody := fmt.Sprintf(`{"ids":[%d,%d,%d]}`, ownNotification.Id, ownNotification.Id, otherNotification.Id)
	validRecorder := httptest.NewRecorder()
	validContext, _ := gin.CreateTestContext(validRecorder)
	validContext.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/notifications/self/read", bytes.NewBufferString(validBody))
	validContext.Request.Header.Set("Content-Type", "application/json")
	validContext.Set("id", user.Id)
	MarkLotteryNotificationsReadForSelf(validContext)
	assert.Contains(t, validRecorder.Body.String(), `"success":true`)

	var storedOwn, storedOther model.LotteryNotification
	require.NoError(t, db.First(&storedOwn, ownNotification.Id).Error)
	require.NoError(t, db.First(&storedOther, otherNotification.Id).Error)
	assert.NotZero(t, storedOwn.ReadAt)
	assert.Zero(t, storedOther.ReadAt)
}
