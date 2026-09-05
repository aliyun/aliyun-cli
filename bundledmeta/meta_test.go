package bundledmeta

import (
	"encoding/json"
	"io/fs"
	"testing"
)

func TestMetadatasReadCanonicalAPI(t *testing.T) {
	data, err := fs.ReadFile(Metadatas, "canonical/ecs/2014-05-26/DescribeRegions.json")
	if err != nil {
		t.Fatalf("ReadFile canonical API: %v", err)
	}
	var api struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &api); err != nil {
		t.Fatalf("Unmarshal canonical API: %v", err)
	}
	if api.Name != "DescribeRegions" {
		t.Fatalf("api name = %q, want DescribeRegions", api.Name)
	}
}

func TestMetadatasFilesystem(t *testing.T) {
	root, err := fs.ReadDir(Metadatas, ".")
	if err != nil {
		t.Fatalf("ReadDir root: %v", err)
	}
	rootDirs := make(map[string]bool, len(root))
	for _, entry := range root {
		rootDirs[entry.Name()] = entry.IsDir()
	}
	if !rootDirs["canonical"] || !rootDirs["metadatas"] {
		t.Fatalf("required metadata directories not found in root entries: %v", root)
	}

	entries, err := fs.ReadDir(Metadatas, "canonical/ecs/2014-05-26")
	if err != nil {
		t.Fatalf("ReadDir canonical version: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == "DescribeRegions.json" && !entry.IsDir() {
			return
		}
	}
	t.Fatal("DescribeRegions.json not found")
}

func TestMetadatasReadProducts(t *testing.T) {
	data, err := fs.ReadFile(Metadatas, "metadatas/products.json")
	if err != nil {
		t.Fatalf("ReadFile products: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("products.json is empty")
	}
}
