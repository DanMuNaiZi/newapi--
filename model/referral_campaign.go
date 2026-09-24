package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultReferralActivationWindowSeconds int64 = 24 * 60 * 60
	referralCampaignLimitMax                     = 2_147_483_647
)

type ReferralCampaign struct {
	Id                        int             `json:"id"`
	Title                     string          `json:"title" gorm:"type:varchar(128);not null"`
	Description               string          `json:"description" gorm:"type:text"`
	Enabled                   bool            `json:"enabled" gorm:"index"`
	StartTime                 int64           `json:"start_time" gorm:"type:bigint;index"`
	EndTime                   int64           `json:"end_time" gorm:"type:bigint;index"`
	ActivationWindowSeconds   int64           `json:"activation_window_seconds" gorm:"type:bigint"`
	MaxRewardsPerInviter      int             `json:"max_rewards_per_inviter"`
	TotalRewardLimit          int             `json:"total_reward_limit"`
	RewardSnapshotJSON        string          `json:"-" gorm:"column:reward_snapshot;type:text;not null"`
	InviteeRewardSnapshotJSON string          `json:"-" gorm:"column:invitee_reward_snapshot;type:text"`
	Reward                    *RewardSnapshot `json:"reward,omitempty" gorm:"-"`
	InviteeReward             *RewardSnapshot `json:"invitee_reward,omitempty" gorm:"-"`
	PreserveReward            bool            `json:"-" gorm:"-"`
	PreserveInviteeReward     bool            `json:"-" gorm:"-"`
	CreatedBy                 int             `json:"created_by" gorm:"index"`
	CreatedAt                 int64           `json:"created_at" gorm:"type:bigint"`
	UpdatedAt                 int64           `json:"updated_at" gorm:"type:bigint"`
	RegistrationCount         int64           `json:"registration_count" gorm:"-"`
	ActivatedCount            int64           `json:"activated_count" gorm:"-"`
	RewardedCount             int64           `json:"rewarded_count" gorm:"-"`
	FailedRewardCount         int64           `json:"failed_reward_count" gorm:"-"`
}

type ReferralCampaignEventStatus string

const (
	ReferralEventPending       ReferralCampaignEventStatus = "pending"
	ReferralEventPendingReview ReferralCampaignEventStatus = "pending_review"
	ReferralEventRejected      ReferralCampaignEventStatus = "rejected"
	ReferralEventRewardPending ReferralCampaignEventStatus = "reward_pending"
	ReferralEventRewarded      ReferralCampaignEventStatus = "rewarded"
	ReferralEventRewardFailed  ReferralCampaignEventStatus = "reward_failed"
	ReferralEventExpired       ReferralCampaignEventStatus = "expired"
	ReferralEventLimitReached  ReferralCampaignEventStatus = "limit_reached"
)

type ReferralCampaignEvent struct {
	Id                        int                         `json:"id"`
	CampaignId                int                         `json:"campaign_id" gorm:"index;uniqueIndex:idx_referral_campaign_invitee,priority:1"`
	InviterUserId             int                         `json:"inviter_user_id" gorm:"index"`
	InviteeUserId             int                         `json:"invitee_user_id" gorm:"index;uniqueIndex:idx_referral_campaign_invitee,priority:2"`
	Status                    ReferralCampaignEventStatus `json:"status" gorm:"type:varchar(32);index"`
	RegisteredAt              int64                       `json:"registered_at" gorm:"type:bigint;index"`
	ActivationDeadline        int64                       `json:"activation_deadline" gorm:"type:bigint;index"`
	ActivatedAt               int64                       `json:"activated_at" gorm:"type:bigint"`
	FirstSuccessRequestId     string                      `json:"first_success_request_id,omitempty" gorm:"type:varchar(64)"`
	RewardSnapshotJSON        string                      `json:"-" gorm:"column:reward_snapshot;type:text;not null"`
	InviteeRewardSnapshotJSON string                      `json:"-" gorm:"column:invitee_reward_snapshot;type:text"`
	InviteeReward             *RewardSnapshot             `json:"invitee_reward,omitempty" gorm:"-"`
	QualifiedQuota            int                         `json:"qualified_quota"`
	QualifiedAt               int64                       `json:"qualified_at" gorm:"type:bigint"`
	QualifiedRequestId        string                      `json:"qualified_request_id,omitempty" gorm:"type:varchar(64)"`
	ReviewDecision            string                      `json:"review_decision,omitempty" gorm:"type:varchar(16)"`
	ReviewedBy                int                         `json:"reviewed_by,omitempty"`
	ReviewedAt                int64                       `json:"reviewed_at,omitempty" gorm:"type:bigint"`
	ReviewRemark              string                      `json:"review_remark,omitempty" gorm:"type:text"`
	CreatedAt                 int64                       `json:"created_at" gorm:"type:bigint"`
	UpdatedAt                 int64                       `json:"updated_at" gorm:"type:bigint"`
	InviterUsername           string                      `json:"inviter_username,omitempty" gorm:"->;-:migration"`
	InviteeUsername           string                      `json:"invitee_username,omitempty" gorm:"->;-:migration"`
	Reward                    *RewardSnapshot             `json:"reward,omitempty" gorm:"-"`
	RewardGrantId             int                         `json:"reward_grant_id,omitempty" gorm:"->;-:migration"`
	RewardStatus              RewardGrantStatus           `json:"reward_status,omitempty" gorm:"->;-:migration"`
	RewardFailureReason       string                      `json:"reward_failure_reason,omitempty" gorm:"->;-:migration"`
}

