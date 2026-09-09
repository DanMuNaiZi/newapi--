package model

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

type PublicPoolSiteStatus string

const (
	PublicPoolSiteStatusEnabled  PublicPoolSiteStatus = "enabled"
	PublicPoolSiteStatusDisabled PublicPoolSiteStatus = "disabled"
	publicPoolSortOrderMax                            = 2_147_483_647
	publicPoolSortOrderMin                            = -2_147_483_648
)

var publicPoolCredentialPattern = regexp.MustCompile(`(?i)(?:\bauthorization\s*[:=]\s*bearer\s+\S+|\bbearer\s+[a-z0-9._~+/=-]{8,}|["']?(?:password|passwd|pwd|api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|cookie|session(?:id|_id)?)["']?\s*[:=]\s*["']?\S+|\bsk-[a-z0-9_-]{12,}|(?:^|[\s"'=:])[a-z0-9+/_-]{24,}={0,2}(?:$|[\s"',;]))`)

type PublicPoolContributionStatus string

const (
	PublicPoolContributionStatusPending  PublicPoolContributionStatus = "pending"
	PublicPoolContributionStatusApproved PublicPoolContributionStatus = "approved"
	PublicPoolContributionStatusRejected PublicPoolContributionStatus = "rejected"
)

type PublicPoolSite struct {
	Id                 int                  `json:"id"`
	Name               string               `json:"name" gorm:"type:varchar(128);not null"`
	URL                string               `json:"url" gorm:"type:varchar(1024);not null"`
	Description        string               `json:"description" gorm:"type:text"`
	Status             PublicPoolSiteStatus `json:"status" gorm:"type:varchar(32);index"`
	SortOrder          int                  `json:"sort_order" gorm:"type:int;index"`
	RewardSnapshotJSON string               `json:"-" gorm:"column:reward_snapshot;type:text"`
	Reward             *RewardSnapshot      `json:"reward,omitempty" gorm:"-"`
	CreatedAt          int64                `json:"created_at" gorm:"type:bigint"`
	UpdatedAt          int64                `json:"updated_at" gorm:"type:bigint"`
}

type PublicPoolContribution struct {
	Id                  int                          `json:"id"`
	UserId              int                          `json:"user_id" gorm:"index;index:idx_public_pool_user_site,priority:1;not null"`
	SiteId              int                          `json:"site_id" gorm:"index;index:idx_public_pool_user_site,priority:2;not null"`
	Description         string                       `json:"description" gorm:"type:text"`
	Proof               string                       `json:"proof" gorm:"type:text"`
	Status              PublicPoolContributionStatus `json:"status" gorm:"type:varchar(32);index"`
	ReviewerId          int                          `json:"reviewer_id" gorm:"index"`
	ReviewNote          string                       `json:"review_note" gorm:"type:text"`
	ReviewedAt          int64                        `json:"reviewed_at" gorm:"type:bigint"`
	RewardSnapshotJSON  string                       `json:"-" gorm:"column:reward_snapshot;type:text"`
	CreatedAt           int64                        `json:"created_at" gorm:"type:bigint;index"`
	UpdatedAt           int64                        `json:"updated_at" gorm:"type:bigint"`
	Username            string                       `json:"username,omitempty" gorm:"->;-:migration"`
	SiteName            string                       `json:"site_name,omitempty" gorm:"->;-:migration"`
	Reward              *RewardSnapshot              `json:"reward,omitempty" gorm:"-"`
	RewardGrantId       int                          `json:"reward_grant_id,omitempty" gorm:"->;-:migration"`
	RewardStatus        RewardGrantStatus            `json:"reward_status,omitempty" gorm:"->;-:migration"`
	RewardFailureReason string                       `json:"reward_failure_reason,omitempty" gorm:"->;-:migration"`
}

