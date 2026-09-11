package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

// Referral review deliberately returns a small DTO, never User, Log.Other,
// provider credentials, IP addresses, or upstream model diagnostics.
type ReferralReviewUser struct {
	Id              int    `json:"id"`
	Username        string `json:"username"`
	CreatedAt       int64  `json:"created_at"`
	GithubId        string `json:"github_id"`
	GithubCreatedAt int64  `json:"github_created_at"`
	GithubAgeExempt bool   `json:"github_age_exempt"`
}

type ReferralReviewCall struct {
	Id               int    `json:"id"`
	CreatedAt        int64  `json:"created_at"`
	RequestId        string `json:"request_id"`
	ModelName        string `json:"model_name"`
	Quota            int    `json:"quota"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Successful       bool   `json:"successful"`
	Qualifying       bool   `json:"qualifying"`
	PublicPool       bool   `json:"public_pool"`
}

type ReferralReviewUsage struct {
	Requests      int64                `json:"requests"`
	ConsumedQuota int64                `json:"consumed_quota"`
	RecentCalls   []ReferralReviewCall `json:"recent_calls"`
	LogsAvailable bool                 `json:"logs_available"`
	PeriodStart   int64                `json:"period_start"`
	PeriodEnd     int64                `json:"period_end"`
}

type ReferralEventReview struct {
	Event          ReferralCampaignEvent   `json:"event"`
	Inviter        ReferralReviewUser      `json:"inviter"`
	Invitee        ReferralReviewUser      `json:"invitee"`
	HasConsumption bool                    `json:"has_consumption"`
	CanApprove     bool                    `json:"can_approve"`
	Usage          ReferralReviewUsage     `json:"usage"`
	RelatedEvents  []ReferralCampaignEvent `json:"related_events"`
	RewardGrants   []RewardGrant           `json:"reward_grants"`
}

// Keep the first positive settlement in the primary DB so pruning or disabling
// the optional log database cannot erase an already earned right to review.
// Historical unpaid events can use retained logs, but never infer a charge from
// a model's price or an activation flag alone.
func referralConsumptionEvidence(event ReferralCampaignEvent) (int, int64, string, error) {
	if event.QualifiedQuota > 0 && event.QualifiedQuota <= common.MaxQuota && event.QualifiedAt >= event.RegisteredAt && event.QualifiedAt <= event.ActivationDeadline {
		return event.QualifiedQuota, event.QualifiedAt, event.QualifiedRequestId, nil
	}
	if LOG_DB == nil {
		return 0, 0, "", nil
	}
	// A bounded legacy fallback: unknown/trimmed evidence cannot authorize payment.
	var logs []Log
	err := LOG_DB.Where("user_id = ? AND type = ? AND quota > 0 AND created_at >= ? AND created_at <= ?", event.InviteeUserId, LogTypeConsume, event.RegisteredAt, event.ActivationDeadline).
		Order("created_at ASC").Order("id ASC").Limit(100).Find(&logs).Error
	if err != nil {
		return 0, 0, "", err
	}
	for _, log := range logs {
		other, err := common.StrToMap(log.Other)
		if err != nil {
			continue
		}
		if log.Quota <= common.MaxQuota && isReferralActivationEligible(RecordConsumeLogParams{ActivateReferral: other["referral_activation_eligible"] == true, BillingSettled: other["billing_settlement_succeeded"] == true, Group: log.Group, Quota: log.Quota, IsStream: log.IsStream, Other: other}) {
			return log.Quota, log.CreatedAt, log.RequestId, nil
		}
	}
	return 0, 0, "", nil
}

func ReviewReferralCampaignEvent(eventId, reviewerId int, decision, remark string) (*ReferralCampaignEvent, error) {
	remark = strings.TrimSpace(remark)
	if eventId <= 0 || reviewerId <= 0 || (decision != "approve" && decision != "reject") || len(remark) > 2000 {
		return nil, errors.New("invalid referral review")
	}
	event := &ReferralCampaignEvent{}
	if err := DB.First(event, eventId).Error; err != nil {
		return nil, err
	}
	quota, qualifiedAt, requestId, err := referralConsumptionEvidence(*event)
	if err != nil {
		return nil, err
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		// Serialize cap reservation across all reviewers/instances. Always take
		// the campaign lock before the event lock in this decision path.
		var campaign ReferralCampaign
		if err := lockForUpdate(tx).First(&campaign, event.CampaignId).Error; err != nil {
			return err
		}
		if err := lockForUpdate(tx).First(event, eventId).Error; err != nil {
			return err
		}
		if event.ReviewDecision == "approved" {
			if decision == "approve" {
				return nil
			}
			return errors.New("an approved referral cannot be rejected")
		}
		if event.ReviewDecision == "rejected" || event.Status == ReferralEventRewarded || event.Status == ReferralEventRejected || event.Status == ReferralEventExpired {
			return errors.New("referral event is no longer reviewable")
		}
		if event.QualifiedQuota > 0 {
			quota, qualifiedAt, requestId = event.QualifiedQuota, event.QualifiedAt, event.QualifiedRequestId
		}
		if decision == "approve" && (quota <= 0 || quota > common.MaxQuota || qualifiedAt < event.RegisteredAt || qualifiedAt > event.ActivationDeadline) {
			return errors.New("a successful non-public-pool call with actual quota consumption is required")
		}
		if decision == "approve" {
			// Legacy pending/failed grants remain reserved, even before review,
			// so deploying this gate cannot silently expand campaign budgets.
			reserved := tx.Model(&ReferralCampaignEvent{}).Where("campaign_id = ? AND id <> ? AND status IN ?", campaign.Id, event.Id,
				[]ReferralCampaignEventStatus{ReferralEventRewardPending, ReferralEventRewarded, ReferralEventRewardFailed})
			var total, own int64
			if err := reserved.Session(&gorm.Session{}).Count(&total).Error; err != nil {
				return err
			}
			if err := reserved.Where("inviter_user_id = ?", event.InviterUserId).Count(&own).Error; err != nil {
				return err
			}
			if (campaign.TotalRewardLimit > 0 && total >= int64(campaign.TotalRewardLimit)) || (campaign.MaxRewardsPerInviter > 0 && own >= int64(campaign.MaxRewardsPerInviter)) {
				return errors.New("referral campaign reward limit reached")
			}
		}
		now := common.GetTimestamp()
		updates := map[string]interface{}{"reviewed_by": reviewerId, "reviewed_at": now, "review_remark": remark, "updated_at": now}
		if decision == "approve" {
			updates["review_decision"], updates["status"] = "approved", ReferralEventRewardPending
			updates["qualified_quota"], updates["qualified_at"], updates["qualified_request_id"] = quota, qualifiedAt, requestId
		} else {
			updates["review_decision"], updates["status"] = "rejected", ReferralEventRejected
		}
		if err := tx.Model(event).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(event, eventId).Error
	})
	if err != nil || decision == "reject" {
		return event, err
	}
	return grantApprovedReferralRewards(event)
}

func grantApprovedReferralRewards(event *ReferralCampaignEvent) (*ReferralCampaignEvent, error) {
	if event.ReviewDecision != "approved" || event.ReviewedBy <= 0 || event.QualifiedQuota <= 0 {
		return nil, errors.New("referral reward requires manual approval")
	}
	var grantError error
	for _, side := range []struct {
		userId int
		raw    string
	}{
		{event.InviterUserId, event.RewardSnapshotJSON},
		{event.InviteeUserId, event.InviteeRewardSnapshotJSON},
	} {
		if strings.TrimSpace(side.raw) == "" {
			continue // Zero reward, including the invitee side of historical events.
		}
		snapshot, err := DecodeRewardSnapshot(side.raw)
		if err == nil {
			_, err = GrantReward(RewardSourceReferralCampaign, event.Id, side.userId, snapshot)
		}
		grantError = errors.Join(grantError, err)
	}
	status := ReferralEventRewarded
	if grantError != nil {
		status = ReferralEventRewardFailed
	}
	// A slower failed attempt must never overwrite a concurrent successful retry.
	if err := DB.Model(&ReferralCampaignEvent{}).Where("id = ? AND status <> ?", event.Id, ReferralEventRewarded).
		Updates(map[string]interface{}{"status": status, "updated_at": common.GetTimestamp()}).Error; err != nil {
		return event, err
	}
	if err := DB.First(event, event.Id).Error; err != nil {
		return event, err
	}
	event.Reward = hydrateReferralReward(event.RewardSnapshotJSON)
	event.InviteeReward = hydrateReferralReward(event.InviteeRewardSnapshotJSON)
	if event.Status == ReferralEventRewarded {
		return event, nil
	}
	return event, grantError
}

func GetReferralCampaignEventReview(eventId int) (*ReferralEventReview, error) {
	view := &ReferralEventReview{}
	if eventId <= 0 {
		return nil, errors.New("invalid referral event")
	}
	if err := DB.First(&view.Event, eventId).Error; err != nil {
		return nil, err
	}
	for _, side := range []struct {
		id     int
		target *ReferralReviewUser
	}{
		{view.Event.InviterUserId, &view.Inviter}, {view.Event.InviteeUserId, &view.Invitee},
	} {
		if err := DB.Model(&User{}).Select("id", "username", "created_at", "github_id", "github_created_at", "github_age_exempt").Where("id = ?", side.id).Scan(side.target).Error; err != nil {
			return nil, err
		}
	}
	view.Event.Reward = hydrateReferralReward(view.Event.RewardSnapshotJSON)
	view.Event.InviteeReward = hydrateReferralReward(view.Event.InviteeRewardSnapshotJSON)
	quota, qualifiedAt, requestId, err := referralConsumptionEvidence(view.Event)
	if err != nil {
		return nil, err
	}
	view.HasConsumption = quota > 0
	view.Event.QualifiedQuota, view.Event.QualifiedAt, view.Event.QualifiedRequestId = quota, qualifiedAt, requestId
	view.CanApprove = view.HasConsumption && view.Event.ReviewDecision == "" && view.Event.Status != ReferralEventRewarded && view.Event.Status != ReferralEventExpired && view.Event.Status != ReferralEventRejected
	view.Usage = ReferralReviewUsage{PeriodStart: view.Event.RegisteredAt, PeriodEnd: common.GetTimestamp(), RecentCalls: []ReferralReviewCall{}}
	if LOG_DB != nil {
		query := LOG_DB.Model(&Log{}).Where("user_id = ? AND created_at >= ? AND created_at <= ? AND type IN ?", view.Event.InviteeUserId, view.Usage.PeriodStart, view.Usage.PeriodEnd, []int{LogTypeConsume, LogTypeError})
		var totals struct {
			Requests      int64
			ConsumedQuota int64
		}
		err := query.Session(&gorm.Session{}).Select("COUNT(*) AS requests, COALESCE(SUM(CASE WHEN type = ? AND quota > 0 THEN quota ELSE 0 END), 0) AS consumed_quota", LogTypeConsume).Scan(&totals).Error
		var logs []Log
		if err == nil {
			err = query.Order("created_at DESC").Order("id DESC").Limit(20).Find(&logs).Error
		}
		if err == nil {
			view.Usage.LogsAvailable = true
			view.Usage.Requests, view.Usage.ConsumedQuota = totals.Requests, totals.ConsumedQuota
			for _, log := range logs {
				other, _ := common.StrToMap(log.Other)
				success := log.Type == LogTypeConsume && isReferralActivationEligible(RecordConsumeLogParams{ActivateReferral: other["referral_activation_eligible"] == true, BillingSettled: other["billing_settlement_succeeded"] == true, Group: log.Group, Quota: log.Quota, IsStream: log.IsStream, Other: other})
				// Historical model_name may itself be an upstream alias; if a mapping
				// exists without a stored client model, display no model at all.
				requested, _ := other["request_model_name"].(string)
				if requested == "" && other["is_model_mapped"] != true && other["upstream_model_name"] == nil && other["admin_info"] == nil {
					requested = log.ModelName
				}
				view.Usage.RecentCalls = append(view.Usage.RecentCalls, ReferralReviewCall{Id: log.Id, CreatedAt: log.CreatedAt, RequestId: log.RequestId, ModelName: requested, Quota: log.Quota, PromptTokens: log.PromptTokens, CompletionTokens: log.CompletionTokens, Successful: success, PublicPool: log.Group == constant.PublicPoolGroup, Qualifying: success && log.Group != constant.PublicPoolGroup && log.Quota > 0 && log.CreatedAt <= view.Event.ActivationDeadline})
			}
		}
	}
	if err := DB.Model(&ReferralCampaignEvent{}).Select("referral_campaign_events.*, invitees.username AS invitee_username").
		Joins("LEFT JOIN users AS invitees ON invitees.id = referral_campaign_events.invitee_user_id").
		Where("inviter_user_id = ?", view.Event.InviterUserId).Order("registered_at DESC").Order("referral_campaign_events.id DESC").Limit(20).Scan(&view.RelatedEvents).Error; err != nil {
		return nil, err
	}
	if err := DB.Where("source_type = ? AND source_id = ?", RewardSourceReferralCampaign, view.Event.Id).Order("recipient_user_id ASC").Find(&view.RewardGrants).Error; err != nil {
		return nil, err
	}
	return view, nil
}
