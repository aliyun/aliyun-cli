package lib

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestCopyDirectoryUploadsAndSnapshotFailures(t *testing.T) {
	previousSignal := chProgressSignal
	t.Cleanup(func() { chProgressSignal = previousSignal })
	chProgressSignal = make(chan chProgressSignalType, 100)
	var uploaded atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			uploaded.Add(1)
			w.Header().Set("ETag", `"directory"`)
		}
	}))
	defer server.Close()
	cc := CopyCommand{command: localOSSCommand(t, server.URL)}
	cc.monitor.init(operationTypePut)
	cc.cpOption.force, cc.cpOption.threshold = true, 1024
	bucket, err := cc.command.ossBucket("bucket")
	require.NoError(t, err)
	dir := t.TempDir()
	file := fileInfoType{dir: dir, filePath: "folder"}
	require.NoError(t, os.Mkdir(filepath.Join(dir, file.filePath), 0700))
	cloud := CloudURL{bucket: "bucket", object: "prefix/"}
	for _, disabled := range []bool{true, false} {
		cc.cpOption.disableDirObject = disabled
		skip, err, isDir, size, _ := cc.uploadFile(bucket, cloud, file)
		require.NoError(t, err)
		require.Equal(t, disabled, skip)
		require.True(t, isDir)
		require.Zero(t, size)
		cc.updateMonitor(skip, err, isDir, size)
	}
	require.EqualValues(t, 1, uploaded.Load())
	snapshot := cc.monitor.getSnapshot()
	require.EqualValues(t, 1, snapshot.dirNum)
	require.EqualValues(t, 1, snapshot.skipNumDir)
	skip, err, _, _, _ := cc.uploadFile(bucket, cloud, fileInfoType{dir: dir, filePath: "missing"})
	require.ErrorIs(t, err, os.ErrNotExist)
	require.False(t, skip)
	db, err := leveldb.OpenFile(filepath.Join(t.TempDir(), "snapshot"), nil)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	cc.cpOption.snapshotPath, cc.cpOption.snapshotldb = "snapshot", db
	_, err, isDir, _, _ := cc.uploadFile(bucket, cloud, file)
	require.ErrorContains(t, err, leveldb.ErrClosed.Error())
	require.True(t, isDir)
	file.filePath = "file"
	require.NoError(t, os.WriteFile(filepath.Join(dir, file.filePath), []byte("content"), 0600))
	_, err, _, _, _ = cc.uploadFile(bucket, cloud, file)
	require.ErrorContains(t, err, leveldb.ErrClosed.Error())
	require.EqualValues(t, 3, uploaded.Load())
	cc.cpOption.snapshotPath, cc.cpOption.startTime = "", time.Now().Add(time.Hour).Unix()
	require.NoError(t, cc.uploadFileWithReport(bucket, cloud, file))
	require.EqualValues(t, 1, cc.monitor.getSnapshot().skipNum)
}

func TestCopyStatisticsPropagateMetadataFailures(t *testing.T) {
	previousSignal := chProgressSignal
	t.Cleanup(func() { chProgressSignal = previousSignal })
	chProgressSignal = make(chan chProgressSignalType, 100)
	for _, scenario := range []string{"list forbidden", "head forbidden", "invalid size", "version"} {
		t.Run(scenario, func(t *testing.T) {
			var version string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(scenario, "forbidden") {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				version = r.URL.Query().Get("versionId")
				if scenario == "version" {
					w.Header().Set("Content-Length", "7")
				}
			}))
			defer server.Close()
			cc := CopyCommand{command: localOSSCommand(t, server.URL)}
			cc.monitor.init(operationTypeGet)
			cc.cpOption.recursive = scenario == "list forbidden"
			cc.cpOption.versionId = "v1"
			bucket, err := cc.command.ossBucket("bucket")
			require.NoError(t, err)
			cc.objectStatistic(bucket, CloudURL{bucket: "bucket", object: "key"})
			if scenario == "version" {
				require.NoError(t, cc.monitor.seekAheadError)
				require.EqualValues(t, 7, cc.monitor.totalSize)
				require.Equal(t, "v1", version)
			} else {
				require.Error(t, cc.monitor.seekAheadError)
			}
		})
	}
	cc := CopyCommand{command: localOSSCommand(t, "unused")}
	for _, current := range []bool{false, true} {
		cc.cpOption.onlyCurrentDir = current
		missing := filepath.Join(t.TempDir(), "missing")
		require.ErrorIs(t, cc.getFileList(missing, make(chan fileInfoType, 1)), os.ErrNotExist)
		require.ErrorIs(t, cc.getFileListStatistic(missing), os.ErrNotExist)
	}
	missing := FileURL{urlStr: filepath.Join(t.TempDir(), "missing")}
	files, finished := make(chan fileInfoType, 1), make(chan error, 1)
	cc.fileProducer([]StorageURLer{missing}, files, finished)
	require.ErrorIs(t, <-finished, os.ErrNotExist)
	_, open := <-files
	require.False(t, open)
}

