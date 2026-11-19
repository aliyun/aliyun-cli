//go:build linux || darwin || windows

package main

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

// Pre-validation failures still emit telemetry; built-in oss must report its
// product semantics (cliext) there too, without any network access.
func TestMainBuiltInOssTelemetryCommandTypeIsCLIExt(t *testing.T) {
	for _, command := range []string{"ls", "cp"} {
		t.Run(command, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			got := runTelemetryPermissionCommand(t, home, false, []string{"oss", command, "--versin"})
			require.Greater(t, got.exitCode, 0, "pre-validation failure must stay a failure: %s", got.stderr)
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, telemetry.CommandTypeCLIExt, event.CommandType)
			require.Equal(t, "oss "+command, event.Command)
			require.NotEqual(t, "Success", event.Result)
		})
	}
}
