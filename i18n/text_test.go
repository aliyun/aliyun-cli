package i18n

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestText(t *testing.T) {
	language = "en"
	tx := T("hello", "你好")
	assert.Equal(t, "hello", tx.Get("en"))

	tx.Put("pinyin", "nihao")
	assert.Equal(t, "nihao", tx.Get("pinyin"))
	assert.Equal(t, "hello", tx.Text())
}