func TestSyncDownloadsBacksUpAndDeletesOnlyExtraEntries(t *testing.T) {
	previousCommand, previousOptions, previousSignal := copyCommand.command, copyCommand.cpOption, chProgressSignal
	t.Cleanup(func() {
		copyCommand.command, copyCommand.cpOption, chProgressSignal = previousCommand, previousOptions, previousSignal
	})
	var deleted atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "7")
			w.Header().Set("Last-Modified", "Thu, 01 Oct 2026 00:00:00 GMT")
			return
		}
		if r.Method == http.MethodPost {
			var request struct {
				Objects []struct{ Key string } `xml:"Object"`
			}
			if err := xml.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			deleted.Add(int32(len(request.Objects)))
			io.WriteString(w, `<DeleteResult/>`)
			return
		}
		if strings.Trim(r.URL.Path, "/") == "bucket" {
			io.WriteString(w, `<ListBucketResult><Contents><Key>kept</Key><Size>7</Size><LastModified>2026-10-01T00:00:00.000Z</LastModified></Contents></ListBucketResult>`)
			return
		}
		io.WriteString(w, "content")
	}))
	defer server.Close()
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "copy", true: "backup"}[remove], func(t *testing.T) {
			dest := t.TempDir()
			backup := filepath.Join(t.TempDir(), "backup")
			require.NoError(t, os.WriteFile(filepath.Join(dest, "extra"), []byte("old"), 0600))
			base := dataCommand(t, server.URL, syncCommand.command, []string{"oss://bucket/", dest + string(os.PathSeparator)}, OptionMapType{OptionDelete: boolPointer(remove), OptionBackupDir: stringPointer(backup)})
			copyCommand.command = dataCommand(t, server.URL, copyCommand.command, base.args, OptionMapType{OptionRecursion: boolPointer(true)})
			copyCommand.cpOption.bSyncCommand = true
			sc := SyncCommand{command: base}
			require.NoError(t, sc.RunCommand())
			data, err := os.ReadFile(filepath.Join(dest, "kept"))
			require.NoError(t, err)
			require.Equal(t, "content", string(data))
			if remove {
				require.NoFileExists(t, filepath.Join(dest, "extra"))
				data, err = os.ReadFile(filepath.Join(backup, "extra"))
				require.NoError(t, err)
				require.Equal(t, "old", string(data))
				require.Equal(t, 1, sc.syncOption.removeCount)
			} else {
				require.FileExists(t, filepath.Join(dest, "extra"))
			}
		})
	}
	require.Zero(t, deleted.Load())
}

