package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/pbmeta"
	openapiruntime "github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"google.golang.org/protobuf/proto"
)

func TestJSONLAndProtobufAnyProduceSameDryRun(t *testing.T) {
	jsonRoot := t.TempDir()
	pbRoot := t.TempDir()
	writeAnyMetadataPlugin(t, jsonRoot, false)
	writeAnyMetadataPlugin(t, pbRoot, true)

	argv := []string{
		"--biz-body", `{"name":"demo","count":9007199254740993,"enabled":true}`,
		"--items", `[1,"x",true,{"k":"v"}]`,
		"--labels", `{"large":9007199254740993,"enabled":false}`,
		"--config", `{"Dynamic":{"nested":[1]}}`,
		"--empty", "null",
	}
	jsonRequest := loadAnyDryRun(t, jsonRoot, argv)
	pbRequest := loadAnyDryRun(t, pbRoot, argv)

	if !reflect.DeepEqual(jsonRequest, pbRequest) {
		jsonData, _ := json.MarshalIndent(jsonRequest, "", "  ")
		pbData, _ := json.MarshalIndent(pbRequest, "", "  ")
		t.Fatalf("JSONL/PB dry-run mismatch\nJSONL: %s\nPB: %s", jsonData, pbData)
	}
	body, ok := jsonRequest.Body.(map[string]any)
	if !ok || body["name"] != "demo" {
		t.Fatalf("direct Any body = %#v", jsonRequest.Body)
	}
	if _, wrapped := body["body"]; wrapped {
		t.Fatalf("direct Any body was wrapped: %#v", body)
	}
	if got := jsonRequest.Query["Items"]; got != `[1,"x",true,{"k":"v"}]` {
		t.Fatalf("array<any> query = %q", got)
	}
	if got := jsonRequest.Query["Labels"]; got != `{"enabled":false,"large":9007199254740993}` {
		t.Fatalf("map<any> query = %q", got)
	}
	if got := jsonRequest.Query["Config"]; got != `{"Dynamic":{"nested":[1]}}` {
		t.Fatalf("nested Any query = %q", got)
	}
	if _, exists := jsonRequest.Query["Empty"]; exists {
		t.Fatalf("null query must be omitted: %#v", jsonRequest.Query)
	}
}

func loadAnyDryRun(t *testing.T, root string, argv []string) *openapiruntime.AssembledRequest {
	t.Helper()
	api, err := NewUserPluginSource(root).LoadAPI("demo", "2020-01-01", "UpdateThing")
	if err != nil {
		t.Fatalf("LoadAPI: %v", err)
	}
	parsed, err := argparser.Parse(api.Parameters, argv)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	response, err := openapiruntime.NewExecutor().Execute(context.Background(), &openapiruntime.ExecContext{
		API: api, Args: parsed.Args, DryRun: true,
	})
	if err != nil {
		t.Fatalf("Execute dry-run: %v", err)
	}
	return response.Assembled
}

