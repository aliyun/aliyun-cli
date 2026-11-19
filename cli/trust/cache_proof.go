package trust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"
)

// Version 2 also bounds artifact-keys signing-key expiry. Earlier proofs can
// have overlong deadlines and cannot safely authorize cached notifications.
const cacheProofSchemaVersion = 2

// CacheProof invalidates derived notifications after expiry or recovery changes.
// It is local-only state and must never be included in user-facing output.
type CacheProof struct {
	SchemaVersion int       `json:"schema_version"`
	StatePath     string    `json:"state_path"`
	StateHash     string    `json:"state_hash"`
	ValidUntil    time.Time `json:"valid_until"`
	Purpose       string    `json:"purpose,omitempty"`
}

func stateHash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func proofHash(data []byte, purpose string) string {
	if purpose == "" {
		return stateHash(data)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return ""
	}
	// Other artifacts may advance independently. Only this artifact's version
	// and shared root/key/recovery facts invalidate its derived notifications.
	state.TimestampVersion = 0
	version := state.ArtifactVersions[purpose]
	if purpose == PurposePluginIndex && state.IndexVersion > version {
		version = state.IndexVersion
	}
	state.IndexVersion = 0
	state.ArtifactVersions = map[string]uint64{purpose: version}
	return stateHash(mustJSON(state))
}

func (p CacheProof) Valid(now time.Time) bool {
	if p.SchemaVersion != cacheProofSchemaVersion || p.StatePath == "" || p.StateHash == "" || !now.Before(p.ValidUntil) {
		return false
	}
	data, err := os.ReadFile(p.StatePath)
	return err == nil && proofHash(data, p.Purpose) == p.StateHash
}
