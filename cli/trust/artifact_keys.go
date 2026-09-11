package trust

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const ArtifactKeysType = "artifact-keys"

// ArtifactKeyEntry is one business key in artifact-keys.json.
type ArtifactKeyEntry struct {
	KeyID     string   `json:"key_id"`
	Algorithm string   `json:"algorithm"`
	PublicKey string   `json:"public_key"`
	Purposes  []string `json:"purposes"`
	NotBefore string   `json:"not_before,omitempty"`
	NotAfter  string   `json:"not_after,omitempty"`
}

// ArtifactKeysMeta is the signed body of artifact-keys.json (Design B+).
type ArtifactKeysMeta struct {
	Type          string             `json:"type"`
	SchemaVersion int                `json:"schema_version"`
	Version       int64              `json:"version"`
	ExpiresAt     string             `json:"expires_at"`
	Authority     string             `json:"authority,omitempty"`
	Keys          []ArtifactKeyEntry `json:"keys"`
	Policy        map[string]any     `json:"policy,omitempty"`
}

// ArtifactKeysFile is the on-wire Root-signed artifact-keys document.
type ArtifactKeysFile struct {
	Signed     ArtifactKeysMeta `json:"signed"`
	Signatures []MetaSignature  `json:"signatures"`
}

// ParseArtifactKeys parses artifact-keys.json.
func ParseArtifactKeys(data []byte) (*ArtifactKeysFile, error) {
	var doc ArtifactKeysFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse artifact-keys: %w", err)
	}
	if doc.Signed.Type != "" && doc.Signed.Type != ArtifactKeysType {
		return nil, fmt.Errorf("unexpected artifact-keys type %q", doc.Signed.Type)
	}
	if doc.Signed.Version <= 0 {
		return nil, fmt.Errorf("artifact-keys version must be positive")
	}
	if strings.TrimSpace(doc.Signed.ExpiresAt) == "" {
		return nil, fmt.Errorf("artifact-keys missing expires_at")
	}
	if len(doc.Signed.Keys) == 0 {
		return nil, fmt.Errorf("artifact-keys has no keys")
	}
	return &doc, nil
}

// VerifyArtifactKeys verifies Root threshold signatures and expiry.
func VerifyArtifactKeys(doc *ArtifactKeysFile, rootKeys []VerifyKey, threshold int, now time.Time, lastVersion int64) error {
	if doc == nil {
		return fmt.Errorf("artifact-keys is nil")
	}
	exp, err := time.Parse(time.RFC3339, doc.Signed.ExpiresAt)
	if err != nil {
		return fmt.Errorf("artifact-keys expires_at: %w", err)
	}
	if now.After(exp) {
		return fmt.Errorf("artifact-keys expired at %s", exp.UTC().Format(time.RFC3339))
	}
	if lastVersion > 0 && doc.Signed.Version < lastVersion {
		return fmt.Errorf("artifact-keys version %d older than last accepted %d", doc.Signed.Version, lastVersion)
	}
	raw, err := json.Marshal(doc.Signed)
	if err != nil {
		return err
	}
	payload, err := CanonicalSignedBytes(raw)
	if err != nil {
		return err
	}
	if threshold <= 0 {
		threshold = 1
	}
	return VerifyThresholdSignatures(payload, doc.Signatures, rootKeys, threshold)
}

// KeysForPurpose returns VerifyKeys whose purposes include purpose
// (plugin-index / cli-upgrade-manifest) and are within validity window.
func (f *ArtifactKeysFile) KeysForPurpose(purpose string, now time.Time) ([]VerifyKey, error) {
	var out []VerifyKey
	for _, e := range f.Signed.Keys {
		if !purposeMatch(e.Purposes, purpose) {
			continue
		}
		if e.NotBefore != "" {
			t, err := time.Parse(time.RFC3339, e.NotBefore)
			if err != nil {
				return nil, fmt.Errorf("key %s not_before: %w", e.KeyID, err)
			}
			if now.Before(t) {
				continue
			}
		}
		if e.NotAfter != "" {
			t, err := time.Parse(time.RFC3339, e.NotAfter)
			if err != nil {
				return nil, fmt.Errorf("key %s not_after: %w", e.KeyID, err)
			}
			if now.After(t) {
				continue
			}
		}
		pub, err := DecodePublicKey(e.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("key %s: %w", e.KeyID, err)
		}
		role := purposeToRole(purpose)
		out = append(out, VerifyKey{KeyID: e.KeyID, PublicKey: pub, Roles: []string{role}})
	}
	return out, nil
}

func purposeMatch(purposes []string, want string) bool {
	for _, p := range purposes {
		if p == want {
			return true
		}
	}
	return false
}

func purposeToRole(purpose string) string {
	switch purpose {
	case "plugin-index":
		return RolePlugins
	case "cli-upgrade-manifest":
		return RoleUpgrade
	default:
		return purpose
	}
}

// PurposeForRole maps trust roles to artifact-keys purposes.
func PurposeForRole(role string) string {
	switch role {
	case RolePlugins:
		return "plugin-index"
	case RoleUpgrade:
		return "cli-upgrade-manifest"
	default:
		return role
	}
}
