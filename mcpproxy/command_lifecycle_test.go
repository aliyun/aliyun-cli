package mcpproxy

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMCPCommandServerLifecycle(t *testing.T) {
	for _, mode := range []string{"listen failure", "interrupt", "reauthorization failure"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "interrupt" && runtime.GOOS == "windows" {
				t.Skip("Windows cannot send os.Interrupt to a process")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestMCPCommandServerLifecycleProcess$"}
			if dir := flag.Lookup("test.gocoverdir"); dir != nil && dir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+dir.Value.String())
			}
			cmd := exec.CommandContext(ctx, os.Args[0], args...)
			cmd.Env = append(os.Environ(), "MCP_LIFECYCLE_TEST="+mode)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			require.NoError(t, ctx.Err(), "%s", output)
			require.Contains(t, string(output), "PASS")
		})
	}
}

func TestMCPProxyStopBeforeStart(t *testing.T) {
	proxy := testProxy(t)
	require.NoError(t, proxy.Stop())
	require.NoError(t, proxy.Stop())
	require.NoError(t, proxy.Start())
	require.Nil(t, proxy.Server)
}

func TestMCPCommandServerLifecycleProcess(t *testing.T) {
	mode := os.Getenv("MCP_LIFECYCLE_TEST")
	if mode == "" {
		return
	}
	ctx, _ := testCLIProfile(t)
	installAPIHandler(t, func(*http.Request) (*http.Response, error) {
		return sdkJSONResponse(`{"apiMcpServers":[{"id":"test","name":"test","urls":{"mcp":"https://upstream.invalid/mcp/test"}}]}`), nil
	})
	opts := ProxyConfig{Host: "127.0.0.1", Port: -1, RegionType: RegionCN, McpProfile: &McpProfile{
		MCPOAuthAccessToken: "access", MCPOAuthAccessTokenExpire: time.Now().Add(time.Hour).Unix(),
		MCPOAuthRefreshToken: "refresh", MCPOAuthRefreshTokenExpire: time.Now().Add(24 * time.Hour).Unix(),
	}}
	if mode == "reauthorization failure" {
		opts.Port = 0
		opts.McpProfile.MCPOAuthRefreshTokenExpire = time.Now().Add(-time.Hour).Unix()
		previous := isInteractiveInput
		isInteractiveInput = func() bool { return false }
		t.Cleanup(func() { isInteractiveInput = previous })
	}
	if mode == "interrupt" {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		opts.Port = listener.Addr().(*net.TCPAddr).Port
		require.NoError(t, listener.Close())
		go func() {
			client := &http.Client{Timeout: time.Second}
			for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
				response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", opts.Port))
				if err == nil {
					response.Body.Close()
					process, err := os.FindProcess(os.Getpid())
					if err == nil {
						err = process.Signal(os.Interrupt)
					}
					if err != nil {
						t.Error(err)
					}
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
	err := startMCPProxy(ctx, opts)
	if mode == "listen failure" {
		require.ErrorContains(t, err, "proxy server failed")
	} else if mode == "reauthorization failure" {
		require.ErrorContains(t, err, "re-authorization failed")
		require.Contains(t, ctx.Stdout().(*bytes.Buffer).String(), "Shutting down gracefully")
	} else {
		require.NoError(t, err)
		require.Contains(t, ctx.Stdout().(*bytes.Buffer).String(), "MCP Proxy stopped successfully")
	}
}
