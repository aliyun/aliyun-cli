package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/jsonl"
)

func TestValidateMetadataPlugin(t *testing.T) {
	root := t.TempDir()
	writeAnyMetadataPlugin(t, root, false)
	pluginDir := filepath.Join(root, "aliyun-cli-demo")
	descriptor := &schema.MetadataDescriptor{
		Format:        schema.FormatJSON,
		Schema:        schema.SchemaName,
		SchemaVersion: schema.SchemaVersion,
		Layout:        jsonl.LayoutName,
		LayoutVersion: schema.LayoutVersion,
		Index:         schema.MetadataIndexFile,
		Data:          schema.MetadataDataFile,
	}

	if err := ValidateMetadataPlugin(pluginDir, descriptor); err != nil {
		t.Fatalf("valid metadata plugin rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataDataFile), []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMetadataPlugin(pluginDir, descriptor); err == nil || !strings.Contains(err.Error(), "invalid metadata plugin") {
		t.Fatalf("corrupt metadata plugin error = %v", err)
	}
}

func TestValidateMetadataPluginRejectsUnsupportedContract(t *testing.T) {
	err := ValidateMetadataPlugin(t.TempDir(), &schema.MetadataDescriptor{
		Format:        "unknown",
		Schema:        schema.SchemaName,
		SchemaVersion: schema.SchemaVersion,
		Layout:        "unknown",
		LayoutVersion: schema.LayoutVersion,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported metadata contract") {
		t.Fatalf("unsupported contract error = %v", err)
	}
}
