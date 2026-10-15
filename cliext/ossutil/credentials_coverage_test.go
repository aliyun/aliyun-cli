package ossutil

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestExternalCredentialsArePassedAsTemporarySession(t *testing.T) {
	t.Setenv("CLI_TEST_OSS_CREDENTIALS", "1")
	t.Setenv(config.EnvDisableExternalProcess, "")
	dir := t.TempDir()
	filename := filepath.Join(dir, "config.json")
	command := fmt.Sprintf("%q -test.run=^TestOssCredentialChild$", os.Args[0])
	data, err := json.Marshal(map[string]any{"current": "default", "profiles": []any{map[string]any{"name": "default", "mode": "External", "process_command": command, "region_id": "cn-hangzhou"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	flag := ctx.Flags().Get(config.ConfigurePathFlagName)
	flag.SetAssigned(true)
	flag.SetValue(filename)
	c := NewContext(ctx)
	if err := c.PrepareEnv(); err != nil {
		t.Fatal(err)
	}
	payload, err := base64.StdEncoding.DecodeString(c.envMap["OSSUTIL_CONFIG_VALUE"])
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"mode": "StsToken", "access_key_id": "id", "access_key_secret": "secret", "sts_token": "token", "region_id": "cn-hangzhou"} {
		if got[k] != want {
			t.Errorf("config %s=%v,want=%q", k, got[k], want)
		}
	}
	if c.envMap["OSSUTIL_COMPAT_MODE"] != "alicli" {
		t.Fatal("compatibility mode missing")
	}
	t.Setenv(config.EnvDisableExternalProcess, "true")
	if err := c.PrepareEnv(); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled external credentials=%v", err)
	}
}
func TestOssCredentialChild(t *testing.T) {
	if os.Getenv("CLI_TEST_OSS_CREDENTIALS") == "1" {
		_, _ = io.WriteString(os.Stdout, `{"mode":"StsToken","access_key_id":"id","access_key_secret":"secret","sts_token":"token","region_id":"cn-hangzhou"}`)
		os.Exit(0)
	}
}

type brokenOutput struct{}

func (brokenOutput) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestOssUtilityOutputFailures(t *testing.T) {
	var nilContext *Context
	nilContext.info("ignored")
	(&Context{}).info("ignored")
	ctx, out, _ := newOriginCtx()
	c := NewContext(ctx)
	c.info()
	format := "version %s"
	c.info(format, "2.0")
	c.info("ready", 1)
	if !strings.Contains(out.String(), "version 2.0") || !strings.Contains(out.String(), "ready 1") {
		t.Fatalf("utility output=%q", out.String())
	}
	c.originCtx = cli.NewCommandContext(io.Discard, brokenOutput{})
	c.errorf("failure %s", "message")
	if _, err := EncodeMapBase64(map[string]any{"invalid": make(chan int)}); err == nil {
		t.Fatal("unserializable config accepted")
	}
}
