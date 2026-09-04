package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestGenRelayInfoKeepsClientRequestedModelAfterCompactSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	ctx.Set(string(constant.ContextKeyOriginalModel), "gpt-5.6-sol-openai-compact")
	ctx.Set(string(constant.ContextKeyRequestModelName), "gpt-5.6-sol")

	info := GenRelayInfoOpenAI(ctx, nil)

	require.Equal(t, "gpt-5.6-sol", info.RequestModelName)
	require.Equal(t, "gpt-5.6-sol-openai-compact", info.OriginModelName)
}

func TestMaskMappedModelInClientErrorUsesRequestedModel(t *testing.T) {
	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "auth_unavailable: no auth available (providers=codex, model=gpt-5.6-terra)",
		Type:    "upstream_error",
		Code:    "auth_unavailable",
	}, 503)
	info := &RelayInfo{
		RequestModelName: "codex",
		ChannelMeta: &ChannelMeta{
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
		},
	}

	MaskMappedModelInClientError(info, apiErr)

	require.Equal(t, "auth_unavailable: no auth available (providers=codex, model=codex)", apiErr.Error())
	require.Equal(t, "auth_unavailable: no auth available (providers=codex, model=codex)", apiErr.ToOpenAIError().Message)
}

func TestMaskMappedModelInClientErrorDoesNotReplaceModelPrefix(t *testing.T) {
	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "model=gpt-5.6-terra-mini is unavailable",
		Type:    "upstream_error",
		Code:    "auth_unavailable",
	}, 503)
	info := &RelayInfo{
		RequestModelName: "codex",
		ChannelMeta: &ChannelMeta{
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
		},
	}

	MaskMappedModelInClientError(info, apiErr)

	require.Equal(t, "model=gpt-5.6-terra-mini is unavailable", apiErr.ToOpenAIError().Message)
}

func TestClientVisibleErrorMessageUsesRequestedModelInErrorLog(t *testing.T) {
	info := &RelayInfo{
		RequestModelName: "gpt-5.6-sol",
		ChannelMeta: &ChannelMeta{
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
		},
	}

	content := ClientVisibleErrorMessage(info, "status_code=503, auth_unavailable: no auth available (providers=codex, model=gpt-5.6-terra)")

	require.Equal(t, "status_code=503, auth_unavailable: no auth available (providers=codex, model=gpt-5.6-sol)", content)
}

func TestClientVisibleErrorMessageMasksMappedModelAcrossCommonFormats(t *testing.T) {
	info := &RelayInfo{
		RequestModelName: "gpt-5.6-sol",
		ChannelMeta: &ChannelMeta{
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
		},
	}

	content := ClientVisibleErrorMessage(info, `The model gpt-5.6-terra does not exist; body={"model":"gpt-5.6-terra"}`)

	require.Equal(t, `The model gpt-5.6-sol does not exist; body={"model":"gpt-5.6-sol"}`, content)
}

func TestClientVisibleErrorMessageLeavesUnmappedModelUntouched(t *testing.T) {
	info := &RelayInfo{
		RequestModelName: "gpt-5.6-sol",
		ChannelMeta: &ChannelMeta{
			IsModelMapped:     false,
			UpstreamModelName: "gpt-5.6-terra",
		},
	}

	message := "upstream model=gpt-5.6-terra is unavailable"
	require.Equal(t, message, ClientVisibleErrorMessage(info, message))

	other := make(map[string]interface{})
	AppendMappedModelLogInfo(info, other)
	assert.Empty(t, other)
}
