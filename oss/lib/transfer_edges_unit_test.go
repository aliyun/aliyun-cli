package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestCopyPartitionedSymlinkMetadata(t *testing.T) {
	previous := chProgressSignal
	chProgressSignal = make(chan chProgressSignalType, 10)
	t.Cleanup(func() { chProgressSignal = previous })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "5")
			return
		}
		require.Equal(t, "folder/", r.URL.Query().Get("prefix"))
		io.WriteString(w, `<ListBucketResult><Contents><Key>folder/link</Key><Type>Symlink</Type><Size>0</Size></Contents><Contents><Key>folder/ignored</Key><Type>Normal</Type><Size>3</Size></Contents></ListBucketResult>`)
	}))
	defer server.Close()
	cc := CopyCommand{command: localOSSCommand(t, server.URL)}
	cc.cpOption.recursive, cc.cpOption.onlyCurrentDir = true, true
	cc.cpOption.opType = operationTypeGet
	cc.cpOption.partitionIndex, cc.cpOption.partitionCount = 1, 1
	cc.cpOption.filters = []filterOptionType{{IncludePrompt, "link"}}
	cc.monitor.init(operationTypeGet)
	bucket, err := cc.command.ossBucket("bucket")
	require.NoError(t, err)
	cloud := CloudURL{bucket: "bucket", object: "folder/"}
	cc.objectStatistic(bucket, cloud)
	require.Equal(t, int64(5), cc.monitor.totalSize)
	require.Equal(t, int64(1), cc.monitor.totalNum)
	objects, errs := make(chan objectInfoType, 3), make(chan error, 1)
	cc.objectProducer(bucket, cloud, objects, errs)
	require.NoError(t, <-errs)
	object := <-objects
	require.Equal(t, "folder/", object.prefix)
	require.Equal(t, "link", object.relativeKey)
	require.Equal(t, int64(5), object.size)
	require.Empty(t, objects)
	common := make(chan objectInfoType, 3)
	require.NoError(t, getObjectListCommon(bucket, cloud, common, true, cc.cpOption.filters, nil))
	object = <-common
	require.Equal(t, "link", object.relativeKey)
	require.Equal(t, int64(5), object.size)
}

func TestCopyMetadataReadAndDestinationFailures(t *testing.T) {
	for _, tc := range []struct {
		name, length, modified string
		status                 int
	}{
		{"denied", "", "", 403}, {"missing size", "", "", 200}, {"invalid time", "7", "invalid", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.length != "" {
					w.Header().Set("Content-Length", tc.length)
				}
				if tc.modified != "" {
					w.Header().Set("Last-Modified", tc.modified)
				}
				w.WriteHeader(tc.status)
			}))
			defer server.Close()
			cc := CopyCommand{command: localOSSCommand(t, server.URL)}
			cc.cpOption.force = true
			bucket, err := cc.command.ossBucket("bucket")
			require.NoError(t, err)
			object := objectInfoType{relativeKey: "key", size: -1}
			_, err, _, _ = cc.downloadSingleFile(bucket, object, filepath.Join(t.TempDir(), "download"))
			require.Error(t, err)
			_, err, _, _ = cc.copySingleFile(bucket, object, CloudURL{bucket: "bucket", object: "key"}, CloudURL{bucket: "other", object: "key"})
			require.Error(t, err)
		})
	}
	server, _ := newDataServer(t)
	defer server.Close()
	cc := CopyCommand{command: localOSSCommand(t, server.URL)}
	cc.cpOption.force = true
	bucket, err := cc.command.ossBucket("bucket")
	require.NoError(t, err)
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
	_, err, _, _ = cc.downloadSingleFile(bucket, objectInfoType{relativeKey: "key", size: 7}, file+string(os.PathSeparator)+"child")
	require.Error(t, err)
	skipped, err, _, _ := cc.downloadSingleFile(bucket, objectInfoType{relativeKey: "directory/", size: 0}, filepath.Join(dir, "directory"))
	require.NoError(t, err)
	require.False(t, skipped)
	require.DirExists(t, filepath.Join(dir, "directory"))
	cc.cpOption.startTime = time.Now().Add(time.Hour).Unix()
	skipped, err, size, _ := cc.downloadSingleFile(bucket, objectInfoType{relativeKey: "key", size: 7}, file)
	require.NoError(t, err)
	require.True(t, skipped)
	require.Equal(t, int64(7), size)
	_, err, _, _ = cc.copySingleFile(bucket, objectInfoType{relativeKey: "key", size: 7}, CloudURL{bucket: "bucket"}, CloudURL{bucket: "other"})
	require.NoError(t, err)
}

