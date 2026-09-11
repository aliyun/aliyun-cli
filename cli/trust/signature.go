package trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const SignatureSchema = 1

// SignatureFile is the detached signature document stored next to an artifact
// (e.g. plugin_pkg_index.json.sig).
type SignatureFile struct {
	Schema    int    `json:"schema"`
	KeyID     string `json:"keyid"`
	Algorithm string `json:"algorithm"` // "ed25519"
	Signature string `json:"signature"` // base64 raw Ed25519 signature
	SignedAt  string `json:"signed_at,omitempty"`
}

// Sign creates a detached SignatureFile over payload bytes.
func Sign(keyID string, priv ed25519.PrivateKey, payload []byte, now time.Time) (*SignatureFile, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid ed25519 private key size")
	}
	if keyID == "" {
		return nil, fmt.Errorf("keyid is required")
	}
	sig := ed25519.Sign(priv, payload)
	return &SignatureFile{
		Schema:    SignatureSchema,
		KeyID:     keyID,
		Algorithm: "ed25519",
		Signature: base64.StdEncoding.EncodeToString(sig),
		SignedAt:  now.UTC().Format(time.RFC3339),
	}, nil
}

// MarshalSignature encodes a SignatureFile as JSON.
func MarshalSignature(sig *SignatureFile) ([]byte, error) {
	return json.MarshalIndent(sig, "", "  ")
}

// ParseSignature parses a SignatureFile from JSON bytes.
func ParseSignature(data []byte) (*SignatureFile, error) {
	var sig SignatureFile
	if err := json.Unmarshal(data, &sig); err != nil {
		return nil, fmt.Errorf("parse signature: %w", err)
	}
	if sig.Schema != 0 && sig.Schema != SignatureSchema {
		return nil, fmt.Errorf("unsupported signature schema %d", sig.Schema)
	}
	if !strings.EqualFold(sig.Algorithm, "ed25519") {
		return nil, fmt.Errorf("unsupported signature algorithm %q", sig.Algorithm)
	}
	if sig.KeyID == "" || sig.Signature == "" {
		return nil, fmt.Errorf("signature missing keyid or signature field")
	}
	return &sig, nil
}

// VerifyDetached verifies that sig is a valid Ed25519 signature over payload
// using a key with the given role from keys.
func VerifyDetached(payload []byte, sig *SignatureFile, keys []VerifyKey, role string) error {
	if sig == nil {
		return fmt.Errorf("signature is nil")
	}
	k, ok := FindKey(keys, sig.KeyID, role)
	if !ok {
		return fmt.Errorf("unknown or unauthorized keyid %q for role %q", sig.KeyID, role)
	}
	raw, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(raw) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length %d", len(raw))
	}
	if !ed25519.Verify(k.PublicKey, payload, raw) {
		return fmt.Errorf("signature verification failed for keyid %q", sig.KeyID)
	}
	return nil
}
