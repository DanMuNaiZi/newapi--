package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRewardFixture(t *testing.T) {
	t.Helper()
	setupLotteryFixture(t)
	require.NoError(t, DB.AutoMigrate(&RewardGrant{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RewardGrant{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RewardGrant{}).Error)
	})
}

func TestGrantRewardCreditsQuotaExactlyOnce(t *testing.T) {
	setupRewardFixture(t)
	user := &User{Username: "lottery-reward-user", Password: "password", Status: common.UserStatusEnabled, AffCode: "lottery-reward-user"}
	require.NoError(t, DB.Create(user).Error)
	snapshot := RewardSnapshot{Type: RewardTypeQuota, Quota: 500}

	first, err := GrantReward(RewardSourcePublicPool, 101, user.Id, snapshot)
	require.NoError(t, err)
	second, err := GrantReward(RewardSourcePublicPool, 101, user.Id, snapshot)
	require.NoError(t, err)

	assert.Equal(t, first.Id, second.Id)
	assert.Equal(t, RewardGrantStatusSucceeded, second.Status)
	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, 500, stored.Quota)
}

func TestGrantRewardKeepsFailedDeliveryRetryable(t *testing.T) {
	setupRewardFixture(t)
	grant, err := GrantReward(RewardSourcePublicPool, 202, 999999, RewardSnapshot{Type: RewardTypeQuota, Quota: 100})
	require.Error(t, err)
	require.NotNil(t, grant)
	assert.Equal(t, RewardGrantStatusFailed, grant.Status)
	assert.NotEmpty(t, grant.FailureReason)
	assert.Equal(t, 1, grant.RetryCount)
}

func TestGrantRewardCreatesFrozenSubscriptionExactlyOnce(t *testing.T) {
	setupRewardFixture(t)
	user := &User{Username: "subscription-reward-user", Password: "password", Status: common.UserStatusEnabled, Group: "default", AffCode: "subscription-reward-user"}
	require.NoError(t, DB.Create(user).Error)
	plan := &SubscriptionPlan{Title: "Frozen reward", DurationUnit: "month", DurationValue: 1, Enabled: true, TotalAmount: 500, UpgradeGroup: "vip"}
	plan.NormalizeDefaults()
	require.NoError(t, DB.Create(plan).Error)
	rawPlan, err := common.Marshal(plan)
	require.NoError(t, err)
	snapshot := RewardSnapshot{
		Type:                     RewardTypeSubscription,
		SubscriptionPlanId:       plan.Id,
		SubscriptionPlanTitle:    plan.Title,
		SubscriptionPlanSnapshot: string(rawPlan),
	}

	_, err = GrantReward(RewardSourceReferralCampaign, 303, user.Id, snapshot)
	require.NoError(t, err)
	_, err = GrantReward(RewardSourceReferralCampaign, 303, user.Id, snapshot)
	require.NoError(t, err)
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).
		Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).
		Count(&count).Error)
	assert.EqualValues(t, 1, count)
	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, "vip", stored.Group)
}

func TestListRewardSubscriptionPlanOptionsExposesOnlyEnabledPlans(t *testing.T) {
	setupRewardFixture(t)
	enabled := &SubscriptionPlan{Title: "Enabled reward plan", DurationUnit: "month", DurationValue: 1, Enabled: true}
	disabled := &SubscriptionPlan{Title: "Disabled reward plan", DurationUnit: "month", DurationValue: 1, Enabled: true}
	require.NoError(t, DB.Create([]*SubscriptionPlan{enabled, disabled}).Error)
	require.NoError(t, DB.Model(disabled).Update("enabled", false).Error)

	plans, err := ListRewardSubscriptionPlanOptions()
	require.NoError(t, err)

	options := make(map[int]RewardSubscriptionPlanOption, len(plans))
	for _, plan := range plans {
		options[plan.Id] = plan
	}
	assert.Equal(t, enabled.Title, options[enabled.Id].Title)
	_, exists := options[disabled.Id]
	assert.False(t, exists)
}
