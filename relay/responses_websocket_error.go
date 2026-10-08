package relay

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	appmodel "github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
	errorType, errorCode := service.ChannelErrorAuditLabels(info, upstreamErr, secrets...)
	var record *appmodel.ChannelErrorRecord
	var err error
	if channelID > 0 {
		record, err = appmodel.UpsertChannelErrorRecord(appmodel.ChannelErrorRecordInput{
			ChannelID:          channelID,
			RequestPath:        requestPath,
			RequestModel:       requestModel,
			UpstreamStatusCode: upstreamErr.GetOriginalStatusCode(),
			FinalStatusCode:    upstreamErr.StatusCode,
			ErrorType:          errorType,
			ErrorCode:          errorCode,
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
	return service.SanitizeClientUpstreamError(info, upstreamErr, secrets...)
}

// A Responses failure can be an explicit error terminal or a terminal whose
// response status is failed. A regular incomplete result carries no error.
func responsesWSResponseFailure(event *dto.ResponsesStreamResponse) *types.OpenAIError {
	if event == nil {
		return nil
	}
	explicitFailure := event.Type == "response.failed" || event.Type == "response.error"
	switch event.Type {
	case "response.failed", "response.error", "response.incomplete", "response.completed", "response.done":
	default:
		return nil
	}
	if event.Response != nil {
		if upstreamError := event.Response.GetOpenAIError(); upstreamError != nil {
			return upstreamError
		}
		var status string
		_ = common.Unmarshal(event.Response.Status, &status)
		explicitFailure = explicitFailure || status == "failed"
	}
	if !explicitFailure {
		return nil
	}
	message, code := event.Message, event.Code
	if message == "" {
		message = "upstream response failed"
	}
	if code == "" {
		code = "server_error"
	}
	return &types.OpenAIError{Type: "server_error", Code: code, Message: message}
}

// Idle or unrelated frames have no owning billing request. They still belong
// to the selected channel and must pass its display policy.
func (s *responsesWSSession) forwardUnattributedUpstreamMessage(incoming responsesWSMessage, info *relaycommon.RelayInfo, lastResponseID string) error {
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	if websocket.IsCloseError(incoming.err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return incoming.err
	}
	var event dto.ResponsesStreamResponse
	parseErr := common.Unmarshal(incoming.body, &event)
	var failure *types.NewAPIError
	var rejection responsesWSErrorEvent
	upstreamModel := ""
	if incoming.err != nil {
		failure = types.NewErrorWithStatusCode(incoming.err, types.ErrorCodeReadResponseBodyFailed, http.StatusBadGateway, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	} else if parseErr != nil {
		failure = types.NewErrorWithStatusCode(parseErr, types.ErrorCodeBadResponseBody, http.StatusBadGateway, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	} else if event.Type == "error" {
		_ = common.Unmarshal(incoming.body, &rejection)
		if rejection.ResponseID != "" && rejection.ResponseID == lastResponseID {
			return nil
		}
		if rejection.Error == nil {
			rejection.Error = &types.OpenAIError{Type: "server_error", Code: event.Code, Message: event.Message}
		}
		status := rejection.Status
		if status < 400 || status > 599 {
			status = http.StatusBadRequest
		}
		failure = types.WithOpenAIError(*rejection.Error, status, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	} else if upstreamError := responsesWSResponseFailure(&event); upstreamError != nil {
		if event.Response != nil {
			if event.Response.ID != "" && event.Response.ID == lastResponseID {
				return nil
			}
			upstreamModel = event.Response.Model
		}
		failure = types.WithOpenAIError(*upstreamError, http.StatusBadGateway, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}
	if failure == nil {
		return s.writeClient(incoming.kind, incoming.body)
	}
	// Request-specific billing, quota and stream state must never be changed by
	// a frame that belongs to another response on this connection.
	meta := *info.ChannelMeta
	info = &relaycommon.RelayInfo{RequestModelName: info.ClientModelName(), OriginModelName: info.ClientModelName(), ChannelMeta: &meta}
	c := &gin.Context{Request: s.request.Clone(s.ctx)}
	c.Request.Header = make(http.Header)
	c.Set(common.RequestIdKey, s.requestID+"-idle")
	channel := s.lockedChannel
	if current, err := appmodel.CacheGetChannel(s.lockedChannelID); err == nil && current != nil {
		channel = current
	}
	if channel != nil {
		info.ChannelOtherSettings = channel.GetOtherSettings()
		service.ResetStatusCode(failure, channel.GetStatusCodeMapping())
	}
	clientErr := s.clientErrorForChannelFailure(c, channel, info, failure)
	clientErr.SetMessage(common.MessageWithRequestId(clientErr.Error(), c.GetString(common.RequestIdKey)))
	var payload []byte
	var err error
	switch event.Type {
	case "error":
		wireError := clientErr.ToOpenAIError()
		rejection.Status, rejection.Error = clientErr.StatusCode, &wireError
		payload, err = common.Marshal(rejection)
	case "response.failed", "response.error", "response.incomplete", "response.completed", "response.done":
		payload, err = buildResponsesWSTerminalErrorPayload(incoming.body, upstreamModel, info.ClientModelName(), clientErr.ToOpenAIError())
	default:
		payload, err = buildResponsesWSErrorPayload("", "", clientErr)
	}
	if err != nil {
		return err
	}
	if err = s.writeClient(websocket.TextMessage, payload); err != nil {
		return err
	}
	if incoming.err != nil {
		return incoming.err
	}
	return parseErr
}
