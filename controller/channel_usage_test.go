package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveChannelUsagePeriod(t *testing.T) {
	now := time.Date(2026, time.September, 7, 12, 30, 0, 0, time.UTC)

	tests := []struct {
		period       string
		duration     time.Duration
		bucket       time.Duration
		expectedName string
	}{
		{period: "", duration: 24 * time.Hour, bucket: time.Hour, expectedName: "day"},
		{period: "day", duration: 24 * time.Hour, bucket: time.Hour, expectedName: "day"},
		{period: "week", duration: 7 * 24 * time.Hour, bucket: 24 * time.Hour, expectedName: "week"},
		{period: "month", duration: 30 * 24 * time.Hour, bucket: 24 * time.Hour, expectedName: "month"},
	}

	for _, test := range tests {
		t.Run(test.expectedName+test.period, func(t *testing.T) {
			window, err := resolveChannelUsagePeriod(test.period, now)
			require.NoError(t, err)
			assert.Equal(t, test.expectedName, window.Period)
			assert.Equal(t, now.Add(-test.duration).Unix(), window.StartTime)
			assert.Equal(t, now.Unix(), window.EndTime)
			assert.Equal(t, int64(test.bucket/time.Second), window.BucketSeconds)
		})
	}

	_, err := resolveChannelUsagePeriod("year", now)
	require.Error(t, err)
}
