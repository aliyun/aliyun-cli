package mock

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInterceptDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	original := `[{"name":"record","cmd":"ecs *","exit_code":0,"stdout":"nope","stderr":"","times":0}]`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv(EnvMockEnabled, "")

	var stdout, stderr bytes.Buffer
	result := Intercept(Options{
		Args:     []string{"ecs", "DescribeRegions"},
		Stdout:   &stdout,
		Stderr:   &stderr,
		MockPath: path,
	})

	if result.Handled {
		t.Fatalf("Handled = true, want false")
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("stdout/stderr = %q/%q, want empty", stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != original {
		t.Fatalf("file content changed: %q", string(data))
	}
}

func TestInterceptHitWritesOutputAndPersistsTimes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := Save(path, []Record{{
		Name:     "hit",
		Cmd:      "ecs *",
		ExitCode: 7,
		Stdout:   "mock stdout\n",
		Stderr:   "mock stderr\n",
		Times:    2,
	}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv(EnvMockEnabled, "true")

	var stdout, stderr bytes.Buffer
	result := Intercept(Options{
		Args:     []string{"ecs", "DescribeRegions"},
		Stdout:   &stdout,
		Stderr:   &stderr,
		MockPath: path,
	})

	if !result.Handled {
		t.Fatalf("Handled = false, want true")
	}
	if result.ExitCode != 7 {
		t.Fatalf("ExitCode = %d, want 7", result.ExitCode)
	}
	if stdout.String() != "mock stdout\n" {
		t.Fatalf("stdout = %q, want mock stdout", stdout.String())
	}
	if stderr.String() != "mock stderr\n" {
		t.Fatalf("stderr = %q, want mock stderr", stderr.String())
	}
	records, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records length = %d, want 1", len(records))
	}
	if records[0].Times != 1 {
		t.Fatalf("Times = %v, want 1", records[0].Times)
	}
}

func TestInterceptMalformedFileIsHandledError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv(EnvMockEnabled, "true")

	var stdout, stderr bytes.Buffer
	result := Intercept(Options{
		Args:     []string{"ecs", "DescribeRegions"},
		Stdout:   &stdout,
		Stderr:   &stderr,
		MockPath: path,
	})

	if !result.Handled {
		t.Fatalf("Handled = false, want true")
	}
	if result.ExitCode == 0 {
		t.Fatalf("ExitCode = 0, want nonzero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); got == "" || !bytes.Contains([]byte(got), []byte("ERROR: load mock data failed ")) {
		t.Fatalf("stderr = %q, want load error", got)
	}
}

func TestInterceptSaveFailureReturnsHandledError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := Save(path, []Record{{
		Name:   "save-fails",
		Cmd:    "ecs *",
		Stdout: "mock stdout before save failure",
		Stderr: "mock stderr before save failure",
		Times:  1,
	}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	originalSaveRecords := saveRecords
	saveRecords = func(string, []Record) error {
		return fmt.Errorf("disk full")
	}
	t.Cleanup(func() {
		saveRecords = originalSaveRecords
	})
	t.Setenv(EnvMockEnabled, "true")

	var stdout, stderr bytes.Buffer
	result := Intercept(Options{
		Args:     []string{"ecs", "DescribeRegions"},
		Stdout:   &stdout,
		Stderr:   &stderr,
		MockPath: path,
	})

	if !result.Handled {
		t.Fatalf("Handled = false, want true")
	}
	if result.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", result.ExitCode)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); got != "ERROR: save mock data failed disk full\n" {
		t.Fatalf("stderr = %q, want save error", got)
	}
}

func TestInterceptMissReturnsUnhandledAndLeavesFileUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := Save(path, []Record{{
		Name:   "miss",
		Cmd:    "ecs DescribeRegions",
		Stdout: "should not write",
		Times:  0,
	}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile original: %v", err)
	}
	t.Setenv(EnvMockEnabled, "true")

	var stdout, stderr bytes.Buffer
	result := Intercept(Options{
		Args:     []string{"ram", "ListUsers"},
		Stdout:   &stdout,
		Stderr:   &stderr,
		MockPath: path,
	})

	if result.Handled {
		t.Fatalf("Handled = true, want false")
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("stdout/stderr = %q/%q, want empty", stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("file content changed: %q", string(data))
	}
}

func TestInterceptOneShotRecordIsRemovedAfterHit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	if err := Save(path, []Record{{
		Name:   "once",
		Cmd:    "ecs *",
		Stdout: "once",
		Times:  1,
	}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv(EnvMockEnabled, "true")

	var stdout, stderr bytes.Buffer
	result := Intercept(Options{
		Args:     []string{"ecs", "DescribeRegions"},
		Stdout:   &stdout,
		Stderr:   &stderr,
		MockPath: path,
	})

	if !result.Handled {
		t.Fatalf("Handled = false, want true")
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if stdout.String() != "once" {
		t.Fatalf("stdout = %q, want once", stdout.String())
	}
	records, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("records length = %d, want 0", len(records))
	}
}

func TestWritesAndWritefIgnoreNilWriter(t *testing.T) {
	writes(nil, "ignored")
	writef(nil, "ignored %s", "value")
}
