package model

import (
	"errors"

	"gorm.io/gorm"
)

type ChannelUsageTrendPoint struct {
	StartTime               int64 `json:"start_time"`
	EndTime                 int64 `json:"end_time"`
	ResourceQuota           int64 `json:"resource_quota"`
	ChargedQuota            int64 `json:"charged_quota"`
	PromptTokens            int64 `json:"prompt_tokens"`
	CompletionTokens        int64 `json:"completion_tokens"`
	SuccessCount            int64 `json:"success_count"`
	FailureCount            int64 `json:"failure_count"`
	RequestCount            int64 `json:"request_count"`
	ResourceCoveredRequests int64 `json:"resource_covered_requests"`
}

type ChannelUsageSummary struct {
	Period                   string                   `json:"period"`
	StartTime                int64                    `json:"start_time"`
	EndTime                  int64                    `json:"end_time"`
	EarliestAvailableTime    int64                    `json:"earliest_available_time"`
	ResourceCoverageStart    int64                    `json:"resource_coverage_start_time"`
	ResourceCoverageComplete bool                     `json:"resource_coverage_complete"`
	ResourceQuota            int64                    `json:"resource_quota"`
	ChargedQuota             int64                    `json:"charged_quota"`
	PromptTokens             int64                    `json:"prompt_tokens"`
	CompletionTokens         int64                    `json:"completion_tokens"`
	SuccessCount             int64                    `json:"success_count"`
	FailureCount             int64                    `json:"failure_count"`
	RequestCount             int64                    `json:"request_count"`
	ResourceCoveredRequests  int64                    `json:"resource_covered_requests"`
	Trend                    []ChannelUsageTrendPoint `json:"trend"`
}

type channelUsageAggregate struct {
	ResourceQuota           int64
	ChargedQuota            int64
	PromptTokens            int64
	CompletionTokens        int64
	SuccessCount            int64
	FailureCount            int64
	RequestCount            int64
	ResourceCoveredRequests int64
	UncoveredResourceRows   int64
}

const channelUsageAggregateSelect = `
COALESCE(SUM(CASE WHEN type = 2 THEN COALESCE(resource_quota, 0) ELSE 0 END), 0) AS resource_quota,
COALESCE(SUM(CASE WHEN type = 2 THEN quota WHEN type = 6 THEN -quota ELSE 0 END), 0) AS charged_quota,
COALESCE(SUM(CASE WHEN type = 2 THEN prompt_tokens ELSE 0 END), 0) AS prompt_tokens,
COALESCE(SUM(CASE WHEN type = 2 THEN completion_tokens ELSE 0 END), 0) AS completion_tokens,
COUNT(DISTINCT CASE WHEN type = 2 AND COALESCE(request_id, '') <> '' THEN request_id END)
  + COALESCE(SUM(CASE WHEN type = 2 AND COALESCE(request_id, '') = '' THEN 1 ELSE 0 END), 0) AS success_count,
COUNT(DISTINCT CASE WHEN type = 5 AND COALESCE(request_id, '') <> '' THEN request_id END)
  + COALESCE(SUM(CASE WHEN type = 5 AND COALESCE(request_id, '') = '' THEN 1 ELSE 0 END), 0) AS failure_count,
COUNT(DISTINCT CASE WHEN type = 2 AND resource_quota IS NOT NULL AND COALESCE(request_id, '') <> '' THEN request_id END)
  + COALESCE(SUM(CASE WHEN type = 2 AND resource_quota IS NOT NULL AND COALESCE(request_id, '') = '' THEN 1 ELSE 0 END), 0) AS resource_covered_requests,
COALESCE(SUM(CASE WHEN type = 2 AND resource_quota IS NULL THEN 1 ELSE 0 END), 0) AS uncovered_resource_rows`

