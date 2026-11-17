package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestCnameReplicationAndNamedConfigurationValidation(t *testing.T) {
	for _, tc := range []struct {
		method, item, want string
	}{
		{"", "", "method value is empty"},
		{"invalid", "", "only support get,put,delete"},
		{"delete", "token", "only support get bucket token"},
		{"get", "certificate", "only support put bucket"},
		{"get", "unknown", "item only support"},
		{"put", "", "cname is emtpy"},
		{"delete", "", "cname is emtpy"},
		{"get", "token", "cname is emtpy"},
		{"put", "token", "cname is emtpy"},
	} {
		base := localOSSCommand(t, "http://127.0.0.1")
		base.args = []string{"oss://bucket"}
		base.options[OptionMethod], base.options[OptionItem] = &tc.method, &tc.item
		require.ErrorContains(t, (&BucketCnameCommand{command: base}).RunCommand(), tc.want)
	}
	for _, newCommand := range []func(Command) Commander{
		func(c Command) Commander { return &BucketInventoryCommand{command: c} },
		func(c Command) Commander { return &BucketStyleCommand{command: c} },
	} {
		for _, method := range []string{"get", "delete"} {
			base := localOSSCommand(t, "http://127.0.0.1")
			base.args, base.options[OptionMethod] = []string{"oss://bucket"}, &method
			require.ErrorContains(t, newCommand(base).RunCommand(), "need at least 2 parameters")
		}
	}
	for _, tc := range []struct{ method, item, target, want string }{
		{"", "", "oss://bucket", "method value is empty"},
		{"invalid", "", "oss://bucket", "method value is not"},
		{"get", "invalid", "oss://bucket", "item value is not"},
		{"get", "", "local", "cloud url"},
		{"delete", "", "oss://bucket", "rule ID is empty"},
	} {
		base := localOSSCommand(t, "http://127.0.0.1")
		base.args = []string{tc.target}
		base.options[OptionMethod], base.options[OptionItem] = &tc.method, &tc.item
		require.ErrorContains(t, (&ReplicationCommand{command: base}).RunCommand(), tc.want)
	}
	for _, input := range []string{filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		base := localOSSCommand(t, "http://127.0.0.1")
		base.args = []string{"oss://bucket", input}
		base.options[OptionMethod], base.options[OptionItem] = stringPointer("put"), stringPointer("certificate")
		require.Error(t, (&BucketCnameCommand{command: base}).RunCommand())
	}
}

func TestBucketInventoryAndStyleListsPreserveOutputOnFailures(t *testing.T) {
	for _, kind := range []string{"inventory", "style"} {
		t.Run(kind, func(t *testing.T) {
			var mode atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch mode.Load() {
				case 1:
					w.WriteHeader(403)
				case 2:
					io.WriteString(w, "<")
				default:
					if kind == "style" {
						io.WriteString(w, `<StyleList><Style><Name>thumbnail</Name></Style></StyleList>`)
					} else {
						io.WriteString(w, `<ListInventoryConfigurationsResult><InventoryConfiguration><Id>daily</Id></InventoryConfiguration></ListInventoryConfigurationsResult>`)
					}
				}
			}))
			defer server.Close()
			base := localOSSCommand(t, server.URL)
			base.args, base.options[OptionMethod] = []string{"oss://bucket"}, stringPointer("list")
			run := func() error {
				if kind == "style" {
					return (&BucketStyleCommand{command: base}).RunCommand()
				}
				return (&BucketInventoryCommand{command: base}).RunCommand()
			}
			for _, failure := range []int32{1, 2} {
				mode.Store(failure)
				require.Error(t, run())
			}
			mode.Store(0)
			output := captureBucketOutput(t, func() { require.NoError(t, run()) })
			if kind == "style" {
				require.Contains(t, output, "thumbnail")
			} else {
				require.Contains(t, output, "daily")
			}
			file := filepath.Join(t.TempDir(), "output")
			require.NoError(t, os.WriteFile(file, []byte("preserved"), 0600))
			base.args = append(base.args, file)
			withBucketInput(t, "n\n", func() { require.NoError(t, run()) })
			data, err := os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, "preserved", string(data))
			withBucketInput(t, "y\n", func() { require.NoError(t, run()) })
			data, err = os.ReadFile(file)
			require.NoError(t, err)
			require.NotEqual(t, "preserved", string(data))
			base.args[1] = filepath.Join(file, "child")
			var pathError *os.PathError
			require.ErrorAs(t, run(), &pathError)
			base.options[OptionEndpoint] = stringPointer("")
			require.ErrorContains(t, run(), "endpoint")
		})
	}
}

