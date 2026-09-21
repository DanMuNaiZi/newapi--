package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newErrorDisplayTestRelayInfo(settings *dto.UpstreamErrorDisplaySettings) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RequestModelName: "gpt-5.6-sol",
		OriginModelName:  "gpt-5.6-terra",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         17,
			ApiKey:            "sk-channel-secret",
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				UpstreamErrorDisplay: settings,
			},
		},
	}
}

func newUnsupportedModelError() *types.NewAPIError {
	return types.WithOpenAIError(types.OpenAIError{
		Message: "The 'gpt-5.6-terra' model is not supported for sk-channel-secret",
		Type:    "invalid_request_error",
		Code:    "unsupported_model",
	}, http.StatusBadRequest)
}

func TestBuildClientRelayErrorDefaultsToGeneric503(t *testing.T) {
	upstreamErr := newUnsupportedModelError()
	policy := &channelErrorDisplayPolicy{
		Default: dto.NormalizeUpstreamErrorDisplaySettings(nil),
	}

	clientErr := buildClientRelayError(types.RelayFormatOpenAI, newErrorDisplayTestRelayInfo(nil), upstreamErr, policy)

	assert.Equal(t, http.StatusServiceUnavailable, clientErr.StatusCode)
	assert.Equal(t, dto.DefaultUpstreamErrorDisplayMessage, clientErr.Error())
	assert.Equal(t, types.ErrorTypeOpenAIError, clientErr.GetErrorType())
	assert.Equal(t, types.ErrorCodeUpstreamServiceUnavailable, clientErr.GetErrorCode())
	assert.Equal(t, http.StatusBadRequest, upstreamErr.StatusCode)
	assert.Contains(t, upstreamErr.Error(), "gpt-5.6-terra")
}

func TestBuildClientRelayErrorHidesAllUpstreamHTTPErrorClassesByDefault(t *testing.T) {
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}
	formats := []types.RelayFormat{
		types.RelayFormatOpenAI,
		types.RelayFormatOpenAIResponses,
		types.RelayFormatOpenAIAudio,
		types.RelayFormatOpenAIImage,
		types.RelayFormatEmbedding,
		types.RelayFormatRerank,
		types.RelayFormatGemini,
		types.RelayFormatOpenAIRealtime,
	}
	policy := &channelErrorDisplayPolicy{
		Default: dto.NormalizeUpstreamErrorDisplaySettings(nil),
	}

	for _, status := range statuses {
		for _, format := range formats {
			upstreamErr := types.WithOpenAIError(types.OpenAIError{
				Message: "private upstream detail",
				Type:    "upstream_type",
				Code:    "upstream_code",
			}, status)

			clientErr := buildClientRelayError(format, newErrorDisplayTestRelayInfo(nil), upstreamErr, policy)

			assert.Equal(t, http.StatusServiceUnavailable, clientErr.StatusCode, "format=%s upstream_status=%d", format, status)
			assert.Equal(t, dto.DefaultUpstreamErrorDisplayMessage, clientErr.Error(), "format=%s upstream_status=%d", format, status)
			assert.NotContains(t, clientErr.Error(), "private upstream detail")
		}
	}
}

func TestBuildClientRelayErrorOriginalUsesMappedStatusAndSanitizedMessage(t *testing.T) {
	upstreamErr := newUnsupportedModelError()
	service.ResetStatusCode(upstreamErr, `{"400":502}`)
	policy := &channelErrorDisplayPolicy{
		Default: dto.NormalizeUpstreamErrorDisplaySettings(nil),
		Record: &model.ChannelErrorRecord{
			DisplayMode: dto.ChannelErrorDisplayModeOriginal,
		},
	}

	clientErr := buildClientRelayError(types.RelayFormatOpenAI, newErrorDisplayTestRelayInfo(nil), upstreamErr, policy)

	assert.Equal(t, http.StatusBadGateway, clientErr.StatusCode)
	assert.Contains(t, clientErr.Error(), "gpt-5.6-sol")
	assert.NotContains(t, clientErr.Error(), "gpt-5.6-terra")
	assert.NotContains(t, clientErr.Error(), "sk-channel-secret")
	assert.Equal(t, types.ErrorCode("unsupported_model"), clientErr.GetErrorCode())
	assert.Equal(t, http.StatusBadRequest, clientErr.GetOriginalStatusCode())
	assert.Contains(t, upstreamErr.Error(), "sk-channel-secret")
}

