package trust

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Fixed Design A CDN paths (relative to an official artifact origin).
const (
	DefaultArtifactOrigin = "https://aliyuncli.alicdn.com"
	RootsPathPrefix       = "/trust/roots/"
	TimestampPath         = "/trust/timestamp.json"
	RecoveryPath          = "/trust/recovery/latest.json"
	RecoveryMirrorPath    = "/trust/recovery/latest.json" // same path on alternate origin
)

// RootFetcher downloads trust metadata by absolute URL.
type RootFetcher func(url string) ([]byte, error)

// UpdateRootChain walks N -> N+1 using Design A dual-threshold rules until 404/empty.
// Returns the highest accepted RootMetaFile.
func UpdateRootChain(trustDir, rootsBaseURL string, fetch RootFetcher, bootstrap BootstrapTrust, now time.Time) (*RootMetaFile, error) {
	if fetch == nil {
		return nil, fmt.Errorf("root fetcher is nil")
	}
	rootsBaseURL = strings.TrimRight(rootsBaseURL, "/") + "/"

	currentVer, _ := LoadHighestRootVersion(trustDir)
	var current *RootMetaFile
	if currentVer > 0 {
		if data, err := LoadRootMetaVersion(trustDir, currentVer); err == nil {
			if doc, err := ParseRootMeta(data); err == nil {
				current = doc
			}
		}
	}
	if current == nil {
		current = DefaultRootMetaFromBootstrap(bootstrap)
		currentVer = current.Signed.Version
	}

	for {
		nextVer := current.Signed.Version + 1
		url := rootsBaseURL + strconv.FormatInt(nextVer, 10) + ".root.json"
		data, err := fetch(url)
		if err != nil || len(data) == 0 {
			break
		}
		next, err := ParseRootMeta(data)
		if err != nil {
			return nil, fmt.Errorf("parse %d.root.json: %w", nextVer, err)
		}
		if err := acceptRootTransition(current, next, data, trustDir, now); err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

func acceptRootTransition(old, next *RootMetaFile, raw []byte, trustDir string, now time.Time) error {
	expect := old.Signed.Version + 1
	if next.Signed.Version != expect {
		return fmt.Errorf("root jump rejected: have %d, got %d", old.Signed.Version, next.Signed.Version)
	}

	oldKeys, oldRole, err := old.RootKeysAsVerify()
	if err != nil {
		// Bootstrap synthetic root may only have RootKeys from bootstrap.
		oldKeys, err = EmbeddedBootstrap().RootVerifyKeys()
		if err != nil {
			return err
		}
		oldRole = EmbeddedBootstrap().RootRole
	}
	if err := VerifyRootMeta(next, oldKeys, oldRole.Threshold, now, expect); err != nil {
		return fmt.Errorf("old-root threshold for %d.root.json: %w", expect, err)
	}

	newKeys, newRole, err := next.RootKeysAsVerify()
	if err != nil {
		return err
	}
	if err := VerifyRootMeta(next, newKeys, newRole.Threshold, now, expect); err != nil {
		return fmt.Errorf("new-root threshold for %d.root.json: %w", expect, err)
	}

	if err := SaveRootMetaVersion(trustDir, next.Signed.Version, raw); err != nil {
		return err
	}
	return nil
}

// DeriveRootsBaseURL maps an artifact origin or index URL to .../trust/roots/.
func DeriveRootsBaseURL(originOrIndexURL string) string {
	u := strings.TrimSpace(originOrIndexURL)
	if u == "" {
		return strings.TrimRight(DefaultArtifactOrigin, "/") + RootsPathPrefix
	}
	for _, suf := range []string{
		"/plugins-pb/plugin_pkg_index.json",
		"/plugins-v2/plugin_pkg_index.json",
		"/plugins/plugin_pkg_index.json",
		"/plugin_pkg_index.json",
		"/trust/root.json",
		"/upgrade/upgrade_manifest.json",
		"/upgrade_manifest.json",
	} {
		if i := strings.Index(u, suf); i >= 0 {
			return strings.TrimRight(u[:i], "/") + RootsPathPrefix
		}
	}
	return strings.TrimRight(u, "/") + RootsPathPrefix
}
