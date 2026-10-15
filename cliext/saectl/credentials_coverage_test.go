package saectl

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

type credentialTransport func(*http.Request) (*http.Response, error)

func (f credentialTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTemporaryCredentialModes(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	t.Setenv(config.EnvDisableExternalProcess, "")
	t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "")
	for _, tc := range []struct {
		name, mode string
		status     int
		wantErr    bool
	}{
		{name: "static session", mode: "StsToken", status: 200}, {name: "credential URI", mode: "CredentialsURI", status: 200}, {name: "credential URI rejected", mode: "CredentialsURI", status: 403, wantErr: true}, {name: "invalid saved mode", mode: "invalid", status: 200, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			profile := map[string]any{"name": "default", "mode": tc.mode, "access_key_id": "id", "access_key_secret": "secret", "sts_token": "token", "credentials_uri": "https://credentials.example.invalid/temporary", "region_id": "cn-hangzhou"}
			data, err := json.Marshal(map[string]any{"current": "default", "profiles": []any{profile}})
			if err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(dir, "config.json")
			if err := os.WriteFile(filename, data, 0600); err != nil {
				t.Fatal(err)
			}
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			config.AddFlags(ctx.Flags())
			f := ctx.Flags().Get(config.ConfigurePathFlagName)
			f.SetAssigned(true)
			f.SetValue(filename)
			requests := 0
			http.DefaultTransport = credentialTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Host != "credentials.example.invalid" {
					t.Errorf("unexpected credential host=%s", r.URL.Host)
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(`{"Code":"Success","AccessKeyId":"id","AccessKeySecret":"secret","SecurityToken":"token"}`))}, nil
			})
			c := NewContext(ctx)
			c.configPath = dir
			err = c.PrepareEnv()
			if tc.wantErr {
				if err == nil {
					t.Fatal("invalid credentials accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.envMap["ALICLOUD_STS_TOKEN"] != "token" || c.envMap["ALICLOUD_ACCESS_KEY"] != "id" {
				t.Fatalf("credential environment = %v", c.envMap)
			}
			if tc.mode == "CredentialsURI" && requests != 1 {
				t.Fatalf("credential requests=%d", requests)
			}
		})
	}
}
