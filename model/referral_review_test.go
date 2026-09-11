package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordReferralTestUsage(userId int, requestId, group string) error {
	return ActivateReferralCampaignForUser(userId, requestId, group, RecordConsumeLogParams{ActivateReferral: true, BillingSettled: true, Group: group, Quota: 1})
}

func TestReferralCallNeverAwardsBeforeManualApproval(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	require.NoError(t, recordReferralTestUsage(users[1].Id, "paid-request", "default"))
	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Zero(t, inviter.Quota, "a successful call must not automatically issue a referral reward")
}

func TestReferralReviewRequiresPositiveEligibleConsumption(t *testing.T) {
	for _, tc := range []struct {
		name  string
		usage RecordConsumeLogParams
	}{
		{"zero charge", RecordConsumeLogParams{ActivateReferral: true, BillingSettled: true, Group: "default"}},
		{"public pool", RecordConsumeLogParams{ActivateReferral: true, BillingSettled: true, Group: constant.PublicPoolGroup, Quota: 50}},
		{"failed request", RecordConsumeLogParams{Group: "default", Quota: 50}},
		{"unknown stream", RecordConsumeLogParams{ActivateReferral: true, BillingSettled: true, Group: "default", Quota: 50, IsStream: true}},
		{"client disconnected", RecordConsumeLogParams{ActivateReferral: true, BillingSettled: true, Group: "default", Quota: 50, Other: map[string]interface{}{"stream_status": map[string]interface{}{"status": "error", "end_reason": "client_gone"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupReferralCampaignFixture(t)
			users := createReferralUsers(t, 2)
			createQuotaReferralCampaign(t, 250, 0, 0)
			_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
			require.NoError(t, err)
			require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "request", tc.usage.Group, tc.usage))
			var event ReferralCampaignEvent
			require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
			_, err = ReviewReferralCampaignEvent(event.Id, 1, "approve", "")
			assert.ErrorContains(t, err, "actual quota consumption")
			var grants int64
			require.NoError(t, DB.Model(&RewardGrant{}).Count(&grants).Error)
			assert.Zero(t, grants)
		})
	}
}

func TestReferralDualRewardsFreezeAtRegistrationAndRetryOnlyMissingSide(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	campaign := createQuotaReferralCampaign(t, 250, 0, 0)
	inviteeRaw, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 80})
	require.NoError(t, err)
	require.NoError(t, DB.Model(campaign).Update("invitee_reward_snapshot", inviteeRaw).Error)
	_, err = HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	require.NoError(t, DB.Model(campaign).Updates(map[string]interface{}{"reward_snapshot": "", "invitee_reward_snapshot": ""}).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", users[1].Id).Update("quota", common.MaxQuota).Error)
	require.NoError(t, recordReferralTestUsage(users[1].Id, "positive", "default"))
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	_, err = RetryReferralCampaignReward(event.Id)
	require.Error(t, err)
	_, err = ReviewReferralCampaignEvent(event.Id, 1, "approve", "reviewed")
	require.Error(t, err) // Invitee balance cap; inviter already succeeded.
	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, 250, inviter.Quota)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", users[1].Id).Update("quota", 0).Error)
	retried, err := RetryReferralCampaignReward(event.Id)
	require.NoError(t, err)
	assert.Equal(t, ReferralEventRewarded, retried.Status)
	_, err = ReviewReferralCampaignEvent(event.Id, 1, "approve", "duplicate approval")
	require.NoError(t, err)
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, 250, inviter.Quota)
	var invitee User
	require.NoError(t, DB.First(&invitee, users[1].Id).Error)
	assert.Equal(t, 80, invitee.Quota)
	var grants int64
	require.NoError(t, DB.Model(&RewardGrant{}).Count(&grants).Error)
	assert.EqualValues(t, 2, grants)
}

func TestReferralLegacyUnapprovedRetryAndRejectedEventCannotPay(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	require.NoError(t, DB.Model(&event).Update("status", ReferralEventRewardFailed).Error)
	_, err = RetryReferralCampaignReward(event.Id)
	assert.ErrorContains(t, err, "manual approval")
	require.NoError(t, recordReferralTestUsage(users[1].Id, "paid", "default"))
	_, err = ReviewReferralCampaignEvent(event.Id, 1, "reject", "not a genuine referral")
	require.NoError(t, err)
	_, err = ReviewReferralCampaignEvent(event.Id, 1, "approve", "")
	require.Error(t, err)
	_, err = RetryReferralCampaignReward(event.Id)
	require.Error(t, err)
}

