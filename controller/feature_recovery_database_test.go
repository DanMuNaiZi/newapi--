package controller

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relaydto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func recoveryMatrixQRCode() string {
	return "data:image/png;base64," + strings.Repeat("A", 159*1024)
}

// These persisted model shapes are unchanged from the deployed custom baseline
// de534a21. The audit matrix separately freezes the released User/Log schemas.
// Seed before InitDB so migrations must preserve real orders and snapshots.
func seedRecoveredUpgradeRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TopUp{}, &model.SubscriptionPlan{}, &model.SubscriptionOrder{}, &model.UserSubscription{}, &model.Redemption{}, &model.LotteryPlan{}, &model.LotteryPrize{}, &model.LotteryResult{}))
	require.NoError(t, db.Create(&model.Option{Key: manualWechatQRCodeOptionKey, Value: recoveryMatrixQRCode()}).Error)
	plan := &model.SubscriptionPlan{Title: "Baseline subscription", PriceAmount: 19.9, Currency: "CNY", DurationUnit: model.SubscriptionDurationDay, DurationValue: 3, Enabled: true, TotalAmount: 4321, AllowWalletOverflow: common.GetPointer(false), MaxPurchasePerUser: 1, ConsumePriority: common.GetPointer(8)}
	require.NoError(t, db.Create(plan).Error)
	snapshotPlan := *plan
	snapshotPlan.Id = 0 // Historical controller snapshots omit the plan ID.
	snapshot, err := common.Marshal(snapshotPlan)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TopUp{UserId: 1, Amount: 2, Money: 14.6, TradeNo: "recovery-baseline-topup", PaymentProvider: model.PaymentProviderManualWechat, PaymentMethod: model.PaymentMethodManualWechat, Status: common.TopUpStatusSuccess, CreateTime: 100, CompleteTime: 101}).Error)
	require.NoError(t, db.Create(&model.SubscriptionOrder{UserId: 1, PlanId: plan.Id, Money: 19.9, TradeNo: "recovery-baseline-subscription", PaymentProvider: model.PaymentProviderManualWechat, PaymentMethod: model.PaymentMethodManualWechat, Status: common.TopUpStatusSuccess, CreateTime: 100, CompleteTime: 101, ProviderPayload: "historical completion"}).Error)
	require.NoError(t, db.Create(&model.UserSubscription{UserId: 1, PlanId: plan.Id, AmountTotal: 4321, AmountUsed: 123, StartTime: 100, EndTime: 200, Status: "expired", Source: "order", AllowWalletOverflow: false, UserConsumePriority: common.GetPointer(3)}).Error)
	require.NoError(t, db.Create(&model.Redemption{UserId: 1, Key: "baseline-subscription-code", Name: "Baseline reward", RewardType: model.RedemptionRewardSubscription, SubscriptionPlanId: plan.Id, SubscriptionSnapshot: string(snapshot), Batch: "historical-batch", SourceRef: "baseline-source", Remark: "retained remark", Status: common.RedemptionCodeStatusEnabled}).Error)
	lottery := &model.LotteryPlan{Title: "Baseline lottery", Status: model.LotteryPlanStatusFinished, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 1}
	require.NoError(t, db.Create(lottery).Error)
	prize := &model.LotteryPrize{PlanId: lottery.Id, Name: "Baseline subscription prize", Quantity: 1, RewardType: model.LotteryRewardSubscription, SubscriptionPlanId: plan.Id, SubscriptionSnapshot: string(snapshot), FulfillmentMode: model.LotteryFulfillmentSelfClaim}
	require.NoError(t, db.Create(prize).Error)
	prizeSnapshot, err := common.Marshal(prize)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.LotteryResult{PlanId: lottery.Id, UserId: 1, PrizeId: prize.Id, PrizeSnapshot: string(prizeSnapshot), RewardType: prize.RewardType, SubscriptionPlanId: plan.Id, SubscriptionSnapshot: string(snapshot), FulfillmentMode: model.LotteryFulfillmentSelfClaim, FulfillmentStatus: "pending"}).Error)
}

