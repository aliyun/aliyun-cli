package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/klauspost/compress/zstd"
)

const (
	packMagic     = "ALIMETA1"
	headerSize    = 24
	entrySize     = 24
	dictionaryID  = 1
	dictionaryMax = 64 << 10
)

type sourceFile struct {
	name string
	raw  []byte
}

type packedFile struct {
	pathOffset     uint32
	pathLength     uint32
	dataOffset     uint64
	compressedSize uint32
	rawSize        uint32
}

func main() {
	canonicalDir := flag.String("canonical", "", "canonical metadata directory")
	productsFile := flag.String("products", "", "products metadata file")
	outDir := flag.String("out", "packed", "output directory")
	flag.Parse()
	if *canonicalDir == "" || *productsFile == "" {
		fatal(fmt.Errorf("-canonical and -products are required"))
	}

	files, err := readFiles(*canonicalDir)
	if err != nil {
		fatal(err)
	}
	products, err := os.ReadFile(*productsFile)
	if err != nil {
		fatal(err)
	}
	files = append(files, sourceFile{name: "metadatas/products.json", raw: products})
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	dictionary := buildDictionary(files)
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(1),
		zstd.WithEncoderDictRaw(dictionaryID, dictionary),
	)
	if err != nil {
		fatal(err)
	}
	defer encoder.Close()

	var paths bytes.Buffer
	var blob bytes.Buffer
	entries := make([]packedFile, 0, len(files))
	for _, file := range files {
		compressed := encoder.EncodeAll(file.raw, nil)
		entries = append(entries, packedFile{
			pathOffset:     uint32(paths.Len()),
			pathLength:     uint32(len(file.name)),
			dataOffset:     uint64(blob.Len()),
			compressedSize: uint32(len(compressed)),
			rawSize:        uint32(len(file.raw)),
		})
		_, _ = paths.WriteString(file.name)
		_, _ = blob.Write(compressed)
	}

	var pack bytes.Buffer
	pack.Grow(headerSize + len(dictionary) + len(entries)*entrySize + paths.Len() + blob.Len())
	_, _ = pack.WriteString(packMagic)
	writeUint32(&pack, uint32(len(dictionary)))
	writeUint32(&pack, uint32(len(entries)))
	writeUint32(&pack, uint32(paths.Len()))
	writeUint32(&pack, 0)
	_, _ = pack.Write(dictionary)
	for _, entry := range entries {
		writeUint32(&pack, entry.pathOffset)
		writeUint32(&pack, entry.pathLength)
		writeUint64(&pack, entry.dataOffset)
		writeUint32(&pack, entry.compressedSize)
		writeUint32(&pack, entry.rawSize)
	}
	_, _ = pack.Write(paths.Bytes())
	_, _ = pack.Write(blob.Bytes())

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	packOutput := filepath.Join(*outDir, "canonical.pack")
	if err := os.WriteFile(packOutput, pack.Bytes(), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("packed %d files into %s (%s dictionary, %s paths, %s blob, %s total)\n",
		len(files), packOutput, human(len(dictionary)), human(paths.Len()), human(blob.Len()), human(pack.Len()))
}

func readFiles(canonicalDir string) ([]sourceFile, error) {
	files := make([]sourceFile, 0, 32_000)
	err := filepath.WalkDir(canonicalDir, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(canonicalDir, name)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files = append(files, sourceFile{
			name: path.Join("canonical", filepath.ToSlash(rel)),
			raw:  raw,
		})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, err
}

func buildDictionary(files []sourceFile) []byte {
	var dictionary []byte
	for i := 0; i < len(files) && len(dictionary) < dictionaryMax*5; i += 97 {
		sample := files[i].raw
		if len(sample) > 1024 {
			sample = sample[:1024]
		}
		dictionary = append(dictionary, sample...)
	}
	if len(dictionary) > dictionaryMax {
		dictionary = dictionary[len(dictionary)-dictionaryMax:]
	}
	return dictionary
}

func writeUint32(dst *bytes.Buffer, value uint32) {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	_, _ = dst.Write(encoded[:])
}

func writeUint64(dst *bytes.Buffer, value uint64) {
	var encoded [8]byte
	binary.LittleEndian.PutUint64(encoded[:], value)
	_, _ = dst.Write(encoded[:])
}

func human(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := unit, 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
