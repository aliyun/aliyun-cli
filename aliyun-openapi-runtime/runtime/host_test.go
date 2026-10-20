package runtime

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestStaticHostReturnsConfiguredValues(t *testing.T) {
	settings := Settings{
		ReadTimeout: time.Second, ConnectTimeout: 2 * time.Second,
		RetryCount: 3, EndpointType: "vpc", Language: "zh",
		SkipSecureVerify: true, CLIVersion: "1.2.3", UserAgent: "custom-client",
	}
	disabled := false
	transport := TransportOptions{
		Headers: map[string]string{"traceparent": "test-trace"},
		CallContext: CallContextOptions{
			SourceIP: "192.0.2.1", SecureTransport: "true", SkipProducts: []string{"ecs"},
		},
		ThrottlingRetry: ThrottlingRetryOptions{Enabled: &disabled, MaxAttempts: 5, MaxDelayMS: 1000},
	}
	var host Host = StaticHost{RegionID: "cn-hangzhou", SettingsVal: settings, TransportOptionsVal: transport}
	if got := host.Region(); got != "cn-hangzhou" {
		t.Fatalf("Region() = %q", got)
	}
	if got := host.Settings(); got != settings {
		t.Fatalf("Settings() = %#v, want %#v", got, settings)
	}
	if got := host.TransportOptions(); !reflect.DeepEqual(got, transport) {
		t.Fatalf("TransportOptions() = %#v, want %#v", got, transport)
	}
}

func TestStaticHostCredentialPreservesResultAndError(t *testing.T) {
	credential := staticAKCredential(t)
	cause := errors.New("credential unavailable")
	for name, host := range map[string]StaticHost{
		"unset": {}, "resolved": {Cred: credential}, "failure": {CredErr: cause},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := host.Credential()
			if got != host.Cred || err != host.CredErr {
				t.Fatalf("Credential() = %v, %v; want %v, %v", got, err, host.Cred, host.CredErr)
			}
		})
	}
}

func TestSettingsUseVPC(t *testing.T) {
	for _, tt := range []struct {
		endpointType string
		want         bool
	}{
		{"", false}, {"public", false}, {"vpc", true}, {"VPC", false}, {" vpc ", false},
	} {
		t.Run(tt.endpointType, func(t *testing.T) {
			if got := (Settings{EndpointType: tt.endpointType}).UseVPC(); got != tt.want {
				t.Fatalf("UseVPC() = %v for endpoint type %q, want %v", got, tt.endpointType, tt.want)
			}
		})
	}
}
