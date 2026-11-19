package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	malformed := `password=example-secret&ext_params=%zz`
	for _, tc := range []struct{ name, body, want string }{
		{"original example", example, example},
		{"valid nested JSON", `ext_params={"key":"example-string"}`, `ext_params={"key":"example-string"}`},
		{"sensitive nested JSON", `ext_params={"password":"example-secret"}`, `ext_params={"password":"exam***"}`},
		{"malformed nested JSON and sensitive field", `ext_params={password=example-secret}&password=example-secret`, `ext_params={password=exam***}&password=exam***`},
		{"malformed form", malformed, fmt.Sprintf("(body omitted: %d bytes of unparseable content)", len(malformed))},
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
		})
	}
}

// Unparseable bodies must never reach the dry-run preview, in either the JSON
// or the human rendering, for every class named by assessment finding S2.
func TestCSDryRunOpaqueBodyFixturesDoNotLeak(t *testing.T) {
	clearAgentDetectionEnv(t)
	const secret = "FAKE_SECRET_20261103"
	for name, body := range map[string]string{
		"broken form encoding": "password=" + secret + "&name=%zz",
		"xml":                  "<Request><Password>" + secret + "</Password></Request>",
		"damaged json":         `{"password":"` + secret,
		"raw text":             "name: fake-user\nPassword: " + secret,
		"long opaque tail":     strings.Repeat("padding-", 200) + secret,
	} {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			profileJSON := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"test-access-key-id","access_key_secret":"test-access-key-secret","region_id":"cn-hangzhou","language":"en"}]}`)
			if err := os.WriteFile(configPath, profileJSON, 0o600); err != nil {
				t.Fatal(err)
			}
			for _, flags := range [][]string{
				{"--cli-dry-run-json"},
				{"--cli-dry-run"},
			} {
				profile, err := config.LoadProfile(configPath, "default")
				if err != nil {
					t.Fatal(err)
				}
				var stdout, stderr bytes.Buffer
				root := newRootCommand(profile, &stdout)
				ctx := newCommandContext(&stdout, &stderr)
				ctx.EnterCommand(root)
				args := append([]string{"cs", "POST", "/clusters", "--body", body, "--config-path", configPath}, flags...)
				root.Execute(ctx, args)
				if stderr.Len() != 0 {
					t.Fatalf("%s: unexpected stderr: %s", flags[0], &stderr)
				}
				output := stdout.String()
				if strings.Contains(output, secret) {
					t.Fatalf("%s: fake secret leaked: %s", flags[0], output)
				}
				want := fmt.Sprintf("(body omitted: %d bytes of unparseable content)", len(body))
				if !strings.Contains(output, want) {
					t.Fatalf("%s: missing omission marker %q in: %s", flags[0], want, output)
				}
				if flags[0] == "--cli-dry-run-json" {
					var decoded openapi.CliDryRunOutput
					if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
						t.Fatalf("invalid JSON output: %v; stdout: %s", err, output)
					}
					if decoded.Body != want {
						t.Fatalf("JSON body = %q, want %q", decoded.Body, want)
					}
				}
			}
		})
	}
}
