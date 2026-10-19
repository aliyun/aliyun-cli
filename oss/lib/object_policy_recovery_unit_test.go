package lib

import (
	"encoding/xml"
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

func TestObjectPolicyValidatesFiltersBucketsAndRevertRangesBeforeRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	for _, tc := range []struct {
		name, command, target, filter, want string
		options                             OptionMapType
	}{
		{name: "ACL directory filter", command: "acl", filter: "dir/*.txt", want: "dir info"},
		{name: "ACL filter without recursion", command: "acl", filter: "*.txt", want: "only work with --recursive"},
		{name: "metadata directory filter", command: "meta", filter: "dir/*.txt", want: "dir info"},
		{name: "metadata filter without recursion", command: "meta", filter: "*.txt", want: "only work with --recursive"},
		{name: "revert directory filter", command: "revert", filter: "dir/*.txt", want: "dir info"},
		{name: "revert inverted time range", command: "revert", options: OptionMapType{OptionStartTime: stringPointer("20"), OptionEndTime: stringPointer("10")}, want: "start time 20 is larger than end time 10"},
		{name: "revert invalid payer", command: "revert", options: OptionMapType{OptionRequestPayer: stringPointer("owner")}, want: "payer value must be requester or bucketowner"},
		{name: "ACL invalid bucket", command: "acl", target: "oss://INVALID/key", want: "bucket"},
		{name: "metadata invalid bucket", command: "meta", target: "oss://INVALID/key", want: "bucket"},
		{name: "restore invalid bucket", command: "restore", target: "oss://INVALID/key", want: "bucket"},
		{name: "revert invalid bucket", command: "revert", target: "oss://INVALID/key", want: "bucket"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.target
			if target == "" {
				target = "oss://bucket/key"
			}
			os.Args = []string{"ossutil", target}
			if tc.filter != "" {
				os.Args = append(os.Args, "--include", tc.filter)
			}
			var err error
			switch tc.command {
			case "acl":
				cmd := SetACLCommand{command: dataCommand(t, server.URL, setACLCommand.command, []string{target, "private"}, tc.options)}
				err = cmd.RunCommand()
			case "meta":
				cmd := SetMetaCommand{command: dataCommand(t, server.URL, setMetaCommand.command, []string{target, "x-oss-meta-team:cli"}, tc.options)}
				err = cmd.RunCommand()
			case "restore":
				cmd := RestoreCommand{command: dataCommand(t, server.URL, restoreCommand.command, []string{target}, tc.options)}
				err = cmd.RunCommand()
			case "revert":
				cmd := RevertCommand{command: dataCommand(t, server.URL, revertCommand.command, []string{target}, tc.options)}
				err = cmd.RunCommand()
			}
			require.ErrorContains(t, err, tc.want)
			require.Zero(t, requests.Load())
		})
	}
}

func TestObjectPolicyACLRejectsInvalidPermissionsAndHonorsCancellation(t *testing.T) {
	var reads, writes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Has("bucketInfo") {
			reads.Add(1)
			io.WriteString(w, `<BucketInfo><Bucket><Name>bucket</Name></Bucket></BucketInfo>`)
			return
		}
		writes.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	for _, mode := range []string{"bucket", "object", "batch", "confirmed batch"} {
		t.Run(mode, func(t *testing.T) {
			target := "oss://bucket/key"
			options := OptionMapType{}
			if mode == "bucket" {
				target, options[OptionBucket] = "oss://bucket", boolPointer(true)
			}
			if strings.Contains(mode, "batch") {
				options[OptionRecursion] = boolPointer(true)
			}
			if mode == "confirmed batch" {
				options[OptionForce] = boolPointer(false)
			}
			cmd := SetACLCommand{command: dataCommand(t, server.URL, setACLCommand.command, []string{target, "invalid"}, options)}
			withBucketInput(t, "Y\n", func() { require.ErrorContains(t, cmd.RunCommand(), "invalid acl: invalid") })
		})
	}
	cmd := SetACLCommand{command: dataCommand(t, server.URL, setACLCommand.command, []string{"oss://bucket", "private"}, OptionMapType{OptionRecursion: boolPointer(true), OptionForce: boolPointer(false)})}
	output := captureBucketOutput(t, func() {
		withBucketInput(t, "n\n", func() { require.NoError(t, cmd.RunCommand()) })
	})
	require.Contains(t, output, "operation is canceled")
	require.EqualValues(t, 1, reads.Load())
	require.Zero(t, writes.Load())
}

