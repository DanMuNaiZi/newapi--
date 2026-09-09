package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

type publicPoolSiteRequest struct {
	Name        string                     `json:"name"`
	URL         string                     `json:"url"`
	Description string                     `json:"description"`
	Status      model.PublicPoolSiteStatus `json:"status"`
	SortOrder   int                        `json:"sort_order"`
	Reward      *dto.RewardSpec            `json:"reward"`
}

type publicPoolContributionRequest struct {
	SiteId      int    `json:"site_id"`
	Description string `json:"description"`
	Proof       string `json:"proof"`
}

type publicPoolContributionReviewRequest struct {
	Status     model.PublicPoolContributionStatus `json:"status"`
	ReviewNote string                             `json:"review_note"`
}

type publicPoolStatusResponse struct {
	service.PublicPoolStatus
	ChannelCount int64 `json:"channel_count"`
}

type publicPoolContributionPage struct {
	Items    []model.PublicPoolContribution `json:"items"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"page_size"`
}

func ListPublicPoolSitesForSelf(c *gin.Context) {
	sites, err := model.ListPublicPoolSites(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, sites)
}

func ListPublicPoolContributionsForSelf(c *gin.Context) {
	contributions, err := model.ListPublicPoolContributionsForUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, contributions)
}

func CreatePublicPoolContributionForSelf(c *gin.Context) {
	request := publicPoolContributionRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	contribution := &model.PublicPoolContribution{
		UserId:      c.GetInt("id"),
		SiteId:      request.SiteId,
		Description: request.Description,
		Proof:       request.Proof,
	}
	if err := model.CreatePublicPoolContribution(contribution); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, contribution)
}

func GetPublicPoolStatus(c *gin.Context) {
	status := service.GetPublicPoolStatus()
	channelCount, err := model.CountAvailablePublicPoolChannels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if channelCount == 0 && status.Available {
		status.Available = false
		status.Reason = "public pool has no available channels"
	}
	common.ApiSuccess(c, publicPoolStatusResponse{PublicPoolStatus: status, ChannelCount: channelCount})
}

func AdminListRewardSubscriptionPlans(c *gin.Context) {
	plans, err := model.ListRewardSubscriptionPlanOptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plans)
}

func AdminListPublicPoolSites(c *gin.Context) {
	sites, err := model.ListPublicPoolSites(false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, sites)
}

func AdminCreatePublicPoolSite(c *gin.Context) {
	request := publicPoolSiteRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	site, err := publicPoolSiteFromRequest(0, request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CreatePublicPoolSite(site); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "public_pool.site_create", publicPoolSiteAudit(site))
	common.ApiSuccess(c, site)
}

func AdminUpdatePublicPoolSite(c *gin.Context) {
	id, err := publicPoolPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	request := publicPoolSiteRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	site, err := publicPoolSiteFromRequest(id, request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdatePublicPoolSite(site); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "public_pool.site_update", publicPoolSiteAudit(site))
	common.ApiSuccess(c, site)
}

func AdminDeletePublicPoolSite(c *gin.Context) {
	id, err := publicPoolPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeletePublicPoolSite(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "public_pool.site_delete", map[string]interface{}{"site_id": id})
	common.ApiSuccess(c, nil)
}