func TestMakeBucketMetadataACLAndXMLReadFailures(t *testing.T) {
	requests := make(chan http.Header, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests <- r.Header.Clone() }))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.args = []string{"oss://bucket"}
	base.options[OptionMeta] = stringPointer("x-oss-meta-team:cli")
	require.NoError(t, (&MakeBucketCommand{command: base}).RunCommand())
	require.Equal(t, "cli", (<-requests).Get("X-Oss-Meta-Team"))
	for _, metadata := range []string{"invalid", "Expires:invalid"} {
		base.options[OptionMeta] = &metadata
		require.Error(t, (&MakeBucketCommand{command: base}).RunCommand())
	}
	base.options[OptionMeta] = stringPointer("")
	for _, language := range []string{"en", "ch"} {
		mc := MakeBucketCommand{command: base}
		withBucketInput(t, "private\n", func() {
			acl, err := mc.getACL("invalid", language)
			require.NoError(t, err)
			require.Equal(t, oss.ACLPrivate, acl)
		})
		withBucketInput(t, "", func() { _, err := mc.getACL("invalid", language); require.ErrorContains(t, err, "invalid acl") })
	}
	client, err := base.ossClient("bucket")
	require.NoError(t, err)
	for _, path := range []string{filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		require.Error(t, (&MakeBucketCommand{command: base}).createBucketXmlFile(client, "bucket", path, nil))
	}
	input := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(input, []byte(`<CreateBucketConfiguration/>`), 0600))
	require.NoError(t, (&MakeBucketCommand{command: base}).createBucketXmlFile(client, "bucket", input, []oss.Option{oss.Meta("team", "cli")}))
	require.Equal(t, "cli", (<-requests).Get("X-Oss-Meta-Team"))
	base.args, base.options[OptionACL] = []string{"oss://bucket"}, stringPointer("invalid")
	withBucketInput(t, "", func() { require.Error(t, (&MakeBucketCommand{command: base}).RunCommand()) })
	base.options[OptionEndpoint] = stringPointer("")
	require.ErrorContains(t, (&MakeBucketCommand{command: base}).RunCommand(), "endpoint")
}

func TestCatAndAppendPropagateMetadataAndTransportErrors(t *testing.T) {
	var heads atomic.Int32
	var deny atomic.Bool
	requests := make(chan string, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deny.Load() {
			w.WriteHeader(403)
			return
		}
		if r.Method == http.MethodHead {
			if heads.Add(1) > 1 {
				w.WriteHeader(403)
			}
			return
		}
		requests <- r.URL.Query().Get("versionId") + ":" + r.Header.Get("X-Oss-Request-Payer")
		io.WriteString(w, "content")
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.args = []string{"oss://bucket/key"}
	base.options[OptionRequestPayer], base.options[OptionVersionId] = stringPointer("requester"), stringPointer("v1")
	output := captureBucketOutput(t, func() { require.NoError(t, (&CatCommand{command: base}).RunCommand()) })
	require.Equal(t, "content", output)
	require.Equal(t, "v1:requester", <-requests)
	base.options[OptionRequestPayer] = stringPointer("invalid")
	require.ErrorContains(t, (&CatCommand{command: base}).RunCommand(), "invalid request payer")
	base.options[OptionRequestPayer] = stringPointer("")
	deny.Store(true)
	require.ErrorContains(t, (&CatCommand{command: base}).RunCommand(), "403")
	for _, target := range []string{"local", "oss://bucket", "oss://INVALID/key"} {
		base.args[0] = target
		require.Error(t, (&CatCommand{command: base}).RunCommand())
	}
	base.args[0], base.options[OptionEndpoint] = "oss://bucket/key", stringPointer("")
	require.ErrorContains(t, (&CatCommand{command: base}).RunCommand(), "endpoint")
	file := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0600))
	base = localOSSCommand(t, server.URL)
	base.args = []string{file, "oss://bucket/key"}
	for _, scenario := range []string{"payer", "client", "bucket", "existence", "metadata"} {
		c := base
		c.options = localOSSCommand(t, server.URL).options
		c.args = append([]string(nil), base.args...)
		switch scenario {
		case "payer":
			c.options[OptionRequestPayer] = stringPointer("bad")
		case "client":
			c.options[OptionEndpoint] = stringPointer("")
		case "bucket":
			c.args[1] = "oss://INVALID/key"
		case "existence":
			deny.Store(true)
		case "metadata":
			deny.Store(false)
			heads.Store(0)
		}
		require.Error(t, (&AppendFileCommand{command: c}).RunCommand())
	}
	cmd := AppendFileCommand{command: base}
	cmd.afOption.fileName, cmd.afOption.objectName = file, "key"
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	for _, metadata := range []string{"invalid", "Expires:invalid"} {
		cmd.afOption.ossMeta = metadata
		require.Error(t, cmd.AppendFromFile(bucket, 0))
	}
	cmd.afOption.fileName = filepath.Join(file, "missing")
	require.Error(t, cmd.AppendFromFile(bucket, 0))
	cmd.afOption.fileName, cmd.afOption.ossMeta = file, ""
	deny.Store(true)
	require.ErrorContains(t, cmd.AppendFromFile(bucket, 0), "403")
}

