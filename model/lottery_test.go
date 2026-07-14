package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLotteryFixture(t *testing.T) []int {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(
		&Redemption{},
		&LotteryPlan{},
		&LotteryPlanGroup{},
		&LotteryPlanUser{},
		&LotteryPrize{},
		&LotteryParticipant{},
		&LotteryDrawRun{},
		&LotteryResult{},
		&Redemption{},
		&LotteryNotification{},
	))
	for _, table := range []interface{}{
		&LotteryResult{},
		&LotteryNotification{},
		&LotteryDrawRun{},
		&LotteryParticipant{},
		&LotteryPrize{},
		&LotteryPlanGroup{},
		&LotteryPlanUser{},
		&LotteryPlan{},
		&UserSubscription{},
		&SubscriptionPlan{},
	} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "lottery-%").Delete(&User{}).Error)
	t.Cleanup(func() {
		for _, table := range []interface{}{
			&LotteryResult{},
			&Redemption{},
			&LotteryNotification{},
			&LotteryDrawRun{},
			&LotteryParticipant{},
			&LotteryPrize{},
			&LotteryPlanGroup{},
			&LotteryPlanUser{},
			&LotteryPlan{},
			&UserSubscription{},
			&SubscriptionPlan{},
		} {
			require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
		}
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "lottery-%").Delete(&User{}).Error)
	})

	users := []*User{
		{Username: "lottery-a", Password: "password", Status: common.UserStatusEnabled, Group: "vip", AffCode: "lottery-aff-a"},
		{Username: "lottery-b", Password: "password", Status: common.UserStatusEnabled, Group: "vip", AffCode: "lottery-aff-b"},
		{Username: "lottery-c", Password: "password", Status: common.UserStatusEnabled, Group: "default", AffCode: "lottery-aff-c"},
	}
	require.NoError(t, DB.Create(&users).Error)
	return []int{users[0].Id, users[1].Id, users[2].Id}
}

func TestLotteryExplicitAllowListRequiresSelfJoin(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "VIP lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityUsers,
		MaxParticipants:       2,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(plan, []int{userIDs[0]}, nil, []*LotteryPrize{
		{Name: "VIP prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto},
	}))

	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	require.Error(t, JoinLotteryPlan(plan.Id, userIDs[1]))

	var count int64
	require.NoError(t, DB.Model(&LotteryParticipant{}).Where("plan_id = ? AND status = ?", plan.Id, LotteryParticipantStatusJoined).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestLotteryDrawHonorsPresetPrizeAndNoRepeatWinner(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Draw lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       4,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	prizes := []*LotteryPrize{
		{Name: "Grand prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto},
		{Name: "Second prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 50, FulfillmentMode: LotteryFulfillmentAuto},
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, prizes))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[1]))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[2]))
	require.NoError(t, SetLotteryParticipantPreset(plan.Id, userIDs[0], prizes[0].Id))

	run, err := DrawLotteryPlan(plan.Id, LotteryDrawTriggerManual, "test")
	require.NoError(t, err)
	assert.Equal(t, LotteryDrawRunStatusFinished, run.Status)

	var results []LotteryResult
	require.NoError(t, DB.Where("plan_id = ?", plan.Id).Order("prize_id asc").Find(&results).Error)
	require.Len(t, results, 2)
	assert.Equal(t, userIDs[0], results[0].UserId)
	assert.Equal(t, prizes[0].Id, results[0].PrizeId)
	assert.NotEqual(t, results[0].UserId, results[1].UserId)
}

func TestLotteryDrawRecordsEmptyCompletion(t *testing.T) {
	setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Empty lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       3,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() - 1,
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{
		{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto},
	}))

	run, err := DrawLotteryPlan(plan.Id, LotteryDrawTriggerScheduled, "")
	require.NoError(t, err)
	assert.Equal(t, LotteryDrawRunStatusEmpty, run.Status)

	var stored LotteryPlan
	require.NoError(t, DB.First(&stored, plan.Id).Error)
	assert.Equal(t, LotteryPlanStatusFinished, stored.Status)
}

