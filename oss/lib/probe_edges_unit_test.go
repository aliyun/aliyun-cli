package lib

import (
	"context"
	"fmt"
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

func TestProbeTemporaryTransferResourcesAndLog(t *testing.T) {
	t.Chdir(t.TempDir())
	var uploads, deletes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(404)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), "testossprobe")
			uploads.Add(1)
		case http.MethodGet:
			io.WriteString(w, "content")
		case http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(204)
		}
	}))
	defer server.Close()
	log, logName, err := logFileMake()
	require.NoError(t, err)
	defer log.Close()
	pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
	pc.pbOption.logFile, pc.pbOption.logName = log, logName
	pc.pbOption.bucketName, pc.pbOption.disableNetDetect = "bucket", true
	captureBucketOutput(t, func() { require.NoError(t, pc.probeUpload()) })
	require.EqualValues(t, 1, uploads.Load())
	require.EqualValues(t, 1, deletes.Load())
	files, err := os.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, filepath.Base(logName), files[0].Name())
	pc.command.args = []string{filepath.Join(t.TempDir(), "download")}
	require.NoError(t, pc.probeDownloadWithParameter())
	require.EqualValues(t, 2, uploads.Load())
	require.EqualValues(t, 2, deletes.Load())
	data, err := os.ReadFile(pc.pbOption.dlFilePath)
	require.NoError(t, err)
	require.Equal(t, "content", string(data))
	data, err = os.ReadFile(logName)
	require.NoError(t, err)
	require.Contains(t, string(data), "upload file:success")
}

func TestProbeCommandInitializationAndExecutionErrors(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, tc := range []struct {
		upload, download bool
		fromURL, want    string
	}{
		{true, true, "", "both true"}, {false, false, "", "both false"}, {false, true, "invalid", "invalid url"}, {true, false, "", "bucketName"},
	} {
		base := localOSSCommand(t, "http://127.0.0.1")
		base.options[OptionUpload], base.options[OptionDownload], base.options[OptionUrl] = &tc.upload, &tc.download, &tc.fromURL
		pc := ProbeCommand{command: base}
		require.ErrorContains(t, pc.RunCommand(), tc.want)
		require.FileExists(t, pc.pbOption.logName)
	}
	for _, bypass := range []string{"cycle-symlink", "download", "none"} {
		_, opts, err := parseOSSOptions(nil)
		require.NoError(t, err)
		opts[OptionConfigFile] = stringPointer(filepath.Join(t.TempDir(), "missing"))
		if bypass == "cycle-symlink" {
			opts[OptionProbeItem] = &bypass
		}
		if bypass == "download" {
			opts[OptionDownload] = boolPointer(true)
			opts[OptionUrl] = stringPointer("http://127.0.0.1/file")
		}
		pc := ProbeCommand{command: probeCommand.command}
		err = pc.Init(nil, opts)
		if bypass == "none" {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
	base := localOSSCommand(t, "http://127.0.0.1")
	base.options[OptionProbeItem] = stringPointer("cycle-symlink")
	base.args = []string{t.TempDir()}
	pc := ProbeCommand{command: base}
	require.NoError(t, pc.RunCommand())
}

func TestProbeContextLoopsStopAfterSuccessfulTransfers(t *testing.T) {
	for _, upload := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		var requests atomic.Int32
		var transferred atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requests.Add(1) > 1 {
				cancel()
				w.WriteHeader(403)
				return
			}
			if upload {
				n, err := io.Copy(io.Discard, r.Body)
				require.NoError(t, err)
				transferred.Add(n)
			} else {
				n, err := io.WriteString(w, "content")
				require.NoError(t, err)
				transferred.Add(int64(n))
			}
		}))
		pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
		bucket, err := pc.command.ossBucket("bucket")
		require.NoError(t, err)
		var stat StatBandWidth
		if upload {
			pc.PutObjectWithContext(bucket, &stat, strings.NewReader("content"), ctx)
		} else {
			pc.GetObjectWithContext(bucket, "key", &stat, ctx)
		}
		cancel()
		server.Close()
		require.GreaterOrEqual(t, requests.Load(), int32(2))
		require.Equal(t, int64(7), transferred.Load())
	}
}

func TestProbeDiagnosticErrorsBeforeBackgroundWork(t *testing.T) {
	for _, status := range []int{403, 404, 200} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if status == 200 {
				w.Header().Set("Content-Length", "7")
			}
			w.WriteHeader(status)
		}))
		pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
		pc.command.options[OptionParallel] = stringPointer(fmt.Sprint(MaxParallel + 1))
		pc.pbOption.bucketName, pc.pbOption.objectName, pc.pbOption.probeItem = "bucket", "key", "download-speed"
		if status != 200 {
			require.Error(t, pc.DetectBandWidth())
		}
		require.Error(t, pc.DetectDownloadTime())
		require.Error(t, pc.prepareRandomObject("exists"))
		server.Close()
	}
	for _, truncated := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if truncated {
				w.Header().Set("Content-Length", "10")
				io.WriteString(w, "short")
			} else {
				w.WriteHeader(403)
			}
		}))
		cmd := localOSSCommand(t, server.URL)
		bucket, err := cmd.ossBucket("bucket")
		require.NoError(t, err)
		jobs, results, failures := make(chan downloadPart, 1), make(chan downloadPart, 1), make(chan error, 1)
		jobs <- downloadPart{Start: 0, End: 9}
		close(jobs)
		var stat StatBandWidth
		downloadWorker(0, downloadWorkerArg{bucket: bucket, key: "key"}, jobs, results, failures, make(chan bool), &stat)
		require.Empty(t, results)
		if truncated {
			require.Error(t, <-failures)
		}
		server.Close()
	}
}

func TestProbeHTTPOverwriteAndFileCreationErrors(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); io.WriteString(w, "content") }))
	defer server.Close()
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("preserved"), 0600))
	log, err := os.Create(filepath.Join(dir, "probe.log"))
	require.NoError(t, err)
	defer log.Close()
	pc := ProbeCommand{command: Command{args: []string{file}}}
	pc.pbOption.fromUrl, pc.pbOption.logFile = server.URL+"/content", log
	withBucketInput(t, "n\n", func() { require.NoError(t, pc.downloadWithHttpUrl()) })
	require.Zero(t, requests.Load())
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, "preserved", string(data))
	withBucketInput(t, "y\n", func() { require.NoError(t, pc.downloadWithHttpUrl()) })
	require.EqualValues(t, 1, requests.Load())
	data, err = os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, "content", string(data))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "content"), 0700))
	pc.command.args = []string{dir}
	withBucketInput(t, "y\n", func() { require.Error(t, pc.downloadWithHttpUrl()) })
	pc.command.args = []string{filepath.Join(dir, "missing")}
	pc.pbOption.fromUrl = "other://host/content"
	require.Error(t, pc.downloadWithHttpUrl())
	pc.command.args = []string{filepath.Join(file, "child")}
	pc.pbOption.fromUrl = server.URL + "/content"
	require.Error(t, pc.downloadWithHttpUrl())
}
