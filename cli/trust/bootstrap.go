package trust

// Embedded bootstrap trust anchors for Design A (builtin Root + Recovery).
//
// Public keys only. Private seeds are offline and never shipped in the CLI.
// The key material below is a development bootstrap set for local/CI demos until
// production Root/Recovery keys are cut and substituted at release time.

const (
	BootstrapSchemaVersion = 1
	BootstrapRootVersion   = 1
	BootstrapRootThreshold = 2
	BootstrapRecoveryThreshold = 2
)

// BootstrapTrust is the offline trust anchor baked into the CLI binary.
type BootstrapTrust struct {
	SchemaVersion int
	RootVersion   int64
	RootRole      RoleSpec
	RootKeys      map[string]KeySpec
	RecoveryRole  RoleSpec
	RecoveryKeys  map[string]KeySpec
}

// RoleSpec describes a threshold signing role.
type RoleSpec struct {
	Threshold int
	KeyIDs    []string
}

// KeySpec is an Ed25519 public key entry.
type KeySpec struct {
	Algorithm string // "Ed25519"
	PublicKey string // standard base64
}

// EmbeddedBootstrap returns the compiled-in Root and Recovery anchors.
func EmbeddedBootstrap() BootstrapTrust {
	return BootstrapTrust{
		SchemaVersion: BootstrapSchemaVersion,
		RootVersion:   BootstrapRootVersion,
		RootRole: RoleSpec{
			Threshold: BootstrapRootThreshold,
			KeyIDs:    []string{"root-a", "root-b", "root-c"},
		},
		RootKeys: map[string]KeySpec{
			"root-a": {Algorithm: "Ed25519", PublicKey: "w5o+mm2S3+BnIC6VRmDIkpF2MdCD/3/WVckw7W4aSGc="},
			"root-b": {Algorithm: "Ed25519", PublicKey: "8yoxXpLE6JrQRayPKu8ac8WAdCoLM3hzNE49xIuGQ10="},
			"root-c": {Algorithm: "Ed25519", PublicKey: "Ia+1RKbG4xPWL5HBzmzI6f0zkYK/510/XqHdx48U8MA="},
		},
		RecoveryRole: RoleSpec{
			Threshold: BootstrapRecoveryThreshold,
			KeyIDs:    []string{"recovery-a", "recovery-b", "recovery-c"},
		},
		RecoveryKeys: map[string]KeySpec{
			"recovery-a": {Algorithm: "Ed25519", PublicKey: "R9gdsE0EZkzuNulOMCOPuTVkkCltrvKkJjgdGYQ59qc="},
			"recovery-b": {Algorithm: "Ed25519", PublicKey: "yGHg4dWwAtXSh2jEgN+a6h183ez2YHA7Mo9qDOcxT24="},
			"recovery-c": {Algorithm: "Ed25519", PublicKey: "xynwp1HpVjXibg+pqBuXKJJLFw28q0RGE7CKzSKmIM0="},
		},
	}
}

// RootVerifyKeys converts bootstrap Root keys into VerifyKey entries.
func (b BootstrapTrust) RootVerifyKeys() ([]VerifyKey, error) {
	return keyMapToVerifyKeys(b.RootKeys, b.RootRole.KeyIDs, RoleRoot)
}

// RecoveryVerifyKeys converts bootstrap Recovery keys into VerifyKey entries.
func (b BootstrapTrust) RecoveryVerifyKeys() ([]VerifyKey, error) {
	return keyMapToVerifyKeys(b.RecoveryKeys, b.RecoveryRole.KeyIDs, RoleRecovery)
}

func keyMapToVerifyKeys(keys map[string]KeySpec, order []string, role string) ([]VerifyKey, error) {
	out := make([]VerifyKey, 0, len(order))
	for _, id := range order {
		spec, ok := keys[id]
		if !ok {
			continue
		}
		pub, err := DecodePublicKey(spec.PublicKey)
		if err != nil {
			return nil, err
		}
		out = append(out, VerifyKey{KeyID: id, PublicKey: pub, Roles: []string{role}})
	}
	return out, nil
}