func writeAnyMetadataPlugin(t *testing.T, root string, protobuf bool) {
	t.Helper()
	pluginDir := filepath.Join(root, "aliyun-cli-demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}

	definition := anyCommandDefinition()
	var data []byte
	var offset int64
	dataFile := schema.MetadataDataFile
	descriptor := &schema.MetadataDescriptor{
		Format: "json", Schema: jsonl.SchemaName, SchemaVersion: jsonl.SchemaVersion,
		Layout: jsonl.LayoutName, LayoutVersion: jsonl.LayoutVersion,
		Index: schema.MetadataIndexFile, Data: schema.MetadataDataFile,
	}
	if protobuf {
		payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(anyPBCommandDefinition())
		if err != nil {
			t.Fatal(err)
		}
		prefix := encodeAnyTestVarint(len(payload))
		data = append(prefix, payload...)
		offset = int64(len(prefix))
		dataFile = pbmeta.DataFileName
		descriptor.Format = "protobuf"
		descriptor.Layout = pbmeta.LayoutName
		descriptor.Data = dataFile
	} else {
		payload, err := json.Marshal(definition)
		if err != nil {
			t.Fatal(err)
		}
		data = append(payload, '\n')
	}
	if err := os.WriteFile(filepath.Join(pluginDir, dataFile), data, 0o644); err != nil {
		t.Fatal(err)
	}

	digest := sha256.Sum256(data)
	index := jsonl.Index{
		Schema: jsonl.SchemaName, SchemaVersion: jsonl.SchemaVersion, LayoutVersion: jsonl.LayoutVersion,
		DataFile: dataFile, DataSize: int64(len(data)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		Product: jsonl.Product{GlobalEndpoint: "demo.example.com"},
		APIs: []jsonl.Record{{
			APIVersion: "2020-01-01", APIName: "UpdateThing", CommandName: "update-thing",
			Offset: offset, Length: int64(len(data)) - offset,
		}},
	}
	if !protobuf {
		index.APIs[0].Length-- // exclude the JSONL newline
	}
	writeAnyJSON(t, filepath.Join(pluginDir, schema.MetadataIndexFile), index)
	manifest := schema.PluginManifest{
		Name: "aliyun-cli-demo", Version: "1.0.0", Type: "meta", ProductCode: "demo", Command: "demo",
		APIVersions: schema.ManifestAPIVersions{Default: "2020-01-01", Supported: []string{"2020-01-01"}},
		Metadata:    descriptor,
	}
	writeAnyJSON(t, filepath.Join(pluginDir, "manifest.json"), manifest)
}

func anyCommandDefinition() schema.CommandDefinition {
	return schema.CommandDefinition{
		Name: "UpdateThing", CmdName: "update-thing",
		Operation: &schema.OperationConfig{
			Action: "UpdateThing", APIStyle: "ROA", APIVersion: "2020-01-01",
			Method: "POST", Protocol: "HTTPS", URL: "/things",
		},
		Parameters: []schema.ArgumentDefinition{
			{Name: "body", RawName: "body", Type: "any", Options: []string{"--biz-body"}, Location: "body"},
			{Name: "items", RawName: "Items", Type: "array", ElementType: "any", Options: []string{"--items"}, Location: "query", ParamStyle: "json"},
			{Name: "labels", RawName: "Labels", Type: "map", ValueType: "any", Options: []string{"--labels"}, Location: "query", ParamStyle: "json"},
			{Name: "config", RawName: "Config", Type: "object", Options: []string{"--config"}, Location: "query", ParamStyle: "json", Fields: []schema.ArgumentDefinition{
				{Name: "dynamic", RawName: "Dynamic", Type: "any"},
			}},
			{Name: "empty", RawName: "Empty", Type: "any", Options: []string{"--empty"}, Location: "query", ParamStyle: "json"},
		},
	}
}

func anyPBCommandDefinition() *pbmeta.CommandDefinition {
	return &pbmeta.CommandDefinition{
		Name: "UpdateThing", CmdName: "update-thing",
		Operation: &pbmeta.Operation{
			Action: "UpdateThing", ApiStyle: "ROA", ApiVersion: "2020-01-01",
			Method: "POST", Protocol: "HTTPS", Url: "/things",
		},
		Parameters: []*pbmeta.Argument{
			{Name: "body", RawName: "body", Type: "any", Options: []string{"--biz-body"}, Location: "body"},
			{Name: "items", RawName: "Items", Type: "array", ElementType: "any", Options: []string{"--items"}, Location: "query", ParamStyle: "json"},
			{Name: "labels", RawName: "Labels", Type: "map", ValueType: "any", Options: []string{"--labels"}, Location: "query", ParamStyle: "json"},
			{Name: "config", RawName: "Config", Type: "object", Options: []string{"--config"}, Location: "query", ParamStyle: "json", Fields: []*pbmeta.Argument{
				{Name: "dynamic", RawName: "Dynamic", Type: "any"},
			}},
			{Name: "empty", RawName: "Empty", Type: "any", Options: []string{"--empty"}, Location: "query", ParamStyle: "json"},
		},
	}
}

func writeAnyJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func encodeAnyTestVarint(value int) []byte {
	var output []byte
	for value > 0x7f {
		output = append(output, byte(value&0x7f)|0x80)
		value >>= 7
	}
	return append(output, byte(value))
}
