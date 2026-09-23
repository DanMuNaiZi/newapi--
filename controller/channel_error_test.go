package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type channelErrorAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    common.PageInfo `json:"data"`
}

func setupChannelErrorControllerTest(t *testing.T) (*model.Channel, *model.ChannelErrorRecord) {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ChannelErrorRecord{}))
	channel := &model.Channel{Name: "channel-error-api", Key: "test-key", OtherSettings: "{}"}
	require.NoError(t, db.Create(channel).Error)
	record, err := model.UpsertChannelErrorRecord(model.ChannelErrorRecordInput{
		ChannelID:          channel.Id,
		RequestPath:        "/v1/responses",
		RequestModel:       "gpt-5.6-sol",
		UpstreamStatusCode: http.StatusBadRequest,
		FinalStatusCode:    http.StatusBadGateway,
		ErrorType:          "openai_error",
		ErrorCode:          "unsupported_model",
		SampleMessage:      "model is not supported",
		RequestID:          "req-api",
	})
	require.NoError(t, err)
	return channel, record
}

func TestUpdateChannelErrorDisplayPersistsValidatedDefaults(t *testing.T) {
	channel, _ := setupChannelErrorControllerTest(t)
	body := []byte(`{"show_details":true,"status_code":502,"message":"sanitized upstream error"}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/channel/1/error-display", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(channel.Id)}}

	UpdateChannelErrorDisplay(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool                             `json:"success"`
		Data    dto.UpstreamErrorDisplaySettings `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.True(t, response.Data.ShowDetails)
	assert.Equal(t, http.StatusBadGateway, response.Data.StatusCode)

	reloaded, err := model.GetChannelById(channel.Id, false)
	require.NoError(t, err)
	settings := reloaded.GetOtherSettings()
	require.NotNil(t, settings.UpstreamErrorDisplay)
	assert.Equal(t, "sanitized upstream error", settings.UpstreamErrorDisplay.Message)
}

func TestUpdateChannelErrorDisplayRejectsInvalidStatus(t *testing.T) {
	channel, _ := setupChannelErrorControllerTest(t)
	body := []byte(`{"show_details":false,"status_code":200,"message":"bad"}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/channel/1/error-display", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(channel.Id)}}

	UpdateChannelErrorDisplay(c)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "status_code")
}

func TestGetChannelErrorsSupportsDocumentedFiltersAndPageParameter(t *testing.T) {
	channel, record := setupChannelErrorControllerTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/channel/1/errors?page=1&page_size=10&status_code=400&request_model=gpt-5.6-sol&keyword=supported", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(channel.Id)}}

	GetChannelErrors(c)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Page     int                        `json:"page"`
			PageSize int                        `json:"page_size"`
			Total    int                        `json:"total"`
			Items    []model.ChannelErrorRecord `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 1, response.Data.Page)
	assert.Equal(t, 10, response.Data.PageSize)
	assert.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, record.ID, response.Data.Items[0].ID)
}

func TestUpdateChannelErrorRecordDisplayPersistsCustomRule(t *testing.T) {
	channel, record := setupChannelErrorControllerTest(t)
	body := []byte(`{"display_mode":"custom","display_status_code":409,"display_message":"account model mismatch"}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/channel/1/errors/1/display", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(channel.Id)},
		{Key: "error_id", Value: strconv.FormatUint(uint64(record.ID), 10)},
	}

	UpdateChannelErrorRecordDisplay(c)

	var response struct {
		Success bool                     `json:"success"`
		Data    model.ChannelErrorRecord `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, dto.ChannelErrorDisplayModeCustom, response.Data.DisplayMode)
	assert.Equal(t, http.StatusConflict, response.Data.DisplayStatusCode)
	assert.Equal(t, "account model mismatch", response.Data.DisplayMessage)
}