func normalizePublicPoolSite(site *PublicPoolSite) error {
	if site == nil {
		return errors.New("public pool site is required")
	}
	site.Name = strings.TrimSpace(site.Name)
	site.URL = strings.TrimSpace(site.URL)
	site.Description = strings.TrimSpace(site.Description)
	if site.Name == "" || len(site.Name) > 128 {
		return errors.New("public pool site name is invalid")
	}
	if len(site.URL) > 1024 {
		return errors.New("public pool site URL is too long")
	}
	if len(site.Description) > 4000 {
		return errors.New("public pool site description is too long")
	}
	if site.SortOrder < publicPoolSortOrderMin || site.SortOrder > publicPoolSortOrderMax {
		return errors.New("public pool site sort order is outside the database range")
	}
	parsedURL, err := url.Parse(site.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Hostname() == "" || parsedURL.User != nil {
		return errors.New("public pool site URL must be an http or https URL without credentials")
	}
	if site.Status == "" {
		site.Status = PublicPoolSiteStatusEnabled
	}
	if site.Status != PublicPoolSiteStatusEnabled && site.Status != PublicPoolSiteStatusDisabled {
		return errors.New("invalid public pool site status")
	}
	return nil
}

func CreatePublicPoolSite(site *PublicPoolSite) error {
	if err := normalizePublicPoolSite(site); err != nil {
		return err
	}
	now := common.GetTimestamp()
	site.CreatedAt = now
	site.UpdatedAt = now
	if err := DB.Create(site).Error; err != nil {
		return err
	}
	hydratePublicPoolSiteReward(site)
	return nil
}

func UpdatePublicPoolSite(site *PublicPoolSite) error {
	if site == nil || site.Id <= 0 {
		return errors.New("invalid public pool site")
	}
	if err := normalizePublicPoolSite(site); err != nil {
		return err
	}
	updates := map[string]interface{}{
		"name":            site.Name,
		"url":             site.URL,
		"description":     site.Description,
		"status":          site.Status,
		"sort_order":      site.SortOrder,
		"reward_snapshot": site.RewardSnapshotJSON,
		"updated_at":      common.GetTimestamp(),
	}
	if err := DB.Model(&PublicPoolSite{}).Where("id = ?", site.Id).Updates(updates).Error; err != nil {
		return err
	}
	hydratePublicPoolSiteReward(site)
	return nil
}

func DeletePublicPoolSite(id int) error {
	if id <= 0 {
		return errors.New("invalid public pool site")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var site PublicPoolSite
		if err := lockForUpdate(tx).First(&site, id).Error; err != nil {
			return err
		}
		var contributionCount int64
		if err := tx.Model(&PublicPoolContribution{}).Where("site_id = ?", id).Count(&contributionCount).Error; err != nil {
			return err
		}
		if contributionCount > 0 {
			return errors.New("public pool site with contributions can only be disabled")
		}
		return tx.Delete(&site).Error
	})
}

func ListPublicPoolSites(enabledOnly bool) ([]PublicPoolSite, error) {
	query := DB.Model(&PublicPoolSite{})
	if enabledOnly {
		query = query.Where("status = ?", PublicPoolSiteStatusEnabled)
	}
	var sites []PublicPoolSite
	err := query.Order("sort_order ASC").Order("id ASC").Find(&sites).Error
	for index := range sites {
		hydratePublicPoolSiteReward(&sites[index])
	}
	return sites, err
}

func hydratePublicPoolSiteReward(site *PublicPoolSite) {
	if site == nil || strings.TrimSpace(site.RewardSnapshotJSON) == "" {
		return
	}
	snapshot, err := DecodeRewardSnapshot(site.RewardSnapshotJSON)
	if err != nil {
		return
	}
	snapshot.SubscriptionPlanSnapshot = ""
	site.Reward = &snapshot
}

func normalizePublicPoolContribution(contribution *PublicPoolContribution) error {
	if contribution == nil || contribution.UserId <= 0 || contribution.SiteId <= 0 {
		return errors.New("invalid public pool contribution")
	}
	contribution.Description = strings.TrimSpace(contribution.Description)
	contribution.Proof = strings.TrimSpace(contribution.Proof)
	if contribution.Description == "" && contribution.Proof == "" {
		return errors.New("public pool contribution description or proof is required")
	}
	if len(contribution.Description) > 4000 || len(contribution.Proof) > 4000 {
		return errors.New("public pool contribution is too long")
	}
	if publicPoolCredentialPattern.MatchString(contribution.Description) || publicPoolCredentialPattern.MatchString(contribution.Proof) {
		return errors.New("public pool contribution must not contain credentials")
	}
	return nil
}

