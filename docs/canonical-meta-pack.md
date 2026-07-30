# Canonical metadata bundle

The CLI stores Canonical OpenAPI metadata in a generated, random-access bundle
for release builds. The source JSON remains in the `aliyun-openapi-meta`
submodule. Development builds read that directory directly, while release
builds generate a temporary bundle under `bundledmeta/.generated` and embed it.

This separation is intentional:

- `aliyun-openapi-meta` remains the source of truth.
- Local development does not need a generated pack.
- The generated bundle is a single embedded file instead of about 31,000
  individually embedded JSON files.
- Commands that do not use Canonical metadata do not parse an index or create a
  decompressor.

## Generated files

`go generate ./bundledmeta` produces
`bundledmeta/.generated/canonical.pack`. It contains Canonical API JSON files,
`aliyun-openapi-meta/metadatas/products.json`, the binary index, the shared
Zstandard dictionary, and independently compressed data frames.

The generated pack is ignored by Git and exists only as a build artifact. It is
embedded when the `aliyun_cli_packed_meta` build tag is enabled.

## Generation

From the root of the main repository, run:

```sh
go generate ./bundledmeta
go test -tags aliyun_cli_packed_meta ./bundledmeta ./meta ./export ./openapi/runtimehost
go build -tags aliyun_cli_packed_meta ./main/main.go
```

The generator performs the following deterministic steps:

1. Walk `aliyun-openapi-meta/canonical`.
2. Add `metadatas/products.json` and convert Canonical paths to slash-separated
   `canonical/...` form.
3. Sort every embedded path lexicographically.
4. Build a 64 KiB shared dictionary from deterministic samples of the sorted
   JSON input.
5. Compress every file as an independent Zstandard frame using the shared
   dictionary.
6. Write the header, fixed-width binary index, path table, and compressed
   frames to `canonical.pack`.

Independent frames are required for random access. Reading one API uses its
index offset and decompresses only that API.

## Pack format

All integers are unsigned little-endian values. The format starts with a
24-byte header:

| Offset | Size | Value |
| ---: | ---: | --- |
| 0 | 8 | ASCII magic `ALIMETA1` |
| 8 | 4 | dictionary size |
| 12 | 4 | number of index entries |
| 16 | 4 | path table size |
| 20 | 4 | reserved, currently zero |

The header is followed by:

1. Zstandard dictionary bytes.
2. One 24-byte index entry per file.
3. Concatenated UTF-8 path bytes.
4. Concatenated independent Zstandard frames.

Each index entry contains:

| Field | Size | Meaning |
| --- | ---: | --- |
| path offset | 4 | Offset in the path table |
| path length | 4 | Path length in bytes |
| data offset | 8 | Offset in the compressed frame section |
| compressed size | 4 | Compressed frame length |
| raw size | 4 | Expected decompressed length |

Entries are sorted by path. Runtime lookup performs a binary search directly
over the embedded index bytes, so it does not deserialize JSON or allocate a
large path map.

## Updating metadata

To publish newer metadata in the CLI:

1. Update the `aliyun-openapi-meta` submodule to a commit available to everyone
   who builds the CLI.
2. Commit the updated submodule pointer in the main repository.
3. Run `make test` to verify development mode against the source JSON.
4. Run `make test-release` and `make build` to generate and verify the temporary
   packed metadata.

Do not point the main repository at a local-only metadata commit. Local changes
can be used to evaluate a bundle, but the committed submodule pointer must
resolve in the metadata remote.

## Runtime behavior

`bundledmeta.Metadatas` is exposed as an `fs.FS`. In development mode it wraps
`os.DirFS`; in packed builds it uses the binary pack reader. Existing Canonical
metadata consumers continue to use paths such as:

```text
canonical/ecs/2014-05-26/DescribeRegions.json
metadatas/products.json
```

The pack header and binary index are read lazily. The reusable Zstandard
decoder is initialized only when a Canonical JSON file is requested.
