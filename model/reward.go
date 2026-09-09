package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RewardType string

const (
	RewardTypeQuota        RewardType = "quota"
	RewardTypeSubscription RewardType = "subscription"
)

type RewardGrantStatus string

const (
	RewardGrantStatusPending   RewardGrantStatus = "pending"
	RewardGrantStatusSucceeded RewardGrantStatus = "succeeded"
	RewardGrantStatusFailed    RewardGrantStatus = "failed"
)

const (
	RewardSourcePublicPool       = "public_pool"
	RewardSourceReferralCampaign = "referral_campaign"
)

type RewardSnapshot struct {
	Type                     RewardType `json:"type"`
	InputAmount              string     `json:"amount,omitempty"`
	InputUnit                string     `json:"unit,omitempty"`
	Quota                    int        `json:"quota,omitempty"`
	QuotaPerUnit             string     `json:"quota_per_unit,omitempty"`
	USDExchangeRate          string     `json:"usd_exchange_rate,omitempty"`
	SubscriptionPlanId       int        `json:"subscription_plan_id,omitempty"`
	SubscriptionPlanTitle    string     `json:"subscription_plan_title,omitempty"`
	SubscriptionPlanSnapshot string     `json:"subscription_plan_snapshot,omitempty"`
}

type RewardGrant struct {
	Id                 int               `json:"id"`
	SourceType         string            `json:"source_type" gorm:"type:varchar(32);uniqueIndex:idx_reward_grant_source,priority:1"`
	SourceId           int               `json:"source_id" gorm:"uniqueIndex:idx_reward_grant_source,priority:2"`
	RecipientUserId    int               `json:"recipient_user_id" gorm:"index;uniqueIndex:idx_reward_grant_source,priority:3"`
	RewardSnapshot     string            `json:"-" gorm:"type:text;not null"`
	RewardType         RewardType        `json:"reward_type" gorm:"type:varchar(32);index"`
	Quota              int               `json:"quota"`
	SubscriptionPlanId int               `json:"subscription_plan_id" gorm:"index"`
	Status             RewardGrantStatus `json:"status" gorm:"type:varchar(32);index"`
	FailureReason      string            `json:"failure_reason" gorm:"type:text"`
	RetryCount         int               `json:"retry_count"`
	CompletedAt        int64             `json:"completed_at" gorm:"type:bigint"`
	CreatedAt          int64             `json:"created_at" gorm:"type:bigint;index"`
	UpdatedAt          int64             `json:"updated_at" gorm:"type:bigint"`
}

type RewardSubscriptionPlanOption struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

func ListRewardSubscriptionPlanOptions() ([]RewardSubscriptionPlanOption, error) {
	var plans []RewardSubscriptionPlanOption
	err := DB.Model(&SubscriptionPlan{}).
		Select("id", "title").
		Where("enabled = ?", true).
		Order("sort_order DESC").
		Order("id DESC").
		Find(&plans).Error
	return plans, err
}

func EncodeRewardSnapshot(snapshot RewardSnapshot) (string, error) {
	data, err := common.Marshal(snapshot)
	return string(data), err
}

func DecodeRewardSnapshot(raw string) (RewardSnapshot, error) {
	snapshot := RewardSnapshot{}
	if strings.TrimSpace(raw) == "" {
		return snapshot, errors.New("reward snapshot is empty")
	}
	if err := common.UnmarshalJsonStr(raw, &snapshot); err != nil {
		return RewardSnapshot{}, err
	}
	return snapshot, nil
}