type ReferralCampaignScheduleLock struct {
	Id        int   `json:"id"`
	UpdatedAt int64 `json:"updated_at" gorm:"type:bigint"`
}

type ReferralCampaignSelfView struct {
	Campaign *ReferralCampaign   `json:"campaign"`
	Events   []ReferralSelfEvent `json:"events"`
}

// An inviter may see referral progress, not the invitee's usage details or
// private review notes. Keep this contract separate from the administrator DTO.
type ReferralSelfEvent struct {
	Id                 int                         `json:"id"`
	CampaignId         int                         `json:"campaign_id"`
	InviteeUsername    string                      `json:"invitee_username"`
	Status             ReferralCampaignEventStatus `json:"status"`
	RegisteredAt       int64                       `json:"registered_at"`
	ActivationDeadline int64                       `json:"activation_deadline"`
	ActivatedAt        int64                       `json:"activated_at"`
	RewardStatus       RewardGrantStatus           `json:"reward_status,omitempty"`
	ReviewDecision     string                      `json:"review_decision,omitempty"`
	Reward             *RewardSnapshot             `json:"reward,omitempty"`
	InviteeReward      *RewardSnapshot             `json:"invitee_reward,omitempty"`
}

func normalizeReferralCampaign(campaign *ReferralCampaign, validateReward bool) error {
	if campaign == nil {
		return errors.New("referral campaign is required")
	}
	campaign.Title = strings.TrimSpace(campaign.Title)
	campaign.Description = strings.TrimSpace(campaign.Description)
	if campaign.Title == "" || len(campaign.Title) > 128 || len(campaign.Description) > 4000 {
		return errors.New("invalid referral campaign content")
	}
	if campaign.StartTime <= 0 || campaign.EndTime <= campaign.StartTime {
		return errors.New("invalid referral campaign time range")
	}
	if campaign.ActivationWindowSeconds == 0 {
		campaign.ActivationWindowSeconds = defaultReferralActivationWindowSeconds
	}
	if campaign.ActivationWindowSeconds < 60 || campaign.ActivationWindowSeconds > 365*24*60*60 {
		return errors.New("invalid referral activation window")
	}
	if campaign.MaxRewardsPerInviter < 0 || campaign.MaxRewardsPerInviter > referralCampaignLimitMax ||
		campaign.TotalRewardLimit < 0 || campaign.TotalRewardLimit > referralCampaignLimitMax {
		return errors.New("invalid referral campaign limits")
	}
	if validateReward {
		for _, raw := range []string{campaign.RewardSnapshotJSON, campaign.InviteeRewardSnapshotJSON} {
			if strings.TrimSpace(raw) != "" {
				if _, err := DecodeRewardSnapshot(raw); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func CreateReferralCampaign(campaign *ReferralCampaign) error {
	if err := normalizeReferralCampaign(campaign, true); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := lockReferralCampaignSchedule(tx); err != nil {
			return err
		}
		if campaign.Enabled {
			var count int64
			if err := tx.Model(&ReferralCampaign{}).Where("enabled = ? AND end_time > ?", true, common.GetTimestamp()).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("another referral campaign is already enabled")
			}
		}
		now := common.GetTimestamp()
		campaign.CreatedAt = now
		campaign.UpdatedAt = now
		return tx.Create(campaign).Error
	})
}

func UpdateReferralCampaign(campaign *ReferralCampaign) error {
	if campaign == nil || campaign.Id <= 0 {
		return errors.New("invalid referral campaign")
	}
	if err := normalizeReferralCampaign(campaign, !campaign.PreserveReward); err != nil {
		return err
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockReferralCampaignSchedule(tx); err != nil {
			return err
		}
		var current ReferralCampaign
		if err := lockForUpdate(tx).First(&current, campaign.Id).Error; err != nil {
			return err
		}
		if campaign.PreserveReward {
			campaign.RewardSnapshotJSON = current.RewardSnapshotJSON
		}
		if campaign.PreserveReward || campaign.PreserveInviteeReward {
			campaign.InviteeRewardSnapshotJSON = current.InviteeRewardSnapshotJSON
		}
		if campaign.Enabled {
			var count int64
			if err := tx.Model(&ReferralCampaign{}).Where("enabled = ? AND end_time > ? AND id <> ?", true, common.GetTimestamp(), campaign.Id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("another referral campaign is already enabled")
			}
		}
		return tx.Model(&current).Updates(map[string]interface{}{
			"title":                     campaign.Title,
			"description":               campaign.Description,
			"enabled":                   campaign.Enabled,
			"start_time":                campaign.StartTime,
			"end_time":                  campaign.EndTime,
			"activation_window_seconds": campaign.ActivationWindowSeconds,
			"max_rewards_per_inviter":   campaign.MaxRewardsPerInviter,
			"total_reward_limit":        campaign.TotalRewardLimit,
			"reward_snapshot":           campaign.RewardSnapshotJSON,
			"invitee_reward_snapshot":   campaign.InviteeRewardSnapshotJSON,
			"updated_at":                common.GetTimestamp(),
		}).Error
	})
	if err != nil {
		return err
	}
	campaign.Reward = hydrateReferralReward(campaign.RewardSnapshotJSON)
	campaign.InviteeReward = hydrateReferralReward(campaign.InviteeRewardSnapshotJSON)
	return nil
}

