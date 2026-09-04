package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessChannelErrorRecordsRequestedModelAndMasksMappedUpstream(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{
		Username: "relay-model-visibility-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "relay-model-visibility-user",
	}
	require.NoError(t, db.Create(user).Error)

	previousErrorLogEnabled := constant.ErrorLogEnabled
	constant.ErrorLogEnabled = true
	t.Cleanup(func() { constant.ErrorLogEnabled = previousErrorLogEnabled })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	ctx.Set("id", user.Id)
	ctx.Set("username", user.Username)
	ctx.Set("original_model", "gpt-5.6-sol-openai-compact")
	ctx.Set("token_name", "test-token")
	ctx.Set("token_id", 17)
	ctx.Set("group", user.Group)
	ctx.Set("channel_id", 135)
	ctx.Set("channel_name", "mapped-codex")
	ctx.Set("channel_type", constant.ChannelTypeCodex)
	ctx.Set("use_channel", []string{"135"})
	common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Now().Add(-time.Second))
	common.SetContextKey(ctx, constant.ContextKeyIsStream, true)

	relayInfo := &relaycommon.RelayInfo{
		RequestModelName: "gpt-5.6-sol",
		OriginModelName:  "gpt-5.6-sol-openai-compact",
		ChannelMeta: &relaycommon.ChannelMeta{
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
		},
	}
	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "auth_unavailable: no auth available (providers=codex, model=gpt-5.6-terra)",
		Type:    "upstream_error",
		Code:    "auth_unavailable",
	}, http.StatusServiceUnavailable)

	processChannelError(ctx, relayInfo, types.ChannelError{ChannelId: 135}, apiErr)

	var errorLog model.Log
	require.NoError(t, db.Where("type = ? AND user_id = ?", model.LogTypeError, user.Id).First(&errorLog).Error)
	assert.Equal(t, "gpt-5.6-sol", errorLog.ModelName)
	assert.Contains(t, errorLog.Content, "model=gpt-5.6-sol")
	assert.NotContains(t, errorLog.Content, "model=gpt-5.6-terra")

	var other map[string]interface{}
	require.NoError(t, common.Unmarshal([]byte(errorLog.Other), &other))
	assert.Equal(t, "gpt-5.6-sol", other["request_model_name"])
	assert.Equal(t, "gpt-5.6-terra", other["upstream_model_name"])
	assert.Equal(t, true, other["is_model_mapped"])
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	assert.NotContains(t, adminInfo, "upstream_model_name")

	model.HideActualModelNames([]*model.Log{&errorLog})
	var hiddenOther map[string]interface{}
	require.NoError(t, common.Unmarshal([]byte(errorLog.Other), &hiddenOther))
	assert.NotContains(t, hiddenOther, "upstream_model_name")
	assert.NotContains(t, hiddenOther, "is_model_mapped")
}

func TestGetAllLogsHidesMappedUpstreamModelWithoutActualModelPermission(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	user := &model.User{
		Username: "relay-log-visibility-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		AffCode:  "relay-log-visibility-user",
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      model.LogTypeError,
		ModelName: "gpt-5.6-terra",
		Content:   "mapped channel failed",
		Other:     `{"request_model_name":"gpt-5.6-sol","upstream_model_name":"gpt-5.6-terra","is_model_mapped":true,"admin_info":{"upstream_model_name":"gpt-5.6-terra"}}`,
		CreatedAt: common.GetTimestamp(),
	}).Error)

	callAllLogs := func(role int) string {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/log?p=1&page_size=10", nil)
		ctx.Set("id", user.Id)
		ctx.Set("role", role)
		GetAllLogs(ctx)
		return recorder.Body.String()
	}

	withoutActualModelPermission := callAllLogs(common.RoleAuthorizedAdmin)
	assert.Contains(t, withoutActualModelPermission, "gpt-5.6-sol")
	assert.NotContains(t, withoutActualModelPermission, "gpt-5.6-terra")
	assert.NotContains(t, withoutActualModelPermission, "upstream_model_name")

	rootLogs := callAllLogs(common.RoleRootUser)
	assert.Contains(t, rootLogs, "gpt-5.6-sol")
	assert.Contains(t, rootLogs, "gpt-5.6-terra")
	assert.Contains(t, rootLogs, "actual_model_name")
}
