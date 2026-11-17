package lib

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestObjectFileReadersAndSnapshotLifecycle(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "objects")
	base := localOSSCommand(t, "http://127.0.0.1")
	base.options[OptionEncodingType] = stringPointer(URLEncodingType)
	rc, sc := RestoreCommand{command: base}, SetMetaCommand{command: base}
	for _, tc := range []struct {
		text   string
		want   []string
		failed bool
	}{
		{" first\nsecond%20key\n", []string{"first", "second key"}, false},
		{"first\n \n", []string{"first"}, true},
		{"%zz\n", nil, true},
	} {
		require.NoError(t, os.WriteFile(file, []byte(tc.text), 0600))
		for _, producer := range []func(string, chan<- string, chan<- error, []filterOptionType, ...oss.Option){rc.restoreProducer, sc.setObjectMetaProducer} {
			objects, errs := make(chan string, 4), make(chan error, 1)
			producer(file, objects, errs, nil)
			got := []string{}
			for object := range objects {
				got = append(got, object)
			}
			require.Equal(t, tc.want, append([]string(nil), got...))
			if tc.failed {
				require.Error(t, <-errs)
			} else {
				require.NoError(t, <-errs)
			}
		}
		for _, statistic := range []func(string, Monitorer, []filterOptionType, ...oss.Option){rc.restoreStatistic, sc.setObjectMetaStatistic} {
			var monitor Monitor
			monitor.init("Scan")
			statistic(file, &monitor, nil)
			require.True(t, monitor.seekAheadEnd)
			if tc.text == "first\n \n" {
				require.Error(t, monitor.seekAheadError)
			}
			statistic(file, nil, nil)
		}
	}
	for _, target := range []string{dir, filepath.Join(dir, "missing"), file} {
		if target == file {
			require.NoError(t, os.WriteFile(file, nil, 0600))
		}
		require.Error(t, rc.checkObjectFile(target))
		require.Error(t, sc.checkObjectFile(target))
		require.Error(t, rc.parseRestoreConf(target))
	}
	require.NoError(t, os.WriteFile(file, []byte("first\n"), 0600))
	require.NoError(t, rc.checkObjectFile(file))
	require.NoError(t, sc.checkObjectFile(file))
	require.True(t, rc.hasObjFile)
	require.True(t, sc.hasObjFile)
	require.NoError(t, rc.parseRestoreConf(file))
	require.Equal(t, "first\n", rc.configXml)
	for _, producer := range []func(string, chan<- string, chan<- error, []filterOptionType, ...oss.Option){rc.restoreProducer, sc.setObjectMetaProducer} {
		objects, errs := make(chan string, 1), make(chan error, 1)
		producer(filepath.Join(dir, "missing"), objects, errs, nil)
		require.Error(t, <-errs)
	}
	for _, statistic := range []func(string, Monitorer, []filterOptionType, ...oss.Option){rc.restoreStatistic, sc.setObjectMetaStatistic} {
		var monitor Monitor
		statistic(filepath.Join(dir, "missing"), &monitor, nil)
		require.Error(t, monitor.seekAheadError)
	}
	db, err := leveldb.OpenFile(filepath.Join(dir, "snapshot"), nil)
	require.NoError(t, err)
	defer db.Close()
	rc.reOption.snapshotPath, rc.reOption.snapshotldb = "snapshot", db
	sc.smOption.snapshotPath, sc.smOption.snapshotldb = "snapshot", db
	key := rc.formatSnapshotKey("bucket", "first", "restore")
	require.False(t, rc.skipRestore(key))
	require.NoError(t, rc.updateSnapshot(nil, key, 123))
	require.True(t, rc.skipRestore(key))
	key = sc.formatSnapshotKey("bucket", "first", "meta")
	require.False(t, sc.skipSetMeta(key))
	require.NoError(t, sc.updateSnapshot(nil, key, 456))
	require.True(t, sc.skipSetMeta(key))
	rc.updateSkip(2)
	sc.updateSkip(3)
	require.Equal(t, int64(2), rc.monitor.skipNum)
	require.Equal(t, int64(3), sc.monitor.skipNum)
	require.NoError(t, rc.updateSnapshot(errors.New("failed operation"), "failed", 0))
	require.False(t, rc.skipRestore("failed"))
	require.NoError(t, db.Close())
	require.Error(t, rc.updateSnapshot(nil, "closed", 0))
	require.Error(t, sc.updateSnapshot(nil, "closed", 0))
}