func lockReferralCampaignSchedule(tx *gorm.DB) error {
	lock := &ReferralCampaignScheduleLock{Id: 1, UpdatedAt: common.GetTimestamp()}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(lock).Error; err != nil {
		return err
	}
	return lockForUpdate(tx).First(lock, 1).Error
}

func hydrateReferralReward(raw string) *RewardSnapshot {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	snapshot, err := DecodeRewardSnapshot(raw)
	if err != nil {
		return nil
	}
	snapshot.SubscriptionPlanSnapshot = ""
	return &snapshot
}

func ListReferralCampaignsForAdmin() ([]ReferralCampaign, error) {
	var campaigns []ReferralCampaign
	if err := DB.Order("created_at DESC").Order("id DESC").Find(&campaigns).Error; err != nil {
		return nil, err
	}
	for index := range campaigns {
		campaigns[index].Reward = hydrateReferralReward(campaigns[index].RewardSnapshotJSON)
		campaigns[index].InviteeReward = hydrateReferralReward(campaigns[index].InviteeRewardSnapshotJSON)
		var counts []struct {
			Status ReferralCampaignEventStatus
			Count  int64
		}
		if err := DB.Model(&ReferralCampaignEvent{}).
			Select("status, COUNT(*) AS count").
			Where("campaign_id = ?", campaigns[index].Id).
			Group("status").Scan(&counts).Error; err != nil {
			return nil, err
		}
		for _, count := range counts {
			campaigns[index].RegistrationCount += count.Count
			switch count.Status {
			case ReferralEventPendingReview, ReferralEventRejected, ReferralEventRewardPending, ReferralEventRewarded, ReferralEventRewardFailed, ReferralEventLimitReached:
				campaigns[index].ActivatedCount += count.Count
			}
			if count.Status == ReferralEventRewarded {
				campaigns[index].RewardedCount += count.Count
			}
			if count.Status == ReferralEventRewardFailed {
				campaigns[index].FailedRewardCount += count.Count
			}
		}
	}
	return campaigns, nil
}

func GetReferralCampaignForSelf(userId int) (*ReferralCampaignSelfView, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user")
	}
	now := common.GetTimestamp()
	var campaign ReferralCampaign
	err := DB.Where("enabled = ? AND start_time <= ? AND end_time > ?", true, now, now).
		Order("start_time ASC").Order("id ASC").First(&campaign).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	view := &ReferralCampaignSelfView{}
	if err == nil {
		campaign.Reward = hydrateReferralReward(campaign.RewardSnapshotJSON)
		campaign.InviteeReward = hydrateReferralReward(campaign.InviteeRewardSnapshotJSON)
		view.Campaign = &campaign
	}
	events, err := ListReferralCampaignEventsForInviter(userId)
	if err != nil {
		return nil, err
	}
	view.Events = events
	return view, nil
}

