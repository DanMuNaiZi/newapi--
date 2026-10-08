package model

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createReserveTestUser(t *testing.T, quota int) User {
	t.Helper()
	user := User{
		Username:    "reserve-user-" + common.GetRandomString(6),
		Password:    "unused-password-hash",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
		Quota:       quota,
		AffCode:     "reserve-aff-" + common.GetRandomString(8),
	}
	require.NoError(t, DB.Create(&user).Error)
	return user
}

func createReserveTestToken(t *testing.T, remainQuota int) Token {
	t.Helper()
	token := Token{
		UserId:      1,
		Key:         "reserve-token-" + common.GetRandomString(8),
		Name:        "reserve-test",
		Status:      common.TokenStatusEnabled,
		ExpiredTime: -1,
		RemainQuota: remainQuota,
	}
	require.NoError(t, token.Insert())
	return token
}

func getUserQuotaFromDB(t *testing.T, id int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").First(&user, id).Error)
	return user.Quota
}

func getTokenFromDB(t *testing.T, id int) Token {
	t.Helper()
	var token Token
	require.NoError(t, DB.First(&token, id).Error)
	return token
}

func resetBatchUpdateTestState(t *testing.T) {
	t.Helper()
	oldBatchEnabled := common.BatchUpdateEnabled
	common.BatchUpdateEnabled = false
	for i := range BatchUpdateTypeCount {
		batchUpdateLocks[i].Lock()
		batchUpdateStores[i] = make(map[int]int)
		batchUpdateLocks[i].Unlock()
	}
	t.Cleanup(func() {
		common.BatchUpdateEnabled = oldBatchEnabled
		for i := range BatchUpdateTypeCount {
			batchUpdateLocks[i].Lock()
			batchUpdateStores[i] = make(map[int]int)
			batchUpdateLocks[i].Unlock()
		}
	})
}

func TestTryReserveQuotaWithoutRedis(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)

	user := createReserveTestUser(t, 100)
	reserved, err := TryReserveUserQuota(user.Id, 60)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 40, getUserQuotaFromDB(t, user.Id))

	reserved, err = TryReserveUserQuota(user.Id, 41)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 40, getUserQuotaFromDB(t, user.Id))

	token := createReserveTestToken(t, 80)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 25, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	reloaded := getTokenFromDB(t, token.Id)
	assert.Equal(t, 55, reloaded.RemainQuota)
	assert.Equal(t, 25, reloaded.UsedQuota)

	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 56, false)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 55, getTokenFromDB(t, token.Id).RemainQuota)
}

func TestRedisBatchReserveNeverFallsBackToStaleDatabaseBalance(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)
	common.BatchUpdateEnabled = true

	user := createReserveTestUser(t, 10)
	reserved, err := TryReserveUserQuota(user.Id, 8)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 10, getUserQuotaFromDB(t, user.Id), "batch delta is not flushed yet")

	reserved, err = TryReserveUserQuota(user.Id, 3)
	require.NoError(t, err)
	assert.False(t, reserved, "stale DB balance must not authorize a second spend")
	cachedUser, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, cachedUser.Quota)

	token := createReserveTestToken(t, 9)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 7, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 3, false)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 9, getTokenFromDB(t, token.Id).RemainQuota)

	batchUpdate()
	assert.Equal(t, 2, getUserQuotaFromDB(t, user.Id))
	reloadedToken := getTokenFromDB(t, token.Id)
	assert.Equal(t, 2, reloadedToken.RemainQuota)
	assert.Equal(t, 7, reloadedToken.UsedQuota)
}

func TestBatchUpdateAccumulatesTwoMaximumRequestCharges(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	common.BatchUpdateEnabled = true

	user := createReserveTestUser(t, common.MaxQuota*2+100)
	require.NoError(t, DecreaseUserQuota(user.Id, common.MaxQuota, false))
	require.NoError(t, DecreaseUserQuota(user.Id, common.MaxQuota, false))

	batchUpdate()
	assert.Equal(t, 100, getUserQuotaFromDB(t, user.Id))
}

func TestBatchUpdateAccumulatorSaturatesOverflow(t *testing.T) {
	resetBatchUpdateTestState(t)

	addNewRecord(BatchUpdateTypeUserQuota, 1, math.MaxInt)
	addNewRecord(BatchUpdateTypeUserQuota, 1, 1)
	batchUpdateLocks[BatchUpdateTypeUserQuota].Lock()
	assert.Equal(t, math.MaxInt, batchUpdateStores[BatchUpdateTypeUserQuota][1])
	batchUpdateLocks[BatchUpdateTypeUserQuota].Unlock()

	batchUpdateLocks[BatchUpdateTypeUserQuota].Lock()
	batchUpdateStores[BatchUpdateTypeUserQuota] = make(map[int]int)
	batchUpdateLocks[BatchUpdateTypeUserQuota].Unlock()
	addNewRecord(BatchUpdateTypeUserQuota, 1, math.MinInt)
	addNewRecord(BatchUpdateTypeUserQuota, 1, -1)
	batchUpdateLocks[BatchUpdateTypeUserQuota].Lock()
	assert.Equal(t, math.MinInt, batchUpdateStores[BatchUpdateTypeUserQuota][1])
	batchUpdateLocks[BatchUpdateTypeUserQuota].Unlock()
}

