package model

import (
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func prepareChannelErrorRecordTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&ChannelErrorRecord{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ChannelErrorRecord{}).Error)
}

func baseChannelErrorRecordInput() ChannelErrorRecordInput {
	return ChannelErrorRecordInput{
		ChannelID:          17,
		RequestPath:        "/v1/responses",
		RequestModel:       "gpt-5.6-sol",
		UpstreamStatusCode: http.StatusBadRequest,
		FinalStatusCode:    http.StatusBadRequest,
		ErrorType:          "openai_error",
		ErrorCode:          "unsupported_model",
		SampleMessage:      "The model is not supported for this account.",
		RequestID:          "req-first",
		MultiKeyIndex:      3,
	}
}

func TestUpsertChannelErrorRecordAggregatesMatchingErrors(t *testing.T) {
	prepareChannelErrorRecordTest(t)
	input := baseChannelErrorRecordInput()

	first, err := UpsertChannelErrorRecord(input)
	require.NoError(t, err)
	input.RequestID = "req-second"
	input.FinalStatusCode = http.StatusServiceUnavailable
	second, err := UpsertChannelErrorRecord(input)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.EqualValues(t, 2, second.OccurrenceCount)
	assert.Equal(t, "req-second", second.LastRequestID)
	assert.Equal(t, http.StatusBadRequest, second.UpstreamStatusCode)
	assert.Equal(t, http.StatusServiceUnavailable, second.FinalStatusCode)
	assert.NotEmpty(t, second.Signature)
	assert.Equal(t, second.FirstSeenTime, first.FirstSeenTime)
}

func TestUpsertChannelErrorRecordSeparatesSignatureDimensions(t *testing.T) {
	prepareChannelErrorRecordTest(t)
	base := baseChannelErrorRecordInput()

	variants := []ChannelErrorRecordInput{
		base,
		func() ChannelErrorRecordInput {
			value := base
			value.RequestPath = "/v1/chat/completions"
			return value
		}(),
		func() ChannelErrorRecordInput { value := base; value.RequestModel = "gpt-5.6-terra"; return value }(),
		func() ChannelErrorRecordInput {
			value := base
			value.UpstreamStatusCode = http.StatusUnauthorized
			return value
		}(),
		func() ChannelErrorRecordInput { value := base; value.ErrorCode = "different_code"; return value }(),
		func() ChannelErrorRecordInput { value := base; value.SampleMessage = "different message"; return value }(),
	}

	for _, input := range variants {
		_, err := UpsertChannelErrorRecord(input)
		require.NoError(t, err)
	}

	var count int64
	require.NoError(t, DB.Model(&ChannelErrorRecord{}).Count(&count).Error)
	assert.EqualValues(t, len(variants), count)
}

func TestChannelErrorDisplayRuleSurvivesAggregationAndCanBeFiltered(t *testing.T) {
	prepareChannelErrorRecordTest(t)
	input := baseChannelErrorRecordInput()
	record, err := UpsertChannelErrorRecord(input)
	require.NoError(t, err)

	record, err = UpdateChannelErrorDisplayRule(input.ChannelID, record.ID, dto.ChannelErrorDisplayRule{
		Mode:       dto.ChannelErrorDisplayModeCustom,
		StatusCode: http.StatusConflict,
		Message:    "custom client message",
	})
	require.NoError(t, err)
	input.RequestID = "req-latest"
	record, err = UpsertChannelErrorRecord(input)
	require.NoError(t, err)

	assert.Equal(t, dto.ChannelErrorDisplayModeCustom, record.DisplayMode)
	assert.Equal(t, http.StatusConflict, record.DisplayStatusCode)
	assert.Equal(t, "custom client message", record.DisplayMessage)

	items, total, err := GetChannelErrorRecords(ChannelErrorRecordQuery{
		ChannelID:    input.ChannelID,
		StatusCode:   http.StatusBadRequest,
		RequestModel: "gpt-5.6-sol",
		Keyword:      "supported",
		Offset:       0,
		Limit:        20,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, record.ID, items[0].ID)
}

func TestUpsertChannelErrorRecordLimitsStoredSample(t *testing.T) {
	prepareChannelErrorRecordTest(t)
	input := baseChannelErrorRecordInput()
	input.SampleMessage = strings.Repeat("界", MaxChannelErrorSampleBytes)

	record, err := UpsertChannelErrorRecord(input)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(record.SampleMessage), MaxChannelErrorSampleBytes)
	assert.True(t, utf8.ValidString(record.SampleMessage))
}

func TestDeletingChannelDeletesAggregatedErrors(t *testing.T) {
	prepareChannelErrorRecordTest(t)
	channel := &Channel{Name: "delete-with-errors", Key: "test-key"}
	require.NoError(t, DB.Create(channel).Error)
	input := baseChannelErrorRecordInput()
	input.ChannelID = channel.Id
	_, err := UpsertChannelErrorRecord(input)
	require.NoError(t, err)

	require.NoError(t, channel.Delete())

	var count int64
	require.NoError(t, DB.Model(&ChannelErrorRecord{}).Where("channel_id = ?", channel.Id).Count(&count).Error)
	assert.Zero(t, count)
}

func TestBatchDeletingChannelsDeletesAggregatedErrors(t *testing.T) {
	prepareChannelErrorRecordTest(t)
	channels := []Channel{
		{Name: "batch-delete-with-errors-1", Key: "test-key-1"},
		{Name: "batch-delete-with-errors-2", Key: "test-key-2"},
	}
	require.NoError(t, DB.Create(&channels).Error)
	for _, channel := range channels {
		input := baseChannelErrorRecordInput()
		input.ChannelID = channel.Id
		input.RequestModel = channel.Name
		_, err := UpsertChannelErrorRecord(input)
		require.NoError(t, err)
	}

	require.NoError(t, BatchDeleteChannels([]int{channels[0].Id, channels[1].Id}))

	var count int64
	require.NoError(t, DB.Model(&ChannelErrorRecord{}).
		Where("channel_id IN ?", []int{channels[0].Id, channels[1].Id}).
		Count(&count).Error)
	assert.Zero(t, count)
}

func TestChannelValidateSettingsRejectsInvalidUpstreamErrorDisplay(t *testing.T) {
	channel := &Channel{OtherSettings: `{"upstream_error_display":{"show_details":false,"status_code":200,"message":"invalid"}}`}

	err := channel.ValidateSettings()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status_code")
}
