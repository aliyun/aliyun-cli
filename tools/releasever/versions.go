package releasever

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
	"golang.org/x/mod/semver"
)

// marshalIndent encodes a catalog. Tests replace it.
var marshalIndent = json.MarshalIndent

type versionRecord struct {
	Version     string   `json:"version"`
	PublicKeys  []string `json:"public_keys"`
	Description string   `json:"description"`
}

type versionCatalog struct {
	Format   string          `json:"format"`
	Versions []versionRecord `json:"versions"`
}

// AppendStableVersion records one stable release in the JSON history.
// publicKeys is a comma-separated list of the publisher key generations that
// release supports. Existing entries stay in order. Publishing the same
// version again updates that entry in place.
func AppendStableVersion(existing []byte, raw, publicKeys string) ([]byte, error) {
	canonical, err := canonicalVersion(raw)
	if err != nil {
		return nil, err
	}
	if semver.Prerelease(canonical) != "" {
		return nil, errPrerelease(raw)
	}
	keys, err := parsePublicKeyVersions(publicKeys)
	if err != nil {
		return nil, err
	}
	version := strings.TrimPrefix(canonical, "v")
	record := versionRecord{
		Version:     version,
		PublicKeys:  keys,
		Description: publishsig.DescribePublicKeys(keys),
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
	body, err := marshalIndent(catalog, "", "  ")
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

func parsePublicKeyVersions(raw string) ([]string, error) {
	seen := make(map[string]bool)
	var keys []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !validSignatureVersion(part) {
			return nil, fmt.Errorf("invalid public key version %q", part)
		}
		if seen[part] {
			continue
		}
		seen[part] = true
		keys = append(keys, part)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("public key versions are required")
	}
	return keys, nil
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
