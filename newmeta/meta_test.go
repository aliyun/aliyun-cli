package newmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetadataPrefix(t *testing.T) {
	assert.Equal(t, "zh-CN", GetMetadataPrefix(""))
	assert.Equal(t, "en-US", GetMetadataPrefix("en"))
	assert.Equal(t, "zh-CN", GetMetadataPrefix("zh"))
}

func TestGetMetadata(t *testing.T) {
	content, err := GetMetadata("en", "/products.json")
	assert.Nil(t, err)
	assert.Greater(t, len(content), 100)
}