func TestReferralReviewPrivacyAndDurableConsumptionEvidence(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	require.NoError(t, recordReferralTestUsage(users[1].Id, "paid", "default"))
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	log := &Log{UserId: users[1].Id, CreatedAt: common.GetTimestamp(), Type: LogTypeConsume, ModelName: "private-upstream", Quota: 12, PromptTokens: 20, CompletionTokens: 3, Group: "default", RequestId: "paid", TokenName: "secret-token-label", Other: `{"request_model_name":"requested-model","upstream_model_name":"private-upstream","admin_info":{"api_key":"do-not-return"}}`}
	require.NoError(t, LOG_DB.Create(log).Error)
	t.Cleanup(func() { require.NoError(t, LOG_DB.Delete(log).Error) })
	view, err := GetReferralCampaignEventReview(event.Id)
	require.NoError(t, err)
	assert.True(t, view.HasConsumption)
	assert.True(t, view.CanApprove)
	assert.EqualValues(t, 12, view.Usage.ConsumedQuota)
	require.Len(t, view.Usage.RecentCalls, 1)
	assert.Equal(t, "requested-model", view.Usage.RecentCalls[0].ModelName)
	assert.Equal(t, users[1].Username, view.RelatedEvents[0].InviteeUsername)
	raw, err := common.Marshal(view)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "private-upstream")
	assert.NotContains(t, string(raw), "do-not-return")
	assert.NotContains(t, string(raw), "secret-token-label")
	require.NoError(t, LOG_DB.Delete(log).Error)
	view, err = GetReferralCampaignEventReview(event.Id)
	require.NoError(t, err)
	assert.True(t, view.CanApprove, "pruning logs must not erase the saved qualifying settlement")
	assert.Empty(t, view.Usage.RecentCalls)
}

func TestReferralConcurrentApprovalCannotDoubleCredit(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	require.NoError(t, recordReferralTestUsage(users[1].Id, "paid", "default"))
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	start := make(chan struct{})
	var wg sync.WaitGroup
	errors := make([]error, 2)
	for i := range errors {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, errors[index] = ReviewReferralCampaignEvent(event.Id, index+10, "approve", "checked")
		}(i)
	}
	close(start)
	wg.Wait()
	for _, err := range errors {
		require.NoError(t, err)
	}
	var inviter User
	require.NoError(t, DB.First(&inviter, users[0].Id).Error)
	assert.Equal(t, 250, inviter.Quota)
	var grants int64
	require.NoError(t, DB.Model(&RewardGrant{}).Count(&grants).Error)
	assert.EqualValues(t, 1, grants)
}

func TestReferralHistoricalUnpaidEventUsesRetainedChargeButNoNewInviteeReward(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	require.NoError(t, DB.Model(&event).Update("status", ReferralEventRewardPending).Error)
	log := &Log{UserId: users[1].Id, CreatedAt: event.RegisteredAt, Type: LogTypeConsume, Quota: 1, Group: "default", Other: `{"billing_settlement_succeeded":true,"referral_activation_eligible":true}`}
	require.NoError(t, LOG_DB.Create(log).Error)
	t.Cleanup(func() { require.NoError(t, LOG_DB.Delete(log).Error) })
	_, err = ReviewReferralCampaignEvent(event.Id, 1, "approve", "legacy checked")
	require.NoError(t, err)
	var invitee User
	require.NoError(t, DB.First(&invitee, users[1].Id).Error)
	assert.Zero(t, invitee.Quota)
}

func TestCampaignRegistrationDoesNotIssueLegacyInviteeQuota(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)
	previousQuota := common.QuotaForInvitee
	previousPayment := *operation_setting.GetPaymentSetting()
	common.QuotaForInvitee = 500
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() {
		common.QuotaForInvitee = previousQuota
		*operation_setting.GetPaymentSetting() = previousPayment
	})
	finishInvitationRegistration(users[1].Id, users[0].Id)
	var invitee User
	require.NoError(t, DB.First(&invitee, users[1].Id).Error)
	assert.Zero(t, invitee.Quota)
	var events int64
	require.NoError(t, DB.Model(&ReferralCampaignEvent{}).Where("invitee_user_id = ?", invitee.Id).Count(&events).Error)
	assert.EqualValues(t, 1, events)
}

