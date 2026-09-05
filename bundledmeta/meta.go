package bundledmeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

// Metadatas exposes the generated metadata bundle through the filesystem API
// expected by canonicalmeta and the OpenAPI runtime. Development builds read
// the source metadata directory, while release builds embed a generated pack.
var Metadatas fs.FS = newMetadataFS()

const (
	packMagic      = "ALIMETA1"
	packHeaderSize = 24
	packEntrySize  = 24
	dictionaryID   = 1
)

type packedMetadataFS struct {
	packData []byte

	layoutOnce sync.Once
	layout     packedLayout
	layoutErr  error

	decoderOnce sync.Once
	decoder     *zstd.Decoder
	decoderErr  error
}

type packedLayout struct {
	dictionary []byte
	entries    []byte
	paths      []byte
	blob       []byte
	count      int
}

type packedFile struct {
	offset uint64
	size   uint32
	raw    uint32
}

func newPackedMetadataFS(packData []byte) *packedMetadataFS {
	return &packedMetadataFS{packData: packData}
}

func (p *packedMetadataFS) Open(name string) (fs.File, error) {
	name = cleanName(name)
	if p.isDir(name) {
		entries, err := p.ReadDir(name)
		if err != nil {
			return nil, err
		}
		return &packedDir{name: name, entries: entries}, nil
	}
	data, err := p.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return &packedFileHandle{
		Reader: bytes.NewReader(data),
		info:   packedInfo{name: path.Base(name), size: int64(len(data))},
	}, nil
}

func (p *packedMetadataFS) ReadFile(name string) ([]byte, error) {
	name = cleanName(name)
	entry, ok, err := p.lookupFile(name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fs.ErrNotExist
	}
	layout, err := p.loadLayout()
	if err != nil {
		return nil, err
	}
	end := entry.offset + uint64(entry.size)
	if end < entry.offset || end > uint64(len(layout.blob)) {
		return nil, fmt.Errorf("packed canonical entry %s has invalid range", name)
	}
	decoder, err := p.loadDecoder()
	if err != nil {
		return nil, fmt.Errorf("initialize canonical decoder failed: %w", err)
	}
	compressed := layout.blob[int(entry.offset):int(end)]
	data, err := decoder.DecodeAll(compressed, make([]byte, 0, int(entry.raw)))
	if err != nil {
		return nil, fmt.Errorf("decompress packed canonical %s failed: %w", name, err)
	}
	if len(data) != int(entry.raw) {
		return nil, fmt.Errorf("decompressed canonical %s has size %d, want %d", name, len(data), entry.raw)
	}
	return data, nil
}

