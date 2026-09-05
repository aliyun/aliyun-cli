package bundledmeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"sort"
	"testing"

	"github.com/klauspost/compress/zstd"
)

type testPackSource struct {
	name string
	raw  []byte
}

type testPackEntry struct {
	pathOffset uint32
	pathLength uint32
	offset     uint64
	size       uint32
	rawSize    uint32
}

func buildTestMetadataPack(t *testing.T) []byte {
	t.Helper()
	files := []testPackSource{
		{name: "canonical/ecs/2014-05-26/DescribeRegions.json", raw: []byte(`{"name":"DescribeRegions"}`)},
		{name: "canonical/ecs/2014-05-26/RunInstances.json", raw: []byte(`{"name":"RunInstances"}`)},
		{name: "canonical/vpc/2016-04-28/DescribeVpcs.json", raw: []byte(`{"name":"DescribeVpcs"}`)},
		{name: "metadatas/products.json", raw: []byte(`[{"code":"ecs"},{"code":"vpc"}]`)},
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	dictionary := []byte(`{"name":"DescribeRegionsRunInstancesDescribeVpcs","code":"ecs-vpc"}`)
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderConcurrency(1),
		zstd.WithEncoderDictRaw(dictionaryID, dictionary),
	)
	if err != nil {
		t.Fatalf("create test encoder: %v", err)
	}
	defer encoder.Close()

	var paths bytes.Buffer
	var blob bytes.Buffer
	entries := make([]testPackEntry, 0, len(files))
	for _, file := range files {
		compressed := encoder.EncodeAll(file.raw, nil)
		entries = append(entries, testPackEntry{
			pathOffset: uint32(paths.Len()),
			pathLength: uint32(len(file.name)),
			offset:     uint64(blob.Len()),
			size:       uint32(len(compressed)),
			rawSize:    uint32(len(file.raw)),
		})
		_, _ = paths.WriteString(file.name)
		_, _ = blob.Write(compressed)
	}

	var pack bytes.Buffer
	_, _ = pack.WriteString(packMagic)
	writeTestUint32(&pack, uint32(len(dictionary)))
	writeTestUint32(&pack, uint32(len(entries)))
	writeTestUint32(&pack, uint32(paths.Len()))
	writeTestUint32(&pack, 0)
	_, _ = pack.Write(dictionary)
	for _, entry := range entries {
		writeTestUint32(&pack, entry.pathOffset)
		writeTestUint32(&pack, entry.pathLength)
		writeTestUint64(&pack, entry.offset)
		writeTestUint32(&pack, entry.size)
		writeTestUint32(&pack, entry.rawSize)
	}
	_, _ = pack.Write(paths.Bytes())
	_, _ = pack.Write(blob.Bytes())
	return pack.Bytes()
}

func writeTestUint32(dst *bytes.Buffer, value uint32) {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	_, _ = dst.Write(encoded[:])
}

func writeTestUint64(dst *bytes.Buffer, value uint64) {
	var encoded [8]byte
	binary.LittleEndian.PutUint64(encoded[:], value)
	_, _ = dst.Write(encoded[:])
}

