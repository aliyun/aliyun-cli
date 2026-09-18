package plugin

import (
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type archiveBrokenReader struct{}

func (archiveBrokenReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestArchiveWriteFailuresRemovePartialFiles(t *testing.T) {
	for _, extract := range []bool{false, true} {
		for _, broken := range []bool{false, true} {
			dest := filepath.Join(t.TempDir(), "output")
			var r io.Reader = strings.NewReader("oversized")
			if broken {
				r = archiveBrokenReader{}
			}
			var err error
			if extract {
				_, err = writeExtractedFile(dest, 0600, r, 2)
			} else {
				err = writeReaderWithLimit(r, dest, 2)
			}
			require.Error(t, err)
			if broken {
				require.True(t, errors.Is(err, io.ErrUnexpectedEOF))
			} else {
				require.ErrorContains(t, err, "limit")
			}
			require.NoFileExists(t, dest)
		}
		dest := filepath.Join(t.TempDir(), "missing", "output")
		if extract {
			_, err := writeExtractedFile(dest, 0600, strings.NewReader("x"), 1)
			require.Error(t, err)
		} else {
			require.Error(t, writeReaderWithLimit(strings.NewReader("x"), dest, 1))
		}
	}
}
func TestArchiveInvalidSourcesAndDestinations(t *testing.T) {
	for _, format := range []string{"zip", "tgz"} {
		t.Run(format, func(t *testing.T) {
			extract := unzip
			create := createTestZip
			if format == "tgz" {
				extract = untar
				create = createTestTarGz
			}
			dir := t.TempDir()
			dest := filepath.Join(dir, "out")
			require.Error(t, extract(filepath.Join(dir, "missing"), dest))
			require.ErrorContains(t, extract(dir, dest), "directory")
			invalid := filepath.Join(dir, "invalid")
			require.NoError(t, os.WriteFile(invalid, []byte("broken archive"), 0600))
			require.Error(t, extract(invalid, dest))
			for _, entry := range []testFile{{name: "blocked/", isDir: true}, {name: "blocked/child", content: "x"}, {name: "blocked", content: "x"}} {
				archive := filepath.Join(t.TempDir(), "archive")
				require.NoError(t, create(archive, []testFile{entry}))
				target := t.TempDir()
				if entry.name == "blocked" {
					require.NoError(t, os.Mkdir(filepath.Join(target, "blocked"), 0700))
				} else {
					require.NoError(t, os.WriteFile(filepath.Join(target, "blocked"), []byte("preserve"), 0600))
				}
				require.Error(t, extract(archive, target))
			}
		})
	}
	require.ErrorContains(t, validateExtractedFileSize("negative", -1, 0, defaultPluginArchiveLimits), "invalid size")
	for _, name := range []string{"", ".", "\\escape"} {
		_, err := safeArchiveTarget(t.TempDir(), name)
		require.Error(t, err)
	}
	require.Equal(t, defaultPluginArchiveLimits, (*Manager)(nil).effectiveArchiveLimits())
}

func TestArchiveRejectsCorruptedTarAndUnsupportedZipMethod(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "broken.tgz")
	f, err := os.Create(tarPath)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	_, err = gz.Write([]byte("truncated tar header"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
	require.ErrorIs(t, untar(tarPath, filepath.Join(dir, "tar")), io.ErrUnexpectedEOF)
	zipPath := filepath.Join(dir, "unsupported.zip")
	f, err = os.Create(zipPath)
	require.NoError(t, err)
	zw := zip.NewWriter(f)
	_, err = zw.CreateRaw(&zip.FileHeader{Name: "file", Method: 99})
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())
	require.ErrorIs(t, unzip(zipPath, filepath.Join(dir, "zip")), zip.ErrAlgorithm)
}
