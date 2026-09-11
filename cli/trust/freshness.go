package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FreshnessMeta is the optional authenticity/freshness header on signed manifests.
type FreshnessMeta struct {
	IndexVersion int64
	ExpiresAt    time.Time
	HasExpiry    bool
}

// ParseFreshnessFromJSON extracts index_version / expires_at from a JSON object
// without requiring a full typed unmarshal of the payload.
func ParseFreshnessFromJSON(data []byte) (FreshnessMeta, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return FreshnessMeta{}, fmt.Errorf("parse freshness: %w", err)
	}
	var meta FreshnessMeta
	if v, ok := raw["index_version"]; ok {
		var n json.Number
		if err := json.Unmarshal(v, &n); err != nil {
			var i int64
			if err2 := json.Unmarshal(v, &i); err2 != nil {
				return FreshnessMeta{}, fmt.Errorf("index_version: %w", err)
			}
			meta.IndexVersion = i
		} else {
			i, err := n.Int64()
			if err != nil {
				return FreshnessMeta{}, fmt.Errorf("index_version: %w", err)
			}
			meta.IndexVersion = i
		}
	}
	if v, ok := raw["expires_at"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return FreshnessMeta{}, fmt.Errorf("expires_at: %w", err)
		}
		s = strings.TrimSpace(s)
		if s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return FreshnessMeta{}, fmt.Errorf("expires_at must be RFC3339: %w", err)
			}
			meta.ExpiresAt = t
			meta.HasExpiry = true
		}
	}
	return meta, nil
}

// CheckExpiry fails when expires_at is present and now is after it.
func CheckExpiry(meta FreshnessMeta, now time.Time) error {
	if !meta.HasExpiry {
		return nil
	}
	if now.After(meta.ExpiresAt) {
		return fmt.Errorf("manifest expired at %s (now %s); re-fetch from an official channel or reinstall from https://www.alibabacloud.com/help",
			meta.ExpiresAt.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339))
	}
	return nil
}

// CheckMonotonicVersion fails when remote version is lower than the last accepted one.
func CheckMonotonicVersion(meta FreshnessMeta, lastAccepted int64) error {
	if meta.IndexVersion == 0 {
		return nil // legacy unsigned / unversioned manifests
	}
	if lastAccepted > 0 && meta.IndexVersion < lastAccepted {
		return fmt.Errorf("manifest index_version %d is older than last accepted %d (rollback rejected)",
			meta.IndexVersion, lastAccepted)
	}
	return nil
}

// VersionStore persists the highest accepted index_version for a named artifact.
type VersionStore struct {
	Dir string // e.g. ~/.aliyun/trust
}

func DefaultTrustDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".aliyun", "trust")
}

func (s VersionStore) path(name string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	return filepath.Join(s.Dir, safe+"_version")
}

func (s VersionStore) Load(name string) (int64, error) {
	if s.Dir == "" {
		return 0, nil
	}
	data, err := os.ReadFile(s.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, nil
	}
	return n, nil
}

func (s VersionStore) Save(name string, version int64) error {
	if s.Dir == "" || version <= 0 {
		return nil
	}
	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.path(name), []byte(strconv.FormatInt(version, 10)+"\n"), 0644)
}
