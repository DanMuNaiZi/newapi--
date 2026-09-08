/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUsageRankingsResponseMasksUsersAndKeepsSelfRank(t *testing.T) {
	totals := []model.UsageRankingTotal{
		{UserID: 12, Username: "top-user", TotalQuota: 900, TotalTokens: 700, RequestCount: 9},
		{UserID: 11, Username: "middle", TotalQuota: 500, TotalTokens: 400, RequestCount: 5},
		{UserID: 10, Username: "self-user", TotalQuota: 100, TotalTokens: 80, RequestCount: 2},
	}

	response := buildUsageRankingsResponse("week", totals, 10, 2, false)

	require.Len(t, response.Users, 2)
	require.NotNil(t, response.TopUser)
	require.NotNil(t, response.MyRank)
	assert.Equal(t, 3, response.MyRank.Rank)
	assert.Equal(t, 3, response.TotalUsers)
	assert.Equal(t, int64(1500), response.TotalQuota)
	assert.Equal(t, int64(1180), response.TotalTokens)
	assert.Equal(t, int64(16), response.TotalRequests)
	assert.Equal(t, "t***r", response.TopUser.Username)
	assert.Equal(t, "s***r", response.MyRank.Username)
	assert.NotContains(t, response.TopUser.Username, "top-user")
}

func TestNormalizeUsageRankingLimit(t *testing.T) {
	assert.Equal(t, usageRankingDefaultLimit, normalizeUsageRankingLimit(0))
	assert.Equal(t, usageRankingMaxLimit, normalizeUsageRankingLimit(1000))
	assert.Equal(t, 20, normalizeUsageRankingLimit(20))
}

func TestMaskUsageRankingUsernameHandlesUnicode(t *testing.T) {
	assert.Equal(t, "杨*", maskUsageRankingUsername("杨彬"))
	assert.Equal(t, "*", maskUsageRankingUsername("Y"))
	assert.Equal(t, "***", maskUsageRankingUsername("  "))
	assert.Equal(t, "a*", maskUsageRankingUsername("ab"))
	assert.Equal(t, "a***z", maskUsageRankingUsername("abcdefgz"))
}

func TestBuildUsageRankingsResponseUsesStableTieOrderAndKeepsSelfRank(t *testing.T) {
	totals := []model.UsageRankingTotal{
		{UserID: 4, Username: "user-four", TotalQuota: 100, TotalTokens: 20, RequestCount: 2},
		{UserID: 2, Username: "user-two", TotalQuota: 100, TotalTokens: 20, RequestCount: 2},
		{UserID: 3, Username: "user-three", TotalQuota: 100, TotalTokens: 20, RequestCount: 2},
	}

	response := buildUsageRankingsResponse("month", totals, 4, 2, false)
	require.Len(t, response.Users, 2)
	assert.Equal(t, "u***o", response.Users[0].Username)
	assert.Equal(t, "u***e", response.Users[1].Username)
	assert.False(t, response.Users[0].IsSelf)
	assert.False(t, response.Users[1].IsSelf)
	require.NotNil(t, response.MyRank)
	assert.Equal(t, 3, response.MyRank.Rank)
	assert.Equal(t, "u***r", response.MyRank.Username)
	assert.True(t, response.MyRank.IsSelf)
	serialized, err := common.Marshal(response)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), "user-three")
}

func TestBuildUsageRankingsResponseHandlesEmptyUsageAndUnrankedUser(t *testing.T) {
	response := buildUsageRankingsResponse("today", nil, 99, 5, false)

	assert.Equal(t, "today", response.Period)
	assert.Equal(t, 5, response.DisplayCount)
	assert.Zero(t, response.TotalUsers)
	assert.Zero(t, response.TotalQuota)
	assert.Zero(t, response.TotalTokens)
	assert.Zero(t, response.TotalRequests)
	assert.Empty(t, response.Users)
	assert.Nil(t, response.TopUser)
	assert.Nil(t, response.MyRank)
}

func TestBuildUsageRankingsResponseShowsRawIdentityOnlyWhenAuthorized(t *testing.T) {
	totals := []model.UsageRankingTotal{
		{UserID: 12, Username: "top-user", TotalQuota: 900, TotalTokens: 700, RequestCount: 9},
		{UserID: 10, Username: "self-user", TotalQuota: 100, TotalTokens: 80, RequestCount: 2},
	}

	response := buildUsageRankingsResponse("week", totals, 10, 10, true)

	assert.True(t, response.IdentityVisible)
	require.NotNil(t, response.TopUser)
	require.NotNil(t, response.MyRank)
	assert.Equal(t, "top-user", response.TopUser.Username)
	assert.Equal(t, "self-user", response.MyRank.Username)
	require.Len(t, response.Users, 2)
	assert.Equal(t, "top-user", response.Users[0].Username)
	assert.Equal(t, "self-user", response.Users[1].Username)
}

func TestUsageRankingCacheDoesNotReuseIdentityVisibility(t *testing.T) {
	rows := []model.UsageRankingTotal{
		{UserID: 12, Username: "top-user", TotalQuota: 900, TotalTokens: 700, RequestCount: 9},
	}
	usageRankingCacheMu.Lock()
	previous, existed := usageRankingCache["week"]
	usageRankingCache["week"] = usageRankingCacheItem{
		expiresAt: time.Now().Add(time.Minute),
		rows:      rows,
	}
	usageRankingCacheMu.Unlock()
	t.Cleanup(func() {
		usageRankingCacheMu.Lock()
		defer usageRankingCacheMu.Unlock()
		if existed {
			usageRankingCache["week"] = previous
		} else {
			delete(usageRankingCache, "week")
		}
	})

	adminResponse, err := GetUsageRankingsSnapshot("week", 12, 10, true)
	require.NoError(t, err)
	userResponse, err := GetUsageRankingsSnapshot("week", 12, 10, false)
	require.NoError(t, err)

	require.NotNil(t, adminResponse.TopUser)
	require.NotNil(t, userResponse.TopUser)
	assert.Equal(t, "top-user", adminResponse.TopUser.Username)
	assert.Equal(t, "t***r", userResponse.TopUser.Username)
}

func TestUsageRankingPeriodRangesAreDeterministic(t *testing.T) {
	fixedNow := time.Date(2026, time.January, 15, 12, 34, 56, 0, time.UTC)
	tests := []struct {
		period string
		delta  time.Duration
	}{
		{period: "today", delta: 24 * time.Hour},
		{period: "week", delta: 7 * 24 * time.Hour},
		{period: "month", delta: 30 * 24 * time.Hour},
		{period: "year", delta: 365 * 24 * time.Hour},
	}
	for _, testCase := range tests {
		t.Run(testCase.period, func(t *testing.T) {
			config, err := rankingConfig(testCase.period)
			require.NoError(t, err)
			start, end := rankingTimeRange(config, fixedNow)
			assert.Equal(t, fixedNow.Unix(), end)
			assert.Equal(t, fixedNow.Add(-testCase.delta).Unix(), start)
		})
	}
}
