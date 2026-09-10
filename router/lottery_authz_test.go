package router

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

type lotteryRouteResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type lotteryRouteFixture struct {
	db      *gorm.DB
	router  *gin.Engine
	cookies map[int]*http.Cookie

	ordinary *model.User
	other    *model.User
	readOnly *model.User
	operator *model.User
	admin    *model.User
	root     *model.User

	visiblePlan       *model.LotteryPlan
	privatePlan       *model.LotteryPlan
	draftPlan         *model.LotteryPlan
	notification      *model.LotteryNotification
	otherNotification *model.LotteryNotification
}

func setupLotteryRouteFixture(t *testing.T) *lotteryRouteFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMasterNode := common.IsMasterNode
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousGlobalAPIRateLimit := common.GlobalApiRateLimitEnable

	common.RedisEnabled = false
	common.IsMasterNode = true
	common.GlobalApiRateLimitEnable = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.CasbinRule{},
		&model.AuthzRole{},
		&model.LotteryPlan{},
		&model.LotteryPlanGroup{},
		&model.LotteryPlanUser{},
		&model.LotteryPrize{},
		&model.LotteryParticipant{},
		&model.LotteryDrawRun{},
		&model.LotteryResult{},
		&model.LotteryNotification{},
	))
	require.NoError(t, authz.Init(db))

	fixture := &lotteryRouteFixture{db: db}
	fixture.ordinary = &model.User{Username: "ordinary-user", DisplayName: "Ordinary User", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-ordinary"}
	fixture.other = &model.User{Username: "winner-user", DisplayName: "Winner User", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-winner"}
	fixture.readOnly = &model.User{Username: "lottery-reader", DisplayName: "Lottery Reader", Password: "password", Role: common.RoleAuthorizedAdmin, Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-reader"}
	fixture.operator = &model.User{Username: "lottery-operator", DisplayName: "Lottery Operator", Password: "password", Role: common.RoleAuthorizedAdmin, Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-operator"}
	fixture.admin = &model.User{Username: "lottery-admin", DisplayName: "Lottery Admin", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-admin"}
	fixture.root = &model.User{Username: "lottery-root", DisplayName: "Lottery Root", Password: "password", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-root"}
	require.NoError(t, db.Create([]*model.User{fixture.ordinary, fixture.other, fixture.readOnly, fixture.operator, fixture.admin, fixture.root}).Error)

	require.NoError(t, authz.SetUserPermissionsForRole(fixture.readOnly.Id, common.RoleAuthorizedAdmin, authz.PermissionsMap{
		authz.ResourceLottery: {authz.ActionRead: true},
	}))
	require.NoError(t, authz.SetUserPermissionsForRole(fixture.operator.Id, common.RoleAuthorizedAdmin, authz.PermissionsMap{
		authz.ResourceLottery: {authz.ActionRead: true, authz.ActionOperate: true},
	}))

	now := common.GetTimestamp()
	fixture.visiblePlan = &model.LotteryPlan{Title: "Visible plan", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 10, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(fixture.visiblePlan, nil, nil, []*model.LotteryPrize{{Name: "Visible prize", Quantity: 2, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentRedemptionCode}}))
	var prize model.LotteryPrize
	require.NoError(t, db.Where("plan_id = ?", fixture.visiblePlan.Id).First(&prize).Error)
	require.NoError(t, db.Create([]*model.LotteryParticipant{
		{PlanId: fixture.visiblePlan.Id, UserId: fixture.ordinary.Id, UserGroup: "default", Weight: 100, Status: model.LotteryParticipantStatusJoined, JoinedAt: now},
		{PlanId: fixture.visiblePlan.Id, UserId: fixture.other.Id, UserGroup: "default", Weight: 200, Status: model.LotteryParticipantStatusJoined, JoinedAt: now + 1},
	}).Error)
	require.NoError(t, db.Create([]*model.LotteryResult{
		{PlanId: fixture.visiblePlan.Id, UserId: fixture.ordinary.Id, PrizeId: prize.Id, PrizeSnapshot: prize.Name, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentRedemptionCode, FulfillmentStatus: "issued", RedemptionCode: "OWN-REDEEM-CODE", CreatedAt: now},
		{PlanId: fixture.visiblePlan.Id, UserId: fixture.other.Id, PrizeId: prize.Id, PrizeSnapshot: prize.Name, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentRedemptionCode, FulfillmentStatus: "issued", RedemptionCode: "OTHER-REDEEM-CODE", CreatedAt: now + 1},
	}).Error)

	fixture.privatePlan = &model.LotteryPlan{Title: "Private plan", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityUsers, MaxParticipants: 10, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(fixture.privatePlan, []int{fixture.other.Id}, nil, []*model.LotteryPrize{{Name: "Private prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))
	fixture.draftPlan = &model.LotteryPlan{Title: "Draft plan", Status: model.LotteryPlanStatusDraft, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 10, RegistrationStartTime: now - 60, DrawTime: now + 3600}
	require.NoError(t, model.CreateLotteryPlan(fixture.draftPlan, nil, nil, []*model.LotteryPrize{{Name: "Draft prize", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 100, FulfillmentMode: model.LotteryFulfillmentAuto}}))
	fixture.notification = &model.LotteryNotification{UserId: fixture.ordinary.Id, PlanId: fixture.visiblePlan.Id, Type: "lottery_result", Content: "preview must not mutate", CreatedAt: now}
	fixture.otherNotification = &model.LotteryNotification{UserId: fixture.other.Id, PlanId: fixture.visiblePlan.Id, Type: "lottery_result", Content: "other user notification", CreatedAt: now}
	require.NoError(t, db.Create([]*model.LotteryNotification{fixture.notification, fixture.otherNotification}).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("lottery-route-authz-test"))))
	router.GET("/__lottery_test_session/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		var user model.User
		if err := db.First(&user, id).Error; err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		session := sessions.Default(c)
		session.Set("username", user.Username)
		session.Set("role", user.Role)
		session.Set("id", user.Id)
		session.Set("status", user.Status)
		session.Set("group", user.Group)
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(router)
	fixture.router = router
	fixture.cookies = make(map[int]*http.Cookie)
	for _, user := range []*model.User{fixture.ordinary, fixture.other, fixture.readOnly, fixture.operator, fixture.admin, fixture.root} {
		fixture.cookies[user.Id] = fixture.sessionCookie(t, user.Id)
	}

	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		common.IsMasterNode = previousMasterNode
		common.GlobalApiRateLimitEnable = previousGlobalAPIRateLimit
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		if previousDB != nil {
			_ = authz.Init(previousDB)
		}
		_ = sqlDB.Close()
	})
	return fixture
}

func (fixture *lotteryRouteFixture) sessionCookie(t *testing.T, userID int) *http.Cookie {
	t.Helper()
	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/__lottery_test_session/"+strconv.Itoa(userID), nil))
	require.Equal(t, http.StatusNoContent, recorder.Code)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	return cookies[0]
}

func (fixture *lotteryRouteFixture) request(user *model.User, method string, path string, body string, previewToken string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("New-Api-User", strconv.Itoa(user.Id))
	request.AddCookie(fixture.cookies[user.Id])
	if previewToken != "" {
		request.Header.Set("New-Api-User-Preview", previewToken)
	}
	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, request)
	return recorder
}

func decodeLotteryRouteResponse[T any](t *testing.T, recorder *httptest.ResponseRecorder) lotteryRouteResponse[T] {
	t.Helper()
	response := lotteryRouteResponse[T]{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response), recorder.Body.String())
	return response
}

func TestLotteryRoutesEnforceManagedAdminPermissionsAndPrivacyBoundaries(t *testing.T) {
	fixture := setupLotteryRouteFixture(t)

	t.Run("ordinary user gets masked public data and only own code", func(t *testing.T) {
		participants := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/participants", "", "")
		require.Equal(t, http.StatusOK, participants.Code)
		participantResponse := decodeLotteryRouteResponse[[]model.LotteryPublicParticipantView](t, participants)
		require.True(t, participantResponse.Success)
		require.Len(t, participantResponse.Data, 2)
		assert.NotContains(t, participants.Body.String(), fixture.other.Username)
		assert.NotContains(t, participants.Body.String(), fixture.other.DisplayName)

		participantsPage := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/participants/page", "", "")
		require.Equal(t, http.StatusOK, participantsPage.Code)
		participantPageResponse := decodeLotteryRouteResponse[model.LotteryPublicParticipantPage](t, participantsPage)
		require.True(t, participantPageResponse.Success)
		require.Len(t, participantPageResponse.Data.Items, 2)
		assert.NotContains(t, participantsPage.Body.String(), fixture.other.Username)
		assert.NotContains(t, participantsPage.Body.String(), fixture.other.DisplayName)

		publicResults := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/results", "", "")
		require.Equal(t, http.StatusOK, publicResults.Code)
		assert.NotContains(t, publicResults.Body.String(), "OWN-REDEEM-CODE")
		assert.NotContains(t, publicResults.Body.String(), "OTHER-REDEEM-CODE")
		assert.NotContains(t, publicResults.Body.String(), fixture.other.Username)

		publicResultsPage := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/results/page", "", "")
		require.Equal(t, http.StatusOK, publicResultsPage.Code)
		publicResultPageResponse := decodeLotteryRouteResponse[model.LotteryPublicResultPage](t, publicResultsPage)
		require.True(t, publicResultPageResponse.Success)
		require.Len(t, publicResultPageResponse.Data.Items, 2)
		assert.NotContains(t, publicResultsPage.Body.String(), "OWN-REDEEM-CODE")
		assert.NotContains(t, publicResultsPage.Body.String(), "OTHER-REDEEM-CODE")
		assert.NotContains(t, publicResultsPage.Body.String(), fixture.other.Username)
		assert.NotContains(t, publicResultsPage.Body.String(), fixture.other.DisplayName)

		selfResults := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/results/self", "", "")
		require.Equal(t, http.StatusOK, selfResults.Code)
		selfResponse := decodeLotteryRouteResponse[[]model.LotterySelfResultView](t, selfResults)
		require.True(t, selfResponse.Success)
		require.Len(t, selfResponse.Data, 1)
		assert.Equal(t, "OWN-REDEEM-CODE", selfResponse.Data[0].RedemptionCode)
		assert.Contains(t, selfResults.Body.String(), "OWN-REDEEM-CODE")
		assert.NotContains(t, selfResults.Body.String(), "OTHER-REDEEM-CODE")

		selfResultsPage := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/results/self/page", "", "")
		require.Equal(t, http.StatusOK, selfResultsPage.Code)
		selfResultPageResponse := decodeLotteryRouteResponse[model.LotterySelfResultPage](t, selfResultsPage)
		require.True(t, selfResultPageResponse.Success)
		require.Len(t, selfResultPageResponse.Data.Items, 1)
		assert.Equal(t, "OWN-REDEEM-CODE", selfResultPageResponse.Data.Items[0].RedemptionCode)
		assert.NotContains(t, selfResultsPage.Body.String(), "OTHER-REDEEM-CODE")

		markOtherRead := fixture.request(fixture.ordinary, http.MethodPost, "/api/lottery/notifications/self/read", fmt.Sprintf(`{"ids":[%d]}`, fixture.otherNotification.Id), "")
		require.Equal(t, http.StatusOK, markOtherRead.Code)
		assert.True(t, decodeLotteryRouteResponse[map[string]interface{}](t, markOtherRead).Success)
		var otherNotification model.LotteryNotification
		require.NoError(t, fixture.db.First(&otherNotification, fixture.otherNotification.Id).Error)
		assert.Zero(t, otherNotification.ReadAt)
	})

	t.Run("ordinary user cannot load draft or private plan or admin plan", func(t *testing.T) {
		for _, planID := range []int{fixture.privatePlan.Id, fixture.draftPlan.Id} {
			recorder := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/plans/"+strconv.Itoa(planID), "", "")
			require.Equal(t, http.StatusForbidden, recorder.Code)
			assert.False(t, decodeLotteryRouteResponse[map[string]interface{}](t, recorder).Success)
		}

		adminRequest := fixture.request(fixture.ordinary, http.MethodGet, "/api/lottery/admin/plans/"+strconv.Itoa(fixture.visiblePlan.Id), "", "")
		require.Equal(t, http.StatusOK, adminRequest.Code)
		assert.False(t, decodeLotteryRouteResponse[map[string]interface{}](t, adminRequest).Success)
	})

	t.Run("delegated reader sees full management details but cannot operate", func(t *testing.T) {
		participants := fixture.request(fixture.readOnly, http.MethodGet, "/api/lottery/admin/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/participants", "", "")
		require.Equal(t, http.StatusOK, participants.Code)
		participantResponse := decodeLotteryRouteResponse[[]model.LotteryParticipantView](t, participants)
		require.True(t, participantResponse.Success)
		require.Len(t, participantResponse.Data, 2)
		assert.Contains(t, participants.Body.String(), fixture.other.Username)
		assert.Contains(t, participants.Body.String(), fixture.other.DisplayName)

		results := fixture.request(fixture.readOnly, http.MethodGet, "/api/lottery/admin/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/results", "", "")
		require.Equal(t, http.StatusOK, results.Code)
		resultResponse := decodeLotteryRouteResponse[[]model.LotteryResultView](t, results)
		require.True(t, resultResponse.Success)
		require.Len(t, resultResponse.Data, 2)
		assert.Contains(t, results.Body.String(), "OTHER-REDEEM-CODE")

		update := fixture.request(fixture.readOnly, http.MethodPut, "/api/lottery/admin/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/participants", fmt.Sprintf(`{"user_id":%d,"weight":333}`, fixture.ordinary.Id), "")
		require.Equal(t, http.StatusForbidden, update.Code)
		assert.False(t, decodeLotteryRouteResponse[map[string]interface{}](t, update).Success)
	})

	t.Run("delegated operator can update a participant", func(t *testing.T) {
		update := fixture.request(fixture.operator, http.MethodPut, "/api/lottery/admin/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/participants", fmt.Sprintf(`{"user_id":%d,"weight":333}`, fixture.ordinary.Id), "")
		require.Equal(t, http.StatusOK, update.Code)
		assert.True(t, decodeLotteryRouteResponse[map[string]interface{}](t, update).Success)

		var participant model.LotteryParticipant
		require.NoError(t, fixture.db.Where("plan_id = ? AND user_id = ?", fixture.visiblePlan.Id, fixture.ordinary.Id).First(&participant).Error)
		assert.Equal(t, 333, participant.Weight)
	})

	t.Run("built in admin reads draft plan by independent id", func(t *testing.T) {
		recorder := fixture.request(fixture.admin, http.MethodGet, "/api/lottery/admin/plans/"+strconv.Itoa(fixture.draftPlan.Id), "", "")
		require.Equal(t, http.StatusOK, recorder.Code)
		response := decodeLotteryRouteResponse[model.LotteryPlan](t, recorder)
		require.True(t, response.Success)
		assert.Equal(t, fixture.draftPlan.Id, response.Data.Id)
		assert.Equal(t, model.LotteryPlanStatusDraft, response.Data.Status)
	})

	t.Run("root preview stays masked and read only", func(t *testing.T) {
		preview := fixture.request(fixture.root, http.MethodPost, "/api/user/"+strconv.Itoa(fixture.ordinary.Id)+"/preview", "", "")
		require.Equal(t, http.StatusOK, preview.Code)
		previewResponse := decodeLotteryRouteResponse[map[string]interface{}](t, preview)
		require.True(t, previewResponse.Success)
		previewToken, ok := previewResponse.Data["token"].(string)
		require.True(t, ok)
		require.NotEmpty(t, previewToken)

		maskedParticipants := fixture.request(fixture.root, http.MethodGet, "/api/lottery/plans/"+strconv.Itoa(fixture.visiblePlan.Id)+"/participants", "", previewToken)
		require.Equal(t, http.StatusOK, maskedParticipants.Code)
		assert.NotContains(t, maskedParticipants.Body.String(), fixture.other.Username)
		assert.NotContains(t, maskedParticipants.Body.String(), fixture.other.DisplayName)

		previewResults := fixture.request(fixture.root, http.MethodGet, "/api/lottery/results/self", "", previewToken)
		require.Equal(t, http.StatusOK, previewResults.Code)
		assert.NotContains(t, previewResults.Body.String(), "OWN-REDEEM-CODE")
		assert.NotContains(t, previewResults.Body.String(), "OTHER-REDEEM-CODE")

		previewResultsPage := fixture.request(fixture.root, http.MethodGet, "/api/lottery/results/self/page", "", previewToken)
		require.Equal(t, http.StatusOK, previewResultsPage.Code)
		previewResultPageResponse := decodeLotteryRouteResponse[model.LotterySelfResultPage](t, previewResultsPage)
		require.True(t, previewResultPageResponse.Success)
		require.Len(t, previewResultPageResponse.Data.Items, 1)
		assert.Empty(t, previewResultPageResponse.Data.Items[0].RedemptionCode)
		assert.NotContains(t, previewResultsPage.Body.String(), "OWN-REDEEM-CODE")
		assert.NotContains(t, previewResultsPage.Body.String(), "OTHER-REDEEM-CODE")

		adminRequest := fixture.request(fixture.root, http.MethodGet, "/api/lottery/admin/plans/"+strconv.Itoa(fixture.visiblePlan.Id), "", previewToken)
		require.Equal(t, http.StatusForbidden, adminRequest.Code)

		markRead := fixture.request(fixture.root, http.MethodPost, "/api/lottery/notifications/self/read", fmt.Sprintf(`{"ids":[%d]}`, fixture.notification.Id), previewToken)
		require.Equal(t, http.StatusForbidden, markRead.Code)
		var notification model.LotteryNotification
		require.NoError(t, fixture.db.First(&notification, fixture.notification.Id).Error)
		assert.Zero(t, notification.ReadAt)
	})
}
