package controller

import (
	"errors"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type channelUsageWindow struct {
	Period        string
	StartTime     int64
	EndTime       int64
	BucketSeconds int64
}

func resolveChannelUsagePeriod(period string, now time.Time) (channelUsageWindow, error) {
	window := channelUsageWindow{Period: period, EndTime: now.Unix()}
	switch period {
	case "", "day":
		window.Period = "day"
		window.StartTime = now.Add(-24 * time.Hour).Unix()
		window.BucketSeconds = int64(time.Hour / time.Second)
	case "week":
		window.StartTime = now.Add(-7 * 24 * time.Hour).Unix()
		window.BucketSeconds = int64(24 * time.Hour / time.Second)
	case "month":
		window.StartTime = now.Add(-30 * 24 * time.Hour).Unix()
		window.BucketSeconds = int64(24 * time.Hour / time.Second)
	default:
		return channelUsageWindow{}, errors.New("period must be day, week, or month")
	}
	return window, nil
}

func GetChannelUsageSummary(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelId <= 0 {
		common.ApiErrorMsg(c, "invalid channel id")
		return
	}
	if _, err = model.GetChannelById(channelId, false); err != nil {
		common.ApiError(c, err)
		return
	}

	window, err := resolveChannelUsagePeriod(c.Query("period"), time.Now())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	summary, err := model.GetChannelUsageSummary(
		channelId,
		window.Period,
		window.StartTime,
		window.EndTime,
		window.BucketSeconds,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}
