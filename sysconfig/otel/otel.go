package otel

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
)

const (
	EnvTraceparent = "ALIBABA_CLOUD_OTEL_TRACEPARENT"
	EnvBaggage     = "ALIBABA_CLOUD_OTEL_BAGGAGE"
	EnvEnabled     = "ALIBABA_CLOUD_OTEL_ENABLED"

	HeaderTraceparent = "traceparent"
	HeaderBaggage     = "baggage"
)

var (
	traceparentRegex           = regexp.MustCompile(`^[0-9a-f]{2}-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)
	warnWriter       io.Writer = os.Stderr
)

func IsEnabled() bool {
	val, ok := os.LookupEnv(EnvEnabled)
	if ok {
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "false", "0", "off":
			return false
		}
	}

	traceparent := os.Getenv(EnvTraceparent)
	baggage := os.Getenv(EnvBaggage)
	return traceparent != "" || baggage != ""
}

func ValidateTraceparent(value string) bool {
	return traceparentRegex.MatchString(value)
}

func GetHeaders() map[string]string {
	if !IsEnabled() {
		return nil
	}

	headers := make(map[string]string)

	if tp := os.Getenv(EnvTraceparent); tp != "" {
		if ValidateTraceparent(tp) {
			headers[HeaderTraceparent] = tp
		} else {
			fmt.Fprintf(warnWriter, "Warning: invalid traceparent value %q ignored, expected format: 00-<32 hex>-<16 hex>-<2 hex>\n", tp)
		}
	}

	if bg := os.Getenv(EnvBaggage); bg != "" {
		headers[HeaderBaggage] = bg
	}

	return headers
}

func InjectHeaders(headers map[string]string) {
	for k, v := range GetHeaders() {
		headers[k] = v
	}
}

func InjectTeaHeaders(headers map[string]*string) {
	for k, v := range GetHeaders() {
		headers[k] = tea.String(v)
	}
}

func MergeOtelEnvs(envs map[string]string) {
	for _, key := range []string{EnvTraceparent, EnvBaggage, EnvEnabled} {
		if v, ok := os.LookupEnv(key); ok {
			envs[key] = v
		}
	}
}
