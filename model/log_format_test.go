package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]any{
		"model_price": 0.004,
		"admin_info": map[string]any{
			"quota_saturation": map[string]any{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

func TestFormatUserLogsRemovesResourceQuota(t *testing.T) {
	resourceQuota := 500
	logs := []*Log{{ModelName: "gpt-5.4", ResourceQuota: &resourceQuota}}
	formatUserLogs(logs, 0)
	assert.Nil(t, logs[0].ResourceQuota)
	data, err := common.Marshal(logs[0])
	require.NoError(t, err)
	assert.NotContains(t, string(data), "resource_quota")
}

func TestTaskPluginLogVisibilityIsRoleSeparated(t *testing.T) {
	other := common.MapToJsonStr(map[string]any{
		"model_price": 1.25,
		"admin_info": map[string]any{
			"task_plugin": map[string]any{
				"key":     "document-parser",
				"name":    "Document Parser",
				"version": "1.2.3",
			},
		},
		"root_info": map[string]any{
			"upstream_task_id": "upstream-private",
			"task_plugin": map[string]any{
				"generation": 42,
			},
		},
	})

	t.Run("user", func(t *testing.T) {
		logs := []*Log{{Other: other}}
		formatUserLogs(logs, 0)

		parsed, err := common.StrToMap(logs[0].Other)
		require.NoError(t, err)
		assert.NotContains(t, parsed, "admin_info")
		assert.NotContains(t, parsed, "root_info")
		assert.Equal(t, 1.25, parsed["model_price"])
	})

	t.Run("admin", func(t *testing.T) {
		logs := []*Log{{Other: other}}
		FormatAdminLogs(logs)

		parsed, err := common.StrToMap(logs[0].Other)
		require.NoError(t, err)
		assert.Contains(t, parsed, "admin_info")
		assert.NotContains(t, parsed, "root_info")
	})

	t.Run("root", func(t *testing.T) {
		logs := []*Log{{Other: other}}
		FormatRootLogs(logs)

		parsed, err := common.StrToMap(logs[0].Other)
		require.NoError(t, err)
		assert.Contains(t, parsed, "admin_info")
		assert.Contains(t, parsed, "root_info")
	})
}

func TestLegacyLogOtherVisibilityIsRoleSeparated(t *testing.T) {
	other := common.MapToJsonStr(map[string]any{
		"request_path":        "/v1/chat/completions",
		"channel_id":          202,
		"channel_name":        "legacy-secret-channel",
		"channel_type":        1,
		"reject_reason":       "legacy-policy-rejection",
		"upstream_model_name": "legacy-private-model",
		"is_model_mapped":     true,
		"response_model":      map[string]any{"upstream_model": "legacy-private-model", "returned_model": "legacy-provider-model"},
		"admin_info": map[string]any{
			"existing_admin_field": "preserved",
		},
		"root_info": map[string]any{
			"upstream_request_id": "upstream-private",
		},
		"audit_info": map[string]any{
			"method": "POST",
		},
	})

	t.Run("user", func(t *testing.T) {
		logs := []*Log{{
			Id:          99,
			ChannelId:   77,
			ChannelName: "resolved-secret-channel",
			Other:       other,
		}}

		formatUserLogs(logs, 10)

		assert.Equal(t, 11, logs[0].Id)
		assert.Equal(t, 77, logs[0].ChannelId)
		assert.Empty(t, logs[0].ChannelName)
		parsed, err := common.StrToMap(logs[0].Other)
		require.NoError(t, err)
		assert.Equal(t, "/v1/chat/completions", parsed["request_path"])
		for _, key := range []string{
			"channel_id",
			"channel_name",
			"channel_type",
			"reject_reason",
			"upstream_model_name",
			"is_model_mapped",
			"response_model",
			"admin_info",
			"root_info",
			"audit_info",
		} {
			assert.NotContains(t, parsed, key)
		}
	})

	t.Run("admin", func(t *testing.T) {
		logs := []*Log{{Other: other}}

		FormatAdminLogs(logs)

		parsed, err := common.StrToMap(logs[0].Other)
		require.NoError(t, err)
		assert.Equal(t, "legacy-secret-channel", parsed["channel_name"])
		assert.NotContains(t, parsed, "reject_reason")
		assert.NotContains(t, parsed, "root_info")
		assert.Contains(t, parsed, "audit_info")
		adminInfo, ok := parsed["admin_info"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "preserved", adminInfo["existing_admin_field"])
		assert.Equal(t, "legacy-policy-rejection", adminInfo["reject_reason"])
		assert.NotContains(t, parsed, "upstream_model_name")
		assert.NotContains(t, parsed, "is_model_mapped")
		assert.NotContains(t, parsed, "response_model")
		assert.Equal(t, "legacy-private-model", adminInfo["upstream_model_name"])
		assert.Equal(t, true, adminInfo["is_model_mapped"])
		assert.Equal(t, map[string]any{"upstream_model": "legacy-private-model", "returned_model": "legacy-provider-model"}, adminInfo["response_model"])
	})

	t.Run("root", func(t *testing.T) {
		logs := []*Log{{Other: other}}

		FormatRootLogs(logs)

		parsed, err := common.StrToMap(logs[0].Other)
		require.NoError(t, err)
		assert.Equal(t, "legacy-secret-channel", parsed["channel_name"])
		assert.NotContains(t, parsed, "reject_reason")
		assert.Contains(t, parsed, "root_info")
		assert.Contains(t, parsed, "audit_info")
		adminInfo, ok := parsed["admin_info"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "preserved", adminInfo["existing_admin_field"])
		assert.Equal(t, "legacy-policy-rejection", adminInfo["reject_reason"])
		assert.NotContains(t, parsed, "upstream_model_name")
		assert.NotContains(t, parsed, "is_model_mapped")
		assert.NotContains(t, parsed, "response_model")
		assert.Equal(t, "legacy-private-model", adminInfo["upstream_model_name"])
		assert.Equal(t, true, adminInfo["is_model_mapped"])
		assert.Equal(t, map[string]any{"upstream_model": "legacy-private-model", "returned_model": "legacy-provider-model"}, adminInfo["response_model"])
	})
}

func TestLegacyRejectReasonDoesNotOverrideScopedValue(t *testing.T) {
	other := common.MapToJsonStr(map[string]any{
		"reject_reason":       "legacy-value",
		"upstream_model_name": "legacy-private-model",
		"is_model_mapped":     true,
		"response_model":      map[string]any{"upstream_model": "legacy-private-model"},
		"admin_info": map[string]any{
			"reject_reason":       "scoped-value",
			"upstream_model_name": "scoped-private-model",
			"is_model_mapped":     false,
			"response_model":      map[string]any{"upstream_model": "scoped-private-model"},
		},
	})
	logs := []*Log{{Other: other}}

	FormatRootLogs(logs)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, parsed, "reject_reason")
	adminInfo, ok := parsed["admin_info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "scoped-value", adminInfo["reject_reason"])
	assert.NotContains(t, parsed, "upstream_model_name")
	assert.NotContains(t, parsed, "is_model_mapped")
	assert.NotContains(t, parsed, "response_model")
	assert.Equal(t, "scoped-private-model", adminInfo["upstream_model_name"])
	assert.Equal(t, false, adminInfo["is_model_mapped"])
	assert.Equal(t, map[string]any{"upstream_model": "scoped-private-model"}, adminInfo["response_model"])
}

func TestLegacyRejectReasonHandlesNullAdminInfo(t *testing.T) {
	logs := []*Log{{Other: `{"reject_reason":"legacy-value","upstream_model_name":"legacy-private-model","is_model_mapped":true,"response_model":{"upstream_model":"legacy-private-model"},"admin_info":null}`}}

	FormatAdminLogs(logs)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, parsed, "reject_reason")
	adminInfo, ok := parsed["admin_info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "legacy-value", adminInfo["reject_reason"])
	assert.NotContains(t, parsed, "upstream_model_name")
	assert.NotContains(t, parsed, "is_model_mapped")
	assert.NotContains(t, parsed, "response_model")
	assert.Equal(t, "legacy-private-model", adminInfo["upstream_model_name"])
	assert.Equal(t, true, adminInfo["is_model_mapped"])
	assert.Equal(t, map[string]any{"upstream_model": "legacy-private-model"}, adminInfo["response_model"])
}

func TestLogFormattingPreservesLargeIntegerLexemes(t *testing.T) {
	const other = `{"public_id":9007199254740993,"admin_info":{"admin_id":9007199254740995},"root_info":{"generation":18446744073709551615}}`

	t.Run("user", func(t *testing.T) {
		logs := []*Log{{Other: other}}

		formatUserLogs(logs, 0)

		assert.Contains(t, logs[0].Other, `"public_id":9007199254740993`)
		assert.NotContains(t, logs[0].Other, "admin_id")
		assert.NotContains(t, logs[0].Other, "generation")
	})

	t.Run("admin", func(t *testing.T) {
		logs := []*Log{{Other: other}}

		FormatAdminLogs(logs)

		assert.Contains(t, logs[0].Other, `"public_id":9007199254740993`)
		assert.Contains(t, logs[0].Other, `"admin_id":9007199254740995`)
		assert.NotContains(t, logs[0].Other, "generation")
	})

	t.Run("root", func(t *testing.T) {
		logs := []*Log{{Other: other}}

		FormatRootLogs(logs)

		assert.Equal(t, other, logs[0].Other)
	})

	t.Run("unprivileged", func(t *testing.T) {
		const unprivileged = `{"public_id":9007199254740993,"model_price":0.004}`

		userLogs := []*Log{{Other: unprivileged}}
		formatUserLogs(userLogs, 0)
		assert.Equal(t, unprivileged, userLogs[0].Other)

		adminLogs := []*Log{{Other: unprivileged}}
		FormatAdminLogs(adminLogs)
		assert.Equal(t, unprivileged, adminLogs[0].Other)
	})
}

func TestFormatUserLogsRemovesActualModelFields(t *testing.T) {
	logs := []*Log{{
		ModelName: "gpt-5.4",
		Other:     `{"request_model_name":"gpt-5.5","admin_info":{"upstream_model_name":"gpt-5.4","is_model_mapped":true,"channel":11},"model_ratio":2}`,
	}}

	formatUserLogs(logs, 0)

	data, err := common.Marshal(logs[0])
	require.NoError(t, err)
	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.Equal(t, "gpt-5.5", logs[0].ModelName)
	assert.Empty(t, logs[0].ActualModelName)
	assert.NotContains(t, string(data), "actual_model_name")
	assert.NotContains(t, other, "admin_info")
	assert.Contains(t, other, "request_model_name")
	assert.Contains(t, other, "model_ratio")
}

func TestResponseModelLogVisibilityPreservesSafePublicObservation(t *testing.T) {
	for _, tc := range []struct {
		name, observation string
		public            bool
	}{
		{"safe mapped observation", `{"requested_model":"requested","upstream_model":"requested","returned_model":"requested"}`, true},
		{"unmapped returned difference", `{"requested_model":"requested","upstream_model":"requested","returned_model":"requested-2026"}`, true},
		{"empty upstream declaration", `{"requested_model":"requested","upstream_model":"","returned_model":"returned"}`, true},
		{"legacy mapped observation", `{"requested_model":"requested","upstream_model":"private-model","returned_model":"private-model"}`, false},
		{"incomplete historical object", `{"upstream_model":"private-model"}`, false},
		{"unknown upstream name", `{"requested_model":"requested","upstream_model":null,"returned_model":"private-model"}`, false},
		{"unknown returned name", `{"requested_model":"requested","upstream_model":"requested","returned_model":null}`, false},
		{"unexpected historical detail", `{"requested_model":"requested","upstream_model":"requested","returned_model":"requested","private_detail":"private"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := `{"public_id":9007199254740993,"response_model":` + tc.observation + `}`
			user := []*Log{{Other: other}}
			formatUserLogs(user, 0)
			var userFields map[string]common.RawMessage
			require.NoError(t, common.UnmarshalJsonStr(user[0].Other, &userFields))
			assert.Equal(t, tc.public, len(userFields["response_model"]) > 0)
			assert.Contains(t, user[0].Other, `"public_id":9007199254740993`)
			if tc.public {
				assert.JSONEq(t, tc.observation, string(userFields["response_model"]))
			} else {
				assert.NotContains(t, user[0].Other, "private")
			}
			admin := []*Log{{Other: other}}
			FormatAdminLogs(admin)
			var adminFields map[string]common.RawMessage
			require.NoError(t, common.UnmarshalJsonStr(admin[0].Other, &adminFields))
			if tc.public {
				assert.JSONEq(t, tc.observation, string(adminFields["response_model"]))
			} else {
				assert.NotContains(t, adminFields, "response_model")
				var scoped map[string]common.RawMessage
				require.NoError(t, common.Unmarshal(adminFields["admin_info"], &scoped))
				assert.JSONEq(t, tc.observation, string(scoped["response_model"]))
			}
		})
	}
}

func TestPopulateActualModelNamesReadsAdminScope(t *testing.T) {
	logs := []*Log{{
		ModelName: "gpt-5.4",
		Content:   "The 'gpt-5.4' model is unavailable",
		Other:     `{"request_model_name":"gpt-5.5","admin_info":{"upstream_model_name":"gpt-5.4","is_model_mapped":true}}`,
	}, {
		ModelName: "gpt-5.4",
		Content:   "The 'gpt-5.4' model is unavailable",
		Other:     `{"request_model_name":"gpt-5.5","upstream_model_name":"gpt-5.4","is_model_mapped":true}`,
	}, {
		ModelName: "gpt-5.4",
		Content:   "The 'gpt-5.4' model is unavailable",
		Other:     `{"request_model_name":"gpt-5.5"}`,
	}}

	populateActualModelNames(logs)
	FormatAdminLogs(logs)

	for _, log := range logs {
		assert.Equal(t, "gpt-5.5", log.ModelName)
		assert.Equal(t, "gpt-5.4", log.ActualModelName)
		assert.Equal(t, "The 'gpt-5.4' model is unavailable", log.Content)
	}
	HideActualModelNames(logs)
	for _, log := range logs {
		assert.Equal(t, "gpt-5.5", log.ModelName)
		assert.Empty(t, log.ActualModelName)
		assert.NotContains(t, log.Other, "gpt-5.4")
		assert.Equal(t, "The 'gpt-5.5' model is unavailable", log.Content)
	}
}

func TestHideActualModelNamesRemovesUpstreamModelFields(t *testing.T) {
	logs := []*Log{{
		ModelName:       "gpt-5.5",
		ActualModelName: "gpt-5.4",
		Other:           `{"request_model_name":"gpt-5.5","upstream_model_name":"legacy-private-model","is_model_mapped":true,"response_model":{"upstream_model":"legacy-private-model"},"admin_info":{"use_channel":[11],"upstream_model_name":"gpt-5.4","is_model_mapped":true,"response_model":{"upstream_model":"gpt-5.4"}},"root_info":{"upstream_model_name":"root-private-model","is_model_mapped":true,"response_model":{"upstream_model":"root-private-model"}}}`,
	}}

	HideActualModelNames(logs)

	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	adminInfo, ok := other["admin_info"].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, logs[0].ActualModelName)
	assert.Equal(t, "gpt-5.5", other["request_model_name"])
	assert.NotContains(t, adminInfo, "upstream_model_name")
	assert.NotContains(t, adminInfo, "is_model_mapped")
	assert.NotContains(t, adminInfo, "response_model")
	assert.Contains(t, adminInfo, "use_channel")
	assert.NotContains(t, other, "upstream_model_name")
	assert.NotContains(t, other, "is_model_mapped")
	assert.NotContains(t, other, "response_model")
	assert.NotContains(t, logs[0].Other, "legacy-private-model")
	assert.NotContains(t, logs[0].Other, "root-private-model")
}
