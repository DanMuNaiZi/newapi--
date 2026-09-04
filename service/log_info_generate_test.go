package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGenerateTextOtherInfoRecordsClientAndUpstreamMappedModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	startTime := time.Unix(100, 0)
	relayInfo := &relaycommon.RelayInfo{
		RequestModelName:  "gpt-5.6-sol",
		OriginModelName:   "gpt-5.6-terra-openai-compact",
		StartTime:         startTime,
		FirstResponseTime: startTime.Add(time.Second),
		ChannelMeta: &relaycommon.ChannelMeta{
			IsModelMapped:     true,
			UpstreamModelName: "gpt-5.6-terra",
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 1, 1, 1, 0, 0, 0, 1)

	require.Equal(t, "gpt-5.6-sol", other["request_model_name"])
	require.Equal(t, "gpt-5.6-terra", other["upstream_model_name"])
	require.Equal(t, true, other["is_model_mapped"])
}