func TestCopyRangePartAndArgumentBoundaries(t *testing.T) {
	cc := CopyCommand{command: localOSSCommand(t, "http://127.0.0.1")}
	for _, tc := range []struct {
		text string
		want int64
	}{
		{"-3", 3}, {"3-", 7}, {"2-4", 3}, {"x-", 10}, {"-x", 10}, {"x-y", 10}, {"12-", 10}, {"4-2", 10}, {"missing", 10}, {"2-12", 10},
	} {
		got, err := cc.parseRange(tc.text, 10)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
		cc.cpOption.vrange = tc.text
		require.Equal(t, tc.want, cc.getRangeSize(10))
	}
	cc.cpOption.vrange = ""
	require.Equal(t, int64(10), cc.getRangeSize(10))
	cc.command.options[OptionPartSize] = stringPointer("102400")
	for _, tc := range []struct {
		parts    int64
		routines int
	}{{1, 1}, {3, 2}, {20, 4}, {300, 8}, {500, 10}, {501, 12}} {
		size, routines := cc.preparePartOption(tc.parts * 102400)
		require.Equal(t, int64(102400), size)
		require.Equal(t, tc.routines, routines)
	}
	cc.command.options[OptionParallel] = stringPointer("7")
	_, routines := cc.preparePartOption(102400)
	require.Equal(t, 7, routines)
	cc.command.options[OptionPartSize] = stringPointer("")
	for _, size := range []int64{1, 1024 * 1024 * 1024, 1024 * 1024 * 1024 * 1024} {
		part, count := cc.calcPartSize(size)
		require.GreaterOrEqual(t, part, int64(oss.MinPartSize))
		require.Positive(t, count)
		require.GreaterOrEqual(t, part*count, size)
	}
	local, cloud := FileURL{urlStr: "file"}, CloudURL{bucket: "bucket", object: "key"}
	for _, tc := range []struct {
		sources     []StorageURLer
		destination StorageURLer
		op          operationType
	}{
		{[]StorageURLer{CloudURL{}}, local, operationTypeGet},
		{[]StorageURLer{local}, CloudURL{}, operationTypePut},
		{[]StorageURLer{local}, local, operationTypePut},
		{[]StorageURLer{local, cloud}, cloud, operationTypePut},
		{[]StorageURLer{local, local}, cloud, operationTypePut},
		{[]StorageURLer{cloud, cloud}, local, operationTypeGet},
		{[]StorageURLer{cloud, cloud}, cloud, operationTypeCopy},
	} {
		require.Error(t, cc.checkCopyArgs(tc.sources, tc.destination, tc.op))
	}
	cc.cpOption.snapshotPath = "snapshot"
	require.Error(t, cc.checkCopyOptions(operationTypeCopy))
	cc.cpOption.snapshotPath, cc.cpOption.vrange = "", "1-2"
	require.Error(t, cc.checkCopyOptions(operationTypePut))
	cc.cpOption.vrange, cc.cpOption.versionId = "", "version"
	require.Error(t, cc.checkCopyOptions(operationTypePut))
	cc.cpOption.recursive = true
	require.Error(t, cc.checkCopyOptions(operationTypeGet))
	cc.cpOption.recursive = false
	require.NoError(t, cc.checkCopyOptions(operationTypeGet))
	_, err := cc.getStorageURLs([]string{"oss://bucket/%zz"})
	require.NoError(t, err)
	cc.cpOption.encodingType = URLEncodingType
	_, err = cc.getStorageURLs([]string{"oss://bucket/%zz"})
	require.Error(t, err)
}

