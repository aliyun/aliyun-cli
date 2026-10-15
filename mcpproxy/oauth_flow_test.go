package mcpproxy

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func authorizationInput(t *testing.T, input string, interactive bool) {
	t.Helper()
	previousStdin, previousInteractive := os.Stdin, isInteractiveInput
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	require.NoError(t, err)
	_, err = file.WriteString(input)
	require.NoError(t, err)
	_, err = file.Seek(0, 0)
	require.NoError(t, err)
	os.Stdin = file
	isInteractiveInput = func() bool { return interactive }
	t.Cleanup(func() { os.Stdin = previousStdin; isInteractiveInput = previousInteractive; _ = file.Close() })
}

func fakeBrowser(t *testing.T, fail bool, afterStart func()) {
	t.Helper()
	previous := browserCommand
	var commands []*exec.Cmd
	browserCommand = func(_ string, args ...string) *exec.Cmd {
		require.NotEmpty(t, args)
		if afterStart != nil {
			go afterStart()
		}
		command := exec.Command(os.Args[0], "-test.run=^$")
		if fail {
			command = exec.Command("/nonexistent-browser")
		}
		commands = append(commands, command)
		return command
	}
	t.Cleanup(func() {
		browserCommand = previous
		for _, command := range commands {
			if command.Process != nil {
				_ = command.Wait()
			}
		}
	})
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

func TestOAuthFlowManualAndCallback(t *testing.T) {
	for _, auto := range []bool{false, true} {
		t.Run(map[bool]string{false: "manual", true: "callback"}[auto], func(t *testing.T) {
			authorizationInput(t, "authorization-code\n", true)
			manager := NewOAuthCallbackManager()
			if auto {
				fakeBrowser(t, false, func() {
					deadline := time.Now().Add(time.Second)
					for !manager.IsWaiting() && time.Now().Before(deadline) {
						time.Sleep(time.Millisecond)
					}
					manager.HandleCallback("authorization-code", nil)
				})
			}
			oauthServer(t, func(w http.ResponseWriter, req *http.Request) {
				require.NoError(t, req.ParseForm())
				require.Equal(t, "authorization-code", req.Form.Get("code"))
				require.NotEmpty(t, req.Form.Get("code_verifier"))
				require.Equal(t, "http://127.0.0.1:8088/callback", req.Form.Get("redirect_uri"))
				_, _ = io.WriteString(w, `{"access_token":"access","refresh_token":"refresh","expires_in":1000}`)
			})
			out := &bytes.Buffer{}
			ctx := cli.NewCommandContext(out, out)
			result, err := startMCPOAuthFlowWithManager(ctx, "app", RegionCN, manager, "127.0.0.1", 8088, auto, "scope")
			require.NoError(t, err)
			require.Equal(t, "access", result.AccessToken)
			require.False(t, manager.IsWaiting())
			require.Contains(t, out.String(), "OAuth login successful")
		})
	}
}

func TestOAuthFlowInputAndCallbackFailures(t *testing.T) {
	for _, tc := range []struct {
		name, input, want                 string
		interactive, browserFailure, auto bool
	}{
		{name: "manual noninteractive", want: "not interactive"},
		{name: "manual EOF", interactive: true, want: "failed to read authorization code"},
		{name: "browser failure noninteractive", auto: true, browserFailure: true, want: "not interactive"},
		{name: "browser failure EOF", auto: true, browserFailure: true, interactive: true, want: "failed to read authorization code"},
		{name: "callback timeout", auto: true, want: "failed to get authorization code"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			authorizationInput(t, tc.input, tc.interactive)
			fakeBrowser(t, tc.browserFailure, nil)
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

func TestOAuthFlowFallsBackAfterBrowserFailure(t *testing.T) {
	authorizationInput(t, "manual-code\n", true)
	fakeBrowser(t, true, nil)
	oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"fallback","expires_in":10}`)
	})
	result, err := executeOAuthFlow(cli.NewCommandContext(io.Discard, io.Discard), "app", RegionCN, NewOAuthCallbackManager(), "localhost", 0, true, "scope", nil)
	require.NoError(t, err)
	require.Equal(t, "fallback", result.AccessToken)
	require.Empty(t, result.RefreshToken)
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
			authorizationInput(t, "code\n", !tc.inputError)
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
