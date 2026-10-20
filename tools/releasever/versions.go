package releasever

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

type versionRecord struct {
	Version          string `json:"version"`
	SignatureVersion string `json:"signature_version"`
	Description      string `json:"description"`
}

type versionCatalog struct {
	Format   string          `json:"format"`
	Versions []versionRecord `json:"versions"`
}

// AppendStableVersion records one stable release in the JSON history.
// The signature version is the publisher generation that signed the release.
// Existing entries stay in order. Publishing the same version again updates
// its signature note in place.
func AppendStableVersion(existing []byte, raw, signatureVersion string) ([]byte, error) {
	kind, err := Classify(raw)
	if err != nil {
		return nil, err
	}
	if kind != KindStable {
		return nil, errPrerelease(raw)
	}
	canonical, err := canonicalVersion(raw)
	if err != nil {
		return nil, err
	}
	if !validSignatureVersion(signatureVersion) {
		return nil, fmt.Errorf("invalid signature version %q", signatureVersion)
	}
	version := strings.TrimPrefix(canonical, "v")
	record := versionRecord{
		Version:          version,
		SignatureVersion: signatureVersion,
		Description:      publishsig.DescribeSignatureVersion(signatureVersion),
	}

	catalog, err := readCatalog(existing)
	if err != nil {
		return nil, err
	}
	replaced := false
	for i := range catalog.Versions {
		if catalog.Versions[i].Version == version {
			catalog.Versions[i] = record
			replaced = true
			break
		}
	}
	if !replaced {
		catalog.Versions = append(catalog.Versions, record)
	}
	body, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func readCatalog(existing []byte) (versionCatalog, error) {
	if len(bytes.TrimSpace(existing)) == 0 {
		return versionCatalog{Format: publishsig.VersionsFormat, Versions: []versionRecord{}}, nil
	}
	var catalog versionCatalog
	if err := json.Unmarshal(existing, &catalog); err != nil {
		return versionCatalog{}, fmt.Errorf("versions file: %w", err)
	}
	if catalog.Format != publishsig.VersionsFormat {
		return versionCatalog{}, fmt.Errorf("versions file: unsupported format %q", catalog.Format)
	}
	if catalog.Versions == nil {
		catalog.Versions = []versionRecord{}
	}
	return catalog, nil
}

func validSignatureVersion(version string) bool {
	if version == "" || len(version) > 64 {
		return false
	}
	for _, r := range version {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func errPrerelease(raw string) error {
	return &prereleaseError{raw: raw}
}

type prereleaseError struct {
	raw string
}

func (e *prereleaseError) Error() string {
	return "refuse to record prerelease " + e.raw
}
