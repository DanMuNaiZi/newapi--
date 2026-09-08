package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func channelUsageQuota(value int) *int {
	return &value
}

func TestGetChannelUsageSummarySeparatesResourceAndUserQuota(t *testing.T) {
	truncateTables(t)
	require.NoError(t, LOG_DB.Create([]Log{
		{ChannelId: 7, CreatedAt: 50, Type: LogTypeConsume, RequestId: "outside", Quota: 10, ResourceQuota: channelUsageQuota(20)},
		{ChannelId: 7, CreatedAt: 110, Type: LogTypeConsume, RequestId: "public", Quota: 0, ResourceQuota: channelUsageQuota(500), PromptTokens: 10, CompletionTokens: 5},
		{ChannelId: 7, CreatedAt: 130, Type: LogTypeConsume, RequestId: "task", Quota: 100, ResourceQuota: channelUsageQuota(600), PromptTokens: 2, CompletionTokens: 3},
		{ChannelId: 7, CreatedAt: 140, Type: LogTypeConsume, RequestId: "task", Quota: 20},
		{ChannelId: 7, CreatedAt: 145, Type: LogTypeRefund, RequestId: "task", Quota: 20},
		{ChannelId: 7, CreatedAt: 150, Type: LogTypeError, RequestId: "error"},
		{ChannelId: 7, CreatedAt: 160, Type: LogTypeTopup, Quota: 999, ResourceQuota: channelUsageQuota(999)},
		{ChannelId: 8, CreatedAt: 170, Type: LogTypeConsume, Quota: 800, ResourceQuota: channelUsageQuota(900)},
	}).Error)

	summary, err := GetChannelUsageSummary(7, "day", 100, 200, 50)
	require.NoError(t, err)
	assert.Equal(t, int64(1100), summary.ResourceQuota)
	assert.Equal(t, int64(100), summary.ChargedQuota)
	assert.Equal(t, int64(12), summary.PromptTokens)
	assert.Equal(t, int64(8), summary.CompletionTokens)
	assert.Equal(t, int64(2), summary.SuccessCount)
	assert.Equal(t, int64(1), summary.FailureCount)
	assert.Equal(t, int64(3), summary.RequestCount)
	assert.Equal(t, int64(2), summary.ResourceCoveredRequests)
	assert.False(t, summary.ResourceCoverageComplete)
	assert.Equal(t, int64(50), summary.EarliestAvailableTime)
	require.Len(t, summary.Trend, 2)
	assert.Equal(t, int64(1100), summary.Trend[0].ResourceQuota)
	assert.Equal(t, int64(2), summary.Trend[0].RequestCount)
	assert.Equal(t, int64(1), summary.Trend[1].FailureCount)
}
