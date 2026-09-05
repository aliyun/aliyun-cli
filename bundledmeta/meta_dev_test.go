//go:build !aliyun_cli_packed_meta

package bundledmeta

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestNewMetadataFSUsesExplicitDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "canonical"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "metadatas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "metadatas", "products.json"), []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALIYUN_CLI_META_DIR", root)

	data, err := fs.ReadFile(newMetadataFS(), "metadatas/products.json")
	if err != nil {
		t.Fatalf("ReadFile explicit metadata directory: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("products data = %q", data)
	}
}

func TestNewMetadataFSFindsDirectoryFromWorkingDirectory(t *testing.T) {
	t.Setenv("ALIYUN_CLI_META_DIR", "")
	root := t.TempDir()
	metadataDir := filepath.Join(root, defaultMetadataDir)
	if err := os.MkdirAll(filepath.Join(metadataDir, "canonical"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(metadataDir, "metadatas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metadataDir, "metadatas", "products.json"), []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}
	workingDir := filepath.Join(root, "bundledmeta")
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workingDir)

	if _, err := fs.ReadFile(newMetadataFS(), "metadatas/products.json"); err != nil {
		t.Fatalf("ReadFile discovered metadata directory: %v", err)
	}
}

func TestNewMetadataFSFallsBackToDefaultDirectory(t *testing.T) {
	t.Setenv("ALIYUN_CLI_META_DIR", "")
	t.Chdir(t.TempDir())
	if _, err := fs.Stat(newMetadataFS(), "."); err == nil {
		t.Fatal("missing default metadata directory unexpectedly exists")
	}
}

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

func TestIsMetadataDirRejectsWrongEntryTypes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "canonical"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "metadatas", "products.json"), 0755); err != nil {
		t.Fatal(err)
	}
	if isMetadataDir(root) {
		t.Fatal("products.json directory accepted as metadata file")
	}

	if err := os.RemoveAll(filepath.Join(root, "metadatas", "products.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "metadatas", "products.json"), []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "canonical")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "canonical"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	if isMetadataDir(root) {
		t.Fatal("canonical file accepted as metadata directory")
	}
}