func TestSyncKeyLimitsAndDeleteFailures(t *testing.T) {
	previousLimit, previousOptions := MaxSyncNumbers, copyCommand.cpOption
	t.Cleanup(func() { MaxSyncNumbers, copyCommand.cpOption = previousLimit, previousOptions })
	MaxSyncNumbers = 1
	sc := SyncCommand{}
	sc.syncOption.cpDir = filepath.Join(t.TempDir(), "checkpoint")
	files, finished := make(chan fileInfoType, 2), make(chan error, 2)
	files <- fileInfoType{filePath: "one"}
	files <- fileInfoType{filePath: "two"}
	close(files)
	keys := map[string]string{}
	sc.ReadLocalFileKeys(files, finished, keys)
	require.ErrorContains(t, <-finished, "over max sync numbers 1")
	require.NoError(t, <-finished)
	require.Len(t, keys, 2)
	objects := make(chan objectInfoType, 2)
	objects <- objectInfoType{relativeKey: "one", prefix: "prefix/"}
	objects <- objectInfoType{relativeKey: "two", prefix: "prefix/"}
	close(objects)
	keys = map[string]string{}
	sc.ReadOssKeys(keys, CloudURL{bucket: "bucket"}, objects, finished)
	require.ErrorContains(t, <-finished, "over max sync numbers 1")
	require.NoError(t, <-finished)
	require.Equal(t, "prefix/", keys["two"])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	sc.command, sc.syncOption.force = localOSSCommand(t, server.URL), true
	for _, count := range []int{1, MaxBatchCount + 1} {
		keys = map[string]string{}
		for i := range count {
			keys[strings.Repeat("a", i+1)] = "prefix/"
		}
		require.ErrorContains(t, sc.DeleteExtraObjects(keys, CloudURL{bucket: "bucket"}), "403")
	}
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("blocked"), 0600))
	require.Error(t, sc.CheckDestBackupDir(FileURL{urlStr: filepath.Join(blocker, "child")}))
	sc.syncOption.backupDir = ""
	newDest := filepath.Join(root, "new")
	require.NoError(t, sc.CheckDestBackupDir(FileURL{urlStr: newDest}))
	require.DirExists(t, newDest)
	sc.syncOption.backupDir = blocker
	var pathError *os.PathError
	require.ErrorAs(t, sc.CheckDestBackupDir(FileURL{urlStr: newDest}), &pathError)
	require.Equal(t, filepath.Clean(blocker), filepath.Clean(pathError.Path))
	sc.syncOption.backupDir = filepath.Join(root, "backup") + string(os.PathSeparator)
	require.NoError(t, os.MkdirAll(sc.syncOption.backupDir, 0700))
	require.ErrorIs(t, sc.RemoveExtraFiles(map[string]string{"missing": ""}, FileURL{urlStr: newDest + string(os.PathSeparator)}), os.ErrNotExist)
	require.True(t, errors.As(sc.moveFileToPath(filepath.Join(root, "missing"), filepath.Join(root, "copy")), &pathError))
}

