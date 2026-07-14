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
		&LotteryPlan{},
		&LotteryPlanGroup{},
		&LotteryPlanUser{},
		&LotteryPrize{},
		&LotteryParticipant{},
		&LotteryDrawRun{},
		&LotteryResult{},
	))
	for _, table := range []interface{}{
		&LotteryResult{},
		&LotteryDrawRun{},
		&LotteryParticipant{},
		&LotteryPrize{},
		&LotteryPlanGroup{},
		&LotteryPlanUser{},
		&LotteryPlan{},
	} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "lottery-%").Delete(&User{}).Error)
	t.Cleanup(func() {
		for _, table := range []interface{}{
			&LotteryResult{},
			&LotteryDrawRun{},
			&LotteryParticipant{},
			&LotteryPrize{},
			&LotteryPlanGroup{},
			&LotteryPlanUser{},
			&LotteryPlan{},
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
		MaxParticipants:       3,
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
		MaxParticipants:       1,
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