func TestCopyRejectsInvalidTargetsAndDirectorySources(t *testing.T) {
	cc := CopyCommand{command: localOSSCommand(t, "http://127.0.0.1")}
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
	for _, source := range []string{filepath.Join(dir, "missing"), dir} {
		_, err := cc.adjustDestURLForUpload([]StorageURLer{FileURL{urlStr: source}}, CloudURL{bucket: "bucket"})
		require.Error(t, err)
	}
	cc.cpOption.recursive = true
	dest, err := cc.adjustDestURLForUpload([]StorageURLer{FileURL{urlStr: dir}}, CloudURL{bucket: "bucket", object: "prefix"})
	require.NoError(t, err)
	require.Equal(t, "prefix/", dest.object)
	for _, object := range []string{"/key", "\\key"} {
		require.Error(t, cc.uploadFiles([]StorageURLer{FileURL{urlStr: file}}, CloudURL{bucket: "bucket", object: object}))
	}
	cc.cpOption.recursive = false
	for _, object := range []string{"", "directory/"} {
		require.Error(t, cc.downloadFiles(CloudURL{bucket: "bucket", object: object}, FileURL{urlStr: dir}))
		require.Error(t, cc.copyFiles(CloudURL{bucket: "bucket", object: object}, CloudURL{bucket: "other", object: "copy"}))
	}
	require.Error(t, cc.checkCopyFileArgs(CloudURL{bucket: "bucket", object: "same"}, CloudURL{bucket: "bucket", object: "same"}))
	cc.cpOption.recursive = true
	for _, pair := range [][2]string{{"prefix", "prefix/child"}, {"prefix/child", "prefix"}} {
		require.Error(t, cc.checkCopyFileArgs(CloudURL{bucket: "bucket", object: pair[0]}, CloudURL{bucket: "bucket", object: pair[1]}))
	}
	cc.cpOption.meta = "x-oss-meta-team:cli"
	require.NoError(t, cc.checkCopyFileArgs(CloudURL{bucket: "bucket", object: "same"}, CloudURL{bucket: "bucket", object: "same"}))
	require.False(t, cc.filterFile(fileInfoType{dir: dir + string(os.PathSeparator), filePath: "file"}, file))
	cc.cpOption.cpDir = filepath.Join(dir, "checkpoint")
	require.NoError(t, os.Mkdir(cc.cpOption.cpDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(cc.cpOption.cpDir, "ignored"), []byte("ignored"), 0600))
	cc.monitor.init(operationTypePut)
	require.NoError(t, cc.getFileListStatistic(dir))
	require.Equal(t, int64(7), cc.monitor.totalSize)
}

func TestSyncBackupCollisionsAndCopyFallbackErrors(t *testing.T) {
	dir := t.TempDir()
	source, backup := filepath.Join(dir, "source"), filepath.Join(dir, "backup")
	require.NoError(t, os.MkdirAll(filepath.Join(source, "child"), 0700))
	require.NoError(t, os.Mkdir(backup, 0700))
	sc := SyncCommand{}
	sc.syncOption.backupDir = backup + string(os.PathSeparator)
	require.NoError(t, os.WriteFile(filepath.Join(source, "child", "kept"), []byte("kept"), 0600))
	require.NoError(t, sc.RemoveExtraFiles(map[string]string{"child" + string(os.PathSeparator): ""}, FileURL{urlStr: source + string(os.PathSeparator)}))
	require.FileExists(t, filepath.Join(source, "child", "kept"))
	require.NoError(t, os.Remove(filepath.Join(source, "child", "kept")))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "child"), nil, 0600))
	require.Error(t, sc.RemoveExtraFiles(map[string]string{"child" + string(os.PathSeparator): ""}, FileURL{urlStr: source + string(os.PathSeparator)}))
	require.NoError(t, os.Remove(filepath.Join(backup, "child")))
	require.NoError(t, os.Mkdir(filepath.Join(backup, "child"), 0700))
	require.NoError(t, sc.RemoveExtraFiles(map[string]string{"child" + string(os.PathSeparator): ""}, FileURL{urlStr: source + string(os.PathSeparator)}))
	require.NoDirExists(t, filepath.Join(source, "child"))
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
	require.Error(t, sc.moveFileToPath(file, filepath.Join(dir, "missing", "copy")))
	require.FileExists(t, file)
	sc.syncOption.backupDir = file
	require.Error(t, sc.CheckDestBackupDir(FileURL{urlStr: source}))
	missingKeys := make(chan fileInfoType, 2)
	finish := make(chan error, 2)
	sc.GetFileList(filepath.Join(dir, "missing"), missingKeys, finish)
	require.Error(t, <-finish)
	sc.command = Command{}
	require.Error(t, sc.GetOssKeys(CloudURL{bucket: "bucket"}, map[string]string{}))
	for _, pair := range [][2]string{{file, file}, {filepath.Join(dir, "missing"), "oss://bucket"}, {file, "oss://bucket"}, {"%zz", "oss://bucket"}, {"oss://bucket", "%zz"}} {
		cmd := SyncCommand{command: dataCommand(t, "http://127.0.0.1", syncCommand.command, pair[:], OptionMapType{OptionEncodingType: stringPointer(URLEncodingType)})}
		require.Error(t, cmd.RunCommand())
	}
}

