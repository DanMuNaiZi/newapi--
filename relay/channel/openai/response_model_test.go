package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestRewriteResponseModelInJSONBytes(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RequestModelName: "gpt-5.5",
		ChannelMeta:      &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.4"},
	}

	body := rewriteResponseModelInJSONBytes([]byte(`{"id":"1","model":"gpt-5.4","usage":{"total_tokens":1}}`), info)

	var parsed map[string]interface{}
	require.NoError(t, common.Unmarshal(body, &parsed))
	require.Equal(t, "gpt-5.5", parsed["model"])
	require.Contains(t, parsed, "usage")
}

func TestRewriteResponseModelInJSONBytesNestedResponsesModel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RequestModelName: "gpt-5.5",
		ChannelMeta:      &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.4"},
	}

	body := rewriteResponseModelInJSONBytes([]byte(`{"type":"response.completed","response":{"id":"resp_1","model":"gpt-5.4"}}`), info)

	var parsed map[string]interface{}
	require.NoError(t, common.Unmarshal(body, &parsed))
	response, ok := parsed["response"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "gpt-5.5", response["model"])
}
