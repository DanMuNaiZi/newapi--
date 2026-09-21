package controller

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

type channelErrorDisplayPolicy struct {
	Default dto.UpstreamErrorDisplaySettings
	Record  *model.ChannelErrorRecord
}

func buildClientRelayError(relayFormat types.RelayFormat, relayInfo *relaycommon.RelayInfo, upstreamErr *types.NewAPIError, policy *channelErrorDisplayPolicy) *types.NewAPIError {
	if upstreamErr == nil {
		return nil
	}
	clientErr := upstreamErr.ClientCopy()
	if policy == nil {
		return clientErr
	}

	defaultSettings := dto.NormalizeUpstreamErrorDisplaySettings(&policy.Default)
	mode := dto.ChannelErrorDisplayModeInherit
	if policy.Record != nil && policy.Record.DisplayMode != "" {
		mode = policy.Record.DisplayMode
	}

	showOriginal := false
	switch mode {
	case dto.ChannelErrorDisplayModeInherit:
		showOriginal = defaultSettings.ShowDetails
	case dto.ChannelErrorDisplayModeOriginal:
		showOriginal = true
	case dto.ChannelErrorDisplayModeGeneric:
		showOriginal = false
	case dto.ChannelErrorDisplayModeCustom:
		rule := dto.ChannelErrorDisplayRule{
			Mode:       policy.Record.DisplayMode,
			StatusCode: policy.Record.DisplayStatusCode,
			Message:    policy.Record.DisplayMessage,
		}
		if rule.Validate() == nil {
			setGenericClientRelayError(relayFormat, clientErr, rule.Message, rule.StatusCode)
			return clientErr
		}
	default:
		showOriginal = false
	}

	if !showOriginal {
		setGenericClientRelayError(relayFormat, clientErr, defaultSettings.Message, defaultSettings.StatusCode)
		return clientErr
	}

	message := relaycommon.ClientVisibleErrorMessage(relayInfo, clientErr.Error())
	secrets := make([]string, 0, 1)
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		secrets = append(secrets, relayInfo.ApiKey)
	}
	clientErr.SetMessage(service.SanitizeUpstreamErrorMessage(message, secrets...))
	clientErr.ClearMetadata()
	return clientErr
}

func setGenericClientRelayError(relayFormat types.RelayFormat, clientErr *types.NewAPIError, message string, statusCode int) {
	if relayFormat == types.RelayFormatClaude {
		clientErr.SetClaudeClientError(message, "upstream_error", types.ErrorCodeUpstreamServiceUnavailable, statusCode)
		return
	}
	clientErr.SetOpenAIClientError(message, "upstream_error", types.ErrorCodeUpstreamServiceUnavailable, statusCode)
}

func processChannelError(c *gin.Context, relayInfo *relaycommon.RelayInfo, channelError types.ChannelError, err *types.NewAPIError) *channelErrorDisplayPolicy {
	defaultSettings := dto.NormalizeUpstreamErrorDisplaySettings(nil)
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		defaultSettings = dto.NormalizeUpstreamErrorDisplaySettings(relayInfo.ChannelOtherSettings.UpstreamErrorDisplay)
	}
	policy := &channelErrorDisplayPolicy{Default: defaultSettings}

	logger.LogError(c, fmt.Sprintf("channel error (channel #%d, status code: %d): %s", channelError.ChannelId, err.StatusCode, common.LocalLogPreview(err.Error())))
	requestPath := ""
	if c.Request != nil && c.Request.URL != nil {
		requestPath = c.Request.URL.Path
	}
	requestModel := c.GetString("original_model")
	multiKeyIndex := 0
	secrets := []string{channelError.UsingKey}
	if relayInfo != nil {
		if relayInfo.ClientModelName() != "" {
			requestModel = relayInfo.ClientModelName()
		}
		if relayInfo.ChannelMeta != nil {
			multiKeyIndex = relayInfo.ChannelMultiKeyIndex
			secrets = append(secrets, relayInfo.ApiKey)
		}
	}
	sampleMessage := relaycommon.ClientVisibleErrorMessage(relayInfo, err.MaskSensitiveError())
	sampleMessage = service.SanitizeUpstreamErrorMessage(sampleMessage, secrets...)
	record, recordErr := model.UpsertChannelErrorRecord(model.ChannelErrorRecordInput{
		ChannelID:          channelError.ChannelId,
		RequestPath:        requestPath,
		RequestModel:       requestModel,
		UpstreamStatusCode: err.GetOriginalStatusCode(),
		FinalStatusCode:    err.StatusCode,
		ErrorType:          string(err.GetErrorType()),
		ErrorCode:          string(err.GetErrorCode()),
		SampleMessage:      sampleMessage,
		RequestID:          c.GetString(common.RequestIdKey),
		MultiKeyIndex:      multiKeyIndex,
	})
	if recordErr != nil {
		logger.LogWarn(c, fmt.Sprintf("failed to aggregate channel error for channel #%d: %v", channelError.ChannelId, recordErr))
	} else {
		policy.Record = record
	}

	// Do not use context to get channel information here. Retry may already have
	// replaced the selected channel in the context.
	if service.ShouldDisableChannel(err) && channelError.AutoBan {
		gopool.Go(func() {
			service.DisableChannel(channelError, err.ErrorWithStatusCode())
		})
	}

	if constant.ErrorLogEnabled && types.IsRecordErrorLog(err) {
		userID := c.GetInt("id")
		tokenName := c.GetString("token_name")
		modelName := requestModel
		tokenID := c.GetInt("token_id")
		userGroup := c.GetString("group")
		other := make(map[string]interface{})
		if requestPath != "" {
			other["request_path"] = requestPath
		}
		other["error_type"] = err.GetErrorType()
		other["error_code"] = err.GetErrorCode()
		other["status_code"] = err.StatusCode
		other["channel_id"] = channelError.ChannelId
		other["channel_name"] = channelError.ChannelName
		other["channel_type"] = channelError.ChannelType
		adminInfo := make(map[string]interface{})
		adminInfo["use_channel"] = c.GetStringSlice("use_channel")
		if channelError.IsMultiKey {
			adminInfo["is_multi_key"] = true
			adminInfo["multi_key_index"] = multiKeyIndex
		}
		service.AppendChannelAffinityAdminInfo(c, adminInfo)
		relaycommon.AppendMappedModelLogInfo(relayInfo, other)
		other["admin_info"] = adminInfo
		startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
		if startTime.IsZero() {
			startTime = time.Now()
		}
		useTimeSeconds := int(time.Since(startTime).Seconds())
		content := relaycommon.ClientVisibleErrorMessage(relayInfo, err.MaskSensitiveErrorWithStatusCode())
		model.RecordErrorLog(c, userID, channelError.ChannelId, modelName, tokenName, content, tokenID, useTimeSeconds, common.GetContextKeyBool(c, constant.ContextKeyIsStream), userGroup, other)
	}

	return policy
}
