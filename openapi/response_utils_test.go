package openapi

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestMergeWith(t *testing.T) {
	inter := mergeWith(nil, nil)
	assert.Nil(t, inter)
}
