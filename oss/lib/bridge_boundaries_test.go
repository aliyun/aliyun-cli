package lib

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestBridgeParserBoundaries(t *testing.T) {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.SetCommand(NewCommandBridge(listCommand.command))
	for _, args := range [][]string{{"--recursive=true"}, {"--endpoint"}, {"--endpoint", "--cli-output=json"}} {
		_, _, err := parseBridgeFlags(ctx, args)
		require.Error(t, err)
	}
	args, help, err := parseBridgeFlags(ctx, []string{"--", "-literal", "--endpoint"})
	require.NoError(t, err)
	require.False(t, help)
	require.Equal(t, []string{"-literal", "--endpoint"}, args)
	require.Equal(t, []string{"oss://bucket", "-literal"}, ossPositionalArgs(ctx, []string{"--endpoint=x", "-abc", "oss://bucket", "--", "-literal"}))
	require.Equal(t, []string{"--", "--profile", "keep"}, stripOptionWithValue([]string{"--profile=a", "--", "--profile", "keep"}, "--profile"))
	value, found := getArgValue([]string{"--", "--endpoint=ignored"}, "--endpoint")
	require.False(t, found)
	require.Empty(t, value)
	require.ErrorContains(t, validateCommandArgCount(&Command{name: "test", maxArgc: 2}, []string{"1", "2", "3"}), "at most 2 arguments")
}

func TestBridgeHostFlagsAndMissingConfiguration(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, name := range []string{"dryrun", "cli-dry-run", "cli-dry-run-json"} {
		ctx := cli.NewCommandContext(io.Discard, io.Discard)
		ctx.SetCommand(NewCommandBridge(listCommand.command))
		ctx.Flags().Add(&cli.Flag{Name: name, AssignedMode: cli.AssignedNone})
		require.ErrorContains(t, parseAndRunCommandFromCli(ctx, []string{"--" + name}, &listCommand.command), "not supported by built-in OSS")
	}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.SetCommand(NewCommandBridge(listCommand.command))
	ctx.Flags().Get("config-path").SetValue(filepath.Join(t.TempDir(), "missing"))
	ctx.Flags().Get("config-path").SetAssigned(true)
	_, err := ParseAndGetEndpoint(ctx, nil)
	require.Error(t, err)
}

func TestLocalHashThroughBridge(t *testing.T) {
	clearEndpointTestEnv(t)
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0600))
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, io.Discard)
	ctx.SetCommand(NewCommandBridge(hashCommand.command))
	require.NoError(t, parseAndRunCommandFromCli(ctx, []string{path}, &hashCommand.command))
}