func (p *packedMetadataFS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = cleanName(name)
	if name == "." {
		return []fs.DirEntry{
			packedDirEntry{name: "canonical", isDir: true},
			packedDirEntry{name: "metadatas", isDir: true},
		}, nil
	}
	layout, err := p.loadLayout()
	if err != nil {
		return nil, err
	}
	if !p.isDir(name) {
		return nil, fs.ErrNotExist
	}

	prefix := name + "/"
	start := p.lowerBound(layout, prefix)
	entries := make([]fs.DirEntry, 0, 64)
	seen := ""
	for i := start; i < layout.count; i++ {
		fullName, ok := p.entryName(layout, i)
		if !ok || !strings.HasPrefix(fullName, prefix) {
			break
		}
		remainder := strings.TrimPrefix(fullName, prefix)
		child, rest, hasRest := strings.Cut(remainder, "/")
		if child == seen {
			continue
		}
		seen = child
		entries = append(entries, packedDirEntry{name: child, isDir: hasRest && rest != ""})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (p *packedMetadataFS) Stat(name string) (fs.FileInfo, error) {
	name = cleanName(name)
	if name == "." {
		return packedInfo{name: ".", dir: true}, nil
	}
	if entry, ok, err := p.lookupFile(name); err != nil {
		return nil, err
	} else if ok {
		return packedInfo{name: path.Base(name), size: int64(entry.raw)}, nil
	}
	if p.isDir(name) {
		return packedInfo{name: path.Base(name), dir: true}, nil
	}
	return nil, fs.ErrNotExist
}

func (p *packedMetadataFS) lookupFile(name string) (packedFile, bool, error) {
	layout, err := p.loadLayout()
	if err != nil {
		return packedFile{}, false, err
	}
	index := p.lowerBound(layout, name)
	if index >= layout.count {
		return packedFile{}, false, nil
	}
	entryName, ok := p.entryName(layout, index)
	if !ok {
		return packedFile{}, false, errors.New("canonical pack contains an invalid path range")
	}
	if entryName != name {
		return packedFile{}, false, nil
	}
	entry, ok := p.entry(layout, index)
	if !ok {
		return packedFile{}, false, errors.New("canonical pack contains an invalid entry")
	}
	return entry, true, nil
}

func (p *packedMetadataFS) isDir(name string) bool {
	if name == "." || name == "canonical" {
		return true
	}
	layout, err := p.loadLayout()
	if err != nil {
		return false
	}
	prefix := name + "/"
	index := p.lowerBound(layout, prefix)
	if index >= layout.count {
		return false
	}
	entryName, ok := p.entryName(layout, index)
	return ok && strings.HasPrefix(entryName, prefix)
}

func (p *packedMetadataFS) loadLayout() (*packedLayout, error) {
	p.layoutOnce.Do(func() {
		if len(p.packData) < packHeaderSize || string(p.packData[:8]) != packMagic {
			p.layoutErr = errors.New("canonical pack has an invalid header")
			return
		}
		dictionarySize := uint64(binary.LittleEndian.Uint32(p.packData[8:12]))
		count := uint64(binary.LittleEndian.Uint32(p.packData[12:16]))
		pathsSize := uint64(binary.LittleEndian.Uint32(p.packData[16:20]))
		entriesStart := uint64(packHeaderSize) + dictionarySize
		pathsStart := entriesStart + count*packEntrySize
		blobStart := pathsStart + pathsSize
		if entriesStart < dictionarySize || pathsStart < entriesStart || blobStart < pathsStart || blobStart > uint64(len(p.packData)) {
			p.layoutErr = errors.New("canonical pack has invalid section offsets")
			return
		}
		p.layout = packedLayout{
			dictionary: p.packData[packHeaderSize:entriesStart],
			entries:    p.packData[entriesStart:pathsStart],
			paths:      p.packData[pathsStart:blobStart],
			blob:       p.packData[blobStart:],
			count:      int(count),
		}
	})
	if p.layoutErr != nil {
		return nil, p.layoutErr
	}
	return &p.layout, nil
}

func (p *packedMetadataFS) loadDecoder() (*zstd.Decoder, error) {
	layout, err := p.loadLayout()
	if err != nil {
		return nil, err
	}
	p.decoderOnce.Do(func() {
		p.decoder, p.decoderErr = zstd.NewReader(nil,
			zstd.WithDecoderConcurrency(1),
			zstd.WithDecoderDictRaw(dictionaryID, layout.dictionary),
		)
	})
	return p.decoder, p.decoderErr
}

func (p *packedMetadataFS) lowerBound(layout *packedLayout, name string) int {
	return sort.Search(layout.count, func(index int) bool {
		entryName, ok := p.entryName(layout, index)
		return !ok || entryName >= name
	})
}

func (p *packedMetadataFS) entryName(layout *packedLayout, index int) (string, bool) {
	start := index * packEntrySize
	if index < 0 || start < 0 || start+packEntrySize > len(layout.entries) {
		return "", false
	}
	entry := layout.entries[start : start+packEntrySize]
	pathOffset := uint64(binary.LittleEndian.Uint32(entry[0:4]))
	pathLength := uint64(binary.LittleEndian.Uint32(entry[4:8]))
	pathEnd := pathOffset + pathLength
	if pathEnd < pathOffset || pathEnd > uint64(len(layout.paths)) {
		return "", false
	}
	return string(layout.paths[pathOffset:pathEnd]), true
}

func (p *packedMetadataFS) entry(layout *packedLayout, index int) (packedFile, bool) {
	start := index * packEntrySize
	if index < 0 || start < 0 || start+packEntrySize > len(layout.entries) {
		return packedFile{}, false
	}
	entry := layout.entries[start : start+packEntrySize]
	return packedFile{
		offset: binary.LittleEndian.Uint64(entry[8:16]),
		size:   binary.LittleEndian.Uint32(entry[16:20]),
		raw:    binary.LittleEndian.Uint32(entry[20:24]),
	}, true
}

func cleanName(name string) string {
	name = strings.TrimPrefix(path.Clean(strings.TrimPrefix(name, "/")), "/")
	if name == "" {
		return "."
	}
	return name
}

type packedFileHandle struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *packedFileHandle) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *packedFileHandle) Close() error               { return nil }

type packedDir struct {
	name    string
	offset  int
	entries []fs.DirEntry
}

func (d *packedDir) Stat() (fs.FileInfo, error) {
	return packedInfo{name: path.Base(d.name), dir: true}, nil
}

func (d *packedDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: errors.New("is a directory")}
}
func (d *packedDir) Close() error { return nil }

func (d *packedDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.offset >= len(d.entries) && n > 0 {
		return nil, io.EOF
	}
	if n <= 0 || d.offset+n > len(d.entries) {
		n = len(d.entries) - d.offset
	}
	out := d.entries[d.offset : d.offset+n]
	d.offset += n
	return out, nil
}

type packedDirEntry struct {
	name  string
	isDir bool
}

func (e packedDirEntry) Name() string { return e.name }
func (e packedDirEntry) IsDir() bool  { return e.isDir }
func (e packedDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}
func (e packedDirEntry) Info() (fs.FileInfo, error) {
	return packedInfo{name: e.name, dir: e.isDir}, nil
}

type packedInfo struct {
	name string
	size int64
	dir  bool
}

func (i packedInfo) Name() string {
	if i.name == "." || i.name == "" {
		return "."
	}
	return i.name
}
func (i packedInfo) Size() int64 { return i.size }
func (i packedInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0555
	}
	return 0444
}
func (i packedInfo) ModTime() time.Time { return time.Time{} }
func (i packedInfo) IsDir() bool        { return i.dir }
func (i packedInfo) Sys() interface{}   { return nil }
