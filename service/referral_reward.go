package service

import (
	"errors"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
)

var referralUSDPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,12})?$`)

// NormalizeReferralUSDReward returns an empty snapshot for a disabled side.
// Bound decimal syntax before parsing: a huge exponent must not allocate a huge
// coefficient while rounding. Existing non-referral reward clients are unchanged.
func NormalizeReferralUSDReward(rawAmount string) (string, error) {
	amount := strings.TrimSpace(rawAmount)
	if !referralUSDPattern.MatchString(amount) {
		return "", errors.New("referral reward must be a non-negative USD amount with at most 12 decimal places")
	}
	value, err := decimal.NewFromString(amount)
	if err != nil {
		return "", err
	}
	if value.IsZero() {
		return "", nil
	}
	snapshot, err := NormalizeRewardSpec(dto.RewardSpec{Type: string(model.RewardTypeQuota), Amount: amount, Unit: "usd"})
	if err != nil {
		return "", err
	}
	return model.EncodeRewardSnapshot(snapshot)
}