func newRecoveryMatrixUser(t *testing.T, username string, quota int) *model.User {
	t.Helper()
	user := &model.User{Username: username, Password: "fixture-placeholder", AffCode: username, Group: "default", Status: common.UserStatusEnabled, Quota: quota}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

func verifyRecoveredFeatureTransactions(t *testing.T) {
	t.Helper()
	previousQuotaPerUnit := common.QuotaPerUnit
	previousPaymentSetting := *operation_setting.GetPaymentSetting()
	common.OptionMapRWMutex.Lock()
	previousOptions := maps.Clone(common.OptionMap)
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		*operation_setting.GetPaymentSetting() = previousPaymentSetting
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
	})

	t.Run("baseline_snapshots", func(t *testing.T) {
		var old model.Redemption
		result := model.DB.Where("name = ?", "Baseline reward").Find(&old)
		require.NoError(t, result.Error)
		if result.RowsAffected == 0 {
			return // Fresh databases have no historical rows.
		}
		var option model.Option
		require.NoError(t, model.DB.Where(&model.Option{Key: manualWechatQRCodeOptionKey}).First(&option).Error)
		assert.Equal(t, recoveryMatrixQRCode(), option.Value)
		assert.Equal(t, "historical-batch", old.Batch)
		assert.Equal(t, "baseline-source", old.SourceRef)
		assert.Equal(t, "retained remark", old.Remark)
		var historicalPlan model.SubscriptionPlan
		require.NoError(t, model.DB.First(&historicalPlan, old.SubscriptionPlanId).Error)
		require.NotNil(t, historicalPlan.ConsumePriority)
		assert.Equal(t, 8, *historicalPlan.ConsumePriority)
		order := model.GetSubscriptionOrderByTradeNo("recovery-baseline-subscription")
		require.NotNil(t, order)
		assert.Equal(t, common.TopUpStatusSuccess, order.Status)
		assert.Equal(t, "historical completion", order.ProviderPayload)
		topup := model.GetTopUpByTradeNo("recovery-baseline-topup")
		require.NotNil(t, topup)
		assert.Equal(t, common.TopUpStatusSuccess, topup.Status)
		assert.EqualValues(t, 2, topup.Amount)
		var historical model.UserSubscription
		require.NoError(t, model.DB.Where("user_id = ? AND source = ?", 1, "order").First(&historical).Error)
		assert.EqualValues(t, 123, historical.AmountUsed)
		assert.Equal(t, "expired", historical.Status)
		require.NotNil(t, historical.UserConsumePriority)
		assert.Equal(t, 3, *historical.UserConsumePriority)
		outcome, err := model.Redeem(old.Key, 1)
		require.NoError(t, err)
		assert.Equal(t, old.SubscriptionPlanId, outcome.SubscriptionPlanId)
		var redeemed model.UserSubscription
		require.NoError(t, model.DB.First(&redeemed, outcome.SubscriptionId).Error)
		assert.EqualValues(t, 4321, redeemed.AmountTotal)
		assert.False(t, redeemed.AllowWalletOverflow)
		_, err = model.Redeem(old.Key, 1)
		require.ErrorIs(t, err, model.ErrRedeemFailed)
		var lottery model.LotteryResult
		require.NoError(t, model.DB.Where("user_id = ?", 1).First(&lottery).Error)
		var snapshot model.LotteryPrize
		require.NoError(t, common.UnmarshalJsonStr(lottery.PrizeSnapshot, &snapshot))
		assert.Equal(t, "Baseline subscription prize", snapshot.Name)
		require.NoError(t, model.ClaimLotteryResult(lottery.Id, 1))
		require.Error(t, model.ClaimLotteryResult(lottery.Id, 1))
		var subscriptions []model.UserSubscription
		require.NoError(t, model.DB.Where("user_id = ?", 1).Find(&subscriptions).Error)
		require.Len(t, subscriptions, 3, "earned snapshots bypass the exhausted purchase limit without duplicating a claim")
		for _, subscription := range subscriptions {
			assert.EqualValues(t, 4321, subscription.AmountTotal)
			assert.False(t, subscription.AllowWalletOverflow)
		}
	})

	t.Run("large_payment_option", func(t *testing.T) {
		require.NoError(t, model.UpdateOptionsBulk(map[string]string{manualWechatQRCodeOptionKey: recoveryMatrixQRCode()}))
		var option model.Option
		require.NoError(t, model.DB.Where(&model.Option{Key: manualWechatQRCodeOptionKey}).First(&option).Error)
		assert.Equal(t, recoveryMatrixQRCode(), option.Value, "the QR DataURL must not be truncated by the database")
	})

	t.Run("manual_payment_topup_list", func(t *testing.T) {
		user := newRecoveryMatrixUser(t, "matrix-manual-list", 7)
		now := common.GetTimestamp()
		paid, _, err := model.CreateOrReuseManualWechatTopUp(user.Id, 2, 14.6, now)
		require.NoError(t, err)
		require.NoError(t, model.CompleteManualWechatTopUp(paid.TradeNo, "127.0.0.1"))
		pending, _, err := model.CreateOrReuseManualWechatTopUp(user.Id, 3, 21.9, now)
		require.NoError(t, err)
		expired, _, err := model.CreateOrReuseManualWechatTopUp(user.Id, 4, 29.2, now)
		require.NoError(t, err)
		require.NoError(t, model.UpdatePendingTopUpStatus(expired.TradeNo, model.PaymentProviderManualWechat, common.TopUpStatusExpired))
		other := &model.TopUp{UserId: user.Id, Amount: 0, Money: 8, TradeNo: "recovery-other-subscription-mirror", PaymentProvider: model.PaymentProviderStripe, PaymentMethod: model.PaymentMethodStripe, Status: common.TopUpStatusSuccess, CreateTime: now}
		require.NoError(t, other.Insert())
		plan := &model.SubscriptionPlan{Title: "Manual list plan", PriceAmount: 19.9, Currency: "CNY", DurationUnit: model.SubscriptionDurationDay, DurationValue: 3, Enabled: true, TotalAmount: 777}
		require.NoError(t, model.DB.Create(plan).Error)
		subscription, _, err := model.CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, now)
		require.NoError(t, err)
		require.NoError(t, model.CompleteSubscriptionOrder(subscription.TradeNo, "verified", model.PaymentProviderManualWechat, model.PaymentMethodManualWechat))
		mirror := model.GetTopUpByTradeNo(subscription.TradeNo)
		require.NotNil(t, mirror)
		assert.Zero(t, mirror.Amount, "subscription accounting remains a historical top-up ledger entry")
		before := *mirror

		router := gin.New()
		router.Use(func(c *gin.Context) { setManualWechatAdminContext(c, user.Id) })
		router.GET("/api/user/topup", GetAllTopUps)
		router.GET("/api/user/topup/self", GetUserTopUps)
		router.GET("/api/subscription/admin/orders", AdminListSubscriptionOrders)
		manualQuery := fmt.Sprintf("/api/user/topup?payment_provider=manual_wechat&user_id=%d", user.Id)
		allManualTrades := []string{expired.TradeNo, pending.TradeNo, paid.TradeNo}
		if historical := model.GetTopUpByTradeNo("recovery-baseline-topup"); historical != nil {
			allManualTrades = append(allManualTrades, historical.TradeNo)
		}
		for _, tc := range []struct {
			name        string
			path        string
			total       int
			trades      []string
			balanceOnly bool
		}{
			{"manual_all_users", "/api/user/topup?payment_provider=manual_wechat&page_size=100", len(allManualTrades), allManualTrades, true},
			{"manual_page_one", manualQuery + "&page=1&page_size=1", 3, []string{expired.TradeNo}, true},
			{"manual_page_two", manualQuery + "&page=2&page_size=1", 3, []string{pending.TradeNo}, true},
			{"manual_page_three", manualQuery + "&page=3&page_size=1", 3, []string{paid.TradeNo}, true},
			{"manual_page_four", manualQuery + "&page=4&page_size=1", 3, nil, true},
			{"manual_success", manualQuery + "&status=success", 1, []string{paid.TradeNo}, true},
			{"manual_pending", manualQuery + "&status=pending", 1, []string{pending.TradeNo}, true},
			{"manual_expired", manualQuery + "&status=expired", 1, []string{expired.TradeNo}, true},
			{"manual_username_search", manualQuery + "&keyword=" + user.Username, 3, []string{expired.TradeNo, pending.TradeNo, paid.TradeNo}, true},
			{"manual_subscription_search", manualQuery + "&keyword=" + subscription.TradeNo, 0, nil, true},
			{"admin_history", fmt.Sprintf("/api/user/topup?user_id=%d", user.Id), 5, []string{subscription.TradeNo, other.TradeNo, expired.TradeNo, pending.TradeNo, paid.TradeNo}, false},
			{"other_provider_history", fmt.Sprintf("/api/user/topup?payment_provider=stripe&user_id=%d", user.Id), 1, []string{other.TradeNo}, false},
			{"user_history", "/api/user/topup/self", 5, []string{subscription.TradeNo, other.TradeNo, expired.TradeNo, pending.TradeNo, paid.TradeNo}, false},
			{"user_subscription_search", "/api/user/topup/self?keyword=" + subscription.TradeNo, 1, []string{subscription.TradeNo}, false},
			{"subscription_tab", fmt.Sprintf("/api/subscription/admin/orders?payment_provider=manual_wechat&user_id=%d&status=success", user.Id), 1, []string{subscription.TradeNo}, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
				require.Equal(t, http.StatusOK, recorder.Code)
				var response struct {
					Success bool `json:"success"`
					Data    struct {
						Total int `json:"total"`
						Items []struct {
							TradeNo string `json:"trade_no"`
							Amount  int64  `json:"amount"`
						} `json:"items"`
					} `json:"data"`
				}
				require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
				require.True(t, response.Success, recorder.Body.String())
				assert.Equal(t, tc.total, response.Data.Total)
				trades := make([]string, 0, len(response.Data.Items))
				for _, item := range response.Data.Items {
					trades = append(trades, item.TradeNo)
					if tc.balanceOnly {
						assert.Positive(t, item.Amount, "the balance tab must only contain quota purchases")
					}
				}
				assert.Equal(t, append([]string{}, tc.trades...), trades)
			})
		}
		assert.Equal(t, before, *model.GetTopUpByTradeNo(subscription.TradeNo), "list projection must preserve subscription accounting")
		stored, err := model.GetUserById(user.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 1_000_007, stored.Quota, "subscription settlement must not credit the wallet")
	})

	t.Run("manual_payments", func(t *testing.T) {
		user := newRecoveryMatrixUser(t, "matrix-manual", 7)
		order, reused, err := model.CreateOrReuseManualWechatTopUp(user.Id, 2, 14.6, 1000)
		require.NoError(t, err)
		assert.False(t, reused)
		same, reused, err := model.CreateOrReuseManualWechatTopUp(user.Id, 2, 14.6, 1030)
		require.NoError(t, err)
		assert.True(t, reused)
		assert.Equal(t, order.Id, same.Id)
		wrong := &model.TopUp{UserId: user.Id, Amount: 2, Money: 14.6, TradeNo: "recovery-wrong-provider", PaymentProvider: model.PaymentProviderEpay, PaymentMethod: model.PaymentProviderEpay, Status: common.TopUpStatusPending}
		require.NoError(t, wrong.Insert())
		require.ErrorIs(t, model.CompleteManualWechatTopUp(wrong.TradeNo, "127.0.0.1"), model.ErrPaymentMethodMismatch)
		require.Equal(t, common.TopUpStatusPending, model.GetTopUpByTradeNo(wrong.TradeNo).Status)
		for range 2 {
			require.NoError(t, model.CompleteManualWechatTopUp(order.TradeNo, "127.0.0.1"))
		}
		stored, err := model.GetUserById(user.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 1_000_007, stored.Quota)
		var logs int64
		require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("user_id = ? AND type = ?", user.Id, model.LogTypeTopup).Count(&logs).Error)
		assert.EqualValues(t, 1, logs)

		plan := &model.SubscriptionPlan{Title: "Limited manual plan", PriceAmount: 19.9, Currency: "CNY", DurationUnit: model.SubscriptionDurationDay, DurationValue: 3, Enabled: true, TotalAmount: 777, MaxPurchasePerUser: 1, AllowWalletOverflow: common.GetPointer(false)}
		require.NoError(t, model.DB.Create(plan).Error)
		model.InvalidateSubscriptionPlanCache(plan.Id)
		first, _, err := model.CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, 19.9, 2000)
		require.NoError(t, err)
		second, _, err := model.CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, 19.9, 2061)
		require.NoError(t, err)
		require.ErrorIs(t, model.CompleteSubscriptionOrder(first.TradeNo, "", model.PaymentProviderStripe, model.PaymentMethodStripe), model.ErrPaymentMethodMismatch)
		for range 2 {
			require.NoError(t, model.CompleteSubscriptionOrder(first.TradeNo, "verified", model.PaymentProviderManualWechat, model.PaymentMethodManualWechat))
		}
		require.Error(t, model.CompleteSubscriptionOrder(second.TradeNo, "", model.PaymentProviderManualWechat, model.PaymentMethodManualWechat))
		require.Equal(t, common.TopUpStatusPending, model.GetSubscriptionOrderByTradeNo(second.TradeNo).Status)
		assert.Nil(t, model.GetTopUpByTradeNo(second.TradeNo), "purchase-limit rejection must roll back payment accounting")
		completed := model.GetTopUpByTradeNo(first.TradeNo)
		require.NotNil(t, completed)
		assert.Equal(t, model.PaymentProviderManualWechat, completed.PaymentProvider)
		assert.Equal(t, model.PaymentMethodManualWechat, completed.PaymentMethod)
		assert.Equal(t, common.TopUpStatusSuccess, completed.Status)
		var subscriptions []model.UserSubscription
		require.NoError(t, model.DB.Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Find(&subscriptions).Error)
		require.Len(t, subscriptions, 1)
		assert.EqualValues(t, 777, subscriptions[0].AmountTotal)
		assert.False(t, subscriptions[0].AllowWalletOverflow)
		orders, total, err := model.GetAdminSubscriptionOrders(&common.PageInfo{Page: 1, PageSize: 20}, model.AdminSubscriptionOrderFilter{UserId: user.Id, PlanId: plan.Id, PaymentProvider: model.PaymentProviderManualWechat})
		require.NoError(t, err)
		assert.EqualValues(t, 2, total)
		require.Len(t, orders, 2)
		assert.Equal(t, user.Username, orders[0].Username)
		assert.Equal(t, plan.Title, orders[0].PlanTitle)
	})

	for _, tc := range []struct {
		name         string
		quota        int
		quotaPerUnit float64
		expected     error
	}{
		{"wallet_ceiling", common.MaxWalletQuota - 1, 500_000, model.ErrTopUpQuotaLimitExceeded},
		{"conversion_overflow", 3, float64(common.MaxWalletQuota + 1), model.ErrInvalidTopUpQuota},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previous := common.QuotaPerUnit
			common.QuotaPerUnit = tc.quotaPerUnit
			defer func() { common.QuotaPerUnit = previous }()
			user := newRecoveryMatrixUser(t, tc.name, tc.quota)
			order, _, err := model.CreateOrReuseManualWechatTopUp(user.Id, 1, 7.3, 3000)
			require.NoError(t, err)
			require.ErrorIs(t, model.CompleteManualWechatTopUp(order.TradeNo, "127.0.0.1"), tc.expected)
			stored, err := model.GetUserById(user.Id, true)
			require.NoError(t, err)
			assert.Equal(t, tc.quota, stored.Quota)
			assert.Equal(t, common.TopUpStatusPending, model.GetTopUpByTradeNo(order.TradeNo).Status)
		})
	}

	t.Run("redemption_snapshot", func(t *testing.T) {
		user := newRecoveryMatrixUser(t, "matrix-redemption", 9)
		plan := &model.SubscriptionPlan{Title: "Frozen redemption plan", PriceAmount: 1, Currency: "USD", DurationUnit: model.SubscriptionDurationDay, DurationValue: 3, Enabled: true, TotalAmount: 4321, MaxPurchasePerUser: 1, AllowWalletOverflow: common.GetPointer(false)}
		require.NoError(t, model.DB.Create(plan).Error)
		model.InvalidateSubscriptionPlanCache(plan.Id)
		snapshotPlan := *plan
		snapshotPlan.Id = 0
		snapshot, err := common.Marshal(snapshotPlan)
		require.NoError(t, err)
		code := &model.Redemption{Key: "recovery-snapshot-subscription", Name: "Frozen snapshot", RewardType: model.RedemptionRewardSubscription, SubscriptionPlanId: plan.Id, SubscriptionSnapshot: string(snapshot), Status: common.RedemptionCodeStatusEnabled}
		require.NoError(t, code.Insert())
		require.NoError(t, model.DB.Model(plan).Updates(map[string]any{"total_amount": 9999, "allow_wallet_overflow": true}).Error)
		model.InvalidateSubscriptionPlanCache(plan.Id)
		outcome, err := model.Redeem(code.Key, user.Id)
		require.NoError(t, err)
		assert.Equal(t, plan.Id, outcome.SubscriptionPlanId)
		var sub model.UserSubscription
		require.NoError(t, model.DB.First(&sub, outcome.SubscriptionId).Error)
		assert.EqualValues(t, 4321, sub.AmountTotal)
		assert.False(t, sub.AllowWalletOverflow)
		assert.Equal(t, "redemption", sub.Source)
		_, err = model.Redeem(code.Key, user.Id)
		require.ErrorIs(t, err, model.ErrRedeemFailed)
		var count int64
		require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("user_id = ?", user.Id).Count(&count).Error)
		assert.EqualValues(t, 1, count)
		stored, err := model.GetUserById(user.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 9, stored.Quota)
	})

	t.Run("public_pool_reward_retry", func(t *testing.T) {
		user := newRecoveryMatrixUser(t, "matrix-public", common.MaxQuota-50)
		reward, err := service.NormalizeRewardSpec(dto.RewardSpec{Type: string(model.RewardTypeQuota), Amount: "75", Unit: "quota"})
		require.NoError(t, err)
		raw, err := model.EncodeRewardSnapshot(reward)
		require.NoError(t, err)
		site := &model.PublicPoolSite{Name: "Recovery public site", URL: "https://example.com", RewardSnapshotJSON: raw}
		require.NoError(t, model.CreatePublicPoolSite(site))
		contribution := &model.PublicPoolContribution{UserId: user.Id, SiteId: site.Id, Description: "Registration completed"}
		require.NoError(t, model.CreatePublicPoolContribution(contribution))
		reward.Quota = 900
		site.RewardSnapshotJSON, err = model.EncodeRewardSnapshot(reward)
		require.NoError(t, err)
		require.NoError(t, model.UpdatePublicPoolSite(site))
		reviewed, err := model.ReviewPublicPoolContribution(contribution.Id, 1, model.PublicPoolContributionStatusApproved, "verified")
		require.NoError(t, err)
		assert.Equal(t, model.RewardGrantStatusFailed, reviewed.RewardStatus)
		require.NotZero(t, reviewed.RewardGrantId)
		stored, err := model.GetUserById(user.Id, true)
		require.NoError(t, err)
		assert.Equal(t, common.MaxQuota-50, stored.Quota, "a failed reward must leave the wallet unchanged")
		require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", 10).Error)
		for range 2 {
			retried, err := model.RetryPublicPoolContributionReward(contribution.Id)
			require.NoError(t, err)
			assert.Equal(t, reviewed.RewardGrantId, retried.RewardGrantId)
			assert.Equal(t, model.RewardGrantStatusSucceeded, retried.RewardStatus)
		}
		stored, err = model.GetUserById(user.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 85, stored.Quota, "retry must use the contribution snapshot and credit once")
		items, err := model.ListPublicPoolContributionsForUser(user.Id)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, model.RewardGrantStatusSucceeded, items[0].RewardStatus)
		_, err = model.ReviewPublicPoolContribution(contribution.Id, 1, model.PublicPoolContributionStatusApproved, "repeat")
		require.Error(t, err)
		var grant model.RewardGrant
		require.NoError(t, model.DB.First(&grant, reviewed.RewardGrantId).Error)
		assert.Equal(t, 75, grant.Quota)
		assert.Equal(t, 1, grant.RetryCount)
		grant.Id = 0
		require.Error(t, model.DB.Create(&grant).Error, "the reward source key must remain unique")
	})

	t.Run("referral_reward_retry_and_limit", func(t *testing.T) {
		inviter := newRecoveryMatrixUser(t, "matrix-inviter", common.MaxQuota-10)
		invitee := newRecoveryMatrixUser(t, "matrix-invitee", 3)
		rawInviter, err := model.EncodeRewardSnapshot(model.RewardSnapshot{Type: model.RewardTypeQuota, Quota: 50})
		require.NoError(t, err)
		rawInvitee, err := model.EncodeRewardSnapshot(model.RewardSnapshot{Type: model.RewardTypeQuota, Quota: 25})
		require.NoError(t, err)
		now := common.GetTimestamp()
		campaign := &model.ReferralCampaign{Title: "Recovery referral", Enabled: true, StartTime: now - 60, EndTime: now + 3600, MaxRewardsPerInviter: 1, TotalRewardLimit: 1, RewardSnapshotJSON: rawInviter, InviteeRewardSnapshotJSON: rawInvitee}
		require.NoError(t, model.CreateReferralCampaign(campaign))
		for range 2 {
			used, err := model.HandleInvitationRegistration(inviter.Id, invitee.Id)
			require.NoError(t, err)
			assert.True(t, used)
		}
		registered, err := model.GetUserById(inviter.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 1, registered.AffCount)
		var event model.ReferralCampaignEvent
		require.NoError(t, model.DB.Where("invitee_user_id = ?", invitee.Id).First(&event).Error)
		usage := model.RecordConsumeLogParams{Quota: 20, Group: "default", ActivateReferral: true, BillingSettled: true, Other: model.NewLogOther()}
		for _, group := range []string{"public_pool", "default"} {
			ineligible := usage
			ineligible.Group = group
			ineligible.BillingSettled = group == "public_pool"
			require.NoError(t, model.ActivateReferralCampaignForUser(invitee.Id, "ineligible", group, ineligible))
		}
		require.NoError(t, model.DB.First(&event, event.Id).Error)
		assert.Equal(t, model.ReferralEventPending, event.Status)
		for _, requestID := range []string{"first-qualified", "later-qualified"} {
			require.NoError(t, model.ActivateReferralCampaignForUser(invitee.Id, requestID, "default", usage))
		}
		require.NoError(t, model.DB.First(&event, event.Id).Error)
		assert.Equal(t, model.ReferralEventPendingReview, event.Status)
		assert.Equal(t, "first-qualified", event.QualifiedRequestId)
		assert.Equal(t, 20, event.QualifiedQuota)
		failed, err := model.ReviewReferralCampaignEvent(event.Id, 1, "approve", "verified consumption")
		require.Error(t, err)
		require.NotNil(t, failed)
		assert.Equal(t, model.ReferralEventRewardFailed, failed.Status)
		assert.Equal(t, "approved", failed.ReviewDecision)
		storedInvitee, err := model.GetUserById(invitee.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 28, storedInvitee.Quota, "the successful side must not be replayed when the other side fails")
		require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", inviter.Id).Update("quota", 7).Error)
		retried, err := model.RetryReferralCampaignReward(event.Id)
		require.NoError(t, err)
		assert.Equal(t, model.ReferralEventRewarded, retried.Status)
		_, err = model.ReviewReferralCampaignEvent(event.Id, 1, "approve", "repeat approval")
		require.NoError(t, err)
		storedInviter, err := model.GetUserById(inviter.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 57, storedInviter.Quota)
		storedInvitee, err = model.GetUserById(invitee.Id, true)
		require.NoError(t, err)
		assert.Equal(t, 28, storedInvitee.Quota)
		var grants int64
		require.NoError(t, model.DB.Model(&model.RewardGrant{}).Where("source_type = ? AND source_id = ?", model.RewardSourceReferralCampaign, event.Id).Count(&grants).Error)
		assert.EqualValues(t, 2, grants)
		secondInvitee := newRecoveryMatrixUser(t, "matrix-invitee-two", 0)
		_, err = model.HandleInvitationRegistration(inviter.Id, secondInvitee.Id)
		require.NoError(t, err)
		require.NoError(t, model.ActivateReferralCampaignForUser(secondInvitee.Id, "limit-qualified", "default", usage))
		var secondEvent model.ReferralCampaignEvent
		require.NoError(t, model.DB.Where("invitee_user_id = ?", secondInvitee.Id).First(&secondEvent).Error)
		_, err = model.ReviewReferralCampaignEvent(secondEvent.Id, 1, "approve", "over limit")
		require.ErrorContains(t, err, "limit reached")
		require.NoError(t, model.DB.First(&secondEvent, secondEvent.Id).Error)
		assert.Empty(t, secondEvent.ReviewDecision)
		assert.Equal(t, model.ReferralEventPendingReview, secondEvent.Status)
	})

	t.Run("task_referral_final_charge", func(t *testing.T) {
		require.NoError(t, model.DB.Model(&model.ReferralCampaign{}).Where("enabled = ?", true).Update("enabled", false).Error)
		inviter := newRecoveryMatrixUser(t, "matrix-task-inviter", 10)
		raw, err := model.EncodeRewardSnapshot(model.RewardSnapshot{Type: model.RewardTypeQuota, Quota: 25})
		require.NoError(t, err)
		now := common.GetTimestamp()
		campaign := &model.ReferralCampaign{Title: "Final task charge", Enabled: true, StartTime: now - 60, EndTime: now + 3600, RewardSnapshotJSON: raw}
		require.NoError(t, model.CreateReferralCampaign(campaign))
		previousLogs := common.LogConsumeEnabled
		t.Cleanup(func() { common.LogConsumeEnabled = previousLogs })
		for _, tc := range []struct {
			name   string
			status model.TaskStatus
			final  int
			logs   bool
		}{
			{"failed", model.TaskStatusFailure, 0, true},
			{"lower", model.TaskStatusSuccess, 20, true},
			{"no-logs", model.TaskStatusSuccess, 50, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				common.LogConsumeEnabled = tc.logs
				invitee := newRecoveryMatrixUser(t, "matrix-task-"+tc.name, 950)
				used, err := model.HandleInvitationRegistration(inviter.Id, invitee.Id)
				require.NoError(t, err)
				require.True(t, used)
				var event model.ReferralCampaignEvent
				require.NoError(t, model.DB.Where("invitee_user_id = ?", invitee.Id).First(&event).Error)
				resource := 50
				task := &model.Task{TaskID: "matrix-final-" + tc.name, UserId: invitee.Id, Quota: 50, Group: "default", PrivateData: model.TaskPrivateData{BillingSource: service.BillingSourceWallet, RequestId: "matrix-final-" + tc.name, BillingContext: &model.TaskBillingContext{GroupRatio: 1, ResourceQuota: &resource}}}
				require.NoError(t, model.DB.Create(task).Error)
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
				c.Set(common.RequestIdKey, task.PrivateData.RequestId)
				info := &relaycommon.RelayInfo{UserId: invitee.Id, OriginModelName: "test-model", UsingGroup: "default", ChannelMeta: &relaycommon.ChannelMeta{}, TaskRelayInfo: &relaycommon.TaskRelayInfo{Action: "generate"}, PriceData: types.PriceData{Quota: 50, ResourceQuota: 50}, BillingSettlementSucceeded: true}
				service.LogTaskConsumption(c, info, task)
				require.NoError(t, model.DB.First(&event, event.Id).Error)
				assert.Equal(t, model.ReferralEventPending, event.Status)
				assert.Zero(t, event.QualifiedQuota)
				task.Status = tc.status
				require.NoError(t, model.DB.Model(task).Update("status", tc.status).Error)
				if tc.status == model.TaskStatusFailure {
					require.True(t, service.RefundTaskQuota(c, task, "failed task"))
				} else {
					for range 2 {
						service.RecalculateTaskQuotaWithResource(c, task, tc.final, &tc.final, "final task charge")
					}
				}
				require.NoError(t, model.DB.First(&event, event.Id).Error)
				if tc.status == model.TaskStatusSuccess {
					assert.Equal(t, model.ReferralEventPendingReview, event.Status)
					assert.Equal(t, tc.final, event.QualifiedQuota)
				} else {
					assert.Equal(t, model.ReferralEventPending, event.Status)
					_, err := model.ReviewReferralCampaignEvent(event.Id, inviter.Id, "approve", "unpaid task")
					require.ErrorContains(t, err, "actual quota consumption")
				}
				require.NoError(t, model.DB.First(invitee, invitee.Id).Error)
				assert.Equal(t, 1000-tc.final, invitee.Quota)
			})
		}
		require.NoError(t, model.DB.First(inviter, inviter.Id).Error)
		assert.Equal(t, 10, inviter.Quota, "qualification still requires administrator approval")
	})

	t.Run("lottery_payout_guards", func(t *testing.T) {
		for _, mode := range []model.LotteryFulfillmentMode{model.LotteryFulfillmentSelfClaim, model.LotteryFulfillmentRedemptionCode} {
			t.Run(string(mode), func(t *testing.T) {
				user := newRecoveryMatrixUser(t, "matrix-lottery-"+string(mode), 0)
				now := common.GetTimestamp()
				plan := &model.LotteryPlan{Title: "Payout guards", Status: model.LotteryPlanStatusOpen, EligibilityMode: model.LotteryEligibilityAll, MaxParticipants: 2, RegistrationStartTime: now - 60, DrawTime: now + 3600}
				require.NoError(t, model.CreateLotteryPlan(plan, nil, nil, []*model.LotteryPrize{{Name: "Quota", Quantity: 1, RewardType: model.LotteryRewardQuota, Quota: 123, FulfillmentMode: mode, ClaimExpireSeconds: 3600}}))
				require.NoError(t, model.JoinLotteryPlan(plan.Id, user.Id))
				_, err := model.DrawLotteryPlan(plan.Id, model.LotteryDrawTriggerManual, "matrix")
				require.NoError(t, err)
				var result model.LotteryResult
				require.NoError(t, model.DB.Where("plan_id = ? AND user_id = ?", plan.Id, user.Id).First(&result).Error)
				if mode == model.LotteryFulfillmentSelfClaim {
					require.NoError(t, model.DB.Model(user).Update("quota", common.MaxWalletQuota-100).Error)
					require.ErrorIs(t, model.ClaimLotteryResult(result.Id, user.Id), model.ErrWalletQuotaLimitExceeded)
					require.NoError(t, model.DB.First(&result, result.Id).Error)
					assert.Equal(t, "pending", result.FulfillmentStatus)
					require.NoError(t, model.DB.First(user, user.Id).Error)
					assert.Equal(t, common.MaxWalletQuota-100, user.Quota)
					require.NoError(t, model.DB.Model(user).Update("quota", 0).Error)
					require.NoError(t, model.ClaimLotteryResult(result.Id, user.Id))
					require.Error(t, model.ClaimLotteryResult(result.Id, user.Id))
				} else {
					require.NotEmpty(t, result.RedemptionCode)
					require.Error(t, model.ClaimLotteryResult(result.Id, user.Id))
					require.Error(t, model.FulfillLotteryResult(result.Id))
					_, err = model.Redeem(result.RedemptionCode, user.Id)
					require.NoError(t, err)
					_, err = model.Redeem(result.RedemptionCode, user.Id)
					require.Error(t, err)
				}
				require.NoError(t, model.DB.First(user, user.Id).Error)
				assert.Equal(t, 123, user.Quota)
			})
		}
	})

	t.Run("channel_errors_and_usage", func(t *testing.T) {
		channel := &model.Channel{Name: "recovery-error-channel", Key: "fixture-key", Status: common.ChannelStatusEnabled}
		require.NoError(t, model.DB.Create(channel).Error)
		input := model.ChannelErrorRecordInput{ChannelID: channel.Id, RequestPath: "/v1/responses", RequestModel: "matrix-model", UpstreamStatusCode: 400, FinalStatusCode: 400, ErrorType: "upstream_error", ErrorCode: "unsupported_model", SampleMessage: "Unsupported model", RequestID: "first-error"}
		first, err := model.UpsertChannelErrorRecord(input)
		require.NoError(t, err)
		_, err = model.UpdateChannelErrorDisplayRule(channel.Id, first.ID, relaydto.ChannelErrorDisplayRule{Mode: relaydto.ChannelErrorDisplayModeCustom, StatusCode: 503, Message: "Temporarily unavailable"})
		require.NoError(t, err)
		input.RequestID, input.FinalStatusCode, input.MultiKeyIndex = "later-error", 503, 2
		second, err := model.UpsertChannelErrorRecord(input)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.EqualValues(t, 2, second.OccurrenceCount)
		assert.Equal(t, first.FirstSeenTime, second.FirstSeenTime)
		assert.Equal(t, "later-error", second.LastRequestID)
		assert.Equal(t, 2, second.LastMultiKeyIndex)
		assert.Equal(t, 503, second.DisplayStatusCode)
		assert.Equal(t, "Temporarily unavailable", second.DisplayMessage)
		_, err = model.UpdateChannelErrorDisplayRule(channel.Id+1, first.ID, relaydto.ChannelErrorDisplayRule{Mode: relaydto.ChannelErrorDisplayModeInherit})
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		items, total, err := model.GetChannelErrorRecords(model.ChannelErrorRecordQuery{ChannelID: channel.Id, StatusCode: 400, RequestModel: "matrix-model", Keyword: "Unsupported", Limit: 20})
		require.NoError(t, err)
		assert.EqualValues(t, 1, total)
		require.Len(t, items, 1)
		verifyRecoveredChannelUsage(t, channel.Id)
	})

	t.Run("task_resource_settlement", func(t *testing.T) {
		user := newRecoveryMatrixUser(t, "matrix-task-resources", 950)
		channel := &model.Channel{Name: "Task resource matrix", Key: "fixture-key"}
		require.NoError(t, model.DB.Create(channel).Error)
		resource := 100
		task := &model.Task{
			TaskID: "matrix-task-resources", UserId: user.Id, ChannelId: channel.Id, Quota: 50, Group: "default",
			PrivateData: model.TaskPrivateData{BillingSource: service.BillingSourceWallet, RequestId: "matrix-task-resources", BillingContext: &model.TaskBillingContext{GroupRatio: 0.5, ResourceQuota: &resource}},
		}
		require.NoError(t, model.DB.Create(task).Error)
		now := time.Now().Unix()
		require.NoError(t, model.LOG_DB.Create(&model.Log{UserId: user.Id, ChannelId: channel.Id, CreatedAt: now, Type: model.LogTypeConsume, Quota: 50, ResourceQuota: &resource, RequestId: task.PrivateData.RequestId}).Error)
		finalResource := 160
		service.RecalculateTaskQuotaWithResource(context.Background(), task, 80, &finalResource, "matrix settlement")
		service.RecalculateTaskQuotaWithResource(context.Background(), task, 80, &finalResource, "matrix duplicate settlement")
		var persisted model.Task
		require.NoError(t, model.DB.First(&persisted, task.ID).Error)
		require.NotNil(t, persisted.PrivateData.BillingContext)
		require.NotNil(t, persisted.PrivateData.BillingContext.ResourceQuota)
		assert.Equal(t, finalResource, *persisted.PrivateData.BillingContext.ResourceQuota)
		summary, err := model.GetChannelUsageSummary(channel.Id, "day", now-10, now+20, 30)
		require.NoError(t, err)
		assert.EqualValues(t, 160, summary.ResourceQuota)
		assert.EqualValues(t, 80, summary.ChargedQuota)
		assert.EqualValues(t, 1, summary.SuccessCount)
		assert.True(t, summary.ResourceCoverageComplete)
		require.True(t, service.RefundTaskQuota(context.Background(), &persisted, "matrix refund"))
		require.True(t, service.RefundTaskQuota(context.Background(), &persisted, "matrix duplicate refund"))
		summary, err = model.GetChannelUsageSummary(channel.Id, "day", now-10, now+20, 30)
		require.NoError(t, err)
		assert.Zero(t, summary.ResourceQuota)
		assert.Zero(t, summary.ChargedQuota)
		require.NoError(t, model.DB.First(user, user.Id).Error)
		assert.Equal(t, 1000, user.Quota)
	})

	t.Run("restart_retains_recovered_data", func(t *testing.T) {
		var rewardsBefore []model.RewardGrant
		var subscriptionsBefore []model.UserSubscription
		var errorsBefore []model.ChannelErrorRecord
		require.NoError(t, model.DB.Order("id").Find(&rewardsBefore).Error)
		require.NoError(t, model.DB.Order("id").Find(&subscriptionsBefore).Error)
		require.NoError(t, model.DB.Order("id").Find(&errorsBefore).Error)
		for range 2 {
			require.NoError(t, model.InitDB())
			connection, err := model.DB.DB()
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, connection.Close()) })
			require.NoError(t, model.InitLogDB())
		}
		var rewardsAfter []model.RewardGrant
		var subscriptionsAfter []model.UserSubscription
		var errorsAfter []model.ChannelErrorRecord
		require.NoError(t, model.DB.Order("id").Find(&rewardsAfter).Error)
		require.NoError(t, model.DB.Order("id").Find(&subscriptionsAfter).Error)
		require.NoError(t, model.DB.Order("id").Find(&errorsAfter).Error)
		assert.Equal(t, rewardsBefore, rewardsAfter)
		assert.Equal(t, subscriptionsBefore, subscriptionsAfter)
		assert.Equal(t, errorsBefore, errorsAfter)
		var option model.Option
		require.NoError(t, model.DB.Where(&model.Option{Key: manualWechatQRCodeOptionKey}).First(&option).Error)
		assert.Equal(t, recoveryMatrixQRCode(), option.Value)
	})
}

