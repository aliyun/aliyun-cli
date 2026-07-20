package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
)

func TestDirSourceLoadsJSONLPluginByManifestProductCode(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "aliyun-cli-ecs")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "name":"aliyun-cli-ecs",
  "version":"1.2.3",
  "type":"meta",
  "productCode":"ecs",
  "command":"ecs",
  "productName":{"en":"Elastic Compute Service","zh":"云服务器 ECS"},
  "apiVersions":{"default":"2014-05-26","supported":["2014-05-26"]},
  "metadata":{"format":"json","schema":"aliyun-openapi-meta","schemaVersion":1,"layout":"jsonl","layoutVersion":1,"index":"metadata.index.json","data":"metadata.jsonl"}
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	def := schema.CommandDefinition{
		Name: "DescribeInstances", CmdName: "describe-instances", DescriptionZH: "查询实例",
		Operation: &schema.OperationConfig{Action: "DescribeInstances", APIVersion: "2014-05-26", Method: "POST", APIStyle: "RPC"},
	}
	raw, _ := json.Marshal(def)
	data := append(raw, '\n')
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataDataFile), data, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	index := jsonl.Index{
		Schema: jsonl.SchemaName, SchemaVersion: jsonl.SchemaVersion, LayoutVersion: jsonl.LayoutVersion,
		DataFile: schema.MetadataDataFile, DataSize: int64(len(data)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		Product: jsonl.Product{
			GlobalEndpoint: "ecs.aliyuncs.com",
			RegionalEndpoints: map[string]string{
				"cn-hangzhou": "ecs.cn-hangzhou.aliyuncs.com",
			},
		},
		APIs: []jsonl.Record{{APIVersion: "2014-05-26", APIName: def.Name, CommandName: def.CmdName, Offset: 0, Length: int64(len(raw))}},
	}
	indexRaw, _ := json.Marshal(index)
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataIndexFile), indexRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	src := NewUserPluginSource(root)
	product, provenance, err := src.LoadProduct("ecs")
	if err != nil {
		t.Fatal(err)
	}
	if product.Code != "ecs" || product.DefaultVersion != "2014-05-26" {
		t.Fatalf("LoadProduct() = %#v", product)
	}
	if got := product.Endpoints.Public["cn-hangzhou"]; got != "ecs.cn-hangzhou.aliyuncs.com" {
		t.Fatalf("product endpoint = %q", got)
	}
	if provenance.Origin != pluginDir {
		t.Fatalf("provenance origin = %q, want %q", provenance.Origin, pluginDir)
	}
	idx, err := src.LoadIndex("ecs", "2014-05-26")
	if err != nil {
		t.Fatal(err)
	}
	if idx.ResolveCmd("describe-instances") != "DescribeInstances" {
		t.Fatalf("ResolveCmd() = %q", idx.ResolveCmd("describe-instances"))
	}
	api, err := src.LoadAPI("ecs", "2014-05-26", "DescribeInstances")
	if err != nil {
		t.Fatal(err)
	}
	if api.ProductCode != "ecs" || api.Name != "DescribeInstances" {
		t.Fatalf("LoadAPI() = %#v", api)
	}
	if got := api.Endpoints.Public["cn-hangzhou"]; got != "ecs.cn-hangzhou.aliyuncs.com" {
		t.Fatalf("api endpoint = %q", got)
	}
}

func TestDirSourceSkipsExplicitGoPlugin(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "aliyun-cli-go-only")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"aliyun-cli-go-only","type":"go","productCode":"go-only","command":"go-only","bin":{"path":"bin/plugin"}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := NewUserPluginSource(root).LoadProduct("go-only")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadProduct() error = %v, want ErrNotFound", err)
	}
}

func TestDirSourceTargetLookupDoesNotScanUnrelatedPlugins(t *testing.T) {
	root := t.TempDir()
	brokenDir := filepath.Join(root, "unrelated-plugin")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, "manifest.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A lookup for one selected product checks only its canonical directories.
	// The malformed unrelated plugin must neither be read nor affect the miss.
	_, _, err := NewUserPluginSource(root).LoadProduct("ecs")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadProduct() error = %v, want ErrNotFound", err)
	}
}

func TestDirSourceRejectsBareProductDirectory(t *testing.T) {
	root := t.TempDir()
	bareDir := filepath.Join(root, "ecs")
	if err := os.MkdirAll(bareDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bareDir, "manifest.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := NewUserPluginSource(root).LoadProduct("ecs")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadProduct() error = %v, want ErrNotFound", err)
	}
}