func TestReserveFallsBackToDatabaseWhenRedisIsUnavailable(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 20)
	require.NoError(t, populateUserCache(user))
	server.Close()

	// Redis 故障时降级为数据库条件更新：服务保持可用且不会超扣。
	reserved, err := TryReserveUserQuota(user.Id, 5)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 15, getUserQuotaFromDB(t, user.Id))

	reserved, err = TryReserveUserQuota(user.Id, 16)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 15, getUserQuotaFromDB(t, user.Id))
}

func TestSynchronousReserveCompensatesCacheWhenPersistenceFails(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 10)
	require.NoError(t, populateUserCache(user))
	require.NoError(t, DB.Delete(&user).Error)

	reserved, err := TryReserveUserQuota(user.Id, 6)
	assert.False(t, reserved)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	cached, cacheErr := cacheGetUserBase(user.Id)
	require.NoError(t, cacheErr)
	assert.Equal(t, 10, cached.Quota)

	token := createReserveTestToken(t, 12)
	_, err = GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	require.NoError(t, DB.Delete(&token).Error)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 7, false)
	assert.False(t, reserved)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	cachedToken, cacheErr := cacheGetTokenByKey(token.Key)
	require.NoError(t, cacheErr)
	assert.Equal(t, 12, cachedToken.RemainQuota)
	assert.Zero(t, cachedToken.UsedQuota)
}

func TestTokenCacheInitPreservesLiveQuotaAndFenceBlocksStaleSnapshot(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	loaded, err := GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	stale := *loaded

	result, err := cacheApplyTokenQuotaDelta(token.Id, token.Key, -70)
	require.NoError(t, err)
	require.Equal(t, cacheQuotaOK, result)

	// 已存在的哈希只刷新 TTL：数据库快照不得覆盖已被原子预扣的余额。
	code, err := cacheInitToken(stale)
	require.NoError(t, err)
	assert.Equal(t, 2, code)
	cached, err := cacheGetTokenByKey(token.Key)
	require.NoError(t, err)
	assert.Equal(t, 30, cached.RemainQuota)

	// 变更期间：fence 删除缓存并拦截并发读者手中的过期快照。
	require.NoError(t, invalidateTokenCacheForMutation(token.Key))
	code, err = cacheInitToken(stale)
	require.NoError(t, err)
	assert.Zero(t, code, "the pre-mutation snapshot must not be published while fenced")
	_, err = cacheGetTokenByKey(token.Key)
	assert.Error(t, err)

	// fence 过期后可重新从数据库水合。
	server.FastForward(time.Duration(tokenCacheFenceSeconds+1) * time.Second)
	fresh, err := GetTokenByKey(token.Key, false)
	require.NoError(t, err)
	assert.Equal(t, 100, fresh.RemainQuota)
	cached, err = cacheGetTokenByKey(token.Key)
	require.NoError(t, err)
	assert.Equal(t, 100, cached.RemainQuota)
}

func prepareRewardWalletCacheTest(t *testing.T) User {
	t.Helper()
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)
	require.NoError(t, DB.AutoMigrate(&RewardGrant{}, &LotteryResult{}))
	for _, table := range []any{&RewardGrant{}, &LotteryResult{}} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []any{&RewardGrant{}, &LotteryResult{}} {
			require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(table).Error)
		}
	})
	user := createReserveTestUser(t, 100)
	common.BatchUpdateEnabled = true
	reserved, err := TryReserveUserQuota(user.Id, 80)
	require.NoError(t, err)
	require.True(t, reserved)
	require.Equal(t, 100, getUserQuotaFromDB(t, user.Id))
	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	require.Equal(t, 20, cached.Quota)
	return user
}