func TestObjectPolicyACLConsumerReportsFailuresAndHonorsContinueMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !r.URL.Query().Has("acl") {
			t.Errorf("unexpected ACL request: %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>object disappeared</Message></Error>`)
	}))
	defer server.Close()
	for _, continued := range []bool{false, true} {
		t.Run(fmt.Sprint(continued), func(t *testing.T) {
			cmd := SetACLCommand{command: dataCommand(t, server.URL, setACLCommand.command, nil, nil)}
			bucket, err := cmd.command.ossBucket("bucket")
			require.NoError(t, err)
			cmd.monitor.init("Set ACL")
			cmd.saOption.ctnu = continued
			cmd.saOption.reporter, err = GetReporter(true, t.TempDir(), "set ACL")
			require.NoError(t, err)
			t.Cleanup(cmd.saOption.reporter.Clear)
			objects, failures := make(chan string, 2), make(chan error, 3)
			objects <- "first"
			objects <- "second"
			close(objects)
			cmd.setObjectACLConsumer(bucket, oss.ACLPrivate, objects, failures)
			require.ErrorContains(t, <-failures, "NoSuchKey")
			if continued {
				require.ErrorContains(t, <-failures, "NoSuchKey")
				require.NoError(t, <-failures)
				require.Empty(t, objects)
				require.Equal(t, int64(2), cmd.monitor.errNum)
			} else {
				require.Len(t, objects, 1)
				require.Empty(t, failures)
				require.Equal(t, int64(1), cmd.monitor.errNum)
			}
			cmd.saOption.reporter.Clear()
			report, err := os.ReadFile(cmd.saOption.reporter.path)
			require.NoError(t, err)
			require.Contains(t, string(report), "set acl on oss://bucket/first")
			require.Contains(t, string(report), "NoSuchKey")
		})
	}
}

func TestObjectPolicyMetadataRejectsCanceledInputAndMalformedHeaders(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()
	for _, tc := range []struct {
		name, header, want string
		options            OptionMapType
	}{
		{name: "unsupported header", header: "Unsupported:value", want: "unsupported header:Unsupported"},
		{name: "invalid expiration", header: "Expires:not-a-time", want: "cannot parse"},
		{name: "delete with value", header: "x-oss-meta-team:cli", options: OptionMapType{OptionDelete: boolPointer(true)}, want: "do no support value"},
		{name: "missing metadata confirmation", options: OptionMapType{OptionForce: boolPointer(false)}, want: "unknown input, operation is canceled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"oss://bucket/key"}
			if tc.header != "" {
				args = append(args, tc.header)
			}
			cmd := SetMetaCommand{command: dataCommand(t, server.URL, setMetaCommand.command, args, tc.options)}
			withBucketInput(t, "", func() { require.ErrorContains(t, cmd.RunCommand(), tc.want) })
			require.Zero(t, requests.Load())
		})
	}
}

func TestObjectPolicyRestoreKeepsRequesterVersionAndAcceptsAlreadyInProgress(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(fmt.Sprint(configured), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || !r.URL.Query().Has("restore") || r.URL.Query().Get("versionId") != "version-1" || r.Header.Get("X-Oss-Request-Payer") != "requester" {
					t.Errorf("unexpected restore request: %s %s, payer=%s", r.Method, r.URL, r.Header.Get("X-Oss-Request-Payer"))
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if configured && !strings.Contains(string(body), "<Days>1</Days>") {
					t.Errorf("restore configuration missing from body: %s", body)
				}
				w.WriteHeader(http.StatusConflict)
				io.WriteString(w, `<Error><Code>RestoreAlreadyInProgress</Code><Message>restore is pending</Message></Error>`)
			}))
			defer server.Close()
			args := []string{"oss://bucket/key"}
			if configured {
				path := filepath.Join(t.TempDir(), "restore.xml")
				require.NoError(t, os.WriteFile(path, []byte(`<RestoreRequest><Days>1</Days></RestoreRequest>`), 0600))
				args = append(args, path)
			}
			cmd := RestoreCommand{command: dataCommand(t, server.URL, restoreCommand.command, args, OptionMapType{OptionRequestPayer: stringPointer("requester"), OptionVersionId: stringPointer("version-1")})}
			require.NoError(t, cmd.RunCommand())
			require.EqualValues(t, 1, requests.Load())
		})
	}
}

func TestObjectPolicyRestoreConfirmationHasNoBackgroundSideEffects(t *testing.T) {
	cmd := RestoreCommand{command: dataCommand(t, "http://127.0.0.1", restoreCommand.command, []string{"oss://bucket/key"}, nil)}
	output := captureBucketOutput(t, func() {
		require.NoError(t, cmd.checkOptions(CloudURL{bucket: "bucket", object: "key"}, false, false, "", ""))
	})
	require.Empty(t, output)
	for _, input := range []string{"Y\n", "n\n"} {
		output = captureBucketOutput(t, func() {
			withBucketInput(t, input, func() {
				require.NoError(t, cmd.checkOptions(CloudURL{bucket: "bucket"}, true, false, "", ""))
			})
		})
		require.Contains(t, output, "recursivlly restore objects")
		if input == "n\n" {
			require.Contains(t, output, "operation is canceled")
		} else {
			require.NotContains(t, output, "operation is canceled")
		}
	}
}