func TestSyncLocalBackupAndPathOperations(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	backup := filepath.Join(dir, "backup")
	require.NoError(t, os.MkdirAll(filepath.Join(source, "child"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(source, "child", "file"), []byte("preserved"), 0600))
	sc := SyncCommand{}
	require.Error(t, sc.CheckDestBackupDir(FileURL{urlStr: source}))
	sc.syncOption.backupDir = filepath.Join(source, "unsafe")
	require.Error(t, sc.CheckDestBackupDir(FileURL{urlStr: source}))
	sc.syncOption.backupDir = backup
	require.NoError(t, sc.CheckDestBackupDir(FileURL{urlStr: source}))
	require.DirExists(t, backup)
	require.NoError(t, sc.RemoveExtraFiles(map[string]string{filepath.Join("child", "file"): "", "child" + string(os.PathSeparator): ""}, FileURL{urlStr: source + string(os.PathSeparator)}))
	data, err := os.ReadFile(filepath.Join(backup, "child", "file"))
	require.NoError(t, err)
	require.Equal(t, "preserved", string(data))
	require.NoFileExists(t, filepath.Join(source, "child", "file"))
	require.NoError(t, sc.CheckDestBackupDir(FileURL{urlStr: filepath.Join(dir, "new-dest")}))
	require.Error(t, sc.CheckDestBackupDir(FileURL{urlStr: filepath.Join(backup, "child", "file")}))
	abs, err := sc.GetAbsPath("relative")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(abs))
	require.Equal(t, source, mustAbsoluteSyncPath(t, &sc, source))
	entries, err := sc.readDirLimit(backup, 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	_, err = sc.readDirLimit(filepath.Join(dir, "missing"), 1)
	require.Error(t, err)
	require.Error(t, sc.movePath(filepath.Join(dir, "missing"), filepath.Join(dir, "dest")))
	require.Error(t, sc.moveFileToPath(filepath.Join(dir, "missing"), filepath.Join(dir, "dest")))
}

func mustAbsoluteSyncPath(t *testing.T, sc *SyncCommand, path string) string {
	t.Helper()
	value, err := sc.GetAbsPath(path)
	require.NoError(t, err)
	return value
}

func TestCopySkipAndSnapshotDecisions(t *testing.T) {
	server, _ := newDataServer(t)
	defer server.Close()
	cc := CopyCommand{command: localOSSCommand(t, server.URL)}
	bucket, err := cc.command.ossBucket("bucket")
	require.NoError(t, err)
	dest := CloudURL{bucket: "bucket", object: "key"}
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
	when := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	cc.cpOption.startTime = when.Unix() + 1
	skipped, err := cc.skipUpload("key", bucket, "key", dest, when.Unix())
	require.NoError(t, err)
	require.True(t, skipped)
	skipped, err = cc.skipCopy(dest, "key", when)
	require.NoError(t, err)
	require.True(t, skipped)
	require.True(t, cc.skipDownload(file, when, "key"))
	cc.cpOption.startTime, cc.cpOption.endTime = 0, when.Unix()-1
	skipped, err = cc.skipUpload("key", bucket, "key", dest, when.Unix())
	require.NoError(t, err)
	require.True(t, skipped)
	skipped, err = cc.skipCopy(dest, "key", when)
	require.NoError(t, err)
	require.True(t, skipped)
	require.True(t, cc.skipDownload(file, when, "key"))
	cc.cpOption.endTime, cc.cpOption.update = 0, true
	skipped, err = cc.skipUpload("key", bucket, "key", dest, when.Unix())
	require.NoError(t, err)
	require.True(t, skipped)
	skipped, err = cc.skipCopy(dest, "key", when)
	require.NoError(t, err)
	require.True(t, skipped)
	require.True(t, cc.skipDownload(file, when, "key"))
	cc.cpOption.update = false
	withBucketInput(t, "n\n", func() {
		skipped, err = cc.skipUpload("key", bucket, "key", dest, when.Unix())
		require.NoError(t, err)
		require.True(t, skipped)
	})
	withBucketInput(t, "n\n", func() {
		skipped, err = cc.skipCopy(dest, "key", when)
		require.NoError(t, err)
		require.True(t, skipped)
	})
	withBucketInput(t, "n\n", func() { require.True(t, cc.skipDownload(file, when, "key")) })
	withBucketInput(t, "y\n", func() { require.True(t, cc.confirm("key")) })
	cc.cpOption.force = true
	skipped, err = cc.skipUpload("key", bucket, "key", dest, when.Unix())
	require.NoError(t, err)
	require.False(t, skipped)
	skipped, err = cc.skipCopy(dest, "key", when)
	require.NoError(t, err)
	require.False(t, skipped)
	require.False(t, cc.skipDownload(file, when, "key"))
	db, err := leveldb.OpenFile(filepath.Join(dir, "snapshot"), nil)
	require.NoError(t, err)
	defer db.Close()
	cc.cpOption.snapshotPath, cc.cpOption.snapshotldb = "snapshot", db
	key := cc.formatSnapshotKey(file, "bucket", "key")
	require.NoError(t, cc.updateSnapshot(nil, key, when.Unix()))
	skipped, err = cc.skipUpload(key, bucket, "key", dest, when.Unix())
	require.NoError(t, err)
	require.True(t, skipped)
	require.NoError(t, cc.updateSnapshot(nil, "oss://bucket/key", when.Unix()))
	require.True(t, cc.skipDownload(file, when, "oss://bucket/key"))
	require.NoError(t, db.Close())
	require.Error(t, cc.updateSnapshot(nil, "closed", 0))
	cc.cpOption.snapshotPath = ""
	require.NoError(t, cc.truncateFile(file, 2))
	info, err := os.Stat(file)
	require.NoError(t, err)
	require.Equal(t, int64(2), info.Size())
	require.Error(t, cc.truncateFile(filepath.Join(dir, "missing", "file"), 2))
}

func TestCopyDirectoryScanningAndSymlinks(t *testing.T) {
	previousProgress := chProgressSignal
	chProgressSignal = make(chan chProgressSignalType, 10)
	t.Cleanup(func() { chProgressSignal = previousProgress })
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "child"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child", "file"), []byte("data"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file"), []byte("data"), 0600))
	cc := CopyCommand{command: localOSSCommand(t, "http://127.0.0.1")}
	cc.cpOption.cpDir = filepath.Join(dir, "checkpoint")
	cc.cpOption.onlyCurrentDir = true
	cc.monitor.init(operationTypePut)
	cc.fileStatistic([]StorageURLer{FileURL{urlStr: dir}})
	require.Equal(t, int64(1), cc.monitor.totalNum)
	require.Equal(t, int64(4), cc.monitor.totalSize)
	for _, current := range []bool{true, false} {
		cc.cpOption.onlyCurrentDir = current
		ch := make(chan fileInfoType, 10)
		require.NoError(t, cc.getFileList(dir, ch))
		require.NotEmpty(t, ch)
	}
	if err := os.Symlink(filepath.Join(dir, "child"), filepath.Join(dir, "link")); err == nil {
		for _, disabled := range []bool{true, false} {
			cc.cpOption.disableAllSymlink = disabled
			cc.cpOption.enableSymlinkDir = !disabled
			cc.monitor.init(operationTypePut)
			require.NoError(t, cc.getFileListStatistic(dir))
			require.GreaterOrEqual(t, cc.monitor.totalNum, int64(3))
			ch := make(chan fileInfoType, 10)
			require.NoError(t, cc.getFileList(dir, ch))
			require.NotEmpty(t, ch)
		}
		require.NoError(t, os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken")))
		cc.cpOption.disableAllSymlink = false
		require.Error(t, cc.getFileListStatistic(dir))
	}
	cc.fileStatistic([]StorageURLer{FileURL{urlStr: filepath.Join(dir, "missing")}})
	require.Error(t, cc.monitor.seekAheadError)
	_, err := cc.adjustDestURLForDownload(FileURL{urlStr: filepath.Join(dir, "file") + "/"})
	require.Error(t, err)
}

