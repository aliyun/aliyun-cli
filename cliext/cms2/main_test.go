package cms2

import (
	"testing"
)

func TestNewCms2Command(t *testing.T) {
	cmd := NewCms2Command()
	if cmd == nil {
		t.Fatalf("NewCms2Command returned nil")
	}
	if cmd.Name != "cms2" {
		t.Errorf("Name expected 'cms2', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud CloudMonitor (CMS) CLI — manage monitoring integrations, Prometheus, alert rules, and PromQL." {
		t.Errorf("Short en mismatch: %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云云监控 CLI — 管理监控集成、Prometheus 实例、告警规则和 PromQL 查询。" {
		t.Errorf("Short zh mismatch: %s", zh)
	}
	if cmd.Usage != "aliyun cms2 <command> [args...] [options...]" {
		t.Errorf("Usage mismatch: %s", cmd.Usage)
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