func TestLotteryLeaveReleasesParticipantSlot(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Leave lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{
		{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto},
	}))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	require.NoError(t, LeaveLotteryPlan(plan.Id, userIDs[0]))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[1]))
}

func TestLotteryGroupEligibilityAndAdminExclusion(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	groupPlan := &LotteryPlan{
		Title:                 "Group lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityGroups,
		MaxParticipants:       3,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	prizes := []*LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto}}
	require.NoError(t, CreateLotteryPlan(groupPlan, nil, []string{"vip"}, prizes))
	require.NoError(t, JoinLotteryPlan(groupPlan.Id, userIDs[0]))
	require.Error(t, JoinLotteryPlan(groupPlan.Id, userIDs[2]))

	adminPlan := &LotteryPlan{
		Title:                 "Admin lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       3,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(adminPlan, nil, nil, []*LotteryPrize{
		{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto},
	}))
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userIDs[1]).Update("role", common.RoleAdminUser).Error)
	require.Error(t, JoinLotteryPlan(adminPlan.Id, userIDs[1]))
}

func TestLotteryAutoQuotaPrizeIsFulfilledAfterDraw(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Auto quota lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{
		{Name: "Quota", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 321, FulfillmentMode: LotteryFulfillmentAuto},
	}))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	_, err := DrawLotteryPlan(plan.Id, LotteryDrawTriggerManual, "test")
	require.NoError(t, err)

	var user User
	require.NoError(t, DB.First(&user, userIDs[0]).Error)
	assert.Equal(t, 321, user.Quota)

	var result LotteryResult
	require.NoError(t, DB.Where("plan_id = ? AND user_id = ?", plan.Id, userIDs[0]).First(&result).Error)
	assert.Equal(t, "fulfilled", result.FulfillmentStatus)
	var notificationCount int64
	require.NoError(t, DB.Model(&LotteryNotification{}).Where("plan_id = ? AND user_id = ? AND type = ?", plan.Id, userIDs[0], "lottery_result").Count(&notificationCount).Error)
	assert.Equal(t, int64(1), notificationCount)
}

func TestLotterySelfClaimPrizeCreditsQuotaOnce(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Claim quota lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{
		{Name: "Quota", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 123, FulfillmentMode: LotteryFulfillmentSelfClaim, ClaimExpireSeconds: 3600},
	}))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	_, err := DrawLotteryPlan(plan.Id, LotteryDrawTriggerManual, "test")
	require.NoError(t, err)

	var result LotteryResult
	require.NoError(t, DB.Where("plan_id = ? AND user_id = ?", plan.Id, userIDs[0]).First(&result).Error)
	require.NoError(t, ClaimLotteryResult(result.Id, userIDs[0]))
	require.Error(t, ClaimLotteryResult(result.Id, userIDs[0]))

	var user User
	require.NoError(t, DB.First(&user, userIDs[0]).Error)
	assert.Equal(t, 123, user.Quota)
}

func TestLotteryRedemptionCodePrizeCreatesRedeemableCode(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Code prize lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{{
		Name:            "Code prize",
		Quantity:        1,
		RewardType:      LotteryRewardQuota,
		Quota:           222,
		FulfillmentMode: LotteryFulfillmentRedemptionCode,
	}}))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	_, err := DrawLotteryPlan(plan.Id, LotteryDrawTriggerManual, "test")
	require.NoError(t, err)

	var result LotteryResult
	require.NoError(t, DB.Where("plan_id = ? AND user_id = ?", plan.Id, userIDs[0]).First(&result).Error)
	require.NotEmpty(t, result.RedemptionCode)
	assert.Equal(t, "issued", result.FulfillmentStatus)
	quota, err := Redeem(result.RedemptionCode, userIDs[0])
	require.NoError(t, err)
	assert.Equal(t, 222, quota)
}

