package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRewardSpecConvertsRuntimeCurrencies(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	previousRate := operation_setting.USDExchangeRate
	common.QuotaPerUnit = 500_000
	operation_setting.USDExchangeRate = 7.3
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		operation_setting.USDExchangeRate = previousRate
	})

	usd, err := NormalizeRewardSpec(dto.RewardSpec{Type: "quota", Amount: "10", Unit: "usd"})
	require.NoError(t, err)
	cny, err := NormalizeRewardSpec(dto.RewardSpec{Type: "quota", Amount: "73", Unit: "cny"})
	require.NoError(t, err)
	raw, err := NormalizeRewardSpec(dto.RewardSpec{Type: "quota", Amount: "5000000", Unit: "quota"})
	require.NoError(t, err)

	assert.Equal(t, 5_000_000, usd.Quota)
	assert.Equal(t, usd.Quota, cny.Quota)
	assert.Equal(t, usd.Quota, raw.Quota)
}

func TestNormalizeRewardSpecRejectsFractionalRawQuota(t *testing.T) {
	_, err := NormalizeRewardSpec(dto.RewardSpec{Type: "quota", Amount: "1.5", Unit: "quota"})
	require.ErrorContains(t, err, "integer")
}
