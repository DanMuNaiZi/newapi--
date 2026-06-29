package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatUserLogsRemovesActualModelFields(t *testing.T) {
	logs := []*Log{
		{
			ModelName: "gpt-5.4",
			Other:     `{"request_model_name":"gpt-5.5","upstream_model_name":"gpt-5.4","is_model_mapped":true,"admin_info":{"channel":11},"stream_status":{"status":"ok"},"model_ratio":2}`,
		},
	}

	formatUserLogs(logs, 0)

	data, err := common.Marshal(logs[0])
	require.NoError(t, err)
	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.Equal(t, "gpt-5.5", logs[0].ModelName)
	assert.Empty(t, logs[0].ActualModelName)
	assert.NotContains(t, string(data), "actual_model_name")
	assert.NotContains(t, other, "request_model_name")
	assert.NotContains(t, other, "upstream_model_name")
	assert.NotContains(t, other, "is_model_mapped")
	assert.NotContains(t, other, "admin_info")
	assert.NotContains(t, other, "stream_status")
	assert.Contains(t, other, "model_ratio")
}

func TestFormatAdminLogsExposesRequestAndActualModel(t *testing.T) {
	logs := []*Log{
		{
			ModelName: "gpt-5.4",
			Other:     `{"request_model_name":"gpt-5.5","upstream_model_name":"gpt-5.4","is_model_mapped":true,"model_ratio":2}`,
		},
	}

	formatAdminLogs(logs)

	assert.Equal(t, "gpt-5.5", logs[0].ModelName)
	assert.Equal(t, "gpt-5.4", logs[0].ActualModelName)
}
