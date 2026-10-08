package service

import (
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
)

var (
	upstreamResponseBodyPattern     = regexp.MustCompile(`(?is)["']?\b(?:(?:request|response|raw)[_ -]?)?body\b["']?\s*[:=]\s*.*$`)
	upstreamTrailingMetadataPattern = regexp.MustCompile(`(?s)\s+\((?:\{.*\}|\[.*\])\)\s*$`)
	upstreamBearerPattern           = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`)
	upstreamCredentialPattern       = regexp.MustCompile(`(?i)((?:["']?)\b(?:api[_ -]?key|access[_ -]?token|refresh[_ -]?token|session[_ -]?token|(?:proxy[_ -]?)?authorization|client[_ -]?secret|private[_ -]?key|password|secret|token)\b(?:["']?)\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;\}\]]+)`)
)

func SanitizeUpstreamErrorMessage(message string, secrets ...string) string {
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if len(secret) < 4 {
			continue
		}
		message = strings.ReplaceAll(message, secret, "***")
	}
	message = upstreamTrailingMetadataPattern.ReplaceAllString(message, " [metadata omitted]")
	message = upstreamResponseBodyPattern.ReplaceAllString(message, "body: [omitted]")
	message = upstreamBearerPattern.ReplaceAllString(message, "Bearer ***")
	message = upstreamCredentialPattern.ReplaceAllString(message, `${1}***`)
	return strings.TrimSpace(common.MaskSensitiveInfo(message))
}

// SanitizeTaskFailureReason projects a persisted failure for task APIs without
// changing task state or applying the synchronous relay's display policy.
func SanitizeTaskFailureReason(task *model.Task) string {
	if task == nil {
		return ""
	}
	requestedModel := task.Properties.OriginModelName
	if requestedModel == "" {
		requestedModel = "requested model"
	}
	info := &relaycommon.RelayInfo{
		RequestModelName: requestedModel,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: task.Properties.UpstreamModelName,
			IsModelMapped:     task.Properties.UpstreamModelName != "" && task.Properties.UpstreamModelName != requestedModel,
		},
	}
	message := relaycommon.ClientVisibleErrorMessage(info, task.FailReason)
	return SanitizeUpstreamErrorMessage(message, task.PrivateData.Key)
}

// SanitizeClientUpstreamError returns an independent error suitable for a
// client response. Retry, billing, channel health and audit callers retain the
// original provider error, including its status and structured details.
func SanitizeClientUpstreamError(info *relaycommon.RelayInfo, upstreamErr *types.NewAPIError, secrets ...string) *types.NewAPIError {
	if upstreamErr == nil {
		return nil
	}
	clientErr := upstreamErr.ClientCopy()
	if info != nil && info.ChannelMeta != nil {
		secrets = append(secrets, info.ApiKey)
	}
	message := relaycommon.ClientVisibleErrorMessage(info, clientErr.Error())
	clientErr.SetMessage(SanitizeUpstreamErrorMessage(message, secrets...))
	clientErr.ClearMetadata()
	if clientErr.GetErrorType() != types.ErrorTypeOpenAIError && clientErr.GetErrorType() != types.ErrorTypeClaudeError {
		wireType := SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, string(clientErr.GetErrorType())), secrets...)
		code := SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, string(clientErr.GetErrorCode())), secrets...)
		clientErr.SetOpenAIClientError(clientErr.Error(), wireType, types.ErrorCode(code), clientErr.StatusCode)
		return clientErr
	}
	switch wireError := clientErr.RelayError.(type) {
	case types.OpenAIError:
		wireError.Type = SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, wireError.Type), secrets...)
		wireError.Param = SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, wireError.Param), secrets...)
		if wireError.Code != nil {
			encoded, err := common.Marshal(wireError.Code)
			if err == nil {
				encoded, err = sanitizeUpstreamErrorDetail(encoded, info, secrets)
			}
			if err != nil {
				wireError.Code = nil
			} else if common.GetJsonType(encoded) == "string" {
				var code string
				if common.Unmarshal(encoded, &code) == nil {
					wireError.Code = code
				} else {
					wireError.Code = nil
				}
			} else {
				wireError.Code = common.RawMessage(encoded)
			}
		}
		clientErr.RelayError = wireError
	case types.ClaudeError:
		wireError.Type = SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, wireError.Type), secrets...)
		clientErr.SetClaudeClientError(clientErr.Error(), wireError.Type, types.ErrorCode(wireError.Type), clientErr.StatusCode)
	}
	return clientErr
}

