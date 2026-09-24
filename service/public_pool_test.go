package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicPoolAvailabilityControlsTheReservedUsableGroup(t *testing.T) {
	previousEnabled := operation_setting.GetPublicPoolSetting().Enabled
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousSpecialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.ReadAll()
	previousUsableGroups := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		operation_setting.GetPublicPoolSetting().Enabled = previousEnabled
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousUsableGroups))
		ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Clear()
		ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.AddAll(previousSpecialGroups)
	})

	operation_setting.GetPublicPoolSetting().Enabled = true
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":0}`))
	status := GetPublicPoolStatus()
	assert.True(t, status.Enabled)
	assert.True(t, status.Available)
	assert.Equal(t, float64(0), status.GroupRatio)
	assert.Contains(t, GetUserUsableGroups("default"), PublicPoolGroup)
	ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Set("default", map[string]string{
		"-:public_pool": "attempted removal",
	})
	assert.Contains(t, GetUserUsableGroups("default"), PublicPoolGroup)

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":1}`))
	status = GetPublicPoolStatus()
	assert.False(t, status.Available)
	assert.NotEmpty(t, status.Reason)
	assert.NotContains(t, GetUserUsableGroups("default"), PublicPoolGroup)

	operation_setting.GetPublicPoolSetting().Enabled = false
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":0}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","public_pool":"Bypass"}`))
	ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Set("default", map[string]string{
		"+:public_pool": "attempted addition",
	})
	status = GetPublicPoolStatus()
	assert.False(t, status.Available)
	assert.NotContains(t, GetUserUsableGroups("default"), PublicPoolGroup)
}

func TestGetUserGroupRatioIgnoresPublicPoolSpecialRatio(t *testing.T) {
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousSpecialRatios := ratio_setting.GroupGroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(previousSpecialRatios))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":0}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"vip":{"public_pool":2}}`))

	assert.Zero(t, GetUserGroupRatio("vip", PublicPoolGroup))
}

func TestIsChannelAllowedForPublicPoolRejectsUnavailableOrInvalidRoutes(t *testing.T) {
	previousEnabled := operation_setting.GetPublicPoolSetting().Enabled
	previousRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		operation_setting.GetPublicPoolSetting().Enabled = previousEnabled
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":0}`))
	operation_setting.GetPublicPoolSetting().Enabled = false
	assert.False(t, IsChannelAllowedForPublicPool(PublicPoolGroup, "public-pool-model", 1))
	assert.True(t, IsChannelAllowedForPublicPool("default", "public-pool-model", 1))

	operation_setting.GetPublicPoolSetting().Enabled = true
	assert.False(t, IsChannelAllowedForPublicPool(PublicPoolGroup, "public-pool-model", 0))
}
