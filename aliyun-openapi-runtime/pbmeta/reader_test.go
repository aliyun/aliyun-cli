package pbmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
	"google.golang.org/protobuf/proto"
)

func TestReaderDecodesOneIndexedAPI(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	definition := &CommandDefinition{
		ProductCode: "Demo", Name: "GetThing", CmdName: "get-thing",
		Operation: &Operation{Action: "GetThing", ApiVersion: "2020-01-01", Method: "GET", ApiStyle: "ROA", Protocol: "HTTPS", Url: "/things/{id}"},
		Parameters: []*Argument{{
			Name: "limit", RawName: "Limit", Type: "integer", Options: []string{"--limit"},
			Location: "query", HasDefault: true, DefaultJson: []byte("9007199254740993"),
		}},
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	prefix := encodeVarint(len(payload))
	data := append(append([]byte(nil), prefix...), payload...)
	if err := os.WriteFile(filepath.Join(pluginDir, DataFileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	index := jsonl.Index{
		Schema: jsonl.SchemaName, SchemaVersion: jsonl.SchemaVersion, LayoutVersion: LayoutVersion,
		DataFile: DataFileName, DataSize: int64(len(data)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		Product: jsonl.Product{RegionalEndpoints: map[string]string{"cn-hangzhou": "demo.cn-hangzhou.aliyuncs.com"}},
		APIs:    []jsonl.Record{{APIVersion: "2020-01-01", APIName: "GetThing", CommandName: "get-thing", Offset: int64(len(prefix)), Length: int64(len(payload))}},
	}
	indexData, _ := json.Marshal(index)
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataIndexFile), indexData, 0o644); err != nil {
		t.Fatal(err)
	}
	vol, err := storage.NewDirStorage(root).Open("plugin")
	if err != nil {
		t.Fatal(err)
	}
	defer vol.Close()
	reader, err := Open(vol, schema.MetadataIndexFile, DataFileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.VerifyChecksum(); err != nil {
		t.Fatal(err)
	}
	api, err := reader.ReadAPI("2020-01-01", "GetThing")
	if err != nil {
		t.Fatal(err)
	}
	if api.Name != "GetThing" || api.Version != "2020-01-01" || api.URL != "/things/{id}" {
		t.Fatalf("API = %#v", api)
	}
	if got := api.Parameters[0].Default; got == nil || got.(json.Number).String() != "9007199254740993" {
		t.Fatalf("default = %#v", got)
	}
	if got := reader.ProductEndpoints().Public["cn-hangzhou"]; got != "demo.cn-hangzhou.aliyuncs.com" {
		t.Fatalf("endpoint = %q", got)
	}
}

func encodeVarint(value int) []byte {
	var output []byte
	for value > 0x7f {
		output = append(output, byte(value&0x7f)|0x80)
		value >>= 7
	}
	return append(output, byte(value))
}
