package maxc

import (
	"testing"
)

func TestNewMaxcCommand_BasicShape(t *testing.T) {
	cmd := NewMaxcCommand()
	if cmd == nil {
		t.Fatal("NewMaxcCommand returned nil")
	}
	if cmd.Name != "maxc" {
		t.Errorf("expected Name=maxc, got %q", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatal("Short i18n text nil")
	}
	if cmd.Short.Get("en") == "" {
		t.Error("Short en is empty")
	}
	if cmd.Short.Get("zh") == "" {
		t.Error("Short zh is empty")
	}
	if cmd.Usage != "aliyun maxc <command> [args...] [options...]" {
		t.Errorf("Usage mismatch: %s", cmd.Usage)
	}
	if cmd.Hidden {
		t.Error("Hidden expected false")
	}
	if !cmd.EnableUnknownFlag {
		t.Error("EnableUnknownFlag must be true so aliyun root does not parse maxc subcommand flags")
	}
	if !cmd.KeepArgs {
		t.Error("KeepArgs must be true")
	}
	if !cmd.SkipDefaultHelp {
		t.Error("SkipDefaultHelp must be true")
	}
	if cmd.Run == nil {
		t.Fatal("Run function should not be nil")
	}
}
