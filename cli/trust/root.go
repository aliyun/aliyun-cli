package trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const RootSchema = 1

// RootDocument authorizes which keyids may sign plugins/upgrade manifests.
// It is itself signed by one or more Root keys (embedded or previously accepted).
type RootDocument struct {
	Schema     int       `json:"schema"`
	Version    int64     `json:"version"`
	ExpiresAt  string    `json:"expires_at,omitempty"`
	Keys       []RootKey `json:"keys"`
	Signatures []RootSig `json:"signatures"`
}

type RootKey struct {
	KeyID     string   `json:"keyid"`
	PublicKey string   `json:"public_key"` // base64 ed25519
	Roles     []string `json:"roles"`
}

type RootSig struct {
	KeyID     string `json:"keyid"`
	Signature string `json:"signature"` // base64 over canonical signed body
}

type rootSignedBody struct {
	Schema    int       `json:"schema"`
	Version   int64     `json:"version"`
	ExpiresAt string    `json:"expires_at,omitempty"`
	Keys      []RootKey `json:"keys"`
}

// SignedPayload returns canonical JSON of the signed portion of root.json.
func (r *RootDocument) SignedPayload() ([]byte, error) {
	body := rootSignedBody{
		Schema:    r.Schema,
		Version:   r.Version,
		ExpiresAt: r.ExpiresAt,
		Keys:      r.Keys,
	}
	return json.Marshal(body)
}

// ParseRoot parses root.json bytes.
func ParseRoot(data []byte) (*RootDocument, error) {
	var root RootDocument
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse root.json: %w", err)
	}
	if root.Schema != 0 && root.Schema != RootSchema {
		return nil, fmt.Errorf("unsupported root schema %d", root.Schema)
	}
	if root.Version <= 0 {
		return nil, fmt.Errorf("root version must be positive")
	}
	if len(root.Keys) == 0 {
		return nil, fmt.Errorf("root.json has no keys")
	}
	if len(root.Signatures) == 0 {
		return nil, fmt.Errorf("root.json has no signatures")
	}
	return &root, nil
}

// VerifyRoot checks signatures against trusted root keys and expiry.
func VerifyRoot(root *RootDocument, rootKeys []VerifyKey, now time.Time, minVersion int64) error {
	if root == nil {
		return fmt.Errorf("root is nil")
	}
	if minVersion > 0 && root.Version < minVersion {
		return fmt.Errorf("root version %d below minimum acceptable %d", root.Version, minVersion)
	}
	if strings.TrimSpace(root.ExpiresAt) != "" {
		exp, err := time.Parse(time.RFC3339, root.ExpiresAt)
		if err != nil {
			return fmt.Errorf("root expires_at: %w", err)
		}
		if now.After(exp) {
			return fmt.Errorf("root.json expired at %s", exp.UTC().Format(time.RFC3339))
		}
	}
	payload, err := root.SignedPayload()
	if err != nil {
		return err
	}
	verified := false
	for _, s := range root.Signatures {
		k, ok := FindKey(rootKeys, s.KeyID, RoleRoot)
		if !ok {
			// Root keys may also be listed with empty roles.
			k, ok = FindKey(rootKeys, s.KeyID, "")
			if !ok {
				for _, cand := range rootKeys {
					if cand.KeyID == s.KeyID {
						k = cand
						ok = true
						break
					}
				}
			}
		}
		if !ok {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(s.Signature)
		if err != nil || len(raw) != ed25519.SignatureSize {
			continue
		}
		if ed25519.Verify(k.PublicKey, payload, raw) {
			verified = true
			break
		}
	}
	if !verified {
		return fmt.Errorf("root.json signature verification failed")
	}
	return nil
}

// KeysFromRoot converts authorized keys in root.json into VerifyKeys.
func KeysFromRoot(root *RootDocument) ([]VerifyKey, error) {
	var out []VerifyKey
	for _, rk := range root.Keys {
		pub, err := DecodePublicKey(rk.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("root key %q: %w", rk.KeyID, err)
		}
		out = append(out, VerifyKey{KeyID: rk.KeyID, PublicKey: pub, Roles: append([]string(nil), rk.Roles...)})
	}
	return out, nil
}

// SignRoot attaches a Root signature using the given root private key.
func SignRoot(root *RootDocument, keyID string, priv ed25519.PrivateKey) error {
	payload, err := root.SignedPayload()
	if err != nil {
		return err
	}
	sig := ed25519.Sign(priv, payload)
	root.Signatures = append(root.Signatures, RootSig{
		KeyID:     keyID,
		Signature: base64.StdEncoding.EncodeToString(sig),
	})
	return nil
}

// LoadCachedRoot reads a previously accepted root snapshot from the trust dir.
func LoadCachedRoot(trustDir string) (*RootDocument, []byte, error) {
	if trustDir == "" {
		return nil, nil, os.ErrNotExist
	}
	path := filepath.Join(trustDir, "root.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	root, err := ParseRoot(data)
	if err != nil {
		return nil, nil, err
	}
	return root, data, nil
}

// SaveCachedRoot persists an accepted root.json.
func SaveCachedRoot(trustDir string, data []byte) error {
	if trustDir == "" {
		return nil
	}
	if err := os.MkdirAll(trustDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(trustDir, "root.json"), data, 0644)
}

// DeriveTrustRootURL maps a package index URL to the sibling trust/root.json.
// Examples:
//
//	.../plugins-pb/plugin_pkg_index.json -> .../trust/root.json
//	.../plugins/plugin_pkg_index.json    -> .../trust/root.json
func DeriveTrustRootURL(pkgIndexURL string) string {
	u := strings.TrimSpace(pkgIndexURL)
	if u == "" {
		return ""
	}
	replacements := []struct{ old, neu string }{
		{"/plugins-pb/plugin_pkg_index.json", "/trust/root.json"},
		{"/plugins/plugin_pkg_index.json", "/trust/root.json"},
		{"/plugin_pkg_index.json", "/trust/root.json"},
	}
	for _, r := range replacements {
		if strings.HasSuffix(u, r.old) {
			return strings.TrimSuffix(u, r.old) + r.neu
		}
	}
	// Fallback: same directory
	if i := strings.LastIndex(u, "/"); i >= 0 {
		return u[:i+1] + "root.json"
	}
	return ""
}
