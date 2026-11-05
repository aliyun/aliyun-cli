package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/stretchr/testify/require"
)

func TestCSDryRunJSONRedactsFormWithoutContentType(t *testing.T) {
	clearAgentDetectionEnv(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	profileJSON := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"test-access-key-id","access_key_secret":"test-access-key-secret","region_id":"cn-hangzhou","language":"en"}]}`)
	if err := os.WriteFile(configPath, profileJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	profile, err := config.LoadProfile(configPath, "default")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	root := newRootCommand(profile, &stdout)
	ctx := newCommandContext(&stdout, &stderr)
	ctx.EnterCommand(root)
	root.Execute(ctx, []string{
		"cs", "POST", "/clusters", "--body", "password=FAKE_SECRET_123", "--cli-dry-run-json",
		"--config-path", configPath,
	})
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", &stderr)
	}
	var output openapi.CliDryRunOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON output: %v; stdout: %s", err, &stdout)
	}
	if output.Method != "POST" || output.Pathname != "/clusters" || output.Body != "password=FAKE***" {
		t.Fatalf("unexpected dry-run output: %+v", output)
	}
	t.Logf("actual dry-run JSON: %s", &stdout)
}

func TestBtripDryRunJSONFormBody(t *testing.T) {
	clearAgentDetectionEnv(t)
	const example = "out_sub_order_id=1001&ext_params={key=example-string}&modify_pay_amount=5100&out_order_id=2001&sub_order_id=1001&isv_name=name&order_id=2001"
	for _, tc := range []struct{ name, body, want string }{
		{"original example", example, fmt.Sprintf("<opaque body omitted: %d bytes>", len(example))},
		{"valid nested JSON", `ext_params={"key":"example-string"}`, `ext_params={"key":"example-string"}`},
		{"sensitive nested JSON", `ext_params={"password":"example-secret"}`, `ext_params={"password":"exam***"}`},
		{"malformed nested JSON and sensitive field", `ext_params={password=example-secret}&password=example-secret`, fmt.Sprintf("<opaque body omitted: %d bytes>", len(`ext_params={password=example-secret}&password=example-secret`))},
		{"malformed form", `password=example-secret&ext_params=%zz`, fmt.Sprintf("<opaque body omitted: %d bytes>", len(`password=example-secret&ext_params=%zz`))},
		{"JSON form representation", `{"password":"example-secret"}`, `{"password":"exam***"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			profileJSON := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"testid","access_key_secret":"testsecret","region_id":"cn-hangzhou","language":"en"}]}`)
			if err := os.WriteFile(configPath, profileJSON, 0o600); err != nil {
				t.Fatal(err)
			}
			profile, err := config.LoadProfile(configPath, "default")
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			root := newRootCommand(profile, &stdout)
			ctx := newCommandContext(&stdout, &stderr)
			ctx.EnterCommand(root)
			root.Execute(ctx, []string{
				"btripopen", "PUT", "/dtb-flight/v2/modify/action/pay",
				"--region", "cn-hangzhou",
				"--header", "Content-Type=application/x-www-form-urlencoded",
				"--body", tc.body, "--version", "2022-05-20", "--force",
				"--mode", "AK", "--access-key-id", "testid", "--access-key-secret", "testsecret",
				"--endpoint", "example.com", "--config-path", configPath, "--cli-dry-run-json",
			})
			if stderr.Len() != 0 {
				t.Fatalf("unexpected stderr: %s", &stderr)
			}
			var output openapi.CliDryRunOutput
			if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
				t.Fatalf("invalid JSON output: %v; stdout: %s", err, &stdout)
			}
			if output.Body != tc.want || output.Method != "PUT" || output.Endpoint != "example.com" {
				t.Fatalf("unexpected dry-run output: %+v; want body: %s", output, tc.want)
			}
			if strings.Contains(stdout.String(), "example-secret") {
				t.Fatalf("secret leaked: %s", &stdout)
			}
		})
	}
}

func TestCSArgumentsLogDoesNotLeakNestedPassword(t *testing.T) {
	clearAgentDetectionEnv(t)
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGINS_DIR", t.TempDir())
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGIN_AUTO_INSTALL", "false")
	const secret = "FAKE_QA_ARGUMENT_PASSWORD_20261104"
	quoted, _ := json.Marshal(`{"password":"` + secret + `","name":"visible"}`)
	cacheCommand := exec.Command("go", "env", "-json", "GOCACHE", "GOMODCACHE", "GOPATH")
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GOROOT=") {
			cacheCommand.Env = append(cacheCommand.Env, entry)
		}
	}
	cacheOutput, err := cacheCommand.Output()
	if err != nil {
		t.Fatal(err)
	}
	var caches map[string]string
	if err := json.Unmarshal(cacheOutput, &caches); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"fake-id","access_key_secret":"fake-key-secret","region_id":"cn-hangzhou","language":"en"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOMEDRIVE", dir)
	t.Setenv("HOMEPATH", "")
	for _, tc := range []struct {
		name, value, marker string
	}{
		{"broken JSON", `{"password":"` + secret + `",`, "opaque body omitted"},
		{"quoted trailing JSON", string(quoted) + " trailing", "opaque body omitted"},
		{"complete quoted JSON", string(quoted), "FAKE***"},
		{"ordinary quoted text", `"quoted" ordinary text`, "ordinary text"},
	} {
		for _, ai := range []string{"0", "1"} {
			t.Run(tc.name+"/AI="+ai, func(t *testing.T) {
				var mu sync.Mutex
				var captured [][]byte
				var unrelated, uploads int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost && r.URL.Path == "/telemetry" {
						_, _ = io.Copy(io.Discard, r.Body)
						mu.Lock()
						uploads++
						mu.Unlock()
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"ok":true}`))
						return
					}
					if r.Method != http.MethodPost || r.Host != "capture.invalid" || r.URL.Path != "/clusters" {
						mu.Lock()
						unrelated++
						mu.Unlock()
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						http.Error(w, "request body read failed", http.StatusBadRequest)
						return
					}
					mu.Lock()
					captured = append(captured, body)
					mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"ok":true}`))
				}))
				t.Cleanup(server.Close)
				t.Setenv("HTTP_PROXY", server.URL)
				t.Setenv("HTTPS_PROXY", server.URL)
				t.Setenv("NO_PROXY", "")
				t.Setenv("ALIBABA_CLOUD_CLI_AI_MODE", ai)
				for _, extra := range []struct{ method, host, path string }{
					{http.MethodPost, "unrelated.invalid", "/clusters"},
					{http.MethodPost, "capture.invalid", "/unrelated"},
					{http.MethodGet, "capture.invalid", "/clusters"},
				} {
					req, err := http.NewRequest(extra.method, server.URL+extra.path, strings.NewReader("unrelated"))
					if err != nil {
						t.Fatal(err)
					}
					req.Host = extra.host
					response, err := server.Client().Do(req)
					if err != nil {
						t.Fatal(err)
					}
					_, _ = io.Copy(io.Discard, response.Body)
					response.Body.Close()
					if response.StatusCode != http.StatusServiceUnavailable {
						t.Fatalf("unrelated request status = %d", response.StatusCode)
					}
				}
				var stdout, stderr bytes.Buffer
				args := []string{"cs", "create-cluster", "--api-version", "2015-12-15", "--biz-region-id", "cn-hangzhou", "--name", tc.value, "--config-path", configPath, "--profile", "default", "--region", "cn-hangzhou", "--endpoint", "capture.invalid", "--insecure", "--log-level", "DEBUG"}
				cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				for _, entry := range os.Environ() {
					if !strings.HasPrefix(entry, "GOROOT=") {
						cmd.Env = append(cmd.Env, entry)
					}
				}
				home := t.TempDir()
				// Windows prefers HOMEDRIVE/HOMEPATH over USERPROFILE when locating the configuration directory.
				cmd.Env = append(cmd.Env, "HOME="+home, "USERPROFILE="+home, "HOMEDRIVE=", "HOMEPATH=",
					"ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE=0",
					"ALIBABA_CLOUD_CLI_TELEMETRY_LOG_DISABLE=0",
					"ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT="+server.URL+"/telemetry")
				for name, value := range caches {
					cmd.Env = append(cmd.Env, name+"="+value)
				}
				if err := cmd.Run(); err != nil {
					t.Fatalf("CLI failed: %v; stderr=%s", err, &stderr)
				}
				// Upload removes its cache before its final write, so wait for the terminal diagnostic before cleaning HOME.
				require.Eventually(t, func() bool {
					data, err := os.ReadFile(filepath.Join(home, ".aliyun", "logs", "telemetry.log"))
					if err != nil {
						return false
					}
					for _, line := range bytes.Split(data, []byte("\n")) {
						var record struct{ Stage, Status string }
						if json.Unmarshal(line, &record) == nil && record.Stage == "upload" && record.Status == "succeeded" {
							return true
						}
					}
					return false
				}, 10*time.Second, 10*time.Millisecond, "background telemetry upload did not complete")
				want, _ := json.Marshal(map[string]string{"name": tc.value, "region_id": "cn-hangzhou"})
				mu.Lock()
				bodies := append([][]byte(nil), captured...)
				rejected, completedUploads := unrelated, uploads
				mu.Unlock()
				if completedUploads != 1 {
					t.Fatalf("captured %d telemetry uploads, want exactly one", completedUploads)
				}
				if rejected < 3 {
					t.Fatalf("rejected %d unrelated requests, want at least three", rejected)
				}
				t.Logf("captured %d API requests; rejected %d unrelated requests; completed %d telemetry uploads", len(bodies), rejected, completedUploads)
				if len(bodies) != 1 {
					t.Fatalf("captured %d API requests, want exactly one", len(bodies))
				}
				if !bytes.Equal(bodies[0], want) {
					t.Fatalf("wire body changed: got %q, want %q", bodies[0], want)
				}
				if strings.Contains(stdout.String()+stderr.String(), secret) || !strings.Contains(stderr.String(), "Arguments:") || !strings.Contains(stderr.String(), tc.marker) {
					t.Fatalf("unsafe argument output: stdout=%s stderr=%s", &stdout, &stderr)
				}
			})
		}
	}
}

func TestCSDryRunOmitsBOMXMLBodyFile(t *testing.T) {
	clearAgentDetectionEnv(t)
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGINS_DIR", t.TempDir())
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGIN_AUTO_INSTALL", "false")
	const secret = "FAKE_SECRET_20261103"
	quoted, _ := json.Marshal(`{"password":"` + secret + `","name":"visible"}`)
	trailing := string(quoted) + " trailing"
	trailingBody, _ := json.Marshal(map[string]string{"payload": trailing, "name": "visible"})
	for _, tc := range []struct{ name, body, want string }{
		{"quoted trailing JSON", string(trailingBody), ""},
		{"quoted trailing form", "payload=" + url.QueryEscape(trailing) + "&name=visible", ""},
		{"ordinary quoted text", `{"message":"\"quoted\" ordinary text","name":"visible"}`, `{"message":"\"quoted\" ordinary text","name":"visible"}`},
		{"BOM XML", "\ufeff<Password>" + secret + "==</Password>", ""},
		{"zero-width XML", "\u200b<Password>" + secret + "==</Password>", ""},
		{"nested BOM JSON", `{"payload":"\ufeff{\"password\":\"` + secret + `\"}"}`, ""},
		{"nested BOM form", "payload=%EF%BB%BF%7B%22password%22%3A%22" + secret + "%22%7D", ""},
		{"sensitive JSON prefix", `{"password":"[` + secret + `","name":"visible"}`, `{"name":"visible","password":"[FAK***"}`},
		{"sensitive form prefix", "password=%5B" + secret + "&name=visible", "password=[FAK***&name=visible"},
		{"ordinary scalar", `{"message":"[Important] ordinary text","name":"visible"}`, `{"message":"[Important] ordinary text","name":"visible"}`},
		{"ordinary placeholder", `{"message":"{YourContent}","name":"visible"}`, `{"message":"{YourContent}","name":"visible"}`},
		{"ordinary template", `{"message":"${user}","name":"visible"}`, `{"message":"${user}","name":"visible"}`},
	} {
		body := []byte(tc.body)
		for _, route := range []struct {
			name string
			args []string
		}{
			{"camel", []string{"cs", "CreateCluster", "--version", "2015-12-15"}},
			{"REST", []string{"cs", "POST", "/clusters", "--version", "2015-12-15"}},
			{"kebab", []string{"cs", "create-cluster", "--api-version", "2015-12-15"}},
		} {
			for _, mode := range []string{"--cli-dry-run", "--cli-dry-run-json"} {
				t.Run(tc.name+"/"+route.name+"/"+mode, func(t *testing.T) {
					dir := t.TempDir()
					bodyPath := filepath.Join(dir, "body.xml")
					if err := os.WriteFile(bodyPath, body, 0o600); err != nil {
						t.Fatal(err)
					}
					configPath := filepath.Join(dir, "config.json")
					profileJSON := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"testid","access_key_secret":"testsecret","region_id":"cn-hangzhou","language":"en"}]}`)
					if err := os.WriteFile(configPath, profileJSON, 0o600); err != nil {
						t.Fatal(err)
					}
					profile, err := config.LoadProfile(configPath, "default")
					if err != nil {
						t.Fatal(err)
					}
					var stdout, stderr bytes.Buffer
					root := newRootCommand(profile, &stdout)
					ctx := newCommandContext(&stdout, &stderr)
					ctx.EnterCommand(root)
					args := append([]string(nil), route.args...)
					args = append(args, "--body-file", bodyPath, mode, "--endpoint", "example.com", "--region", "cn-hangzhou", "--config-path", configPath, "--log-level", "DEBUG")
					cli.DisableExitCode()
					defer cli.EnableExitCode()
					originalArgs := os.Args
					os.Args = append([]string{"aliyun"}, args...)
					defer func() { os.Args = originalArgs }()
					root.Execute(ctx, args)
					if stderr.Len() != 0 || strings.Contains(stdout.String(), secret) {
						t.Fatalf("unsafe dry-run output: stdout=%s stderr=%s", &stdout, &stderr)
					}
					want := tc.want
					if want == "" {
						want = fmt.Sprintf("<opaque body omitted: %d bytes>", len(body))
					}
					if mode == "--cli-dry-run-json" {
						var output openapi.CliDryRunOutput
						if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || output.Body != want || output.BodyFormat == "" {
							t.Fatalf("unexpected dry-run JSON: %+v, error=%v", output, err)
						}
					} else if !strings.Contains(stdout.String(), want) {
						t.Fatalf("omission marker missing: %s", &stdout)
					}
					unchanged, err := os.ReadFile(bodyPath)
					if err != nil || !bytes.Equal(unchanged, body) {
						t.Fatalf("input bytes changed: error=%v", err)
					}
				})
			}
		}
	}
}
