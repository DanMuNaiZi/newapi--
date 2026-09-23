package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManualWechatPaymentOptionsUsePaymentPermission(t *testing.T) {
	for _, action := range []string{authz.ActionRead, authz.ActionWrite} {
		permission := optionPermission("payment_setting.manual_wechat_enabled", action)

		assert.Equal(t, authz.ResourcePayment, permission.Resource)
		assert.Equal(t, action, permission.Action)
	}
}

func TestManualWechatPaymentOptionsRequireDedicatedAtomicEndpoint(t *testing.T) {
	context, recorder := manualWechatTestContext(
		http.MethodPut,
		`{"key":"payment_setting.manual_wechat_enabled","value":true}`,
	)
	context.Set("id", 1)
	context.Set("role", common.RoleRootUser)

	UpdateOption(context)

	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), "专用接口")
}

func TestGenericOptionsResponseOmitsManualWechatQRCode(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = map[string]string{
		manualWechatEnabledOptionKey: "true",
		manualWechatQRCodeOptionKey:  "data:image/png;base64,SECRET",
		"Price":                      "7.3",
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})

	context, recorder := manualWechatTestContext(http.MethodGet, "")
	context.Set("id", 1)
	context.Set("role", common.RoleRootUser)
	GetOptions(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), manualWechatQRCodeOptionKey)
	assert.NotContains(t, recorder.Body.String(), "data:image/png;base64")
	assert.Contains(t, recorder.Body.String(), `"key":"Price"`)
}
