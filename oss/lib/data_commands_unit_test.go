package lib

import (
	"encoding/xml"
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

func boolPointer(value bool) *bool { return &value }

func dataCommand(t *testing.T, endpoint string, original Command, args []string, options OptionMapType) Command {
	t.Helper()
	base := localOSSCommand(t, endpoint)
	original.options = base.options
	original.configOptions = OptionMapType{}
	original.args = args
	original.options[OptionForce] = boolPointer(true)
	original.options[OptionRoutines] = stringPointer("1")
	original.options[OptionOutputDir] = stringPointer(filepath.Join(t.TempDir(), "reports"))
	original.options[OptionCheckpointDir] = stringPointer(filepath.Join(t.TempDir(), "checkpoint"))
	for name, value := range options {
		original.options[name] = value
	}
	original.assembleOptions(nil)
	return original
}

func newDataServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	writes := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "7")
			w.Header().Set("ETag", `"test-etag"`)
			w.Header().Set("Last-Modified", "Thu, 01 Oct 2026 00:00:00 GMT")
			w.Header().Set("X-Oss-Storage-Class", "Archive")
			w.Header().Set("X-Oss-Meta-Team", "old")
			return
		}
		if r.Method == http.MethodDelete {
			writes.Add(1)
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodPost {
			writes.Add(1)
			switch {
			case q.Has("uploads"):
				io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>key</Key><UploadId>upload</UploadId></InitiateMultipartUploadResult>`)
			case q.Has("delete"):
				var request struct {
					Objects []struct {
						Key string `xml:"Key"`
					} `xml:"Object"`
				}
				require.NoError(t, xml.NewDecoder(r.Body).Decode(&request))
				require.NotEmpty(t, request.Objects)
				io.WriteString(w, `<DeleteResult/>`)
			case q.Has("restore"):
				w.WriteHeader(202)
			default:
				io.WriteString(w, `<CompleteMultipartUploadResult><ETag>test-etag</ETag></CompleteMultipartUploadResult>`)
			}
			return
		}
		if r.Method == http.MethodPut {
			writes.Add(1)
			if r.Header.Get("X-Oss-Copy-Source") != "" {
				if q.Has("uploadId") {
					io.WriteString(w, `<CopyPartResult><LastModified>2026-10-01T00:00:00.000Z</LastModified><ETag>test-etag</ETag></CopyPartResult>`)
				} else {
					io.WriteString(w, `<CopyObjectResult><LastModified>2026-10-01T00:00:00.000Z</LastModified><ETag>test-etag</ETag></CopyObjectResult>`)
				}
			} else {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NotNil(t, body)
				w.Header().Set("ETag", `"test-etag"`)
			}
			return
		}
		switch {
		case q.Has("acl"):
			io.WriteString(w, `<AccessControlPolicy><AccessControlList><Grant>private</Grant></AccessControlList></AccessControlPolicy>`)
		case q.Has("bucketInfo"):
			io.WriteString(w, `<BucketInfo><Bucket><Name>bucket</Name></Bucket></BucketInfo>`)
		case q.Has("uploadId"):
			io.WriteString(w, `<ListPartsResult><Part><PartNumber>1</PartNumber><Size>3</Size></Part></ListPartsResult>`)
		case q.Has("uploads"):
			if q.Get("marker") != "" && q.Get("key-marker") == "" {
				t.Error("multipart pagination requires key-marker")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if q.Get("key-marker") != "" {
				io.WriteString(w, `<ListMultipartUploadsResult/>`)
			} else {
				io.WriteString(w, `<ListMultipartUploadsResult><IsTruncated>true</IsTruncated><NextKeyMarker>next</NextKeyMarker><NextUploadIdMarker>next-upload</NextUploadIdMarker><Upload><Key>first</Key><UploadId>upload1</UploadId><Initiated>2026-10-01T00:00:00.000Z</Initiated></Upload></ListMultipartUploadsResult>`)
			}
		case q.Has("versions"):
			if q.Get("key-marker") != "" {
				io.WriteString(w, `<ListVersionsResult/>`)
			} else {
				io.WriteString(w, `<ListVersionsResult><IsTruncated>true</IsTruncated><NextKeyMarker>next</NextKeyMarker><NextVersionIdMarker>next-version</NextVersionIdMarker><Version><Key>first</Key><VersionId>v1</VersionId><IsLatest>true</IsLatest><LastModified>2026-10-01T00:00:00.000Z</LastModified><Type>Normal</Type><Size>7</Size><StorageClass>Archive</StorageClass></Version><DeleteMarker><Key>second</Key><VersionId>v2</VersionId><IsLatest>true</IsLatest><LastModified>2026-10-01T00:00:00.000Z</LastModified></DeleteMarker><CommonPrefixes><Prefix>directory/</Prefix></CommonPrefixes></ListVersionsResult>`)
			}
		case r.URL.Path == "/":
			io.WriteString(w, `<ListAllMyBucketsResult><Buckets><Bucket><Name>bucket</Name><Location>cn-test</Location><CreationDate>2026-10-01T00:00:00.000Z</CreationDate><StorageClass>Standard</StorageClass></Bucket></Buckets></ListAllMyBucketsResult>`)
		case strings.TrimSuffix(r.URL.Path, "/") == "/bucket":
			if q.Get("marker") != "" {
				io.WriteString(w, `<ListBucketResult/>`)
			} else {
				io.WriteString(w, `<ListBucketResult><IsTruncated>true</IsTruncated><NextMarker>next</NextMarker><Contents><Key>first</Key><LastModified>2026-10-01T00:00:00.000Z</LastModified><Type>Normal</Type><Size>7</Size><StorageClass>Archive</StorageClass></Contents><Contents><Key>second</Key><LastModified>2026-10-01T00:00:00.000Z</LastModified><Type>Normal</Type><Size>7</Size><StorageClass>Archive</StorageClass></Contents><CommonPrefixes><Prefix>directory/</Prefix></CommonPrefixes></ListBucketResult>`)
			}
		default:
			io.WriteString(w, "content")
		}
	}))
	return server, writes
}

func TestListAndDiskUsageLocalResults(t *testing.T) {
	server, _ := newDataServer(t)
	defer server.Close()
	for _, options := range []OptionMapType{
		{}, {OptionShortFormat: boolPointer(true)}, {OptionDirectory: boolPointer(true)},
		{OptionAllversions: boolPointer(true)}, {OptionAllversions: boolPointer(true), OptionDirectory: boolPointer(true)},
		{OptionMultipart: boolPointer(true)}, {OptionMultipart: boolPointer(true), OptionShortFormat: boolPointer(true)},
		{OptionAllType: boolPointer(true), OptionLimitedNum: stringPointer("1")},
	} {
		cmd := ListCommand{command: dataCommand(t, server.URL, listCommand.command, []string{"oss://bucket"}, options)}
		out := captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
		require.Contains(t, out, "first")
	}
	for _, short := range []bool{false, true} {
		cmd := ListCommand{command: dataCommand(t, server.URL, listCommand.command, nil, OptionMapType{OptionShortFormat: &short})}
		out := captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
		require.Contains(t, out, "oss://bucket")
		require.Contains(t, out, "Bucket Number is: 1")
	}
	for _, versions := range []bool{false, true} {
		for _, unit := range []string{"", "KB"} {
			cmd := DuCommand{command: dataCommand(t, server.URL, duSizeCommand.command, []string{"oss://bucket"}, OptionMapType{OptionAllversions: &versions, OptionBlockSize: &unit, OptionRequestPayer: stringPointer("requester")})}
			out := captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
			require.Contains(t, out, "total du size")
			require.Equal(t, int64(3), cmd.duOption.sumPartSize)
			require.Equal(t, int64(1), cmd.duOption.totalPartCount)
			if versions {
				require.Equal(t, int64(7), cmd.duOption.sumObjectSize)
			} else {
				require.Equal(t, int64(14), cmd.duOption.sumObjectSize)
			}
		}
	}
}

func TestCopyUploadsDownloadsAndCopiesLocally(t *testing.T) {
	server, writes := newDataServer(t)
	defer server.Close()
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(source, []byte("content"), 0600))
	for _, tc := range []struct {
		name    string
		args    []string
		options OptionMapType
	}{
		{"upload", []string{source, "oss://bucket/key"}, OptionMapType{OptionMeta: stringPointer("x-oss-meta-team:cli"), OptionTagging: stringPointer("team#cli"), OptionACL: stringPointer("private"), OptionRequestPayer: stringPointer("requester")}},
		{"download", []string{"oss://bucket/key", filepath.Join(dir, "download")}, nil},
		{"copy", []string{"oss://bucket/key", "oss://bucket/copy"}, nil},
	} {
		for _, threshold := range []string{"1", "9999999"} {
			t.Run(tc.name+"/"+threshold, func(t *testing.T) {
				options := OptionMapType{OptionBigFileThreshold: &threshold}
				for name, value := range tc.options {
					options[name] = value
				}
				cmd := CopyCommand{command: dataCommand(t, server.URL, copyCommand.command, tc.args, options)}
				out := captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
				require.Contains(t, out, "Succeed")
				require.Equal(t, int64(1), cmd.monitor.fileNum)
				if tc.name != "upload" && threshold == "9999999" {
					require.Equal(t, int64(7), cmd.monitor.dealSize)
				}
				if tc.name == "download" {
					body, err := os.ReadFile(tc.args[1])
					require.NoError(t, err)
					require.Equal(t, "content", string(body))
				}
			})
		}
	}
	require.GreaterOrEqual(t, writes.Load(), int32(2))
	for _, args := range [][]string{{dir, "oss://bucket/prefix/"}, {"oss://bucket", filepath.Join(dir, "many")}, {"oss://bucket", "oss://bucket2/prefix"}} {
		cmd := CopyCommand{command: dataCommand(t, server.URL, copyCommand.command, args, OptionMapType{OptionRecursion: boolPointer(true)})}
		captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
		require.Positive(t, cmd.monitor.fileNum)
	}
}

func TestRemoveLocalObjectsVersionsAndUploads(t *testing.T) {
	server, writes := newDataServer(t)
	defer server.Close()
	for _, tc := range []struct {
		target  string
		options OptionMapType
	}{
		{"oss://bucket/first", nil},
		{"oss://bucket/first", OptionMapType{OptionVersionId: stringPointer("v1")}},
		{"oss://bucket/first", OptionMapType{OptionAllversions: boolPointer(true)}},
		{"oss://bucket", OptionMapType{OptionRecursion: boolPointer(true)}},
		{"oss://bucket", OptionMapType{OptionRecursion: boolPointer(true), OptionAllversions: boolPointer(true)}},
		{"oss://bucket", OptionMapType{OptionRecursion: boolPointer(true), OptionAllType: boolPointer(true), OptionBucket: boolPointer(true)}},
		{"oss://bucket", OptionMapType{OptionBucket: boolPointer(true)}},
	} {
		cmd := RemoveCommand{command: dataCommand(t, server.URL, removeCommand.command, []string{tc.target}, tc.options)}
		before := writes.Load()
		captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
		require.Greater(t, writes.Load(), before)
	}
}

func TestRestoreSetMetaAndACLAgainstLocalServer(t *testing.T) {
	server, writes := newDataServer(t)
	defer server.Close()
	for _, version := range []string{"", "v1"} {
		cmd := RestoreCommand{command: dataCommand(t, server.URL, restoreCommand.command, []string{"oss://bucket/first"}, OptionMapType{OptionVersionId: &version})}
		require.NoError(t, cmd.RunCommand())
	}
	for _, options := range []OptionMapType{{}, {OptionUpdate: boolPointer(true)}, {OptionDelete: boolPointer(true)}} {
		header := "x-oss-meta-team:cli"
		if _, deleting := options[OptionDelete]; deleting {
			header = "x-oss-meta-team"
		}
		cmd := SetMetaCommand{command: dataCommand(t, server.URL, setMetaCommand.command, []string{"oss://bucket/first", header}, options)}
		require.NoError(t, cmd.RunCommand())
	}
	for _, bucket := range []bool{false, true} {
		target := "oss://bucket/first"
		if bucket {
			target = "oss://bucket"
		}
		cmd := SetACLCommand{command: dataCommand(t, server.URL, setACLCommand.command, []string{target, "public-read"}, OptionMapType{OptionBucket: &bucket})}
		require.NoError(t, cmd.RunCommand())
	}
	require.GreaterOrEqual(t, writes.Load(), int32(7))
	base := dataCommand(t, server.URL, restoreCommand.command, nil, nil)
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	rc := RestoreCommand{command: base}
	rc.monitor.init("Restore")
	objects, errs := make(chan string, 2), make(chan error, 4)
	objects <- "first"
	objects <- "second"
	close(objects)
	rc.restoreConsumer(bucket, CloudURL{bucket: "bucket"}, objects, errs)
	require.NoError(t, <-errs)
	require.Equal(t, int64(2), rc.monitor.okNum)
	sc := SetMetaCommand{command: base}
	sc.monitor.init("Set meta")
	objects = make(chan string, 1)
	objects <- "first"
	close(objects)
	sc.setObjectMetaConsumer(bucket, map[string]string{"x-oss-meta-team": "cli"}, false, false, objects, errs)
	require.NoError(t, <-errs)
	require.Equal(t, int64(1), sc.monitor.okNum)
	acl := SetACLCommand{command: base}
	acl.monitor.init("Set ACL")
	objects = make(chan string, 1)
	objects <- "first"
	close(objects)
	acl.setObjectACLConsumer(bucket, oss.ACLPrivate, objects, errs)
	require.NoError(t, <-errs)
	require.Equal(t, int64(1), acl.monitor.okNum)
}

func TestDataCommandInputAndHeaderBoundaries(t *testing.T) {
	base := localOSSCommand(t, "http://127.0.0.1")
	for _, text := range []string{"unsupported:value", "content-type:value#broken"} {
		_, err := base.parseHeaders(text, false)
		require.Error(t, err)
	}
	headers, err := base.parseHeaders("Content-Type:text/plain#x-oss-meta-team:cli", false)
	require.NoError(t, err)
	require.Equal(t, "text/plain", headers["Content-Type"])
	_, err = base.getOSSOptions(headerOptionMap, headers)
	require.NoError(t, err)
	_, err = base.getOSSOptions(headerOptionMap, map[string]string{"unknown": "value"})
	require.Error(t, err)
	_, err = fetchHeaderOptionMap(headerOptionMap, "unknown")
	require.Error(t, err)
	sc := SetMetaCommand{}
	props := http.Header{"X-Oss-Meta-Team": {"old"}, "Content-Type": {"text/plain"}, "Unrelated": {"ignored"}}
	props.Set(StatACL, "private")
	merged, skipped := sc.mergeHeader(props, map[string]string{"x-oss-meta-team": "old"}, true, false)
	require.True(t, skipped)
	require.Equal(t, "old", merged["x-oss-meta-team"])
	merged, skipped = sc.mergeHeader(props, map[string]string{"x-oss-meta-team": "new"}, true, false)
	require.False(t, skipped)
	require.Equal(t, "new", merged["x-oss-meta-team"])
	merged, _ = sc.mergeHeader(props, map[string]string{"x-oss-meta-team": ""}, false, true)
	require.NotContains(t, merged, "x-oss-meta-team")
	require.NotContains(t, merged, "unrelated")
	require.Equal(t, "private", merged[strings.ToLower(oss.HTTPHeaderOssObjectACL)])
}

func TestBatchRestoreMetadataAndACLWithLocalSnapshots(t *testing.T) {
	server, _ := newDataServer(t)
	defer server.Close()
	objects := filepath.Join(t.TempDir(), "objects")
	require.NoError(t, os.WriteFile(objects, []byte("first\nsecond\n"), 0600))
	for _, fromFile := range []bool{false, true} {
		for _, name := range []string{"restore", "meta"} {
			snapshot := filepath.Join(t.TempDir(), "snapshot")
			for attempt := 0; attempt < 2; attempt++ {
				options := OptionMapType{OptionRecursion: boolPointer(true), OptionSnapshotPath: &snapshot}
				if fromFile {
					options[OptionObjectFile] = &objects
				}
				if name == "restore" {
					cmd := RestoreCommand{command: dataCommand(t, server.URL, restoreCommand.command, []string{"oss://bucket"}, options)}
					captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
					require.Equal(t, int64(2), cmd.monitor.okNum)
					if attempt > 0 {
						require.Equal(t, int64(2), cmd.monitor.skipNum)
					}
				} else {
					cmd := SetMetaCommand{command: dataCommand(t, server.URL, setMetaCommand.command, []string{"oss://bucket", "x-oss-meta-team:cli"}, options)}
					captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
					require.Equal(t, int64(2), cmd.monitor.okNum)
					if attempt > 0 {
						require.Equal(t, int64(2), cmd.monitor.skipNum)
					}
				}
			}
		}
	}
	cmd := SetACLCommand{command: dataCommand(t, server.URL, setACLCommand.command, []string{"oss://bucket", "private"}, OptionMapType{OptionRecursion: boolPointer(true)})}
	captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
	require.Equal(t, int64(2), cmd.monitor.okNum)
}
