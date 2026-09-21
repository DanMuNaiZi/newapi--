package dto

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUpstreamErrorDisplaySettingsUsesSafeDefaults(t *testing.T) {
	settings := NormalizeUpstreamErrorDisplaySettings(nil)

	assert.False(t, settings.ShowDetails)
	assert.Equal(t, http.StatusServiceUnavailable, settings.StatusCode)
	assert.Equal(t, DefaultUpstreamErrorDisplayMessage, settings.Message)
}

func TestNormalizeUpstreamErrorDisplaySettingsRejectsPersistedInvalidValues(t *testing.T) {
	settings := NormalizeUpstreamErrorDisplaySettings(&UpstreamErrorDisplaySettings{
		ShowDetails: true,
		StatusCode:  http.StatusOK,
		Message:     strings.Repeat("x", MaxUpstreamErrorDisplayMessageRunes+1),
	})

	assert.True(t, settings.ShowDetails)
	assert.Equal(t, http.StatusServiceUnavailable, settings.StatusCode)
	assert.Equal(t, DefaultUpstreamErrorDisplayMessage, settings.Message)
}

func TestUpstreamErrorDisplaySettingsValidate(t *testing.T) {
	tests := []struct {
		name    string
		value   UpstreamErrorDisplaySettings
		wantErr string
	}{
		{
			name: "valid",
			value: UpstreamErrorDisplaySettings{
				StatusCode: http.StatusBadGateway,
				Message:    "temporary upstream failure",
			},
		},
		{
			name: "status below range",
			value: UpstreamErrorDisplaySettings{
				StatusCode: 399,
				Message:    "temporary upstream failure",
			},
			wantErr: "status_code",
		},
		{
			name: "empty message",
			value: UpstreamErrorDisplaySettings{
				StatusCode: http.StatusServiceUnavailable,
				Message:    "   ",
			},
			wantErr: "message",
		},
		{
			name: "message too long",
			value: UpstreamErrorDisplaySettings{
				StatusCode: http.StatusServiceUnavailable,
				Message:    strings.Repeat("界", MaxUpstreamErrorDisplayMessageRunes+1),
			},
			wantErr: "message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.value.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestChannelErrorDisplayRuleValidate(t *testing.T) {
	validModes := []ChannelErrorDisplayMode{
		ChannelErrorDisplayModeInherit,
		ChannelErrorDisplayModeGeneric,
		ChannelErrorDisplayModeOriginal,
	}
	for _, mode := range validModes {
		t.Run(string(mode), func(t *testing.T) {
			require.NoError(t, (ChannelErrorDisplayRule{Mode: mode}).Validate())
		})
	}

	require.NoError(t, (ChannelErrorDisplayRule{
		Mode:       ChannelErrorDisplayModeCustom,
		StatusCode: http.StatusConflict,
		Message:    "custom message",
	}).Validate())

	invalidRules := []ChannelErrorDisplayRule{
		{Mode: "invalid"},
		{Mode: ChannelErrorDisplayModeCustom, StatusCode: 399, Message: "custom message"},
		{Mode: ChannelErrorDisplayModeCustom, StatusCode: http.StatusBadRequest, Message: ""},
		{Mode: ChannelErrorDisplayModeCustom, StatusCode: http.StatusBadRequest, Message: strings.Repeat("x", MaxUpstreamErrorDisplayMessageRunes+1)},
	}
	for _, rule := range invalidRules {
		require.Error(t, rule.Validate())
	}
}