func TestObjectPolicyRevertStopsAtOtherKeysAndSurfacesDeleteFailures(t *testing.T) {
	for _, mode := range []string{"stop at other key", "delete denied"} {
		t.Run(mode, func(t *testing.T) {
			var lists, deletes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					lists.Add(1)
					if mode == "stop at other key" {
						io.WriteString(w, `<ListVersionsResult><IsTruncated>true</IsTruncated><NextKeyMarker>different</NextKeyMarker><DeleteMarker><Key>different</Key><VersionId>other</VersionId><IsLatest>true</IsLatest></DeleteMarker></ListVersionsResult>`)
					} else {
						io.WriteString(w, `<ListVersionsResult><DeleteMarker><Key>key</Key><VersionId>marker</VersionId><IsLatest>true</IsLatest></DeleteMarker></ListVersionsResult>`)
					}
					return
				}
				deletes.Add(1)
				var payload struct {
					Quiet   bool                              `xml:"Quiet"`
					Objects []struct{ Key, VersionId string } `xml:"Object"`
				}
				if err := xml.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				} else if !payload.Quiet || len(payload.Objects) != 1 || payload.Objects[0].Key != "key" || payload.Objects[0].VersionId != "marker" {
					t.Errorf("unexpected delete request: %+v", payload)
				}
				w.WriteHeader(http.StatusForbidden)
				io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>delete marker denied</Message></Error>`)
			}))
			defer server.Close()
			cmd := RevertCommand{command: dataCommand(t, server.URL, revertCommand.command, []string{"oss://bucket/key"}, nil)}
			captureBucketOutput(t, func() {
				err := cmd.RunCommand()
				if mode == "stop at other key" {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, "AccessDenied")
				}
			})
			require.EqualValues(t, 1, lists.Load())
			require.Zero(t, cmd.revertOption.revertCount)
			if mode == "stop at other key" {
				require.Zero(t, deletes.Load())
			} else {
				require.EqualValues(t, 1, deletes.Load())
			}
		})
	}
}

func TestObjectPolicyRevertPaginationKeepsFiltersAndRequestPayer(t *testing.T) {
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	os.Args = []string{"ossutil", "revert-versioning", "oss://bucket/prefix/", "--include", "*.txt"}
	var lists, deletes atomic.Int32
	deleted := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Oss-Request-Payer") != "bucketowner" {
			t.Errorf("request payer was lost: %s", r.Header.Get("X-Oss-Request-Payer"))
		}
		if r.Method == http.MethodGet {
			page := lists.Add(1)
			if r.URL.Query().Get("prefix") != "prefix/" {
				t.Errorf("unexpected prefix: %s", r.URL)
			}
			if page == 1 {
				io.WriteString(w, `<ListVersionsResult><IsTruncated>true</IsTruncated><NextKeyMarker>cursor</NextKeyMarker><NextVersionIdMarker>cursor-version</NextVersionIdMarker><DeleteMarker><Key>prefix/first.txt</Key><VersionId>one</VersionId><IsLatest>true</IsLatest></DeleteMarker><DeleteMarker><Key>prefix/ignored.log</Key><VersionId>skip</VersionId><IsLatest>true</IsLatest></DeleteMarker></ListVersionsResult>`)
			} else {
				if r.URL.Query().Get("key-marker") != "cursor" || r.URL.Query().Get("version-id-marker") != "cursor-version" {
					t.Errorf("pagination markers were lost: %s", r.URL)
				}
				io.WriteString(w, `<ListVersionsResult><DeleteMarker><Key>prefix/second.txt</Key><VersionId>two</VersionId><IsLatest>true</IsLatest></DeleteMarker></ListVersionsResult>`)
			}
			return
		}
		deletes.Add(1)
		var payload struct {
			Objects []struct{ Key string } `xml:"Object"`
		}
		if err := xml.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload.Objects) != 1 {
			t.Errorf("unexpected delete request: %+v, %v", payload, err)
		} else {
			deleted <- payload.Objects[0].Key
		}
		io.WriteString(w, `<DeleteResult/>`)
	}))
	defer server.Close()
	cmd := RevertCommand{command: dataCommand(t, server.URL, revertCommand.command, []string{"oss://bucket/prefix/"}, OptionMapType{OptionRecursion: boolPointer(true), OptionRequestPayer: stringPointer("bucketowner")})}
	require.NoError(t, cmd.RunCommand())
	require.EqualValues(t, 2, lists.Load())
	require.EqualValues(t, 2, deletes.Load())
	require.Equal(t, int64(2), cmd.revertOption.revertCount)
	require.Equal(t, "prefix/first.txt", <-deleted)
	require.Equal(t, "prefix/second.txt", <-deleted)
}