// ChannelErrorAuditLabels preserves safe provider codes for aggregation and
// logs while respecting the shared channel_error_records column limits.
func ChannelErrorAuditLabels(info *relaycommon.RelayInfo, upstreamErr *types.NewAPIError, secrets ...string) (errorType, errorCode string) {
	if upstreamErr == nil {
		return "", ""
	}
	if info != nil && info.ChannelMeta != nil {
		secrets = append(secrets, info.ApiKey)
	}
	clientErr := SanitizeClientUpstreamError(info, upstreamErr, secrets...)
	errorType = string(upstreamErr.GetErrorType())
	errorCode = SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, string(upstreamErr.GetErrorCode())), secrets...)
	if upstreamErr.GetErrorType() == types.ErrorTypeOpenAIError {
		if wireError, ok := clientErr.RelayError.(types.OpenAIError); ok && wireError.Code != nil {
			if encoded, err := common.Marshal(wireError.Code); err == nil {
				errorCode = common.JsonRawMessageToString(encoded)
			}
		}
	} else if upstreamErr.GetErrorType() == types.ErrorTypeClaudeError {
		if wireError, ok := clientErr.RelayError.(types.ClaudeError); ok {
			errorCode = wireError.Type
		}
	}
	typeRunes, codeRunes := []rune(errorType), []rune(errorCode)
	return string(typeRunes[:min(len(typeRunes), 128)]), string(codeRunes[:min(len(codeRunes), 256)])
}

// Structured error codes can carry the same private response bodies, headers
// and credentials as message strings. Preserve safe diagnostic values and
// numeric precision while omitting private envelopes at every nesting level.
func sanitizeUpstreamErrorDetail(value common.RawMessage, info *relaycommon.RelayInfo, secrets []string) (common.RawMessage, error) {
	switch common.GetJsonType(value) {
	case "object":
		var fields map[string]common.RawMessage
		if err := common.Unmarshal(value, &fields); err != nil {
			return nil, err
		}
		cleaned := make(map[string]common.RawMessage, len(fields))
		for key, detail := range fields {
			field := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
			switch field {
			case "body", "rawbody", "requestbody", "responsebody", "metadata", "headers", "requestheaders", "responseheaders":
				continue
			case "apikey", "accesstoken", "refreshtoken", "authorization", "proxyauthorization", "clientsecret", "secret", "token", "password", "privatekey", "sessiontoken":
				detail = common.RawMessage(`"***"`)
			default:
				var err error
				detail, err = sanitizeUpstreamErrorDetail(detail, info, secrets)
				if err != nil {
					return nil, err
				}
			}
			key = SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, key), secrets...)
			cleaned[key] = detail
		}
		return common.Marshal(cleaned)
	case "array":
		var items []common.RawMessage
		if err := common.Unmarshal(value, &items); err != nil {
			return nil, err
		}
		for i := range items {
			var err error
			items[i], err = sanitizeUpstreamErrorDetail(items[i], info, secrets)
			if err != nil {
				return nil, err
			}
		}
		return common.Marshal(items)
	case "string":
		var text string
		if err := common.Unmarshal(value, &text); err != nil {
			return nil, err
		}
		text = SanitizeUpstreamErrorMessage(relaycommon.ClientVisibleErrorMessage(info, text), secrets...)
		return common.Marshal(text)
	default:
		return value, nil
	}
}

func NormalizeChannelErrorMessage(message string) string {
	return common.NormalizeChannelErrorMessage(message)
}
