package maxc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

// TestRun_FullChain_Mocked exercises Context.Run end-to-end with every
// external seam stubbed:
//
//   - fake OSS server serves a tiny valid tarball + matching .sha256
//   - loadProfileFunc returns a synthetic AK profile (no real config file)
//   - execCommandFunc captures the spawned command and exits 0
//
// Asserts: install side-effect (binary on disk + .version), credentials
// land in cmd.Env, parent-only --profile flag is stripped before forwarding.
func TestRun_FullChain_Mocked(t *testing.T) {
	// 1. Sandbox the install dir + fake the platform so we exercise a known path.
	platform := "linux-amd64"
	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	origCfg := getConfigurePathFunc
	t.Cleanup(func() {
		runtimeGOOSFunc, runtimeGOARCHFunc = origGOOS, origGOARCH
		getConfigurePathFunc = origCfg
	})
	tmp := t.TempDir()
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	getConfigurePathFunc = func() string { return tmp }

	// 2. Build a working tarball and stand up a fake OSS bucket for it.
	tarBytes := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "#!/bin/sh\necho ok\n", Mode: 0o755},
	})
	sha := sha256Hex(tarBytes)
	mux := http.NewServeMux()
	mux.HandleFunc("/versions/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0.7.0\n"))
	})
	mux.HandleFunc("/0.7.0/"+platform+"/maxc.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarBytes)
	})
	mux.HandleFunc("/0.7.0/"+platform+"/maxc.tar.gz.sha256", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sha + "\n"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", srv.URL)

	// 3. Stub profile loading — avoid touching any real ~/.aliyun config.
	origLoad := loadProfileFunc
	t.Cleanup(func() { loadProfileFunc = origLoad })
	loadProfileFunc = func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "test",
			Mode:            config.AK,
			AccessKeyId:     "test-id",
			AccessKeySecret: "test-secret",
			RegionId:        "cn-hangzhou",
		}, nil
	}

	// 4. Capture what Execute would spawn and force a clean exit.
	type spawned struct {
		name string
		args []string
		cmd  *exec.Cmd
	}
	var got spawned
	origExec := execCommandFunc
	t.Cleanup(func() { execCommandFunc = origExec })
	execCommandFunc = func(name string, arg ...string) *exec.Cmd {
		got.name = name
		got.args = append([]string(nil), arg...)
		got.cmd = exec.Command("/bin/sh", "-c", "exit 0")
		return got.cmd
	}

	// 5. Drive the full pipeline.
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	originCtx := cli.NewCommandContext(out, errOut)
	c := NewContext(originCtx)

	args := []string{"--profile", "prod", "query", "--sql", "select 1"}
	if err := c.Run(args); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 6. Install side-effects landed in our tmpdir.
	wantExec := filepath.Join(tmp, "maxc", "maxc")
	if _, err := os.Stat(wantExec); err != nil {
		t.Errorf("expected installed binary at %s: %v", wantExec, err)
	}
	if v, _ := os.ReadFile(filepath.Join(tmp, "maxc", ".version")); strings.TrimSpace(string(v)) != "0.7.0" {
		t.Errorf(".version = %q, want 0.7.0", string(v))
	}

	// 7. Execute saw the right binary + stripped --profile from args.
	if got.name != wantExec {
		t.Errorf("execCommand name = %q, want %q", got.name, wantExec)
	}
	wantArgs := []string{"query", "--sql", "select 1"}
	if strings.Join(got.args, "|") != strings.Join(wantArgs, "|") {
		t.Errorf("execCommand args = %v, want %v (--profile must be stripped)", got.args, wantArgs)
	}

	// 8. Credentials made it into the child env.
	envStr := strings.Join(got.cmd.Env, "\n")
	for _, want := range []string{
		"ALIBABA_CLOUD_ACCESS_KEY_ID=test-id",
		"ALIBABA_CLOUD_ACCESS_KEY_SECRET=test-secret",
		"MAXCOMPUTE_REGION=cn-hangzhou",
	} {
		if !strings.Contains(envStr, want) {
			t.Errorf("env missing %q\nfull env:\n%s", want, envStr)
		}
	}
}

// TestRun_UnsupportedPlatform_FailsFast verifies we never download or exec
// on a platform we don't ship tarballs for.
func TestRun_UnsupportedPlatform_FailsFast(t *testing.T) {
	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	origCfg := getConfigurePathFunc
	t.Cleanup(func() {
		runtimeGOOSFunc, runtimeGOARCHFunc = origGOOS, origGOARCH
		getConfigurePathFunc = origCfg
	})
	getConfigurePathFunc = func() string { return t.TempDir() }
	runtimeGOOSFunc = func() string { return "freebsd" }
	runtimeGOARCHFunc = func() string { return "amd64" }

	c := NewContext(cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{}))
	err := c.Run(nil)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected 'not supported' error, got %v", err)
	}
}
