package mcpproxy

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if name := os.Getenv("MCP_BROWSER_HELPER"); name != "" && browserExecutableName(os.Args[0]) == browserExecutableName(name) {
		pidFile := os.Getenv("MCP_BROWSER_PID")
		if err := os.WriteFile(pidFile+".tmp", []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			os.Exit(2)
		}
		if err := os.Rename(pidFile+".tmp", pidFile); err != nil {
			os.Exit(2)
		}
		for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
			if _, err := os.Stat(pidFile + ".done"); err == nil {
				os.Exit(0)
			}
			time.Sleep(time.Millisecond)
		}
		os.Exit(3)
	}
	os.Exit(m.Run())
}

func browserExecutableName(path string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(path)), ".exe")
}

func TestBrowserExecutableNameMatchesWindowsCommandArguments(t *testing.T) {
	for _, argument := range []string{"rundll32", "rundll32.exe", "RUNDLL32.EXE", filepath.Join("directory", "rundll32.exe")} {
		require.Equal(t, "rundll32", browserExecutableName(argument))
	}
}

func authorizationInput(t *testing.T, interactive bool) {
	t.Helper()
	previousStdin := os.Stdin
	var file *os.File
	var err error
	if interactive {
		file, err = os.Open(os.DevNull)
	} else {
		file, err = os.CreateTemp(t.TempDir(), "stdin")
	}
	require.NoError(t, err)
	os.Stdin = file
	t.Cleanup(func() { os.Stdin = previousStdin; _ = file.Close() })
}

func fakeBrowser(t *testing.T, fail bool, callback func()) {
	t.Helper()
	directory := t.TempDir()
	if root := os.Getenv("MCP_TEST_EXEC_ROOT"); root != "" {
		require.NoError(t, os.MkdirAll(root, 0700))
		var err error
		directory, err = os.MkdirTemp(root, "mcp-browser-")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(directory)) })
	}
	t.Setenv("PATH", directory)
	if fail {
		return
	}
	name := map[string]string{"linux": "xdg-open", "darwin": "open", "windows": "rundll32.exe"}[runtime.GOOS]
	require.NotEmpty(t, name)
	executable, err := os.Executable()
	require.NoError(t, err)
	path := filepath.Join(directory, name)
	linked := false
	// Windows keeps the parent test executable mapped, so its hard links cannot
	// be removed even after the browser helper has exited.
	if runtime.GOOS != "windows" {
		linked = os.Link(executable, path) == nil
	}
	if !linked {
		source, err := os.Open(executable)
		require.NoError(t, err)
		defer source.Close()
		target, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0700)
		require.NoError(t, err)
		_, copyErr := io.Copy(target, source)
		require.NoError(t, target.Close())
		require.NoError(t, copyErr)
	}
	pidFile := filepath.Join(directory, "browser.pid")
	t.Setenv("MCP_BROWSER_HELPER", name)
	t.Setenv("MCP_BROWSER_PID", pidFile)
	started := make(chan struct{})
	go func() {
		defer close(started)
		if callback != nil {
			callback()
		}
	}()
	t.Cleanup(func() {
		<-started
		require.Eventually(t, func() bool { _, err := os.Stat(pidFile); return err == nil }, 15*time.Second, time.Millisecond)
		data, err := os.ReadFile(pidFile)
		require.NoError(t, err)
		pid, err := strconv.Atoi(string(data))
		require.NoError(t, err)
		process, err := os.FindProcess(pid)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(pidFile+".done", nil, 0600))
		state, err := process.Wait()
		require.NoError(t, err)
		require.True(t, state.Success())
	})
}

func TestFakeBrowserLifecycleRemovesExitedExecutable(t *testing.T) {
	var path string
	t.Run("browser process", func(t *testing.T) {
		fakeBrowser(t, false, nil)
		path = filepath.Join(os.Getenv("PATH"), os.Getenv("MCP_BROWSER_HELPER"))
		require.NoError(t, OpenBrowser("https://browser.example.invalid/callback"))
		if runtime.GOOS == "windows" {
			executable, err := os.Executable()
			require.NoError(t, err)
			parentInfo, err := os.Stat(executable)
			require.NoError(t, err)
			helperInfo, err := os.Stat(path)
			require.NoError(t, err)
			require.False(t, os.SameFile(parentInfo, helperInfo), "the helper must own a separate executable image")
		}
	})
	require.NoFileExists(t, path, "helper executable should be removable after the process is reaped")
}

func callbackAuthorization(t *testing.T, manager *OAuthCallbackManager, code string) {
	t.Helper()
	stop := make(chan struct{})
	fakeBrowser(t, false, func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if manager.IsWaiting() {
				manager.HandleCallback(code, nil)
				return
			}
			time.Sleep(time.Millisecond)
		}
	})
	t.Cleanup(func() { close(stop) })
}

func callbackAuthorizationRequest(t *testing.T, code string) {
	t.Helper()
	stop := make(chan struct{})
	fakeBrowser(t, false, func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			response := httptest.NewRecorder()
			http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest("GET", "/callback?code="+code, nil))
			time.Sleep(time.Millisecond)
		}
	})
	t.Cleanup(func() { close(stop) })
}

