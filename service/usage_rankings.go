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
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/model"
)

const (
	usageRankingDefaultLimit = 10
	usageRankingMaxLimit     = 50
	usageRankingCacheTTL     = 5 * time.Minute
)

type UsageRankingRow struct {
	Rank         int    `json:"rank"`
	Username     string `json:"username"`
	TotalQuota   int64  `json:"total_quota"`
	TotalTokens  int64  `json:"total_tokens"`
	RequestCount int64  `json:"request_count"`
	IsSelf       bool   `json:"is_self"`
}

type UsageRankingsResponse struct {
	Period          string            `json:"period"`
	IdentityVisible bool              `json:"identity_visible"`
	DisplayCount    int               `json:"display_count"`
	TotalUsers      int               `json:"total_users"`
	TotalQuota      int64             `json:"total_quota"`
	TotalTokens     int64             `json:"total_tokens"`
	TotalRequests   int64             `json:"total_requests"`
	TopUser         *UsageRankingRow  `json:"top_user,omitempty"`
	MyRank          *UsageRankingRow  `json:"my_rank,omitempty"`
	Users           []UsageRankingRow `json:"users"`
}

type usageRankingCacheItem struct {
	expiresAt time.Time
	rows      []model.UsageRankingTotal
}

var (
	usageRankingCacheMu sync.Mutex
	usageRankingCache   = map[string]usageRankingCacheItem{}
)

func GetUsageRankingsSnapshot(period string, userID int, limit int, identityVisible bool) (*UsageRankingsResponse, error) {
	config, err := rankingConfig(period)
	if err != nil {
		return nil, err
	}
	period = config.id
	limit = normalizeUsageRankingLimit(limit)

	now := time.Now()
	usageRankingCacheMu.Lock()
	item, cached := usageRankingCache[period]
	if cached && now.Before(item.expiresAt) {
		rows := append([]model.UsageRankingTotal(nil), item.rows...)
		usageRankingCacheMu.Unlock()
		return buildUsageRankingsResponse(period, rows, userID, limit, identityVisible), nil
	}
	usageRankingCacheMu.Unlock()

	endTime := now.Unix()
	startTime := now.Add(-config.duration).Unix()
	rows, err := model.GetUsageRankingTotals(startTime, endTime)
	if err != nil {
		return nil, err
	}

	usageRankingCacheMu.Lock()
	usageRankingCache[period] = usageRankingCacheItem{
		expiresAt: now.Add(usageRankingCacheTTL),
		rows:      append([]model.UsageRankingTotal(nil), rows...),
	}
	usageRankingCacheMu.Unlock()

	return buildUsageRankingsResponse(period, rows, userID, limit, identityVisible), nil
}

func normalizeUsageRankingLimit(limit int) int {
	if limit <= 0 {
		return usageRankingDefaultLimit
	}
	if limit > usageRankingMaxLimit {
		return usageRankingMaxLimit
	}
	return limit
}

func buildUsageRankingsResponse(period string, totals []model.UsageRankingTotal, userID int, limit int, identityVisible bool) *UsageRankingsResponse {
	// The database query already supplies the stable ordering. Keep this sort as
	// a defensive guard for alternate stores and deterministic test fixtures.
	sort.SliceStable(totals, func(i, j int) bool {
		if totals[i].TotalQuota != totals[j].TotalQuota {
			return totals[i].TotalQuota > totals[j].TotalQuota
		}
		if totals[i].TotalTokens != totals[j].TotalTokens {
			return totals[i].TotalTokens > totals[j].TotalTokens
		}
		if totals[i].RequestCount != totals[j].RequestCount {
			return totals[i].RequestCount > totals[j].RequestCount
		}
		return totals[i].UserID < totals[j].UserID
	})

	response := &UsageRankingsResponse{
		Period:          period,
		IdentityVisible: identityVisible,
		DisplayCount:    normalizeUsageRankingLimit(limit),
		TotalUsers:      len(totals),
		Users:           make([]UsageRankingRow, 0, minInt(len(totals), normalizeUsageRankingLimit(limit))),
	}
	for _, total := range totals {
		response.TotalQuota += total.TotalQuota
		response.TotalTokens += total.TotalTokens
		response.TotalRequests += total.RequestCount
	}
	if len(totals) > 0 {
		row := usageRankingRow(totals[0], 1, userID, identityVisible)
		response.TopUser = &row
	}
	for index, total := range totals {
		rank := index + 1
		row := usageRankingRow(total, rank, userID, identityVisible)
		if rank <= response.DisplayCount {
			response.Users = append(response.Users, row)
		}
		if total.UserID == userID {
			self := row
			response.MyRank = &self
		}
	}
	return response
}

func usageRankingRow(total model.UsageRankingTotal, rank int, userID int, identityVisible bool) UsageRankingRow {
	username := total.Username
	if !identityVisible {
		username = maskUsageRankingUsername(total.Username)
	}
	return UsageRankingRow{
		Rank:         rank,
		Username:     username,
		TotalQuota:   total.TotalQuota,
		TotalTokens:  total.TotalTokens,
		RequestCount: total.RequestCount,
		IsSelf:       total.UserID == userID,
	}
}

func maskUsageRankingUsername(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return "***"
	}
	runeCount := utf8.RuneCountInString(username)
	if runeCount == 1 {
		return "*"
	}
	runes := []rune(username)
	if runeCount == 2 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + strings.Repeat("*", minInt(3, len(runes)-2)) + string(runes[len(runes)-1])
}