func TestRewardQuotaCachePreservesPendingReservationAndReplay(t *testing.T) {
	user := prepareRewardWalletCacheTest(t)
	grant, err := GrantReward(RewardSourcePublicPool, 101, user.Id, RewardSnapshot{Type: RewardTypeQuota, Quota: 10})
	require.NoError(t, err)
	require.Equal(t, RewardGrantStatusSucceeded, grant.Status)

	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 30, cached.Quota, "reward credit must preserve the unflushed wallet reservation")
	assert.Equal(t, 110, getUserQuotaFromDB(t, user.Id))
	_, err = GrantReward(RewardSourcePublicPool, 101, user.Id, RewardSnapshot{Type: RewardTypeQuota, Quota: 999})
	require.NoError(t, err)
	_, err = RetryRewardGrant(grant.Id)
	require.NoError(t, err)
	cached, err = GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 30, cached.Quota, "replaying the same grant must not credit the cache again")
	reserved, err := TryReserveUserQuota(user.Id, 80)
	require.NoError(t, err)
	assert.False(t, reserved, "the unflushed DB balance must not fund another request")
	batchUpdate()
	assert.Equal(t, 30, getUserQuotaFromDB(t, user.Id))
}

func TestRewardQuotaCacheOnlyCreditsCommittedRetry(t *testing.T) {
	user := prepareRewardWalletCacheTest(t)
	const callbackName = "test:reject_reward_completion"
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		updates, ok := tx.Statement.Dest.(map[string]any)
		if tx.Statement.Table == "reward_grants" && ok && updates["status"] == RewardGrantStatusSucceeded {
			tx.AddError(errors.New("reward completion rejected"))
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Update().Remove(callbackName) })
	grant, err := GrantReward(RewardSourceReferralCampaign, 102, user.Id, RewardSnapshot{Type: RewardTypeQuota, Quota: 10})
	require.ErrorContains(t, err, "reward completion rejected")
	require.Equal(t, RewardGrantStatusFailed, grant.Status)
	assert.Equal(t, 100, getUserQuotaFromDB(t, user.Id), "the quota increment must roll back with the grant")
	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 20, cached.Quota)

	require.NoError(t, DB.Callback().Update().Remove(callbackName))
	grant, err = RetryRewardGrant(grant.Id)
	require.NoError(t, err)
	assert.Equal(t, RewardGrantStatusSucceeded, grant.Status)
	_, err = RetryRewardGrant(grant.Id)
	require.NoError(t, err)
	cached, err = GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 30, cached.Quota)
	batchUpdate()
	assert.Equal(t, 30, getUserQuotaFromDB(t, user.Id))
}

func TestRewardQuotaCacheSkipsConcurrentSucceededTransaction(t *testing.T) {
	user := prepareRewardWalletCacheTest(t)
	firstRead := make(chan struct{})
	resumeFirst := make(chan struct{})
	var release sync.Once
	var paused atomic.Bool
	const callbackName = "test:pause_pending_reward_read"
	require.NoError(t, DB.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		grant, ok := tx.Statement.Dest.(*RewardGrant)
		if tx.Statement.Table == "reward_grants" && ok && grant.SourceId == 103 && grant.Status == RewardGrantStatusPending && paused.CompareAndSwap(false, true) {
			close(firstRead)
			<-resumeFirst
		}
	}))
	t.Cleanup(func() {
		release.Do(func() { close(resumeFirst) })
		_ = DB.Callback().Query().Remove(callbackName)
	})
	type grantOutcome struct {
		grant *RewardGrant
		err   error
	}
	firstDone := make(chan grantOutcome, 1)
	go func() {
		grant, err := GrantReward(RewardSourcePublicPool, 103, user.Id, RewardSnapshot{Type: RewardTypeQuota, Quota: 10})
		firstDone <- grantOutcome{grant: grant, err: err}
	}()
	select {
	case <-firstRead:
	case <-time.After(3 * time.Second):
		t.Fatal("the first grant did not reach its pending snapshot")
	}
	winningGrant, err := GrantReward(RewardSourcePublicPool, 103, user.Id, RewardSnapshot{Type: RewardTypeQuota, Quota: 999})
	require.NoError(t, err)
	require.Equal(t, RewardGrantStatusSucceeded, winningGrant.Status)
	release.Do(func() { close(resumeFirst) })
	select {
	case outcome := <-firstDone:
		require.NoError(t, outcome.err)
		assert.Equal(t, winningGrant.Id, outcome.grant.Id)
		assert.Equal(t, RewardGrantStatusSucceeded, outcome.grant.Status)
	case <-time.After(3 * time.Second):
		t.Fatal("the concurrent grant did not finish")
	}
	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 30, cached.Quota, "only the winning transaction credits the stored 10-quota snapshot")
	batchUpdate()
	assert.Equal(t, 30, getUserQuotaFromDB(t, user.Id))
}

