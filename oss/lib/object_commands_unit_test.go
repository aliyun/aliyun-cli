package lib

import (
	"errors"
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

func TestObjectTaggingLocalRequestsAndValidation(t *testing.T) {
	var deny atomic.Bool
	requests := make(chan string, 20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests <- r.Method + " " + r.URL.RawQuery + " " + string(body)
		if deny.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
		} else if r.Method == http.MethodGet {
			io.WriteString(w, `<Tagging><TagSet><Tag><Key>team</Key><Value>cli</Value></Tag></TagSet></Tagging>`)
		} else if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	for _, method := range []string{"get", "put", "delete"} {
		base := localOSSCommand(t, server.URL)
		base.args = []string{"oss://bucket/key"}
		base.options[OptionMethod] = &method
		base.options[OptionVersionId] = stringPointer("version")
		base.options[OptionRequestPayer] = stringPointer("requester")
		if method == "put" {
			base.args = append(base.args, "team#cli")
		}
		cmd := &ObjectTagCommand{command: base}
		require.NoError(t, cmd.RunCommand())
		request := <-requests
		require.True(t, strings.HasPrefix(request, strings.ToUpper(method)))
		require.Contains(t, request, "versionId=version")
		if method == "get" {
			require.EqualValues(t, 1, cmd.objectIndex)
			require.True(t, cmd.printHeader)
		} else if method == "put" {
			require.Contains(t, request, "<Key>team</Key>")
			require.Contains(t, request, "<Value>cli</Value>")
		}
		deny.Store(true)
		require.ErrorContains(t, (&ObjectTagCommand{command: base}).RunCommand(), "AccessDenied")
		<-requests
		deny.Store(false)
	}
	for _, tc := range []struct {
		args  []string
		opts  map[string]string
		match string
	}{
		{[]string{"local"}, nil, "cloud"},
		{[]string{"oss://bucket/key"}, map[string]string{OptionMethod: ""}, "method"},
		{[]string{"oss://bucket/key"}, map[string]string{OptionMethod: "invalid"}, "method"},
		{[]string{"oss://bucket/key"}, map[string]string{OptionRequestPayer: "invalid"}, "payer"},
		{[]string{"oss://bucket/key"}, map[string]string{OptionRecursion: "true", OptionVersionId: "version"}, "version-id"},
		{[]string{"oss://bucket/key"}, map[string]string{OptionMethod: "put"}, "2 parameters"},
		{[]string{"oss://bucket/key", "invalid"}, map[string]string{OptionMethod: "put"}, "separated by #"},
		{[]string{"oss://bucket"}, nil, "object key"},
		{[]string{"oss://bucket/key"}, map[string]string{OptionEndpoint: ""}, "endpoint"},
	} {
		base := localOSSCommand(t, server.URL)
		base.args = tc.args
		base.options[OptionMethod] = stringPointer("get")
		for key, value := range tc.opts {
			if key == OptionRecursion {
				enabled := value == "true"
				base.options[key] = &enabled
			} else {
				base.options[key] = stringPointer(value)
			}
		}
		require.ErrorContains(t, (&ObjectTagCommand{command: base}).RunCommand(), tc.match)
	}
}

func TestObjectTaggingBatchWorkersAndFailures(t *testing.T) {
	for _, method := range []string{"get", "put", "delete"} {
		t.Run(method, func(t *testing.T) {
			var tagged atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !r.URL.Query().Has("tagging") {
					io.WriteString(w, `<ListBucketResult><Name>bucket</Name><IsTruncated>false</IsTruncated><Contents><Key>key</Key><Size>5</Size><LastModified>2026-01-01T00:00:00Z</LastModified><StorageClass>Standard</StorageClass></Contents></ListBucketResult>`)
				} else {
					tagged.Add(1)
					if method == "get" {
						io.WriteString(w, `<Tagging><TagSet><Tag><Key>team</Key><Value>cli</Value></Tag></TagSet></Tagging>`)
					} else if method == "delete" {
						w.WriteHeader(http.StatusNoContent)
					}
				}
			}))
			defer server.Close()
			base := localOSSCommand(t, server.URL)
			base.args = []string{"oss://bucket"}
			if method == "put" {
				base.args = append(base.args, "team#cli")
			}
			base.options[OptionMethod] = &method
			recursive := true
			base.options[OptionRecursion] = &recursive
			base.options[OptionOutputDir] = stringPointer(t.TempDir())
			base.options[OptionRoutines] = stringPointer("1")
			cmd := &ObjectTagCommand{command: base}
			require.NoError(t, cmd.RunCommand())
			require.EqualValues(t, 1, tagged.Load())
			if method == "get" {
				require.EqualValues(t, 1, cmd.objectIndex)
			}
		})
	}
	for _, listFailure := range []bool{false, true} {
		cmd := &ObjectTagCommand{}
		cmd.monitor.init("ObjectTagging")
		workErrors, listErrors := make(chan error, 2), make(chan error, 1)
		failure := errors.New("local batch failure")
		if listFailure {
			listErrors <- failure
		} else {
			workErrors <- failure
		}
		require.ErrorIs(t, cmd.waitRoutinueComplete(workErrors, listErrors, 1), failure)
	}
}

