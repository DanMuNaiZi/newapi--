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

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUsageRankingTotalsAggregatesConsumeLogs(t *testing.T) {
	truncateTables(t)
	users := []*User{
		{Username: "ranking-alice", Password: "password", Status: common.UserStatusEnabled, AffCode: "ranking-alice"},
		{Username: "ranking-bob", Password: "password", Status: common.UserStatusEnabled, AffCode: "ranking-bob"},
		{Username: "ranking-topup", Password: "password", Status: common.UserStatusEnabled, AffCode: "ranking-topup"},
		{Username: "ranking-outside", Password: "password", Status: common.UserStatusEnabled, AffCode: "ranking-outside"},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, DB.Create([]Log{
		{UserId: users[0].Id, Username: "alice", CreatedAt: 100, Type: LogTypeConsume, Quota: 100, PromptTokens: 10, CompletionTokens: 5},
		{UserId: users[0].Id, Username: "alice", CreatedAt: 200, Type: LogTypeConsume, Quota: 40, PromptTokens: 2, CompletionTokens: 3},
		{UserId: users[1].Id, Username: "bob", CreatedAt: 150, Type: LogTypeConsume, Quota: 200, PromptTokens: 20, CompletionTokens: 10},
		{UserId: users[2].Id, Username: "ignored", CreatedAt: 150, Type: LogTypeTopup, Quota: 999, PromptTokens: 99},
		{UserId: users[3].Id, Username: "outside", CreatedAt: 300, Type: LogTypeConsume, Quota: 800, PromptTokens: 80},
	}).Error)

	rows, err := GetUsageRankingTotals(100, 250)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, users[1].Id, rows[0].UserID)
	assert.Equal(t, int64(200), rows[0].TotalQuota)
	assert.Equal(t, int64(30), rows[0].TotalTokens)
	assert.Equal(t, int64(1), rows[0].RequestCount)
	assert.Equal(t, users[0].Id, rows[1].UserID)
	assert.Equal(t, int64(140), rows[1].TotalQuota)
	assert.Equal(t, int64(20), rows[1].TotalTokens)
	assert.Equal(t, int64(2), rows[1].RequestCount)
}

func TestGetUsageRankingTotalsExcludesInvalidUsers(t *testing.T) {
	truncateTables(t)
	active := &User{Username: "ranking-active", Password: "password", Status: common.UserStatusEnabled, AffCode: "ranking-active"}
	disabled := &User{Username: "ranking-disabled", Password: "password", Status: common.UserStatusDisabled, AffCode: "ranking-disabled"}
	deleted := &User{Username: "ranking-deleted", Password: "password", Status: common.UserStatusEnabled, AffCode: "ranking-deleted"}
	require.NoError(t, DB.Create([]*User{active, disabled, deleted}).Error)
	require.NoError(t, DB.Delete(deleted).Error)
	require.NoError(t, DB.Create([]Log{
		{UserId: active.Id, Username: active.Username, Type: LogTypeConsume, CreatedAt: 100, Quota: 100, PromptTokens: 10, CompletionTokens: 5},
		{UserId: disabled.Id, Username: disabled.Username, Type: LogTypeConsume, CreatedAt: 100, Quota: 900, PromptTokens: 90, CompletionTokens: 9},
		{UserId: deleted.Id, Username: deleted.Username, Type: LogTypeConsume, CreatedAt: 100, Quota: 800, PromptTokens: 80, CompletionTokens: 8},
		{UserId: 999999, Username: "ranking-missing", Type: LogTypeConsume, CreatedAt: 100, Quota: 700, PromptTokens: 70, CompletionTokens: 7},
	}).Error)

	rows, err := GetUsageRankingTotals(100, 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, active.Id, rows[0].UserID)
	assert.Equal(t, int64(100), rows[0].TotalQuota)
}
