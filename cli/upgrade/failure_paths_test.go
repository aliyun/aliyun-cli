package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractionFailuresPreserveDestination(t *testing.T) {
	for _, format := range []string{"tgz", "zip"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "archive."+format)
			destination := filepath.Join(dir, "directory")
			require.NoError(t, os.Mkdir(destination, 0700))
			if format == "tgz" {
				createTestTarGz(t, archive, "aliyun", []byte("binary"))
			} else {
				createTestZip(t, archive, "aliyun", []byte("binary"))
			}
			require.Error(t, extractBinary(archive, destination, "aliyun"))
			info, err := os.Stat(destination)
			require.NoError(t, err)
			require.True(t, info.IsDir())
			require.NoError(t, os.WriteFile(archive, []byte("corrupt archive"), 0600))
			require.Error(t, extractBinary(archive, destination, "aliyun"))
			require.Error(t, extractBinary(filepath.Join(dir, "missing."+format), destination, "aliyun"))
		})
	}
}

func TestDownloadFilePropagatesReadFailures(t *testing.T) {
	previous := httpClient
	t.Cleanup(func() { httpClient = previous })
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(failingReader{})}, nil
	})}
	require.ErrorContains(t, downloadFile(io.Discard, "https://download.example/archive", filepath.Join(t.TempDir(), "archive")), "simulated body read failure")
}

func TestUpgradeFailsBeforeChangingBinary(t *testing.T) {
	previousClient, previousURL := httpClient, ossVersionURL
	t.Cleanup(func() { httpClient = previousClient; ossVersionURL = previousURL })
	ossVersionURL = "https://download.example/version"
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(bytes.NewReader(nil))}, nil
	})}
	require.ErrorContains(t, upgradeViaDirect(newTestContext(), "0.0.1"), "failed to check for updates")
}

func TestCopyFileRejectsDirectoryDestination(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(source, []byte("preserved"), 0600))
	require.Error(t, copyFile(source, dir, 0600))
	data, err := os.ReadFile(source)
	require.NoError(t, err)
	require.Equal(t, "preserved", string(data))
}

func TestDownloadAndExtractMissingTemporaryDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, missing)
	}
	_, cleanup, err := downloadAndExtract(io.Discard, "https://download.example/archive", "archive.tgz")
	require.ErrorContains(t, err, "failed to create temp directory")
	require.Nil(t, cleanup)
}

func TestExtractionRejectsTruncatedTarAndCorruptZip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "truncated.tgz")
	file, err := os.Create(archive)
	require.NoError(t, err)
	compressed := gzip.NewWriter(file)
	writer := tar.NewWriter(compressed)
	require.NoError(t, writer.WriteHeader(&tar.Header{Name: "aliyun", Size: 1024, Typeflag: tar.TypeReg, Mode: 0755}))
	_, err = writer.Write([]byte("short"))
	require.NoError(t, err)
	require.Error(t, writer.Close())
	require.NoError(t, compressed.Close())
	require.NoError(t, file.Close())
	require.Error(t, extractFromTarGz(archive, filepath.Join(dir, "tar-output"), "aliyun"))
	var data bytes.Buffer
	zipWriter := zip.NewWriter(&data)
	entry, err := zipWriter.CreateHeader(&zip.FileHeader{Name: "aliyun", Method: zip.Store})
	require.NoError(t, err)
	_, err = entry.Write([]byte("new-binary"))
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())
	damaged := bytes.Replace(data.Bytes(), []byte("new-binary"), []byte("bad-binary"), 1)
	archive = filepath.Join(dir, "damaged.zip")
	require.NoError(t, os.WriteFile(archive, damaged, 0600))
	require.ErrorIs(t, extractFromZip(archive, filepath.Join(dir, "zip-output"), "aliyun"), zip.ErrChecksum)
}
