package lib

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetestHostHelp(t *testing.T) {
	for _, language := range []string{"en", "zh"} {
		old := i18n.GetLanguage()
		i18n.SetLanguage(language)
		t.Cleanup(func() { i18n.SetLanguage(old) })
		for _, args := range [][]string{{"oss"}, {"oss", "--help"}, {"oss", "help"}, {"oss", "help", "cp"}, {"oss", "cp", "--help"}, {"oss", "ls", "--help"}} {
			t.Run(language+strings.Join(args, "/"), func(t *testing.T) {
				var out, errout bytes.Buffer
				root := &cli.Command{Name: "aliyun"}
				// Simulate overlapping persistent host flags without relying on main.
				addMachineFlags(root.Flags())
				root.AddSubCommand(NewOssCommand())
				root.Execute(cli.NewCommandContext(&out, &errout), args)
				require.Empty(t, errout.String())
				text := out.String()
				assert.NotContains(t, text, "ossutil cp")
				assert.NotContains(t, text, "ossutil ls")
				assert.NotContains(t, text, "\n  update ")
				assert.Contains(t, text, "aliyun oss ls oss://bucket/prefix/")
				for _, name := range []string{"cli-output", "cli-cursor", "cli-validate", "cli-plan", "cli-failure-report", "cli-ai-mode", "cli-non-interactive"} {
					count := 0
					for _, line := range strings.Split(text, "\n") {
						if strings.HasPrefix(strings.TrimSpace(line), "--"+name+" ") {
							count++
						}
					}
					assert.Equal(t, 1, count, name)
				}
			})
		}
	}
}

func TestRetestEndpointRecovery(t *testing.T) {
	m := &machineInvocation{ai: true, phase: "execution", command: "ls"}
	for _, tc := range []struct{ ec, message, action string }{
		{"0003-00001403", "denied", "check_endpoint"},
		{"", "The bucket you are attempting to access must be addressed using the specified endpoint.", "check_endpoint"},
		{"", "Access denied by policy", "check_permissions"},
	} {
		err := fmt.Errorf("wrapped: %w", &oss.ServiceError{Code: "AccessDenied", StatusCode: 403, Ec: tc.ec, Message: tc.message, RequestID: "request"})
		got := m.adaptError(err).(*ossAgentError).Envelope()
		assert.Equal(t, tc.action, got.Recovery.Action)
		assert.Equal(t, "AccessDenied", got.ErrorCode)
		assert.Equal(t, "request", got.RequestId)
	}
}

func TestRetestConfigEOFDoesNotWrite(t *testing.T) {
	stdin, err := os.Open(os.DevNull)
	require.NoError(t, err)
	old := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() { os.Stdin = old; stdin.Close() })
	path := filepath.Join(t.TempDir(), "config")
	for _, existing := range []bool{false, true} {
		if existing {
			require.NoError(t, os.WriteFile(path, []byte("keep existing"), 0600))
		}
		ctx := cli.NewCommandContext(io.Discard, io.Discard)
		ctx.SetCommand(NewCommandBridge(configCommand.command))
		err := parseAndRunCommandFromCli(ctx, []string{"--config-file", path}, &configCommand.command)
		require.ErrorContains(t, err, "interactive terminal")
		data, readErr := os.ReadFile(path)
		if existing {
			require.NoError(t, readErr)
			assert.Equal(t, "keep existing", string(data))
		} else {
			require.True(t, os.IsNotExist(readErr))
		}
	}
}

func TestRetestPlanAllowsUserAgent(t *testing.T) {
	clearEndpointTestEnv(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, http.MethodHead, r.Method)
		assert.Contains(t, r.UserAgent(), "review-agent")
		w.Header().Set("ETag", `"etag"`)
	}))
	defer server.Close()
	output, err := phaseCInvoke(t, removeCommand.command, server.URL, "oss://bucket/key", "--cli-plan", "--ua", "review-agent")
	require.NoError(t, err)
	assert.Contains(t, output, `"target_exists":true`)
	assert.Equal(t, 1, calls)
}
