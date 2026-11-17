package engine

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestUsageErrorTelemetryClassification(t *testing.T) {
	cause := errors.New("original user-facing diagnostic")
	for _, test := range []struct{ code, want string }{
		{"MISSING_COMMAND", "INVALID_COMMAND"},
		{"UNKNOWN_FLAG", "INVALID_FLAG"},
		{"MISSING_REQUIRED_PARAMETER", "MISSING_REQUIRED_PARAMETER"},
		{"INVALID_ARGUMENT", "INVALID_ARGUMENT"},
		{"INVALID_PARAMETER_VALUE", "INVALID_ARGUMENT"},
		{"INVALID_OPTION_COMBINATION", "INVALID_OPTION_COMBINATION"},
		{"INVALID_HEADER", "INVALID_HEADER"},
		{"INVALID_BODY_FILE", "INVALID_BODY_FILE"},
		{"UNTRUSTED_REASON", ""},
	} {
		t.Run(test.code, func(t *testing.T) {
			err := &UsageError{Code: test.code, Err: cause}
			if err.Error() != cause.Error() {
				t.Fatalf("human diagnostic changed: %s", err)
			}
			if !errors.Is(err, cause) {
				t.Fatal("cause lost")
			}
			var coded interface{ LocalErrorCode() string }
			if !errors.As(fmt.Errorf("request failed: %w", err), &coded) {
				t.Fatal("telemetry code lost")
			}
			if got := coded.LocalErrorCode(); got != test.want {
				t.Fatalf("code = %q, want %q", got, test.want)
			}
		})
	}
}

func TestExplicitRuntimeErrorsPreserveTelemetryCodeAndCause(t *testing.T) {
	cause := errors.New("original user-facing diagnostic")
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{"options", &InvalidOptionCombinationError{Err: cause}, "INVALID_OPTION_COMBINATION"},
		{"header", &InvalidHeaderError{Err: cause}, "INVALID_HEADER"},
		{"body_file", &InvalidBodyFileError{Err: cause}, "INVALID_BODY_FILE"},
		{"query", &QueryFilterError{Err: cause}, "QUERY_FILTER_ERROR"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !errors.Is(test.err, cause) {
				t.Fatal("cause lost")
			}
			if !strings.Contains(test.err.Error(), cause.Error()) {
				t.Fatalf("human diagnostic changed: %s", test.err)
			}
			var coded interface{ LocalErrorCode() string }
			if !errors.As(fmt.Errorf("wrapped: %w", test.err), &coded) {
				t.Fatal("telemetry code lost")
			}
			if got := coded.LocalErrorCode(); got != test.code {
				t.Fatalf("code = %q, want %q", got, test.code)
			}
		})
	}
	if got := (&UnknownCommandError{}).LocalErrorCode(); got != "INVALID_COMMAND" {
		t.Fatalf("unknown command code = %q", got)
	}
}
