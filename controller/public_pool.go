package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
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
	site := &model.PublicPoolSite{
		Name:        request.Name,
		URL:         request.URL,
		Description: request.Description,
		Status:      request.Status,
		SortOrder:   request.SortOrder,
	}
	if err := model.CreatePublicPoolSite(site); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "public_pool.site_create", map[string]interface{}{"site_id": site.Id})
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
	site := &model.PublicPoolSite{
		Id:          id,
		Name:        request.Name,
		URL:         request.URL,
		Description: request.Description,
		Status:      request.Status,
		SortOrder:   request.SortOrder,
	}
	if err := model.UpdatePublicPoolSite(site); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "public_pool.site_update", map[string]interface{}{"site_id": site.Id})
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
	contributions, err := model.ListPublicPoolContributionsForAdmin()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, contributions)
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
	})
	common.ApiSuccess(c, contribution)
}

func publicPoolPathID(c *gin.Context) (int, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid public pool id")
	}
	return id, nil
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
