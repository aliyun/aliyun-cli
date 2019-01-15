package i18n

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnvironment(t *testing.T) {

	assert.Equal(t, "en", GetLanguage())
	SetLanguage("zh")
	assert.Equal(t, "zh", GetLanguage())

}
