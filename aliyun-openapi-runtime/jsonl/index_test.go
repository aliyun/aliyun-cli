package jsonl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/format"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
)

func TestReaderRandomAccess(t *testing.T) {
	root := t.TempDir()
	volumeDir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(volumeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	definitions := []schema.CommandDefinition{
		{
			Name: "DescribeThings", CmdName: "describe-things", DescriptionZH: "描述资源",
			Operation: &schema.OperationConfig{Action: "DescribeThings", APIVersion: "2020-01-01", Method: "POST", APIStyle: "RPC"},
		},
		{
			Name: "GetThing", CmdName: "get-thing", DescriptionEN: "Get one thing",
			Operation: &schema.OperationConfig{Action: "GetThing", APIVersion: "2020-01-01", Method: "GET", APIStyle: "ROA", URL: "/things/{id}"},
		},
	}
	data, records := encodeDefinitions(t, definitions)
	writeTestFile(t, filepath.Join(volumeDir, schema.MetadataDataFile), data)
	writeIndex(t, volumeDir, data, records)

	vol, err := storage.NewDirStorage(root).Open("plugin")
	if err != nil {
		t.Fatal(err)
	}
	defer vol.Close()
	spy := &spyVolume{Volume: vol}
	reader, err := Open(spy, "", "")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if spy.dataReadAll != 0 {
		t.Fatalf("Open() read the complete data file %d times, want 0", spy.dataReadAll)
	}
	record, err := reader.ReadAPI("2020-01-01", "GetThing")
	if err != nil {
		t.Fatalf("ReadAPI() error = %v", err)
	}
	api, err := format.DecodeAPIJSON(record, "test")
	if err != nil {
		t.Fatal(err)
	}
	if api.Name != "GetThing" || api.Version != "2020-01-01" {
		t.Fatalf("decoded API = %s/%s", api.Version, api.Name)
	}
	if spy.readAt != 1 || spy.dataReadAll != 0 {
		t.Fatalf("ReadAPI access: ReadAt=%d data ReadAll=%d, want 1/0", spy.readAt, spy.dataReadAll)
	}
	if err := reader.VerifyChecksum(); err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
	if spy.dataReadAll != 1 {
		t.Fatalf("VerifyChecksum data ReadAll = %d, want 1", spy.dataReadAll)
	}
}

func TestOpenRejectsOutOfBoundsRecord(t *testing.T) {
	root := t.TempDir()
	volumeDir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(volumeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(volumeDir, schema.MetadataDataFile), []byte("{}\n"))
	idx := Index{
		Schema: SchemaName, SchemaVersion: SchemaVersion, LayoutVersion: LayoutVersion,
		DataFile: schema.MetadataDataFile, DataSize: 3,
		APIs: []Record{{APIVersion: "v1", APIName: "A", CommandName: "a", Offset: 2, Length: 9}},
	}
	raw, _ := json.Marshal(idx)
	writeTestFile(t, filepath.Join(volumeDir, schema.MetadataIndexFile), raw)
	vol, err := storage.NewDirStorage(root).Open("plugin")
	if err != nil {
		t.Fatal(err)
	}
	defer vol.Close()
	if _, err := Open(vol, "", ""); err == nil {
		t.Fatal("Open() succeeded with an out-of-bounds record")
	}
}

func TestOpenRejectsTraversalPath(t *testing.T) {
	root := t.TempDir()
	volumeDir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(volumeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "outside.jsonl"), []byte("{}\n"))
	idx := Index{Schema: SchemaName, SchemaVersion: SchemaVersion, LayoutVersion: LayoutVersion, DataFile: "../outside.jsonl", DataSize: 3}
	raw, _ := json.Marshal(idx)
	writeTestFile(t, filepath.Join(volumeDir, schema.MetadataIndexFile), raw)
	vol, err := storage.NewDirStorage(root).Open("plugin")
	if err != nil {
		t.Fatal(err)
	}
	defer vol.Close()
	if _, err := Open(vol, "", ""); err == nil {
		t.Fatal("Open() succeeded with a traversal data path")
	}
}

func encodeDefinitions(t *testing.T, definitions []schema.CommandDefinition) ([]byte, []Record) {
	t.Helper()
	var data []byte
	records := make([]Record, 0, len(definitions))
	for _, definition := range definitions {
		raw, err := json.Marshal(definition)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, Record{
			APIVersion: definition.Operation.APIVersion, APIName: definition.Name,
			CommandName: definition.CmdName, DescriptionZH: definition.DescriptionZH,
			DescriptionEN: definition.DescriptionEN, Offset: int64(len(data)), Length: int64(len(raw)),
		})
		data = append(data, raw...)
		data = append(data, '\n')
	}
	return data, records
}

func writeIndex(t *testing.T, volumeDir string, data []byte, records []Record) {
	t.Helper()
	digest := sha256.Sum256(data)
	idx := Index{
		Schema: SchemaName, SchemaVersion: SchemaVersion, LayoutVersion: LayoutVersion,
		DataFile: schema.MetadataDataFile, DataSize: int64(len(data)),
		DataSHA256: "sha256:" + hex.EncodeToString(digest[:]), APIs: records,
	}
	raw, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(volumeDir, schema.MetadataIndexFile), raw)
}

type spyVolume struct {
	storage.Volume
	dataReadAll int
	readAt      int
}

func (v *spyVolume) ReadAll(entry string) ([]byte, error) {
	if entry == schema.MetadataDataFile {
		v.dataReadAll++
	}
	return v.Volume.ReadAll(entry)
}

func (v *spyVolume) ReadAt(entry string, off, n int64) ([]byte, error) {
	if entry == schema.MetadataDataFile {
		v.readAt++
	}
	return v.Volume.ReadAt(entry, off, n)
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
