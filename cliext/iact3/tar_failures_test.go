package iact3

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestTarBinaryFilesystemAndSizeBoundaries(t *testing.T) {
	name := "iact3"
	for _, scenario := range []string{"blocked parent", "output directory", "truncated body", "oversized entry"} {
		t.Run(scenario, func(t *testing.T) {
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			tw := tar.NewWriter(gz)
			size := int64(4)
			if scenario == "oversized entry" {
				size = maxExtractSize + 1
			}
			if err := tw.WriteHeader(&tar.Header{Name: "nested/" + name, Mode: 0600, Typeflag: tar.TypeReg, Size: size}); err != nil {
				t.Fatal(err)
			}
			if scenario != "truncated body" && scenario != "oversized entry" {
				if _, err := tw.Write([]byte("data")); err != nil {
					t.Fatal(err)
				}
				if err := tw.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			source := filepath.Join(dir, "archive.tar.gz")
			if err := os.WriteFile(source, b.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(dir, "out")
			if scenario == "blocked parent" {
				if err := os.WriteFile(dest, []byte("unchanged"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "output directory" {
				if err := os.MkdirAll(filepath.Join(dest, "nested", name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			err := extractTarGz(source, dest)
			if scenario == "oversized entry" {
				if _, statErr := os.Stat(filepath.Join(dest, "nested", name)); !os.IsNotExist(statErr) {
					t.Fatalf("oversized entry extracted: %v", statErr)
				}
			} else if err == nil {
				t.Fatal("invalid extraction destination or truncated body accepted")
			}
		})
	}
}
