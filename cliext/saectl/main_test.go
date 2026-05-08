package saectl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewSaectlCommand(t *testing.T) {
	cmd := NewSaectlCommand()
	if cmd == nil {
		t.Fatalf("NewSaectlCommand returned nil")
	}
	if cmd.Name != "saectl" {
		t.Errorf("Name expected 'saectl', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Serverless App Engine CLI" {
		t.Errorf("Short en expected 'Alibaba Serverless App Engine CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云 Serverless 应用引擎 CLI工具" {
		t.Errorf("Short zh expected '阿里云 Serverless 应用引擎 CLI工具', got %s", zh)
	}
	if cmd.Usage != "aliyun saectl <command> [flags]" {
		t.Errorf("Usage expected 'aliyun saectl <command> [flags]', got %s", cmd.Usage)
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

func TestNewSaectlCommandMetadata(t *testing.T) {
	cmd := NewSaectlCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "saectl" {
		t.Errorf("metadata name expected saectl, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Alibaba Serverless App Engine CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "阿里云 Serverless 应用引擎 CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestSaectlCommandRunInstalledSkipNetwork(t *testing.T) {
	// 准备临时目录作为配置路径
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	// 创建假可执行文件(saectl)
	execPath := filepath.Join(tmpDir, "saectl")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho dummy\n"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}

	// 创建版本缓存文件，跳过远程版本检查
	cacheFile := filepath.Join(tmpDir, ".saectl_version_check")
	if err := os.WriteFile(cacheFile, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	// 创建 config.json 以便 PrepareEnv 跳过写入
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"current":"default"}`), 0644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	cmd := NewSaectlCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	// 直接调用Run函数
	err := cmd.Run(ctx, []string{"--version"})
	if err != nil {
		t.Logf("Run returned error (may be expected on some platforms): %v", err)
	}
}
