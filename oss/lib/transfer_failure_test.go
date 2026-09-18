package lib

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecursiveTransfersReportFailuresAndJoinWorkers(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, operation := range []string{"download", "copy", "upload"} {
		for _, keepGoing := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: " stop", true: " continue"}[keepGoing], func(t *testing.T) {
				dir := t.TempDir()
				var attempts atomic.Int64
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" && r.URL.Path == "/bucket/" {
						io.WriteString(w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>a</Key><Size>1</Size><LastModified>2026-01-01T00:00:00.000Z</LastModified></Contents><Contents><Key>b</Key><Size>1</Size><LastModified>2026-01-01T00:00:00.000Z</LastModified></Contents></ListBucketResult>`)
						return
					}
					if r.Method == "HEAD" {
						w.Header().Set("Content-Length", "1")
						w.Header().Set("Last-Modified", "Thu, 01 Jan 2026 00:00:00 GMT")
						return
					}
					attempts.Add(1)
					w.WriteHeader(400)
					io.WriteString(w, `<Error><Code>InvalidDigest</Code><Message>transfer denied</Message></Error>`)
				}))
				defer server.Close()
				src, dst := "oss://bucket/", filepath.Join(dir, "download")+string(os.PathSeparator)
				if operation == "copy" {
					dst = "oss://target/"
				}
				if operation == "upload" {
					src = filepath.Join(dir, "source")
					require.NoError(t, os.Mkdir(src, 0700))
					for _, name := range []string{"a", "b"} {
						require.NoError(t, os.WriteFile(filepath.Join(src, name), []byte("x"), 0600))
					}
					dst = "oss://bucket/"
				}
				report := filepath.Join(dir, "failures.jsonl")
				args := []string{src, dst, "-r", "-f", "--jobs=1", "--cli-failure-report", report, "--output-dir", filepath.Join(dir, "legacy")}
				if !keepGoing {
					args = append(args, "--disable-ignore-error")
				}
				_, err := phaseCInvoke(t, copyCommand.command, server.URL, args...)
				require.Error(t, err)
				data, err := os.ReadFile(report)
				require.NoError(t, err)
				var items []failureItem
				for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
					var item failureItem
					require.NoError(t, json.Unmarshal([]byte(line), &item))
					items = append(items, item)
				}
				require.Equal(t, "summary", items[len(items)-1].Type)
				require.Equal(t, "failed", items[len(items)-1].Outcome)
				require.Positive(t, attempts.Load())
				if keepGoing {
					require.EqualValues(t, 2, attempts.Load())
				}
				require.Nil(t, activeMachine)
				require.Nil(t, bridgeCredentialProvider)
			})
		}
	}
}
