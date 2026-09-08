package service

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const PublicPoolGroup = constant.PublicPoolGroup

type PublicPoolStatus struct {
	Enabled    bool    `json:"enabled"`
	Available  bool    `json:"available"`
	GroupRatio float64 `json:"group_ratio"`
	Reason     string  `json:"reason,omitempty"`
}

func GetPublicPoolStatus() PublicPoolStatus {
	status := PublicPoolStatus{Enabled: operation_setting.GetPublicPoolSetting().Enabled}
	if !status.Enabled {
		status.Reason = "public pool is disabled"
		return status
	}
	ratio, exists := ratio_setting.GetGroupRatioSetting().GroupRatio.Get(PublicPoolGroup)
	if !exists {
		status.Reason = "public pool group ratio is not configured"
		return status
	}
	status.GroupRatio = ratio
	if ratio != 0 {
		status.Reason = "public pool group ratio must be zero"
		return status
	}
	status.Available = true
	return status
}

// IsChannelAllowedForPublicPool enforces that the zero-cost reserved group can
// only relay through channels explicitly assigned to that group. Other groups
// keep their existing channel-selection behavior.
func IsChannelAllowedForPublicPool(group string, modelName string, channelID int) bool {
	if group != PublicPoolGroup {
		return true
	}
	if !GetPublicPoolStatus().Available {
		return false
	}
	return model.IsChannelEnabledForGroupModel(PublicPoolGroup, modelName, channelID)
}
