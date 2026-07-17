package otel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func unsetOtelEnvs(t *testing.T) {
	t.Helper()
	t.Setenv(EnvTraceparent, "")
	t.Setenv(EnvBaggage, "")
	t.Setenv(EnvEnabled, "")
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name        string
		enabled     *string // nil = unset, "" = set empty
		traceparent string
		baggage     string
		want        bool
	}{
		{
			name:        "unset enabled, traceparent set",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        true,
		},
		{
			name:    "unset enabled, baggage set",
			baggage: "sessionId=abc-123",
			want:    true,
		},
		{
			name:        "unset enabled, both set",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			baggage:     "sessionId=abc-123",
			want:        true,
		},
		{
			name: "unset enabled, neither set",
			want: false,
		},
		{
			name:        "enabled=false",
			enabled:     strPtr("false"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "enabled=FALSE",
			enabled:     strPtr("FALSE"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "enabled=False",
			enabled:     strPtr("False"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "enabled=0",
			enabled:     strPtr("0"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "enabled=off",
			enabled:     strPtr("off"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "enabled=OFF",
			enabled:     strPtr("OFF"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "enabled=Off",
			enabled:     strPtr("Off"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "enabled=true with traceparent",
			enabled:     strPtr("true"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        true,
		},
		{
			name:    "enabled=true without traceparent or baggage",
			enabled: strPtr("true"),
			want:    false,
		},
		{
			name:        "enabled= (empty) with traceparent",
			enabled:     strPtr(""),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        true,
		},
		{
			name:        "enabled= (whitespace) false",
			enabled:     strPtr(" false "),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetOtelEnvs(t)
			if tt.enabled != nil {
				t.Setenv(EnvEnabled, *tt.enabled)
			}
			if tt.traceparent != "" {
				t.Setenv(EnvTraceparent, tt.traceparent)
			}
			if tt.baggage != "" {
				t.Setenv(EnvBaggage, tt.baggage)
			}
			assert.Equal(t, tt.want, IsEnabled())
		})
	}
}

func TestValidateTraceparent(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "valid traceparent",
			value: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:  true,
		},
		{
			name:  "valid all zeros",
			value: "00-00000000000000000000000000000000-0000000000000000-00",
			want:  true,
		},
		{
			name:  "valid all f",
			value: "ff-ffffffffffffffffffffffffffffffff-ffffffffffffffff-ff",
			want:  true,
		},
		{
			name:  "invalid uppercase hex",
			value: "00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01",
			want:  false,
		},
		{
			name:  "invalid too few segments",
			value: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7",
			want:  false,
		},
		{
			name:  "invalid too many segments",
			value: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-extra",
			want:  false,
		},
		{
			name:  "invalid trace-id too short",
			value: "00-4bf92f3577b34da6a3ce929d0e0e473-00f067aa0ba902b7-01",
			want:  false,
		},
		{
			name:  "invalid trace-id too long",
			value: "00-4bf92f3577b34da6a3ce929d0e0e47360-00f067aa0ba902b7-01",
			want:  false,
		},
		{
			name:  "invalid parent-id too short",
			value: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b-01",
			want:  false,
		},
		{
			name:  "invalid non-hex chars",
			value: "00-4bf92f3577b34da6a3ce929d0e0e473g-00f067aa0ba902b7-01",
			want:  false,
		},
		{
			name:  "empty string",
			value: "",
			want:  false,
		},
		{
			name:  "random string",
			value: "not-a-traceparent",
			want:  false,
		},
		{
			name:  "valid version ff",
			value: "ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ValidateTraceparent(tt.value))
		})
	}
}

func TestGetHeaders(t *testing.T) {
	tests := []struct {
		name           string
		enabled        *string
		traceparent    string
		baggage        string
		wantHeaders    map[string]string
		wantWarnSubstr string
	}{
		{
			name:        "both valid traceparent and baggage",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			baggage:     "sessionId=abc-123,userId=user-001",
			wantHeaders: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				"baggage":     "sessionId=abc-123,userId=user-001",
			},
		},
		{
			name:        "only traceparent",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantHeaders: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			},
		},
		{
			name:    "only baggage",
			baggage: "sessionId=abc-123",
			wantHeaders: map[string]string{
				"baggage": "sessionId=abc-123",
			},
		},
		{
			name:        "invalid traceparent with baggage",
			traceparent: "INVALID-TRACEPARENT",
			baggage:     "sessionId=abc-123",
			wantHeaders: map[string]string{
				"baggage": "sessionId=abc-123",
			},
			wantWarnSubstr: "invalid traceparent",
		},
		{
			name:           "invalid traceparent only",
			traceparent:    "INVALID",
			wantHeaders:    map[string]string{},
			wantWarnSubstr: "invalid traceparent",
		},
		{
			name:        "disabled via enabled=false",
			enabled:     strPtr("false"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			baggage:     "sessionId=abc-123",
			wantHeaders: nil,
		},
		{
			name:        "disabled via enabled=0",
			enabled:     strPtr("0"),
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantHeaders: nil,
		},
		{
			name:        "nothing set",
			wantHeaders: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetOtelEnvs(t)
			if tt.enabled != nil {
				t.Setenv(EnvEnabled, *tt.enabled)
			}
			if tt.traceparent != "" {
				t.Setenv(EnvTraceparent, tt.traceparent)
			}
			if tt.baggage != "" {
				t.Setenv(EnvBaggage, tt.baggage)
			}

			var buf bytes.Buffer
			origWriter := warnWriter
			warnWriter = &buf
			defer func() { warnWriter = origWriter }()

			got := GetHeaders()
			assert.Equal(t, tt.wantHeaders, got)

			if tt.wantWarnSubstr != "" {
				assert.Contains(t, buf.String(), tt.wantWarnSubstr)
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestInjectHeaders(t *testing.T) {
	t.Run("injects into existing map", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		t.Setenv(EnvBaggage, "sessionId=abc-123")

		headers := map[string]string{
			"Content-Type": "application/json",
		}
		InjectHeaders(headers)

		assert.Equal(t, "application/json", headers["Content-Type"])
		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", headers["traceparent"])
		assert.Equal(t, "sessionId=abc-123", headers["baggage"])
	})

	t.Run("overwrites existing traceparent", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

		headers := map[string]string{
			"traceparent": "old-value",
		}
		InjectHeaders(headers)

		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", headers["traceparent"])
	})

	t.Run("no-op when disabled", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvEnabled, "false")
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

		headers := map[string]string{
			"Content-Type": "application/json",
		}
		InjectHeaders(headers)

		assert.Equal(t, 1, len(headers))
		assert.Equal(t, "application/json", headers["Content-Type"])
	})

	t.Run("no-op when nothing set", func(t *testing.T) {
		unsetOtelEnvs(t)

		headers := map[string]string{}
		InjectHeaders(headers)

		assert.Empty(t, headers)
	})
}

func TestInjectTeaHeaders(t *testing.T) {
	t.Run("injects tea string pointers", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		t.Setenv(EnvBaggage, "sessionId=abc-123")

		headers := map[string]*string{}
		InjectTeaHeaders(headers)

		assert.NotNil(t, headers["traceparent"])
		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", *headers["traceparent"])
		assert.NotNil(t, headers["baggage"])
		assert.Equal(t, "sessionId=abc-123", *headers["baggage"])
	})

	t.Run("no-op when disabled", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvEnabled, "off")
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

		headers := map[string]*string{}
		InjectTeaHeaders(headers)

		assert.Empty(t, headers)
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvBaggage, "key=value")

		existing := "existing-value"
		headers := map[string]*string{
			"x-custom": &existing,
		}
		InjectTeaHeaders(headers)

		assert.Equal(t, "existing-value", *headers["x-custom"])
		assert.Equal(t, "key=value", *headers["baggage"])
	})
}