func TestCopyRejectsInvalidCombinationsBeforeTransfer(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(source, []byte("data"), 0600))
	for _, tc := range []struct {
		name    string
		args    []string
		options OptionMapType
		want    string
	}{
		{"symlink modes", []string{source, "oss://bucket/key"}, OptionMapType{OptionEnableSymlinkDir: boolPointer(true), OptionDisableAllSymlink: boolPointer(true)}, "can't be both"},
		{"time order", []string{source, "oss://bucket/key"}, OptionMapType{OptionStartTime: stringPointer("20"), OptionEndTime: stringPointer("10")}, "larger than"},
		{"source encoding", []string{"%zz", "oss://bucket/key"}, OptionMapType{OptionEncodingType: stringPointer(URLEncodingType)}, "not url encoded"},
		{"destination encoding", []string{source, "oss://bucket/%zz"}, OptionMapType{OptionEncodingType: stringPointer(URLEncodingType)}, "not url encoded"},
		{"download tagging", []string{"oss://bucket/key", source}, OptionMapType{OptionTagging: stringPointer("team=cli")}, "No need to set tagging"},
		{"bad tagging", []string{source, "oss://bucket/key"}, OptionMapType{OptionTagging: stringPointer("a=b=c")}, "tagging value error"},
		{"download ACL", []string{"oss://bucket/key", source}, OptionMapType{OptionACL: stringPointer("private")}, "No need to set ACL"},
		{"invalid ACL", []string{source, "oss://bucket/key"}, OptionMapType{OptionACL: stringPointer("invalid")}, "invalid acl"},
		{"payer", []string{source, "oss://bucket/key"}, OptionMapType{OptionRequestPayer: stringPointer("invalid")}, "invalid request payer"},
		{"meta", []string{source, "oss://bucket/key"}, OptionMapType{OptionMeta: stringPointer("unsupported:value")}, "unsupported header"},
		{"snapshot", []string{source, "oss://bucket/key"}, OptionMapType{OptionSnapshotPath: &source}, "load snapshot error"},
		{"upload partition", []string{source, "oss://bucket/key"}, OptionMapType{OptionPartitionDownload: stringPointer("1:2")}, "doesn't support"},
		{"partition format", []string{"oss://bucket/key", source}, OptionMapType{OptionPartitionDownload: stringPointer("invalid")}, "parsar"},
		{"partition value", []string{"oss://bucket/key", source}, OptionMapType{OptionPartitionDownload: stringPointer("a:2")}, "parsar"},
		{"partition index", []string{"oss://bucket/key", source}, OptionMapType{OptionPartitionDownload: stringPointer("3:2")}, "parsar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := CopyCommand{command: dataCommand(t, "http://127.0.0.1", copyCommand.command, tc.args, tc.options)}
			require.ErrorContains(t, cmd.RunCommand(), tc.want)
		})
	}
	t.Run("checkpoint", func(t *testing.T) {
		cmd := CopyCommand{command: dataCommand(t, "http://127.0.0.1", copyCommand.command, []string{source, "oss://bucket/key"}, OptionMapType{OptionCheckpointDir: &source})}
		var pathErr *os.PathError
		require.ErrorAs(t, cmd.RunCommand(), &pathErr)
		require.Equal(t, "mkdir", pathErr.Op)
		require.Equal(t, source, pathErr.Path)
	})
}