func TestListingMultipartBoundariesAndInvalidBuckets(t *testing.T) {
	var failParts atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Has("uploadId") {
			if failParts.Load() {
				w.WriteHeader(403)
				return
			}
			io.WriteString(w, `<ListPartsResult><Part><PartNumber>1</PartNumber><Size>3</Size></Part></ListPartsResult>`)
			return
		}
		if q.Get("key-marker") == "" {
			io.WriteString(w, `<ListMultipartUploadsResult><IsTruncated>true</IsTruncated><NextKeyMarker>next</NextKeyMarker><NextUploadIdMarker>upload-next</NextUploadIdMarker><Upload><Key>key</Key><UploadId>upload</UploadId></Upload></ListMultipartUploadsResult>`)
		} else {
			io.WriteString(w, `<ListMultipartUploadsResult><Upload><Key>second</Key><UploadId>upload2</UploadId></Upload></ListMultipartUploadsResult>`)
		}
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.args = []string{"oss://bucket"}
	cmd := AllPartSizeCommand{command: base}
	output := captureBucketOutput(t, func() { require.NoError(t, cmd.RunCommand()) })
	require.Contains(t, output, "second")
	require.Len(t, cmd.apOption.statList, 2)
	failParts.Store(true)
	require.ErrorContains(t, (&AllPartSizeCommand{command: base}).RunCommand(), "403")
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	_, _, err = cmd.GetObjectPartsSize(bucket, StatPartInfo{objectName: "key", uploadId: "upload"})
	require.Error(t, err)
	for _, endpoint := range []string{"", server.URL} {
		c := localOSSCommand(t, endpoint)
		c.args = []string{"oss://INVALID/key"}
		require.Error(t, (&AllPartSizeCommand{command: c}).RunCommand())
		lp := ListPartCommand{command: c}
		lp.lpOption.cloudUrl = CloudURL{bucket: "INVALID", object: "key"}
		require.Error(t, lp.ListPart())
	}
	base.args = []string{"not-cloud"}
	require.Error(t, (&AllPartSizeCommand{command: base}).RunCommand())
}

func TestServiceCommandsRejectClientAndBucketConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		factory func(Command) Commander
	}{
		{"symlink", []string{"oss://INVALID/link", "oss://bucket/key"}, func(c Command) Commander { return &CreateSymlinkCommand{command: c} }},
		{"read-symlink", []string{"oss://INVALID/link"}, func(c Command) Commander { return &ReadSymlinkCommand{command: c} }},
		{"mkdir", []string{"oss://INVALID/dir/"}, func(c Command) Commander { return &MkdirCommand{command: c} }},
		{"stat", []string{"oss://INVALID/key"}, func(c Command) Commander { return &StatCommand{command: c} }},
		{"options", []string{"oss://INVALID/key"}, func(c Command) Commander { return &OptionsCommand{command: c} }},
		{"signurl", []string{"oss://INVALID/key"}, func(c Command) Commander { return &SignurlCommand{command: c} }},
		{"ls", []string{"oss://INVALID/key"}, func(c Command) Commander { return &ListCommand{command: c} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, endpoint := range []string{"", "http://127.0.0.1"} {
				base := dataCommand(t, endpoint, Command{}, tc.args, nil)
				base.options[OptionOrigin] = stringPointer("https://example.test")
				base.options[OptionAcrMethod] = stringPointer("GET")
				require.Error(t, tc.factory(base).RunCommand())
			}
		})
	}
	base := localOSSCommand(t, "http://127.0.0.1")
	base.args = []string{"oss://bucket/key"}
	base.options[OptionTimeout] = stringPointer("-1")
	require.Error(t, (&SignurlCommand{command: base}).RunCommand())
	base.options[OptionRequestPayer] = stringPointer("invalid")
	require.ErrorContains(t, (&SignurlCommand{command: base}).RunCommand(), "request payer")
	base.args = []string{"local"}
	require.Error(t, (&SignurlCommand{command: base}).RunCommand())
}

func TestListZeroLimitsAndInvalidFiltersDoNotIssueRequests(t *testing.T) {
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(403) }))
	defer server.Close()
	base := dataCommand(t, server.URL, listCommand.command, nil, nil)
	lc := ListCommand{command: base}
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	for _, list := range []func(*int64) (int64, error){
		func(n *int64) (int64, error) {
			return lc.listObjectVersions(bucket, CloudURL{bucket: "bucket"}, false, false, n)
		},
		func(n *int64) (int64, error) {
			return lc.listMultipartUploads(bucket, CloudURL{bucket: "bucket"}, false, true, n)
		},
	} {
		limit := int64(0)
		n, err := list(&limit)
		require.NoError(t, err)
		require.Zero(t, n)
	}
	require.Zero(t, requests.Load())
	lc.command.args = []string{"oss://bucket/key"}
	lc.command.options[OptionRequestPayer] = stringPointer("invalid")
	require.ErrorContains(t, lc.RunCommand(), "invalid request payer")
	lc.command.options[OptionRequestPayer] = stringPointer("requester")
	os.Args = []string{"oss", "ls", "--include=dir/file"}
	require.ErrorContains(t, lc.RunCommand(), "does not support format containing dir info")
	require.Zero(t, requests.Load())
	os.Args = []string{"oss", "ls"}
	lc.command.args = []string{"oss://"}
	require.ErrorContains(t, lc.RunCommand(), "403")
	lc.command.args = nil
	lc.command.options[OptionEndpoint] = stringPointer("")
	require.ErrorContains(t, lc.RunCommand(), "endpoint")
	limit := int64(1)
	output := captureBucketOutput(t, func() {
		require.EqualValues(t, 1, lc.displayMultipartUploadsResult(oss.ListMultipartUploadResult{Uploads: []oss.UncompletedUpload{{Key: "key", UploadID: "upload"}}}, "bucket", false, true, 0, &limit))
	})
	require.Contains(t, output, "upload")
	require.Zero(t, limit)
}

func TestRegionBucketAndCloudBoxListingsReportLocalFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "endpoints")
	require.NoError(t, os.WriteFile(file, []byte(server.URL+"\n"), 0600))
	base := dataCommand(t, server.URL, lrbCommand.command, []string{file}, nil)
	cmd := LrbCommand{command: base}
	output := captureBucketOutput(t, func() { require.ErrorContains(t, cmd.RunCommand(), "403") })
	require.Equal(t, []string{server.URL}, cmd.errorEndpoints)
	require.Contains(t, output, "list bucket failure")
	require.Contains(t, output, server.URL)
	for _, target := range []string{"local", "oss://prefix"} {
		base = dataCommand(t, server.URL, lcbCommand.command, []string{target}, nil)
		require.Error(t, (&LcbCommand{command: base}).RunCommand())
	}
	base = dataCommand(t, "", lcbCommand.command, nil, nil)
	require.ErrorContains(t, (&LcbCommand{command: base}).RunCommand(), "endpoint")
	base.options[OptionEncodingType], base.options[OptionMarker] = stringPointer(URLEncodingType), stringPointer("%zz")
	require.ErrorContains(t, (&LcbCommand{command: base}).RunCommand(), "invalid marker")
}
