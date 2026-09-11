package trust

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const TimestampMetaType = "timestamp"

// TimestampMeta is Design A short-lived freshness metadata.
type TimestampMeta struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
	Version       int64  `json:"version"`
	ExpiresAt     string `json:"expires_at"`
	// Optional pointers to current artifact versions.
	Meta map[string]int64 `json:"meta,omitempty"`
}

// TimestampFile is the on-wire timestamp.json document.
type TimestampFile struct {
	Signed     TimestampMeta   `json:"signed"`
	Signatures []MetaSignature `json:"signatures"`
}

// ParseTimestamp parses timestamp.json.
func ParseTimestamp(data []byte) (*TimestampFile, error) {
	var doc TimestampFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse timestamp: %w", err)
	}
	if doc.Signed.Type != "" && doc.Signed.Type != TimestampMetaType {
		return nil, fmt.Errorf("unexpected timestamp type %q", doc.Signed.Type)
	}
	if doc.Signed.Version <= 0 {
		return nil, fmt.Errorf("timestamp version must be positive")
	}
	if strings.TrimSpace(doc.Signed.ExpiresAt) == "" {
		return nil, fmt.Errorf("timestamp missing expires_at")
	}
	return &doc, nil
}

// VerifyTimestamp checks threshold signatures, expiry, and monotonic version.
func VerifyTimestamp(doc *TimestampFile, keys []VerifyKey, threshold int, now time.Time, lastVersion int64) error {
	if doc == nil {
		return fmt.Errorf("timestamp is nil")
	}
	exp, err := time.Parse(time.RFC3339, doc.Signed.ExpiresAt)
	if err != nil {
		return fmt.Errorf("timestamp expires_at: %w", err)
	}
	if now.After(exp) {
		return fmt.Errorf("timestamp expired at %s", exp.UTC().Format(time.RFC3339))
	}
	if lastVersion > 0 && doc.Signed.Version < lastVersion {
		return fmt.Errorf("timestamp version %d older than last accepted %d", doc.Signed.Version, lastVersion)
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
