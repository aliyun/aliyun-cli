package releasever

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVersionComponentsMustBeNonempty(t *testing.T) {
	for _, version := range []string{"v1..3", "v.2.3", "v1.2."} {
		require.False(t, hasMajorMinorPatch(version))
	}
	require.True(t, hasMajorMinorPatch("v1.2.3-rc.1+build"))
}
