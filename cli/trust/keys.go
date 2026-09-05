package trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
)

const (
	RolePlugins = "plugins"
	RoleUpgrade = "upgrade"
	RoleRoot    = "root"
)

// VerifyKey is a public key the client will accept for a given role.
type VerifyKey struct {
	KeyID     string
	PublicKey ed25519.PublicKey
	Roles     []string // empty means all roles
}

func (k VerifyKey) HasRole(role string) bool {
	if len(k.Roles) == 0 {
		return true
	}
	for _, r := range k.Roles {
		if r == role {
			return true
		}
	}
	return false
}

var (
	keysMu       sync.RWMutex
	runtimeKeys  []VerifyKey
	embeddedKeys []VerifyKey // filled at init; production keys land here when ready
)

func init() {
	// Production plugin/upgrade public keys are NOT embedded here.
	// They are published in CDN trust/root.json (see DeriveTrustRootURL) and
	// loaded at runtime. Optional ALIBABA_CLOUD_CLI_TRUST_PUBKEYS remains for
	// break-glass / local debugging. Root signing keys may be embedded later
	// for Phase-2 authenticity of root.json itself.
	embeddedKeys = nil
}

// SetVerifyKeys replaces the runtime key set (tests / process bootstrap).
func SetVerifyKeys(keys []VerifyKey) {
	keysMu.Lock()
	defer keysMu.Unlock()
	runtimeKeys = append([]VerifyKey(nil), keys...)
}

// ResetVerifyKeys clears runtime overrides (tests).
func ResetVerifyKeys() {
	keysMu.Lock()
	defer keysMu.Unlock()
	runtimeKeys = nil
}

// ActiveVerifyKeys returns runtime keys if set, otherwise embedded keys,
// optionally extended by ALIBABA_CLOUD_CLI_TRUST_PUBKEYS (keyid:base64pub[,role...];...).
func ActiveVerifyKeys() []VerifyKey {
	keysMu.RLock()
	defer keysMu.RUnlock()
	base := embeddedKeys
	if len(runtimeKeys) > 0 {
		base = runtimeKeys
	}
	extra := parsePubKeysEnv(os.Getenv("ALIBABA_CLOUD_CLI_TRUST_PUBKEYS"))
	if len(extra) == 0 {
		out := make([]VerifyKey, len(base))
		copy(out, base)
		return out
	}
	out := make([]VerifyKey, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

func parsePubKeysEnv(raw string) []VerifyKey {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []VerifyKey
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Split(part, ":")
		if len(fields) < 2 {
			continue
		}
		keyID := strings.TrimSpace(fields[0])
		pubB64 := strings.TrimSpace(fields[1])
		var roles []string
		if len(fields) > 2 {
			for _, r := range fields[2:] {
				r = strings.TrimSpace(r)
				if r != "" {
					roles = append(roles, r)
				}
			}
		}
		pub, err := DecodePublicKey(pubB64)
		if err != nil {
			continue
		}
		out = append(out, VerifyKey{KeyID: keyID, PublicKey: pub, Roles: roles})
	}
	return out
}

// DecodePublicKey decodes a standard-base64 Ed25519 public key (32 bytes).
func DecodePublicKey(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes, got %d", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

// DecodePrivateSeed decodes a standard-base64 Ed25519 seed (32 bytes) to a private key.
func DecodePrivateSeed(b64 string) (ed25519.PrivateKey, error) {
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("decode private seed: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("private seed must be %d bytes, got %d", ed25519.SeedSize, len(seed))
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func FindKey(keys []VerifyKey, keyID, role string) (VerifyKey, bool) {
	for _, k := range keys {
		if k.KeyID == keyID && k.HasRole(role) {
			return k, true
		}
	}
	return VerifyKey{}, false
}
