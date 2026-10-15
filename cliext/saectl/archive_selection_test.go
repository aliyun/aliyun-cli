package saectl

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestZipEntrySelectionAndSizeLimit(t *testing.T) {
	name := "saectl"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	entries := []struct {
		name, body string
		mode       os.FileMode
	}{{name: "."}, {name: "../outside", body: "bad"}, {name: "folder/", mode: os.ModeDir | 0700}, {name: "folder/link", mode: os.ModeSymlink | 0700}, {name: "README", body: "documentation"}, {name: "large/" + name, body: "too large"}, {name: "first/" + name, body: "first"}, {name: "second/" + name, body: "second"}}
	for _, entry := range entries {
		h := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		h.SetMode(entry.mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	for pos := 0; pos < len(data); {
		offset := bytes.Index(data[pos:], []byte{'P', 'K', 1, 2})
		if offset < 0 {
			break
		}
		pos += offset
		n := int(binary.LittleEndian.Uint16(data[pos+28:]))
		if string(data[pos+46:pos+46+n]) == "large/"+name {
			binary.LittleEndian.PutUint32(data[pos+24:], uint32(saectlMaxExtractSize+1))
			break
		}
		pos += 46 + n
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "archive.zip")
	if err := os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := unzip(source, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "first", name))
	if err != nil || string(got) != "first" {
		t.Fatalf("selected binary=%q,%v", got, err)
	}
	for _, entry := range []string{"README", "folder/link", "large/" + name, "second/" + name} {
		if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(entry))); !os.IsNotExist(err) {
			t.Errorf("unwanted entry %q exists or unreadable: %v", entry, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "outside")); !os.IsNotExist(err) {
		t.Fatalf("traversal path extracted: %v", err)
	}
}
