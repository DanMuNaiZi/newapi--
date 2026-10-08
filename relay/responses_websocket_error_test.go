package relay

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/require"
)

func TestResponsesWSClientErrorAppliesChannelDisplayPolicyAndMappedModel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RequestModelName: "codex",
		OriginModelName:  "gpt-5.6-terra",
		ChannelMeta: &relaycommon.ChannelMeta{
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
			ApiKey:            "sk-channel-secret",
		},
	}
	upstream := types.WithOpenAIError(types.OpenAIError{
		Message:  "The 'gpt-5.6-terra' model is not supported for sk-channel-secret",
		Type:     "invalid_request_error Authorization: Bearer type-secret",
		Param:    "gpt-5.6-terra token=param-secret",
		Code:     "unsupported_model",
		Metadata: []byte(`{"body":"raw-private-body"}`),
	}, http.StatusBadRequest)
	upstreamMessage := upstream.Error()

	generic := responsesWSClientError(info, upstream, dto.NormalizeUpstreamErrorDisplaySettings(nil), nil)
	require.Equal(t, http.StatusServiceUnavailable, generic.StatusCode)
	require.Equal(t, dto.DefaultUpstreamErrorDisplayMessage, generic.Error())
	require.NotContains(t, generic.Error(), "gpt-5.6-terra")

	original := responsesWSClientError(info, upstream, dto.NormalizeUpstreamErrorDisplaySettings(nil), &model.ChannelErrorRecord{
		DisplayMode: dto.ChannelErrorDisplayModeOriginal,
	})
	require.Equal(t, http.StatusBadRequest, original.StatusCode)
	require.Contains(t, original.Error(), "codex")
	require.NotContains(t, original.Error(), "gpt-5.6-terra")
	require.NotContains(t, original.Error(), "sk-channel-secret")
	originalWire, err := common.Marshal(original.ToOpenAIError())
	require.NoError(t, err)
	require.NotContains(t, string(originalWire), "gpt-5.6-terra")
	require.NotContains(t, string(originalWire), "type-secret")
	require.NotContains(t, string(originalWire), "param-secret")
	require.NotContains(t, string(originalWire), "raw-private-body")
	require.Equal(t, "unsupported_model", original.ToOpenAIError().Code)
	require.Equal(t, upstreamMessage, upstream.Error())

	custom := responsesWSClientError(info, upstream, dto.NormalizeUpstreamErrorDisplaySettings(nil), &model.ChannelErrorRecord{
		DisplayMode:       dto.ChannelErrorDisplayModeCustom,
		DisplayStatusCode: http.StatusConflict,
		DisplayMessage:    "当前账号不支持该模型",
	})
	require.Equal(t, http.StatusConflict, custom.StatusCode)
	require.Equal(t, "当前账号不支持该模型", custom.Error())
}
