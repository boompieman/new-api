package operation_setting

import (
	"os"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

const (
	PinnacleToolPricePerThousandUSD = 5.0
	PinnacleToolModelName           = "pinnacle-api"
	PinnacleToolTokenEnv            = "PINNACLE_APIFY_TOKEN"
)

type PinnacleToolSetting struct {
	Enabled  bool   `json:"enabled"`
	APIToken string `json:"api_token"`
}

var pinnacleToolSetting = PinnacleToolSetting{}

func init() {
	config.GlobalConfig.Register("pinnacle_tool", &pinnacleToolSetting)
}

func IsPinnacleToolEnabled() bool {
	return pinnacleToolSetting.Enabled && GetPinnacleToolAPIToken() != ""
}

func IsPinnacleToolTokenConfigured() bool {
	return GetPinnacleToolAPIToken() != ""
}

func GetPinnacleToolAPIToken() string {
	if token := strings.TrimSpace(os.Getenv(PinnacleToolTokenEnv)); token != "" {
		return token
	}
	return strings.TrimSpace(pinnacleToolSetting.APIToken)
}

func GetPinnacleToolSetting() PinnacleToolSetting {
	return PinnacleToolSetting{
		Enabled: pinnacleToolSetting.Enabled,
	}
}
