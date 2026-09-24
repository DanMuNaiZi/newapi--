package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePublicPoolSettingUpdateRequiresZeroGroupRatio(t *testing.T) {
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousSpecialRatios := ratio_setting.GroupGroupRatio2JSONString()
	previousEnabled := operation_setting.GetPublicPoolSetting().Enabled
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(previousSpecialRatios))
		operation_setting.GetPublicPoolSetting().Enabled = previousEnabled
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":0}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"vip":{"public_pool":0}}`))
	assert.NoError(t, validatePublicPoolSettingUpdate("public_pool_setting.enabled", "true"))

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":0.5}`))
	assert.Error(t, validatePublicPoolSettingUpdate("public_pool_setting.enabled", "true"))

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"public_pool":0}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"vip":{"public_pool":1}}`))
	assert.Error(t, validatePublicPoolSettingUpdate("public_pool_setting.enabled", "true"))

	operation_setting.GetPublicPoolSetting().Enabled = true
	assert.Error(t, validatePublicPoolSettingUpdate("GroupRatio", `{"default":1,"public_pool":1}`))
	assert.Error(t, validatePublicPoolSettingUpdate("GroupGroupRatio", `{"vip":{"public_pool":1}}`))
	assert.NoError(t, validatePublicPoolSettingUpdate("GroupGroupRatio", `{"vip":{"public_pool":0},"default":{}}`))
	assert.NoError(t, validatePublicPoolSettingUpdate("public_pool_setting.enabled", "false"))
	assert.NoError(t, validatePublicPoolSettingUpdate("unrelated", "true"))
}