func AdminListPublicPoolContributions(c *gin.Context) {
	page, pageSize, err := publicPoolAdminPagination(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	status := model.PublicPoolContributionStatus(c.Query("status"))
	contributions, total, err := model.ListPublicPoolContributionsForAdminPage(c.Query("search"), status, pageSize, (page-1)*pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, publicPoolContributionPage{Items: contributions, Total: total, Page: page, PageSize: pageSize})
}

func AdminReviewPublicPoolContribution(c *gin.Context) {
	id, err := publicPoolPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	request := publicPoolContributionReviewRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	contribution, err := model.ReviewPublicPoolContribution(id, c.GetInt("id"), request.Status, request.ReviewNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "public_pool.contribution_review", map[string]interface{}{
		"contribution_id": contribution.Id,
		"status":          contribution.Status,
		"reward_grant_id": contribution.RewardGrantId,
		"reward_status":   contribution.RewardStatus,
	})
	common.ApiSuccess(c, contribution)
}

func AdminRetryPublicPoolContributionReward(c *gin.Context) {
	id, err := publicPoolPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	contribution, err := model.RetryPublicPoolContributionReward(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "public_pool.reward_retry", map[string]interface{}{
		"contribution_id": contribution.Id,
		"reward_status":   contribution.RewardStatus,
	})
	common.ApiSuccess(c, contribution)
}

func publicPoolSiteFromRequest(id int, request publicPoolSiteRequest) (*model.PublicPoolSite, error) {
	site := &model.PublicPoolSite{
		Id:          id,
		Name:        request.Name,
		URL:         request.URL,
		Description: request.Description,
		Status:      request.Status,
		SortOrder:   request.SortOrder,
	}
	if request.Reward == nil {
		return site, nil
	}
	snapshot, err := service.NormalizeRewardSpec(*request.Reward)
	if err != nil {
		return nil, err
	}
	rawSnapshot, err := model.EncodeRewardSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	site.RewardSnapshotJSON = rawSnapshot
	snapshot.SubscriptionPlanSnapshot = ""
	site.Reward = &snapshot
	return site, nil
}

func publicPoolSiteAudit(site *model.PublicPoolSite) map[string]interface{} {
	audit := map[string]interface{}{"site_id": site.Id}
	if site.Reward != nil {
		audit["reward_type"] = site.Reward.Type
		audit["reward_amount"] = site.Reward.InputAmount
		audit["reward_unit"] = site.Reward.InputUnit
		audit["reward_quota"] = site.Reward.Quota
		audit["quota_per_unit"] = site.Reward.QuotaPerUnit
		audit["usd_exchange_rate"] = site.Reward.USDExchangeRate
		audit["subscription_plan_id"] = site.Reward.SubscriptionPlanId
	}
	return audit
}

func publicPoolPathID(c *gin.Context) (int, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid public pool id")
	}
	return id, nil
}

func publicPoolAdminPagination(c *gin.Context) (int, int, error) {
	page := 1
	pageSize := 20
	var err error
	if raw := c.Query("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page <= 0 {
			return 0, 0, errors.New("invalid page")
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil || pageSize <= 0 {
			return 0, 0, errors.New("invalid page size")
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, nil
}

func validatePublicPoolSettingUpdate(key string, value string) error {
	if key == "public_pool_setting.enabled" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return errors.New("invalid public pool enabled value")
		}
		if !enabled {
			return nil
		}
		ratio, exists := ratio_setting.GetGroupRatioSetting().GroupRatio.Get(service.PublicPoolGroup)
		if !exists || ratio != 0 {
			return errors.New("public_pool group ratio must be configured as 0 before enabling the public pool")
		}
		return validatePublicPoolSpecialRatios(ratio_setting.GroupGroupRatio2JSONString())
	}
	if !operation_setting.GetPublicPoolSetting().Enabled {
		return nil
	}
	if key == "GroupGroupRatio" {
		return validatePublicPoolSpecialRatios(value)
	}
	if key == "GroupRatio" {
		var ratios map[string]float64
		if err := common.UnmarshalJsonStr(value, &ratios); err != nil {
			return err
		}
		if ratio, exists := ratios[service.PublicPoolGroup]; !exists || ratio != 0 {
			return errors.New("public_pool group ratio must remain 0 while the public pool is enabled")
		}
	}
	return nil
}

func validatePublicPoolSpecialRatios(value string) error {
	var ratios map[string]map[string]float64
	if err := common.UnmarshalJsonStr(value, &ratios); err != nil {
		return err
	}
	for _, groupRatios := range ratios {
		if ratio, exists := groupRatios[service.PublicPoolGroup]; exists && ratio != 0 {
			return errors.New("public_pool special group ratio must remain 0")
		}
	}
	return nil
}
