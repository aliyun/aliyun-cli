package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveRejectsInvalidTargetsAndFiltersBeforeRequests(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	for _, tc := range []struct {
		name, target, want string
		options            OptionMapType
		filter             string
	}{
		{name: "local path", target: "local-file", want: "oss://"},
		{name: "missing bucket", target: "oss:///key", want: "miss bucket"},
		{name: "invalid bucket", target: "oss://INVALID/key", want: "bucket"},
		{name: "invalid payer", target: "oss://bucket/key", options: OptionMapType{OptionRequestPayer: stringPointer("owner")}, want: "invalid request payer"},
		{name: "version with recursion", target: "oss://bucket/key", options: OptionMapType{OptionVersionId: stringPointer("v1"), OptionRecursion: boolPointer(true)}, want: "version-id"},
		{name: "filter directory", target: "oss://bucket/key", filter: "dir/*.txt", want: "dir info"},
		{name: "filter without recursion", target: "oss://bucket/key", filter: "*.txt", want: "only work with --recursive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = []string{"ossutil", "rm", tc.target}
			if tc.filter != "" {
				os.Args = append(os.Args, "--include", tc.filter)
			}
			cmd := RemoveCommand{command: dataCommand(t, server.URL, removeCommand.command, []string{tc.target}, tc.options)}
			require.ErrorContains(t, cmd.RunCommand(), tc.want)
		})
	}
	require.Zero(t, calls.Load())
}

func TestRemoveMissingObjectDoesNotSendDeleteAndPreservesRequestPayer(t *testing.T) {
	var heads, deletes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "requester", r.Header.Get("X-Oss-Request-Payer"))
		if r.Method == http.MethodHead {
			heads.Add(1)
			w.WriteHeader(http.StatusNotFound)
		} else {
			deletes.Add(1)
		}
	}))
	defer server.Close()
	cmd := RemoveCommand{command: dataCommand(t, server.URL, removeCommand.command, []string{"oss://bucket/key"}, OptionMapType{OptionRequestPayer: stringPointer("requester")})}
	require.NoError(t, cmd.RunCommand())
	require.EqualValues(t, 1, heads.Load())
	require.Zero(t, deletes.Load())
	require.Zero(t, cmd.monitor.objectNum)
}

func TestRemoveReportsDeleteFailuresAndTracksFailedObjects(t *testing.T) {
	for _, mode := range []string{"single", "version", "batch", "all versions", "batch versions", "multipart", "special characters"} {
		t.Run(mode, func(t *testing.T) {
			var listCalls, mutations atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if r.Method == http.MethodGet {
					listCalls.Add(1)
					switch {
					case r.URL.Query().Has("uploads"):
						io.WriteString(w, `<ListMultipartUploadsResult><Upload><Key>key</Key><UploadId>upload</UploadId></Upload></ListMultipartUploadsResult>`)
					case r.URL.Query().Has("versions"):
						io.WriteString(w, `<ListVersionsResult><Version><Key>key</Key><VersionId>v1</VersionId></Version></ListVersionsResult>`)
					case mode == "special characters" && listCalls.Load() == 1:
						io.WriteString(w, `<ListBucketResult/>`)
					default:
						io.WriteString(w, `<ListBucketResult><Contents><Key>key</Key></Contents></ListBucketResult>`)
					}
					return
				}
				mutations.Add(1)
				w.WriteHeader(http.StatusForbidden)
				io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>deletion denied</Message></Error>`)
			}))
			defer server.Close()
			cmd := RemoveCommand{command: localOSSCommand(t, server.URL)}
			cmd.monitor.init()
			cmd.rmOption.typeSet = objectType
			switch mode {
			case "version":
				cmd.rmOption.versionId = "v1"
			case "batch", "special characters":
				cmd.rmOption.recursive = true
			case "all versions":
				cmd.rmOption.allVersions = true
			case "batch versions":
				cmd.rmOption.recursive, cmd.rmOption.allVersions = true, true
			case "multipart":
				cmd.rmOption.typeSet = multipartType
			}
			bucket, err := cmd.command.ossBucket("bucket")
			require.NoError(t, err)
			err = cmd.removeEntry(bucket, CloudURL{bucket: "bucket", object: "key"})
			require.ErrorContains(t, err, "AccessDenied")
			require.EqualValues(t, 1, mutations.Load())
			if mode == "multipart" {
				require.Equal(t, int64(1), cmd.monitor.errUploadIdNum)
			} else if mode != "special characters" {
				require.Equal(t, int64(1), cmd.monitor.errObjectNum)
			}
			if mode == "single" || mode == "version" {
				require.Zero(t, cmd.monitor.getOP())
			}
		})
	}
}

func TestRemoveHonorsBucketCancellationAndAlreadyAbortedUploads(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `<Error><Code>NoSuchUpload</Code><Message>already aborted</Message></Error>`)
	}))
	defer server.Close()
	cmd := RemoveCommand{command: localOSSCommand(t, server.URL)}
	cmd.monitor.init()
	bucket, err := cmd.command.ossBucket("bucket")
	require.NoError(t, err)
	withBucketInput(t, "n\n", func() { require.NoError(t, cmd.removeBucket(bucket, CloudURL{bucket: "bucket"})) })
	require.Zero(t, calls.Load())
	cmd.command.args = []string{"oss://bucket/key"}
	cmd.rmOption.recursive, cmd.rmOption.typeSet = true, allType
	withBucketInput(t, "y\n", func() { require.True(t, cmd.confirmRemoveObject(CloudURL{bucket: "bucket", object: "key"})) })
	require.NoError(t, cmd.ossAbortMultipartUploadRetry(bucket, "key", "upload"))
	require.EqualValues(t, 1, calls.Load())
}

func TestRemoveVersionStatisticsAndUploadSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("uploads") {
			io.WriteString(w, `<ListMultipartUploadsResult><Upload><Key>key</Key><UploadId>upload</UploadId></Upload><Upload><Key>key-other</Key><UploadId>other</UploadId></Upload></ListMultipartUploadsResult>`)
		} else {
			io.WriteString(w, `<ListVersionsResult><DeleteMarker><Key>key</Key><VersionId>marker</VersionId></DeleteMarker><DeleteMarker><Key>skip</Key><VersionId>skip</VersionId></DeleteMarker></ListVersionsResult>`)
		}
	}))
	defer server.Close()
	cmd := RemoveCommand{command: localOSSCommand(t, server.URL)}
	cmd.monitor.init()
	cmd.filters = []filterOptionType{{IncludePrompt, "key"}}
	bucket, err := cmd.command.ossBucket("bucket")
	require.NoError(t, err)
	require.NoError(t, cmd.batchObjectStatisticVersion(bucket, CloudURL{bucket: "bucket"}))
	require.Equal(t, int64(1), cmd.monitor.totalObjectNum)
	cmd.monitor.init()
	require.NoError(t, cmd.multipartUploadsStatistic(bucket, CloudURL{bucket: "bucket", object: "key"}))
	require.Equal(t, int64(1), cmd.monitor.totalUploadIdNum)
	uploads, listed := make(chan uploadIdInfoType, 4), make(chan error, 1)
	cmd.multipartUploadsProducer(bucket, CloudURL{bucket: "bucket", object: "key"}, uploads, listed)
	require.NoError(t, <-listed)
	var keys []string
	for upload := range uploads {
		keys = append(keys, upload.key)
	}
	require.Equal(t, []string{"key"}, keys)
}
