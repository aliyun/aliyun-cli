package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestCopyValidatesFiltersAndTransferOptions(t *testing.T) {
	previousArgs := os.Args
	t.Cleanup(func() { os.Args = previousArgs })
	for _, tc := range []struct {
		name, filter, want string
		args               []string
		options            OptionMapType
	}{
		{"directory filter", "folder/*", "dir info", []string{"local", "oss://bucket/key"}, nil},
		{"nonrecursive filter", "*.txt", "only work with", []string{"local", "oss://bucket/key"}, nil},
		{"local copy", "", "copy files between local file system is not allowed", []string{"local", "other"}, nil},
		{"upload range", "", "range", []string{"local", "oss://bucket/key"}, OptionMapType{OptionRange: stringPointer("0-2")}},
		{"upload version", "", "version", []string{"local", "oss://bucket/key"}, OptionMapType{OptionVersionId: stringPointer("v1")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = []string{"ossutil", "cp"}
			if tc.filter != "" {
				os.Args = append(os.Args, "--include", tc.filter)
			}
			cc := CopyCommand{command: dataCommand(t, "http://127.0.0.1", copyCommand.command, tc.args, tc.options)}
			require.ErrorContains(t, cc.RunCommand(), tc.want)
		})
	}
	for _, reportBlocked := range []bool{true, false} {
		t.Run(map[bool]string{true: "report directory", false: "download partition"}[reportBlocked], func(t *testing.T) {
			os.Args = []string{"ossutil", "cp"}
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(http.StatusForbidden)
			}))
			defer server.Close()
			options := OptionMapType{OptionPartitionDownload: stringPointer("2:3"), OptionVersionId: stringPointer("v1")}
			if reportBlocked {
				blocked := filepath.Join(t.TempDir(), "file")
				require.NoError(t, os.WriteFile(blocked, nil, 0600))
				options = OptionMapType{OptionRecursion: boolPointer(true), OptionOutputDir: &blocked}
			}
			cc := CopyCommand{command: dataCommand(t, server.URL, copyCommand.command, []string{"oss://bucket/key", filepath.Join(t.TempDir(), "target")}, options)}
			defer func() { cc.cpOption.reporter.Clear() }()
			err := cc.RunCommand()
			require.Error(t, err)
			if reportBlocked {
				require.Zero(t, calls.Load())
			} else {
				require.ErrorContains(t, err, "403")
				require.Equal(t, 2, cc.cpOption.partitionIndex)
				require.Equal(t, 3, cc.cpOption.partitionCount)
				require.Equal(t, "v1", cc.cpOption.versionId)
			}
		})
	}
}

func TestCopyUploadJoinsWorkersAfterDirectoryListingFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	previousSignal, previousCount := chProgressSignal, atomic.LoadInt64(&signalNum)
	t.Cleanup(func() { chProgressSignal = previousSignal; atomic.StoreInt64(&signalNum, previousCount) })
	for _, keepGoing := range []bool{false, true} {
		chProgressSignal = make(chan chProgressSignalType, 100)
		cc := CopyCommand{command: localOSSCommand(t, "http://127.0.0.1")}
		cc.monitor.init(operationTypePut)
		cc.cpOption.recursive, cc.cpOption.enableSymlinkDir, cc.cpOption.ctnu = true, true, keepGoing
		cc.cpOption.routines, cc.cpOption.cpDir = 1, filepath.Join(t.TempDir(), "checkpoint")
		err := cc.uploadFiles([]StorageURLer{FileURL{urlStr: dir}}, CloudURL{bucket: "bucket", object: "prefix"})
		require.ErrorIs(t, err, os.ErrNotExist)
		require.True(t, cc.stopWorkers.Load())
		require.ErrorIs(t, cc.monitor.seekAheadError, os.ErrNotExist)
	}
}

