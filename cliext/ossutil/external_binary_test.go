package ossutil

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func forbidManagedOssutil(t *testing.T) {
	t.Helper()
	oldPath, oldVersion, oldDownload := getConfigurePathFunc, getLatestOssUtilVersionFunc, downloadAndUnzipFunc
	t.Cleanup(func() {
		getConfigurePathFunc, getLatestOssUtilVersionFunc, downloadAndUnzipFunc = oldPath, oldVersion, oldDownload
	})
	getConfigurePathFunc = func() string {
		t.Fatal("external binary must not initialize managed installation paths")
		return ""
	}
	getLatestOssUtilVersionFunc = func() (string, error) {
		t.Fatal("external binary must not check the remote version")
		return "", nil
	}
	downloadAndUnzipFunc = func(string, string, string, string) error {
		t.Fatal("external binary must not download or install")
		return nil
	}
}

func TestRun_ExternalBinary(t *testing.T) {
	oldArch := runtimeGOARCHFunc
	runtimeGOARCHFunc = func() string { return "unsupported" }
	t.Cleanup(func() { runtimeGOARCHFunc = oldArch })
	for _, exitCode := range []int{0, 7} {
		t.Run(strconv.Itoa(exitCode), func(t *testing.T) {
			forbidManagedOssutil(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			prepareConfig(t, home, "en")
			t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "")
			t.Setenv("ALIBABACLOUD_IGNORE_PROFILE", "")
			t.Setenv("ALIBABA_CLOUD_OSSUTIL_EXEC_PATH", " "+os.Args[0]+" ")
			t.Setenv("ALIYUN_OSSUTIL_TEST_CHILD_EXIT", strconv.Itoa(exitCode))
			ctx, stdout, stderr := newOriginCtx()
			addConfigFlag(ctx, "profile", "default")
			wantArgs := []string{"ls", "oss://bucket/path with spaces"}
			oldExec := execCommandFunc
			t.Cleanup(func() { execCommandFunc = oldExec })
			execCommandFunc = func(name string, args ...string) *exec.Cmd {
				if name != os.Args[0] || !reflect.DeepEqual(args, wantArgs) {
					t.Fatalf("unexpected invocation: %s %q", name, args)
				}
				return exec.Command(name, append([]string{"-test.run=^TestOssutilExternalBinaryProcess$", "--"}, args...)...)
			}
			input, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			oldStdin := os.Stdin
			os.Stdin = input
			t.Cleanup(func() { os.Stdin = oldStdin; _ = input.Close() })
			if _, err := io.WriteString(writer, "stdin payload"); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			err = NewContext(ctx).Run(append(append([]string{}, wantArgs...), "--profile", "default"))
			if exitCode == 0 && err != nil {
				t.Fatal(err)
			}
			if exitCode != 0 && (err == nil || !strings.Contains(err.Error(), "exit status 7")) {
				t.Fatalf("expected existing subprocess failure semantics, got %v", err)
			}
			var result struct {
				Args   []string
				Input  string
				Compat string
				Config string
			}
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("subprocess stdout %q: %v", stdout.String(), err)
			}
			if !reflect.DeepEqual(result.Args, wantArgs) || result.Input != "stdin payload" || result.Compat != "alicli" || stderr.String() != "external diagnostic" {
				t.Fatalf("subprocess context changed: %+v, stderr=%q", result, stderr.String())
			}
			payload, err := base64.StdEncoding.DecodeString(result.Config)
			if err != nil {
				t.Fatal(err)
			}
			var credentials map[string]any
			if err := json.Unmarshal(payload, &credentials); err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]string{"mode": "AK", "access_key_id": "ak", "access_key_secret": "sk", "region_id": "cn-hangzhou"} {
				if credentials[key] != want {
					t.Errorf("credential %s = %v, want %s", key, credentials[key], want)
				}
			}
			for _, filename := range []string{"ossutil", "ossutil.exe", ".ossutil_version_check"} {
				if _, err := os.Stat(filepath.Join(home, ".aliyun", filename)); !os.IsNotExist(err) {
					t.Errorf("external binary created managed file %s: %v", filename, err)
				}
			}
		})
	}
}

