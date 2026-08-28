package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUtilsCommandsCreateCanonicalTreeAndCompatibleAliases(t *testing.T) {
	utils, aliases := newUtilsCommands()
	require.NotNil(t, utils)
	assert.Equal(t, "utils", utils.Name)

	wantNames := []string{"list-supported-pricing-apis", "mcp-proxy", "go-migrate"}
	assert.Equal(t, wantNames, utils.SubCommandNames())
	require.Len(t, aliases, len(wantNames))

	for i, name := range wantNames {
		canonical := utils.GetSubCommand(name)
		legacy := aliases[i]
		require.NotNil(t, canonical)
		require.NotNil(t, legacy)
		assert.Equal(t, name, legacy.Name)
		assert.True(t, legacy.Hidden)
		assert.NotSame(t, canonical, legacy, "one command instance cannot have two parents")
		assert.Equal(t, canonical.Flags().AvailableFlagNames(), legacy.Flags().AvailableFlagNames())
		if canonical.Run != nil && legacy.Run != nil {
			assert.Equal(t, reflect.ValueOf(canonical.Run).Pointer(), reflect.ValueOf(legacy.Run).Pointer(), "new and compatible paths must share the same handler implementation")
		}
	}
}

func TestRootCommandRegistersOnlyCanonicalUtilsPathVisibly(t *testing.T) {
	root := newRootCommand(config.NewProfile("default"), &bytes.Buffer{})
	utils := root.GetSubCommand("utils")
	require.NotNil(t, utils)

	for _, name := range []string{"list-supported-pricing-apis", "mcp-proxy", "go-migrate"} {
		require.NotNil(t, utils.GetSubCommand(name), "missing canonical utility %s", name)
		legacy := root.GetSubCommand(name)
		require.NotNil(t, legacy, "missing compatible root utility %s", name)
		assert.True(t, legacy.Hidden, "compatible root utility must not be repeated in Root Help")
	}
}
