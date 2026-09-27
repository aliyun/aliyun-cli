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

const (
	RootMetaType = "root"
)

// RootMeta is Design A versioned root metadata (N.root.json signed body).
type RootMeta struct {
	Type          string             `json:"type"`
	SchemaVersion int                `json:"schema_version"`
	Version       int64              `json:"version"`
	ExpiresAt     string             `json:"expires_at,omitempty"`
	Keys          map[string]KeySpec `json:"keys"`
	Roles         map[string]RoleSpec `json:"roles"`
}

// RootMetaFile is the on-wire N.root.json document.
type RootMetaFile struct {
	Signed     RootMeta        `json:"signed"`
	Signatures []MetaSignature `json:"signatures"`
}

// ParseRootMeta parses Design A root metadata.
func ParseRootMeta(data []byte) (*RootMetaFile, error) {
	var doc RootMetaFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse root meta: %w", err)
	}
	if doc.Signed.Type != "" && doc.Signed.Type != RootMetaType {
		return nil, fmt.Errorf("unexpected root type %q", doc.Signed.Type)
	}
	if doc.Signed.SchemaVersion != 0 && doc.Signed.SchemaVersion != BootstrapSchemaVersion {
		return nil, fmt.Errorf("unsupported root schema_version %d", doc.Signed.SchemaVersion)
	}
	if doc.Signed.Version <= 0 {
		return nil, fmt.Errorf("root version must be positive")
	}
	if len(doc.Signed.Keys) == 0 {
		return nil, fmt.Errorf("root meta has no keys")
	}
	if _, ok := doc.Signed.Roles[RoleRoot]; !ok {
		return nil, fmt.Errorf("root meta missing root role")
	}
	return &doc, nil
}

// SignedPayload returns canonical bytes of the signed root object.
func (f *RootMetaFile) SignedPayload() ([]byte, error) {
	raw, err := json.Marshal(f.Signed)
	if err != nil {
		return nil, err
	}
	return CanonicalSignedBytes(raw)
}

// VerifyRootMeta checks expiry and threshold signatures against authorized Root keys.
func VerifyRootMeta(doc *RootMetaFile, authorized []VerifyKey, threshold int, now time.Time, expectVersion int64) error {
	if doc == nil {
		return fmt.Errorf("root meta is nil")
	}
	if expectVersion > 0 && doc.Signed.Version != expectVersion {
		return fmt.Errorf("root version %d != expected %d", doc.Signed.Version, expectVersion)
	}
	if strings.TrimSpace(doc.Signed.ExpiresAt) != "" {
		exp, err := time.Parse(time.RFC3339, doc.Signed.ExpiresAt)
		if err != nil {
			return fmt.Errorf("root expires_at: %w", err)
		}
		if now.After(exp) {
			return fmt.Errorf("root meta expired at %s", exp.UTC().Format(time.RFC3339))
		}
	}
	payload, err := doc.SignedPayload()
	if err != nil {
		return err
	}
	if threshold <= 0 {
		threshold = 1
	}
	return VerifyThresholdSignatures(payload, doc.Signatures, authorized, threshold)
}

// RoleKeys returns VerifyKeys authorized for role from the root meta.
func (f *RootMetaFile) RoleKeys(role string) ([]VerifyKey, RoleSpec, error) {
	spec, ok := f.Signed.Roles[role]
	if !ok {
		return nil, RoleSpec{}, fmt.Errorf("role %q not present in root meta", role)
	}
	out := make([]VerifyKey, 0, len(spec.KeyIDs))
	for _, id := range spec.KeyIDs {
		ks, ok := f.Signed.Keys[id]
		if !ok {
			return nil, RoleSpec{}, fmt.Errorf("role %q references missing key %q", role, id)
		}
		pub, err := DecodePublicKey(ks.PublicKey)
		if err != nil {
			return nil, RoleSpec{}, err
		}
		out = append(out, VerifyKey{KeyID: id, PublicKey: pub, Roles: []string{role}})
	}
	return out, spec, nil
}

// RootKeysAsVerify returns the root role keys from this meta (for signing the next version).
func (f *RootMetaFile) RootKeysAsVerify() ([]VerifyKey, RoleSpec, error) {
	return f.RoleKeys(RoleRoot)
}

// SaveRootMetaVersion stores an accepted N.root.json under trustDir/roots/.
func SaveRootMetaVersion(trustDir string, version int64, data []byte) error {
	if trustDir == "" || version <= 0 {
		return nil
	}
	dir := filepath.Join(trustDir, "roots")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, strconv.FormatInt(version, 10)+".root.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(trustDir, "root_version"), []byte(strconv.FormatInt(version, 10)+"\n"), 0644)
}

// LoadRootMetaVersion loads a cached N.root.json.
func LoadRootMetaVersion(trustDir string, version int64) ([]byte, error) {
	path := filepath.Join(trustDir, "roots", strconv.FormatInt(version, 10)+".root.json")
	return os.ReadFile(path)
}

// LoadHighestRootVersion returns the highest accepted root version from disk.
func LoadHighestRootVersion(trustDir string) (int64, error) {
	if trustDir == "" {
		return 0, nil
	}
	data, err := os.ReadFile(filepath.Join(trustDir, "root_version"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}

// DefaultRootMetaFromBootstrap synthesizes an in-memory root v1 from embedded bootstrap.
func DefaultRootMetaFromBootstrap(b BootstrapTrust) *RootMetaFile {
	keys := map[string]KeySpec{}
	for id, k := range b.RootKeys {
		keys[id] = k
	}
	return &RootMetaFile{
		Signed: RootMeta{
			Type:          RootMetaType,
			SchemaVersion: BootstrapSchemaVersion,
			Version:       b.RootVersion,
			Keys:          keys,
			Roles: map[string]RoleSpec{
				RoleRoot: b.RootRole,
				// Business roles are empty until a published N.root.json authorizes them.
			},
		},
	}
}