func TestMergeOtelEnvs(t *testing.T) {
	t.Run("all three env vars set", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		t.Setenv(EnvBaggage, "sessionId=abc-123")
		t.Setenv(EnvEnabled, "true")

		envs := map[string]string{}
		MergeOtelEnvs(envs)

		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", envs[EnvTraceparent])
		assert.Equal(t, "sessionId=abc-123", envs[EnvBaggage])
		assert.Equal(t, "true", envs[EnvEnabled])
	})

	t.Run("only traceparent set", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

		envs := map[string]string{"existing": "value"}
		MergeOtelEnvs(envs)

		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", envs[EnvTraceparent])
		assert.Equal(t, "value", envs["existing"])
		// EnvBaggage should be present because t.Setenv sets it to "" in unsetOtelEnvs
		// but since unsetOtelEnvs sets empty string via t.Setenv, LookupEnv returns (ok=true, "")
		// That's expected behavior - empty string env vars are still propagated
	})

	t.Run("preserves existing map entries", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

		envs := map[string]string{
			"ALIBABA_CLOUD_ACCESS_KEY_ID": "test-ak",
		}
		MergeOtelEnvs(envs)

		assert.Equal(t, "test-ak", envs["ALIBABA_CLOUD_ACCESS_KEY_ID"])
		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", envs[EnvTraceparent])
	})

	t.Run("enabled=false is still propagated", func(t *testing.T) {
		unsetOtelEnvs(t)
		t.Setenv(EnvEnabled, "false")

		envs := map[string]string{}
		MergeOtelEnvs(envs)

		assert.Equal(t, "false", envs[EnvEnabled])
	})
}

func strPtr(s string) *string {
	return &s
}
