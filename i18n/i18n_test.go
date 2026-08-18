package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLanguageUsesTraditionalChinese(t *testing.T) {
	require.NoError(t, Init())

	assert.Equal(t, LangZhTW, ParseAcceptLanguage(""))
	assert.Equal(t, "無效的參數", Translate("", MsgInvalidParams))
}