func TestReferralSelfResponseExcludesReviewAndUsageEvidence(t *testing.T) {
	setupReferralCampaignFixture(t)
	users := createReferralUsers(t, 2)
	createQuotaReferralCampaign(t, 250, 0, 0)
	_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
	require.NoError(t, err)
	require.NoError(t, recordReferralTestUsage(users[1].Id, "private-request-id", "default"))
	var event ReferralCampaignEvent
	require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
	_, err = ReviewReferralCampaignEvent(event.Id, 42, "reject", "private-review-note")
	require.NoError(t, err)
	view, err := GetReferralCampaignForSelf(users[0].Id)
	require.NoError(t, err)
	require.Len(t, view.Events, 1)
	assert.Equal(t, ReferralEventRejected, view.Events[0].Status)
	raw, err := common.Marshal(view)
	require.NoError(t, err)
	for _, private := range []string{"private-request-id", "private-review-note", "reviewed_by", "review_remark", "qualified_quota", "qualified_request_id", "first_success_request_id"} {
		assert.NotContains(t, string(raw), private)
	}
}

func TestReferralLegacyChargeRequiresVerifiedSuccessfulSettlement(t *testing.T) {
	for _, other := range []string{`{}`, `{"violation_fee":true}`, `{"billing_settlement_succeeded":false,"referral_activation_eligible":true}`, `{"billing_settlement_succeeded":true,"referral_activation_eligible":false}`} {
		t.Run(other, func(t *testing.T) {
			setupReferralCampaignFixture(t)
			users := createReferralUsers(t, 2)
			createQuotaReferralCampaign(t, 250, 0, 0)
			_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
			require.NoError(t, err)
			var event ReferralCampaignEvent
			require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
			require.NoError(t, DB.Model(&event).Update("status", ReferralEventRewardPending).Error)
			log := &Log{UserId: users[1].Id, CreatedAt: event.RegisteredAt, Type: LogTypeConsume, Quota: 50, Group: "default", Other: other}
			require.NoError(t, LOG_DB.Create(log).Error)
			t.Cleanup(func() { require.NoError(t, LOG_DB.Delete(log).Error) })
			_, err = ReviewReferralCampaignEvent(event.Id, 1, "approve", "")
			assert.ErrorContains(t, err, "actual quota consumption")
		})
	}
}

func TestReferralLegacyEvidenceSurvivesSubsequentCall(t *testing.T) {
	for _, afterDeadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "zero charge within window", true: "call after deadline"}[afterDeadline], func(t *testing.T) {
			setupReferralCampaignFixture(t)
			users := createReferralUsers(t, 2)
			createQuotaReferralCampaign(t, 250, 0, 0)
			_, err := HandleInvitationRegistration(users[0].Id, users[1].Id)
			require.NoError(t, err)
			var event ReferralCampaignEvent
			require.NoError(t, DB.Where("invitee_user_id = ?", users[1].Id).First(&event).Error)
			now := common.GetTimestamp()
			deadline := now + 100
			if afterDeadline {
				deadline = now - 10
			}
			require.NoError(t, DB.Model(&event).Updates(map[string]interface{}{"status": ReferralEventRewardPending, "registered_at": now - 100, "activation_deadline": deadline}).Error)
			log := &Log{UserId: users[1].Id, CreatedAt: now - 50, Type: LogTypeConsume, Quota: 25, Group: "default", RequestId: "verified-old-charge", Other: `{"billing_settlement_succeeded":true,"referral_activation_eligible":true}`}
			require.NoError(t, LOG_DB.Create(log).Error)
			t.Cleanup(func() { require.NoError(t, LOG_DB.Delete(log).Error) })
			require.NoError(t, ActivateReferralCampaignForUser(users[1].Id, "later-free-call", "default", RecordConsumeLogParams{ActivateReferral: true, BillingSettled: true, Group: "default"}))
			view, err := GetReferralCampaignEventReview(event.Id)
			require.NoError(t, err)
			assert.True(t, view.CanApprove)
			assert.Equal(t, 25, view.Event.QualifiedQuota)
			_, err = ReviewReferralCampaignEvent(event.Id, 1, "approve", "verified")
			require.NoError(t, err)
		})
	}
}