func CreatePublicPoolContribution(contribution *PublicPoolContribution) error {
	if err := normalizePublicPoolContribution(contribution); err != nil {
		return err
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var site PublicPoolSite
		if err := lockForUpdate(tx).
			Where("id = ? AND status = ?", contribution.SiteId, PublicPoolSiteStatusEnabled).
			First(&site).Error; err != nil {
			return err
		}
		var active PublicPoolContribution
		activeErr := lockForUpdate(tx).
			Where("user_id = ? AND site_id = ? AND status <> ?", contribution.UserId, contribution.SiteId, PublicPoolContributionStatusRejected).
			Order("id DESC").First(&active).Error
		if activeErr == nil {
			return errors.New("public pool contribution already exists")
		}
		if !errors.Is(activeErr, gorm.ErrRecordNotFound) {
			return activeErr
		}
		var existing PublicPoolContribution
		existingErr := lockForUpdate(tx).
			Where("user_id = ? AND site_id = ? AND status = ?", contribution.UserId, contribution.SiteId, PublicPoolContributionStatusRejected).
			Order("id DESC").First(&existing).Error
		if existingErr != nil && !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}
		now := common.GetTimestamp()
		if existingErr == nil {
			updates := map[string]interface{}{
				"description":     contribution.Description,
				"proof":           contribution.Proof,
				"status":          PublicPoolContributionStatusPending,
				"reviewer_id":     0,
				"review_note":     "",
				"reviewed_at":     0,
				"reward_snapshot": site.RewardSnapshotJSON,
				"updated_at":      now,
			}
			if err := tx.Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
			return tx.First(contribution, existing.Id).Error
		}
		contribution.Status = PublicPoolContributionStatusPending
		contribution.ReviewerId = 0
		contribution.ReviewNote = ""
		contribution.ReviewedAt = 0
		contribution.RewardSnapshotJSON = site.RewardSnapshotJSON
		contribution.CreatedAt = now
		contribution.UpdatedAt = now
		return tx.Create(contribution).Error
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(contribution.RewardSnapshotJSON) != "" {
		snapshot, decodeErr := DecodeRewardSnapshot(contribution.RewardSnapshotJSON)
		if decodeErr == nil {
			snapshot.SubscriptionPlanSnapshot = ""
			contribution.Reward = &snapshot
		}
	}
	return nil
}

func ListPublicPoolContributionsForUser(userId int) ([]PublicPoolContribution, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user")
	}
	var contributions []PublicPoolContribution
	err := DB.Model(&PublicPoolContribution{}).
		Select("public_pool_contributions.*, users.username AS username, public_pool_sites.name AS site_name, reward_grants.id AS reward_grant_id, reward_grants.status AS reward_status").
		Joins("LEFT JOIN users ON users.id = public_pool_contributions.user_id").
		Joins("LEFT JOIN public_pool_sites ON public_pool_sites.id = public_pool_contributions.site_id").
		Joins("LEFT JOIN reward_grants ON reward_grants.source_type = ? AND reward_grants.source_id = public_pool_contributions.id AND reward_grants.recipient_user_id = public_pool_contributions.user_id", RewardSourcePublicPool).
		Where("public_pool_contributions.user_id = ?", userId).
		Order("public_pool_contributions.created_at DESC").
		Order("public_pool_contributions.id DESC").
		Scan(&contributions).Error
	hydratePublicPoolContributionRewards(contributions)
	return contributions, err
}

func ListPublicPoolContributionsForAdmin() ([]PublicPoolContribution, error) {
	contributions, _, err := ListPublicPoolContributionsForAdminPage("", "", 100, 0)
	return contributions, err
}