func GrantReward(sourceType string, sourceId int, recipientUserId int, snapshot RewardSnapshot) (*RewardGrant, error) {
	if strings.TrimSpace(sourceType) == "" || sourceId <= 0 || recipientUserId <= 0 {
		return nil, errors.New("invalid reward source")
	}
	rawSnapshot, err := EncodeRewardSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	candidate := &RewardGrant{
		SourceType:         sourceType,
		SourceId:           sourceId,
		RecipientUserId:    recipientUserId,
		RewardSnapshot:     rawSnapshot,
		RewardType:         snapshot.Type,
		Quota:              snapshot.Quota,
		SubscriptionPlanId: snapshot.SubscriptionPlanId,
		Status:             RewardGrantStatusPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(candidate).Error; err != nil {
		return nil, err
	}
	grant := &RewardGrant{}
	if err := DB.Where("source_type = ? AND source_id = ? AND recipient_user_id = ?", sourceType, sourceId, recipientUserId).First(grant).Error; err != nil {
		return nil, err
	}
	if grant.Status == RewardGrantStatusSucceeded {
		return grant, nil
	}

	attemptErr := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(grant, grant.Id).Error; err != nil {
			return err
		}
		if grant.Status == RewardGrantStatusSucceeded {
			return nil
		}
		storedSnapshot, err := DecodeRewardSnapshot(grant.RewardSnapshot)
		if err != nil {
			return err
		}
		switch storedSnapshot.Type {
		case RewardTypeQuota:
			if storedSnapshot.Quota <= 0 || storedSnapshot.Quota > common.MaxQuota {
				return errors.New("invalid quota reward")
			}
			var recipient User
			if err := lockForUpdate(tx).Select("id", "quota").First(&recipient, recipientUserId).Error; err != nil {
				return err
			}
			if recipient.Quota > common.MaxQuota-storedSnapshot.Quota {
				return errors.New("quota reward would exceed the supported balance range")
			}
			result := tx.Model(&User{}).Where("id = ?", recipientUserId).
				Update("quota", gorm.Expr("quota + ?", storedSnapshot.Quota))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("reward recipient does not exist")
			}
		case RewardTypeSubscription:
			plan := &SubscriptionPlan{}
			if err := common.UnmarshalJsonStr(storedSnapshot.SubscriptionPlanSnapshot, plan); err != nil {
				return err
			}
			if _, err := CreateEarnedUserSubscriptionFromPlanTx(tx, recipientUserId, plan, sourceType); err != nil {
				return err
			}
		default:
			return errors.New("unsupported reward type")
		}
		completedAt := common.GetTimestamp()
		return tx.Model(grant).Updates(map[string]interface{}{
			"status":         RewardGrantStatusSucceeded,
			"failure_reason": "",
			"completed_at":   completedAt,
			"updated_at":     completedAt,
		}).Error
	})
	if attemptErr != nil {
		_ = DB.Model(&RewardGrant{}).
			Where("id = ? AND status <> ?", grant.Id, RewardGrantStatusSucceeded).
			Updates(map[string]interface{}{
				"status":         RewardGrantStatusFailed,
				"failure_reason": attemptErr.Error(),
				"retry_count":    gorm.Expr("retry_count + 1"),
				"updated_at":     common.GetTimestamp(),
			}).Error
		if reloadErr := DB.First(grant, grant.Id).Error; reloadErr != nil {
			return grant, reloadErr
		}
		// SQLite does not support SELECT ... FOR UPDATE. A concurrent worker may
		// therefore finish the same unique grant while this transaction reports a
		// lock error. Treat the persisted success as authoritative so callers do
		// not surface a false delivery failure or retry an already-completed grant.
		if grant.Status == RewardGrantStatusSucceeded {
			return grant, nil
		}
		return grant, attemptErr
	}
	_ = DB.First(grant, grant.Id).Error
	if grant.Status == RewardGrantStatusSucceeded {
		gopool.Go(func() {
			_ = invalidateUserCache(recipientUserId)
		})
	}
	return grant, nil
}

func RetryRewardGrant(id int) (*RewardGrant, error) {
	if id <= 0 {
		return nil, errors.New("invalid reward grant")
	}
	grant := &RewardGrant{}
	if err := DB.First(grant, id).Error; err != nil {
		return nil, err
	}
	snapshot, err := DecodeRewardSnapshot(grant.RewardSnapshot)
	if err != nil {
		return nil, err
	}
	return GrantReward(grant.SourceType, grant.SourceId, grant.RecipientUserId, snapshot)
}
