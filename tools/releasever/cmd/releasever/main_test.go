package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReleaseVersionMainChild(t *testing.T) {
	if os.Getenv("ALIYUN_CLI_TEST_TOOL_MAIN") == "1" {
		os.Args = []string{"releasever", "classify", "1.2.3"}
		main()
	}
}

func TestMainForwardsVersionClassificationArguments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReleaseVersionMainChild$")
	command.Env = append(os.Environ(), "ALIYUN_CLI_TEST_TOOL_MAIN=1")
	output, err := command.CombinedOutput()
	require.NoError(t, ctx.Err())
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "stable")
}
