package releasever

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestClassifyAndPlan(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		wantKind      Kind
		wantLatest    bool
		wantVersion   bool
		wantOfficial  bool
		wantErrSubstr string
	}{
		{name: "stable", version: "3.5.1", wantKind: KindStable, wantLatest: true, wantVersion: true, wantOfficial: true},
		{name: "stable tag", version: "v3.5.1", wantKind: KindStable, wantLatest: true, wantVersion: true, wantOfficial: true},
		{name: "stable with spaces trimmed", version: "  3.5.1  ", wantKind: KindStable, wantLatest: true, wantVersion: true, wantOfficial: true},
		{name: "historical beta", version: "3.5.1-beta", wantKind: KindPrerelease},
		{name: "beta tag", version: "v3.5.1-beta", wantKind: KindPrerelease},
		{name: "beta.1", version: "3.5.1-beta.1", wantKind: KindPrerelease},
		{name: "rc.1", version: "3.5.1-rc.1", wantKind: KindPrerelease},
		{name: "rc.1 tag", version: "v3.5.1-rc.1", wantKind: KindPrerelease},
		{name: "rc without number", version: "3.5.1-rc", wantKind: KindPrerelease},
		{name: "alpha.1", version: "3.5.1-alpha.1", wantKind: KindPrerelease},
		{name: "preview", version: "3.5.1-preview.2", wantKind: KindPrerelease},

		{name: "empty", version: "", wantErrSubstr: "empty"},
		{name: "latest", version: "latest", wantErrSubstr: "invalid release version"},
		{name: "missing patch", version: "3.5", wantErrSubstr: "invalid release version"},
		{name: "major only", version: "3", wantErrSubstr: "invalid release version"},
		{name: "bare v", version: "v", wantErrSubstr: "invalid release version"},
		{name: "trailing hyphen", version: "3.5.1-", wantErrSubstr: "invalid release version"},
		{name: "trailing dot", version: "3.5.1-beta.", wantErrSubstr: "invalid release version"},
		{name: "leading zero prerelease", version: "3.5.1-01", wantErrSubstr: "invalid release version"},
		{name: "leading zero rc", version: "3.5.1-rc.01", wantErrSubstr: "invalid release version"},
		{name: "not semver", version: "not-a-version", wantErrSubstr: "invalid release version"},
		{name: "internal space", version: "3.5.1 beta", wantErrSubstr: "invalid release version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, err := Classify(tt.version)
			plan, planErr := ReleasePlan(tt.version)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("Classify(%q) err = %v, want substring %q", tt.version, err, tt.wantErrSubstr)
				}
				if planErr == nil || !strings.Contains(planErr.Error(), tt.wantErrSubstr) {
					t.Fatalf("ReleasePlan(%q) err = %v, want substring %q", tt.version, planErr, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Classify(%q) unexpected err: %v", tt.version, err)
			}
			if kind != tt.wantKind {
				t.Fatalf("Classify(%q) = %q, want %q", tt.version, kind, tt.wantKind)
			}
			if planErr != nil {
				t.Fatalf("ReleasePlan(%q) unexpected err: %v", tt.version, planErr)
			}
			if plan.Kind != tt.wantKind || plan.UpdateLatest != tt.wantLatest || plan.WriteStableVersion != tt.wantVersion || plan.MarkOfficial != tt.wantOfficial {
				t.Fatalf("ReleasePlan(%q) = %+v, want kind=%s latest=%v version=%v official=%v",
					tt.version, plan, tt.wantKind, tt.wantLatest, tt.wantVersion, tt.wantOfficial)
			}
		})
	}
}

func TestPrereleaseDoesNotPromoteStable(t *testing.T) {
	for _, version := range []string{"3.5.1-beta", "3.5.1-beta.1", "3.5.1-rc.1", "v3.5.1-rc.1"} {
		plan, err := ReleasePlan(version)
		if err != nil {
			t.Fatalf("ReleasePlan(%q): %v", version, err)
		}
		if plan.UpdateLatest || plan.WriteStableVersion || plan.MarkOfficial {
			t.Fatalf("%s must not update latest/stable or become official: %+v", version, plan)
		}
	}
}

func TestExecuteClassifyAndPlan(t *testing.T) {
	t.Run("classify stable", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "classify", "3.5.1")
		if code != 0 || stdout != "stable\n" || stderr != "" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
	t.Run("classify prerelease", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "classify", "--", "3.5.1-rc.1")
		if code != 0 || stdout != "prerelease\n" || stderr != "" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
	t.Run("classify invalid", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "classify", "latest")
		if code == 0 || stdout != "" || !strings.Contains(stderr, "invalid release version") {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
	t.Run("plan stable", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "plan", "3.5.1")
		if code != 0 || stderr != "" {
			t.Fatalf("code=%d stderr=%q", code, stderr)
		}
		assertPlan(t, stdout, "stable", "1", "1", "1")
	})
	t.Run("plan beta.1", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "plan", "3.5.1-beta.1")
		if code != 0 || stderr != "" {
			t.Fatalf("code=%d stderr=%q", code, stderr)
		}
		assertPlan(t, stdout, "prerelease", "0", "0", "0")
	})
	t.Run("plan invalid refuses", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "plan", "3.5.1-")
		if code == 0 || stdout != "" || !strings.Contains(stderr, "invalid release version") {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
	t.Run("usage", func(t *testing.T) {
		stdout, stderr, code := runCLI(t)
		if code == 0 || !strings.Contains(stderr, "usage:") {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
	t.Run("unknown command", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "promote", "3.5.1")
		if code == 0 || !strings.Contains(stderr, "unknown command") {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
	t.Run("empty version arg", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "classify", "")
		if code == 0 || !strings.Contains(stderr, "usage:") {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, os.ErrClosed
}

func TestExecutePlanWriteError(t *testing.T) {
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "3.5.1"}, errWriter{}, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "closed") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestWritePlan(t *testing.T) {
	var buf bytes.Buffer
	if err := writePlan(&buf, Plan{Kind: KindStable, UpdateLatest: true, WriteStableVersion: true, MarkOfficial: true}); err != nil {
		t.Fatal(err)
	}
	assertPlan(t, buf.String(), "stable", "1", "1", "1")
}

func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Execute(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func assertPlan(t *testing.T, got, kind, latest, version, official string) {
	t.Helper()
	want := "kind=" + kind + "\n" +
		"update_latest=" + latest + "\n" +
		"write_stable_version=" + version + "\n" +
		"mark_official=" + official + "\n"
	if got != want {
		t.Fatalf("plan =\n%s\nwant:\n%s", got, want)
	}
}