func TestObjectStatAndCORSLocalResponses(t *testing.T) {
	var deny atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deny.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
			return
		}
		switch {
		case r.Method == http.MethodOptions:
			if r.Header.Get("Origin") != "https://example.test" || r.Header.Get("Access-Control-Request-Method") != "GET" || r.Header.Get("Access-Control-Request-Headers") != "x-test" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "https://example.test")
		case r.Method == http.MethodHead:
			w.Header().Set("Last-Modified", "Thu, 15 Oct 2026 00:00:00 GMT")
			w.Header().Set("Content-Length", "7")
			w.Header().Set("ETag", `"etag"`)
			w.Header().Set("X-Oss-Meta-Team", "cli")
		case r.URL.Query().Has("acl"):
			io.WriteString(w, `<AccessControlPolicy><Owner><ID>owner</ID></Owner><AccessControlList><Grant>private</Grant></AccessControlList></AccessControlPolicy>`)
		default:
			io.WriteString(w, `<BucketInfo><Bucket><Name>bucket</Name><Location>oss-cn-test</Location><CreationDate>2026-01-01T00:00:00Z</CreationDate><StorageClass>Standard</StorageClass><DataRedundancyType>ZRS</DataRedundancyType><AccessMonitor>Enabled</AccessMonitor><ServerSideEncryptionRule><SSEAlgorithm>KMS</SSEAlgorithm><KMSMasterKeyID>key</KMSMasterKeyID><KMSDataEncryption>SM4</KMSDataEncryption></ServerSideEncryptionRule></Bucket></BucketInfo>`)
		}
	}))
	defer server.Close()
	for _, object := range []string{"oss://bucket", "oss://bucket/key"} {
		base := localOSSCommand(t, server.URL)
		base.args = []string{object}
		base.options[OptionVersionId] = stringPointer("version")
		base.options[OptionRequestPayer] = stringPointer("requester")
		stdout, err := os.Create(filepath.Join(t.TempDir(), "stat"))
		require.NoError(t, err)
		previous := os.Stdout
		os.Stdout = stdout
		runErr := (&StatCommand{command: base}).RunCommand()
		os.Stdout = previous
		require.NoError(t, stdout.Close())
		require.NoError(t, runErr)
		data, err := os.ReadFile(stdout.Name())
		require.NoError(t, err)
		if object == "oss://bucket" {
			require.Contains(t, string(data), "ZRS")
			require.Contains(t, string(data), "KMS")
		} else {
			require.Contains(t, string(data), "etag")
			require.Contains(t, string(data), "private")
			require.Contains(t, string(data), "cli")
		}
		deny.Store(true)
		require.ErrorContains(t, (&StatCommand{command: base}).RunCommand(), "AccessDenied")
		deny.Store(false)
	}
	base := localOSSCommand(t, server.URL)
	base.args = []string{"oss://bucket/key"}
	base.options[OptionOrigin] = stringPointer("https://example.test")
	base.options[OptionAcrMethod] = stringPointer("get")
	base.options[OptionAcrHeaders] = stringPointer("x-test")
	require.NoError(t, (&OptionsCommand{command: base}).RunCommand())
	deny.Store(true)
	require.ErrorContains(t, (&OptionsCommand{command: base}).RunCommand(), "AccessDenied")
	for _, path := range []string{"local", "oss://"} {
		base.args = []string{path}
		require.Error(t, (&StatCommand{command: base}).RunCommand())
		require.Error(t, (&OptionsCommand{command: base}).RunCommand())
	}
}

func TestRequestPaymentAndVersioningValues(t *testing.T) {
	requests := make(chan string, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		requests <- string(data)
		io.WriteString(w, `<RequestPaymentConfiguration><Payer>Requester</Payer></RequestPaymentConfiguration>`)
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.options[OptionMethod] = stringPointer("put")
	for _, payer := range []string{string(oss.Requester), string(oss.BucketOwner)} {
		base.args = []string{"oss://bucket", payer}
		require.NoError(t, (&RequestPaymentCommand{command: base}).RunCommand())
		require.Contains(t, <-requests, "<Payer>"+payer+"</Payer>")
	}
	base.options[OptionMethod] = stringPointer("get")
	base.args = []string{"oss://bucket"}
	cmd := &RequestPaymentCommand{command: base}
	require.NoError(t, cmd.RunCommand())
	<-requests
	require.Equal(t, "Requester", cmd.paymentResult.Payer)
	base.options[OptionMethod] = stringPointer("put")
	base.args = []string{"oss://bucket", "suspended"}
	require.NoError(t, (&BucketVersioningCommand{command: base}).RunCommand())
	require.Contains(t, <-requests, "<Status>Suspended</Status>")
	for _, args := range [][]string{{"local"}, {"oss://bucket"}, {"oss://bucket", "invalid"}} {
		base.args = args
		require.Error(t, (&RequestPaymentCommand{command: base}).RunCommand())
		require.Error(t, (&BucketVersioningCommand{command: base}).RunCommand())
	}
}
