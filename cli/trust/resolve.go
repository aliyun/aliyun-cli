package trust

import (
	"fmt"
	"time"
)

// ResolveVerifyKeys returns keys to use for role.
// Order: optional online root.json (verified with bootstrap root keys) → ActiveVerifyKeys().
// If root fetch/verify fails, falls back to ActiveVerifyKeys() unless requireRoot is set.
func ResolveVerifyKeys(
	role string,
	fetchRoot func() ([]byte, error),
	bootstrapRootKeys []VerifyKey,
	policy Policy,
	requireRoot bool,
) ([]VerifyKey, error) {
	base := ActiveVerifyKeys()
	if fetchRoot == nil {
		return filterRole(base, role), nil
	}

	data, err := fetchRoot()
	if err != nil || len(data) == 0 {
		if requireRoot {
			if err == nil {
				err = fmt.Errorf("empty root.json")
			}
			return nil, fmt.Errorf("load root.json: %w", err)
		}
		return filterRole(base, role), nil
	}

	root, err := ParseRoot(data)
	if err != nil {
		if requireRoot {
			return nil, err
		}
		return filterRole(base, role), nil
	}

	rootKeys := bootstrapRootKeys
	if len(rootKeys) == 0 {
		// Allow ActiveVerifyKeys entries with RoleRoot / empty roles to verify root.
		for _, k := range ActiveVerifyKeys() {
			if k.HasRole(RoleRoot) || len(k.Roles) == 0 {
				rootKeys = append(rootKeys, k)
			}
		}
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now()
	}

	store := VersionStore{Dir: policy.TrustDir}
	lastRoot, _ := store.Load("root")

	// Phase 1 bootstrap: when no Root public key is embedded yet, still accept
	// delegated keys from CDN trust/root.json (operator-managed file). Authenticity
	// of root.json itself then rides on HTTPS/CDN until Root keys are pinned.
	if len(rootKeys) == 0 {
		fromRoot, err := KeysFromRoot(root)
		if err != nil {
			if requireRoot {
				return nil, err
			}
			return filterRole(base, role), nil
		}
		if lastRoot > 0 && root.Version < lastRoot {
			err := fmt.Errorf("root version %d older than last accepted %d", root.Version, lastRoot)
			if requireRoot {
				return nil, err
			}
			return filterRole(base, role), nil
		}
		_ = store.Save("root", root.Version)
		_ = SaveCachedRoot(policy.TrustDir, data)
		merged := append([]VerifyKey(nil), fromRoot...)
		merged = append(merged, base...)
		return filterRole(merged, role), nil
	}

	if err := VerifyRoot(root, rootKeys, now, policy.MinRootVersion); err != nil {
		if requireRoot {
			return nil, err
		}
		return filterRole(base, role), nil
	}
	if lastRoot > 0 && root.Version < lastRoot {
		err := fmt.Errorf("root version %d older than last accepted %d", root.Version, lastRoot)
		if requireRoot {
			return nil, err
		}
		return filterRole(base, role), nil
	}

	fromRoot, err := KeysFromRoot(root)
	if err != nil {
		if requireRoot {
			return nil, err
		}
		return filterRole(base, role), nil
	}
	_ = store.Save("root", root.Version)
	_ = SaveCachedRoot(policy.TrustDir, data)

	// Prefer root-delegated keys; keep embedded/env keys as additional trust anchors.
	merged := append([]VerifyKey(nil), fromRoot...)
	merged = append(merged, base...)
	return filterRole(merged, role), nil
}

func filterRole(keys []VerifyKey, role string) []VerifyKey {
	if role == "" {
		return keys
	}
	var out []VerifyKey
	for _, k := range keys {
		if k.HasRole(role) {
			out = append(out, k)
		}
	}
	return out
}
