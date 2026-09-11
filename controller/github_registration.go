package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-gonic/gin"
)

const maxGitHubWhitelistRemarkLength = 255

var resolveGitHubIdentity = oauth.ResolveGitHubUser

type githubRegistrationDecision struct {
	CreatedAt int64
	AgeExempt bool
}

type githubIdentityRequest struct {
	Username string `json:"username"`
}

type githubWhitelistCreateRequest struct {
	Username string `json:"username"`
	GitHubId string `json:"github_id"`
	Remark   string `json:"remark"`
}

type githubWhitelistRemarkRequest struct {
	Remark string `json:"remark"`
}

func evaluateGitHubRegistration(oauthUser *oauth.OAuthUser, now time.Time) (githubRegistrationDecision, error) {
	githubID, err := strconv.ParseInt(oauthUser.ProviderUserID, 10, 64)
	if err != nil || githubID <= 0 {
		return githubRegistrationDecision{}, oauth.NewOAuthErrorWithRaw(i18n.MsgOAuthGitHubIdentityUnverified, nil, "GitHub numeric identity is invalid")
	}

	entry, err := model.GetGitHubRegistrationWhitelistByGitHubID(oauthUser.ProviderUserID)
	if err != nil {
		return githubRegistrationDecision{}, oauth.NewOAuthErrorWithRaw(i18n.MsgOAuthGitHubIdentityUnverified, nil, "failed to verify GitHub whitelist identity")
	}
	if entry != nil {
		return githubRegistrationDecision{CreatedAt: oauthUser.CreatedAt, AgeExempt: true}, nil
	}

	if oauthUser.CreatedAt <= 0 {
		return githubRegistrationDecision{}, oauth.NewOAuthErrorWithRaw(i18n.MsgOAuthGitHubIdentityUnverified, nil, "GitHub created_at is missing or invalid")
	}

	minAgeDays := common.GitHubRegistrationMinAgeDays
	if minAgeDays <= 0 {
		return githubRegistrationDecision{CreatedAt: oauthUser.CreatedAt}, nil
	}
	cutoff := now.UTC().AddDate(0, 0, -minAgeDays)
	if time.Unix(oauthUser.CreatedAt, 0).UTC().After(cutoff) {
		return githubRegistrationDecision{}, oauth.NewOAuthErrorWithRaw(
			i18n.MsgOAuthGitHubAccountTooYoung,
			map[string]any{"Days": minAgeDays},
			fmt.Sprintf("GitHub account must be at least %d days old", minAgeDays),
		)
	}
	return githubRegistrationDecision{CreatedAt: oauthUser.CreatedAt}, nil
}

func ResolveGitHubRegistrationIdentity(c *gin.Context) {
	var request githubIdentityRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "Invalid request")
		return
	}
	identity, err := resolveGitHubIdentity(c.Request.Context(), request.Username)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "Unable to verify the GitHub account"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": identity})
}

func ListGitHubRegistrationWhitelist(c *gin.Context) {
	entries, err := model.ListGitHubRegistrationWhitelist()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": entries})
}

func CreateGitHubRegistrationWhitelist(c *gin.Context) {
	var request githubWhitelistCreateRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "Invalid request")
		return
	}
	request.Username = strings.TrimSpace(request.Username)
	request.GitHubId = strings.TrimSpace(request.GitHubId)
	request.Remark = strings.TrimSpace(request.Remark)
	if len(request.Remark) > maxGitHubWhitelistRemarkLength {
		common.ApiErrorMsg(c, "Remark is too long")
		return
	}

	identity, err := resolveGitHubIdentity(c.Request.Context(), request.Username)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "Unable to verify the GitHub account"})
		return
	}
	verifiedID := strconv.FormatInt(identity.ID, 10)
	if request.GitHubId == "" || request.GitHubId != verifiedID {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "The GitHub identity changed; resolve it again before saving"})
		return
	}
	existing, err := model.GetGitHubRegistrationWhitelistByGitHubID(verifiedID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "This GitHub account is already exempt"})
		return
	}

	actorID := c.GetInt("id")
	entry := &model.GitHubRegistrationWhitelist{
		GitHubId:        verifiedID,
		GitHubLogin:     identity.Login,
		GitHubCreatedAt: identity.CreatedAt,
		Remark:          request.Remark,
		CreatedBy:       actorID,
		UpdatedBy:       actorID,
	}
	if err := model.CreateGitHubRegistrationWhitelist(entry); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "github_registration.whitelist_create", map[string]interface{}{
		"github_id":    entry.GitHubId,
		"github_login": entry.GitHubLogin,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": entry})
}

func UpdateGitHubRegistrationWhitelist(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "Invalid whitelist entry ID")
		return
	}
	var request githubWhitelistRemarkRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "Invalid request")
		return
	}
	request.Remark = strings.TrimSpace(request.Remark)
	if len(request.Remark) > maxGitHubWhitelistRemarkLength {
		common.ApiErrorMsg(c, "Remark is too long")
		return
	}
	entry, err := model.UpdateGitHubRegistrationWhitelistRemark(id, request.Remark, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Whitelist entry not found"})
		return
	}
	recordManageAudit(c, "github_registration.whitelist_update", map[string]interface{}{"github_id": entry.GitHubId})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": entry})
}

func DeleteGitHubRegistrationWhitelist(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "Invalid whitelist entry ID")
		return
	}
	entry, err := model.DeleteGitHubRegistrationWhitelist(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Whitelist entry not found"})
		return
	}
	recordManageAudit(c, "github_registration.whitelist_delete", map[string]interface{}{
		"github_id":    entry.GitHubId,
		"github_login": entry.GitHubLogin,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
