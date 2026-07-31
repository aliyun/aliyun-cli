package pbmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/format/indexed"
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
		Operation: &Operation{
			Action: "GetThing", ApiVersion: "2020-01-01", Method: "GET", ApiStyle: "ROA",
			Protocol: "HTTPS", Url: "/things/{id}", IsSse: true,
			ReqBodyType: "formData", ContentType: "application/x-www-form-urlencoded",
		},
		Parameters: []*Argument{{
			Name: "limit", RawName: "Limit", Type: "integer", Options: []string{"--limit"},
			Location: "query", Example: "12****",
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
	index := indexed.Index{
		Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion,
		DataFile: DataFileName, DataSize: int64(len(data)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		Product: indexed.Product{RegionalEndpoints: map[string]string{"cn-hangzhou": "demo.cn-hangzhou.aliyuncs.com"}},
		APIs:    []indexed.Record{{APIVersion: "2020-01-01", APIName: "GetThing", CommandName: "get-thing", Offset: int64(len(prefix)), Length: int64(len(payload))}},
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
	if !api.IsSSE {
		t.Fatal("protobuf is_sse was not mapped")
	}
	if api.ReqBodyType != "formData" || api.ContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("protobuf request body metadata = %q, %q", api.ReqBodyType, api.ContentType)
	}
	if got := api.Parameters[0].Example; got != "12****" {
		t.Fatalf("parameter example = %q, want %q", got, "12****")
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
