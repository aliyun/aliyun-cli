package lib

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestRestoreMetadataRunConfigurationFailures(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("configuration"), 0600))
	for _, target := range []string{"local", "oss://"} {
		rc := RestoreCommand{command: dataCommand(t, "http://127.0.0.1", restoreCommand.command, []string{target}, nil)}
		sc := SetMetaCommand{command: dataCommand(t, "http://127.0.0.1", setMetaCommand.command, []string{target}, nil)}
		require.Error(t, rc.RunCommand())
		require.Error(t, sc.RunCommand())
	}
	for _, options := range []OptionMapType{{OptionSnapshotPath: &file}, {OptionObjectFile: stringPointer(filepath.Join(dir, "missing"))}} {
		options[OptionRecursion] = boolPointer(true)
		rc := RestoreCommand{command: dataCommand(t, "http://127.0.0.1", restoreCommand.command, []string{"oss://bucket"}, options)}
		sc := SetMetaCommand{command: dataCommand(t, "http://127.0.0.1", setMetaCommand.command, []string{"oss://bucket"}, options)}
		require.Error(t, rc.RunCommand())
		require.Error(t, sc.RunCommand())
	}
	rc := RestoreCommand{command: dataCommand(t, "http://127.0.0.1", restoreCommand.command, []string{"oss://bucket/key"}, OptionMapType{OptionRequestPayer: stringPointer("invalid")})}
	require.Error(t, rc.RunCommand())
	for _, target := range []string{filepath.Join(dir, "missing"), dir} {
		rc.command = dataCommand(t, "http://127.0.0.1", restoreCommand.command, []string{"oss://bucket/key", target}, nil)
		require.Error(t, rc.RunCommand())
	}
	base := dataCommand(t, "http://127.0.0.1", restoreCommand.command, []string{"oss://bucket", "private"}, OptionMapType{OptionOutputDir: &file})
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	rc = RestoreCommand{command: base}
	require.Error(t, rc.batchRestoreObjects(bucket, CloudURL{bucket: "bucket"}, true))
	sc := SetMetaCommand{command: base}
	sc.smOption.ctnu = true
	require.Error(t, sc.batchSetObjectMeta(bucket, CloudURL{bucket: "bucket"}, nil, false, false, true, 1))
	sc.hasObjFile = true
	require.Error(t, sc.batchSetObjectsMetaFromFile(bucket, CloudURL{bucket: "bucket"}, nil, false, false, true, 1))
	ac := SetACLCommand{command: base}
	require.Error(t, ac.batchSetObjectACL(bucket, CloudURL{bucket: "bucket"}, true, 1))
}

func TestRestoreAndMetadataSnapshotWriteFailures(t *testing.T) {
	server, _ := newDataServer(t)
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	db, err := leveldb.OpenFile(filepath.Join(t.TempDir(), "snapshot"), nil)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	rc := RestoreCommand{command: base}
	rc.reOption.snapshotPath, rc.reOption.snapshotldb = "snapshot", db
	require.ErrorContains(t, rc.ossRestoreObject(bucket, "key", "", true), "dump snapshot error")
	sc := SetMetaCommand{command: base}
	sc.smOption.snapshotPath, sc.smOption.snapshotldb = "snapshot", db
	require.ErrorContains(t, sc.setObjectMeta(bucket, "key", map[string]string{"x-oss-meta-team": "cli"}, false, false, true, "v1"), "dump snapshot error")
	sc.smOption.snapshotPath = ""
	require.NoError(t, sc.setObjectMeta(bucket, "key", map[string]string{"x-oss-meta-team": "old"}, true, false, false, "v1"))
	require.Equal(t, uint64(1), sc.skipCount)
	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			return
		}
		w.WriteHeader(403)
	}))
	defer denied.Close()
	sc.command, rc.command = localOSSCommand(t, denied.URL), localOSSCommand(t, denied.URL)
	bucket, err = sc.command.ossBucket("bucket")
	require.NoError(t, err)
	require.Error(t, sc.setObjectMeta(bucket, "key", nil, true, false, false, ""))
	sc.smOption.snapshotPath = "snapshot"
	require.Error(t, sc.setObjectMeta(bucket, "key", nil, false, false, true, ""))
	require.Error(t, rc.ossRestoreObject(bucket, "key", "", true))
	for _, continued := range []bool{false, true} {
		sc.smOption.ctnu, rc.reOption.ctnu = continued, continued
		for _, consume := range []func(<-chan string, chan<- error){
			func(objects <-chan string, errs chan<- error) {
				sc.setObjectMetaConsumer(bucket, nil, false, false, objects, errs)
			},
			func(objects <-chan string, errs chan<- error) {
				rc.restoreConsumer(bucket, CloudURL{bucket: "bucket"}, objects, errs)
			},
		} {
			objects, errs := make(chan string, 2), make(chan error, 3)
			objects <- "first"
			objects <- "second"
			close(objects)
			consume(objects, errs)
			require.Error(t, <-errs)
			if continued {
				require.Error(t, <-errs)
				require.NoError(t, <-errs)
			} else {
				require.Len(t, objects, 1)
			}
		}
	}
}

