package i18n

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLibrary(t *testing.T) {
	//Test T(en string, zh string)*Text
	text := T("hello", "你好")
	assert.Equal(t, "", text.id)
	assert.Equal(t, "hello", text.dic["en"])
	assert.Equal(t, "你好", text.dic["zh"])

	text = T("", "你好")
	assert.Equal(t, "", text.id)
	assert.Equal(t, "", text.dic["en"])
	assert.Equal(t, "你好", text.dic["zh"])

	text = T("hello", "")
	assert.Equal(t, "", text.id)
	assert.Equal(t, "hello", text.dic["en"])
	assert.Equal(t, "", text.dic["zh"])

	//Test others
	tx, ok := getText("2", "zh")
	assert.False(t, ok)
	assert.Equal(t, "", tx)

	text = putText("1", "zh", "你好")
	assert.Equal(t, "1", text.id)
	assert.Equal(t, "", text.dic["en"])
	assert.Equal(t, "你好", text.dic["zh"])

	tx, ok = getText("1", "zh")
	assert.True(t, ok)
	assert.Equal(t, "你好", tx)
}
