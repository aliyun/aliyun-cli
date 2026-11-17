package argparser

import (
	"errors"
	"fmt"
	"testing"
)

func TestParserErrorsExposeTelemetryCodesThroughWrapping(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{&InvalidArgumentError{}, "INVALID_ARGUMENT"},
		{&ExternalFlagRejectError{Flag: "private", Message: "rejected"}, "INVALID_FLAG"},
		{&UnknownFlagError{Flag: "private"}, "INVALID_FLAG"},
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
