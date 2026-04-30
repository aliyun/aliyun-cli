package acrskill

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewAcrSkillCommand(t *testing.T) {
	cmd := NewAcrSkillCommand()
	if cmd == nil {
		t.Fatalf("NewAcrSkillCommand returned nil")
	}
	if cmd.Name != "acr-skill" {
		t.Errorf("Name expected 'acr-skill', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud ACR Skill Management CLI" {
		t.Errorf("Short en expected 'Alibaba Cloud ACR Skill Management CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云ACR服务Skill管理CLI工具" {
		t.Errorf("Short zh expected '阿里云ACR服务Skill管理CLI工具', got %s", zh)
	}
	if cmd.Usage != "aliyun acr-skill validate -d ./my-skill" {
		t.Errorf("Usage expected 'aliyun acr-skill validate -d ./my-skill', got %s", cmd.Usage)
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

func TestNewAcrSkillCommandMetadata(t *testing.T) {
	cmd := NewAcrSkillCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "acr-skill" {
		t.Errorf("metadata name expected acr-skill, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Alibaba Cloud ACR Skill Management CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "阿里云ACR服务Skill管理CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestAcrSkillCommandRunInstalledSkipDownload(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	execPath := filepath.Join(tmpDir, "acr-skill")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho dummy\n"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}

	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")

	cmd := NewAcrSkillCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	err := cmd.Run(ctx, []string{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	errStr := err.Error()
	if !bytes.Contains([]byte(errStr), []byte("profile default is not configure yet")) &&
		!bytes.Contains([]byte(errStr), []byte("can't get credential")) &&
		!bytes.Contains([]byte(errStr), []byte("config failed")) {
		t.Errorf("unexpected error: %v", err)
	}
}