func TestExternalBinaryRelativePath(t *testing.T) {
	forbidManagedOssutil(t)
	t.Chdir(t.TempDir())
	writeExecutable(t, "ossutil", "")
	t.Setenv("ALIBABA_CLOUD_OSSUTIL_EXEC_PATH", "ossutil")
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	c := NewContext(nil)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if err := c.validateExecPathOverride(); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "ossutil"); c.execFilePath != want {
		t.Fatalf("executable path = %q, want %q", c.execFilePath, want)
	}
}

func TestExternalBinaryRemovedWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not allow removing the current working directory")
	}
	forbidManagedOssutil(t)
	dir := filepath.Join(t.TempDir(), "removed-cwd")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Getwd(); err == nil {
		t.Skip("the host can still resolve a removed working directory")
	}
	t.Setenv("ALIBABA_CLOUD_OSSUTIL_EXEC_PATH", "ossutil")
	ctx, _, _ := newOriginCtx()
	err := NewContext(ctx).Run([]string{"ls"})
	var pathErr *os.PathError
	if err == nil || !strings.Contains(err.Error(), `ALIBABA_CLOUD_OSSUTIL_EXEC_PATH="ossutil"`) || !errors.As(err, &pathErr) || pathErr.Op != "getwd" {
		t.Fatalf("expected wrapped working-directory error, got %v", err)
	}
}

func TestExternalBinaryWindowsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ossutil.exe")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	c := &Context{execFilePath: path, osType: "windows"}
	if err := c.validateExecPathOverride(); err != nil {
		t.Fatalf("Windows must not require Unix executable permission bits: %v", err)
	}
}

func TestOssutilExternalBinaryProcess(t *testing.T) {
	code := os.Getenv("ALIYUN_OSSUTIL_TEST_CHILD_EXIT")
	if code == "" {
		return
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(99)
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) > 0 {
		args = args[1:]
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"Args": args, "Input": string(input), "Compat": os.Getenv("OSSUTIL_COMPAT_MODE"), "Config": os.Getenv("OSSUTIL_CONFIG_VALUE"),
	})
	_, _ = fmt.Fprint(os.Stderr, "external diagnostic")
	exitCode, _ := strconv.Atoi(code)
	os.Exit(exitCode)
}

func TestRun_ExternalBinaryInvalid(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"missing", "existing file"}, {"directory", "regular file"}, {"not executable", "not executable"},
		{"symlink loop", "symbolic links"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forbidManagedOssutil(t)
			if (tc.name == "not executable" || tc.name == "symlink loop") && runtime.GOOS == "windows" {
				t.Skip("Unix executable permissions and symlink-loop behavior")
			}
			path := filepath.Join(t.TempDir(), "ossutil")
			if tc.name == "directory" {
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			} else if tc.name == "not executable" {
				if err := os.WriteFile(path, nil, 0644); err != nil {
					t.Fatal(err)
				}
			} else if tc.name == "symlink loop" {
				if err := os.Symlink("ossutil", path); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("ALIBABA_CLOUD_OSSUTIL_EXEC_PATH", path)
			ctx, _, _ := newOriginCtx()
			err := NewContext(ctx).Run([]string{"ls"})
			for _, want := range []string{"ALIBABA_CLOUD_OSSUTIL_EXEC_PATH", strconv.Quote(path), tc.want} {
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("error %v should contain %q", err, want)
				}
			}
			if tc.name == "symlink loop" {
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) || pathErr.Op != "stat" || os.IsNotExist(pathErr) {
					t.Fatalf("expected wrapped non-missing stat error, got %v", err)
				}
			}
		})
	}
}
