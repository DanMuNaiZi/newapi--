package model

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManualWechatOrderIndexesAreMigrated(t *testing.T) {
	assert.True(t, DB.Migrator().HasIndex(&TopUp{}, "idx_topups_provider_status_created"))
	assert.True(t, DB.Migrator().HasIndex(&SubscriptionOrder{}, "idx_subscription_orders_provider_status_created"))
}

func TestCreateOrReuseManualWechatTopUpReusesRecentMatchingOrder(t *testing.T) {
	truncateTables(t)
	user := insertUserForPaymentGuardTest(t, 701, 0)

	first, reused, err := CreateOrReuseManualWechatTopUp(user.Id, 5, 36.5, 1_000)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.Equal(t, PaymentProviderManualWechat, first.PaymentProvider)
	assert.Equal(t, common.TopUpStatusPending, first.Status)
	assert.True(t, strings.HasPrefix(first.TradeNo, "MWX-T-"))
	assert.Len(t, first.TradeNo, len("MWX-T-")+12)
	assert.Equal(t, strings.ToUpper(first.TradeNo), first.TradeNo)

	second, reused, err := CreateOrReuseManualWechatTopUp(user.Id, 5, 36.5, 1_030)
	require.NoError(t, err)
	assert.True(t, reused)
	assert.Equal(t, first.TradeNo, second.TradeNo)

	third, reused, err := CreateOrReuseManualWechatTopUp(user.Id, 5, 36.5, 1_061)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.NotEqual(t, first.TradeNo, third.TradeNo)
}

func TestCreateOrReuseManualWechatTopUpDoesNotReuseDifferentPayableAmount(t *testing.T) {
	truncateTables(t)
	user := insertUserForPaymentGuardTest(t, 709, 0)

	first, reused, err := CreateOrReuseManualWechatTopUp(user.Id, 2, 14.60, 1_000)
	require.NoError(t, err)
	assert.False(t, reused)

	second, reused, err := CreateOrReuseManualWechatTopUp(user.Id, 2, 14.61, 1_030)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.NotEqual(t, first.TradeNo, second.TradeNo)
}

func TestCreateOrReuseManualWechatSubscriptionOrderScopesPlan(t *testing.T) {
	truncateTables(t)
	user := insertUserForPaymentGuardTest(t, 702, 0)
	firstPlan := insertSubscriptionPlanForPaymentGuardTest(t, 801)
	secondPlan := insertSubscriptionPlanForPaymentGuardTest(t, 802)

	first, reused, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, firstPlan.Id, firstPlan.PriceAmount, 2_000)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.True(t, strings.HasPrefix(first.TradeNo, "MWX-S-"))

	samePlan, reused, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, firstPlan.Id, firstPlan.PriceAmount, 2_010)
	require.NoError(t, err)
	assert.True(t, reused)
	assert.Equal(t, first.TradeNo, samePlan.TradeNo)

	otherPlan, reused, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, secondPlan.Id, secondPlan.PriceAmount, 2_010)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.NotEqual(t, first.TradeNo, otherPlan.TradeNo)
}

func TestCompleteManualWechatTopUpCreditsExactlyOnceAndGuardsProvider(t *testing.T) {
	truncateTables(t)
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 703, 0)
	manualOrder, _, err := CreateOrReuseManualWechatTopUp(user.Id, 2, 14.6, 3_000)
	require.NoError(t, err)

	require.NoError(t, CompleteManualWechatTopUp(manualOrder.TradeNo, "127.0.0.1"))
	require.NoError(t, CompleteManualWechatTopUp(manualOrder.TradeNo, "127.0.0.1"))
	assert.Equal(t, 1_000_000, getUserQuotaForPaymentGuardTest(t, user.Id))
	assert.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, manualOrder.TradeNo))

	insertTopUpForPaymentGuardTest(t, "OTHER-PROVIDER", user.Id, PaymentProviderEpay)
	err = CompleteManualWechatTopUp("OTHER-PROVIDER", "127.0.0.1")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "OTHER-PROVIDER"))
}

func TestCompleteManualWechatTopUpLeavesOrderPendingOnQuotaOverflow(t *testing.T) {
	truncateTables(t)
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = float64(common.MaxWalletQuota + 1)
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 707, 3)
	order, _, err := CreateOrReuseManualWechatTopUp(user.Id, 1, 7.3, 4_000)
	require.NoError(t, err)

	err = CompleteManualWechatTopUp(order.TradeNo, "127.0.0.1")
	require.Error(t, err)
	assert.Equal(t, 3, getUserQuotaForPaymentGuardTest(t, user.Id))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, order.TradeNo))
}

