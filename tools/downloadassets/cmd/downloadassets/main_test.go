package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDownloadAssetsMainChild(t *testing.T) {
	if os.Getenv("ALIYUN_CLI_TEST_TOOL_MAIN") == "1" {
		os.Args = []string{"downloadassets"}
		main()
	}
}

func TestMainRejectsMissingVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDownloadAssetsMainChild$")
	command.Env = append(os.Environ(), "ALIYUN_CLI_TEST_TOOL_MAIN=1")
	output, err := command.CombinedOutput()
	require.NoError(t, ctx.Err())
	var status *exec.ExitError
	require.ErrorAs(t, err, &status, string(output))
	require.Equal(t, 1, status.ExitCode())
	require.Contains(t, string(output), "usage: downloadassets <version>")
}