func TestCopyStoppedConsumersDrainWithoutRequests(t *testing.T) {
	cc := CopyCommand{command: localOSSCommand(t, "http://127.0.0.1")}
	cc.stopWorkers.Store(true)
	files, objects := make(chan fileInfoType, 1), make(chan objectInfoType, 2)
	files <- fileInfoType{filePath: "unused"}
	close(files)
	objects <- objectInfoType{relativeKey: "first"}
	objects <- objectInfoType{relativeKey: "second"}
	close(objects)
	done := make(chan error, 3)
	cc.uploadConsumer(nil, CloudURL{}, files, done)
	cc.downloadConsumer(nil, "", objects, done)
	require.Empty(t, objects)
	objects = make(chan objectInfoType, 1)
	objects <- objectInfoType{relativeKey: "copy"}
	close(objects)
	cc.copyConsumer(nil, CloudURL{}, CloudURL{}, objects, done)
	for range 3 {
		require.NoError(t, <-done)
	}
	require.Empty(t, files)
	require.Empty(t, objects)
}

func TestCopyRangeDownloadPreservesVersionAndWritesRequestedBytes(t *testing.T) {
	previousSignal, previousCount, previousLevel := chProgressSignal, atomic.LoadInt64(&signalNum), logLevel
	t.Cleanup(func() {
		chProgressSignal = previousSignal
		atomic.StoreInt64(&signalNum, previousCount)
		logLevel = previousLevel
	})
	chProgressSignal, logLevel = make(chan chProgressSignalType, 100), oss.Info
	var heads, gets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("versionId") != "v1" {
			t.Errorf("version missing from %s", r.URL)
		}
		if r.Method == http.MethodHead {
			heads.Add(1)
			w.Header().Set("Content-Length", "7")
			w.Header().Set("Last-Modified", "Thu, 01 Oct 2026 00:00:00 GMT")
			return
		}
		gets.Add(1)
		if r.Header.Get("Range") != "bytes=0-2" {
			t.Errorf("unexpected range: %q", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", "bytes 0-2/7")
		w.WriteHeader(http.StatusPartialContent)
		io.WriteString(w, "con")
	}))
	defer server.Close()
	cc := CopyCommand{command: localOSSCommand(t, server.URL)}
	cc.monitor.init(operationTypeGet)
	cc.cpOption.force, cc.cpOption.threshold, cc.cpOption.vrange, cc.cpOption.versionId = true, 1024, "0-2", "v1"
	cc.cpOption.options = []oss.Option{oss.VersionId("v1")}
	bucket, err := cc.command.ossBucket("bucket")
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, cc.downloadSingleFileWithReport(bucket, objectInfoType{relativeKey: "key", size: -1}, dir+string(os.PathSeparator)))
	data, err := os.ReadFile(filepath.Join(dir, "key"))
	require.NoError(t, err)
	require.Equal(t, "con", string(data))
	require.EqualValues(t, 1, heads.Load())
	require.EqualValues(t, 1, gets.Load())
	require.EqualValues(t, 1, cc.monitor.getSnapshot().okNum)
}

func TestCopyRejectsInvalidBucketsBeforeStartingWorkers(t *testing.T) {
	cc := CopyCommand{command: localOSSCommand(t, "http://127.0.0.1")}
	bad := CloudURL{bucket: "INVALID", object: "key"}
	good := CloudURL{bucket: "bucket", object: "key"}
	require.Error(t, cc.uploadFiles(nil, bad))
	require.Error(t, cc.downloadFiles(bad, FileURL{}))
	require.Error(t, cc.copyFiles(bad, good))
	_, err := cc.skipCopy(bad, "key", time.Now())
	require.Error(t, err)
	require.Error(t, cc.copyFiles(good, CloudURL{bucket: "other", object: "/invalid"}))
	partSize, partCount := cc.calcPartSize(5 * 1024 * 1024)
	require.GreaterOrEqual(t, partSize, int64(oss.MinPartSize))
	require.LessOrEqual(t, partCount, int64(MinIdealPartNum))
	require.Equal(t, (int64(5*1024*1024)-1)/partSize+1, partCount)
}
