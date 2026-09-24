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
	Id          int                  `json:"id"`
	Name        string               `json:"name" gorm:"type:varchar(128);not null"`
	URL         string               `json:"url" gorm:"type:varchar(1024);not null"`
	Description string               `json:"description" gorm:"type:text"`
	Status      PublicPoolSiteStatus `json:"status" gorm:"type:varchar(32);index"`
	SortOrder   int                  `json:"sort_order" gorm:"type:int;index"`
	CreatedAt   int64                `json:"created_at" gorm:"type:bigint"`
	UpdatedAt   int64                `json:"updated_at" gorm:"type:bigint"`
}

type PublicPoolContribution struct {
	Id          int                          `json:"id"`
	UserId      int                          `json:"user_id" gorm:"index;not null"`
	SiteId      int                          `json:"site_id" gorm:"index;not null"`
	Description string                       `json:"description" gorm:"type:text"`
	Proof       string                       `json:"proof" gorm:"type:text"`
	Status      PublicPoolContributionStatus `json:"status" gorm:"type:varchar(32);index"`
	ReviewerId  int                          `json:"reviewer_id" gorm:"index"`
	ReviewNote  string                       `json:"review_note" gorm:"type:text"`
	ReviewedAt  int64                        `json:"reviewed_at" gorm:"type:bigint"`
	CreatedAt   int64                        `json:"created_at" gorm:"type:bigint;index"`
	UpdatedAt   int64                        `json:"updated_at" gorm:"type:bigint"`
	Username    string                       `json:"username,omitempty" gorm:"->;-:migration"`
	SiteName    string                       `json:"site_name,omitempty" gorm:"->;-:migration"`
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
	return DB.Create(site).Error
}

func UpdatePublicPoolSite(site *PublicPoolSite) error {
	if site == nil || site.Id <= 0 {
		return errors.New("invalid public pool site")
	}
	if err := normalizePublicPoolSite(site); err != nil {
		return err
	}
	updates := map[string]interface{}{
		"name":        site.Name,
		"url":         site.URL,
		"description": site.Description,
		"status":      site.Status,
		"sort_order":  site.SortOrder,
		"updated_at":  common.GetTimestamp(),
	}
	return DB.Model(&PublicPoolSite{}).Where("id = ?", site.Id).Updates(updates).Error
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
	return sites, err
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
	return DB.Transaction(func(tx *gorm.DB) error {
		var site PublicPoolSite
		if err := lockForUpdate(tx).
			Where("id = ? AND status = ?", contribution.SiteId, PublicPoolSiteStatusEnabled).
			First(&site).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		contribution.Status = PublicPoolContributionStatusPending
		contribution.ReviewerId = 0
		contribution.ReviewNote = ""
		contribution.ReviewedAt = 0
		contribution.CreatedAt = now
		contribution.UpdatedAt = now
		return tx.Create(contribution).Error
	})
}

func ListPublicPoolContributionsForUser(userId int) ([]PublicPoolContribution, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user")
	}
	var contributions []PublicPoolContribution
	err := DB.Model(&PublicPoolContribution{}).
		Select("public_pool_contributions.*, users.username AS username, public_pool_sites.name AS site_name").
		Joins("LEFT JOIN users ON users.id = public_pool_contributions.user_id").
		Joins("LEFT JOIN public_pool_sites ON public_pool_sites.id = public_pool_contributions.site_id").
		Where("public_pool_contributions.user_id = ?", userId).
		Order("public_pool_contributions.created_at DESC").
		Order("public_pool_contributions.id DESC").
		Scan(&contributions).Error
	return contributions, err
}

func ListPublicPoolContributionsForAdmin() ([]PublicPoolContribution, error) {
	var contributions []PublicPoolContribution
	err := DB.Model(&PublicPoolContribution{}).
		Select("public_pool_contributions.*, users.username AS username, public_pool_sites.name AS site_name").
		Joins("LEFT JOIN users ON users.id = public_pool_contributions.user_id").
		Joins("LEFT JOIN public_pool_sites ON public_pool_sites.id = public_pool_contributions.site_id").
		Order("public_pool_contributions.created_at DESC").
		Order("public_pool_contributions.id DESC").
		Scan(&contributions).Error
	return contributions, err
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
	return result, err
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
