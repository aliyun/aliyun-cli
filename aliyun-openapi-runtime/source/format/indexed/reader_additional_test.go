package indexed

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
)

func openAdditionalReader(t *testing.T) (*Reader, storage.Volume) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("firstsecond")
	writeTestFile(t, filepath.Join(dir, schema.MetadataDataFile), data)
	records := []Record{
		{APIVersion: "v1", APIName: "First", CommandName: "first", CmdFullName: "first-full", DescriptionZH: "一", DescriptionEN: "one", Deprecated: true, Offset: 0, Length: 5},
		{APIVersion: "v2", APIName: "Second", CommandName: "second", Offset: 5, Length: 6},
	}
	writeIndex(t, dir, data, records)
	vol, err := storage.NewDirStorage(root).Open("plugin")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := Open(vol, "", "")
	if err != nil {
		vol.Close()
		t.Fatal(err)
	}
	return reader, vol
}

func TestReaderAccessorsAndLookupFailures(t *testing.T) {
	reader, vol := openAdditionalReader(t)
	defer vol.Close()

	copy := reader.Index()
	copy.APIs[0].APIName = "changed"
	if reader.Index().APIs[0].APIName != "First" {
		t.Fatal("Index returned an alias instead of a copy")
	}
	idx, err := reader.APIIndex("demo", "v1")
	if err != nil {
		t.Fatal(err)
	}
	entry := idx.Entries["First"]
	if idx.ProductCode != "demo" || entry.CmdName != "first" || entry.CmdFullName != "first-full" || !entry.Deprecated || entry.Description.EN != "one" {
		t.Fatalf("APIIndex = %#v", idx)
	}
	if _, err := reader.APIIndex("demo", "missing"); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("missing APIIndex error = %v", err)
	}
	if _, err := reader.ReadAPI("v1", "missing"); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("missing ReadAPI error = %v", err)
	}
	if (*Reader)(nil).ForVolume(vol) != nil {
		t.Fatal("nil ForVolume must remain nil")
	}
	bound := reader.ForVolume(shortReadVolume{Volume: vol})
	if _, err := bound.ReadAPI("v1", "First"); err == nil || !strings.Contains(err.Error(), "short metadata record") {
		t.Fatalf("short read error = %v", err)
	}
	bound = reader.ForVolume(errorReadVolume{Volume: vol})
	if _, err := bound.ReadAPI("v1", "First"); !errors.Is(err, errAdditionalRead) {
		t.Fatalf("read error = %v", err)
	}
}

func TestProductEndpoints(t *testing.T) {
	p := Product{
		GlobalEndpoint:       "global.example",
		RegionalEndpoints:    map[string]string{"cn-a": "public.example"},
		RegionalVPCEndpoints: map[string]string{"cn-a": "vpc.example"},
	}
	got := p.Endpoints()
	if got.Global != "global.example" || got.Public["cn-a"] != "public.example" || got.VPC["cn-a"] != "vpc.example" {
		t.Fatalf("Endpoints = %#v", got)
	}
	reader, vol := openAdditionalReader(t)
	defer vol.Close()
	if reader.ProductEndpoints().Global != "" {
		t.Fatalf("unexpected fixture endpoints: %#v", reader.ProductEndpoints())
	}
}

func TestVerifyChecksumErrors(t *testing.T) {
	reader, vol := openAdditionalReader(t)
	defer vol.Close()

	missing := *reader
	missing.index.DataSHA256 = ""
	if err := missing.VerifyChecksum(); err == nil || !strings.Contains(err.Error(), "missing dataSha256") {
		t.Fatalf("missing checksum error = %v", err)
	}
	invalid := *reader
	invalid.index.DataSHA256 = "sha256:not-hex"
	if err := invalid.VerifyChecksum(); err == nil || !strings.Contains(err.Error(), "invalid metadata dataSha256") {
		t.Fatalf("invalid checksum error = %v", err)
	}
	wrong := *reader
	wrong.index.DataSHA256 = strings.Repeat("0", 64)
	if err := wrong.VerifyChecksum(); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("mismatch error = %v", err)
	}
	readFailure := reader.ForVolume(errorAllVolume{Volume: vol})
	if err := readFailure.VerifyChecksum(); !errors.Is(err, errAdditionalRead) {
		t.Fatalf("checksum read error = %v", err)
	}
}

