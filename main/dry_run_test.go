package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

func TestCSDryRunJSONRedactsFormWithoutContentType(t *testing.T) {
	clearAgentDetectionEnv(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	profileJSON := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"test-access-key-id","access_key_secret":"test-access-key-secret","region_id":"cn-hangzhou","language":"en"}]}`)
	if err := os.WriteFile(configPath, profileJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	profile, err := config.LoadProfile(configPath, "default")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	root := newRootCommand(profile, &stdout)
	ctx := newCommandContext(&stdout, &stderr)
	ctx.EnterCommand(root)
	root.Execute(ctx, []string{
		"cs", "POST", "/clusters", "--body", "password=FAKE_SECRET_123", "--cli-dry-run-json",
		"--config-path", configPath,
	})
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", &stderr)
	}
	var output openapi.CliDryRunOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON output: %v; stdout: %s", err, &stdout)
	}
	if output.Method != "POST" || output.Pathname != "/clusters" || output.Body != "password=FAKE***" {
		t.Fatalf("unexpected dry-run output: %+v", output)
	}
	t.Logf("actual dry-run JSON: %s", &stdout)
}

func TestBtripDryRunJSONFormBody(t *testing.T) {
	clearAgentDetectionEnv(t)
	const example = "out_sub_order_id=1001&ext_params={key=example-string}&modify_pay_amount=5100&out_order_id=2001&sub_order_id=1001&isv_name=name&order_id=2001"
	for _, tc := range []struct{ name, body, want string }{
		{"original example", example, fmt.Sprintf("<opaque body omitted: %d bytes>", len(example))},
		{"valid nested JSON", `ext_params={"key":"example-string"}`, `ext_params={"key":"example-string"}`},
		{"sensitive nested JSON", `ext_params={"password":"example-secret"}`, `ext_params={"password":"exam***"}`},
		{"malformed nested JSON and sensitive field", `ext_params={password=example-secret}&password=example-secret`, fmt.Sprintf("<opaque body omitted: %d bytes>", len(`ext_params={password=example-secret}&password=example-secret`))},
		{"malformed form", `password=example-secret&ext_params=%zz`, fmt.Sprintf("<opaque body omitted: %d bytes>", len(`password=example-secret&ext_params=%zz`))},
		{"JSON form representation", `{"password":"example-secret"}`, `{"password":"exam***"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			profileJSON := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"testid","access_key_secret":"testsecret","region_id":"cn-hangzhou","language":"en"}]}`)
			if err := os.WriteFile(configPath, profileJSON, 0o600); err != nil {
				t.Fatal(err)
			}
			profile, err := config.LoadProfile(configPath, "default")
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			root := newRootCommand(profile, &stdout)
			ctx := newCommandContext(&stdout, &stderr)
			ctx.EnterCommand(root)
			root.Execute(ctx, []string{
				"btripopen", "PUT", "/dtb-flight/v2/modify/action/pay",
				"--region", "cn-hangzhou",
				"--header", "Content-Type=application/x-www-form-urlencoded",
				"--body", tc.body, "--version", "2022-05-20", "--force",
				"--mode", "AK", "--access-key-id", "testid", "--access-key-secret", "testsecret",
				"--endpoint", "example.com", "--config-path", configPath, "--cli-dry-run-json",
			})
			if stderr.Len() != 0 {
				t.Fatalf("unexpected stderr: %s", &stderr)
			}
			var output openapi.CliDryRunOutput
			if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
				t.Fatalf("invalid JSON output: %v; stdout: %s", err, &stdout)
			}
			if output.Body != tc.want || output.Method != "PUT" || output.Endpoint != "example.com" {
				t.Fatalf("unexpected dry-run output: %+v; want body: %s", output, tc.want)
			}
			if strings.Contains(stdout.String(), "example-secret") {
				t.Fatalf("secret leaked: %s", &stdout)
			}
		})
	}
}

func TestCSDryRunOmitsBOMXMLBodyFile(t *testing.T) {
	clearAgentDetectionEnv(t)
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGINS_DIR", t.TempDir())
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGIN_AUTO_INSTALL", "false")
	const secret = "FAKE_SECRET_20261103"
	for _, tc := range []struct{ name, body, want string }{
		{"BOM XML", "\ufeff<Password>" + secret + "==</Password>", ""},
		{"zero-width XML", "\u200b<Password>" + secret + "==</Password>", ""},
		{"nested BOM JSON", `{"payload":"\ufeff{\"password\":\"` + secret + `\"}"}`, ""},
		{"nested BOM form", "payload=%EF%BB%BF%7B%22password%22%3A%22" + secret + "%22%7D", ""},
		{"sensitive JSON prefix", `{"password":"[` + secret + `","name":"visible"}`, `{"name":"visible","password":"[FAK***"}`},
		{"sensitive form prefix", "password=%5B" + secret + "&name=visible", "password=[FAK***&name=visible"},
	} {
		body := []byte(tc.body)
		for _, route := range []struct {
			name string
			args []string
		}{
			{"camel", []string{"cs", "CreateCluster", "--version", "2015-12-15"}},
			{"REST", []string{"cs", "POST", "/clusters", "--version", "2015-12-15"}},
			{"kebab", []string{"cs", "create-cluster", "--api-version", "2015-12-15"}},
		} {
			for _, mode := range []string{"--cli-dry-run", "--cli-dry-run-json"} {
				t.Run(tc.name+"/"+route.name+"/"+mode, func(t *testing.T) {
					dir := t.TempDir()
					bodyPath := filepath.Join(dir, "body.xml")
					if err := os.WriteFile(bodyPath, body, 0o600); err != nil {
						t.Fatal(err)
					}
					configPath := filepath.Join(dir, "config.json")
					profileJSON := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"testid","access_key_secret":"testsecret","region_id":"cn-hangzhou","language":"en"}]}`)
					if err := os.WriteFile(configPath, profileJSON, 0o600); err != nil {
						t.Fatal(err)
					}
					profile, err := config.LoadProfile(configPath, "default")
					if err != nil {
						t.Fatal(err)
					}
					var stdout, stderr bytes.Buffer
					root := newRootCommand(profile, &stdout)
					ctx := newCommandContext(&stdout, &stderr)
					ctx.EnterCommand(root)
					args := append([]string(nil), route.args...)
					args = append(args, "--body-file", bodyPath, mode, "--endpoint", "example.com", "--region", "cn-hangzhou", "--config-path", configPath, "--log-level", "DEBUG")
					cli.DisableExitCode()
					defer cli.EnableExitCode()
					originalArgs := os.Args
					os.Args = append([]string{"aliyun"}, args...)
					defer func() { os.Args = originalArgs }()
					root.Execute(ctx, args)
					if stderr.Len() != 0 || strings.Contains(stdout.String(), secret) {
						t.Fatalf("unsafe dry-run output: stdout=%s stderr=%s", &stdout, &stderr)
					}
					want := tc.want
					if want == "" {
						want = fmt.Sprintf("<opaque body omitted: %d bytes>", len(body))
					}
					if mode == "--cli-dry-run-json" {
						var output openapi.CliDryRunOutput
						if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || output.Body != want || output.BodyFormat == "" {
							t.Fatalf("unexpected dry-run JSON: %+v, error=%v", output, err)
						}
					} else if !strings.Contains(stdout.String(), want) {
						t.Fatalf("omission marker missing: %s", &stdout)
					}
					unchanged, err := os.ReadFile(bodyPath)
					if err != nil || !bytes.Equal(unchanged, body) {
						t.Fatalf("input bytes changed: error=%v", err)
					}
				})
			}
		}
	}
}
