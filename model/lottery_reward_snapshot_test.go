package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLotteryDrawStoresCompletePrizeSnapshot(t *testing.T) {
	userIDs := setupLotteryFixture(t)
	plan := &LotteryPlan{
		Title:                 "Prize snapshot lottery",
		Status:                LotteryPlanStatusOpen,
		EligibilityMode:       LotteryEligibilityAll,
		MaxParticipants:       2,
		RegistrationStartTime: common.GetTimestamp() - 60,
		DrawTime:              common.GetTimestamp() + 3600,
	}
	prizes := []*LotteryPrize{{
		Name:               "Original quota",
		Quantity:           1,
		RewardType:         LotteryRewardQuota,
		Quota:              321,
		FulfillmentMode:    LotteryFulfillmentAuto,
		ClaimExpireSeconds: 3600,
	}}
	require.NoError(t, CreateLotteryPlan(plan, nil, nil, prizes))
	require.NoError(t, JoinLotteryPlan(plan.Id, userIDs[0]))
	_, err := DrawLotteryPlan(plan.Id, LotteryDrawTriggerManual, "snapshot test")
	require.NoError(t, err)

	var result LotteryResult
	require.NoError(t, DB.Where("plan_id = ? AND user_id = ?", plan.Id, userIDs[0]).First(&result).Error)
	require.NotEmpty(t, result.PrizeSnapshot)
	var snapshot LotteryPrize
	require.NoError(t, common.Unmarshal([]byte(result.PrizeSnapshot), &snapshot))
	assert.Equal(t, "Original quota", snapshot.Name)
	assert.Equal(t, 321, snapshot.Quota)
	assert.Equal(t, LotteryRewardQuota, snapshot.RewardType)
	assert.Equal(t, plan.Id, snapshot.PlanId)
}
