package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSymlinkAndDirectoryLocalOperations(t *testing.T) {
	var exists, denied atomic.Bool
	requests := make(chan string, 20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path + " " + r.Header.Get("X-Oss-Symlink-Target")
		if denied.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
			return
		}
		if r.Method == http.MethodHead && !exists.Load() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("X-Oss-Symlink-Target", "folder/target")
		w.Header().Set("Last-Modified", "Thu, 15 Oct 2026 00:00:00 GMT")
		w.Header().Set("ETag", `"etag"`)
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.options[OptionRequestPayer] = stringPointer("requester")
	for _, target := range []string{"folder/target", "oss://bucket/folder/target"} {
		base.args = []string{"oss://bucket/link", target}
		require.NoError(t, (&CreateSymlinkCommand{command: base}).RunCommand())
		require.Equal(t, "PUT /bucket/link folder%2Ftarget", <-requests)
	}
	base.args = []string{"oss://bucket/link"}
	base.options[OptionVersionId] = stringPointer("version")
	require.NoError(t, (&ReadSymlinkCommand{command: base}).RunCommand())
	require.Contains(t, <-requests, "GET /bucket/link")
	base.args = []string{"oss://bucket/folder"}
	require.NoError(t, (&MkdirCommand{command: base}).RunCommand())
	require.Contains(t, <-requests, "HEAD /bucket/folder/")
	require.Contains(t, <-requests, "PUT /bucket/folder/")
	exists.Store(true)
	require.ErrorContains(t, (&MkdirCommand{command: base}).RunCommand(), "already exists")
	<-requests
	denied.Store(true)
	require.ErrorContains(t, (&MkdirCommand{command: base}).RunCommand(), "StatusCode=403")
	<-requests
	base.args = []string{"oss://bucket/link"}
	require.ErrorContains(t, (&ReadSymlinkCommand{command: base}).RunCommand(), "AccessDenied")
	<-requests
	base.args = []string{"oss://bucket/link", "target"}
	require.ErrorContains(t, (&CreateSymlinkCommand{command: base}).RunCommand(), "AccessDenied")
	<-requests
	for _, path := range []string{"local", "oss://", "oss://bucket"} {
		base.args = []string{path}
		require.Error(t, (&ReadSymlinkCommand{command: base}).RunCommand())
		require.Error(t, (&MkdirCommand{command: base}).RunCommand())
		base.args = append(base.args, "target")
		require.Error(t, (&CreateSymlinkCommand{command: base}).RunCommand())
	}
	for _, target := range []string{"oss://other/key", "oss://bucket", "oss://"} {
		base.args = []string{"oss://bucket/link", target}
		require.Error(t, (&CreateSymlinkCommand{command: base}).RunCommand())
	}
}

func TestAppendFileLocalNewAndExistingObjects(t *testing.T) {
	var exists atomic.Bool
	var uploaded atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			if exists.Load() {
				w.Header().Set("Content-Length", "3")
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}
		data, _ := io.ReadAll(r.Body)
		if string(data) != "hello" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		position, _ := strconv.Atoi(r.URL.Query().Get("position"))
		w.Header().Set("X-Oss-Next-Append-Position", strconv.Itoa(position+len(data)))
		uploaded.Add(1)
	}))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0600))
	base := localOSSCommand(t, server.URL)
	base.args = []string{file, "oss://bucket/append"}
	base.options[OptionRequestPayer] = stringPointer("requester")
	base.options[OptionMeta] = stringPointer("Content-Type:text/plain")
	require.NoError(t, (&AppendFileCommand{command: base}).RunCommand())
	exists.Store(true)
	require.ErrorContains(t, (&AppendFileCommand{command: base}).RunCommand(), "setting meta")
	base.options[OptionMeta] = stringPointer("")
	require.NoError(t, (&AppendFileCommand{command: base}).RunCommand())
	require.EqualValues(t, 2, uploaded.Load())
	for _, path := range []string{file + ".missing", filepath.Dir(file)} {
		base.args[0] = path
		require.Error(t, (&AppendFileCommand{command: base}).RunCommand())
	}
	base.args = []string{file, "oss://bucket"}
	require.ErrorContains(t, (&AppendFileCommand{command: base}).RunCommand(), "object key")
	base.args = []string{file, "local"}
	require.Error(t, (&AppendFileCommand{command: base}).RunCommand())
}