func verifyRecoveredChannelUsage(t *testing.T, channelID int) {
	t.Helper()
	start := int64(1_700_000_000)
	resource := 70
	freeResource := 50
	rows := []model.Log{
		{ChannelId: channelID, CreatedAt: start, Type: model.LogTypeConsume, Quota: 100, ResourceQuota: &resource, PromptTokens: 10, CompletionTokens: 5, RequestId: "usage-paid"},
		{ChannelId: channelID, CreatedAt: start + 1, Type: model.LogTypeConsume, Quota: 40, ResourceQuota: &resource, PromptTokens: 4, CompletionTokens: 2, RequestId: "usage-paid"},
		{ChannelId: channelID, CreatedAt: start + 2, Type: model.LogTypeRefund, Quota: 30, RequestId: "usage-paid"},
		{ChannelId: channelID, CreatedAt: start + 3, Type: model.LogTypeError, RequestId: "usage-error"},
		{ChannelId: channelID, CreatedAt: start + 4, Type: model.LogTypeError, RequestId: "usage-error"},
		{ChannelId: channelID, CreatedAt: start + 60, Type: model.LogTypeConsume, Quota: 0, ResourceQuota: &freeResource, PromptTokens: 4, CompletionTokens: 2, RequestId: "usage-public", Group: "public_pool"},
		{ChannelId: channelID, CreatedAt: start + 61, Type: model.LogTypeConsume, Quota: 20, PromptTokens: 7, CompletionTokens: 1},
		{ChannelId: channelID, CreatedAt: start + 120, Type: model.LogTypeConsume, Quota: 999, ResourceQuota: &resource, RequestId: "usage-excluded"},
	}
	require.NoError(t, model.LOG_DB.Create(&rows).Error)
	summary, err := model.GetChannelUsageSummary(channelID, "day", start, start+120, 60)
	require.NoError(t, err)
	assert.EqualValues(t, 190, summary.ResourceQuota)
	assert.EqualValues(t, 130, summary.ChargedQuota)
	assert.EqualValues(t, 25, summary.PromptTokens)
	assert.EqualValues(t, 10, summary.CompletionTokens)
	assert.EqualValues(t, 3, summary.SuccessCount)
	assert.EqualValues(t, 1, summary.FailureCount)
	assert.EqualValues(t, 4, summary.RequestCount)
	assert.EqualValues(t, 2, summary.ResourceCoveredRequests)
	assert.False(t, summary.ResourceCoverageComplete)
	assert.Equal(t, start, summary.ResourceCoverageStart)
	assert.Equal(t, start, summary.EarliestAvailableTime)
	require.Len(t, summary.Trend, 2)
	assert.Equal(t, model.ChannelUsageTrendPoint{StartTime: start, EndTime: start + 60, ResourceQuota: 140, ChargedQuota: 110, PromptTokens: 14, CompletionTokens: 7, SuccessCount: 1, FailureCount: 1, RequestCount: 2, ResourceCoveredRequests: 1}, summary.Trend[0])
	assert.Equal(t, model.ChannelUsageTrendPoint{StartTime: start + 60, EndTime: start + 120, ResourceQuota: 50, ChargedQuota: 20, PromptTokens: 11, CompletionTokens: 3, SuccessCount: 2, RequestCount: 2, ResourceCoveredRequests: 1}, summary.Trend[1])
}