func TestBuildClientRelayErrorHonorsInheritedDetailsAndPerErrorCustomRule(t *testing.T) {
	showDetails := &dto.UpstreamErrorDisplaySettings{
		ShowDetails: true,
		StatusCode:  http.StatusServiceUnavailable,
		Message:     dto.DefaultUpstreamErrorDisplayMessage,
	}
	upstreamErr := newUnsupportedModelError()

	inherited := buildClientRelayError(types.RelayFormatOpenAI, newErrorDisplayTestRelayInfo(showDetails), upstreamErr, &channelErrorDisplayPolicy{
		Default: dto.NormalizeUpstreamErrorDisplaySettings(showDetails),
		Record:  &model.ChannelErrorRecord{DisplayMode: dto.ChannelErrorDisplayModeInherit},
	})
	assert.Equal(t, http.StatusBadRequest, inherited.StatusCode)
	assert.Contains(t, inherited.Error(), "not supported")

	custom := buildClientRelayError(types.RelayFormatOpenAI, newErrorDisplayTestRelayInfo(showDetails), upstreamErr, &channelErrorDisplayPolicy{
		Default: dto.NormalizeUpstreamErrorDisplaySettings(showDetails),
		Record: &model.ChannelErrorRecord{
			DisplayMode:       dto.ChannelErrorDisplayModeCustom,
			DisplayStatusCode: http.StatusConflict,
			DisplayMessage:    "该账号不支持此模型",
		},
	})
	assert.Equal(t, http.StatusConflict, custom.StatusCode)
	assert.Equal(t, "该账号不支持此模型", custom.Error())
	assert.Equal(t, types.ErrorCodeUpstreamServiceUnavailable, custom.GetErrorCode())

	generic := buildClientRelayError(types.RelayFormatOpenAI, newErrorDisplayTestRelayInfo(showDetails), upstreamErr, &channelErrorDisplayPolicy{
		Default: dto.NormalizeUpstreamErrorDisplaySettings(showDetails),
		Record:  &model.ChannelErrorRecord{DisplayMode: dto.ChannelErrorDisplayModeGeneric},
	})
	assert.Equal(t, http.StatusServiceUnavailable, generic.StatusCode)
	assert.Equal(t, dto.DefaultUpstreamErrorDisplayMessage, generic.Error())
}

func TestBuildClientRelayErrorLeavesLocalErrorsUnchangedWithoutChannelPolicy(t *testing.T) {
	localErr := types.NewOpenAIError(
		assert.AnError,
		types.ErrorCodeInvalidRequest,
		http.StatusUnprocessableEntity,
	)

	clientErr := buildClientRelayError(types.RelayFormatOpenAI, nil, localErr, nil)

	assert.Equal(t, http.StatusUnprocessableEntity, clientErr.StatusCode)
	assert.Equal(t, assert.AnError.Error(), clientErr.Error())
	assert.Equal(t, localErr.GetErrorCode(), clientErr.GetErrorCode())
}

func TestBuildClientRelayErrorUsesClaudeWireShapeForGenericPolicy(t *testing.T) {
	upstreamErr := types.WithClaudeError(types.ClaudeError{
		Type:    "invalid_request_error",
		Message: "upstream detail",
	}, http.StatusBadRequest)
	policy := &channelErrorDisplayPolicy{Default: dto.NormalizeUpstreamErrorDisplaySettings(nil)}

	clientErr := buildClientRelayError(types.RelayFormatClaude, newErrorDisplayTestRelayInfo(nil), upstreamErr, policy)

	assert.Equal(t, http.StatusServiceUnavailable, clientErr.StatusCode)
	assert.Equal(t, types.ClaudeError{
		Type:    "upstream_error",
		Message: dto.DefaultUpstreamErrorDisplayMessage,
	}, clientErr.ToClaudeError())
}

func TestProcessChannelErrorAggregatesSanitizedFailures(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ChannelErrorRecord{}))
	previousErrorLogEnabled := constant.ErrorLogEnabled
	constant.ErrorLogEnabled = false
	t.Cleanup(func() { constant.ErrorLogEnabled = previousErrorLogEnabled })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(common.RequestIdKey, "req-first")
	relayInfo := newErrorDisplayTestRelayInfo(nil)
	channelErr := types.NewChannelError(17, 48, "codex", true, "sk-channel-secret", false)
	upstreamErr := newUnsupportedModelError()

	firstPolicy := processChannelError(c, relayInfo, *channelErr, upstreamErr)
	require.NotNil(t, firstPolicy)
	require.NotNil(t, firstPolicy.Record)
	assert.EqualValues(t, 1, firstPolicy.Record.OccurrenceCount)
	assert.Equal(t, http.StatusBadRequest, firstPolicy.Record.UpstreamStatusCode)
	assert.Equal(t, "gpt-5.6-sol", firstPolicy.Record.RequestModel)
	assert.NotContains(t, firstPolicy.Record.SampleMessage, "sk-channel-secret")
	assert.NotContains(t, firstPolicy.Record.SampleMessage, "gpt-5.6-terra")

	c.Set(common.RequestIdKey, "req-second")
	secondPolicy := processChannelError(c, relayInfo, *channelErr, upstreamErr)
	require.NotNil(t, secondPolicy.Record)
	assert.Equal(t, firstPolicy.Record.ID, secondPolicy.Record.ID)
	assert.EqualValues(t, 2, secondPolicy.Record.OccurrenceCount)
	assert.Equal(t, "req-second", secondPolicy.Record.LastRequestID)
}