func TestRewardSubscriptionCacheRefreshKeepsPendingWalletReservation(t *testing.T) {
	user := prepareRewardWalletCacheTest(t)
	plan := &SubscriptionPlan{Title: "Earned plan", DurationUnit: "day", DurationValue: 1, TotalAmount: 50, UpgradeGroup: "vip"}
	require.NoError(t, DB.Create(plan).Error)
	planSnapshot, err := common.Marshal(plan)
	require.NoError(t, err)
	grant, err := GrantReward(RewardSourceReferralCampaign, 104, user.Id, RewardSnapshot{
		Type: RewardTypeSubscription, SubscriptionPlanId: plan.Id, SubscriptionPlanSnapshot: string(planSnapshot),
	})
	require.NoError(t, err)
	_, err = RetryRewardGrant(grant.Id)
	require.NoError(t, err)
	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, "vip", cached.Group)
	assert.Equal(t, 20, cached.Quota, "group refresh must retain the wallet reservation")
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	reserved, err := TryReserveUserQuota(user.Id, 80)
	require.NoError(t, err)
	assert.False(t, reserved)
	batchUpdate()
	assert.Equal(t, 20, getUserQuotaFromDB(t, user.Id))
}

func TestLotteryRewardQuotaCachePreservesPendingReservation(t *testing.T) {
	for _, mode := range []LotteryFulfillmentMode{LotteryFulfillmentAuto, LotteryFulfillmentSelfClaim} {
		t.Run(string(mode), func(t *testing.T) {
			user := prepareRewardWalletCacheTest(t)
			status := "pending"
			if mode == LotteryFulfillmentAuto {
				status = "pending_auto"
			}
			result := LotteryResult{PlanId: 201, UserId: user.Id, RewardType: LotteryRewardQuota, Quota: 10, FulfillmentMode: mode, FulfillmentStatus: status}
			require.NoError(t, DB.Create(&result).Error)
			if mode == LotteryFulfillmentAuto {
				require.NoError(t, FulfillLotteryResult(result.Id))
				require.Error(t, FulfillLotteryResult(result.Id))
			} else {
				require.NoError(t, ClaimLotteryResult(result.Id, user.Id))
				require.Error(t, ClaimLotteryResult(result.Id, user.Id))
			}
			cached, err := GetUserCache(user.Id)
			require.NoError(t, err)
			assert.Equal(t, 30, cached.Quota)
			reserved, err := TryReserveUserQuota(user.Id, 80)
			require.NoError(t, err)
			assert.False(t, reserved)
			batchUpdate()
			assert.Equal(t, 30, getUserQuotaFromDB(t, user.Id))
		})
	}
}

func TestLotteryRewardQuotaCacheOnlyCreditsCommittedClaim(t *testing.T) {
	user := prepareRewardWalletCacheTest(t)
	result := LotteryResult{PlanId: 202, UserId: user.Id, RewardType: LotteryRewardQuota, Quota: 10, FulfillmentMode: LotteryFulfillmentSelfClaim, FulfillmentStatus: "pending"}
	require.NoError(t, DB.Create(&result).Error)
	const callbackName = "test:reject_lottery_completion"
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "lottery_results" {
			tx.AddError(errors.New("lottery completion rejected"))
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Update().Remove(callbackName) })
	require.ErrorContains(t, ClaimLotteryResult(result.Id, user.Id), "lottery completion rejected")
	assert.Equal(t, 100, getUserQuotaFromDB(t, user.Id))
	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 20, cached.Quota)
	require.NoError(t, DB.Callback().Update().Remove(callbackName))
	require.NoError(t, ClaimLotteryResult(result.Id, user.Id))
	cached, err = GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 30, cached.Quota)
	batchUpdate()
	assert.Equal(t, 30, getUserQuotaFromDB(t, user.Id))
}

func TestLotteryRewardSubscriptionCacheRefreshKeepsPendingWalletReservation(t *testing.T) {
	user := prepareRewardWalletCacheTest(t)
	plan := &SubscriptionPlan{Title: "Lottery plan", DurationUnit: "day", DurationValue: 1, TotalAmount: 50, UpgradeGroup: "vip"}
	require.NoError(t, DB.Create(plan).Error)
	planSnapshot, err := common.Marshal(plan)
	require.NoError(t, err)
	result := LotteryResult{PlanId: 203, UserId: user.Id, RewardType: LotteryRewardSubscription, SubscriptionPlanId: plan.Id, SubscriptionSnapshot: string(planSnapshot), FulfillmentMode: LotteryFulfillmentAuto, FulfillmentStatus: "pending_auto"}
	require.NoError(t, DB.Create(&result).Error)
	require.NoError(t, FulfillLotteryResult(result.Id))
	require.Error(t, FulfillLotteryResult(result.Id))
	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, "vip", cached.Group)
	assert.Equal(t, 20, cached.Quota)
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	batchUpdate()
	assert.Equal(t, 20, getUserQuotaFromDB(t, user.Id))
}
