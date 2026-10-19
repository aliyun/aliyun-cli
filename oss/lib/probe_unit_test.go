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

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func localOSSCommand(t *testing.T, endpoint string) Command {
	t.Helper()
	_, opts, err := parseOSSOptions([]string{"--endpoint=" + endpoint, "--access-key-id=id", "--access-key-secret=secret", "--disable-crc64", "--force-path-style", "--retry-times=1"}, "")
	require.NoError(t, err)
	return Command{options: opts, configOptions: OptionMapType{}}
}

func TestProbeReaderAndBandwidthStatistics(t *testing.T) {
	reader := TestAppendReader{RandText: []byte("abc")}
	data := make([]byte, 8)
	n, err := reader.Read(data)
	require.NoError(t, err)
	require.Equal(t, 8, n)
	require.Equal(t, "abcabcab", string(data))
	reader.Close()
	n, err = reader.Read(data)
	require.ErrorContains(t, err, "closed")
	require.Zero(t, n)
	var stat StatBandWidth
	stat.Reset(3)
	stat.AddBytes(4)
	stat.SetMaxSpeed(2.5)
	for _, event := range []oss.ProgressEvent{{EventType: oss.TransferStartedEvent, RwBytes: 10}, {EventType: oss.TransferDataEvent, RwBytes: 5}, {EventType: oss.TransferCompletedEvent, RwBytes: 6}} {
		stat.ProgressChanged(&event)
	}
	snap := stat.GetStat()
	require.Equal(t, int64(15), snap.TotalBytes)
	require.Equal(t, 3, snap.Parallel)
	require.Equal(t, 2.5, snap.MaxSpeed)
	require.Positive(t, snap.StartTick)
	require.Equal(t, int64(4), getPartEnd(0, 10, 5))
	require.Equal(t, int64(9), getPartEnd(8, 10, 5))
}

func TestProbeLocalPathsAndValidation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0600))
	pc := ProbeCommand{}
	require.ErrorContains(t, pc.CheckCycleSymlinkWithDeepTravel(), "parameter")
	pc.command.args = []string{filepath.Join(dir, "missing")}
	require.Error(t, pc.CheckCycleSymlinkWithDeepTravel())
	pc.command.args = []string{file}
	require.NoError(t, pc.CheckCycleSymlinkWithDeepTravel())
	require.NoError(t, os.Mkdir(filepath.Join(dir, "child"), 0700))
	pc.command.args = []string{dir}
	require.NoError(t, pc.CheckCycleSymlinkWithDeepTravel())
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken")); err == nil {
		require.Error(t, pc.CheckCycleSymlinkWithDeepTravel())
	}
	for _, target := range []string{"", dir, filepath.Join(dir, "new") + string(os.PathSeparator), filepath.Join(dir, "child", "copy")} {
		got, err := prepareLocalFileName("folder/source", target)
		require.NoError(t, err)
		if target == "" {
			cwd, err := os.Getwd()
			require.NoError(t, err)
			require.Equal(t, filepath.Join(cwd, "source"), got)
		} else if target == dir || strings.HasSuffix(target, string(os.PathSeparator)) {
			require.Equal(t, filepath.Join(target, "source"), got)
		} else {
			require.Equal(t, target, got)
		}
	}
	_, err := prepareLocalFileName("source", file+string(os.PathSeparator)+"target")
	require.Error(t, err)
	for _, input := range []string{"invalid", "http:///file", "http://host/"} {
		_, _, err := urlCheck(input)
		require.Error(t, err)
	}
	host, name, err := urlCheck("https://host/folder/file?query=1")
	require.NoError(t, err)
	require.Equal(t, "host", host)
	require.Equal(t, "file", name)
	pc.command.args = nil
	name, err = pc.getFileNameArg()
	require.NoError(t, err)
	require.Empty(t, name)
	pc.command.args = []string{"oss://bucket/key"}
	_, err = pc.getFileNameArg()
	require.ErrorContains(t, err, "not a local file")
	pc.command.args = []string{file}
	name, err = pc.getFileNameArg()
	require.NoError(t, err)
	require.Equal(t, file, name)
	pc.command = localOSSCommand(t, "http://127.0.0.1")
	require.ErrorContains(t, pc.probeDownload(), "bucketName")
	require.ErrorContains(t, pc.probeUpload(), "bucketName")
	pc.pbOption.upMode = "invalid"
	require.ErrorContains(t, pc.probeUpload(), "invalid mode")
	pc.pbOption.upMode = normalMode
	pc.pbOption.bucketName = "bucket"
	pc.command.options = OptionMapType{}
	require.ErrorContains(t, pc.probeUpload(), "endpoint")
	require.ErrorContains(t, pc.probeDownload(), "endpoint")
	pc.pbOption.fromUrl = "invalid"
	require.ErrorContains(t, pc.probeDownload(), "url")
	pc.pbOption.fromUrl = ""
	pc.pbOption.bucketName = ""
	require.ErrorContains(t, pc.DetectBandWidth(), "bucketname")
	require.ErrorContains(t, pc.DetectDownloadTime(), "bucketname")
	pc.command = Command{}
	pc.pbOption.bucketName = "bucket"
	require.Error(t, pc.DetectBandWidth())
	require.Error(t, pc.DetectDownloadTime())
	pc.command = localOSSCommand(t, "http://127.0.0.1")
	pc.pbOption.probeItem = "download-speed"
	require.ErrorContains(t, pc.DetectBandWidth(), "object is empty")
	require.ErrorContains(t, pc.DetectDownloadTime(), "object is empty")
	for _, item := range []string{"cycle-symlink", "unsupported", "download-speed", "download-time"} {
		cmd := probeCommand.command
		cmd.args = nil
		cmd.options = OptionMapType{OptionProbeItem: &item}
		pc := ProbeCommand{command: cmd}
		require.Error(t, pc.RunCommand())
	}
}

func TestProbeHTTPDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, "download content")
	}))
	defer server.Close()
	dir := t.TempDir()
	log, err := os.Create(filepath.Join(dir, "probe.log"))
	require.NoError(t, err)
	defer log.Close()
	pc := ProbeCommand{}
	pc.pbOption.logFile = log
	pc.pbOption.fromUrl = server.URL + "/content"
	pc.command.args = []string{filepath.Join(dir, "output")}
	require.NoError(t, pc.downloadWithHttpUrl())
	data, err := os.ReadFile(pc.pbOption.dlFilePath)
	require.NoError(t, err)
	require.Equal(t, "download content", string(data))
	require.Equal(t, int64(len(data)), pc.pbOption.dlFileSize)
	pc.command.args = []string{filepath.Join(dir, "not-created")}
	pc.pbOption.fromUrl = server.URL + "/missing"
	require.ErrorContains(t, pc.downloadWithHttpUrl(), "404")
	pc.pbOption.fromUrl = "invalid"
	require.ErrorContains(t, pc.downloadWithHttpUrl(), "urlCheck")
	pc.pbOption.fromUrl = server.URL + "/content"
	pc.command.args = []string{"oss://bucket/key"}
	require.ErrorContains(t, pc.downloadWithHttpUrl(), "getFileNameArg")
}

func TestProbeObjectTransfersAgainstLocalServer(t *testing.T) {
	var writes, deletes, reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodGet:
			reads.Add(1)
			io.WriteString(w, "remote content")
		case http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPut, http.MethodPost:
			writes.Add(1)
			data, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NotEmpty(t, data)
			w.Header().Set("X-Oss-Next-Append-Position", fmt.Sprint(len(data)))
		}
	}))
	defer server.Close()
	pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
	pc.pbOption.bucketName = "bucket"
	dir := t.TempDir()
	file := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0600))
	require.NoError(t, pc.probeUploadFileNormal(file, "normal"))
	require.NoError(t, pc.probeUploadFileAppend(file, "append"))
	require.EqualValues(t, 2, writes.Load())
	for _, transfer := range []func(string, string) error{pc.probeUploadFileNormal, pc.probeUploadFileAppend, pc.probeUploadFileMultiPart} {
		require.Error(t, transfer(filepath.Join(dir, "missing"), "key"))
	}
	require.NoError(t, pc.prepareRandomObject("random"))
	require.EqualValues(t, 3, writes.Load())
	pc.command.args = []string{filepath.Join(dir, "download")}
	require.NoError(t, pc.probeDownloadObject(CloudURL{bucket: "bucket", object: "remote"}, true))
	data, err := os.ReadFile(pc.pbOption.dlFilePath)
	require.NoError(t, err)
	require.Equal(t, "remote content", string(data))
	require.EqualValues(t, 1, deletes.Load())
	require.EqualValues(t, 1, reads.Load())
	pc.pbOption.objectName = "remote"
	pc.command.args = []string{filepath.Join(dir, "download2")}
	require.NoError(t, pc.probeDownloadWithParameter())
	require.EqualValues(t, 2, reads.Load())
	pc.command = localOSSCommand(t, "://invalid")
	require.Error(t, pc.prepareRandomObject("key"))
	require.Error(t, pc.deleteObject("key"))
	for _, transfer := range []func(string, string) error{pc.probeUploadFileNormal, pc.probeUploadFileAppend, pc.probeUploadFileMultiPart} {
		require.Error(t, transfer(file, "key"))
	}
}

func TestProbeRangeWorkerAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "bytes=1-3", r.Header.Get("Range"))
		w.Header().Set("Content-Range", "bytes 1-3/5")
		w.WriteHeader(http.StatusPartialContent)
		io.WriteString(w, "ell")
	}))
	defer server.Close()
	cmd := localOSSCommand(t, server.URL)
	bucket, err := cmd.ossBucket("bucket")
	require.NoError(t, err)
	parts := []downloadPart{{Index: 0, Start: 1, End: 3}}
	jobs := make(chan downloadPart, 1)
	downloadScheduler(jobs, parts)
	results, failed, die := make(chan downloadPart, 1), make(chan error, 1), make(chan bool)
	var stat StatBandWidth
	downloadWorker(0, downloadWorkerArg{bucket, "key"}, jobs, results, failed, die, &stat)
	require.Equal(t, parts[0], <-results)
	require.Empty(t, failed)
	require.Equal(t, int64(3), stat.GetStat().TotalBytes)
	jobs = make(chan downloadPart, 1)
	downloadScheduler(jobs, parts)
	close(die)
	downloadWorker(0, downloadWorkerArg{bucket, "key"}, jobs, results, failed, die, &stat)
	require.Empty(t, results)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pc := ProbeCommand{}
	pc.PutObjectWithContext(bucket, &stat, strings.NewReader("content"), ctx)
	pc.GetObjectWithContext(bucket, "key", &stat, ctx)
}

func TestProbeUploadAndDownloadReportsLocally(t *testing.T) {
	server, _ := newDataServer(t)
	defer server.Close()
	var uploadCount atomic.Int32
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(404)
			return
		}
		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			uploadCount.Add(1)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			w.Header().Set("X-Oss-Next-Append-Position", fmt.Sprint(len(body)))
			w.Header().Set("ETag", "local-etag")
			if r.URL.Query().Has("uploads") {
				io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>target</Key><UploadId>upload</UploadId></InitiateMultipartUploadResult>`)
			} else if r.Method == http.MethodPost {
				io.WriteString(w, `<CompleteMultipartUploadResult><ETag>local-etag</ETag></CompleteMultipartUploadResult>`)
			}
		}
	}))
	defer uploadServer.Close()
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(source, []byte("content"), 0600))
	log, err := os.Create(filepath.Join(dir, "probe.log"))
	require.NoError(t, err)
	defer log.Close()
	for _, mode := range []string{normalMode, appendMode, multipartMode} {
		pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
		pc.command.args = []string{source}
		pc.pbOption.logFile, pc.pbOption.logName = log, log.Name()
		pc.pbOption.disableNetDetect = true
		pc.pbOption.upMode = mode
		pc.pbOption.bucketName, pc.pbOption.objectName = "bucket", "target"
		// A rejected overwrite must leave the input file intact.
		old := activeMachine
		t.Cleanup(func() { activeMachine = old })
		activeMachine = &machineInvocation{nonInteractive: true}
		captureBucketOutput(t, func() { require.NoError(t, pc.probeUpload()) })
		activeMachine = old
		data, err := os.ReadFile(source)
		require.NoError(t, err)
		require.Equal(t, "content", string(data))
		pc.command = localOSSCommand(t, uploadServer.URL)
		pc.command.args = []string{source}
		captureBucketOutput(t, func() { require.NoError(t, pc.probeUpload()) })
	}
	require.GreaterOrEqual(t, uploadCount.Load(), int32(3))
	for _, fromHTTP := range []bool{true, false} {
		pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
		pc.command.args = []string{filepath.Join(dir, fmt.Sprint("download-", fromHTTP))}
		pc.pbOption.logFile, pc.pbOption.logName = log, log.Name()
		pc.pbOption.disableNetDetect = true
		pc.pbOption.bucketName, pc.pbOption.objectName = "bucket", "target"
		if fromHTTP {
			pc.pbOption.fromUrl = server.URL + "/bucket/target"
		}
		captureBucketOutput(t, func() { require.NoError(t, pc.probeDownload()) })
		require.Equal(t, int64(7), pc.pbOption.dlFileSize)
		data, err := os.ReadFile(pc.pbOption.dlFilePath)
		require.NoError(t, err)
		require.Equal(t, "content", string(data))
	}
	data, err := os.ReadFile(log.Name())
	require.NoError(t, err)
	require.Contains(t, string(data), "download file:success")
	pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
	pc.pbOption.logFile, pc.pbOption.logName = log, log.Name()
	pc.pbOption.disableNetDetect = true
	pc.pbOption.bucketName, pc.pbOption.objectName = "bucket", "target"
	pc.command.args = []string{"oss://bucket/key"}
	require.ErrorContains(t, pc.probeUpload(), "getFileNameArg")
	pc.command.args = []string{filepath.Join(dir, "missing")}
	require.ErrorContains(t, pc.probeUpload(), "not exist")
	pc.command.args = []string{dir}
	require.ErrorContains(t, pc.probeUpload(), "not file")
	pc.pbOption.fromUrl = server.URL + "/bucket/target"
	pc.command.args = []string{"oss://bucket/key"}
	captureBucketOutput(t, func() { require.Error(t, pc.probeDownload()) })
	t.Setenv("PATH", t.TempDir())
	pc.pbOption.disableNetDetect = false
	pc.pbOption.netAddr = "127.0.0.1"
	pc.ossNetDetection("127.0.0.1")
	data, err = os.ReadFile(log.Name())
	require.NoError(t, err)
	require.Contains(t, string(data), "Command => ping")
}
