package lib

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestDataCommandsPropagatePermanentHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>local denial</Message></Error>`)
	}))
	defer server.Close()
	base := dataCommand(t, server.URL, copyCommand.command, []string{"oss://bucket/key"}, nil)
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	cloud := CloudURL{bucket: "bucket", object: "key"}
	rm := RemoveCommand{command: base}
	rm.monitor.init()
	rm.rmOption.force = true
	_, err = rm.ossIsObjectExistRetry(bucket, "key")
	require.Error(t, err)
	for _, operation := range []func() error{
		func() error { return rm.ossDeleteObjectRetry(bucket, "key") },
		func() error { _, err := rm.ossBatchDeleteObjectsRetry(bucket, []string{"key"}); return err },
		func() error { return rm.ossDeleteObjectRetryVersion(bucket, "key", "v1") },
		func() error {
			_, err := rm.ossBatchDeleteObjectsRetryVersion(bucket, []oss.DeleteObject{{Key: "key", VersionId: "v1"}})
			return err
		},
		func() error { return rm.ossAbortMultipartUploadRetry(bucket, "key", "upload") },
		func() error { return rm.ossDeleteBucketRetry(&bucket.Client, "bucket") },
		func() error { return rm.batchObjectStatistic(bucket, cloud) },
		func() error { return rm.batchObjectStatisticVersion(bucket, cloud) },
		func() error { return rm.multipartUploadsStatistic(bucket, cloud) },
		func() error { return rm.removeObjectAllVersion(bucket, cloud) },
		func() error { return rm.batchDeleteObjects(bucket, cloud) },
		func() error { return rm.batchDeleteObjectsVersion(bucket, cloud) },
		func() error { return rm.removeSpecialCharacterObjects(bucket, cloud) },
		func() error { return rm.removeMultipartUploadsEntry(bucket, cloud) },
		func() error { return rm.removeBucket(bucket, cloud) },
	} {
		require.Error(t, operation())
	}
	count, err := rm.ossBatchDeleteObjectsRetry(bucket, nil)
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = rm.ossBatchDeleteObjectsRetryVersion(bucket, nil)
	require.NoError(t, err)
	require.Zero(t, count)
	rc := RestoreCommand{command: base}
	for _, configured := range []bool{true, false} {
		rc.hasConfig, rc.configXml = configured, `<RestoreRequest><Days>1</Days></RestoreRequest>`
		require.Error(t, rc.ossRestoreObjectRetry(bucket, "key"))
	}
	sc := SetMetaCommand{command: base}
	require.Error(t, sc.setObjectMeta(bucket, "key", map[string]string{"x-oss-meta-team": "cli"}, true, false, false, "version"))
	require.Error(t, sc.ossSetObjectMetaRetry(bucket, "key"))
	acl := SetACLCommand{command: base}
	require.Error(t, acl.ossSetObjectACLRetry(bucket, "key", oss.ACLPrivate, "version"))
	require.Error(t, acl.ossSetBucketACLRetry(&bucket.Client, "bucket", oss.ACLPrivate))
	require.Error(t, acl.setBucketACL(&bucket.Client, CloudURL{bucket: "bucket"}, false))
	cc := CopyCommand{command: base}
	cc.cpOption.force = true
	cc.cpOption.cpDir = filepath.Join(t.TempDir(), "checkpoint")
	require.NoError(t, os.MkdirAll(cc.cpOption.cpDir, 0700))
	file := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
	for _, operation := range []func() error{
		func() error { return cc.ossPutObjectRetry(bucket, "key", "content") },
		func() error { return cc.ossUploadFileRetry(bucket, "key", file) },
		func() error { return cc.ossResumeUploadRetry(bucket, "key", file, 102400) },
		func() error { return cc.ossDownloadFileRetry(bucket, "key", filepath.Join(t.TempDir(), "download")) },
		func() error {
			return cc.ossResumeDownloadRetry(bucket, "key", filepath.Join(t.TempDir(), "download"), 7, 102400)
		},
		func() error { return cc.ossCopyObjectRetry(bucket, "key", "bucket", "dest") },
		func() error { return cc.ossResumeCopyRetry("bucket", "key", "bucket", "dest", 102400) },
	} {
		require.Error(t, operation())
	}
	_, err = base.ossGetObjectMetaRetry(bucket, "key")
	require.Error(t, err)
	var monitor Monitor
	base.objectStatistic(bucket, cloud, &monitor, nil)
	require.Error(t, monitor.seekAheadError)
	base.objectStatistic(bucket, cloud, nil, nil)
	objects, errs := make(chan string, 2), make(chan error, 1)
	base.objectProducer(bucket, cloud, objects, errs, nil)
	require.Error(t, <-errs)
	require.Empty(t, objects)
}

func TestBatchWaitersAndErrorClassification(t *testing.T) {
	cause := errors.New("local failure")
	for _, continued := range []bool{true, false} {
		for _, source := range []string{"list", "worker", "success"} {
			rc, sc, acl := RestoreCommand{}, SetMetaCommand{}, SetACLCommand{}
			rc.monitor.init("restore")
			sc.monitor.init("meta")
			acl.monitor.init("acl")
			rc.reOption.ctnu, sc.smOption.ctnu, acl.saOption.ctnu = continued, continued, continued
			for _, wait := range []func(<-chan error, <-chan error, int64) error{rc.waitRoutinueComplete, sc.waitRoutinueComplete, acl.waitRoutinueComplete} {
				workers, listing := make(chan error, 2), make(chan error, 1)
				if source == "list" {
					listing <- cause
				} else {
					if source == "worker" {
						workers <- cause
					}
					workers <- nil
					listing <- nil
				}
				err := wait(workers, listing, 1)
				if source == "list" || (source == "worker" && !continued) {
					require.ErrorIs(t, err, cause)
				} else {
					require.NoError(t, err)
				}
			}
		}
	}
	cmd := Command{}
	option := batchOptionType{ctnu: true}
	require.False(t, cmd.filterError(nil, &option))
	require.False(t, cmd.filterError(cause, &option))
	require.True(t, cmd.filterError(ObjectError{err: cause}, &option))
	cc := CopyCommand{}
	require.False(t, cc.filterError(nil))
	for _, err := range []error{FileError{err: cause}, ObjectError{err: cause}, BucketError{err: cause}, cause} {
		require.True(t, cc.filterError(err))
	}
	require.False(t, cc.filterError(CopyError{err: cause}))
	for _, code := range []string{"NoSuchBucket", "InvalidAccessKeyId", "SignatureDoesNotMatch", "AccessDenied", "RequestTimeTooSkewed", "InvalidBucketName"} {
		option.ctnu, cc.cpOption.ctnu = true, true
		err := ObjectError{err: oss.ServiceError{Code: code}}
		require.False(t, cmd.filterError(err, &option))
		require.False(t, cc.filterError(err))
		require.False(t, option.ctnu)
		require.False(t, cc.cpOption.ctnu)
	}
}