func TestMakeBucketLocalOptionsAndXML(t *testing.T) {
	requests := make(chan string, 20)
	var denied atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests <- r.Header.Get("X-Oss-Acl") + " " + string(body)
		if denied.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
		}
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.args = []string{"oss://bucket"}
	base.options[OptionACL] = stringPointer("private")
	base.options[OptionRedundancyType] = stringPointer("zrs")
	for _, storage := range []string{"Standard", "IA", "Archive", "ColdArchive"} {
		base.options[OptionStorageClass] = &storage
		require.NoError(t, (&MakeBucketCommand{command: base}).RunCommand())
		request := <-requests
		require.True(t, strings.HasPrefix(request, "private"))
		require.Contains(t, request, "ZRS")
	}
	xml := filepath.Join(t.TempDir(), "bucket.xml")
	require.NoError(t, os.WriteFile(xml, []byte(`<CreateBucketConfiguration><StorageClass>IA</StorageClass></CreateBucketConfiguration>`), 0600))
	base.args = append(base.args, xml)
	require.NoError(t, (&MakeBucketCommand{command: base}).RunCommand())
	require.Contains(t, <-requests, "<StorageClass>IA</StorageClass>")
	base.args = []string{"oss://bucket"}
	denied.Store(true)
	require.ErrorContains(t, (&MakeBucketCommand{command: base}).RunCommand(), "AccessDenied")
	<-requests
	base.options[OptionRedundancyType] = stringPointer("invalid")
	require.ErrorContains(t, (&MakeBucketCommand{command: base}).RunCommand(), "redundancy-type")
	for _, path := range []string{"local", "oss://", "oss://bucket/key"} {
		base.args = []string{path}
		require.Error(t, (&MakeBucketCommand{command: base}).RunCommand())
	}
}

func TestSignURLRetainsVersionPayerAndQuery(t *testing.T) {
	base := localOSSCommand(t, "http://example.test")
	base.args = []string{"oss://bucket/folder/key"}
	base.options[OptionTimeout] = stringPointer("60")
	base.options[OptionVersionId] = stringPointer("version")
	base.options[OptionTrafficLimit] = stringPointer("819200")
	base.options[OptionRequestPayer] = stringPointer("requester")
	query := []string{"response-content-type:text/plain"}
	base.options[OptionQueryParam] = &query
	enabled := true
	base.options[OptionDisableEncodeSlash] = &enabled
	cmd := &SignurlCommand{command: base}
	require.NoError(t, cmd.RunCommand())
	u, err := url.Parse(cmd.signUrl)
	require.NoError(t, err)
	require.Equal(t, "/bucket/folder/key", u.Path)
	require.Equal(t, "version", u.Query().Get("versionId"))
	require.Equal(t, "requester", u.Query().Get("x-oss-request-payer"))
	require.Equal(t, "text/plain", u.Query().Get("response-content-type"))
	require.NotEmpty(t, u.Query().Get("Signature"))
	base.options[OptionTrafficLimit] = stringPointer("-1")
	require.ErrorContains(t, (&SignurlCommand{command: base}).RunCommand(), "greater than 0")
	base.options[OptionTrafficLimit] = stringPointer("0")
	base.options[OptionRequestPayer] = stringPointer("invalid")
	require.ErrorContains(t, (&SignurlCommand{command: base}).RunCommand(), "payer")
}

