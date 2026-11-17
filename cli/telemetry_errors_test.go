package cli_test

import (
	"fmt"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func TestCLIUsageErrorsProduceTelemetryCodes(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{&cli.InvalidOptionCombinationError{}, "INVALID_OPTION_COMBINATION"},
		{cli.NewInvalidCommandError("private-command", nil), "INVALID_COMMAND"},
		{cli.NewInvalidFlagError("private-flag", nil), "INVALID_FLAG"},
		{&cli.HelpOptionError{}, "HELP_OPTION_ERROR"},
	} {
		t.Run(test.code, func(t *testing.T) {
			event, err := telemetry.BuildEvent(telemetry.BuildInput{Err: fmt.Errorf("private context: %w", test.err)})
			require.NoError(t, err)
			require.Equal(t, "UserFault", event.Result)
			require.Equal(t, test.code, event.LocalErrorCode)
		})
	}
}
