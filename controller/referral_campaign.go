package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type referralCampaignRequest struct {
	Title                   string         `json:"title"`
	Description             string         `json:"description"`
	Enabled                 bool           `json:"enabled"`
	StartTime               int64          `json:"start_time"`
	EndTime                 int64          `json:"end_time"`
	ActivationWindowSeconds int64          `json:"activation_window_seconds"`
	MaxRewardsPerInviter    int            `json:"max_rewards_per_inviter"`
	TotalRewardLimit        int            `json:"total_reward_limit"`
	PreserveReward          bool           `json:"preserve_reward"`
	Reward                  dto.RewardSpec `json:"reward"`
}

type referralCampaignEventPage struct {
	Items    []model.ReferralCampaignEvent `json:"items"`
	Total    int64                         `json:"total"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"page_size"`
}

func GetReferralCampaignForSelf(c *gin.Context) {
	view, err := model.GetReferralCampaignForSelf(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, view)
}

func AdminListReferralCampaigns(c *gin.Context) {
	campaigns, err := model.ListReferralCampaignsForAdmin()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, campaigns)
}

func AdminCreateReferralCampaign(c *gin.Context) {
	request := referralCampaignRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	campaign, err := referralCampaignFromRequest(0, c.GetInt("id"), request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CreateReferralCampaign(campaign); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "referral_campaign.create", referralCampaignAudit(campaign))
	common.ApiSuccess(c, campaign)
}

func AdminUpdateReferralCampaign(c *gin.Context) {
	id, err := referralCampaignPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	request := referralCampaignRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	campaign, err := referralCampaignFromRequest(id, c.GetInt("id"), request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateReferralCampaign(campaign); err != nil {
		common.ApiError(c, err)
		return
	}
	audit := referralCampaignAudit(campaign)
	if request.PreserveReward {
		audit["preserve_reward"] = true
	}
	recordManageAudit(c, "referral_campaign.update", audit)
	common.ApiSuccess(c, campaign)
}

func AdminListReferralCampaignEvents(c *gin.Context) {
	campaignID, err := referralCampaignPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page, pageSize, err := referralCampaignPagination(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items, total, err := model.ListReferralCampaignEventsForAdmin(campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, referralCampaignEventPage{Items: items, Total: total, Page: page, PageSize: pageSize})
}

func AdminRetryReferralCampaignReward(c *gin.Context) {
	id, err := referralCampaignPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	event, err := model.RetryReferralCampaignReward(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "referral_campaign.reward_retry", map[string]interface{}{
		"event_id":      event.Id,
		"campaign_id":   event.CampaignId,
		"reward_status": event.RewardStatus,
	})
	common.ApiSuccess(c, event)
}

func referralCampaignFromRequest(id int, createdBy int, request referralCampaignRequest) (*model.ReferralCampaign, error) {
	campaign := &model.ReferralCampaign{
		Id:                      id,
		Title:                   request.Title,
		Description:             request.Description,
		Enabled:                 request.Enabled,
		StartTime:               request.StartTime,
		EndTime:                 request.EndTime,
		ActivationWindowSeconds: request.ActivationWindowSeconds,
		MaxRewardsPerInviter:    request.MaxRewardsPerInviter,
		TotalRewardLimit:        request.TotalRewardLimit,
		CreatedBy:               createdBy,
	}
	if id > 0 && request.PreserveReward {
		campaign.PreserveReward = true
		return campaign, nil
	}
	snapshot, err := service.NormalizeRewardSpec(request.Reward)
	if err != nil {
		return nil, err
	}
	rawReward, err := model.EncodeRewardSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	snapshot.SubscriptionPlanSnapshot = ""
	campaign.RewardSnapshotJSON = rawReward
	campaign.Reward = &snapshot
	return campaign, nil
}

func referralCampaignAudit(campaign *model.ReferralCampaign) map[string]interface{} {
	audit := map[string]interface{}{
		"campaign_id":               campaign.Id,
		"enabled":                   campaign.Enabled,
		"start_time":                campaign.StartTime,
		"end_time":                  campaign.EndTime,
		"activation_window_seconds": campaign.ActivationWindowSeconds,
		"max_rewards_per_inviter":   campaign.MaxRewardsPerInviter,
		"total_reward_limit":        campaign.TotalRewardLimit,
	}
	if campaign.Reward != nil {
		audit["reward_type"] = campaign.Reward.Type
		audit["reward_amount"] = campaign.Reward.InputAmount
		audit["reward_unit"] = campaign.Reward.InputUnit
		audit["reward_quota"] = campaign.Reward.Quota
		audit["quota_per_unit"] = campaign.Reward.QuotaPerUnit
		audit["usd_exchange_rate"] = campaign.Reward.USDExchangeRate
		audit["subscription_plan_id"] = campaign.Reward.SubscriptionPlanId
	}
	return audit
}

func referralCampaignPathID(c *gin.Context) (int, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid referral campaign id")
	}
	return id, nil
}

func referralCampaignPagination(c *gin.Context) (int, int, error) {
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