func ListReferralCampaignEventsForInviter(inviterUserId int) ([]ReferralSelfEvent, error) {
	var events []ReferralCampaignEvent
	err := DB.Model(&ReferralCampaignEvent{}).
		Select("referral_campaign_events.id, referral_campaign_events.campaign_id, referral_campaign_events.status, referral_campaign_events.registered_at, referral_campaign_events.activation_deadline, referral_campaign_events.activated_at, referral_campaign_events.review_decision, referral_campaign_events.reward_snapshot, referral_campaign_events.invitee_reward_snapshot, invitees.username AS invitee_username, reward_grants.status AS reward_status").
		Joins("LEFT JOIN users AS invitees ON invitees.id = referral_campaign_events.invitee_user_id").
		Joins("LEFT JOIN reward_grants ON reward_grants.source_type = ? AND reward_grants.source_id = referral_campaign_events.id AND reward_grants.recipient_user_id = referral_campaign_events.inviter_user_id", RewardSourceReferralCampaign).
		Where("referral_campaign_events.inviter_user_id = ?", inviterUserId).
		Order("referral_campaign_events.registered_at DESC").Order("referral_campaign_events.id DESC").
		Limit(50).Scan(&events).Error
	if err != nil {
		return nil, err
	}
	items := make([]ReferralSelfEvent, 0, len(events))
	for _, event := range events {
		items = append(items, ReferralSelfEvent{
			Id: event.Id, CampaignId: event.CampaignId, InviteeUsername: common.MaskUsername(event.InviteeUsername),
			Status: event.Status, RegisteredAt: event.RegisteredAt, ActivationDeadline: event.ActivationDeadline, ActivatedAt: event.ActivatedAt,
			ReviewDecision: event.ReviewDecision, RewardStatus: event.RewardStatus, Reward: hydrateReferralReward(event.RewardSnapshotJSON), InviteeReward: hydrateReferralReward(event.InviteeRewardSnapshotJSON),
		})
	}
	return items, nil
}

func HandleInvitationRegistration(inviterUserId int, inviteeUserId int) (bool, error) {
	if inviterUserId <= 0 || inviteeUserId <= 0 || inviterUserId == inviteeUserId {
		return false, errors.New("invalid invitation registration")
	}
	usedCampaign := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var inviter User
		if err := lockForUpdate(tx).First(&inviter, inviterUserId).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		var campaign ReferralCampaign
		campaignErr := tx.Where("enabled = ? AND start_time <= ? AND end_time > ?", true, now, now).
			Order("start_time DESC").Order("id DESC").First(&campaign).Error
		if campaignErr != nil && !errors.Is(campaignErr, gorm.ErrRecordNotFound) {
			return campaignErr
		}
		if errors.Is(campaignErr, gorm.ErrRecordNotFound) {
			if common.QuotaForInviter <= 0 {
				return nil
			}
			updates := map[string]interface{}{
				"aff_count":   gorm.Expr("aff_count + 1"),
				"aff_quota":   gorm.Expr("aff_quota + ?", common.QuotaForInviter),
				"aff_history": gorm.Expr("aff_history + ?", common.QuotaForInviter),
			}
			return tx.Model(&User{}).Where("id = ?", inviterUserId).Updates(updates).Error
		}
		usedCampaign = true
		updates := map[string]interface{}{"aff_count": gorm.Expr("aff_count + 1")}
		event := &ReferralCampaignEvent{
			CampaignId:                campaign.Id,
			InviterUserId:             inviterUserId,
			InviteeUserId:             inviteeUserId,
			Status:                    ReferralEventPending,
			RegisteredAt:              now,
			ActivationDeadline:        now + campaign.ActivationWindowSeconds,
			RewardSnapshotJSON:        campaign.RewardSnapshotJSON,
			InviteeRewardSnapshotJSON: campaign.InviteeRewardSnapshotJSON,
			CreatedAt:                 now,
			UpdatedAt:                 now,
		}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(event)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		return tx.Model(&User{}).Where("id = ?", inviterUserId).Updates(updates).Error
	})
	return usedCampaign, err
}

