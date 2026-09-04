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
package model

import "github.com/QuantumNous/new-api/common"

// UsageRankingTotal is the aggregated consume-log usage for one user.
type UsageRankingTotal struct {
	UserID       int    `json:"user_id" gorm:"column:user_id"`
	Username     string `json:"username" gorm:"column:username"`
	TotalQuota   int64  `json:"total_quota" gorm:"column:total_quota"`
	TotalTokens  int64  `json:"total_tokens" gorm:"column:total_tokens"`
	RequestCount int64  `json:"request_count" gorm:"column:request_count"`
}

// GetUsageRankingTotals returns users with consume activity in the requested
// time range. The query intentionally uses only portable aggregate functions
// so it works with the supported log database dialects.
func GetUsageRankingTotals(startTime int64, endTime int64) ([]UsageRankingTotal, error) {
	var rows []UsageRankingTotal
	query := LOG_DB.Table("logs").
		Select("user_id, MAX(username) AS username, COALESCE(SUM(quota), 0) AS total_quota, COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) AS total_tokens, COUNT(*) AS request_count").
		Where("type = ?", LogTypeConsume).
		Group("user_id").
		Having("COALESCE(SUM(quota), 0) > 0 OR COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) > 0").
		Order("total_quota DESC, total_tokens DESC, request_count DESC, user_id ASC")
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []UsageRankingTotal{}, nil
	}

	// Logs can live in a separate database. Aggregate there first, then use
	// bounded main-database lookups to keep only currently enabled users.
	// Besides avoiding a cross-database join, this prevents an installation
	// with many enabled users from exceeding a driver's IN-parameter limit.
	activeUserIDs := make(map[int]struct{}, len(rows))
	const userIDBatchSize = 500
	for start := 0; start < len(rows); start += userIDBatchSize {
		end := min(start+userIDBatchSize, len(rows))
		userIDs := make([]int, 0, end-start)
		for _, row := range rows[start:end] {
			if row.UserID > 0 {
				userIDs = append(userIDs, row.UserID)
			}
		}
		if len(userIDs) == 0 {
			continue
		}
		var batchActiveUserIDs []int
		if err := DB.Model(&User{}).
			Where("status = ? AND id IN ?", common.UserStatusEnabled, userIDs).
			Pluck("id", &batchActiveUserIDs).Error; err != nil {
			return nil, err
		}
		for _, userID := range batchActiveUserIDs {
			activeUserIDs[userID] = struct{}{}
		}
	}

	filteredRows := make([]UsageRankingTotal, 0, len(rows))
	for _, row := range rows {
		if _, ok := activeUserIDs[row.UserID]; ok {
			filteredRows = append(filteredRows, row)
		}
	}
	return filteredRows, nil
}
