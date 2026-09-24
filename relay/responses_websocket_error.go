package relay

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	appmodel "github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// clientErrorForChannelFailure records one selected-channel failure and builds
// the error copy that Responses WebSocket may expose. The original error is
// returned to the retry, refund, disable and billing paths by the caller.
func (s *responsesWSSession) clientErrorForChannelFailure(c *gin.Context, channel *appmodel.Channel, info *relaycommon.RelayInfo, upstreamErr *types.NewAPIError) *types.NewAPIError {
	if upstreamErr == nil {
		return nil
	}
	if channel == nil && s.lockedChannel != nil {
		channel = s.lockedChannel
	}
	channelID, channelType, channelName, multiKey, key, autoBan := responsesWSChannelErrorFields(channel, info)
	channelError := types.NewChannelError(channelID, channelType, channelName, multiKey, key, autoBan)
	service.ProcessChannelError(c, *channelError, upstreamErr, info)

	requestPath := ""
	if c != nil && c.Request != nil && c.Request.URL != nil {
		requestPath = c.Request.URL.Path
	}
	requestModel := ""
	if info != nil {
		requestModel = info.ClientModelName()
	}
	if requestModel == "" && c != nil {
		requestModel = c.GetString("original_model")
	}
	multiKeyIndex := 0
	secrets := []string{key}
	if info != nil && info.ChannelMeta != nil {
		multiKeyIndex = info.ChannelMultiKeyIndex
		secrets = append(secrets, info.ApiKey)
	}
	sample := relaycommon.ClientVisibleErrorMessage(info, upstreamErr.MaskSensitiveError())
	sample = service.SanitizeUpstreamErrorMessage(sample, secrets...)
	var record *appmodel.ChannelErrorRecord
	var err error
	if channelID > 0 {
		record, err = appmodel.UpsertChannelErrorRecord(appmodel.ChannelErrorRecordInput{
			ChannelID:          channelID,
			RequestPath:        requestPath,
			RequestModel:       requestModel,
			UpstreamStatusCode: upstreamErr.GetOriginalStatusCode(),
			FinalStatusCode:    upstreamErr.StatusCode,
			ErrorType:          string(upstreamErr.GetErrorType()),
			ErrorCode:          string(upstreamErr.GetErrorCode()),
			SampleMessage:      sample,
			RequestID:          c.GetString(common.RequestIdKey),
			MultiKeyIndex:      multiKeyIndex,
		})
		if err != nil {
			logger.LogWarn(c, fmt.Sprintf("failed to aggregate Responses WebSocket channel error for channel #%d: %v", channelID, err))
		}
	}

	settings := dto.NormalizeUpstreamErrorDisplaySettings(nil)
	if info != nil && info.ChannelOtherSettings.UpstreamErrorDisplay != nil {
		settings = dto.NormalizeUpstreamErrorDisplaySettings(info.ChannelOtherSettings.UpstreamErrorDisplay)
	}
	return responsesWSClientError(info, upstreamErr, settings, record, secrets...)
}

func responsesWSChannelErrorFields(channel *appmodel.Channel, info *relaycommon.RelayInfo) (int, int, string, bool, string, bool) {
	if channel != nil {
		return channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, channel.Key, channel.GetAutoBan()
	}
	if info != nil && info.ChannelMeta != nil {
		meta := info.ChannelMeta
		return meta.ChannelId, meta.ChannelType, "", meta.ChannelIsMultiKey, meta.ApiKey, false
	}
	return 0, 0, "", false, "", false
}

func responsesWSClientError(info *relaycommon.RelayInfo, upstreamErr *types.NewAPIError, settings dto.UpstreamErrorDisplaySettings, record *appmodel.ChannelErrorRecord, secrets ...string) *types.NewAPIError {
	clientErr := upstreamErr.ClientCopy()
	mode := dto.ChannelErrorDisplayModeInherit
	if record != nil && record.DisplayMode != "" {
		mode = record.DisplayMode
	}
	if mode == dto.ChannelErrorDisplayModeCustom && record != nil {
		rule := dto.ChannelErrorDisplayRule{Mode: mode, StatusCode: record.DisplayStatusCode, Message: record.DisplayMessage}
		if rule.Validate() == nil {
			clientErr.SetOpenAIClientError(rule.Message, "upstream_error", types.ErrorCodeUpstreamServiceUnavailable, rule.StatusCode)
			return clientErr
		}
	}
	showOriginal := false
	switch mode {
	case dto.ChannelErrorDisplayModeInherit:
		showOriginal = settings.ShowDetails
	case dto.ChannelErrorDisplayModeOriginal:
		showOriginal = true
	}
	if !showOriginal {
		clientErr.SetOpenAIClientError(settings.Message, "upstream_error", types.ErrorCodeUpstreamServiceUnavailable, settings.StatusCode)
		return clientErr
	}
	message := relaycommon.ClientVisibleErrorMessage(info, clientErr.Error())
	clientErr.SetMessage(service.SanitizeUpstreamErrorMessage(strings.TrimSpace(message), secrets...))
	clientErr.ClearMetadata()
	return clientErr
}
