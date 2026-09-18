package runtimehost

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

func TestCommandAIModeInvocationFlagPrecedence(t *testing.T) {
	isolateAgentDetectionEnvs(t)
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.Flags().Add(config.NewConfigurePathFlag())
	f := config.ConfigurePathFlag(ctx.Flags())
	f.SetAssigned(true)
	f.SetValue(filepath.Join(t.TempDir(), "config.json"))
	_, enabled := commandAIModeState(ctx, []string{"--cli-ai-mode"})
	require.True(t, enabled)
	_, enabled = commandAIModeState(ctx, []string{"--cli-ai-mode", "--no-cli-ai-mode"})
	require.False(t, enabled)
}