func TestAuthorizationCodeInputValidation(t *testing.T) {
	out := &bytes.Buffer{}
	code, err := promptAuthorizationCode(out, bufio.NewReader(strings.NewReader("\nhttps://localhost/callback?code=bad\ncode=bad\n  only-code \n")))
	require.NoError(t, err)
	require.Equal(t, "only-code", code)
	require.Contains(t, out.String(), "Input is empty")
	require.Contains(t, out.String(), "authorization code only")
	_, err = promptAuthorizationCode(io.Discard, bufio.NewReader(strings.NewReader("")))
	require.ErrorIs(t, err, io.EOF)
}

func TestOAuthFlowCallbackPreservesPKCE(t *testing.T) {
	manager := NewOAuthCallbackManager()
	callbackAuthorization(t, manager, "authorization-code")
	oauthServer(t, func(w http.ResponseWriter, req *http.Request) {
		require.NoError(t, req.ParseForm())
		require.Equal(t, "authorization-code", req.Form.Get("code"))
		require.NotEmpty(t, req.Form.Get("code_verifier"))
		require.Equal(t, "http://127.0.0.1:8088/callback", req.Form.Get("redirect_uri"))
		_, _ = io.WriteString(w, `{"access_token":"access","refresh_token":"refresh","expires_in":1000}`)
	})
	out := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, out)
	result, err := startMCPOAuthFlowWithManager(ctx, "app", RegionCN, manager, "127.0.0.1", 8088, true, "scope")
	require.NoError(t, err)
	require.Equal(t, "access", result.AccessToken)
	require.False(t, manager.IsWaiting())
	require.Contains(t, out.String(), "OAuth login successful")
}

func TestOAuthFlowInputAndCallbackFailures(t *testing.T) {
	for _, tc := range []struct {
		name, want                        string
		interactive, browserFailure, auto bool
	}{
		{name: "manual noninteractive", want: "not interactive"},
		{name: "manual EOF", interactive: true, want: "failed to read authorization code"},
		{name: "browser failure noninteractive", auto: true, browserFailure: true, want: "not interactive"},
		{name: "browser failure EOF", auto: true, browserFailure: true, interactive: true, want: "failed to read authorization code"},
		{name: "callback timeout", auto: true, want: "failed to get authorization code"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			authorizationInput(t, tc.interactive)
			if tc.auto {
				fakeBrowser(t, tc.browserFailure, nil)
			}
			previous := oauthTimeout
			oauthTimeout = 20 * time.Millisecond
			t.Cleanup(func() { oauthTimeout = previous })
			manager := NewOAuthCallbackManager()
			_, err := startMCPOAuthFlowWithManager(cli.NewCommandContext(io.Discard, io.Discard), "app", RegionCN, manager, "localhost", 0, tc.auto, "scope")
			require.ErrorContains(t, err, tc.want)
			require.False(t, manager.IsWaiting())
		})
	}
}

func TestTeeWriterPreservesErrorsAndOutput(t *testing.T) {
	first, second := &bytes.Buffer{}, &bytes.Buffer{}
	writer := &teeWriter{writers: []io.Writer{first, second}}
	n, err := writer.Write([]byte("prompt"))
	require.NoError(t, err)
	require.Equal(t, 6, n)
	require.Equal(t, "prompt", first.String())
	require.Equal(t, first.String(), second.String())
	writer.writers = []io.Writer{errorWriter{}}
	_, err = writer.Write([]byte("prompt"))
	require.ErrorContains(t, err, "write failed")
	previous := os.Stderr
	file, err := os.CreateTemp(t.TempDir(), "stderr")
	require.NoError(t, err)
	os.Stderr = file
	t.Cleanup(func() { os.Stderr = previous; _ = file.Close() })
	require.True(t, isStderrRedirected())
	require.NotNil(t, getStderrWriter(nil))
	require.NoError(t, file.Close())
	require.True(t, isStderrRedirected())
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestReauthorizationUpdatesProfileAndReportsFailures(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		inputError, saveFailure, check bool
	}{
		{name: "success"},
		{name: "manual input failure", inputError: true},
		{name: "save failure", saveFailure: true},
		{name: "scheduled reauthorization", check: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxy := testProxy(t)
			r := proxy.TokenRefresher
			r.profile.MCPOAuthRefreshTokenValidity = 9000
			if tc.check {
				r.profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() + 1
			}
			if tc.inputError {
				authorizationInput(t, false)
			} else {
				r.autoOpenBrowser = true
				callbackAuthorization(t, r.callbackManager, "code")
			}
			oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"access_token":"reauthorized","refresh_token":"new-refresh","expires_in":1000}`)
			})
			if tc.saveFailure {
				require.NoError(t, os.MkdirAll(getMCPConfigPath()+".tmp", 0700))
			}
			before := time.Now().Unix()
			var err error
			if tc.check {
				r.checkAndRefresh()
			} else {
				err = r.reauthorizeWithProxy()
			}
			if tc.inputError {
				require.ErrorContains(t, err, "OAuth re-authorization failed")
				require.Equal(t, int64(1), r.stats.TokenRefreshErrors)
			} else if tc.saveFailure {
				require.ErrorContains(t, err, "failed to save reauthorized tokens")
				require.Error(t, <-r.fatalErrCh)
			} else {
				require.NoError(t, err)
				require.Equal(t, "reauthorized", r.profile.MCPOAuthAccessToken)
				require.GreaterOrEqual(t, r.profile.MCPOAuthRefreshTokenExpire, before+9000)
				require.Equal(t, int64(1), r.stats.TokenRefreshes)
				require.Equal(t, "reauthorized", (<-r.tokenCh).Token)
			}
			require.False(t, r.reauthorizing)
		})
	}
}