func TestPackedMetadataFSFilesystemOperations(t *testing.T) {
	metadata := newPackedMetadataFS(buildTestMetadataPack(t))
	apiPath := "canonical/ecs/2014-05-26/DescribeRegions.json"

	data, err := fs.ReadFile(metadata, "/"+apiPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != `{"name":"DescribeRegions"}` {
		t.Fatalf("ReadFile data = %s", data)
	}
	if _, err := fs.ReadFile(metadata, "canonical/missing.json"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing ReadFile error = %v, want fs.ErrNotExist", err)
	}

	root, err := fs.ReadDir(metadata, "/")
	if err != nil {
		t.Fatalf("ReadDir root: %v", err)
	}
	if len(root) != 2 || root[0].Name() != "canonical" || root[1].Name() != "metadatas" || !root[0].IsDir() || !root[1].IsDir() {
		t.Fatalf("unexpected root entries: %#v", root)
	}
	version, err := fs.ReadDir(metadata, "canonical/ecs/2014-05-26")
	if err != nil {
		t.Fatalf("ReadDir version: %v", err)
	}
	if len(version) != 2 || version[0].Name() != "DescribeRegions.json" || version[1].Name() != "RunInstances.json" {
		t.Fatalf("unexpected version entries: %#v", version)
	}
	if _, err := fs.ReadDir(metadata, "canonical/does-not-exist"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing ReadDir error = %v, want fs.ErrNotExist", err)
	}

	for _, test := range []struct {
		name    string
		wantDir bool
		want    string
	}{
		{name: ".", wantDir: true, want: "."},
		{name: "canonical/ecs", wantDir: true, want: "ecs"},
		{name: apiPath, want: "DescribeRegions.json"},
	} {
		info, err := fs.Stat(metadata, test.name)
		if err != nil {
			t.Fatalf("Stat(%q): %v", test.name, err)
		}
		if info.Name() != test.want || info.IsDir() != test.wantDir {
			t.Fatalf("Stat(%q) = name %q dir %v", test.name, info.Name(), info.IsDir())
		}
	}
	if _, err := fs.Stat(metadata, "missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing Stat error = %v, want fs.ErrNotExist", err)
	}

	file, err := metadata.Open(apiPath)
	if err != nil {
		t.Fatalf("Open file: %v", err)
	}
	info, err := file.Stat()
	if err != nil || info.Mode().Perm() != 0444 || info.Size() != int64(len(data)) {
		t.Fatalf("file Stat = %#v, %v", info, err)
	}
	opened, err := io.ReadAll(file)
	if err != nil || !bytes.Equal(opened, data) {
		t.Fatalf("opened file = %q, %v", opened, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close file: %v", err)
	}

	directory, err := metadata.Open("canonical/ecs/2014-05-26")
	if err != nil {
		t.Fatalf("Open directory: %v", err)
	}
	dirInfo, err := directory.Stat()
	if err != nil || !dirInfo.IsDir() || dirInfo.Mode()&fs.ModeDir == 0 {
		t.Fatalf("directory Stat = %#v, %v", dirInfo, err)
	}
	if n, err := directory.Read(make([]byte, 1)); n != 0 || err == nil {
		t.Fatalf("directory Read = %d, %v", n, err)
	}
	readDir := directory.(fs.ReadDirFile)
	first, err := readDir.ReadDir(1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first ReadDir = %#v, %v", first, err)
	}
	remaining, err := readDir.ReadDir(10)
	if err != nil || len(remaining) != 1 {
		t.Fatalf("remaining ReadDir = %#v, %v", remaining, err)
	}
	if entries, err := readDir.ReadDir(1); len(entries) != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("exhausted ReadDir = %#v, %v", entries, err)
	}
	if err := directory.Close(); err != nil {
		t.Fatalf("Close directory: %v", err)
	}

	directory, err = metadata.Open("canonical/ecs/2014-05-26")
	if err != nil {
		t.Fatalf("reopen directory: %v", err)
	}
	all, err := directory.(fs.ReadDirFile).ReadDir(0)
	if err != nil || len(all) != 2 {
		t.Fatalf("ReadDir all = %#v, %v", all, err)
	}

	entryInfo, err := root[0].Info()
	if err != nil || entryInfo.Name() != "canonical" || root[0].Type()&fs.ModeDir == 0 {
		t.Fatalf("root entry info = %#v, %v", entryInfo, err)
	}
}

