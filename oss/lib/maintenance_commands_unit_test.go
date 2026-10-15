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
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

type maintenanceTransport func(*http.Request) (*http.Response, error)

func (f maintenanceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOSSUpdateReplacesOnlyTemporaryBinaryAndRestoresOnFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	previousArgs, previousClient, previousVersion := os.Args, http.DefaultClient, vVersion
	t.Cleanup(func() { os.Args, http.DefaultClient, vVersion = previousArgs, previousClient, previousVersion })
	path := filepath.Join(t.TempDir(), "ossutil.exe")
	os.Args = []string{path}
	require.NoError(t, os.WriteFile(path, []byte("original"), 0700))
	var rejectBinary bool
	http.DefaultClient = &http.Client{Transport: maintenanceTransport(func(r *http.Request) (*http.Response, error) {
		body, status := "replacement", http.StatusOK
		if r.URL.Path == "/"+updateVersionObject {
			body = "3.0.0\n 3.1.0\n"
		} else if rejectBinary {
			body, status = "download denied", http.StatusForbidden
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	base := localOSSCommand(t, "unused")
	base.options[OptionForce] = new(bool)
	for _, language := range []string{LEnglishLanguage, strings.ToLower(ChineseLanguage)} {
		base.options[OptionLanguage] = &language
		vVersion = "v3.1.0"
		require.NoError(t, (&UpdateCommand{command: base}).RunCommand())
		vVersion = "3.0.0"
		withBucketInput(t, "n\n", func() { require.NoError(t, (&UpdateCommand{command: base}).RunCommand()) })
	}
	force := true
	base.options[OptionForce] = &force
	vVersion = "3.0.0"
	require.NoError(t, (&UpdateCommand{command: base}).RunCommand())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "replacement", string(data))
	require.NoError(t, os.WriteFile(path, []byte("original"), 0700))
	rejectBinary = true
	require.ErrorContains(t, (&UpdateCommand{command: base}).RunCommand(), "download denied")
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	http.DefaultClient.Transport = maintenanceTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("transport unavailable") })
	require.ErrorContains(t, (&UpdateCommand{command: base}).RunCommand(), "transport unavailable")
}

func TestMultipartListAndSizePagination(t *testing.T) {
	var invalid atomic.Bool
	var partPages atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("uploads") {
			io.WriteString(w, `<ListMultipartUploadsResult><Bucket>bucket</Bucket><IsTruncated>false</IsTruncated><Upload><Key>part key</Key><UploadId>upload</UploadId></Upload></ListMultipartUploadsResult>`)
			return
		}
		partPages.Add(1)
		next := "1"
		if invalid.Load() {
			next = "invalid"
		}
		truncated := r.URL.Query().Get("part-number-marker") == "0"
		flag := "false"
		if truncated {
			flag = "true"
		}
		io.WriteString(w, `<ListPartsResult><Bucket>bucket</Bucket><Key>part key</Key><UploadId>upload</UploadId><IsTruncated>`+flag+`</IsTruncated><NextPartNumberMarker>`+next+`</NextPartNumberMarker><Part><PartNumber>1</PartNumber><ETag>etag</ETag><Size>1024</Size><LastModified>2026-01-01T00:00:00Z</LastModified></Part></ListPartsResult>`)
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.args = []string{"oss://bucket/part key", "upload"}
	list := &ListPartCommand{command: base}
	require.NoError(t, list.RunCommand())
	require.EqualValues(t, 2, partPages.Load())
	base.args = []string{"oss://bucket"}
	base.options[OptionEncodingType] = stringPointer(URLEncodingType)
	all := &AllPartSizeCommand{command: base}
	require.NoError(t, all.RunCommand())
	require.Equal(t, []StatPartInfo{{"part key", "upload"}}, all.apOption.statList)
	bucket, err := base.ossBucket("bucket")
	require.NoError(t, err)
	count, size, err := all.GetObjectPartsSize(bucket, all.apOption.statList[0])
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.EqualValues(t, 2048, size)
	invalid.Store(true)
	_, _, err = all.GetObjectPartsSize(bucket, all.apOption.statList[0])
	require.ErrorContains(t, err, "invalid")
	list.command.args = []string{"oss://bucket/key", "upload"}
	require.ErrorContains(t, list.RunCommand(), "invalid")
	for _, args := range [][]string{{"local", "upload"}, {"oss://bucket", "upload"}} {
		base.args = args
		require.Error(t, (&ListPartCommand{command: base}).RunCommand())
	}
}

func TestRevertDeleteMarkersAgainstLocalServer(t *testing.T) {
	var deleted atomic.Int32
	var reject atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reject.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
			return
		}
		if r.Method == http.MethodGet {
			io.WriteString(w, `<ListVersionsResult><Name>bucket</Name><IsTruncated>false</IsTruncated><DeleteMarker><Key>key</Key><VersionId>marker</VersionId><IsLatest>true</IsLatest><LastModified>2026-01-01T00:00:00Z</LastModified></DeleteMarker><DeleteMarker><Key>different</Key><VersionId>other</VersionId><IsLatest>false</IsLatest><LastModified>2026-01-01T00:00:00Z</LastModified></DeleteMarker></ListVersionsResult>`)
		} else {
			data, _ := io.ReadAll(r.Body)
			if strings.Contains(string(data), "<VersionId>marker</VersionId>") {
				deleted.Add(1)
			}
			io.WriteString(w, `<DeleteResult/>`)
		}
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.args = []string{"oss://bucket/key"}
	base.options[OptionRequestPayer] = stringPointer("requester")
	cmd := &RevertCommand{command: base}
	require.NoError(t, cmd.RunCommand())
	require.EqualValues(t, 1, deleted.Load())
	require.EqualValues(t, 1, cmd.revertOption.revertCount)
	marker := &oss.ObjectDeleteMarkerProperties{Key: "key", LastModified: time.Unix(100, 0)}
	cmd.revertOption.startTime = 101
	require.False(t, cmd.filterDeleteMarker(marker))
	cmd.revertOption.startTime, cmd.revertOption.endTime = 0, 99
	require.False(t, cmd.filterDeleteMarker(marker))
	cmd.revertOption.endTime = 100
	require.True(t, cmd.filterDeleteMarker(marker))
	reject.Store(true)
	require.ErrorContains(t, (&RevertCommand{command: base}).RunCommand(), "AccessDenied")
	base.args = []string{"oss://bucket"}
	require.ErrorContains(t, (&RevertCommand{command: base}).RunCommand(), "object key")
	base.args = []string{"local"}
	require.Error(t, (&RevertCommand{command: base}).RunCommand())
}

func TestHashCommandLocalContentAndReadFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "content")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0600))
	for _, kind := range []string{"md5", "crc64"} {
		base := Command{args: []string{path}, options: OptionMapType{OptionHashType: &kind}}
		output := captureBucketOutput(t, func() { require.NoError(t, (&HashCommand{command: base}).RunCommand()) })
		if kind == "md5" {
			require.Contains(t, output, "5D41402ABC4B2A76B9719D911017C592")
			require.Contains(t, output, "XUFAKrxLKna5cZ2REBfFkg==")
		} else {
			require.Contains(t, output, "CRC64-ECMA")
		}
	}
	base := Command{args: []string{path + ".missing"}}
	require.Error(t, (&HashCommand{command: base}).RunCommand())
	errReader := maintenanceErrorReader{}
	require.ErrorContains(t, hashMD5(errReader), "read failed")
	require.ErrorContains(t, hashCRC64(errReader), "read failed")
}

type maintenanceErrorReader struct{}

func (maintenanceErrorReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
