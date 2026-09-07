package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const RecoveryMetaType = "trust-recovery"

// RecoveryPatch is Design A trust-data hot patch signed by Recovery keys.
type RecoveryPatch struct {
	Type                     string `json:"type"`
	SchemaVersion            int    `json:"schema_version"`
	RecoveryEpoch            int64  `json:"recovery_epoch"`
	IncidentID               string `json:"incident_id,omitempty"`
	IssuedAt                 string `json:"issued_at,omitempty"`
	ExpiresAt                string `json:"expires_at"`
	MinimumRecoveryProtocol  int    `json:"minimum_recovery_protocol,omitempty"`
	CompromisedRootVersions  *struct {
		Min int64 `json:"min"`
		Max int64 `json:"max"`
	} `json:"compromised_root_versions,omitempty"`
	RevokedKeyIDs            []string `json:"revoked_key_ids,omitempty"`
	ReplacementBootstrapRoot *struct {
		Version   int64              `json:"version"`
		Threshold int                `json:"threshold"`
		Keys      map[string]KeySpec `json:"keys"`
	} `json:"replacement_bootstrap_root,omitempty"`
	Invalidate map[string]bool `json:"invalidate,omitempty"`
}

// RecoveryFile is the on-wire recovery document.
type RecoveryFile struct {
	Signed     RecoveryPatch   `json:"signed"`
	Signatures []MetaSignature `json:"signatures"`
}

// RecoveryState is persisted local recovery status.
type RecoveryState struct {
	HighestRecoveryEpoch       int64  `json:"highest_recovery_epoch"`
	ActiveBootstrapRootVersion int64  `json:"active_bootstrap_root_version"`
	IncidentID                 string `json:"incident_id,omitempty"`
	InstalledAt                string `json:"installed_at,omitempty"`
	CheckedAt                  string `json:"checked_at,omitempty"`
}

// ParseRecovery parses a recovery patch document.
func ParseRecovery(data []byte) (*RecoveryFile, error) {
	var doc RecoveryFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse recovery: %w", err)
	}
	if doc.Signed.Type != "" && doc.Signed.Type != RecoveryMetaType {
		return nil, fmt.Errorf("unexpected recovery type %q", doc.Signed.Type)
	}
	if doc.Signed.RecoveryEpoch <= 0 {
		return nil, fmt.Errorf("recovery_epoch must be positive")
	}
	if strings.TrimSpace(doc.Signed.ExpiresAt) == "" {
		return nil, fmt.Errorf("recovery missing expires_at")
	}
	return &doc, nil
}

// VerifyRecovery checks Recovery threshold signatures and expiry.
func VerifyRecovery(doc *RecoveryFile, keys []VerifyKey, threshold int, now time.Time, highestEpoch int64) error {
	if doc == nil {
		return fmt.Errorf("recovery is nil")
	}
	exp, err := time.Parse(time.RFC3339, doc.Signed.ExpiresAt)
	if err != nil {
		return fmt.Errorf("recovery expires_at: %w", err)
	}
	if now.After(exp) {
		return fmt.Errorf("recovery patch expired at %s", exp.UTC().Format(time.RFC3339))
	}
	if highestEpoch > 0 && doc.Signed.RecoveryEpoch < highestEpoch {
		return fmt.Errorf("recovery_epoch %d older than highest %d", doc.Signed.RecoveryEpoch, highestEpoch)
	}
	if highestEpoch > 0 && doc.Signed.RecoveryEpoch == highestEpoch {
		// Same epoch is a no-op refresh; still must verify.
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
	return VerifyThresholdSignatures(payload, doc.Signatures, keys, threshold)
}

// LoadRecoveryState reads ~/.aliyun/trust/recovery-state.json.
func LoadRecoveryState(trustDir string) (RecoveryState, error) {
	var st RecoveryState
	if trustDir == "" {
		return st, nil
	}
	data, err := os.ReadFile(filepath.Join(trustDir, "recovery-state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return st, err
	}
	return st, nil
}

// SaveRecoveryState persists recovery-state.json.
func SaveRecoveryState(trustDir string, st RecoveryState) error {
	if trustDir == "" {
		return nil
	}
	if err := os.MkdirAll(trustDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(trustDir, "recovery-state.json"), data, 0644)
}

// ApplyRecoveryPatch installs a verified recovery patch into the local trust dir.
func ApplyRecoveryPatch(trustDir string, doc *RecoveryFile, now time.Time) error {
	if doc == nil || doc.Signed.ReplacementBootstrapRoot == nil {
		return fmt.Errorf("recovery patch missing replacement_bootstrap_root")
	}
	repl := doc.Signed.ReplacementBootstrapRoot
	if repl.Version <= 0 || repl.Threshold <= 0 || len(repl.Keys) == 0 {
		return fmt.Errorf("invalid replacement_bootstrap_root")
	}

	// Build a synthetic root meta for the replacement bootstrap and store it.
	keyIDs := make([]string, 0, len(repl.Keys))
	for id := range repl.Keys {
		keyIDs = append(keyIDs, id)
	}
	meta := &RootMetaFile{
		Signed: RootMeta{
			Type:          RootMetaType,
			SchemaVersion: BootstrapSchemaVersion,
			Version:       repl.Version,
			Keys:          repl.Keys,
			Roles: map[string]RoleSpec{
				RoleRoot: {Threshold: repl.Threshold, KeyIDs: keyIDs},
			},
		},
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := SaveRootMetaVersion(trustDir, repl.Version, raw); err != nil {
		return err
	}

	// Clear caches when requested.
	if doc.Signed.Invalidate != nil {
		if doc.Signed.Invalidate["timestamp_cache"] {
			_ = os.Remove(filepath.Join(trustDir, "timestamp.json"))
			_ = os.Remove(filepath.Join(trustDir, "timestamp_version"))
		}
		if doc.Signed.Invalidate["plugin_index_cache"] {
			_ = os.Remove(filepath.Join(trustDir, "plugin_pkg_index_version"))
		}
		if doc.Signed.Invalidate["upgrade_manifest_cache"] {
			_ = os.Remove(filepath.Join(trustDir, "upgrade_manifest_version"))
		}
		if doc.Signed.Invalidate["root_cache"] {
			// Keep the newly installed replacement root; remove older numbered roots.
			entries, _ := os.ReadDir(filepath.Join(trustDir, "roots"))
			for _, e := range entries {
				name := e.Name()
				if !strings.HasSuffix(name, ".root.json") {
					continue
				}
				verStr := strings.TrimSuffix(name, ".root.json")
				if verStr == fmt.Sprintf("%d", repl.Version) {
					continue
				}
				_ = os.Remove(filepath.Join(trustDir, "roots", name))
			}
		}
	}

	st := RecoveryState{
		HighestRecoveryEpoch:       doc.Signed.RecoveryEpoch,
		ActiveBootstrapRootVersion: repl.Version,
		IncidentID:                 doc.Signed.IncidentID,
		InstalledAt:                now.UTC().Format(time.RFC3339),
		CheckedAt:                  now.UTC().Format(time.RFC3339),
	}
	return SaveRecoveryState(trustDir, st)
}
