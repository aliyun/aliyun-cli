package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryGetProductIsCaseInsensitive(t *testing.T) {
	repository, err := MockLoadRepository([]Product{{Code: "ECS"}})
	require.NoError(t, err)
	product, ok := repository.GetProduct("ecs")
	assert.True(t, ok)
	assert.Equal(t, "ECS", product.Code)
	_, ok = repository.GetProduct("missing")
	assert.False(t, ok)
}
