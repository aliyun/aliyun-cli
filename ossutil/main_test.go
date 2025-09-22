package ossutil

import (
	"bytes"
	"os"
	"path/filepath"
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