func TestRestoreMetadataAndACLValidation(t *testing.T) {
	base := dataCommand(t, "http://127.0.0.1", restoreCommand.command, []string{"oss://bucket/key"}, nil)
	rc, sc := RestoreCommand{command: base}, SetMetaCommand{command: base}
	for _, tc := range []struct {
		cloud         CloudURL
		recursive     bool
		version, file string
	}{
		{CloudURL{}, false, "", ""},
		{CloudURL{bucket: "bucket"}, false, "", ""},
		{CloudURL{bucket: "bucket", object: "key"}, true, "", "objects"},
		{CloudURL{bucket: "bucket"}, true, "version", ""},
	} {
		require.Error(t, rc.checkOptions(tc.cloud, tc.recursive, true, tc.version, tc.file))
		require.Error(t, sc.checkOptions(tc.cloud, false, false, true, tc.recursive, "en", tc.version, tc.file))
	}
	require.Error(t, sc.checkOptions(CloudURL{bucket: "bucket", object: "key"}, true, true, true, false, "en", "", ""))
	for _, mode := range []setACLType{bucketACL, objectACL} {
		for _, acl := range []string{"private", "public-read", "public-read-write"} {
			got, err := base.checkACL(acl, mode)
			require.NoError(t, err)
			require.Equal(t, oss.ACLType(acl), got)
		}
		_, err := base.checkACL("invalid", mode)
		require.Error(t, err)
		ac := SetACLCommand{command: base}
		ac.command.args = []string{"oss://bucket/key"}
		withBucketInput(t, "private\n", func() {
			got, err := ac.getACL(mode, true)
			require.NoError(t, err)
			require.Equal(t, oss.ACLPrivate, got)
		})
		withBucketInput(t, "\n", func() { _, err := ac.getACL(mode, false); require.Error(t, err) })
	}
	for _, tc := range []struct {
		args    []string
		options OptionMapType
		want    string
	}{
		{[]string{"oss://bucket/key", "private"}, OptionMapType{OptionRecursion: boolPointer(true), OptionVersionId: stringPointer("v")}, "version-id"},
		{[]string{"local", "private"}, nil, "invalid cloud url"},
		{[]string{"oss://", "private"}, nil, "miss bucket"},
		{[]string{"oss://bucket", "private"}, nil, "object empty"},
		{[]string{"oss://bucket/key", "private"}, OptionMapType{OptionBucket: boolPointer(true)}, "object not empty"},
		{[]string{"oss://bucket", "private"}, OptionMapType{OptionBucket: boolPointer(true), OptionRecursion: boolPointer(true)}, "recursive"},
	} {
		cmd := SetACLCommand{command: dataCommand(t, "http://127.0.0.1", setACLCommand.command, tc.args, tc.options)}
		require.ErrorContains(t, cmd.RunCommand(), tc.want)
	}
}
