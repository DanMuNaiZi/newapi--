package controller

import (
	"bytes"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLotteryPrizeRequestParsesStringAndNumberDecimalAmounts(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	previousExchangeRate := operation_setting.USDExchangeRate
	common.QuotaPerUnit = 500_000
	operation_setting.USDExchangeRate = 0
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		operation_setting.USDExchangeRate = previousExchangeRate
	})

	for _, payload := range []string{
		`{"prizes":[{"reward_type":"quota","reward_amount":"0.000249","reward_unit":"usd"}]}`,
		`{"prizes":[{"reward_type":"quota","reward_amount":0.000249,"reward_unit":"usd"}]}`,
	} {
		t.Run(payload, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/lottery/admin/plans", bytes.NewBufferString(payload))
			ctx.Request.Header.Set("Content-Type", "application/json")
			request := lotteryPlanRequest{}
			require.NoError(t, ctx.ShouldBindJSON(&request))
			require.Len(t, request.Prizes, 1)
			prize, audit, err := normalizeLotteryPrizeRequest(request.Prizes[0])
			require.NoError(t, err)
			assert.Equal(t, 125, prize.Quota)
			assert.Equal(t, "0.000249", audit.InputAmount)
			assert.Equal(t, "0", audit.USDExchangeRate)
		})
	}
}

func lotteryDecimal(value string) *decimal.Decimal {
	parsed := decimal.RequireFromString(value)
	return &parsed
}

func lotteryInt(value int) *int {
	return &value
}

func TestNormalizeLotteryPrizeRequestConvertsSupportedRewardUnits(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	previousExchangeRate := operation_setting.USDExchangeRate
	common.QuotaPerUnit = 500_000
	operation_setting.USDExchangeRate = 7.3
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		operation_setting.USDExchangeRate = previousExchangeRate
	})

	tests := []struct {
		name   string
		amount string
		unit   lotteryRewardUnit
		quota  int
	}{
		{name: "usd", amount: "10", unit: lotteryRewardUnitUSD, quota: 5_000_000},
		{name: "cny", amount: "73", unit: lotteryRewardUnitCNY, quota: 5_000_000},
		{name: "raw quota", amount: "5000000", unit: lotteryRewardUnitQuota, quota: 5_000_000},
		{name: "round half away from zero", amount: "0.000001", unit: lotteryRewardUnitUSD, quota: 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			prize, audit, err := normalizeLotteryPrizeRequest(lotteryPrizeRequest{
				Name:            "Quota prize",
				Quantity:        1,
				RewardType:      model.LotteryRewardQuota,
				RewardAmount:    lotteryDecimal(testCase.amount),
				RewardUnit:      testCase.unit,
				FulfillmentMode: model.LotteryFulfillmentAuto,
			})

			require.NoError(t, err)
			assert.Equal(t, testCase.quota, prize.Quota)
			assert.Equal(t, testCase.amount, audit.InputAmount)
			assert.Equal(t, string(testCase.unit), audit.InputUnit)
			assert.Equal(t, testCase.quota, audit.Quota)
			assert.Equal(t, "500000", audit.QuotaPerUnit)
			assert.Equal(t, "7.3", audit.USDExchangeRate)
		})
	}
}

func TestNormalizeLotteryPrizeRequestKeepsLegacyQuotaCompatibility(t *testing.T) {
	prize, audit, err := normalizeLotteryPrizeRequest(lotteryPrizeRequest{
		Name:            "Legacy quota",
		Quantity:        1,
		RewardType:      model.LotteryRewardQuota,
		Quota:           lotteryInt(1234),
		FulfillmentMode: model.LotteryFulfillmentSelfClaim,
	})

	require.NoError(t, err)
	assert.Equal(t, 1234, prize.Quota)
	assert.Equal(t, "1234", audit.InputAmount)
	assert.Equal(t, string(lotteryRewardUnitQuota), audit.InputUnit)
}

func TestNormalizeLotteryPrizeRequestRejectsAmbiguousOrUnsafeAmounts(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	previousExchangeRate := operation_setting.USDExchangeRate
	common.QuotaPerUnit = 500_000
	operation_setting.USDExchangeRate = 7.3
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		operation_setting.USDExchangeRate = previousExchangeRate
	})

	tests := []struct {
		name    string
		request lotteryPrizeRequest
	}{
		{
			name:    "new and legacy fields conflict",
			request: lotteryPrizeRequest{RewardType: model.LotteryRewardQuota, Quota: lotteryInt(1), RewardAmount: lotteryDecimal("1"), RewardUnit: lotteryRewardUnitUSD},
		},
		{
			name:    "raw quota must be an integer",
			request: lotteryPrizeRequest{RewardType: model.LotteryRewardQuota, RewardAmount: lotteryDecimal("1.5"), RewardUnit: lotteryRewardUnitQuota},
		},
		{
			name:    "rounded result below one",
			request: lotteryPrizeRequest{RewardType: model.LotteryRewardQuota, RewardAmount: lotteryDecimal("0.0000001"), RewardUnit: lotteryRewardUnitUSD},
		},
		{
			name:    "overflow",
			request: lotteryPrizeRequest{RewardType: model.LotteryRewardQuota, RewardAmount: lotteryDecimal("5000"), RewardUnit: lotteryRewardUnitUSD},
		},
		{
			name:    "missing amount",
			request: lotteryPrizeRequest{RewardType: model.LotteryRewardQuota, RewardUnit: lotteryRewardUnitUSD},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := normalizeLotteryPrizeRequest(testCase.request)
			assert.Error(t, err)
		})
	}
}

func TestNormalizeLotteryPrizeRequestRejectsInvalidCurrencyConfiguration(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	previousExchangeRate := operation_setting.USDExchangeRate
	common.QuotaPerUnit = 500_000
	operation_setting.USDExchangeRate = 0
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		operation_setting.USDExchangeRate = previousExchangeRate
	})

	_, _, err := normalizeLotteryPrizeRequest(lotteryPrizeRequest{
		RewardType:   model.LotteryRewardQuota,
		RewardAmount: lotteryDecimal("73"),
		RewardUnit:   lotteryRewardUnitCNY,
	})

	assert.Error(t, err)
}

func TestNormalizeLotteryPrizeRequestRejectsNonFiniteCurrencyConfigurationWithoutPanicking(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	previousExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		operation_setting.USDExchangeRate = previousExchangeRate
	})

	tests := []struct {
		name            string
		quotaPerUnit    float64
		usdExchangeRate float64
		unit            lotteryRewardUnit
	}{
		{name: "NaN quota per unit", quotaPerUnit: math.NaN(), usdExchangeRate: 7.3, unit: lotteryRewardUnitUSD},
		{name: "infinite exchange rate", quotaPerUnit: 500_000, usdExchangeRate: math.Inf(1), unit: lotteryRewardUnitCNY},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			common.QuotaPerUnit = testCase.quotaPerUnit
			operation_setting.USDExchangeRate = testCase.usdExchangeRate

			assert.NotPanics(t, func() {
				_, _, err := normalizeLotteryPrizeRequest(lotteryPrizeRequest{
					RewardType:   model.LotteryRewardQuota,
					RewardAmount: lotteryDecimal("1"),
					RewardUnit:   testCase.unit,
				})
				assert.Error(t, err)
			})
		})
	}
}
