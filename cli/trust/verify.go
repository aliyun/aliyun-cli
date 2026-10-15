package trust

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

func DecodeEnvelope(data []byte) (SignedEnvelope, []byte, error) {
	var envelope SignedEnvelope
	if err := strictJSON(data, &envelope); err != nil {
		return envelope, nil, fmt.Errorf("decode signed envelope: %w", err)
	}
	if envelope.SchemaVersion != SchemaVersion {
		return envelope, nil, fmt.Errorf("unsupported envelope schema version %d", envelope.SchemaVersion)
	}
	payload, err := base64.StdEncoding.Strict().DecodeString(envelope.Payload)
	if err != nil {
		return envelope, nil, fmt.Errorf("decode envelope payload: %w", err)
	}
	return envelope, payload, nil
}

func strictJSON(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

// DecodeStrictJSON decodes exactly one JSON value and rejects unknown fields.
func DecodeStrictJSON(data []byte, dst any) error {
	return strictJSON(data, dst)
}

func VerifyEnvelope(data []byte, keys []Key, role Role, purpose string, now time.Time, deny map[string]bool) ([]byte, []string, error) {
	envelope, payload, err := DecodeEnvelope(data)
	if err != nil {
		return nil, nil, err
	}
	if role.Threshold < 1 || role.Threshold > len(role.KeyIDs) {
		return nil, nil, fmt.Errorf("invalid signature threshold")
	}
	allowed := make(map[string]bool, len(role.KeyIDs))
	for _, id := range role.KeyIDs {
		allowed[id] = true
	}
	keyMap := make(map[string]Key, len(keys))
	for _, key := range keys {
		if _, duplicate := keyMap[key.KeyID]; duplicate {
			return nil, nil, fmt.Errorf("duplicate key id %q", key.KeyID)
		}
		keyMap[key.KeyID] = key
	}
	verified := make(map[string]bool)
	for _, signature := range envelope.Signatures {
		if verified[signature.KeyID] || deny[signature.KeyID] || !allowed[signature.KeyID] {
			continue
		}
		key, ok := keyMap[signature.KeyID]
		if !ok || signature.Algorithm != AlgorithmEd25519 || key.Algorithm != AlgorithmEd25519 {
			continue
		}
		if purpose != "" && !contains(key.Purposes, purpose) {
			continue
		}
		if (!key.NotBefore.IsZero() && now.Before(key.NotBefore)) || (!key.NotAfter.IsZero() && !now.Before(key.NotAfter)) {
			continue
		}
		pub, pubErr := base64.StdEncoding.Strict().DecodeString(key.PublicKey)
		sig, sigErr := base64.StdEncoding.Strict().DecodeString(signature.Signature)
		if pubErr != nil || sigErr != nil || len(pub) != ed25519.PublicKeySize || len(sig) != ed25519.SignatureSize {
			continue
		}
		if ed25519.Verify(ed25519.PublicKey(pub), payload, sig) {
			verified[signature.KeyID] = true
		}
	}
	if len(verified) < role.Threshold {
		return nil, nil, fmt.Errorf("signature threshold not met: got %d, need %d", len(verified), role.Threshold)
	}
	ids := make([]string, 0, len(verified))
	for id := range verified {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return payload, ids, nil
}

func VerifyRootUpdate(data []byte, old Root, now time.Time, deny map[string]bool) (Root, error) {
	oldRole, ok := old.Roles["root"]
	if !ok {
		return Root{}, fmt.Errorf("old root role is missing")
	}
	payload, _, err := VerifyEnvelope(data, old.Keys, oldRole, "", now, deny)
	if err != nil {
		return Root{}, fmt.Errorf("old root approval: %w", err)
	}
	var next Root
	if err := strictJSON(payload, &next); err != nil {
		return Root{}, err
	}
	if next.Type != "root" || next.SchemaVersion != SchemaVersion || next.Version != old.Version+1 ||
		next.Authority != old.Authority || !now.Before(next.ExpiresAt) {
		return Root{}, fmt.Errorf("invalid or non-contiguous root update")
	}
	newRole, ok := next.Roles["root"]
	if !ok {
		return Root{}, fmt.Errorf("new root role is missing")
	}
	if _, _, err := VerifyEnvelope(data, next.Keys, newRole, "", now, deny); err != nil {
		return Root{}, fmt.Errorf("new root approval: %w", err)
	}
	return next, nil
}

func VerifyArtifactKeys(data []byte, root Root, minimum uint64, now time.Time, deny map[string]bool) (ArtifactKeys, error) {
	role, ok := root.Roles["artifact-keys"]
	if !ok {
		return ArtifactKeys{}, fmt.Errorf("artifact-keys role is missing")
	}
	payload, _, err := VerifyEnvelope(data, root.Keys, role, "", now, deny)
	if err != nil {
		return ArtifactKeys{}, err
	}
	var result ArtifactKeys
	if err := strictJSON(payload, &result); err != nil {
		return result, err
	}
	if result.Type != "artifact-keys" || result.SchemaVersion != SchemaVersion ||
		result.Authority != root.Authority || result.Version < minimum || !now.Before(result.ExpiresAt) {
		return result, fmt.Errorf("invalid, expired, or rolled-back artifact keys")
	}
	return result, nil
}

func VerifyTyped[T any](data []byte, keys ArtifactKeys, roleName, purpose string, now time.Time, deny map[string]bool) (T, []string, error) {
	var result T
	role, ok := keys.Roles[roleName]
	if !ok {
		return result, nil, fmt.Errorf("%s role is missing", roleName)
	}
	payload, ids, err := VerifyEnvelope(data, keys.Keys, role, purpose, now, deny)
	if err != nil {
		return result, nil, err
	}
	if err := strictJSON(payload, &result); err != nil {
		return result, nil, err
	}
	return result, ids, nil
}

func VerifyDescriptor(data []byte, descriptor ArtifactDescriptor, now time.Time) error {
	if descriptor.Length < 0 || int64(len(data)) != descriptor.Length {
		return fmt.Errorf("artifact length mismatch")
	}
	if !now.Before(descriptor.ExpiresAt) {
		return fmt.Errorf("artifact descriptor expired")
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), strings.TrimPrefix(descriptor.SHA256, "sha256:")) {
		return fmt.Errorf("artifact SHA-256 mismatch")
	}
	return nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
