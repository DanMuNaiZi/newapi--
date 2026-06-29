package common

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
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

func TestRelayInfoResponseModelNamePrefersRequestModel(t *testing.T) {
	info := &RelayInfo{
		RequestModelName: "gpt-5.5",
		OriginModelName:  "gpt-5.5-compact",
		ChannelMeta:      &ChannelMeta{UpstreamModelName: "gpt-5.4"},
	}

	require.Equal(t, "gpt-5.5", info.ResponseModelName())
}

func TestRelayInfoResponseModelNameFallsBack(t *testing.T) {
	require.Equal(t, "", (*RelayInfo)(nil).ResponseModelName())
	require.Equal(t, "origin", (&RelayInfo{OriginModelName: "origin"}).ResponseModelName())
	require.Equal(t, "upstream", (&RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: "upstream"}}).ResponseModelName())
}