func TestListRegionBucketsAndCloudBoxesLocally(t *testing.T) {
	var denied atomic.Bool
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if denied.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
			return
		}
		if r.URL.Query().Has("cloudboxes") {
			truncated := "false"
			if r.URL.Query().Get("marker") == "" {
				truncated = "true"
			}
			io.WriteString(w, `<ListCloudBoxResult><IsTruncated>`+truncated+`</IsTruncated><NextMarker>next</NextMarker><Owner><DisplayName>owner</DisplayName></Owner><CloudBoxes><CloudBox><ID>cb-local</ID><Name>local</Name><Region>cn-test</Region><ControlEndpoint>control</ControlEndpoint><DataEndpoint>data</DataEndpoint></CloudBox></CloudBoxes></ListCloudBoxResult>`)
		} else {
			io.WriteString(w, `<ListAllMyBucketsResult><IsTruncated>false</IsTruncated><Buckets><Bucket><Name>bucket</Name><Location>oss-cn-test</Location><CreationDate>2026-01-01T00:00:00Z</CreationDate><StorageClass>Standard</StorageClass></Bucket></Buckets></ListAllMyBucketsResult>`)
		}
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	cmd := &LrbCommand{command: base}
	output := captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
	require.Len(t, cmd.listResult, 1)
	require.Contains(t, output, "oss://bucket")
	endpoints := filepath.Join(t.TempDir(), "endpoints")
	require.NoError(t, os.WriteFile(endpoints, []byte("# comment\n"+server.URL+"\n"), 0600))
	base.args = []string{endpoints}
	cmd = &LrbCommand{command: base}
	require.NoError(t, cmd.RunCommand())
	require.Len(t, cmd.listResult, 1)
	base.args = nil
	base.options[OptionLimitedNum] = stringPointer("-1")
	calls.Store(0)
	output = captureBucketOutput(t, func() { require.NoError(t, (&LcbCommand{command: base}).RunCommand()) })
	require.Contains(t, output, "cb-local")
	require.Contains(t, output, "cn-test")
	require.EqualValues(t, 2, calls.Load())
	base.options[OptionLimitedNum] = stringPointer("1")
	calls.Store(0)
	require.NoError(t, (&LcbCommand{command: base}).RunCommand())
	require.EqualValues(t, 1, calls.Load())
	denied.Store(true)
	require.ErrorContains(t, (&LrbCommand{command: base}).RunCommand(), "AccessDenied")
	require.ErrorContains(t, (&LcbCommand{command: base}).RunCommand(), "AccessDenied")
	base.args = []string{endpoints + ".missing"}
	require.Error(t, (&LrbCommand{command: base}).RunCommand())
	base.args = nil
	base.options[OptionEncodingType] = stringPointer(URLEncodingType)
	base.options[OptionMarker] = stringPointer("%zz")
	require.ErrorContains(t, (&LcbCommand{command: base}).RunCommand(), "marker")
}

func TestUserQoSLocalOutputAndErrors(t *testing.T) {
	var denied atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if denied.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
			return
		}
		io.WriteString(w, `<QoSConfiguration><Region>cn-test</Region><TotalUploadBandwidth>1024</TotalUploadBandwidth></QoSConfiguration>`)
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.options[OptionMethod] = stringPointer("get")
	output := filepath.Join(t.TempDir(), "qos")
	base.args = []string{output}
	run := func() { require.NoError(t, (&UserQosCommand{command: base}).RunCommand()) }
	run()
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(data), "cn-test")
	require.Contains(t, string(data), "1024")
	withBucketInput(t, "n\n", run)
	withBucketInput(t, "y\n", run)
	base.args = nil
	require.Contains(t, captureBucketOutput(t, run), "cn-test")
	denied.Store(true)
	require.ErrorContains(t, (&UserQosCommand{command: base}).RunCommand(), "AccessDenied")
	for _, method := range []string{"", "put"} {
		base.options[OptionMethod] = &method
		require.ErrorContains(t, (&UserQosCommand{command: base}).RunCommand(), "method")
	}
}
