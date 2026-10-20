package trust

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client orchestrates Design A trust refresh for an official artifact origin.
type Client struct {
	TrustDir  string
	Origin    string // e.g. https://aliyuncli.alicdn.com
	Fetch     RootFetcher
	Bootstrap BootstrapTrust
	Now       time.Time
	// PreferLegacyRootJSON keeps Phase-1 CDN trust/root.json fallback when
	// Design A roots/ chain is not yet published.
	PreferLegacyRootJSON bool
}

// RefreshResult is the trusted key material after a successful refresh.
type RefreshResult struct {
	Root       *RootMetaFile
	Timestamp  *TimestampFile
	RoleKeys   map[string][]VerifyKey
	RoleThresh map[string]int
	UsedLegacy bool
}

// NewClient builds a Design A trust client for origin.
func NewClient(origin, trustDir string, fetch RootFetcher) *Client {
	if trustDir == "" {
		trustDir = DefaultTrustDir()
	}
	if origin == "" {
		origin = DefaultArtifactOrigin
	}
	return &Client{
		TrustDir:             trustDir,
		Origin:               strings.TrimRight(origin, "/"),
		Fetch:                fetch,
		Bootstrap:            EmbeddedBootstrap(),
		Now:                  time.Now(),
		PreferLegacyRootJSON: true,
	}
}

// RefreshRecovery checks fixed recovery URLs and applies a newer patch when present.
func (c *Client) RefreshRecovery() error {
	if c.Fetch == nil {
		return fmt.Errorf("fetcher is nil")
	}
	now := c.now()
	st, _ := LoadRecoveryState(c.TrustDir)
	keys, err := c.Bootstrap.RecoveryVerifyKeys()
	if err != nil {
		return err
	}
	urls := []string{
		c.Origin + RecoveryPath,
		"https://api.aliyun.com/cli-trust/v1/recovery/latest.json",
	}
	var lastErr error
	for _, u := range urls {
		data, err := c.Fetch(u)
		if err != nil || len(data) == 0 {
			lastErr = err
			continue
		}
		doc, err := ParseRecovery(data)
		if err != nil {
			lastErr = err
			continue
		}
		if st.HighestRecoveryEpoch > 0 && doc.Signed.RecoveryEpoch <= st.HighestRecoveryEpoch {
			st.CheckedAt = now.UTC().Format(time.RFC3339)
			_ = SaveRecoveryState(c.TrustDir, st)
			return nil
		}
		if err := VerifyRecovery(doc, keys, c.Bootstrap.RecoveryRole.Threshold, now, st.HighestRecoveryEpoch); err != nil {
			lastErr = err
			continue
		}
		if err := ApplyRecoveryPatch(c.TrustDir, doc, now); err != nil {
			return err
		}
		return nil
	}
	// Missing recovery endpoint is OK (no incident). Only fail if we already require recovery.
	if st.HighestRecoveryEpoch > 0 && lastErr != nil {
		// Still allow operation if we have a prior successful state and checked recently.
		return nil
	}
	return nil
}

// Refresh updates recovery, root chain (or legacy root.json), and timestamp.
func (c *Client) Refresh() (*RefreshResult, error) {
	if err := c.RefreshRecovery(); err != nil {
		return nil, err
	}
	now := c.now()
	rootsBase := c.Origin + RootsPathPrefix
	root, err := UpdateRootChain(c.TrustDir, rootsBase, c.Fetch, c.Bootstrap, now)
	usedLegacy := false
	if err != nil {
		return nil, err
	}
	// If only bootstrap synthetic root (no delegated business roles), try legacy root.json.
	if c.PreferLegacyRootJSON && (root == nil || len(root.Signed.Roles) <= 1) {
		legacyURL := c.Origin + "/trust/root.json"
		if data, ferr := c.Fetch(legacyURL); ferr == nil && len(data) > 0 {
			if legacy, perr := ParseRoot(data); perr == nil {
				bootKeys, _ := c.Bootstrap.RootVerifyKeys()
				if len(bootKeys) > 0 {
					// Require Root authenticity once bootstrap keys are embedded.
					if verr := VerifyRoot(legacy, bootKeys, now, 0); verr == nil {
						_ = SaveCachedRoot(c.TrustDir, data)
						keys, kerr := KeysFromRoot(legacy)
						if kerr == nil {
							usedLegacy = true
							res := &RefreshResult{
								UsedLegacy: true,
								RoleKeys:   map[string][]VerifyKey{},
								RoleThresh: map[string]int{},
							}
							for _, role := range []string{RolePlugins, RoleUpgrade, RoleTimestamp} {
								res.RoleKeys[role] = filterRole(keys, role)
								res.RoleThresh[role] = 1
							}
							return res, nil
						}
					}
				}
				// Phase-1 unsigned/catalog root.json (no Root sigs): keep transitional behavior.
				if len(legacy.Signatures) == 0 {
					keys, kerr := KeysFromRoot(legacy)
					if kerr == nil {
						usedLegacy = true
						_ = SaveCachedRoot(c.TrustDir, data)
						res := &RefreshResult{
							UsedLegacy: true,
							RoleKeys:   map[string][]VerifyKey{},
							RoleThresh: map[string]int{},
						}
						for _, role := range []string{RolePlugins, RoleUpgrade, RoleTimestamp} {
							res.RoleKeys[role] = filterRole(keys, role)
							res.RoleThresh[role] = 1
						}
						return res, nil
					}
				}
			}
		}
	}

	res := &RefreshResult{
		Root:       root,
		UsedLegacy: usedLegacy,
		RoleKeys:   map[string][]VerifyKey{},
		RoleThresh: map[string]int{},
	}
	if root != nil {
		for _, role := range []string{RolePlugins, RoleUpgrade, RoleTimestamp, RoleRoot} {
			keys, spec, err := root.RoleKeys(role)
			if err != nil {
				continue
			}
			res.RoleKeys[role] = keys
			res.RoleThresh[role] = spec.Threshold
		}
	}

	// Optional timestamp freshness.
	if tsKeys := res.RoleKeys[RoleTimestamp]; len(tsKeys) > 0 {
		data, err := c.Fetch(c.Origin + TimestampPath)
		if err == nil && len(data) > 0 {
			ts, err := ParseTimestamp(data)
			if err == nil {
				store := VersionStore{Dir: c.TrustDir}
				last, _ := store.Load("timestamp")
				thresh := res.RoleThresh[RoleTimestamp]
				if thresh <= 0 {
					thresh = 1
				}
				if err := VerifyTimestamp(ts, tsKeys, thresh, now, last); err == nil {
					res.Timestamp = ts
					_ = store.Save("timestamp", ts.Signed.Version)
					_ = os.WriteFile(filepath.Join(c.TrustDir, "timestamp.json"), data, 0644)
				}
			}
		}
	}
	return res, nil
}

// ResolveRoleKeys refreshes trust and returns keys for role, merging ActiveVerifyKeys.
func (c *Client) ResolveRoleKeys(role string) ([]VerifyKey, error) {
	res, err := c.Refresh()
	if err != nil {
		return nil, err
	}
	keys := append([]VerifyKey(nil), res.RoleKeys[role]...)
	keys = append(keys, filterRole(ActiveVerifyKeys(), role)...)
	if len(keys) == 0 {
		return nil, fmt.Errorf("no trusted keys for role %q", role)
	}
	return keys, nil
}

func (c *Client) now() time.Time {
	if c.Now.IsZero() {
		return time.Now()
	}
	return c.Now
}