func TestRecoveredChannelUsageSeparateDatabaseMatrix(t *testing.T) {
	_, _ = setupAccessTokenAudit(t)
	previousSQLite := common.SQLitePath
	t.Cleanup(func() { common.SQLitePath = previousSQLite })
	for _, tc := range []struct{ name, env string }{{"sqlite", ""}, {"mysql", "AUDIT_MYSQL_DSN"}, {"postgres", "AUDIT_POSTGRES_DSN"}} {
		dsn := os.Getenv(tc.env)
		if tc.env != "" && dsn == "" {
			t.Run(tc.name, func(t *testing.T) { t.Skip(tc.env + " is not configured") })
			continue
		}
		for _, upgrade := range []bool{false, true} {
			name := "fresh"
			if upgrade {
				name = "upgrade"
			}
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				db, isolatedDSN := newAuditTestDatabase(t, tc.name, dsn)
				if upgrade {
					require.NoError(t, db.AutoMigrate(&releasedAuditLog{}))
					require.NoError(t, db.Create(&releasedAuditLog{ChannelId: 910000, CreatedAt: 100, Type: model.LogTypeConsume, Quota: 20, RequestId: "recovery-legacy-log"}).Error)
				}
				if tc.name == "sqlite" {
					common.SQLitePath = isolatedDSN
					t.Setenv("LOG_SQL_DSN", "local")
				} else {
					t.Setenv("LOG_SQL_DSN", isolatedDSN)
				}
				for range 2 {
					require.NoError(t, model.InitLogDB())
					connection, err := model.LOG_DB.DB()
					require.NoError(t, err)
					t.Cleanup(func() { assert.NoError(t, connection.Close()) })
				}
				assert.NotSame(t, model.DB, model.LOG_DB)
				if upgrade {
					legacy, err := model.GetChannelUsageSummary(910000, "day", 0, 200, 200)
					require.NoError(t, err)
					assert.EqualValues(t, 20, legacy.ChargedQuota)
					assert.Zero(t, legacy.ResourceQuota)
					assert.False(t, legacy.ResourceCoverageComplete)
				}
				verifyRecoveredChannelUsage(t, 910001)
			})
		}
	}
}
