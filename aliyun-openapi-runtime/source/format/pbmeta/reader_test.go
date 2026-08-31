package pbmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/indexed"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
	"google.golang.org/protobuf/proto"
)

func TestToCanonicalMapsResponseMetadata(t *testing.T) {
	t.Run("protobuf", func(t *testing.T) {
		responses := []byte(`{"200":{"schema":{"type":"object"}}}`)
		components := []byte(`{"schemas":{"Result":{"type":"object"}}}`)
		canonical, err := toCanonical(&CommandDefinition{
			Name:       "ListTools",
			Responses:  responses,
			Components: components,
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(canonical.Responses) != string(responses) {
			t.Fatalf("Responses = %s, want %s", canonical.Responses, responses)
		}
		if string(canonical.Components) != string(components) {
			t.Fatalf("Components = %s, want %s", canonical.Components, components)
		}
	})
	t.Run("omits empty", func(t *testing.T) {
		canonical, err := toCanonical(&CommandDefinition{Name: "GetThing"})
		if err != nil {
			t.Fatal(err)
		}
		if len(canonical.Responses) != 0 || len(canonical.Components) != 0 {
			t.Fatalf("expected empty response metadata, got %#v", canonical)
		}
	})
}

func TestToCanonicalRejectsIncompleteRecursivePBShapes(t *testing.T) {
	tests := []struct {
		name string
		arg  *Argument
		want string
	}{
		{name: "array missing element", arg: &Argument{Type: "array"}, want: "array is missing element"},
		{name: "map missing value", arg: &Argument{Type: "map"}, want: "map is missing value"},
		{
			name: "nested array missing element",
			arg:  &Argument{Type: "map", Value: &TypeShape{Type: "array"}},
			want: "parameters[0].value: array is missing element",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := toCanonical(&CommandDefinition{Parameters: []*Argument{test.arg}})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

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
			Protocol: "HTTPS", Url: "/things/*", IsSse: true, HasWildcardPath: true,
			ReqBodyType: "formData", ContentType: "application/x-www-form-urlencoded",
		},
		Parameters: []*Argument{
			{
				Name: "limit", RawName: "Limit", Type: "integer", Options: []string{"--limit"},
				Location: "query", Example: "12****", Format: "int32",
				Enum: []string{"1", "10"}, Minimum: "1", Maximum: "10",
				DocRequired: true,
			},
			{
				Name: "request_path", RawName: "requestPath", Type: "string",
				Options: []string{"--request-path"}, Location: "path", IsWildcard: true,
			},
			{
				Name: "privileges", RawName: "Privileges", Type: "map",
				Value: &TypeShape{
					Type: "array",
					Element: &TypeShape{Type: "object", Fields: []*Argument{{
						Name: "enabled", RawName: "Enabled", Type: "boolean", DocRequired: true,
					}, {
						Name: "name", RawName: "Name", Type: "string",
						Pattern: "^[a-z]+$", Format: "custom",
					}}},
				},
			},
		},
	}
	responses := []byte(`{"200":{"description_zh":"成功","schema":{"type":"object"}}}`)
	components := []byte(`{"schemas":{"ThingResult":{"type":"object"}}}`)
	definition.Responses = responses
	definition.Components = components
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
	if api.Name != "GetThing" || api.Version != "2020-01-01" || api.URL != "/things/*" {
		t.Fatalf("API = %#v", api)
	}
	if !api.IsSSE {
		t.Fatal("protobuf is_sse was not mapped")
	}
	if !api.HasWildcardPath || !api.Parameters[1].IsWildcard {
		t.Fatalf("protobuf wildcard metadata was not mapped: %#v", api)
	}
	if api.ReqBodyType != "formData" || api.ContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("protobuf request body metadata = %q, %q", api.ReqBodyType, api.ContentType)
	}
	if got := api.Parameters[0].Example; got != "12****" {
		t.Fatalf("parameter example = %q, want %q", got, "12****")
	}
	if got := api.Parameters[0]; !reflect.DeepEqual(got.Enum, []string{"1", "10"}) ||
		got.Minimum != "1" || got.Maximum != "10" || !got.DocRequired {
		t.Fatalf("protobuf numeric constraints were not mapped: %#v", got)
	}
	privileges := api.Parameters[2]
	if privileges.ValueType == nil || privileges.ValueType.ItemType == nil ||
		len(privileges.ValueType.ItemType.Fields) != 2 ||
		privileges.ValueType.ItemType.Fields[0].RawName != "Enabled" {
		t.Fatalf("protobuf recursive composite metadata was not mapped: %#v", privileges)
	}
	if got := privileges.ValueType.ItemType.Fields[1].Pattern; got != "^[a-z]+$" {
		t.Fatalf("protobuf recursive pattern = %q", got)
	}
	if !privileges.ValueType.ItemType.Fields[0].DocRequired {
		t.Fatalf("protobuf recursive docRequired was not mapped: %#v", privileges)
	}
	if got := reader.ProductEndpoints().Public["cn-hangzhou"]; got != "demo.cn-hangzhou.aliyuncs.com" {
		t.Fatalf("endpoint = %q", got)
	}
	if string(api.Responses) != string(responses) {
		t.Fatalf("Responses = %s, want %s", api.Responses, responses)
	}
	if string(api.Components) != string(components) {
		t.Fatalf("Components = %s, want %s", api.Components, components)
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
