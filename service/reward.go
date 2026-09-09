package service

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

func NormalizeRewardSpec(spec dto.RewardSpec) (model.RewardSnapshot, error) {
	rewardType := model.RewardType(strings.TrimSpace(spec.Type))
	switch rewardType {
	case model.RewardTypeQuota:
		return normalizeQuotaReward(spec)
	case model.RewardTypeSubscription:
		unit := strings.TrimSpace(spec.Unit)
		if spec.SubscriptionPlanId <= 0 || strings.TrimSpace(spec.Amount) != "" || (unit != "" && unit != "quota") {
			return model.RewardSnapshot{}, errors.New("invalid subscription reward")
		}
		plan, err := model.GetSubscriptionPlanById(spec.SubscriptionPlanId)
		if err != nil {
			return model.RewardSnapshot{}, err
		}
		if !plan.Enabled {
			return model.RewardSnapshot{}, errors.New("subscription reward plan is disabled")
		}
		planSnapshot, err := common.Marshal(plan)
		if err != nil {
			return model.RewardSnapshot{}, err
		}
		return model.RewardSnapshot{
			Type:                     rewardType,
			SubscriptionPlanId:       plan.Id,
			SubscriptionPlanTitle:    plan.Title,
			SubscriptionPlanSnapshot: string(planSnapshot),
		}, nil
	default:
		return model.RewardSnapshot{}, errors.New("invalid reward type")
	}
}

func normalizeQuotaReward(spec dto.RewardSpec) (model.RewardSnapshot, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(spec.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) || spec.SubscriptionPlanId != 0 {
		return model.RewardSnapshot{}, errors.New("invalid quota reward amount")
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return model.RewardSnapshot{}, errors.New("invalid quota per unit configuration")
	}
	if operation_setting.USDExchangeRate <= 0 || math.IsNaN(operation_setting.USDExchangeRate) || math.IsInf(operation_setting.USDExchangeRate, 0) {
		return model.RewardSnapshot{}, errors.New("invalid USD exchange rate configuration")
	}
	unit := strings.TrimSpace(spec.Unit)
	quotaDecimal := decimal.Zero
	switch unit {
	case "usd":
		quotaDecimal = amount.Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	case "cny":
		quotaDecimal = amount.Div(decimal.NewFromFloat(operation_setting.USDExchangeRate)).Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	case "quota":
		if !amount.Equal(amount.Truncate(0)) {
			return model.RewardSnapshot{}, errors.New("raw reward quota must be an integer")
		}
		quotaDecimal = amount
	default:
		return model.RewardSnapshot{}, errors.New("invalid reward unit")
	}
	rounded := quotaDecimal.Round(0)
	if rounded.LessThan(decimal.NewFromInt(1)) || rounded.GreaterThan(decimal.NewFromInt(common.MaxQuota)) {
		return model.RewardSnapshot{}, errors.New("reward quota is outside the supported range")
	}
	return model.RewardSnapshot{
		Type:            model.RewardTypeQuota,
		InputAmount:     amount.String(),
		InputUnit:       unit,
		Quota:           common.QuotaFromDecimal(rounded),
		QuotaPerUnit:    strconv.FormatFloat(common.QuotaPerUnit, 'f', -1, 64),
		USDExchangeRate: strconv.FormatFloat(operation_setting.USDExchangeRate, 'f', -1, 64),
	}, nil
}