func TestCopyRetriesRejectedRequestsUntilTransferSucceeds(t *testing.T) {
	for _, operation := range []string{"directory", "upload", "multipart upload", "download", "multipart download", "copy", "multipart copy"} {
		t.Run(operation, func(t *testing.T) {
			var attempts atomic.Int32
			client := &http.Client{Transport: maintenanceTransport(func(r *http.Request) (*http.Response, error) {
				status, body := http.StatusOK, "content"
				if attempts.Add(1) == 1 {
					status, body = http.StatusTooManyRequests, `<Error><Code>Throttling</Code><Message>retry locally</Message></Error>`
				} else {
					switch {
					case r.Method == http.MethodHead:
						body = ""
					case r.Method == http.MethodPost && r.URL.Query().Has("uploads"):
						body = `<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>key</Key><UploadId>upload</UploadId></InitiateMultipartUploadResult>`
					case r.Method == http.MethodPost:
						body = `<CompleteMultipartUploadResult><ETag>etag</ETag></CompleteMultipartUploadResult>`
					case r.Method == http.MethodPut && r.Header.Get("X-Oss-Copy-Source") != "":
						if r.URL.Query().Has("uploadId") {
							body = `<CopyPartResult><ETag>etag</ETag></CopyPartResult>`
						} else {
							body = `<CopyObjectResult><ETag>etag</ETag></CopyObjectResult>`
						}
					case r.Method == http.MethodPut:
						if r.Body != nil {
							if _, err := io.Copy(io.Discard, r.Body); err != nil {
								t.Error(err)
							}
						}
						body = ""
					}
					if r.Method == http.MethodGet && r.Header.Get("Range") != "" {
						var start, end int
						if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end); err != nil {
							t.Error(err)
						} else {
							body, status = body[start:end+1], http.StatusPartialContent
						}
					}
				}
				header := http.Header{"Content-Length": {fmt.Sprint(len(body))}, "Etag": {`"etag"`}}
				if r.Method == http.MethodHead && status == 200 {
					header.Set("Content-Length", "7")
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: r}, nil
			})}
			previous := newOSSClient
			newOSSClient = func(endpoint, id, secret string, options ...oss.ClientOption) (*oss.Client, error) {
				return previous(endpoint, id, secret, append(options, oss.HTTPClient(client))...)
			}
			defer func() { newOSSClient = previous }()
			cc := CopyCommand{command: localOSSCommand(t, "http://oss.test")}
			cc.command.options[OptionRetryTimes] = stringPointer("2")
			bucket, err := cc.command.ossBucket("bucket")
			require.NoError(t, err)
			file := filepath.Join(t.TempDir(), "source")
			require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
			cc.cpOption.cpDir = t.TempDir()
			download := filepath.Join(t.TempDir(), "download")
			switch operation {
			case "directory":
				err = cc.ossPutObjectRetry(bucket, "key/", "")
			case "upload":
				err = cc.ossUploadFileRetry(bucket, "key", file)
			case "multipart upload":
				err = cc.ossResumeUploadRetry(bucket, "key", file, 1024*1024)
			case "download":
				err = cc.ossDownloadFileRetry(bucket, "key", download)
			case "multipart download":
				err = cc.ossResumeDownloadRetry(bucket, "key", download, 7, 1024*1024)
			case "copy":
				err = cc.ossCopyObjectRetry(bucket, "key", "bucket", "copy")
			case "multipart copy":
				err = cc.ossResumeCopyRetry("bucket", "key", "bucket", "copy", 1024*1024)
			}
			require.NoError(t, err)
			require.GreaterOrEqual(t, attempts.Load(), int32(2))
			if strings.Contains(operation, "download") {
				data, err := os.ReadFile(download)
				require.NoError(t, err)
				require.Equal(t, "content", string(data))
			}
		})
	}
}

func TestSyncPayerFiltersAndEmptyDirectoryBackup(t *testing.T) {
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
	for _, tc := range []struct{ payer, filter, want string }{
		{"invalid", "", "invalid request payer"},
		{"requester", "dir/file", "does not support format containing dir info"},
		{"requester", "*.txt", "is not directory"},
	} {
		os.Args = []string{"oss", "sync"}
		if tc.filter != "" {
			os.Args = append(os.Args, "--include="+tc.filter)
		}
		base := dataCommand(t, "http://127.0.0.1", syncCommand.command, []string{file, "oss://bucket/"}, OptionMapType{OptionRequestPayer: &tc.payer})
		sc := SyncCommand{command: base}
		require.ErrorContains(t, sc.RunCommand(), tc.want)
		if tc.payer == "requester" {
			require.Len(t, sc.syncOption.payerOptions, 1)
		}
		if tc.filter == "*.txt" {
			require.Equal(t, []filterOptionType{{name: IncludePrompt, pattern: "*.txt"}}, sc.syncOption.filters)
		}
	}
	source, backup := t.TempDir(), t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(source, "empty"), 0700))
	sc := SyncCommand{}
	sc.syncOption.backupDir = backup + string(os.PathSeparator)
	require.NoError(t, sc.RemoveExtraFiles(map[string]string{"empty" + string(os.PathSeparator): ""}, FileURL{urlStr: source + string(os.PathSeparator)}))
	require.NoDirExists(t, filepath.Join(source, "empty"))
	require.DirExists(t, filepath.Join(backup, "empty"))
	require.Equal(t, 1, sc.syncOption.removeCount)
	adjusted := sc.adjustCloudUrl(CloudURL{bucket: "bucket", object: "prefix"}).(CloudURL)
	require.Equal(t, "prefix/", adjusted.object)
}