func ListPublicPoolContributionsForAdminPage(search string, status PublicPoolContributionStatus, limit int, offset int) ([]PublicPoolContribution, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	query := DB.Model(&PublicPoolContribution{}).
		Joins("LEFT JOIN users ON users.id = public_pool_contributions.user_id").
		Joins("LEFT JOIN public_pool_sites ON public_pool_sites.id = public_pool_contributions.site_id")
	if keyword := strings.ToLower(strings.TrimSpace(search)); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("LOWER(users.username) LIKE ? OR LOWER(public_pool_sites.name) LIKE ?", pattern, pattern)
	}
	if status != "" {
		if status != PublicPoolContributionStatusPending && status != PublicPoolContributionStatusApproved && status != PublicPoolContributionStatusRejected {
			return nil, 0, errors.New("invalid public pool contribution status")
		}
		query = query.Where("public_pool_contributions.status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var contributions []PublicPoolContribution
	err := query.
		Select("public_pool_contributions.*, users.username AS username, public_pool_sites.name AS site_name, reward_grants.id AS reward_grant_id, reward_grants.status AS reward_status, reward_grants.failure_reason AS reward_failure_reason").
		Joins("LEFT JOIN reward_grants ON reward_grants.source_type = ? AND reward_grants.source_id = public_pool_contributions.id AND reward_grants.recipient_user_id = public_pool_contributions.user_id", RewardSourcePublicPool).
		Order("public_pool_contributions.created_at DESC").
		Order("public_pool_contributions.id DESC").
		Limit(limit).Offset(offset).
		Scan(&contributions).Error
	hydratePublicPoolContributionRewards(contributions)
	return contributions, total, err
}

func hydratePublicPoolContributionRewards(contributions []PublicPoolContribution) {
	for index := range contributions {
		if strings.TrimSpace(contributions[index].RewardSnapshotJSON) == "" {
			continue
		}
		snapshot, err := DecodeRewardSnapshot(contributions[index].RewardSnapshotJSON)
		if err != nil {
			continue
		}
		snapshot.SubscriptionPlanSnapshot = ""
		contributions[index].Reward = &snapshot
	}
}

func ReviewPublicPoolContribution(id int, reviewerId int, status PublicPoolContributionStatus, reviewNote string) (*PublicPoolContribution, error) {
	if id <= 0 || reviewerId <= 0 {
		return nil, errors.New("invalid public pool contribution review")
	}
	if status != PublicPoolContributionStatusApproved && status != PublicPoolContributionStatusRejected {
		return nil, errors.New("invalid public pool contribution review status")
	}
	reviewNote = strings.TrimSpace(reviewNote)
	if len(reviewNote) > 4000 {
		return nil, errors.New("public pool review note is too long")
	}
	result := &PublicPoolContribution{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(result, id).Error; err != nil {
			return err
		}
		if result.Status != PublicPoolContributionStatusPending {
			return errors.New("public pool contribution has already been reviewed")
		}
		if status == PublicPoolContributionStatusApproved && strings.TrimSpace(result.RewardSnapshotJSON) == "" {
			return errors.New("public pool contribution has no frozen reward")
		}
		now := common.GetTimestamp()
		updates := map[string]interface{}{
			"status":      status,
			"reviewer_id": reviewerId,
			"review_note": reviewNote,
			"reviewed_at": now,
			"updated_at":  now,
		}
		if err := tx.Model(result).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(result, id).Error
	})
	if err != nil || status != PublicPoolContributionStatusApproved {
		return result, err
	}
	snapshot, err := DecodeRewardSnapshot(result.RewardSnapshotJSON)
	if err != nil {
		return result, err
	}
	grant, grantErr := GrantReward(RewardSourcePublicPool, result.Id, result.UserId, snapshot)
	if grant != nil {
		result.RewardGrantId = grant.Id
		result.RewardStatus = grant.Status
		result.RewardFailureReason = grant.FailureReason
	}
	snapshot.SubscriptionPlanSnapshot = ""
	result.Reward = &snapshot
	if grantErr != nil {
		common.SysLog("failed to deliver public pool reward: " + grantErr.Error())
	}
	return result, nil
}

func RetryPublicPoolContributionReward(id int) (*PublicPoolContribution, error) {
	if id <= 0 {
		return nil, errors.New("invalid public pool contribution")
	}
	contribution := &PublicPoolContribution{}
	if err := DB.First(contribution, id).Error; err != nil {
		return nil, err
	}
	if contribution.Status != PublicPoolContributionStatusApproved || strings.TrimSpace(contribution.RewardSnapshotJSON) == "" {
		return nil, errors.New("public pool contribution reward is not retryable")
	}
	snapshot, err := DecodeRewardSnapshot(contribution.RewardSnapshotJSON)
	if err != nil {
		return nil, err
	}
	grant, grantErr := GrantReward(RewardSourcePublicPool, contribution.Id, contribution.UserId, snapshot)
	if grant != nil {
		contribution.RewardGrantId = grant.Id
		contribution.RewardStatus = grant.Status
		contribution.RewardFailureReason = grant.FailureReason
	}
	snapshot.SubscriptionPlanSnapshot = ""
	contribution.Reward = &snapshot
	return contribution, grantErr
}

func CountAvailablePublicPoolChannels() (int64, error) {
	var count int64
	err := DB.Table("abilities").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities."+commonGroupCol+" = ? AND abilities.enabled = ? AND channels.status = ?", constant.PublicPoolGroup, true, common.ChannelStatusEnabled).
		Distinct("abilities.channel_id").
		Count(&count).Error
	return count, err
}
