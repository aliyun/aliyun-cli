package saectl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func prepareConfig(t *testing.T, home string) {
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	configJSON := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"en"}]}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func newOriginCtx() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	return ctx, out, errOut
}

func addConfigFlag(ctx *cli.Context, name string, value string) {
	f := &cli.Flag{Name: name, AssignedMode: cli.AssignedOnce, Category: "config"}
	f.SetAssigned(true)
	f.SetValue(value)
	ctx.Flags().Add(f)
}

func TestPrepareEnv_Success(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home)
	
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}
	
	if c.envMap["ALICLOUD_REGION"] != "cn-hangzhou" {
		t.Fatalf("region mismatch: %v", c.envMap["ALICLOUD_REGION"])
	}
	if c.envMap["ALICLOUD_ACCESS_KEY"] != "ak" {
		t.Fatalf("ak mismatch: %v", c.envMap["ALICLOUD_ACCESS_KEY"])
	}
	if c.envMap["ALICLOUD_SECRET_KEY"] != "sk" {
		t.Fatalf("sk mismatch: %v", c.envMap["ALICLOUD_SECRET_KEY"])
	}
}

func TestRemoveFlagsForMainCli_Success(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "profile", "test")
	
	c := NewContext(ctx)
	args, err := c.RemoveFlagsForMainCli([]string{"aliyun", "sae", "get", "app", "--profile", "test"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--profile") {
		t.Fatalf("profile flag should be removed: %s", joined)
	}
	if !strings.Contains(joined, "get app") {
		t.Fatalf("original args missing: %s", joined)
	}
}
