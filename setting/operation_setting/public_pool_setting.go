package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type PublicPoolSetting struct {
	Enabled bool `json:"enabled"`
}

var publicPoolSetting = PublicPoolSetting{}

func init() {
	config.GlobalConfig.Register("public_pool_setting", &publicPoolSetting)
}

func GetPublicPoolSetting() *PublicPoolSetting {
	return &publicPoolSetting
}