func ActivateReferralCampaignForUser(inviteeUserId int, requestId string, group string, usage RecordConsumeLogParams) error {
	if inviteeUserId <= 0 || group == constant.PublicPoolGroup || usage.Quota < 0 || usage.Quota > common.MaxQuota || !isReferralActivationEligible(usage) {
		return nil
	}
	var event ReferralCampaignEvent
	statuses := []ReferralCampaignEventStatus{ReferralEventPending, ReferralEventPendingReview, ReferralEventRewardPending, ReferralEventRewardFailed}
	err := DB.Where("invitee_user_id = ? AND status IN ? AND (review_decision IS NULL OR review_decision = ?)", inviteeUserId, statuses, "").
		Order("registered_at ASC").Order("id ASC").First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// Read the separate optional log DB before taking a main-DB lock. Recover
	// verified earlier consumption before a later request can expire/migrate it.
	var savedQuota int
	var savedAt int64
	var savedRequestId string
	if event.QualifiedQuota == 0 && (event.Status == ReferralEventRewardPending || event.Status == ReferralEventRewardFailed || common.GetTimestamp() > event.ActivationDeadline) {
		savedQuota, savedAt, savedRequestId, err = referralConsumptionEvidence(event)
		if err != nil {
			return err
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ? AND status IN ? AND (review_decision IS NULL OR review_decision = ?)", event.Id, statuses, "").First(&event).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		now := common.GetTimestamp()
		updates := map[string]interface{}{"updated_at": now}
		if event.QualifiedQuota == 0 && savedQuota > 0 {
			updates["qualified_quota"], updates["qualified_at"], updates["qualified_request_id"] = savedQuota, savedAt, savedRequestId
		}
		if now > event.ActivationDeadline {
			if event.QualifiedQuota == 0 && savedQuota == 0 {
				updates["status"] = ReferralEventExpired
			}
			return tx.Model(&event).Updates(updates).Error
		}
		// Preserve legacy reserved slots; approval, not a subsequent call,
		// decides whether a pending/failed grant consumes the campaign budget.
		if event.Status != ReferralEventRewardPending && event.Status != ReferralEventRewardFailed {
			updates["status"] = ReferralEventPendingReview
		}
		if event.ActivatedAt == 0 {
			updates["activated_at"], updates["first_success_request_id"] = now, requestId
		}
		if event.QualifiedQuota == 0 && savedQuota == 0 && usage.Quota > 0 {
			updates["qualified_quota"], updates["qualified_at"], updates["qualified_request_id"] = usage.Quota, now, requestId
		}
		return tx.Model(&event).Updates(updates).Error
	})
}
func ListReferralCampaignEventsForAdmin(campaignId int, limit int, offset int) ([]ReferralCampaignEvent, int64, error) {
	if campaignId <= 0 {
		return nil, 0, errors.New("invalid referral campaign")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	query := DB.Model(&ReferralCampaignEvent{}).Where("referral_campaign_events.campaign_id = ?", campaignId)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var events []ReferralCampaignEvent
	err := query.
		Select("referral_campaign_events.*, inviters.username AS inviter_username, invitees.username AS invitee_username, reward_grants.id AS reward_grant_id, reward_grants.status AS reward_status, reward_grants.failure_reason AS reward_failure_reason").
		Joins("LEFT JOIN users AS inviters ON inviters.id = referral_campaign_events.inviter_user_id").
		Joins("LEFT JOIN users AS invitees ON invitees.id = referral_campaign_events.invitee_user_id").
		Joins("LEFT JOIN reward_grants ON reward_grants.source_type = ? AND reward_grants.source_id = referral_campaign_events.id AND reward_grants.recipient_user_id = referral_campaign_events.inviter_user_id", RewardSourceReferralCampaign).
		Order("referral_campaign_events.registered_at DESC").Order("referral_campaign_events.id DESC").
		Limit(limit).Offset(offset).Scan(&events).Error
	for index := range events {
		events[index].Reward = hydrateReferralReward(events[index].RewardSnapshotJSON)
		events[index].InviteeReward = hydrateReferralReward(events[index].InviteeRewardSnapshotJSON)
	}
	return events, total, err
}

func RetryReferralCampaignReward(eventId int) (*ReferralCampaignEvent, error) {
	if eventId <= 0 {
		return nil, errors.New("invalid referral event")
	}
	event := &ReferralCampaignEvent{}
	if err := DB.First(event, eventId).Error; err != nil {
		return nil, err
	}
	if event.Status != ReferralEventRewardFailed && event.Status != ReferralEventRewardPending {
		return nil, errors.New("referral reward is not retryable")
	}
	if event.ReviewDecision != "approved" || event.ReviewedBy <= 0 || event.QualifiedQuota <= 0 {
		return nil, errors.New("referral reward requires manual approval")
	}
	return grantApprovedReferralRewards(event)
}
