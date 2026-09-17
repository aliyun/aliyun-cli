package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	for _, tc := range []struct{ name, body, want string }{
		{"original example", example, example},
		{"valid nested JSON", `ext_params={"key":"example-string"}`, `ext_params={"key":"example-string"}`},
		{"sensitive nested JSON", `ext_params={"password":"example-secret"}`, `ext_params={"password":"exam***"}`},
		{"malformed nested JSON and sensitive field", `ext_params={password=example-secret}&password=example-secret`, `ext_params={password=example-secret}&password=exam***`},
		{"malformed form", `password=example-secret&ext_params=%zz`, `password=example-secret&ext_params=%zz`},
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
