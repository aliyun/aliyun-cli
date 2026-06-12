package codeup

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewCodeupCliCommand(t *testing.T) {
	cmd := NewCodeupCliCommand()
	if cmd == nil {
		t.Fatalf("NewCodeupCliCommand returned nil")
	}
	if cmd.Name != "codeup-cli" {
		t.Errorf("Name expected 'codeup-cli', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Migrate third-party code repositories to Alibaba Cloud Codeup" {
		t.Errorf("Short en expected 'Migrate third-party code repositories to Alibaba Cloud Codeup', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "三方代码平台迁移到阿里云Codeup平台的命令行工具" {
		t.Errorf("Short zh expected '三方代码平台迁移到阿里云Codeup平台的命令行工具', got %s", zh)
	}
	if cmd.Usage != "aliyun codeup-cli <command> [args...]" {
		t.Errorf("Usage expected 'aliyun codeup-cli <command> [args...]', got %s", cmd.Usage)
	}
	if cmd.Hidden {
		t.Errorf("Hidden expected false")
	}
	if !cmd.DisablePersistentFlags {
		t.Errorf("DisablePersistentFlags expected true")
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

func TestNewCodeupCliCommandMetadata(t *testing.T) {
	cmd := NewCodeupCliCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "codeup-cli" {
		t.Errorf("metadata name expected codeup-cli, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Migrate third-party code repositories to Alibaba Cloud Codeup" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "三方代码平台迁移到阿里云Codeup平台的命令行工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}
