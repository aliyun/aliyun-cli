package openapi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

// Exercise the real installer: mocking Install would hide its progress output.
func TestCommandPluginInstallOutput(t *testing.T) {
	for _, mode := range []string{"auto", "interactive", "explicit"} {
		for _, failDownload := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/failure=%v", mode, failDownload), func(t *testing.T) {
				cleanup := setTestHomeDir(t, t.TempDir())
				defer cleanup()
				t.Setenv(plugin.EnvPluginsDir, t.TempDir())
				const name = "aliyun-cli-outputfixture"
				var archive bytes.Buffer
				gz := gzip.NewWriter(&archive)
				tw := tar.NewWriter(gz)
				manifest := `{"name":"aliyun-cli-outputfixture","version":"1.0.0","type":"go","command":"outputfixture","bin":{"path":"fixture"}}`
				for path, body := range map[string]string{"manifest.json": manifest, "fixture": "fixture binary"} {
					require.NoError(t, tw.WriteHeader(&tar.Header{Name: path, Size: int64(len(body)), Mode: 0755}))
					_, err := tw.Write([]byte(body))
					require.NoError(t, err)
				}
				require.NoError(t, tw.Close())
				require.NoError(t, gz.Close())
				var baseURL string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch {
					case strings.HasSuffix(r.URL.Path, "plugin_pkg_index.json"):
						fmt.Fprintf(w, `{"plugins":[{"name":%q,"command":"outputfixture","versions":{"1.0.0":{%q:{"url":%q,"checksum":"%x"}}}}]}`, name, plugin.GetCurrentPlatform(), baseURL+"/fixture.tgz", sha256.Sum256(archive.Bytes()))
					case r.URL.Path == "/fixture.tgz":
						if failDownload {
							w.WriteHeader(http.StatusServiceUnavailable)
							return
						}
						_, _ = w.Write(archive.Bytes())
					default:
						t.Errorf("unexpected request: %s", r.URL)
						http.NotFound(w, r)
					}
				}))
				defer server.Close()
				baseURL = server.URL
				mgr, err := plugin.NewManager()
				require.NoError(t, err)
				require.NoError(t, mgr.ApplySourceBaseOverride(baseURL))
				var stdout, stderr bytes.Buffer
				ctx := cli.NewCommandContext(&stdout, &stderr)
				command := NewCommando(&stdout, config.Profile{Language: "en"})
				switch mode {
				case "auto":
					_, err = command.autoInstallPlugin(ctx, mgr, name, "outputfixture", false)
				case "interactive":
					originalStdin := stdin
					stdin = strings.NewReader("yes\n")
					defer func() { stdin = originalStdin }()
					_, err = command.interactiveInstallPlugin(ctx, mgr, name, "outputfixture", false)
				case "explicit":
					err = mgr.Install(ctx, name, "", false)
				}
				if failDownload {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				if mode == "explicit" {
					require.Contains(t, stdout.String(), "Downloading")
					if !failDownload {
						require.Contains(t, stdout.String(), "installed successfully")
					}
					return
				}
				require.Empty(t, stdout.String(), "installation must not contaminate command results")
				require.Contains(t, stderr.String(), "Downloading")
				if !failDownload {
					require.Contains(t, stderr.String(), "installed successfully")
					// The caller's output writer must remain usable for the command result.
					require.NoError(t, json.NewEncoder(ctx.Stdout()).Encode(map[string]bool{"ok": true}))
					require.JSONEq(t, `{"ok":true}`, stdout.String())
				}
			})
		}
	}
}
