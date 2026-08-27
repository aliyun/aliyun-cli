package pbmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/indexed"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
	"google.golang.org/protobuf/proto"
)

func openAdditionalPBReader(t *testing.T, payload []byte) (*Reader, storage.Volume) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, DataFileName), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	index := indexed.Index{
		Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion,
		DataFile: DataFileName, DataSize: int64(len(payload)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		Product: indexed.Product{GlobalEndpoint: "global.example"},
		APIs: []indexed.Record{{
			APIVersion: "v1", APIName: "Action", CommandName: "action", DescriptionEN: "action description",
			Offset: 0, Length: int64(len(payload)),
		}},
	}
	raw, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, schema.MetadataIndexFile), raw, 0o644); err != nil {
		t.Fatal(err)
	}
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

func TestReaderDelegatesIndexOperations(t *testing.T) {
	payload, err := proto.Marshal(&CommandDefinition{
		Name: "Action", CmdName: "action",
		Operation: &Operation{Action: "Action", ApiVersion: "v1", Method: "POST", ApiStyle: "RPC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	reader, vol := openAdditionalPBReader(t, payload)
	defer vol.Close()

	if reader.Index().APIs[0].APIName != "Action" {
		t.Fatalf("Index = %#v", reader.Index())
	}
	if reader.ProductEndpoints().Global != "global.example" {
		t.Fatalf("ProductEndpoints = %#v", reader.ProductEndpoints())
	}
	idx, err := reader.APIIndex("demo", "v1")
	if err != nil || idx.Entries["Action"].CmdName != "action" {
		t.Fatalf("APIIndex = %#v, %v", idx, err)
	}
	if _, err := reader.APIIndex("demo", "missing"); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("missing APIIndex error = %v", err)
	}
	if _, err := reader.ReadAPI("v1", "missing"); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("missing ReadAPI error = %v", err)
	}
	if err := reader.VerifyChecksum(); err != nil {
		t.Fatal(err)
	}
}

func TestReaderRejectsInvalidProtobufAndOpenErrors(t *testing.T) {
	reader, vol := openAdditionalPBReader(t, []byte{0xff, 0xff})
	defer vol.Close()
	if _, err := reader.ReadAPI("v1", "Action"); err == nil || !strings.Contains(err.Error(), "decode protobuf api v1/Action") {
		t.Fatalf("ReadAPI error = %v", err)
	}
	if _, err := Open(vol, "missing.index", ""); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("Open missing index error = %v", err)
	}
}

func TestReaderWrapsCanonicalShapeErrors(t *testing.T) {
	payload, err := proto.Marshal(&CommandDefinition{
		Name: "Action", CmdName: "action",
		Operation:  &Operation{Action: "Action", ApiVersion: "v1", Method: "POST", ApiStyle: "RPC"},
		Parameters: []*Argument{{Name: "items", RawName: "Items", Type: "array"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reader, vol := openAdditionalPBReader(t, payload)
	defer vol.Close()
	_, err = reader.ReadAPI("v1", "Action")
	if err == nil || !strings.Contains(err.Error(), "decode protobuf api v1/Action") || !strings.Contains(err.Error(), "array is missing element") {
		t.Fatalf("ReadAPI canonical error = %v", err)
	}
}

func TestCanonicalPBMappingWithoutOptionalFields(t *testing.T) {
	canonical, err := toCanonical(&CommandDefinition{
		Name: "Simple", CmdName: "simple", CmdFullName: "simple-full", DescriptionZh: "简", DescriptionEn: "simple",
		Method: "GET", MultiVersion: true, Deprecated: true, KebabExample: "--foo bar", CamelExample: "--Foo bar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if canonical.Operation != nil || canonical.Parameters != nil || canonical.Name != "Simple" || !canonical.MultiVersion || !canonical.Deprecated {
		t.Fatalf("canonical = %#v", canonical)
	}
	if values, err := toCanonicalArguments(nil); err != nil || values != nil {
		t.Fatalf("empty arguments = %#v, %v", values, err)
	}
	if _, err := toCanonicalArguments([]*Argument{nil}); err == nil || !strings.Contains(err.Error(), "nil argument") {
		t.Fatalf("nil argument error = %v", err)
	}
}

func TestCanonicalPBShapeValidationBranches(t *testing.T) {
	tests := []struct {
		name string
		arg  *Argument
		want string
	}{
		{name: "object nil field", arg: &Argument{Type: "object", Fields: []*Argument{nil}}, want: "fields[0]: nil argument"},
		{name: "object nested array", arg: &Argument{Type: "object", Fields: []*Argument{{Type: "array"}}}, want: "array is missing element"},
		{name: "shape map missing value", arg: &Argument{Type: "array", Element: &TypeShape{Type: "map"}}, want: "map is missing value"},
		{name: "shape object nil field", arg: &Argument{Type: "array", Element: &TypeShape{Type: "object", Fields: []*Argument{nil}}}, want: "nil argument"},
		{name: "shape object nested map", arg: &Argument{Type: "array", Element: &TypeShape{Type: "object", Fields: []*Argument{{Type: "map"}}}}, want: "map is missing value"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := toCanonicalArguments([]*Argument{tc.arg})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCanonicalPBRecursiveShapes(t *testing.T) {
	argument := &Argument{
		Name: "items", RawName: "Items", Type: "array", Element: &TypeShape{
			Type: "map", Value: &TypeShape{Type: "object", Fields: []*Argument{{
				Name: "value", RawName: "Value", Type: "string", Format: "uuid", Enum: []string{"a"},
				Minimum: "1", Maximum: "2", MinLength: "1", MaxLength: "2", Pattern: "a", Required: true,
			}}},
		},
	}
	values, err := toCanonicalArguments([]*Argument{argument})
	if err != nil {
		t.Fatal(err)
	}
	field := values[0].Element.Value.Fields[0]
	if field.Name != "value" || field.Format != "uuid" || field.Pattern != "a" || !field.Required {
		t.Fatalf("recursive field = %#v", field)
	}
	shape, err := toCanonicalTypeShape(nil, "shape")
	if err != nil || shape != nil {
		t.Fatalf("nil shape = %#v, %v", shape, err)
	}
	if _, err := toCanonicalTypeShape(&TypeShape{Fields: []*Argument{nil}}, "shape"); err == nil || !strings.Contains(err.Error(), "nil argument") {
		t.Fatalf("shape field conversion error = %v", err)
	}
}
