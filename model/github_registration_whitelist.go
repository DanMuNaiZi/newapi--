package model

import (
	"errors"

	"gorm.io/gorm"
)

type GitHubRegistrationWhitelist struct {
	Id              int    `json:"id"`
	GitHubId        string `json:"github_id" gorm:"column:github_id;size:32;uniqueIndex"`
	GitHubLogin     string `json:"github_login" gorm:"column:github_login;size:39;index"`
	GitHubCreatedAt int64  `json:"github_created_at" gorm:"column:github_created_at"`
	Remark          string `json:"remark" gorm:"type:varchar(255)"`
	CreatedBy       int    `json:"created_by" gorm:"column:created_by;index"`
	UpdatedBy       int    `json:"updated_by" gorm:"column:updated_by;index"`
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func ListGitHubRegistrationWhitelist() ([]GitHubRegistrationWhitelist, error) {
	var entries []GitHubRegistrationWhitelist
	err := DB.Order("id desc").Find(&entries).Error
	return entries, err
}

func GetGitHubRegistrationWhitelistByGitHubID(githubID string) (*GitHubRegistrationWhitelist, error) {
	var entry GitHubRegistrationWhitelist
	err := DB.Where("github_id = ?", githubID).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entry, err
}

func GetGitHubRegistrationWhitelistByID(id int) (*GitHubRegistrationWhitelist, error) {
	var entry GitHubRegistrationWhitelist
	err := DB.First(&entry, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entry, err
}

func CreateGitHubRegistrationWhitelist(entry *GitHubRegistrationWhitelist) error {
	return DB.Create(entry).Error
}

func UpdateGitHubRegistrationWhitelistRemark(id int, remark string, actorID int) (*GitHubRegistrationWhitelist, error) {
	entry, err := GetGitHubRegistrationWhitelistByID(id)
	if err != nil || entry == nil {
		return entry, err
	}
	entry.Remark = remark
	entry.UpdatedBy = actorID
	if err := DB.Model(entry).Updates(map[string]any{
		"remark":     remark,
		"updated_by": actorID,
	}).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

func DeleteGitHubRegistrationWhitelist(id int) (*GitHubRegistrationWhitelist, error) {
	entry, err := GetGitHubRegistrationWhitelistByID(id)
	if err != nil || entry == nil {
		return entry, err
	}
	if err := DB.Delete(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}
