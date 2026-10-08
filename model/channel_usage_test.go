package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChannelUsageSummarySeparatesResourceAndChargedQuota(t *testing.T) {
	require.NotNil(t, LOG_DB)
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))
	channelID := 930001
	start := int64(1_700_000_000)
	cleanup := LOG_DB.Where("channel_id = ?", channelID).Delete(&Log{})
	require.NoError(t, cleanup.Error)
	resource := 70
	resourceRefund := 0
	rows := []*Log{
		{ChannelId: channelID, CreatedAt: start + 10, Type: LogTypeConsume, Quota: 100, ResourceQuota: &resource, PromptTokens: 10, CompletionTokens: 5, RequestId: "request-1"},
		{ChannelId: channelID, CreatedAt: start + 20, Type: LogTypeConsume, Quota: 40, ResourceQuota: &resource, PromptTokens: 4, CompletionTokens: 2, RequestId: "request-1"},
		{ChannelId: channelID, CreatedAt: start + 30, Type: LogTypeRefund, Quota: 30, ResourceQuota: &resourceRefund, RequestId: "request-1"},
		{ChannelId: channelID, CreatedAt: start + 40, Type: LogTypeError, RequestId: "request-2"},
	}
	for _, row := range rows {
		require.NoError(t, LOG_DB.Create(row).Error)
	}
	summary, err := GetChannelUsageSummary(channelID, "day", start, start+3600, 3600)
	require.NoError(t, err)
	assert.EqualValues(t, 140, summary.ResourceQuota)
	assert.EqualValues(t, 110, summary.ChargedQuota)
	assert.EqualValues(t, 1, summary.SuccessCount)
	assert.EqualValues(t, 1, summary.FailureCount)
	assert.EqualValues(t, 2, summary.RequestCount)
	assert.EqualValues(t, 1, summary.ResourceCoveredRequests)
	assert.True(t, summary.ResourceCoverageComplete)
	assert.Len(t, summary.Trend, 1)
}

func TestGetChannelUsageSummaryUsesHalfOpenTimeBuckets(t *testing.T) {
	require.NotNil(t, LOG_DB)
	channelID := 930002
	start := int64(1_700_100_000)
	require.NoError(t, LOG_DB.Where("channel_id = ?", channelID).Delete(&Log{}).Error)
	resource := 5
	for _, row := range []*Log{
		{ChannelId: channelID, CreatedAt: start, Type: LogTypeConsume, Quota: 10, ResourceQuota: &resource, RequestId: "left"},
		{ChannelId: channelID, CreatedAt: start + 60, Type: LogTypeConsume, Quota: 20, ResourceQuota: &resource, RequestId: "right"},
	} {
		require.NoError(t, LOG_DB.Create(row).Error)
	}
	summary, err := GetChannelUsageSummary(channelID, "day", start, start+120, 60)
	require.NoError(t, err)
	require.Len(t, summary.Trend, 2)
	assert.EqualValues(t, 10, summary.Trend[0].ChargedQuota)
	assert.EqualValues(t, 20, summary.Trend[1].ChargedQuota)
}
