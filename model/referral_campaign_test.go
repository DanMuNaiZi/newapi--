package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupReferralCampaignFixture(t *testing.T) {
	t.Helper()
	setupRewardFixture(t)
	require.NoError(t, DB.AutoMigrate(&ReferralCampaign{}, &ReferralCampaignEvent{}, &ReferralCampaignScheduleLock{}))
	for _, table := range []interface{}{&ReferralCampaignEvent{}, &ReferralCampaign{}, &ReferralCampaignScheduleLock{}} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "referral-%").Delete(&User{}).Error)
	originalInviterQuota := common.QuotaForInviter
	t.Cleanup(func() {
		common.QuotaForInviter = originalInviterQuota
		for _, table := range []interface{}{&ReferralCampaignEvent{}, &ReferralCampaign{}, &ReferralCampaignScheduleLock{}} {
			require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
		}
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "referral-%").Delete(&User{}).Error)
	})
}

func createReferralUsers(t *testing.T, count int) []*User {
	t.Helper()
	users := make([]*User, 0, count)
	for i := 0; i < count; i++ {
		users = append(users, &User{
			Username: "referral-user-" + common.GetRandomString(8),
			Password: "password",
			Status:   common.UserStatusEnabled,
			AffCode:  "referral-aff-" + common.GetRandomString(8),
		})
	}
	require.NoError(t, DB.Create(&users).Error)
	return users
}

func createQuotaReferralCampaign(t *testing.T, quota int, perInviterLimit int, totalLimit int) *ReferralCampaign {
	t.Helper()
	rawReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: quota})
	require.NoError(t, err)
	now := common.GetTimestamp()
	campaign := &ReferralCampaign{
		Title:                   "Referral campaign",
		Enabled:                 true,
		StartTime:               now - 60,
		EndTime:                 now + 3600,
		ActivationWindowSeconds: 3600,
		MaxRewardsPerInviter:    perInviterLimit,
		TotalRewardLimit:        totalLimit,
		RewardSnapshotJSON:      rawReward,
		CreatedBy:               1,
	}
	require.NoError(t, CreateReferralCampaign(campaign))
	return campaign
}

func TestInvitationWithoutCampaignKeepsLegacyInviterReward(t *testing.T) {
	setupReferralCampaignFixture(t)
	common.QuotaForInviter = 120
	users := createReferralUsers(t, 2)

	usedCampaign, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	assert.False(t, usedCampaign)

	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, 120, inviter.AffQuota)
	assert.Equal(t, 120, inviter.AffHistoryQuota)
}

func TestInvitationWithoutCampaignAndWithoutLegacyRewardKeepsLegacyNoop(t *testing.T) {
	setupReferralCampaignFixture(t)
	common.QuotaForInviter = 0
	users := createReferralUsers(t, 2)

	usedCampaign, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	assert.False(t, usedCampaign)

	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Zero(t, inviter.AffCount)
	assert.Zero(t, inviter.AffQuota)
	assert.Zero(t, inviter.AffHistoryQuota)
}

func TestReferralCampaignActivatesOnlyOnEligibleCallAndRewardsOnce(t *testing.T) {
	setupReferralCampaignFixture(t)
	common.QuotaForInviter = 120
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)

	usedCampaign, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	assert.True(t, usedCampaign)

	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Zero(t, inviter.AffQuota)

	require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "public-request", constant.PublicPoolGroup))
	var pending ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&pending).Error)
	assert.Equal(t, ReferralEventPending, pending.Status)

	require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "paid-request", "default"))
	require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "paid-request-again", "default"))
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, 250, inviter.Quota)
	require.NoError(t, DB.First(&pending, pending.Id).Error)
	assert.Equal(t, ReferralEventRewarded, pending.Status)
	assert.Equal(t, "paid-request", pending.FirstSuccessRequestId)
}

func TestReferralCampaignRewardLimitReservesFailedAndSucceededEvents(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 3)
	createQuotaReferralCampaign(t, 100, 1, 0)
	require.NoError(t, func() error { _, err := HandleInvitationRegistration(users[0].Id, users[1].Id); return err }())
	require.NoError(t, func() error { _, err := HandleInvitationRegistration(users[0].Id, users[2].Id); return err }())

	require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "first", "default"))
	require.NoError(t, ActivateReferralCampaignForUser(users[2].Id, "second", "default"))

	var second ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[2].Id).First(&second).Error)
	assert.Equal(t, ReferralEventLimitReached, second.Status)
	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, 100, inviter.Quota)
	campaigns, err := ListReferralCampaignsForAdmin()
	require.NoError(t, err)
	require.Len(t, campaigns, 1)
	assert.EqualValues(t, 2, campaigns[0].ActivatedCount)
	assert.EqualValues(t, 1, campaigns[0].RewardedCount)
}

