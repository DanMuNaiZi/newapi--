package controller

import (
	"bytes"
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

func TestLotteryPublicDetailsEndpointsReturnParticipantAndWinnerNames(t *testing.T) {
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
	require.Contains(t, participantsRecorder.Body.String(), `"username":"lottery-public-user"`)
	require.NotContains(t, participantsRecorder.Body.String(), `"weight"`)
	require.NotContains(t, participantsRecorder.Body.String(), `"preset_prize_id"`)

	resultsRecorder := httptest.NewRecorder()
	resultsContext, _ := gin.CreateTestContext(resultsRecorder)
	resultsContext.Params = gin.Params{{Key: "id", Value: fmt.Sprint(plan.Id)}}
	resultsContext.Set("id", user.Id)
	GetLotteryPlanResultsForSelf(resultsContext)
	require.Contains(t, resultsRecorder.Body.String(), `"username":"lottery-public-user"`)
	require.Contains(t, resultsRecorder.Body.String(), `"prize_name":"Public prize"`)
	require.NotContains(t, resultsRecorder.Body.String(), `"redemption_code"`)
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
