package trust

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PlusClient is Design B+: Discovery for location + builtin Root for authenticity.
type PlusClient struct {
	TrustDir      string
	DiscoveryURL  string
	Fetch         RootFetcher
	Bootstrap     BootstrapTrust
	Now           time.Time
	HostAllowlist []string
}

// PlusRefreshResult holds Discovery-driven trust material.
type PlusRefreshResult struct {
	Discovery    *DiscoveryDocument
	Root         *RootMetaFile
	ArtifactKeys *ArtifactKeysFile
	Timestamp    *TimestampFile
	RoleKeys     map[string][]VerifyKey
}

// NewPlusClient constructs a B+ client with the fixed Discovery URL.
func NewPlusClient(trustDir string, fetch RootFetcher) *PlusClient {
	if trustDir == "" {
		trustDir = DefaultTrustDir()
	}
	return &PlusClient{
		TrustDir:      trustDir,
		DiscoveryURL:  DefaultDiscoveryURL,
		Fetch:         fetch,
		Bootstrap:     EmbeddedBootstrap(),
		Now:           time.Now(),
		HostAllowlist: append([]string(nil), BuiltinHostAllowlist...),
	}
}

func (c *PlusClient) now() time.Time {
	if c.Now.IsZero() {
		return time.Now()
	}
	return c.Now
}

// LoadDiscovery fetches and validates Discovery (with last-known-good cache fallback).
func (c *PlusClient) LoadDiscovery() (*DiscoveryDocument, error) {
	if c.Fetch == nil {
		return nil, fmt.Errorf("fetcher is nil")
	}
	now := c.now()
	cachePath := filepath.Join(c.TrustDir, "discovery.json")
	data, err := c.Fetch(c.DiscoveryURL)
	if err == nil && len(data) > 0 {
		doc, perr := ParseDiscovery(data, now)
		if perr == nil {
			_ = os.MkdirAll(c.TrustDir, 0755)
			_ = os.WriteFile(cachePath, data, 0644)
			return doc, nil
		}
		err = perr
	}
	// Fall back to unexpired cached discovery.
	if cached, rerr := os.ReadFile(cachePath); rerr == nil {
		if doc, perr := ParseDiscovery(cached, now); perr == nil {
			return doc, nil
		}
	}
	if err == nil {
		err = fmt.Errorf("discovery unavailable")
	}
	return nil, err
}

// Refresh runs Discovery → Root chain → artifact-keys → timestamp.
func (c *PlusClient) Refresh() (*PlusRefreshResult, error) {
	now := c.now()
	// Recovery first (same as Design A), using Discovery URI when present.
	doc, err := c.LoadDiscovery()
	if err != nil {
		return nil, err
	}

	// Optional recovery via Discovery URI, else fixed A paths on first official origin.
	recURLs := []string{}
	if strings.TrimSpace(doc.RecoveryPatchURI) != "" {
		recURLs = append(recURLs, doc.RecoveryPatchURI)
	}
	for _, o := range doc.OfficialArtifactOrigins {
		recURLs = append(recURLs, strings.TrimRight(o, "/")+RecoveryPath)
	}
	if err := c.refreshRecovery(recURLs); err != nil {
		return nil, err
	}

	rootsBase := RootsBaseFromDiscovery(doc)
	root, err := UpdateRootChain(c.TrustDir, rootsBase, c.Fetch, c.Bootstrap, now)
	if err != nil {
		return nil, err
	}

	rootKeys, rootRole, err := root.RootKeysAsVerify()
	if err != nil {
		rootKeys, err = c.Bootstrap.RootVerifyKeys()
		if err != nil {
			return nil, err
		}
		rootRole = c.Bootstrap.RootRole
	}

	keysData, err := c.Fetch(doc.ArtifactKeysURI)
	if err != nil || len(keysData) == 0 {
		return nil, fmt.Errorf("fetch artifact-keys: %w", err)
	}
	ak, err := ParseArtifactKeys(keysData)
	if err != nil {
		return nil, err
	}
	store := VersionStore{Dir: c.TrustDir}
	lastAK, _ := store.Load("artifact_keys")
	if err := VerifyArtifactKeys(ak, rootKeys, rootRole.Threshold, now, lastAK); err != nil {
		return nil, err
	}
	_ = store.Save("artifact_keys", ak.Signed.Version)
	_ = os.WriteFile(filepath.Join(c.TrustDir, "artifact-keys.json"), keysData, 0644)

	res := &PlusRefreshResult{
		Discovery:    doc,
		Root:         root,
		ArtifactKeys: ak,
		RoleKeys:     map[string][]VerifyKey{},
	}
	for _, role := range []string{RolePlugins, RoleUpgrade} {
		keys, err := ak.KeysForPurpose(PurposeForRole(role), now)
		if err != nil {
			return nil, err
		}
		res.RoleKeys[role] = keys
	}

	if strings.TrimSpace(doc.TimestampURI) != "" {
		tsData, err := c.Fetch(doc.TimestampURI)
		if err == nil && len(tsData) > 0 {
			ts, err := ParseTimestamp(tsData)
			if err == nil {
				// Prefer timestamp keys from root role when present; else reuse root keys.
				tsKeys := rootKeys
				thresh := 1
				if root != nil {
					if k, spec, err := root.RoleKeys(RoleTimestamp); err == nil && len(k) > 0 {
						tsKeys = k
						thresh = spec.Threshold
					}
				}
				lastTS, _ := store.Load("timestamp")
				if err := VerifyTimestamp(ts, tsKeys, thresh, now, lastTS); err == nil {
					res.Timestamp = ts
					_ = store.Save("timestamp", ts.Signed.Version)
					_ = os.WriteFile(filepath.Join(c.TrustDir, "timestamp.json"), tsData, 0644)
				}
			}
		}
	}
	return res, nil
}

// ResolveRoleKeys refreshes B+ trust and returns keys for role.
func (c *PlusClient) ResolveRoleKeys(role string) ([]VerifyKey, error) {
	res, err := c.Refresh()
	if err != nil {
		return nil, err
	}
	keys := append([]VerifyKey(nil), res.RoleKeys[role]...)
	keys = append(keys, filterRole(ActiveVerifyKeys(), role)...)
	if len(keys) == 0 {
		return nil, fmt.Errorf("no trusted keys for role %q via discovery", role)
	}
	return keys, nil
}

func (c *PlusClient) refreshRecovery(urls []string) error {
	now := c.now()
	st, _ := LoadRecoveryState(c.TrustDir)
	keys, err := c.Bootstrap.RecoveryVerifyKeys()
	if err != nil {
		return err
	}
	for _, u := range urls {
		if strings.TrimSpace(u) == "" {
			continue
		}
		if err := ValidateTrustURL(u, c.HostAllowlist); err != nil {
			continue
		}
		data, err := c.Fetch(u)
		if err != nil || len(data) == 0 {
			continue
		}
		doc, err := ParseRecovery(data)
		if err != nil {
			continue
		}
		if st.HighestRecoveryEpoch > 0 && doc.Signed.RecoveryEpoch <= st.HighestRecoveryEpoch {
			st.CheckedAt = now.UTC().Format(time.RFC3339)
			_ = SaveRecoveryState(c.TrustDir, st)
			return nil
		}
		if err := VerifyRecovery(doc, keys, c.Bootstrap.RecoveryRole.Threshold, now, st.HighestRecoveryEpoch); err != nil {
			continue
		}
		return ApplyRecoveryPatch(c.TrustDir, doc, now)
	}
	return nil
}
