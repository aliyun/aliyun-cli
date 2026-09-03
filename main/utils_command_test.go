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
	assert.False(t, utils.GetSubCommand("list-supported-pricing-apis").Hidden)
	assert.False(t, utils.GetSubCommand("mcp-proxy").Hidden)
	assert.False(t, utils.GetSubCommand("go-migrate").Hidden)
	assert.Equal(t,
		"aliyun utils list-supported-pricing-apis --product Ecs\n  aliyun ecs RunInstances --InstanceType ecs.e-c1m1.large ... --estimate-cost",
		utils.GetSubCommand("list-supported-pricing-apis").Sample,
	)
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

	canonicalMCP := utils.GetSubCommand("mcp-proxy")
	legacyMCP := root.GetSubCommand("mcp-proxy")
	assert.Regexp(t, `^aliyun utils mcp-proxy(?: |$)`, canonicalMCP.GetUsageWithParent())
	assert.Regexp(t, `^aliyun mcp-proxy(?: |$)`, legacyMCP.GetUsageWithParent())
	assert.Equal(t, "aliyun utils mcp-proxy --region-type CN --port 8088", canonicalMCP.Sample)
}
