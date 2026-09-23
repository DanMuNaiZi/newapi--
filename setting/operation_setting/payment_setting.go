package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type PaymentSetting struct {
	AmountOptions            []int           `json:"amount_options"`
	AmountDiscount           map[int]float64 `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠
	ManualWechatEnabled      bool            `json:"manual_wechat_enabled"`
	ManualWechatQRCodeImage  string          `json:"manual_wechat_qr_code_image"`
	ManualWechatExpiresAt    int64           `json:"manual_wechat_expires_at"`
	ManualWechatInstructions string          `json:"manual_wechat_instructions"`

	ComplianceConfirmed    bool   `json:"compliance_confirmed"`
	ComplianceTermsVersion string `json:"compliance_terms_version"`
	ComplianceConfirmedAt  int64  `json:"compliance_confirmed_at"`
	ComplianceConfirmedBy  int    `json:"compliance_confirmed_by"`
	ComplianceConfirmedIP  string `json:"compliance_confirmed_ip"`
}

const CurrentComplianceTermsVersion = "v1"

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:            []int{10, 20, 50, 100, 200, 500},
	AmountDiscount:           map[int]float64{},
	ManualWechatInstructions: "请使用微信扫码支付，并在付款备注中填写订单号",
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}

func IsPaymentComplianceConfirmed() bool {
	return paymentSetting.ComplianceConfirmed &&
		paymentSetting.ComplianceTermsVersion == CurrentComplianceTermsVersion
}
