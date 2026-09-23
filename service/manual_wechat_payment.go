package service

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	ManualWechatQRCodeMaxBytes       = 512 * 1024
	ManualWechatQRCodeMinDimension   = 256
	ManualWechatQRCodeMaxDimension   = 4096
	ManualWechatInstructionsMaxRunes = 500
	DefaultManualWechatInstructions  = "请使用微信扫码支付，并在付款备注中填写订单号"
	ManualWechatPaymentDisplayName   = "微信扫码支付（人工确认）"
)

type ManualWechatPaymentSettings struct {
	Enabled      bool   `json:"enabled"`
	QRCodeImage  string `json:"qr_code_image"`
	ExpiresAt    int64  `json:"expires_at"`
	Instructions string `json:"instructions"`
}

func GetManualWechatPaymentSettings() ManualWechatPaymentSettings {
	settings := operation_setting.GetPaymentSetting()
	instructions := strings.TrimSpace(settings.ManualWechatInstructions)
	if instructions == "" {
		instructions = DefaultManualWechatInstructions
	}
	return ManualWechatPaymentSettings{
		Enabled:      settings.ManualWechatEnabled,
		QRCodeImage:  strings.TrimSpace(settings.ManualWechatQRCodeImage),
		ExpiresAt:    settings.ManualWechatExpiresAt,
		Instructions: instructions,
	}
}

func IsManualWechatPaymentAvailable(settings ManualWechatPaymentSettings, now int64) bool {
	return settings.Enabled && settings.QRCodeImage != "" && (settings.ExpiresAt == 0 || settings.ExpiresAt > now)
}

func ValidateManualWechatPaymentSettings(settings ManualWechatPaymentSettings, now int64) error {
	settings.QRCodeImage = strings.TrimSpace(settings.QRCodeImage)
	settings.Instructions = strings.TrimSpace(settings.Instructions)
	if utf8.RuneCountInString(settings.Instructions) > ManualWechatInstructionsMaxRunes {
		return fmt.Errorf("付款说明不能超过 %d 个字符", ManualWechatInstructionsMaxRunes)
	}
	if settings.ExpiresAt < 0 {
		return fmt.Errorf("二维码过期时间无效")
	}
	if settings.Enabled && settings.QRCodeImage == "" {
		return fmt.Errorf("启用微信人工收款前必须上传二维码")
	}
	if settings.Enabled && settings.ExpiresAt != 0 && settings.ExpiresAt <= now {
		return fmt.Errorf("二维码已经过期")
	}
	if settings.QRCodeImage == "" {
		return nil
	}

	commaIndex := strings.IndexByte(settings.QRCodeImage, ',')
	if commaIndex <= len("data:") || commaIndex == len(settings.QRCodeImage)-1 {
		return fmt.Errorf("二维码必须是有效的 Data URL")
	}
	header := strings.ToLower(strings.TrimSpace(settings.QRCodeImage[:commaIndex]))
	allowedFormats := map[string]string{
		"data:image/png;base64":  "png",
		"data:image/jpeg;base64": "jpeg",
		"data:image/webp;base64": "webp",
	}
	expectedFormat, supported := allowedFormats[header]
	if !supported {
		return fmt.Errorf("二维码格式仅支持 PNG、JPEG 或 WebP")
	}
	encoded := strings.TrimSpace(settings.QRCodeImage[commaIndex+1:])
	if base64.StdEncoding.DecodedLen(len(encoded)) > ManualWechatQRCodeMaxBytes {
		return fmt.Errorf("二维码图片不能超过 512 KiB")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("二维码 Base64 数据无效")
	}
	if len(decoded) > ManualWechatQRCodeMaxBytes {
		return fmt.Errorf("二维码图片不能超过 512 KiB")
	}
	config, actualFormat, _, err := DecodeBase64ImageData(encoded)
	if err != nil {
		return fmt.Errorf("无法解析二维码图片")
	}
	if actualFormat != expectedFormat {
		return fmt.Errorf("二维码声明格式与实际格式不一致")
	}
	if config.Width < ManualWechatQRCodeMinDimension || config.Height < ManualWechatQRCodeMinDimension ||
		config.Width > ManualWechatQRCodeMaxDimension || config.Height > ManualWechatQRCodeMaxDimension {
		return fmt.Errorf("二维码图片尺寸必须在 256 到 4096 像素之间")
	}
	return nil
}
