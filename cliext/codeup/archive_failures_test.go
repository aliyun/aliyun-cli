package codeup

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func corruptZipFixture(t *testing.T, entry, kind string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	h := &zip.FileHeader{Name: entry, Method: zip.Store}
	h.SetMode(0600)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("archive corruption fixture")
	if _, err := w.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	switch kind {
	case "checksum":
		offset := bytes.Index(data, body)
		if offset < 0 {
			t.Fatal("fixture payload missing")
		}
		data[offset] ^= 1
	case "oversized":
		central := bytes.Index(data, []byte{'P', 'K', 1, 2})
		if central < 0 {
			t.Fatal("fixture central header missing")
		}
		binary.LittleEndian.PutUint32(data[central+24:], uint32(maxExtractFileSize+1))
	case "compression":
		local := bytes.Index(data, []byte{'P', 'K', 3, 4})
		central := bytes.Index(data, []byte{'P', 'K', 1, 2})
		if local < 0 || central < 0 {
			t.Fatal("fixture headers missing")
		}
		binary.LittleEndian.PutUint16(data[local+8:], 99)
		binary.LittleEndian.PutUint16(data[central+10:], 99)
	}
	return data
}

func TestZipFailureBoundaries(t *testing.T) {
	binaryName := "codeup-cli"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	for _, tc := range []struct {
		name, kind             string
		blockParent, blockFile bool
		want                   string
	}{
		{name: "checksum", kind: "checksum", want: "checksum"},
		{name: "oversized entry", kind: "oversized", want: "exceeds limit"},
		{name: "unsupported compression", kind: "compression", want: "algorithm"},
		{name: "blocked output parent", blockParent: true},
		{name: "output is directory", blockFile: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			data := corruptZipFixture(t, "sub/"+binaryName, tc.kind)
			source := filepath.Join(dir, "source.zip")
			if err := os.WriteFile(source, data, 0600); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(dir, "out")
			if tc.blockParent {
				if err := os.WriteFile(dest, []byte("unchanged"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.blockFile {
				if err := os.MkdirAll(filepath.Join(dest, "sub", binaryName), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dest, "sub", binaryName, "child"), []byte("unchanged"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if !tc.blockParent && !tc.blockFile {
				if err := os.MkdirAll(filepath.Join(dest, "sub"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			err := extractZip(source, filepath.Join(dest, "sub", binaryName))
			if err == nil || (tc.want != "" && !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("archive error=%v,want=%q", err, tc.want)
			}
			if tc.blockParent {
				got, err := os.ReadFile(dest)
				if err != nil || string(got) != "unchanged" {
					t.Fatalf("existing destination changed=%q,%v", got, err)
				}
			}
		})
	}
}
