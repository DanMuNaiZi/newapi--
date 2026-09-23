package service

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func manualWechatPNGDataURL(t *testing.T, width int, height int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buffer bytes.Buffer
	require.NoError(t, png.Encode(&buffer, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func manualWechatJPEGDataURL(t *testing.T, width int, height int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buffer bytes.Buffer
	require.NoError(t, jpeg.Encode(&buffer, img, nil))
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func manualWechatWebPDataURL() string {
	// 256x256 lossless WebP containing a single solid color.
	return "data:image/webp;base64,UklGRigAAABXRUJQVlA4TBwAAAAv/8A/AAdQgVQIIAAKmv7HAFCk//8pov+p//0H"
}

func TestValidateManualWechatPaymentSettingsAcceptsSupportedImage(t *testing.T) {
	testCases := []struct {
		name    string
		dataURL string
	}{
		{name: "png", dataURL: manualWechatPNGDataURL(t, 256, 256)},
		{name: "jpeg", dataURL: manualWechatJPEGDataURL(t, 256, 256)},
		{name: "webp", dataURL: manualWechatWebPDataURL()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			settings := ManualWechatPaymentSettings{
				Enabled:      true,
				QRCodeImage:  testCase.dataURL,
				ExpiresAt:    2_000,
				Instructions: "请填写订单号",
			}

			require.NoError(t, ValidateManualWechatPaymentSettings(settings, 1_000))
		})
	}
}

func TestValidateManualWechatPaymentSettingsRejectsInvalidEnabledSettings(t *testing.T) {
	testCases := []struct {
		name     string
		settings ManualWechatPaymentSettings
		contains string
	}{
		{
			name:     "missing image",
			settings: ManualWechatPaymentSettings{Enabled: true},
			contains: "二维码",
		},
		{
			name: "expired image",
			settings: ManualWechatPaymentSettings{
				Enabled:     true,
				QRCodeImage: manualWechatPNGDataURL(t, 256, 256),
				ExpiresAt:   999,
			},
			contains: "过期",
		},
		{
			name: "unsupported mime",
			settings: ManualWechatPaymentSettings{
				Enabled:     true,
				QRCodeImage: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte("<svg/>")),
			},
			contains: "格式",
		},
		{
			name: "forged mime",
			settings: ManualWechatPaymentSettings{
				Enabled:     true,
				QRCodeImage: strings.Replace(manualWechatPNGDataURL(t, 256, 256), "data:image/png", "data:image/jpeg", 1),
			},
			contains: "实际格式",
		},
		{
			name: "damaged image",
			settings: ManualWechatPaymentSettings{
				Enabled:     true,
				QRCodeImage: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not an image")),
			},
			contains: "无法解析",
		},
		{
			name: "decoded image too large",
			settings: ManualWechatPaymentSettings{
				Enabled:     true,
				QRCodeImage: "data:image/png;base64," + base64.StdEncoding.EncodeToString(make([]byte, ManualWechatQRCodeMaxBytes+1)),
			},
			contains: "512 KiB",
		},
		{
			name: "dimensions too small",
			settings: ManualWechatPaymentSettings{
				Enabled:     true,
				QRCodeImage: manualWechatPNGDataURL(t, 128, 128),
			},
			contains: "尺寸",
		},
		{
			name: "dimensions too large",
			settings: ManualWechatPaymentSettings{
				Enabled:     true,
				QRCodeImage: manualWechatPNGDataURL(t, ManualWechatQRCodeMaxDimension+1, 256),
			},
			contains: "尺寸",
		},
		{
			name: "instructions too long",
			settings: ManualWechatPaymentSettings{
				Enabled:      true,
				QRCodeImage:  manualWechatPNGDataURL(t, 256, 256),
				Instructions: strings.Repeat("a", ManualWechatInstructionsMaxRunes+1),
			},
			contains: "说明",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateManualWechatPaymentSettings(testCase.settings, 1_000)
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.contains)
		})
	}
}

func TestValidateManualWechatPaymentSettingsAllowsDisabledEmptyConfiguration(t *testing.T) {
	require.NoError(t, ValidateManualWechatPaymentSettings(ManualWechatPaymentSettings{}, 1_000))
}
