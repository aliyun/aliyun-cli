package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCasesFromTLCDump(t *testing.T) {
	dumpPath := filepath.Join("..", "..", "formal", "parser", "generated", "abstract_cases.dump")
	content, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("read dump: %v", err)
	}

	abstractCases, err := ParseDump(content)
	if err != nil {
		t.Fatalf("parse dump: %v", err)
	}
	if len(abstractCases) <= 65 {
		t.Fatalf("abstract case count = %d, want regenerated complete token-class space, not old 65-case pilot", len(abstractCases))
	}

	generated, err := BuildGeneratedCases(abstractCases)
	if err != nil {
		t.Fatalf("build generated cases: %v", err)
	}
	if len(generated.Cases) != len(abstractCases) {
		t.Fatalf("generated case count = %d, want %d", len(generated.Cases), len(abstractCases))
	}

	var found *GeneratedCase
	for i := range generated.Cases {
		c := &generated.Cases[i]
		if c.StateName == "pending_PortRange" && c.TokenRaw == "-1/-1" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatalf("missing pending_PortRange/-1/-1 case")
	}
	wantArgv := []string{"--PortRange", "-1/-1"}
	if !sameStrings(found.Argv, wantArgv) {
		t.Fatalf("argv = %#v, want %#v", found.Argv, wantArgv)
	}
	if !found.AllowedDashValueEnhancement {
		t.Fatalf("allowedDashValueEnhancement = false, want true")
	}
	if found.Expected.Err.Text != "" {
		t.Fatalf("expected err text = %q, want empty", found.Expected.Err.Text)
	}
	portRange := found.Expected.Flags["PortRange"]
	if !portRange.Assigned || portRange.Value != "-1/-1" {
		t.Fatalf("expected PortRange = %#v, want assigned value -1/-1", portRange)
	}

	coverage := BuildCoverage(generated.Cases)
	if coverage.TotalCases != len(generated.Cases) {
		t.Fatalf("coverage total = %d, want %d", coverage.TotalCases, len(generated.Cases))
	}
	if coverage.AllowedEnhancementCases == 0 {
		t.Fatalf("coverage enhancement count = 0, want at least one dash-leading enhancement case")
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