func TestListManualWechatTopUpsFiltersAndIncludesUsername(t *testing.T) {
	truncateTables(t)
	user := &User{Id: 704, Username: "manual_list_user", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(user).Error)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&TopUp{
		UserId:          user.Id,
		Amount:          5,
		Money:           36.5,
		TradeNo:         "MWX-T-LIST00000001",
		PaymentMethod:   PaymentMethodManualWechat,
		PaymentProvider: PaymentProviderManualWechat,
		Status:          common.TopUpStatusPending,
		CreateTime:      now,
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          user.Id,
		Amount:          5,
		Money:           36.5,
		TradeNo:         "EPAY-LIST-0001",
		PaymentMethod:   PaymentProviderEpay,
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
		CreateTime:      now,
	}).Error)

	items, total, err := GetAdminTopUps(&common.PageInfo{Page: 1, PageSize: 20}, AdminTopUpFilter{
		PaymentProvider: PaymentProviderManualWechat,
		Status:          common.TopUpStatusPending,
		Keyword:         "manual_list",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "MWX-T-LIST00000001", items[0].TradeNo)
	assert.Equal(t, user.Username, items[0].Username)
}

func TestListManualWechatSubscriptionOrdersFiltersAndIncludesLabels(t *testing.T) {
	truncateTables(t)
	user := &User{Id: 705, Username: "manual_subscription_list", Status: common.UserStatusEnabled}
	plan := &SubscriptionPlan{
		Id: 805, Title: "Manual Pro", PriceAmount: 29.9, Currency: "CNY",
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1, Enabled: true, TotalAmount: 1_000,
	}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(plan).Error)
	order, _, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, time.Now().Unix())
	require.NoError(t, err)

	items, total, err := GetAdminSubscriptionOrders(&common.PageInfo{Page: 1, PageSize: 20}, AdminSubscriptionOrderFilter{
		PaymentProvider: PaymentProviderManualWechat,
		Status:          common.TopUpStatusPending,
		UserId:          user.Id,
		PlanId:          plan.Id,
		Keyword:         "Manual Pro",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, order.TradeNo, items[0].TradeNo)
	assert.Equal(t, user.Username, items[0].Username)
	assert.Equal(t, plan.Title, items[0].PlanTitle)
}

func TestManualWechatSubscriptionCompleteAndExpire(t *testing.T) {
	truncateTables(t)
	user := &User{Id: 706, Username: "manual_subscription_complete", Status: common.UserStatusEnabled}
	plan := &SubscriptionPlan{
		Id: 806, Title: "Complete Pro", PriceAmount: 19.9, Currency: "CNY",
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1, Enabled: true, TotalAmount: 1_000,
	}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(plan).Error)

	completed, _, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, 5_000)
	require.NoError(t, err)
	require.NoError(t, CompleteSubscriptionOrder(completed.TradeNo, "", PaymentProviderManualWechat, PaymentMethodManualWechat))
	require.NoError(t, CompleteSubscriptionOrder(completed.TradeNo, "", PaymentProviderManualWechat, PaymentMethodManualWechat))
	assert.Equal(t, int64(1), countUserSubscriptionsForPaymentGuardTest(t, user.Id))
	var completedTopUp TopUp
	require.NoError(t, DB.Where("trade_no = ?", completed.TradeNo).First(&completedTopUp).Error)
	assert.Equal(t, PaymentMethodManualWechat, completedTopUp.PaymentMethod)
	assert.Equal(t, PaymentProviderManualWechat, completedTopUp.PaymentProvider)

	expired, _, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, 5_061)
	require.NoError(t, err)
	require.NoError(t, ExpireSubscriptionOrder(expired.TradeNo, PaymentProviderManualWechat))
	err = CompleteSubscriptionOrder(expired.TradeNo, "", PaymentProviderManualWechat, PaymentMethodManualWechat)
	require.ErrorIs(t, err, ErrSubscriptionOrderStatusInvalid)
	assert.Equal(t, int64(1), countUserSubscriptionsForPaymentGuardTest(t, user.Id))
}

func TestManualWechatSubscriptionRechecksPurchaseLimitWhenCompleting(t *testing.T) {
	truncateTables(t)
	user := &User{Id: 708, Username: "manual_subscription_limit", Status: common.UserStatusEnabled}
	plan := &SubscriptionPlan{
		Id: 808, Title: "Limited Pro", PriceAmount: 19.9, Currency: "CNY",
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1, Enabled: true,
		TotalAmount: 1_000, MaxPurchasePerUser: 1,
	}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(plan).Error)

	first, _, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, 6_000)
	require.NoError(t, err)
	second, _, err := CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, 6_061)
	require.NoError(t, err)
	require.NoError(t, CompleteSubscriptionOrder(first.TradeNo, "", PaymentProviderManualWechat, PaymentMethodManualWechat))

	err = CompleteSubscriptionOrder(second.TradeNo, "", PaymentProviderManualWechat, PaymentMethodManualWechat)
	require.Error(t, err)
	assert.Equal(t, int64(1), countUserSubscriptionsForPaymentGuardTest(t, user.Id))
	second = GetSubscriptionOrderByTradeNo(second.TradeNo)
	require.NotNil(t, second)
	assert.Equal(t, common.TopUpStatusPending, second.Status)
}