func TestLotterySubscriptionRewardUsesSnapshotAndIgnoresPurchaseLimit(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &SubscriptionPlan{
		Title:              "Reward subscription",
		PriceAmount:        0,
		DurationUnit:       "month",
		DurationValue:      1,
		Enabled:            true,
		MaxPurchasePerUser: 1,
		TotalAmount:        456,
	}
	plan.NormalizeDefaults()
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := CreateUserSubscriptionFromPlanTx(tx, userIDs[0], plan, "order")
		return err
	}))

	lotteryPlan := &LotteryPlan{
		Title:                 "Subscription lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(lotteryPlan, nil, nil, []*LotteryPrize{
		{Name: "Subscription", Quantity: 1, RewardType: LotteryRewardSubscription, SubscriptionPlanId: plan.Id, FulfillmentMode: LotteryFulfillmentAuto},
	}))
	require.NoError(t, JoinLotteryPlan(lotteryPlan.Id, userIDs[0]))
	_, err := DrawLotteryPlan(lotteryPlan.Id, LotteryDrawTriggerManual, "test")
	require.NoError(t, err)

	var subscriptions []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", userIDs[0], plan.Id).Find(&subscriptions).Error)
	require.Len(t, subscriptions, 2)
	assert.Equal(t, "lottery", subscriptions[1].Source)
}

func TestLotteryJoiningMaximumParticipantsDrawsImmediately(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Full lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       1,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{
		{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto},
	}))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))

	var stored LotteryPlan
	require.NoError(t, DB.First(&stored, plan.Id).Error)
	assert.Equal(t, LotteryPlanStatusFinished, stored.Status)
}

