// Package buildtrace writes the release trace files for one archive:
// a CycloneDX SBOM taken from the binary's Go build info, and a provenance
// document that binds that archive's digest to the tag, commit, and submodule.
package buildtrace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	formatID       = "aliyun-cli-build-provenance v1"
	submodulePath  = "aliyun-openapi-meta"
	cycloneVersion = "1.5"
)

// marshalIndent and digestFile are the JSON and digest steps. Tests replace them.
var (
	marshalIndent = json.MarshalIndent
	digestFile    = fileSHA
)

// Options describes one published archive.
type Options struct {
	Binary    string
	Asset     string
	Version   string
	Tag       string
	Commit    string
	Gitlink   string
	RunURL    string
	BuildInfo string // when set, used instead of running `go version -m`
}

// Provenance is the JSON document stored next to an archive.
type Provenance struct {
	Format      string `json:"format"`
	Asset       string `json:"asset"`
	SHA256      string `json:"sha256"`
	Tag         string `json:"tag"`
	Commit      string `json:"commit"`
	Submodule   Source `json:"submodule"`
	WorkflowRun string `json:"workflowRun"`
	SBOM        Digest `json:"sbom"`
}

// Source identifies the metadata submodule recorded in the release commit.
type Source struct {
	Path    string `json:"path"`
	Gitlink string `json:"gitlink"`
}

// Digest names a sidecar file and its SHA-256.
type Digest struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

// SBOMName is the CycloneDX file published for an archive.
func SBOMName(assetName string) string {
	return assetName + ".cdx.json"
}

// ProvenanceName is the build-source file published for an archive.
func ProvenanceName(assetName string) string {
	return assetName + ".provenance.json"
}

// Write reads the binary and archive, then writes the SBOM and provenance
// beside the archive.
func Write(opt Options) error {
	if err := validate(opt); err != nil {
		return err
	}
	info := opt.BuildInfo
	if info == "" {
		out, err := exec.Command("go", "version", "-m", opt.Binary).CombinedOutput()
		if err != nil {
			return fmt.Errorf("buildtrace: go version -m: %w\n%s", err, out)
		}
		info = string(out)
	}
	sbom, err := cyclonedx(info, filepath.Base(opt.Asset), opt.Version)
	if err != nil {
		return err
	}
	sbomPath := opt.Asset + ".cdx.json"
	if err := os.WriteFile(sbomPath, sbom, 0o644); err != nil {
		return err
	}
	return writeProvenance(opt, sbomPath)
}

// Retarget copies the source identity from an existing provenance onto a
// renamed archive, recomputing both digests. latest aliases use this because
// the signed asset name changed.
func Retarget(from, assetPath, sbomPath string) error {
	raw, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	var doc Provenance
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("buildtrace: provenance: %w", err)
	}
	if doc.Format != formatID || !validSHA(doc.Commit) || doc.Submodule.Path != submodulePath || !validSHA(doc.Submodule.Gitlink) || doc.Tag == "" || !strings.HasPrefix(doc.WorkflowRun, "https://") {
		return fmt.Errorf("buildtrace: provenance %s is incomplete", from)
	}
	if err := retargetSBOM(sbomPath, filepath.Base(assetPath)); err != nil {
		return err
	}
	assetSum, err := fileSHA(assetPath)
	if err != nil {
		return err
	}
	sbomSum, err := digestFile(sbomPath)
	if err != nil {
		return err
	}
	doc.Asset = filepath.Base(assetPath)
	doc.SHA256 = assetSum
	doc.SBOM = Digest{File: filepath.Base(sbomPath), SHA256: sbomSum}
	body, err := marshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(assetPath+".provenance.json", append(body, '\n'), 0o644)
}

func retargetSBOM(path, assetName string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc bom
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("buildtrace: sbom: %w", err)
	}
	updated := false
	for i := range doc.Metadata.Properties {
		if doc.Metadata.Properties[i].Name == "aliyun-cli:asset" {
			doc.Metadata.Properties[i].Value = assetName
			updated = true
		}
	}
	if !updated {
		return nil
	}
	body, err := marshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func writeProvenance(opt Options, sbomPath string) error {
	assetSum, err := fileSHA(opt.Asset)
	if err != nil {
		return err
	}
	sbomSum, err := fileSHA(sbomPath)
	if err != nil {
		return err
	}
	doc := Provenance{
		Format:      formatID,
		Asset:       filepath.Base(opt.Asset),
		SHA256:      assetSum,
		Tag:         opt.Tag,
		Commit:      opt.Commit,
		Submodule:   Source{Path: submodulePath, Gitlink: opt.Gitlink},
		WorkflowRun: opt.RunURL,
		SBOM:        Digest{File: filepath.Base(sbomPath), SHA256: sbomSum},
	}
	body, err := marshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(opt.Asset+".provenance.json", append(body, '\n'), 0o644)
}