func TestPackedMetadataFSCorruptData(t *testing.T) {
	t.Run("invalid header", func(t *testing.T) {
		metadata := newPackedMetadataFS([]byte("not a pack"))
		if _, err := metadata.ReadFile("canonical/a.json"); err == nil {
			t.Fatal("invalid header was accepted")
		}
	})

	t.Run("invalid section offsets", func(t *testing.T) {
		pack := make([]byte, packHeaderSize)
		copy(pack, packMagic)
		binary.LittleEndian.PutUint32(pack[8:12], ^uint32(0))
		metadata := newPackedMetadataFS(pack)
		if _, err := metadata.ReadDir("canonical"); err == nil {
			t.Fatal("invalid section offsets were accepted")
		}
	})

	t.Run("invalid path range", func(t *testing.T) {
		pack := append([]byte(nil), buildTestMetadataPack(t)...)
		entryStart := packHeaderSize + int(binary.LittleEndian.Uint32(pack[8:12]))
		binary.LittleEndian.PutUint32(pack[entryStart:entryStart+4], ^uint32(0))
		metadata := newPackedMetadataFS(pack)
		if _, _, err := metadata.lookupFile("canonical/a.json"); err == nil {
			t.Fatal("invalid path range was accepted")
		}
	})

	t.Run("invalid compressed range", func(t *testing.T) {
		pack := append([]byte(nil), buildTestMetadataPack(t)...)
		entryStart := packHeaderSize + int(binary.LittleEndian.Uint32(pack[8:12]))
		binary.LittleEndian.PutUint64(pack[entryStart+8:entryStart+16], ^uint64(0))
		metadata := newPackedMetadataFS(pack)
		if _, err := metadata.ReadFile("canonical/ecs/2014-05-26/DescribeRegions.json"); err == nil {
			t.Fatal("invalid compressed range was accepted")
		}
	})

	t.Run("invalid compressed data", func(t *testing.T) {
		pack := append([]byte(nil), buildTestMetadataPack(t)...)
		blobStart := testPackBlobStart(pack)
		pack[blobStart] ^= 0xff
		metadata := newPackedMetadataFS(pack)
		if _, err := metadata.ReadFile("canonical/ecs/2014-05-26/DescribeRegions.json"); err == nil {
			t.Fatal("invalid compressed data was accepted")
		}
	})

	t.Run("raw size mismatch", func(t *testing.T) {
		pack := append([]byte(nil), buildTestMetadataPack(t)...)
		entryStart := packHeaderSize + int(binary.LittleEndian.Uint32(pack[8:12]))
		raw := binary.LittleEndian.Uint32(pack[entryStart+20 : entryStart+24])
		binary.LittleEndian.PutUint32(pack[entryStart+20:entryStart+24], raw+1)
		metadata := newPackedMetadataFS(pack)
		if _, err := metadata.ReadFile("canonical/ecs/2014-05-26/DescribeRegions.json"); err == nil {
			t.Fatal("raw size mismatch was accepted")
		}
	})
}

func TestPackedMetadataFSHelpers(t *testing.T) {
	for input, want := range map[string]string{
		"":           ".",
		"/":          ".",
		"/canonical": "canonical",
		"a/../b":     "b",
	} {
		if got := cleanName(input); got != want {
			t.Fatalf("cleanName(%q) = %q, want %q", input, got, want)
		}
	}

	metadata := newPackedMetadataFS(buildTestMetadataPack(t))
	layout, err := metadata.loadLayout()
	if err != nil {
		t.Fatalf("loadLayout: %v", err)
	}
	if _, ok := metadata.entryName(layout, -1); ok {
		t.Fatal("negative entryName index accepted")
	}
	if _, ok := metadata.entryName(layout, layout.count); ok {
		t.Fatal("out-of-range entryName index accepted")
	}
	if _, ok := metadata.entry(layout, -1); ok {
		t.Fatal("negative entry index accepted")
	}
	if _, ok := metadata.entry(layout, layout.count); ok {
		t.Fatal("out-of-range entry index accepted")
	}
	if info := (packedInfo{}); info.Name() != "." || info.ModTime().IsZero() == false || info.Sys() != nil {
		t.Fatalf("unexpected zero packedInfo: %#v", info)
	}
	fileEntry := packedDirEntry{name: "file.json"}
	if fileEntry.IsDir() || fileEntry.Type() != 0 {
		t.Fatalf("unexpected file entry: %#v", fileEntry)
	}
}

func testPackBlobStart(pack []byte) int {
	dictionarySize := int(binary.LittleEndian.Uint32(pack[8:12]))
	count := int(binary.LittleEndian.Uint32(pack[12:16]))
	pathsSize := int(binary.LittleEndian.Uint32(pack[16:20]))
	return packHeaderSize + dictionarySize + count*packEntrySize + pathsSize
}
