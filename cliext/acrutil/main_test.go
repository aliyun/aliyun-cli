package acrutil

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewAcrutilCommand(t *testing.T) {
	cmd := NewAcrutilCommand()
	if cmd == nil {
		t.Fatalf("NewAcrutilCommand returned nil")
	}
	if cmd.Name != "acrutil" {
		t.Errorf("Name expected 'acrutil', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud ACR Enterprise Edition Instance CLI Tool" {
		t.Errorf("Short en expected 'Alibaba Cloud ACR Enterprise Edition Instance CLI Tool', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云ACR企业版实例CLI工具" {
		t.Errorf("Short zh expected '阿里云ACR企业版实例CLI工具', got %s", zh)
	}
	if cmd.Usage != "acrutil <command> [args...]" {
		t.Errorf("Usage expected 'acrutil <command> [args...]', got %s", cmd.Usage)
	}
	if cmd.Hidden {
		t.Errorf("Hidden expected false")
	}
	if !cmd.DisablePersistentFlags {
		t.Errorf("DisablePersistentFlags expected true")
	}
	if cmd.Run == nil {
		t.Errorf("Run function should not be nil")
	}
}

func TestNewAcrutilCommandHasSkillSubCommand(t *testing.T) {
	cmd := NewAcrutilCommand()
	skillCmd := cmd.GetSubCommand("skill")
	if skillCmd == nil {
		t.Fatalf("skill subcommand not found")
	}
	if skillCmd.Name != "skill" {
		t.Errorf("skill subcommand Name expected 'skill', got %s", skillCmd.Name)
	}
	if skillCmd.Short == nil {
		t.Fatalf("skill Short i18n text nil")
	}
	if en := skillCmd.Short.Get("en"); en != "ACR Skill Management" {
		t.Errorf("skill Short en expected 'ACR Skill Management', got %s", en)
	}
	if zh := skillCmd.Short.Get("zh"); zh != "ACR Skill管理" {
		t.Errorf("skill Short zh expected 'ACR Skill管理', got %s", zh)
	}
	if !skillCmd.EnableUnknownFlag {
		t.Errorf("skill EnableUnknownFlag expected true")
	}
	if !skillCmd.KeepArgs {
		t.Errorf("skill KeepArgs expected true")
	}
	if !skillCmd.SkipDefaultHelp {
		t.Errorf("skill SkipDefaultHelp expected true")
	}
	if skillCmd.Run == nil {
		t.Errorf("skill Run function should not be nil")
	}
}

func TestNewAcrutilCommandMetadata(t *testing.T) {
	cmd := NewAcrutilCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "acrutil" {
		t.Errorf("metadata name expected acrutil, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Alibaba Cloud ACR Enterprise Edition Instance CLI Tool" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "阿里云ACR企业版实例CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}
