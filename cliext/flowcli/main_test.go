package flowcli

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewFlowcliCommand(t *testing.T) {
	cmd := NewFlowcliCommand()
	if cmd == nil {
		t.Fatalf("NewFlowcliCommand returned nil")
	}
	if cmd.Name != "flow-cli" {
		t.Errorf("Name expected 'flow-cli', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud DevOps Flow CLI for custom pipeline steps" {
		t.Errorf("Short en unexpected: %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "云效流水线 Flow-CLI，用于自定义开发步骤" {
		t.Errorf("Short zh unexpected: %s", zh)
	}
	if cmd.Usage != "aliyun flow-cli <command> [args...]" {
		t.Errorf("Usage unexpected: %s", cmd.Usage)
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

func TestNewFlowcliCommandMetadata(t *testing.T) {
	cmd := NewFlowcliCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "flow-cli" {
		t.Errorf("metadata name expected flow-cli, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
}
