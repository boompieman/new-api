package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPinnacleToolUsesEnvironmentTokenWithoutExposingItInSettings(t *testing.T) {
	previous := pinnacleToolSetting
	t.Cleanup(func() {
		pinnacleToolSetting = previous
	})
	pinnacleToolSetting = PinnacleToolSetting{Enabled: true, APIToken: "database-token"}
	t.Setenv(PinnacleToolTokenEnv, "environment-token")

	assert.Equal(t, "environment-token", GetPinnacleToolAPIToken())
	assert.True(t, IsPinnacleToolEnabled())
	assert.Equal(t, PinnacleToolSetting{Enabled: true}, GetPinnacleToolSetting())
}

func TestPinnacleToolRequiresBothEnabledFlagAndToken(t *testing.T) {
	previous := pinnacleToolSetting
	t.Cleanup(func() {
		pinnacleToolSetting = previous
	})
	t.Setenv(PinnacleToolTokenEnv, "")

	pinnacleToolSetting = PinnacleToolSetting{Enabled: false, APIToken: "database-token"}
	assert.False(t, IsPinnacleToolEnabled())
	assert.True(t, IsPinnacleToolTokenConfigured())

	pinnacleToolSetting = PinnacleToolSetting{Enabled: true}
	assert.False(t, IsPinnacleToolEnabled())
	assert.False(t, IsPinnacleToolTokenConfigured())
}
