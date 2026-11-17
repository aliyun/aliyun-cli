package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/bundledmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/require"
)

func TestGenerateMetadataWritesCommandAndProductDocuments(t *testing.T) {
	fixture, err := filepath.Abs("../canonicalmeta/testdata")
	require.NoError(t, err)
	previous := bundledmeta.Metadatas
	t.Cleanup(func() { bundledmeta.Metadatas = previous })
	bundledmeta.Metadatas = os.DirFS(fixture)
	t.Chdir(t.TempDir())
	root := &cli.Command{Name: "aliyun", Short: i18n.T("CLI", "CLI")}
	root.AddSubCommand(&cli.Command{Name: "version", Short: i18n.T("Version", "版本")})
	for range 2 {
		generateMetadata(root)
		data, err := os.ReadFile(filepath.Join("cli-metadata", "commands.json"))
		require.NoError(t, err)
		var documents map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &documents))
		require.NotEmpty(t, documents)
		version, err := os.ReadFile(filepath.Join("cli-metadata", "version"))
		require.NoError(t, err)
		require.Equal(t, cli.Version, string(version))
		require.FileExists(t, filepath.Join("cli-metadata", "metadatas", "demo", "CreateReport.json"))
	}
}

func TestGenerateMetadataHandlesOutputAndCatalogFailures(t *testing.T) {
	previous := bundledmeta.Metadatas
	t.Cleanup(func() { bundledmeta.Metadatas = previous })
	bundledmeta.Metadatas = fstest.MapFS{}
	for _, blocked := range []bool{true, false} {
		t.Run(map[bool]string{true: "blocked output", false: "missing catalog"}[blocked], func(t *testing.T) {
			t.Chdir(t.TempDir())
			if blocked {
				require.NoError(t, os.WriteFile("cli-metadata", []byte("blocked"), 0600))
			}
			generateMetadata(&cli.Command{Name: "aliyun", Short: i18n.T("CLI", "CLI")})
			if blocked {
				data, err := os.ReadFile("cli-metadata")
				require.NoError(t, err)
				require.Equal(t, "blocked", string(data))
			} else {
				require.FileExists(t, filepath.Join("cli-metadata", "commands.json"))
				require.NoFileExists(t, filepath.Join("cli-metadata", "metadatas", "products.json"))
			}
		})
	}
}

func TestDumpFilesDoesNotReplaceDestinationDirectoryWithFile(t *testing.T) {
	output := t.TempDir()
	blocked := filepath.Join(output, "testdata", "dump", "nested", "child.txt")
	require.NoError(t, os.MkdirAll(blocked, 0700))
	dumpFiles(testDumpFS, "testdata/dump/nested", output)
	require.DirExists(t, blocked)
}