func validate(opt Options) error {
	if opt.Binary == "" || opt.Asset == "" || opt.Version == "" {
		return fmt.Errorf("buildtrace: binary, asset, and version are required")
	}
	if !strings.HasPrefix(opt.Tag, "v") || strings.TrimPrefix(opt.Tag, "v") != opt.Version {
		return fmt.Errorf("buildtrace: tag %q does not match version %q", opt.Tag, opt.Version)
	}
	if !validSHA(opt.Commit) || !validSHA(opt.Gitlink) {
		return fmt.Errorf("buildtrace: commit and gitlink must be full SHAs")
	}
	if !strings.HasPrefix(opt.RunURL, "https://") {
		return fmt.Errorf("buildtrace: workflow run URL is required")
	}
	info, err := os.Stat(opt.Asset)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("buildtrace: asset is missing or empty")
	}
	return nil
}

func cyclonedx(buildInfo, assetName, version string) ([]byte, error) {
	mods, goVersion, err := parseBuildInfo(buildInfo)
	if err != nil {
		return nil, err
	}
	components := make([]component, 0, len(mods.deps))
	for _, dep := range mods.deps {
		components = append(components, component{
			Type:    "library",
			Name:    dep.path,
			Version: dep.version,
			PURL:    "pkg:golang/" + dep.path + "@" + dep.version,
			Scope:   "required",
		})
	}
	doc := bom{
		BOMFormat:   "CycloneDX",
		SpecVersion: cycloneVersion,
		Version:     1,
		Metadata: metadata{
			Component: component{
				Type:    "application",
				Name:    mods.mainPath,
				Version: version,
				PURL:    "pkg:golang/" + mods.mainPath + "@" + version,
			},
			Tools: tools{Components: []component{{
				Type:    "application",
				Name:    "go",
				Version: goVersion,
			}}},
			Properties: []property{{
				Name:  "aliyun-cli:asset",
				Value: assetName,
			}},
		},
		Components: components,
	}
	body, err := marshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

type moduleLine struct {
	path    string
	version string
}

type parsedModules struct {
	mainPath string
	deps     []moduleLine
}

func parseBuildInfo(text string) (parsedModules, string, error) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return parsedModules{}, "", fmt.Errorf("buildtrace: empty build info")
	}
	goVersion := ""
	if _, ver, ok := strings.Cut(lines[0], ": "); ok {
		goVersion = strings.TrimSpace(ver)
	}
	if goVersion == "" {
		return parsedModules{}, "", fmt.Errorf("buildtrace: build info has no Go version")
	}
	var parsed parsedModules
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) > 0 && fields[0] == "" {
			fields = fields[1:]
		}
		if len(fields) < 3 {
			continue
		}
		kind := strings.TrimSpace(fields[0])
		switch kind {
		case "mod":
			parsed.mainPath = fields[1]
		case "dep":
			version := fields[2]
			// replace lines look like: dep path ver sum => replacePath replaceVer
			// A replace without a version keeps the original module version.
			if len(fields) >= 7 && fields[4] == "=>" && fields[6] != "" {
				version = fields[6]
			}
			parsed.deps = append(parsed.deps, moduleLine{path: fields[1], version: version})
		}
	}
	if parsed.mainPath == "" {
		return parsedModules{}, "", fmt.Errorf("buildtrace: build info has no main module")
	}
	return parsed, goVersion, nil
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("buildtrace: %s is empty", filepath.Base(path))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

type bom struct {
	BOMFormat   string      `json:"bomFormat"`
	SpecVersion string      `json:"specVersion"`
	Version     int         `json:"version"`
	Metadata    metadata    `json:"metadata"`
	Components  []component `json:"components"`
}

type metadata struct {
	Component  component  `json:"component"`
	Tools      tools      `json:"tools"`
	Properties []property `json:"properties"`
}

type tools struct {
	Components []component `json:"components"`
}

type component struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
	PURL    string `json:"purl"`
	Scope   string `json:"scope,omitempty"`
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