func scanChannelUsageAggregate(query *gorm.DB) (channelUsageAggregate, error) {
	aggregate := channelUsageAggregate{}
	err := query.Select(channelUsageAggregateSelect).Scan(&aggregate).Error
	aggregate.RequestCount = aggregate.SuccessCount + aggregate.FailureCount
	return aggregate, err
}

func GetChannelUsageSummary(channelId int, period string, startTime int64, endTime int64, bucketSeconds int64) (*ChannelUsageSummary, error) {
	if channelId <= 0 || startTime < 0 || endTime <= startTime || bucketSeconds <= 0 {
		return nil, errors.New("invalid channel usage summary range")
	}
	baseQuery := LOG_DB.Model(&Log{}).
		Where("channel_id = ? AND created_at >= ? AND created_at < ? AND type IN ?", channelId, startTime, endTime, []int{LogTypeConsume, LogTypeError, LogTypeRefund})
	aggregate, err := scanChannelUsageAggregate(baseQuery)
	if err != nil {
		return nil, err
	}

	summary := &ChannelUsageSummary{
		Period:                  period,
		StartTime:               startTime,
		EndTime:                 endTime,
		ResourceQuota:           aggregate.ResourceQuota,
		ChargedQuota:            aggregate.ChargedQuota,
		PromptTokens:            aggregate.PromptTokens,
		CompletionTokens:        aggregate.CompletionTokens,
		SuccessCount:            aggregate.SuccessCount,
		FailureCount:            aggregate.FailureCount,
		RequestCount:            aggregate.RequestCount,
		ResourceCoveredRequests: aggregate.ResourceCoveredRequests,
		Trend:                   make([]ChannelUsageTrendPoint, 0, (endTime-startTime+bucketSeconds-1)/bucketSeconds),
	}
	// A request can emit multiple consume rows (for example an async task's
	// initial charge and later adjustment). Treat the period as complete only
	// when every consume row carries resource quota, rather than hiding a partial
	// row behind another covered row with the same request ID.
	summary.ResourceCoverageComplete = aggregate.UncoveredResourceRows == 0
	if err := LOG_DB.Model(&Log{}).
		Where("channel_id = ? AND type IN ?", channelId, []int{LogTypeConsume, LogTypeError, LogTypeRefund}).
		Select("COALESCE(MIN(created_at), 0)").
		Scan(&summary.EarliestAvailableTime).Error; err != nil {
		return nil, err
	}
	if err := LOG_DB.Model(&Log{}).
		Where("channel_id = ? AND type = ? AND resource_quota IS NOT NULL", channelId, LogTypeConsume).
		Select("COALESCE(MIN(created_at), 0)").
		Scan(&summary.ResourceCoverageStart).Error; err != nil {
		return nil, err
	}

	for bucketStart := startTime; bucketStart < endTime; bucketStart += bucketSeconds {
		bucketEnd := bucketStart + bucketSeconds
		if bucketEnd > endTime {
			bucketEnd = endTime
		}
		bucketAggregate, err := scanChannelUsageAggregate(LOG_DB.Model(&Log{}).
			Where("channel_id = ? AND created_at >= ? AND created_at < ? AND type IN ?", channelId, bucketStart, bucketEnd, []int{LogTypeConsume, LogTypeError, LogTypeRefund}))
		if err != nil {
			return nil, err
		}
		summary.Trend = append(summary.Trend, ChannelUsageTrendPoint{
			StartTime:               bucketStart,
			EndTime:                 bucketEnd,
			ResourceQuota:           bucketAggregate.ResourceQuota,
			ChargedQuota:            bucketAggregate.ChargedQuota,
			PromptTokens:            bucketAggregate.PromptTokens,
			CompletionTokens:        bucketAggregate.CompletionTokens,
			SuccessCount:            bucketAggregate.SuccessCount,
			FailureCount:            bucketAggregate.FailureCount,
			RequestCount:            bucketAggregate.RequestCount,
			ResourceCoveredRequests: bucketAggregate.ResourceCoveredRequests,
		})
	}
	return summary, nil
}
