package trust

import (
	"encoding/json"
	"time"
)

const (
	SchemaVersion        = 1
	AlgorithmEd25519     = "Ed25519"
	PurposePluginIndex   = "plugin-index"
	PurposeUpgrade       = "cli-upgrade-manifest"
	PurposeSecurityPatch = "cli-security-patch"
)

// SignedEnvelope is the only signed wire format. Signatures cover the decoded
// Payload bytes exactly; callers must never re-marshal payload before checking.
type SignedEnvelope struct {
	SchemaVersion int         `json:"schema_version"`
	Payload       string      `json:"payload"`
	Signatures    []Signature `json:"signatures"`
}

type Signature struct {
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"`
}

type Key struct {
	KeyID     string    `json:"key_id"`
	Algorithm string    `json:"algorithm"`
	PublicKey string    `json:"public_key"`
	Purposes  []string  `json:"purposes,omitempty"`
	NotBefore time.Time `json:"not_before,omitempty"`
	NotAfter  time.Time `json:"not_after,omitempty"`
}

type Role struct {
	KeyIDs    []string `json:"key_ids"`
	Threshold int      `json:"threshold"`
}

type Root struct {
	Type          string          `json:"type"`
	SchemaVersion int             `json:"schema_version"`
	Version       uint64          `json:"version"`
	ExpiresAt     time.Time       `json:"expires_at"`
	Authority     string          `json:"authority"`
	Keys          []Key           `json:"keys"`
	Roles         map[string]Role `json:"roles"`
}

type ArtifactKeys struct {
	Type          string          `json:"type"`
	SchemaVersion int             `json:"schema_version"`
	Version       uint64          `json:"version"`
	ExpiresAt     time.Time       `json:"expires_at"`
	Authority     string          `json:"authority"`
	Keys          []Key           `json:"keys"`
	Roles         map[string]Role `json:"roles"`
}

type Discovery struct {
	SchemaVersion                int       `json:"schema_version"`
	Authority                    string    `json:"authority"`
	RootMetadataURI              string    `json:"root_metadata_uri"`
	ArtifactKeysURI              string    `json:"artifact_keys_uri"`
	TimestampURI                 string    `json:"timestamp_uri"`
	RecoveryPatchURI             string    `json:"recovery_patch_uri,omitempty"`
	OfficialArtifactOrigins      []string  `json:"official_artifact_origins"`
	SupportedSignatureAlgorithms []string  `json:"supported_signature_algorithms"`
	SupportedArtifactTypes       []string  `json:"supported_artifact_types"`
	CacheMaxAgeSeconds           int64     `json:"cache_max_age_seconds"`
	ExpiresAt                    time.Time `json:"expires_at"`
}

type ArtifactDescriptor struct {
	URL       string    `json:"url"`
	SHA256    string    `json:"sha256"`
	Length    int64     `json:"length"`
	Version   uint64    `json:"version"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Timestamp struct {
	Type          string                        `json:"type"`
	SchemaVersion int                           `json:"schema_version"`
	Version       uint64                        `json:"version"`
	ExpiresAt     time.Time                     `json:"expires_at"`
	Authority     string                        `json:"authority"`
	Artifacts     map[string]ArtifactDescriptor `json:"artifacts"`
}

type UpgradeManifest struct {
	Type           string                        `json:"type"`
	SchemaVersion  int                           `json:"schema_version"`
	Version        uint64                        `json:"version"`
	ExpiresAt      time.Time                     `json:"expires_at"`
	ReleaseVersion string                        `json:"release_version"`
	Artifacts      map[string]ArtifactDescriptor `json:"artifacts"`
}

type RecoveryPatch struct {
	Type             string          `json:"type"`
	SchemaVersion    int             `json:"schema_version"`
	RecoveryEpoch    uint64          `json:"recovery_epoch"`
	PatchEpoch       uint64          `json:"patch_epoch"`
	NotBefore        time.Time       `json:"not_before"`
	ExpiresAt        time.Time       `json:"expires_at"`
	AffectedRootMin  uint64          `json:"affected_root_min"`
	AffectedRootMax  uint64          `json:"affected_root_max"`
	NewBootstrapRoot json.RawMessage `json:"new_bootstrap_root"`
	DenyKeyIDs       []string        `json:"deny_key_ids,omitempty"`
	ClearOrigins     []string        `json:"clear_origins,omitempty"`
}

type SecurityPatchManifest struct {
	Type          string                        `json:"type"`
	SchemaVersion int                           `json:"schema_version"`
	Version       uint64                        `json:"version"`
	PatchEpoch    uint64                        `json:"patch_epoch"`
	ExpiresAt     time.Time                     `json:"expires_at"`
	Artifacts     map[string]ArtifactDescriptor `json:"artifacts"`
}

type State struct {
	RootVersion      uint64            `json:"root_version"`
	KeysVersion      uint64            `json:"keys_version"`
	TimestampVersion uint64            `json:"timestamp_version"`
	IndexVersion     uint64            `json:"index_version,omitempty"`
	ArtifactVersions map[string]uint64 `json:"artifact_versions,omitempty"`
	RecoveryEpoch    uint64            `json:"recovery_epoch"`
	PatchEpoch       uint64            `json:"patch_epoch"`
	DenyKeyIDs       []string          `json:"deny_key_ids,omitempty"`
}

type VerifiedArtifact struct {
	Bytes      []byte
	Descriptor ArtifactDescriptor
	KeyIDs     []string
	SignedAt   time.Time
}
