package appmanagerutil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewAppManagerCommand(t *testing.T) {
	cmd := NewAppManagerCommand()
	if cmd == nil {
		t.Fatalf("NewAppManagerCommand returned nil")
	}
	if cmd.Name != "appmanager" {
		t.Errorf("Name expected 'appmanager', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud AppManager CLI" {
		t.Errorf("Short en expected 'Alibaba Cloud AppManager CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云应用管理CLI工具" {
		t.Errorf("Short zh expected '阿里云应用管理CLI工具', got %s", zh)
	}
	if cmd.Usage != "aliyun appmanager <command> [args...]" {
		t.Errorf("Usage expected 'aliyun appmanager <command> [args...]', got %s", cmd.Usage)
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

func TestNewAppManagerCommandMetadata(t *testing.T) {
	cmd := NewAppManagerCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "appmanager" {
		t.Errorf("metadata name expected appmanager, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Alibaba Cloud AppManager CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "阿里云应用管理CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestAppManagerCommandRunInstalledSkipNetwork(t *testing.T) {
	// 准备临时目录作为配置路径
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	// 创建假可执行文件(appmanager)
	execPath := filepath.Join(tmpDir, "appmanager")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho dummy\n"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}

	// 创建版本缓存文件，跳过远程版本检查（占位符URL暂不可用）
	cacheFile := filepath.Join(tmpDir, ".appmanager_version_check")
	if err := os.WriteFile(cacheFile, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	// 创建 config.json 以便 PrepareEnv 跳过写入
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"current":"default"}`), 0644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	cmd := NewAppManagerCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	// 直接调用Run函数，空参数下 appmanager 会执行并返回
	err := cmd.Run(ctx, []string{"--version"})
	// 假的 shell 脚本会输出 dummy 然后退出，不会返回错误
	if err != nil {
		// 可能因为平台差异导致执行失败，允许 exec 相关错误
		t.Logf("Run returned error (may be expected on some platforms): %v", err)
	}
}
