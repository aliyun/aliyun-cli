package openapi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func TestLegacyErrorsProduceSafeTelemetryCodes(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{"required", &LegacyMissingRequiredError{}, "MISSING_REQUIRED_PARAMETER"},
		{"doc_required", &LegacyDocRequiredError{}, "MISSING_REQUIRED_PARAMETER"},
		{"product", &InvalidProductError{}, "INVALID_COMMAND"},
		{"api", &InvalidApiError{}, "INVALID_COMMAND"},
		{"product_or_plugin", &InvalidProductOrPluginError{}, "INVALID_COMMAND"},
		{"unified_api", &InvalidUnifiedApiError{}, "INVALID_COMMAND"},
		{"baseline_command", &InvalidBaselineCommandError{}, "INVALID_COMMAND"},
		{"parameter", &InvalidParameterError{}, "INVALID_FLAG"},
		{"argument", &InvalidArgumentError{}, "INVALID_ARGUMENT"},
		{"constraint", &ConstraintViolationError{}, "INVALID_ARGUMENT"},
		{"options", &InvalidOptionCombinationError{}, "INVALID_OPTION_COMBINATION"},
		{"header", &InvalidHeaderError{}, "INVALID_HEADER"},
		{"body_file", &InvalidBodyFileError{}, "INVALID_BODY_FILE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, err := telemetry.BuildEvent(telemetry.BuildInput{Err: fmt.Errorf("private customer context: %w", test.err)})
			require.NoError(t, err)
			require.Equal(t, "UserFault", event.Result)
			require.Equal(t, test.code, event.LocalErrorCode)
			require.Empty(t, event.APIErrorCode)
			require.Zero(t, event.HTTPStatusCode)
		})
	}
	// Ordinary failures must not be relabelled as local usage errors by text.
	event, err := telemetry.BuildEvent(telemetry.BuildInput{Err: errors.New("INVALID_ARGUMENT private value")})
	require.NoError(t, err)
	require.Equal(t, "Failure", event.Result)
	require.Empty(t, event.LocalErrorCode)
}
