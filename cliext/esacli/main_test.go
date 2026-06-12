package esacli

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewEsacliCommand(t *testing.T) {
	cmd := NewEsacliCommand()
	if cmd == nil {
		t.Fatalf("NewEsacliCommand returned nil")
	}
	if cmd.Name != "esa-cli" {
		t.Errorf("Name expected 'esa-cli', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud ESA CLI for Edge Routine development" {
		t.Errorf("Short en unexpected: %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云 ESA 边缘 Routine 开发工具" {
		t.Errorf("Short zh unexpected: %s", zh)
	}
	if cmd.Usage != "aliyun esa-cli <command> [args...]" {
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

func TestNewEsacliCommandMetadata(t *testing.T) {
	cmd := NewEsacliCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "esa-cli" {
		t.Errorf("metadata name expected esa-cli, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
}
