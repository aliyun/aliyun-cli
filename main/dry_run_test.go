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

func TestCSDryRunJSONRedactsRawPassword(t *testing.T) {
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
	if bytes.Contains(stdout.Bytes(), []byte("FAKE_SECRET_123")) {
		t.Fatal("dry-run output contains the unmasked password")
	}
	var output openapi.CliDryRunOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON output: %v; stdout: %s", err, &stdout)
	}
	if output.Method != "POST" || output.Pathname != "/clusters" || output.Body != "password=FAKE%2A%2A%2A" {
		t.Fatalf("unexpected dry-run output: %+v", output)
	}
	t.Logf("actual dry-run JSON: %s", &stdout)
}
