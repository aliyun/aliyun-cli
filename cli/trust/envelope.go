package trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// MetaSignature is one threshold signature over a signed envelope payload.
type MetaSignature struct {
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
}

// SignedEnvelope is the Design A/B+ wrapper: signatures cover canonical JSON of Signed.
type SignedEnvelope struct {
	Signed     json.RawMessage `json:"signed"`
	Signatures []MetaSignature `json:"signatures"`
}

// CanonicalSignedBytes returns a deterministic encoding of the signed object.
// Prefer the raw signed JSON object when it is already a compact object; otherwise
// re-marshal with sorted object keys via encoding/json (Go map key order is sorted).
func CanonicalSignedBytes(signed json.RawMessage) ([]byte, error) {
	trimmed := strings.TrimSpace(string(signed))
	if trimmed == "" {
		return nil, fmt.Errorf("signed payload is empty")
	}
	// Re-marshal through map[string]any so object keys are sorted stably.
	var v any
	if err := json.Unmarshal(signed, &v); err != nil {
		return nil, fmt.Errorf("decode signed payload: %w", err)
	}
	return marshalCanonical(v)
}

func marshalCanonical(v any) ([]byte, error) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf := []byte{'{'}
		for i, k := range keys {
			if i > 0 {
				buf = append(buf, ',')
			}
			kb, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			buf = append(buf, kb...)
			buf = append(buf, ':')
			vb, err := marshalCanonical(t[k])
			if err != nil {
				return nil, err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, '}')
		return buf, nil
	case []any:
		buf := []byte{'['}
		for i, elem := range t {
			if i > 0 {
				buf = append(buf, ',')
			}
			eb, err := marshalCanonical(elem)
			if err != nil {
				return nil, err
			}
			buf = append(buf, eb...)
		}
		buf = append(buf, ']')
		return buf, nil
	default:
		return json.Marshal(t)
	}
}

// VerifyThresholdSignatures verifies that at least threshold distinct key_ids
// from authorized produce a valid Ed25519 signature over payload.
func VerifyThresholdSignatures(payload []byte, sigs []MetaSignature, authorized []VerifyKey, threshold int) error {
	if threshold <= 0 {
		return fmt.Errorf("invalid signature threshold %d", threshold)
	}
	if len(authorized) == 0 {
		return fmt.Errorf("no authorized keys for threshold verification")
	}
	byID := make(map[string]VerifyKey, len(authorized))
	for _, k := range authorized {
		byID[k.KeyID] = k
	}
	seen := map[string]struct{}{}
	okCount := 0
	for _, s := range sigs {
		if _, dup := seen[s.KeyID]; dup {
			continue
		}
		k, ok := byID[s.KeyID]
		if !ok {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(s.Signature)
		if err != nil || len(raw) != ed25519.SignatureSize {
			continue
		}
		if ed25519.Verify(k.PublicKey, payload, raw) {
			seen[s.KeyID] = struct{}{}
			okCount++
			if okCount >= threshold {
				return nil
			}
		}
	}
	return fmt.Errorf("signature threshold not met: need %d, got %d", threshold, okCount)
}

// SignMetaPayload appends a MetaSignature for keyID over payload.
func SignMetaPayload(sigs *[]MetaSignature, keyID string, priv ed25519.PrivateKey, payload []byte) error {
	if len(priv) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid ed25519 private key size")
	}
	sig := ed25519.Sign(priv, payload)
	*sigs = append(*sigs, MetaSignature{
		KeyID:     keyID,
		Signature: base64.StdEncoding.EncodeToString(sig),
	})
	return nil
}