func TestOpenRejectsInvalidMetadataContracts(t *testing.T) {
	valid := Index{Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion, DataSize: 1}
	tests := []struct {
		name      string
		index     Index
		data      []byte
		dataIsDir bool
		want      string
	}{
		{name: "contract", index: Index{}, data: []byte("x"), want: "unsupported metadata index contract"},
		{name: "size", index: valid, data: []byte("xx"), want: "data size mismatch"},
		{name: "missing fields", index: withAdditionalRecords(valid, Record{Offset: 0, Length: 1}), data: []byte("x"), want: "missing apiVersion"},
		{name: "negative offset", index: withAdditionalRecords(valid, Record{APIVersion: "v", APIName: "A", CommandName: "a", Offset: -1, Length: 1}), data: []byte("x"), want: "invalid byte range"},
		{name: "zero length", index: withAdditionalRecords(valid, Record{APIVersion: "v", APIName: "A", CommandName: "a", Length: 0}), data: []byte("x"), want: "invalid byte range"},
		{name: "overlap", index: withAdditionalRecords(Index{Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion, DataSize: 3}, Record{APIVersion: "v", APIName: "A", CommandName: "a", Offset: 0, Length: 2}, Record{APIVersion: "v", APIName: "B", CommandName: "b", Offset: 1, Length: 2}), data: []byte("xxx"), want: "overlaps"},
		{name: "duplicate", index: withAdditionalRecords(Index{Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion, DataSize: 2}, Record{APIVersion: "v", APIName: "A", CommandName: "a", Offset: 0, Length: 1}, Record{APIVersion: "v", APIName: "A", CommandName: "a", Offset: 1, Length: 1}), data: []byte("xx"), want: "duplicate metadata record"},
		{name: "directory", index: valid, dataIsDir: true, want: "is a directory"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "plugin")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.dataIsDir {
				if err := os.Mkdir(filepath.Join(dir, schema.MetadataDataFile), 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				writeTestFile(t, filepath.Join(dir, schema.MetadataDataFile), tc.data)
			}
			raw, err := jsonMarshalAdditional(tc.index)
			if err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, filepath.Join(dir, schema.MetadataIndexFile), raw)
			vol, err := storage.NewDirStorage(root).Open("plugin")
			if err != nil {
				t.Fatal(err)
			}
			defer vol.Close()
			_, err = Open(vol, "", "")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Open error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestOpenPathAndDecodeErrors(t *testing.T) {
	reader, vol := openAdditionalReader(t)
	defer vol.Close()
	_ = reader

	for _, name := range []string{"../index", "/index", `bad\index`} {
		if _, err := Open(vol, name, ""); err == nil || !strings.Contains(err.Error(), "unsafe metadata entry path") {
			t.Fatalf("Open index %q error = %v", name, err)
		}
	}
	if _, err := Open(vol, "missing.json", ""); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("missing index error = %v", err)
	}

	root := t.TempDir()
	dir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, schema.MetadataIndexFile), []byte("{"))
	badVol, err := storage.NewDirStorage(root).Open("plugin")
	if err != nil {
		t.Fatal(err)
	}
	defer badVol.Close()
	if _, err := Open(badVol, "", ""); err == nil || !strings.Contains(err.Error(), "decode metadata index") {
		t.Fatalf("decode error = %v", err)
	}

	idx := Index{Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion, DataFile: "other.data", DataSize: 1}
	raw, _ := jsonMarshalAdditional(idx)
	writeTestFile(t, filepath.Join(dir, schema.MetadataIndexFile), raw)
	if _, err := Open(badVol, "", schema.MetadataDataFile); err == nil || !strings.Contains(err.Error(), "does not match manifest data") {
		t.Fatalf("data mismatch error = %v", err)
	}
	idx.DataFile = `bad\data`
	raw, _ = jsonMarshalAdditional(idx)
	writeTestFile(t, filepath.Join(dir, schema.MetadataIndexFile), raw)
	if _, err := Open(badVol, "", ""); err == nil || !strings.Contains(err.Error(), "unsafe metadata entry path") {
		t.Fatalf("unsafe data error = %v", err)
	}
}

func withAdditionalRecords(index Index, records ...Record) Index {
	index.APIs = records
	return index
}

func jsonMarshalAdditional(value any) ([]byte, error) {
	return json.Marshal(value)
}

var errAdditionalRead = errors.New("additional read failure")

type shortReadVolume struct{ storage.Volume }

func (v shortReadVolume) ReadAt(string, int64, int64) ([]byte, error) { return []byte("x"), nil }

type errorReadVolume struct{ storage.Volume }

func (v errorReadVolume) ReadAt(string, int64, int64) ([]byte, error) { return nil, errAdditionalRead }

type errorAllVolume struct{ storage.Volume }

func (v errorAllVolume) ReadAll(string) ([]byte, error) { return nil, errAdditionalRead }
