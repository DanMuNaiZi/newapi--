package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreConsumeUserSubscriptionPrefersUserPriority(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&SubscriptionPreConsumeRecord{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM subscription_pre_consume_records")
	})

	now := GetDBTimestamp()
	planPriorityOne := 1
	planPriorityTwo := 2
	userPriorityOne := 1
	userPriorityTwo := 2
	firstPlan := &SubscriptionPlan{Id: 9811, Title: "First", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1000, ConsumePriority: &planPriorityOne}
	secondPlan := &SubscriptionPlan{Id: 9812, Title: "Second", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1000, ConsumePriority: &planPriorityTwo}
	seedSubscriptionResetPlan(t, firstPlan)
	seedSubscriptionResetPlan(t, secondPlan)
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9821, UserId: 981, PlanId: firstPlan.Id, AmountTotal: 1000, EndTime: now + 3600, Status: "active", UserConsumePriority: &userPriorityTwo})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9822, UserId: 981, PlanId: secondPlan.Id, AmountTotal: 1000, EndTime: now + 7200, Status: "active", UserConsumePriority: &userPriorityOne})

	result, err := PreConsumeUserSubscription("priority-user", 981, "gpt-5", 0, 100)

	require.NoError(t, err)
	assert.Equal(t, 9822, result.UserSubscriptionId)
}

func TestPreConsumeUserSubscriptionUsesPlanPriorityBeforeExpiry(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&SubscriptionPreConsumeRecord{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM subscription_pre_consume_records")
	})

	now := GetDBTimestamp()
	planPriorityOne := 1
	planPriorityTwo := 2
	firstPlan := &SubscriptionPlan{Id: 9831, Title: "First", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1000, ConsumePriority: &planPriorityOne}
	secondPlan := &SubscriptionPlan{Id: 9832, Title: "Second", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1000, ConsumePriority: &planPriorityTwo}
	seedSubscriptionResetPlan(t, firstPlan)
	seedSubscriptionResetPlan(t, secondPlan)
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9841, UserId: 983, PlanId: firstPlan.Id, AmountTotal: 1000, EndTime: now + 7200, Status: "active"})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9842, UserId: 983, PlanId: secondPlan.Id, AmountTotal: 1000, EndTime: now + 3600, Status: "active"})

	result, err := PreConsumeUserSubscription("priority-plan", 983, "gpt-5", 0, 100)

	require.NoError(t, err)
	assert.Equal(t, 9841, result.UserSubscriptionId)
}

func TestResetUserSubscriptionConsumePriorityRestoresPlanPriority(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	planPriorityOne := 1
	planPriorityTwo := 2
	userPriorityOne := 1
	userPriorityTwo := 2
	firstPlan := &SubscriptionPlan{Id: 9851, Title: "First", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1000, ConsumePriority: &planPriorityOne}
	secondPlan := &SubscriptionPlan{Id: 9852, Title: "Second", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1000, ConsumePriority: &planPriorityTwo}
	seedSubscriptionResetPlan(t, firstPlan)
	seedSubscriptionResetPlan(t, secondPlan)
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9861, UserId: 985, PlanId: firstPlan.Id, AmountTotal: 1000, EndTime: now + 7200, Status: "active", UserConsumePriority: &userPriorityTwo})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9862, UserId: 985, PlanId: secondPlan.Id, AmountTotal: 1000, EndTime: now + 3600, Status: "active", UserConsumePriority: &userPriorityOne})

	require.NoError(t, ResetUserSubscriptionConsumePriority(985))
	result, err := PreConsumeUserSubscription("priority-reset", 985, "gpt-5", 0, 100)

	require.NoError(t, err)
	assert.Equal(t, 9861, result.UserSubscriptionId)
}