func TestRemoveFilteredStatisticsAndDeleteMarkers(t *testing.T) {
	server, writes := newDataServer(t)
	defer server.Close()
	rc := RemoveCommand{command: localOSSCommand(t, server.URL)}
	bucket, err := rc.command.ossBucket("bucket")
	require.NoError(t, err)
	for _, recursive := range []bool{false, true} {
		rc.monitor.init()
		rc.rmOption.recursive = recursive
		rc.filters = []filterOptionType{{IncludePrompt, "first"}}
		require.NoError(t, rc.multipartUploadsStatistic(bucket, CloudURL{bucket: "bucket", object: "first"}))
		require.Equal(t, int64(1), rc.monitor.totalUploadIdNum)
	}
	rc.monitor.init()
	require.NoError(t, rc.batchObjectStatistic(bucket, CloudURL{bucket: "bucket"}))
	require.Equal(t, int64(1), rc.monitor.totalObjectNum)
	rc.monitor.init()
	require.NoError(t, rc.batchObjectStatisticVersion(bucket, CloudURL{bucket: "bucket"}))
	require.Equal(t, int64(1), rc.monitor.totalObjectNum)
	rc.monitor.init()
	require.NoError(t, rc.removeObjectAllVersion(bucket, CloudURL{bucket: "bucket", object: "second"}))
	require.Equal(t, int64(1), rc.monitor.objectNum)
	require.Positive(t, writes.Load())
}

func TestListMarkerFailuresAndFiltering(t *testing.T) {
	base := dataCommand(t, "http://127.0.0.1", listCommand.command, nil, nil)
	base.options[OptionEncodingType] = stringPointer(URLEncodingType)
	lc := ListCommand{command: base}
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	for _, name := range []string{OptionMarker, OptionVersionIdMarker, OptionUploadIDMarker} {
		base.options[name] = stringPointer("%zz")
		limited := int64(-1)
		if name == OptionMarker {
			require.Error(t, lc.listBuckets(""))
			_, err = lc.listObjects(bucket, CloudURL{bucket: "bucket"}, false, false, &limited)
			require.Error(t, err)
		}
		if name != OptionUploadIDMarker {
			_, err = lc.listObjectVersions(bucket, CloudURL{bucket: "bucket"}, false, false, &limited)
			require.Error(t, err)
		}
		if name != OptionVersionIdMarker {
			_, err = lc.listMultipartUploads(bucket, CloudURL{bucket: "bucket"}, false, true, &limited)
			require.Error(t, err)
		}
		base.options[name] = stringPointer("")
	}
	lc.command.options[OptionDirectory] = boolPointer(true)
	require.Error(t, lc.listBuckets(""))
	lc.filters = []filterOptionType{{IncludePrompt, "match"}}
	objects := oss.ListObjectsResult{Objects: []oss.ObjectProperties{{Key: "ignored"}, {Key: "match"}, {Key: "match"}}, CommonPrefixes: []string{"ignored", "match", "match"}}
	versions := oss.ListObjectVersionsResult{ObjectVersions: []oss.ObjectVersionProperties{{Key: "ignored"}, {Key: "match"}, {Key: "match"}}, ObjectDeleteMarkers: []oss.ObjectDeleteMarkerProperties{{Key: "ignored"}, {Key: "match"}, {Key: "match"}}, CommonPrefixes: []string{"ignored", "match", "match"}}
	uploads := oss.ListMultipartUploadResult{Uploads: []oss.UncompletedUpload{{Key: "ignored"}, {Key: "match"}, {Key: "match"}}}
	for _, show := range []func(*int64) int64{
		func(n *int64) int64 { return lc.showObjects(objects, "bucket", true, n) },
		func(n *int64) int64 { return lc.showDirectories(objects, "bucket", n) },
		func(n *int64) int64 { return lc.showDirectoriesVersion(versions, "bucket", n) },
		func(n *int64) int64 { return lc.showObjectVersions(versions, "bucket", n, true) },
		func(n *int64) int64 { return lc.showMultipartUploads(uploads, "bucket", true, n) },
	} {
		limited := int64(1)
		captureBucketOutput(t, func() { require.Equal(t, int64(1), show(&limited)) })
		require.Zero(t, limited)
	}
}