func TestReferralCampaignTotalRewardLimitAppliesAcrossInviters(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 4)
	createQuotaReferralCampaign(t, 100, 0, 1)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	_, err = HandleInvitationRegistration(users[2].Id, users[3].Id)
	require.NoError(t, err)

	require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "first", "default"))
	require.NoError(t, ActivateReferralCampaignForUser(users[3].Id, "second", "default"))
	var second ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[3].Id).First(&second).Error)
	assert.Equal(t, ReferralEventLimitReached, second.Status)
}

func TestReferralCampaignFailedRewardCanBeRetriedWithoutDuplicateCredit(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", users[0].Id).Update("quota", common.MaxQuota).Error)
	createQuotaReferralCampaign(t, 100, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)

	err = ActivateReferralCampaignForUser(users[1].Id, "first", "default")
	require.Error(t, err)
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	assert.Equal(t, ReferralEventRewardFailed, event.Status)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", users[0].Id).Update("quota", common.MaxQuota-100).Error)
	retried, err := RetryReferralCampaignReward(event.Id)
	require.NoError(t, err)
	assert.Equal(t, ReferralEventRewarded, retried.Status)
	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, common.MaxQuota, inviter.Quota)
}

func TestReferralCampaignAllowsOnlyOneEnabledFutureCampaign(t *testing.T) {
	setupReferralCampaignFixture(t)
	createQuotaReferralCampaign(t, 100, 0, 0)
	rawReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 100})
	require.NoError(t, err)
	now := common.GetTimestamp()
	err = CreateReferralCampaign(&ReferralCampaign{
		Title:                   "Overlapping campaign",
		Enabled:                 true,
		StartTime:               now + 60,
		EndTime:                 now + 7200,
		ActivationWindowSeconds: 3600,
		RewardSnapshotJSON:      rawReward,
	})
	assert.ErrorContains(t, err, "already enabled")
}

func TestReferralCampaignSelfViewDoesNotPresentFutureCampaignAsActive(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 1)
	rawReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 100})
	require.NoError(t, err)
	now := common.GetTimestamp()
	require.NoError(t, CreateReferralCampaign(&ReferralCampaign{
		Title:                   "Future campaign",
		Enabled:                 true,
		StartTime:               now + 3600,
		EndTime:                 now + 7200,
		ActivationWindowSeconds: defaultReferralActivationWindowSeconds,
		RewardSnapshotJSON:      rawReward,
	}))

	view, err := GetReferralCampaignForSelf(users[0].Id)
	require.NoError(t, err)
	assert.Nil(t, view.Campaign)
}

func TestReferralCampaignRejectsLimitsOutsideCrossDatabaseIntegerRange(t *testing.T) {
	setupReferralCampaignFixture(t)
	rawReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 100})
	require.NoError(t, err)

	err = CreateReferralCampaign(&ReferralCampaign{
		Title:                   "oversized limit",
		Enabled:                 false,
		StartTime:               common.GetTimestamp() + 60,
		EndTime:                 common.GetTimestamp() + 3600,
		ActivationWindowSeconds: defaultReferralActivationWindowSeconds,
		MaxRewardsPerInviter:    referralCampaignLimitMax + 1,
		RewardSnapshotJSON:      rawReward,
	})
	assert.ErrorContains(t, err, "limits")
}

func TestReferralCampaignUsesTheRegistrationActivationWindowAfterCampaignEnd(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	campaign := createQuotaReferralCampaign(t, 100, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	require.NoError(t, DB.Model(campaign).Update("end_time", common.GetTimestamp()-1).Error)

	require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "after-end", "default"))
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	assert.Equal(t, ReferralEventRewarded, event.Status)
}

func TestReferralCampaignExpiresAnUnactivatedRegistration(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 100, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&ReferralCampaignEvent{}).
		Where("invitee_user_id = ?", users[1].Id).
		Update("activation_deadline", common.GetTimestamp()-1).Error)

	require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "late", "default"))
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	assert.Equal(t, ReferralEventExpired, event.Status)
	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Zero(t, inviter.Quota)
}

func TestReferralActivationEligibilityRejectsPoolAndFailedStreams(t *testing.T) {
	assert.False(t, isReferralActivationEligible(RecordConsumeLogParams{ActivateReferral: false, Group: "default"}))
	assert.False(t, isReferralActivationEligible(RecordConsumeLogParams{ActivateReferral: true, Group: constant.PublicPoolGroup}))
	assert.False(t, isReferralActivationEligible(RecordConsumeLogParams{
		ActivateReferral: true,
		Group:            "default",
		Other: map[string]interface{}{
			"stream_status": map[string]interface{}{"status": "error", "end_reason": "client_gone"},
		},
	}))
	assert.True(t, isReferralActivationEligible(RecordConsumeLogParams{
		ActivateReferral: true,
		Group:            "default",
		Other: map[string]interface{}{
			"stream_status": map[string]interface{}{"status": "ok", "end_reason": "eof"},
		},
	}))
}
