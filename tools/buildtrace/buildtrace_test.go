package buildtrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleInfo = `aliyun: go1.25.14
	path	github.com/aliyun/aliyun-cli/v3/main
	mod	github.com/aliyun/aliyun-cli/v3	(devel)
	dep	github.com/aliyun/aliyun-openapi-runtime	v0.0.0	h1:abc	=>	./aliyun-openapi-runtime	v0.0.0-local
	dep	example.com/mod	v1.2.3	h1:zzz	=>	./localmod
	dep	golang.org/x/sys	v0.47.0	h1:def
	build	vcs.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`

func TestWriteAndRetarget(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "aliyun-cli-linux-3.5.2-amd64.tgz")
	if err := os.WriteFile(asset, []byte("archive-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	opt := Options{
		Binary:    "unused",
		Asset:     asset,
		Version:   "3.5.2",
		Tag:       "v3.5.2",
		Commit:    strings.Repeat("a", 40),
		Gitlink:   strings.Repeat("b", 40),
		RunURL:    "https://github.com/aliyun/aliyun-cli/actions/runs/1",
		BuildInfo: sampleInfo,
	}
	if err := Write(opt); err != nil {
		t.Fatal(err)
	}
	sbomRaw, err := os.ReadFile(asset + ".cdx.json")
	if err != nil {
		t.Fatal(err)
	}
	var sbom bom
	if err := json.Unmarshal(sbomRaw, &sbom); err != nil {
		t.Fatal(err)
	}
	if sbom.BOMFormat != "CycloneDX" || sbom.Metadata.Component.Version != "3.5.2" {
		t.Fatalf("sbom = %+v", sbom.Metadata.Component)
	}
	if sbom.Metadata.Properties[0].Value != "aliyun-cli-linux-3.5.2-amd64.tgz" {
		t.Fatal(sbom.Metadata.Properties)
	}
	if SBOMName("a.tgz") != "a.tgz.cdx.json" || ProvenanceName("a.tgz") != "a.tgz.provenance.json" {
		t.Fatal("sidecar names")
	}
	if len(sbom.Components) != 3 || sbom.Components[0].Version != "v0.0.0-local" || sbom.Components[1].Version != "v1.2.3" {
		t.Fatalf("components = %+v", sbom.Components)
	}
	if sbom.Metadata.Tools.Components[0].Name != "go" || sbom.Metadata.Tools.Components[0].Version != "go1.25.14" {
		t.Fatalf("tools = %+v", sbom.Metadata.Tools)
	}

	var doc Provenance
	raw, err := os.ReadFile(asset + ".provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA(asset)
	if err != nil {
		t.Fatal(err)
	}
	if doc.SHA256 != sum || doc.Commit != opt.Commit || doc.Submodule.Gitlink != opt.Gitlink || doc.Tag != "v3.5.2" {
		t.Fatalf("provenance = %+v", doc)
	}
	if doc.SBOM.File != "aliyun-cli-linux-3.5.2-amd64.tgz.cdx.json" {
		t.Fatal(doc.SBOM)
	}

	latest := filepath.Join(dir, "aliyun-cli-linux-latest-amd64.tgz")
	if err := os.WriteFile(latest, []byte("archive-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	latestSBOM := latest + ".cdx.json"
	if err := os.WriteFile(latestSBOM, sbomRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Retarget(asset+".provenance.json", latest, latestSBOM); err != nil {
		t.Fatal(err)
	}
	var retargeted Provenance
	body, err := os.ReadFile(latest + ".provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &retargeted); err != nil {
		t.Fatal(err)
	}
	latestSum, err := fileSHA(latestSBOM)
	if err != nil {
		t.Fatal(err)
	}
	if retargeted.Asset != "aliyun-cli-linux-latest-amd64.tgz" || retargeted.Commit != opt.Commit || retargeted.SHA256 != sum || retargeted.SBOM.SHA256 != latestSum {
		t.Fatalf("retargeted = %+v", retargeted)
	}
	var latestBOM bom
	if err := json.Unmarshal(mustRead(t, latestSBOM), &latestBOM); err != nil {
		t.Fatal(err)
	}
	if latestBOM.Metadata.Properties[0].Value != "aliyun-cli-linux-latest-amd64.tgz" {
		t.Fatal(latestBOM.Metadata.Properties)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestWriteRejectsBadIdentity(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "a.tgz")
	if err := os.WriteFile(asset, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	opt := Options{
		Binary: "b", Asset: asset, Version: "3.5.2", Tag: "v3.5.2",
		Commit: strings.Repeat("a", 40), Gitlink: strings.Repeat("b", 40),
		RunURL:    "https://github.com/aliyun/aliyun-cli/actions/runs/1",
		BuildInfo: sampleInfo,
	}
	opt.Tag = "3.5.2"
	if err := Write(opt); err == nil {
		t.Fatal("expected tag mismatch to fail")
	}
	opt.Tag = "v3.5.2"
	opt.Commit = "abc"
	if err := Write(opt); err == nil {
		t.Fatal("expected short commit to fail")
	}
	opt.Commit = strings.Repeat("a", 40)
	opt.BuildInfo = "not-build-info"
	if err := Write(opt); err == nil {
		t.Fatal("expected bad build info to fail")
	}
	if err := Retarget(filepath.Join(dir, "missing.json"), asset, asset); err == nil {
		t.Fatal("expected missing provenance to fail")
	}
}

func TestNamesAndFailurePaths(t *testing.T) {
	if SBOMName("a.tgz") != "a.tgz.cdx.json" || ProvenanceName("a.tgz") != "a.tgz.provenance.json" {
		t.Fatal("sidecar names")
	}
	dir := t.TempDir()
	asset := filepath.Join(dir, "a.tgz")
	if err := os.WriteFile(asset, []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := Options{
		Binary: "b", Asset: asset, Version: "3.5.2", Tag: "v3.5.2",
		Commit: strings.Repeat("a", 40), Gitlink: strings.Repeat("b", 40),
		RunURL:    "https://github.com/aliyun/aliyun-cli/actions/runs/1",
		BuildInfo: sampleInfo,
	}
	base.Binary = ""
	if err := Write(base); err == nil {
		t.Fatal("expected missing binary to fail")
	}
	base.Binary = "b"
	base.RunURL = "http://example.invalid/run"
	if err := Write(base); err == nil {
		t.Fatal("expected insecure run URL to fail")
	}
	base.RunURL = "https://github.com/aliyun/aliyun-cli/actions/runs/1"
	base.Commit = strings.Repeat("A", 40)
	if err := Write(base); err == nil {
		t.Fatal("expected uppercase commit to fail")
	}
	base.Commit = strings.Repeat("a", 40)
	base.Asset = filepath.Join(dir, "missing.tgz")
	if err := Write(base); err == nil {
		t.Fatal("expected missing asset to fail")
	}
	empty := filepath.Join(dir, "empty.tgz")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	base.Asset = empty
	if err := Write(base); err == nil {
		t.Fatal("expected empty asset to fail")
	}
	base.Asset = asset
	base.BuildInfo = "\n"
	if err := Write(base); err == nil || !strings.Contains(err.Error(), "empty build info") {
		t.Fatalf("empty build info: %v", err)
	}
	base.BuildInfo = "tool: go1.25.14\n\tdep\tgolang.org/x/sys\tv0.1.0\th1:abc\n"
	if err := Write(base); err == nil || !strings.Contains(err.Error(), "no main module") {
		t.Fatalf("missing module: %v", err)
	}
	base.BuildInfo = ""
	base.Binary = filepath.Join(dir, "not-a-binary")
	if err := Write(base); err == nil || !strings.Contains(err.Error(), "go version -m") {
		t.Fatalf("go version -m: %v", err)
	}
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	base.Binary = bin
	if err := Write(base); err != nil {
		t.Fatal(err)
	}

	prov := asset + ".provenance.json"
	if err := os.WriteFile(prov, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Retarget(prov, asset, asset+".cdx.json"); err == nil {
		t.Fatal("expected broken provenance to fail")
	}
	if err := os.WriteFile(prov, []byte(`{"format":"nope"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Retarget(prov, asset, asset+".cdx.json"); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("incomplete provenance: %v", err)
	}

	if err := Write(Options{
		Binary: "b", Asset: asset, Version: "3.5.2", Tag: "v3.5.2",
		Commit: strings.Repeat("a", 40), Gitlink: strings.Repeat("b", 40),
		RunURL:    "https://github.com/aliyun/aliyun-cli/actions/runs/1",
		BuildInfo: sampleInfo,
	}); err != nil {
		t.Fatal(err)
	}
	if err := Retarget(prov, asset, filepath.Join(dir, "missing.cdx.json")); err == nil {
		t.Fatal("expected missing sbom to fail")
	}
	if err := os.WriteFile(asset+".cdx.json", []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Retarget(prov, asset, asset+".cdx.json"); err == nil || !strings.Contains(err.Error(), "sbom") {
		t.Fatalf("broken sbom: %v", err)
	}
	plain := filepath.Join(dir, "plain.json")
	if err := os.WriteFile(plain, []byte(`{"metadata":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Retarget(prov, asset, plain); err != nil {
		t.Fatal(err)
	}
	missingAsset := filepath.Join(dir, "gone.tgz")
	if err := Retarget(prov, missingAsset, plain); err == nil {
		t.Fatal("expected missing archive digest to fail")
	}
	if _, err := fileSHA(dir); err == nil {
		t.Fatal("expected directory digest to fail")
	}
	if _, err := fileSHA(filepath.Join(dir, "nope")); err == nil {
		t.Fatal("expected missing digest to fail")
	}
	if _, err := fileSHA(empty); err == nil {
		t.Fatal("expected empty digest to fail")
	}
}
