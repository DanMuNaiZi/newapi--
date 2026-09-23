package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
)

func GetChannelErrors(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorMsg(c, "invalid channel id")
		return
	}
	if _, err := model.GetChannelById(channelID, false); err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo := common.GetPageQuery(c)
	if rawPage := strings.TrimSpace(c.Query("page")); rawPage != "" {
		page, err := strconv.Atoi(rawPage)
		if err != nil || page < 1 {
			common.ApiErrorMsg(c, "invalid page")
			return
		}
		pageInfo.Page = page
	}
	statusCode := 0
	if rawStatusCode := strings.TrimSpace(c.Query("status_code")); rawStatusCode != "" {
		statusCode, err = strconv.Atoi(rawStatusCode)
		if err != nil || statusCode < http.StatusBadRequest || statusCode > 599 {
			common.ApiErrorMsg(c, "invalid status_code")
			return
		}
	}
	items, total, err := model.GetChannelErrorRecords(model.ChannelErrorRecordQuery{
		ChannelID:    channelID,
		StatusCode:   statusCode,
		RequestModel: c.Query("request_model"),
		Keyword:      c.Query("keyword"),
		Offset:       pageInfo.GetStartIdx(),
		Limit:        pageInfo.GetPageSize(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func UpdateChannelErrorDisplay(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorMsg(c, "invalid channel id")
		return
	}
	settings := dto.UpstreamErrorDisplaySettings{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		common.ApiError(c, err)
		return
	}
	settings.Message = strings.TrimSpace(settings.Message)
	if err := settings.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.GetChannelById(channelID, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	otherSettings := channel.GetOtherSettings()
	otherSettings.UpstreamErrorDisplay = &settings
	channel.SetOtherSettings(otherSettings)
	if err := model.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("settings", channel.OtherSettings).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "channel.error_display.update", map[string]interface{}{
		"channel_id":   channelID,
		"show_details": settings.ShowDetails,
		"status_code":  settings.StatusCode,
	})
	common.ApiSuccess(c, settings)
}

func UpdateChannelErrorRecordDisplay(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorMsg(c, "invalid channel id")
		return
	}
	recordID, err := strconv.ParseUint(c.Param("error_id"), 10, 64)
	if err != nil || recordID == 0 {
		common.ApiErrorMsg(c, "invalid error id")
		return
	}
	rule := dto.ChannelErrorDisplayRule{}
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	rule.Message = strings.TrimSpace(rule.Message)
	if err := rule.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	record, err := model.UpdateChannelErrorDisplayRule(channelID, uint(recordID), rule)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "channel.error_display_rule.update", map[string]interface{}{
		"channel_id": channelID,
		"error_id":   recordID,
		"mode":       rule.Mode,
	})
	common.ApiSuccess(c, record)
}