func TestProcessLotteryScheduleOpensAndDrawsDuePlans(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	now := common.GetTimestamp()
	openPlan := &LotteryPlan{
		Title:                 "Scheduled open lottery",
		Status:                LotteryPlanStatusScheduled,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: now - 1,
		DrawTime:              now + 3600,
	}
	duePlan := &LotteryPlan{
		Title:                 "Scheduled draw lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: now - 3600,
		DrawTime:              now + 60,
	}
	prize := []*LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto}}
	require.NoError(t, CreateLotteryPlan(openPlan, nil, nil, prize))
	require.NoError(t, CreateLotteryPlan(duePlan, nil, nil, []*LotteryPrize{{Name: "Due prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto}}))
	require.NoError(t, JoinLotteryPlan(duePlan.Id, userIDs[0]))
	require.NoError(t, DB.Model(&LotteryPlan{}).Where("id = ?", duePlan.Id).Update("draw_time", now-1).Error)

	require.NoError(t, ProcessLotterySchedule(now))
	require.NoError(t, DB.First(&openPlan, openPlan.Id).Error)
	require.NoError(t, DB.First(&duePlan, duePlan.Id).Error)
	assert.Equal(t, LotteryPlanStatusOpen, openPlan.Status)
	assert.Equal(t, LotteryPlanStatusFinished, duePlan.Status)
}

func TestListLotteryPlansForUserOnlyReturnsEligibleAndParticipatedPlans(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	now := common.GetTimestamp()
	publicPlan := &LotteryPlan{
		Title:                 "Public lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       5,
		RegistrationStartTime: now - 60,
		DrawTime:              now + 3600,
	}
	privatePlan := &LotteryPlan{
		Title:                 "Private lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityUsers,
		MaxParticipants:       5,
		RegistrationStartTime: now - 60,
		DrawTime:              now + 3600,
	}
	historyPlan := &LotteryPlan{
		Title:                 "History lottery",
		Status:                LotteryPlanStatusFinished,
		EligibilityMode:       LotteryEligibilityUsers,
		MaxParticipants:       5,
		RegistrationStartTime: now - 7200,
		DrawTime:              now - 3600,
	}
	prize := []*LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto}}
	require.NoError(t, CreateLotteryPlan(publicPlan, nil, nil, prize))
	require.NoError(t, CreateLotteryPlan(privatePlan, []int{userIDs[0]}, nil, []*LotteryPrize{{Name: "Private prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto}}))
	require.NoError(t, CreateLotteryPlan(historyPlan, []int{userIDs[0]}, nil, []*LotteryPrize{{Name: "History prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto}}))
	require.NoError(t, DB.Create(&LotteryParticipant{PlanId: historyPlan.Id, UserId: userIDs[0], UserGroup: "vip", Weight: 100, Status: LotteryParticipantStatusJoined, JoinedAt: now - 4000}).Error)

	visibleToAllowed, err := ListLotteryPlansForUser(userIDs[0])
	require.NoError(t, err)
	assert.Len(t, visibleToAllowed, 3)

	visibleToOther, err := ListLotteryPlansForUser(userIDs[1])
	require.NoError(t, err)
	require.Len(t, visibleToOther, 1)
	assert.Equal(t, publicPlan.Id, visibleToOther[0].Id)
}

func TestSetLotteryParticipantWeightRejectsInvalidValues(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{Title: "Weight lottery", Status: LotteryPlanStatusOpen, EligibilityMode: LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: common.GetTimestamp() - 60, DrawTime: common.GetTimestamp() + 3600}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{{Name: "Prize", Quantity: 1, RewardType: LotteryRewardQuota, Quota: 100, FulfillmentMode: LotteryFulfillmentAuto}}))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	require.Error(t, SetLotteryParticipantWeight(plan.Id, userIDs[0], 0))
	require.Error(t, SetLotteryParticipantWeight(plan.Id, userIDs[0], 1000001))
	require.NoError(t, SetLotteryParticipantWeight(plan.Id, userIDs[0], 500))

	var participant LotteryParticipant
	require.NoError(t, DB.Where("plan_id = ? AND user_id = ?", plan.Id, userIDs[0]).First(&participant).Error)
	assert.Equal(t, 500, participant.Weight)
}

func TestPublishedLotteryPlanOnlyAllowsCopyAndDelayedDrawOrCancellation(t *testing.T) {
	setupLotteryFixture(t)
	now := common.GetTimestamp()
	plan := &LotteryPlan{
		Title:                 "Published lottery",
		Description:           "Original copy",
		Status:                LotteryPlanStatusScheduled,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       5,
		RegistrationStartTime: now + 60,
		DrawTime:              now + 3600,
	}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, []*LotteryPrize{{
		Name:            "Prize",
		Quantity:        1,
		RewardType:      LotteryRewardQuota,
		Quota:           100,
		FulfillmentMode: LotteryFulfillmentAuto,
	}}))

	updated, err := UpdatePublishedLotteryPlan(plan.Id, LotteryPlanPublishedUpdate{
		Title:       stringPointer("Updated lottery"),
		Description: stringPointer("Updated copy"),
		DrawTime:    int64Pointer(now + 7200),
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated lottery", updated.Title)
	assert.Equal(t, "Updated copy", updated.Description)
	assert.Equal(t, now+7200, updated.DrawTime)

	_, err = UpdatePublishedLotteryPlan(plan.Id, LotteryPlanPublishedUpdate{
		DrawTime: int64Pointer(now + 3600),
	})
	require.Error(t, err)

	require.NoError(t, CancelLotteryPlan(plan.Id))
	var stored LotteryPlan
	require.NoError(t, DB.First(&stored, plan.Id).Error)
	assert.Equal(t, LotteryPlanStatusCancelled, stored.Status)
	require.Error(t, CancelLotteryPlan(plan.Id))
}

func TestLotteryWinnerNotificationsTrackExternalDelivery(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	notification := &LotteryNotification{
		UserId:    userIDs[0],
		PlanId:    1,
		Type:      "lottery_result",
		Content:   "You won a lottery reward.",
		CreatedAt: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(notification).Error)

	pending, err := ListPendingLotteryWinnerNotifications(10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	marked, err := MarkLotteryNotificationExternallyNotified(notification.Id, common.GetTimestamp())
	require.NoError(t, err)
	assert.True(t, marked)

	pending, err = ListPendingLotteryWinnerNotifications(10)
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func stringPointer(value string) *string {
	return &value
}

func int64Pointer(value int64) *int64 {
	return &value
}
