package runtime

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationErrorsExposeTelemetryCodesThroughWrapping(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{&MissingRequiredError{Flags: []string{"--name"}}, "MISSING_REQUIRED_PARAMETER"},
		{&EndpointNotResolvedError{Product: "ecs", Region: "unknown"}, "ENDPOINT_NOT_RESOLVED"},
	} {
		var coded interface{ LocalErrorCode() string }
		if !errors.As(fmt.Errorf("wrapped: %w", test.err), &coded) {
			t.Fatalf("missing telemetry code for %T", test.err)
		}
		if got := coded.LocalErrorCode(); got != test.code {
			t.Fatalf("%T code = %q, want %q", test.err, got, test.code)
		}
	}
}
