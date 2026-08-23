package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSensitiveOptionKeyHidesAPICredentials(t *testing.T) {
	assert.True(t, isSensitiveOptionKey("pinnacle_tool.api_token"))
	assert.True(t, isSensitiveOptionKey("provider.api_key"))
	assert.True(t, isSensitiveOptionKey("SMTPToken"))
	assert.False(t, isSensitiveOptionKey("pinnacle_tool.enabled"))
}