func TestDiskUsageValidationAndPartFailures(t *testing.T) {
	for _, tc := range []struct {
		target  string
		options OptionMapType
	}{
		{"local", nil}, {"oss://bucket", OptionMapType{OptionRequestPayer: stringPointer("invalid")}}, {"oss://bucket", OptionMapType{OptionBlockSize: stringPointer("invalid")}},
	} {
		cmd := DuCommand{command: dataCommand(t, "http://127.0.0.1", duSizeCommand.command, []string{tc.target}, tc.options)}
		require.Error(t, cmd.RunCommand())
	}
	for _, status := range []int{200, 403, 404} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if status != 200 {
				w.WriteHeader(status)
				return
			}
			if r.URL.Query().Get("part-number-marker") == "" || r.URL.Query().Get("part-number-marker") == "0" {
				io.WriteString(w, `<ListPartsResult><IsTruncated>true</IsTruncated><NextPartNumberMarker>1</NextPartNumberMarker><Part><PartNumber>1</PartNumber><Size>3</Size></Part></ListPartsResult>`)
			} else {
				io.WriteString(w, `<ListPartsResult/>`)
			}
		}))
		cmd := DuCommand{command: localOSSCommand(t, server.URL)}
		cmd.duOption.bucketName, cmd.duOption.payer = "bucket", "requester"
		bucket, err := cmd.command.ossBucket("bucket")
		require.NoError(t, err)
		err = cmd.statPartSize(bucket, MultiPartObject{objectName: "key", uploadId: "upload"})
		if status == 403 {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		if status == 200 {
			require.Equal(t, int64(3), cmd.duOption.sumPartSize)
		}
		if status == 403 {
			cmd.duOption.countTypeMap, cmd.duOption.sizeTypeMap = map[string]int64{}, map[string]int64{}
			require.Error(t, cmd.getAllObjectSize(bucket))
			require.Error(t, cmd.getAllObjectVersionsSize(bucket))
			objects, errs := make(chan MultiPartObject, 1), make(chan error, 1)
			objects <- MultiPartObject{objectName: "key", uploadId: "upload"}
			close(objects)
			require.Error(t, cmd.uploadIdConsumer(bucket, objects, errs))
			require.Error(t, <-errs)
			objects, errs = make(chan MultiPartObject, 1), make(chan error, 1)
			cmd.uploadIdProducer(bucket, objects, errs)
			require.Error(t, <-errs)
		}
		server.Close()
	}
	for _, fromList := range []bool{false, true} {
		cmd := DuCommand{}
		workers, listing := make(chan error, 1), make(chan error, 1)
		cause := errors.New("local failure")
		if fromList {
			listing <- cause
		} else {
			workers <- cause
		}
		require.ErrorIs(t, cmd.waitRoutinueComplete(workers, listing, 1), cause)
	}
}

func TestBucketXMLInputBoundaries(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	require.NoError(t, os.WriteFile(empty, nil, 0600))
	for _, path := range []string{"", filepath.Join(dir, "missing"), dir, empty} {
		args := []string{"oss://bucket"}
		if path != "" {
			args = append(args, path)
		}
		cname := BucketCnameCommand{command: Command{args: args}}
		replication := ReplicationCommand{command: Command{args: args}}
		require.Error(t, cname.PutBucketCnameWithCertificate())
		require.Error(t, replication.PutBucketReplication())
		require.Error(t, replication.PutBucketRTC())
	}
}
