//go:build !aliyun_cli_packed_meta

package bundledmeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindMetadataDirFromSupportedWorkingDirectories(t *testing.T) {
	root := t.TempDir()
	metadataDir := filepath.Join(root, defaultMetadataDir)
	if err := os.MkdirAll(filepath.Join(metadataDir, "metadatas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(metadataDir, "canonical"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metadataDir, "metadatas", "products.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	starts := []string{
		root,
		filepath.Join(root, "bundledmeta"),
		filepath.Join(root, "openapi", "runtimehost"),
	}
	for _, start := range starts {
		if err := os.MkdirAll(start, 0755); err != nil {
			t.Fatal(err)
		}
		got, ok := findMetadataDir(start)
		if !ok {
			t.Fatalf("metadata not found from %s", start)
		}
		if got != metadataDir {
			t.Fatalf("metadata dir from %s = %s, want %s", start, got, metadataDir)
		}
	}
}

func TestFindMetadataDirRejectsIncompleteTree(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, defaultMetadataDir, "canonical"), 0755); err != nil {
		t.Fatal(err)
	}
	if got, ok := findMetadataDir(root); ok {
		t.Fatalf("incomplete metadata tree accepted: %s", got)
	}
}
