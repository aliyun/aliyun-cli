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

func TestSyncReportTracksCompletedDeletePhase(t *testing.T) {
	clearEndpointTestEnv(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "empty")
	require.NoError(t, os.Mkdir(src, 0700))
	var deletes atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			io.WriteString(w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>obsolete</Key><Size>1</Size><LastModified>2026-01-01T00:00:00.000Z</LastModified></Contents></ListBucketResult>`)
		case "POST":
			deletes.Add(1)
			io.WriteString(w, `<DeleteResult/>`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	report := filepath.Join(dir, "failures.jsonl")
	_, err := phaseCInvoke(t, syncCommand.command, server.URL, src+string(os.PathSeparator), "oss://bucket/", "--delete", "-f", "--cli-failure-report", report, "--output-dir", filepath.Join(dir, "legacy"))
	require.NoError(t, err)
	require.EqualValues(t, 1, deletes.Load())
	data, err := os.ReadFile(report)
	require.NoError(t, err)
	var item failureItem
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(string(data))), &item))
	require.Equal(t, "succeeded", item.Outcome)
	require.Equal(t, "completed", item.DeletePhase)
}
