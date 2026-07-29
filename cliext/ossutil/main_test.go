package ossutil

import (
	"bytes"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewOssutilCommand(t *testing.T) {
	cmd := NewOssutilCommand()
	if cmd == nil {
		t.Fatalf("NewOssutilCommand returned nil")
	}
	if cmd.Name != "ossutil" {
		t.Errorf("Name expected 'ossutil', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba OSS Service CLI" {
		t.Errorf("Short en expected 'Alibaba OSS Service CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云OSS服务CLI工具" {
		t.Errorf("Short zh expected '阿里云OSS服务CLI工具', got %s", zh)
	}
	if cmd.Usage != "aliyun ossutil ls --region cn-hangzhou" {
		t.Errorf("Usage expected 'aliyun ossutil ls --region cn-hangzhou', got %s", cmd.Usage)
	}
	if cmd.Hidden {
		t.Errorf("Hidden expected false")
	}
	if !cmd.EnableUnknownFlag {
		t.Errorf("EnableUnknownFlag expected true")
	}
	if !cmd.KeepArgs {
		t.Errorf("KeepArgs expected true")
	}
	if !cmd.SkipDefaultHelp {
		t.Errorf("SkipDefaultHelp expected true")
	}
	if cmd.Run == nil {
		t.Errorf("Run function should not be nil")
	}
}

func TestNewOssutilCommandMetadata(t *testing.T) {
	cmd := NewOssutilCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "ossutil" {
		t.Errorf("metadata name expected ossutil, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Alibaba OSS Service CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "阿里云OSS服务CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestOssutilCommandRunInstalledSkipNetwork(t *testing.T) {
	// 准备临时目录作为配置路径
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	// 创建假可执行文件(ossutil)
	execPath := filepath.Join(tmpDir, "ossutil")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho dummy\n"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}
	// 创建版本检查缓存，避免触发远程版本请求
	cacheFile := filepath.Join(tmpDir, ".ossutil_version_check")
	if err := os.WriteFile(cacheFile, []byte("0"), 0644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	// 设置忽略profile，避免真实配置依赖
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")

	cmd := NewOssutilCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	// 直接调用Run函数（不经过Command.Execute解析）
	err := cmd.Run(ctx, []string{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	errStr := err.Error()
	if !bytes.Contains([]byte(errStr), []byte("profile default is not configure yet")) &&
		!bytes.Contains([]byte(errStr), []byte("can't get credential")) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOssutilEstimateCostRouting(t *testing.T) {
	// File/bucket commands meter transfer/storage, not API pricing: reject
	// before any install/credential machinery (zero-value Context proves no
	// environment is touched) and point at both supported forms.
	ctx := &Context{}
	err := ctx.Run([]string{"ls", "--estimate-cost"})
	if err == nil || !strings.Contains(err.Error(), "aliyun ossutil api <operation> --estimate-cost") {
		t.Fatalf("expected file-command rejection with hint, got %v", err)
	}
	err = ctx.Run([]string{"cp", "a", "oss://b", "--estimate-cost-context", "K=V"})
	if err == nil || !strings.Contains(err.Error(), "only applies to OpenAPI calls") {
		t.Fatalf("expected estimate-cost-context rejection, got %v", err)
	}
	// api subcommand without an operation name: parsed locally, still no
	// environment access.
	err = ctx.Run([]string{"api", "--estimate-cost"})
	if err == nil || !strings.Contains(err.Error(), "requires an operation name") {
		t.Fatalf("expected missing-operation error, got %v", err)
	}
}

func TestOssutilKebabToPascal(t *testing.T) {
	cases := map[string]string{
		"put-bucket":    "PutBucket",
		"storage-class": "StorageClass",
		"PutBucket":     "PutBucket",
		"acl":           "Acl",
	}
	for in, want := range cases {
		if got := kebabToPascal(in); got != want {
			t.Fatalf("kebabToPascal(%q) = %q, want %q", in, got, want)
		}
	}
}

// newApiEstimateTestContext builds a Context whose originCtx carries a
// hermetic profile (temp config via --config-path) so runApiEstimateCost can
// route all the way into the shared quote pipeline without touching the host
// machine's configuration.
func newApiEstimateTestContext(t *testing.T) *Context {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{Name: "ossutil"}
	config.AddFlags(cmd.Flags())
	openapi.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"current":"t","profiles":[{"name":"t","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pathFlag := config.ConfigurePathFlag(ctx.Flags())
	pathFlag.SetAssigned(true)
	pathFlag.SetValue(configPath)
	return &Context{originCtx: ctx}
}

func TestOssutilApiEstimateCostFullParseAndRoute(t *testing.T) {
	// Exercises every branch of the raw-arg parser (operation, kebab flag with
	// separate value, --name=value form, bare bool flag, --region fallback,
	// --estimate-cost marker, --estimate-cost-context K=V) and proves the
	// result reaches the quote pipeline: the dial error names the fake
	// endpoint, so parsing + EstimateOssCost ran end to end without invoking
	// anything real.
	t.Setenv("ALIBABA_CLOUD_PRICING_ENDPOINT", "estimate-cost.test.invalid")
	c := newApiEstimateTestContext(t)
	err := c.runApiEstimateCost([]string{
		"put-bucket",
		"--storage-class", "Standard",
		"--acl=private",
		"--versioning",
		"--region", "cn-hangzhou",
		"--estimate-cost",
		"--estimate-cost-context", "EstimatedStorageGB=100",
	})
	if err == nil || !strings.Contains(err.Error(), "estimate-cost.test.invalid") {
		t.Fatalf("expected quote dial error against fake endpoint, got %v", err)
	}
}

func TestOssutilApiEstimateCostInvalidContextPair(t *testing.T) {
	c := newApiEstimateTestContext(t)
	err := c.runApiEstimateCost([]string{"put-bucket", "--estimate-cost", "--estimate-cost-context", "not-a-pair"})
	if err == nil || !strings.Contains(err.Error(), "Key=Value") {
		t.Fatalf("expected Key=Value usage error, got %v", err)
	}
}
