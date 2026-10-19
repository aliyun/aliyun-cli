package codeup

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type codeupDownloadTransport func(*http.Request) (*http.Response, error)

func (f codeupDownloadTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type codeupDownloadBody struct {
	io.Reader
	closed bool
}

func (b *codeupDownloadBody) Close() error { b.closed = true; return nil }

func codeupInstallDownloadTransport(t *testing.T, f codeupDownloadTransport) {
	t.Helper()
	original := httpClient
	httpClient = &http.Client{Transport: f}
	t.Cleanup(func() { httpClient = original })
}

func codeupZIPBytes(t *testing.T, target string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range []struct {
		name string
		mode os.FileMode
		body string
	}{
		{name: "bin/", mode: os.ModeDir | 0755},
		{name: "bin/" + target, mode: os.ModeSymlink | 0755, body: "elsewhere"},
		{name: "README.txt", mode: 0644, body: "installation notes"},
		{name: "release/" + target, mode: 0755, body: "downloaded executable"},
	} {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		header.SetMode(entry.mode)
		file, err := writer.CreateHeader(header)
		require.NoError(t, err)
		_, err = io.WriteString(file, entry.body)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func codeupRawZIPBytes(t *testing.T, header zip.FileHeader, body string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.CreateRaw(&header)
	require.NoError(t, err)
	_, err = io.WriteString(file, body)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func TestCodeupDownloadInstallsMatchingBinaryAndCleansArchive(t *testing.T) {
	for _, platform := range []string{"linux", "windows"} {
		t.Run(platform, func(t *testing.T) {
			originalOS := runtimeGOOSFunc
			runtimeGOOSFunc = func() string { return platform }
			t.Cleanup(func() { runtimeGOOSFunc = originalOS })
			name := "codeup-cli"
			if platform == "windows" {
				name += ".exe"
			}
			body := &codeupDownloadBody{Reader: bytes.NewReader(codeupZIPBytes(t, name))}
			requests := 0
			codeupInstallDownloadTransport(t, func(request *http.Request) (*http.Response, error) {
				requests++
				require.Equal(t, http.MethodGet, request.Method)
				require.Equal(t, "https://download.example.invalid/codeup.zip", request.URL.String())
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			})
			directory := t.TempDir()
			archive := filepath.Join(directory, "download.zip")
			executable := filepath.Join(directory, name)
			require.NoError(t, os.WriteFile(executable, []byte("previous version"), 0755))
			require.NoError(t, DownloadAndExtract("https://download.example.invalid/codeup.zip", archive, executable))
			installed, err := os.ReadFile(executable)
			require.NoError(t, err)
			require.Equal(t, "downloaded executable", string(installed))
			require.NoFileExists(t, archive)
			require.True(t, body.closed)
			require.Equal(t, 1, requests)
		})
	}
}

type codeupReadFailure struct{ first bool }

func (r *codeupReadFailure) Read(buffer []byte) (int, error) {
	if !r.first {
		r.first = true
		return copy(buffer, "partial archive"), nil
	}
	return 0, io.ErrUnexpectedEOF
}

type codeupZeroReader struct{}

func (codeupZeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}

func TestCodeupDownloadFailuresCloseBodyAndRemovePartialArchive(t *testing.T) {
	for _, test := range []struct {
		name string
		body func() io.Reader
		want string
	}{
		{name: "interrupted download", body: func() io.Reader { return &codeupReadFailure{} }, want: "failed to write to file"},
		{name: "download limit", body: func() io.Reader { return io.LimitReader(codeupZeroReader{}, int64(maxDownloadSize)+1) }, want: "download size exceeds limit"},
		{name: "invalid archive", body: func() io.Reader { return strings.NewReader("not a ZIP archive") }, want: "failed to open zip"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := &codeupDownloadBody{Reader: test.body()}
			codeupInstallDownloadTransport(t, func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			})
			directory := t.TempDir()
			archive := filepath.Join(directory, "download.zip")
			executable := filepath.Join(directory, "codeup-cli")
			err := DownloadAndExtract("https://download.example.invalid/codeup.zip", archive, executable)
			require.ErrorContains(t, err, test.want)
			require.NoFileExists(t, archive)
			require.NoFileExists(t, executable)
			require.True(t, body.closed)
		})
	}
}

func TestCodeupDownloadRejectsRequestStatusAndDestinationFailures(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.zip")
	executable := filepath.Join(directory, "codeup-cli")
	requests := 0
	codeupInstallDownloadTransport(t, func(*http.Request) (*http.Response, error) {
		requests++
		return nil, errors.New("connection interrupted")
	})
	require.ErrorContains(t, DownloadAndExtract(":invalid", destination, executable), "failed to create request")
	require.Zero(t, requests)
	require.ErrorContains(t, DownloadAndExtract("https://download.example.invalid/codeup.zip", destination, executable), "connection interrupted")
	require.Equal(t, 1, requests)
	for _, status := range []int{http.StatusForbidden, http.StatusOK} {
		body := &codeupDownloadBody{Reader: strings.NewReader("unused")}
		codeupInstallDownloadTransport(t, func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: body}, nil
		})
		err := DownloadAndExtract("https://download.example.invalid/codeup.zip", filepath.Join(directory, "missing", "download.zip"), executable)
		if status == http.StatusOK {
			require.ErrorContains(t, err, "failed to create file")
		} else {
			require.ErrorContains(t, err, "status code 403")
		}
		require.True(t, body.closed)
	}
	require.NoFileExists(t, executable)
}

func TestCodeupExtractRejectsInvalidEntriesAndCleansCorruptBinary(t *testing.T) {
	originalOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "linux" }
	t.Cleanup(func() { runtimeGOOSFunc = originalOS })
	for _, test := range []struct {
		name, want string
		header     zip.FileHeader
		body       string
	}{
		{name: "unrelated binary", header: zip.FileHeader{Name: "other-cli"}, want: "binary not found"},
		{name: "oversized entry", header: zip.FileHeader{Name: "codeup-cli", UncompressedSize64: uint64(maxExtractFileSize) + 1}, want: "file size exceeds limit"},
		{name: "unknown compression", header: zip.FileHeader{Name: "codeup-cli", Method: 99}, want: "failed to open file in zip"},
		{name: "checksum mismatch", header: zip.FileHeader{Name: "codeup-cli", CRC32: 1, CompressedSize64: 7, UncompressedSize64: 7}, body: "corrupt", want: "failed to extract file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			archive := filepath.Join(directory, "input.zip")
			executable := filepath.Join(directory, "codeup-cli")
			require.NoError(t, os.WriteFile(archive, codeupRawZIPBytes(t, test.header, test.body), 0600))
			err := extractZip(archive, executable)
			require.ErrorContains(t, err, test.want)
			require.NoFileExists(t, executable)
		})
	}
	archive := filepath.Join(t.TempDir(), "valid.zip")
	require.NoError(t, os.WriteFile(archive, codeupZIPBytes(t, "codeup-cli"), 0600))
	require.ErrorContains(t, extractZip(archive, filepath.Join(t.TempDir(), "missing", "codeup-cli")), "failed to create file")
}
