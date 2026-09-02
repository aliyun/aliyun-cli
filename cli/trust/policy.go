package trust

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	EnvTrustEnforce = "ALIBABA_CLOUD_CLI_TRUST_ENFORCE"
	// EnvTrustAllowUnsigned is an explicit escape hatch for break-glass / airgap.
	// It never overrides a present-but-invalid signature.
	EnvTrustAllowUnsigned = "ALIBABA_CLOUD_CLI_TRUST_ALLOW_UNSIGNED"
)

// Policy controls how missing signatures are treated.
type Policy struct {
	Enforce        bool // require a valid signature
	AllowUnsigned  bool // explicit legacy/break-glass (ignored when Enforce is true)
	Now            time.Time
	TrustDir       string
	MinRootVersion int64
}

// DefaultPolicy reads environment defaults.
func DefaultPolicy() Policy {
	return Policy{
		Enforce:       envTruthy(os.Getenv(EnvTrustEnforce)),
		AllowUnsigned: envTruthy(os.Getenv(EnvTrustAllowUnsigned)),
		Now:           time.Now(),
		TrustDir:      DefaultTrustDir(),
	}
}

func envTruthy(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ArtifactCheck is the result of verifying a downloaded manifest.
type ArtifactCheck struct {
	Verified  bool
	KeyID     string
	Freshness FreshnessMeta
	Warning   string // non-fatal advisory (e.g. unsigned in transition mode)
}

// VerifyArtifactBytes verifies optional detached signature + freshness for a role.
//
// Rules:
//   - If sigBytes is non-empty: signature MUST verify (never skip).
//   - If sigBytes is empty: fail when Enforce; warn+allow when transition; fail when
//     AllowUnsigned is false and Enforce is false but we still want... actually
//     transition default is allow unsigned with warning when !Enforce.
func VerifyArtifactBytes(payload, sigBytes []byte, role, versionStoreName string, keys []VerifyKey, policy Policy) (*ArtifactCheck, error) {
	now := policy.Now
	if now.IsZero() {
		now = time.Now()
	}

	fresh, err := ParseFreshnessFromJSON(payload)
	if err != nil {
		// Payload might not be JSON object (unlikely for our indexes); treat as no freshness.
		fresh = FreshnessMeta{}
	}

	store := VersionStore{Dir: policy.TrustDir}
	last, _ := store.Load(versionStoreName)
	if err := CheckMonotonicVersion(fresh, last); err != nil {
		return nil, err
	}
	if err := CheckExpiry(fresh, now); err != nil {
		return nil, err
	}

	if len(sigBytes) == 0 {
		if policy.Enforce && !policy.AllowUnsigned {
			return nil, fmt.Errorf("missing signature for %s artifact; refusing to trust unsigned manifest (set %s only for break-glass)",
				role, EnvTrustAllowUnsigned)
		}
		if policy.Enforce && policy.AllowUnsigned {
			// Enforce + explicit allow: still refuse — enforce wins.
			return nil, fmt.Errorf("missing signature for %s artifact; %s cannot override %s",
				role, EnvTrustAllowUnsigned, EnvTrustEnforce)
		}
		warn := fmt.Sprintf("warning: %s manifest is unsigned; authenticity is not verified", role)
		if !policy.AllowUnsigned {
			// transition default
			return &ArtifactCheck{Verified: false, Freshness: fresh, Warning: warn}, nil
		}
		return &ArtifactCheck{Verified: false, Freshness: fresh, Warning: warn}, nil
	}

	sig, err := ParseSignature(sigBytes)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("signature present for %s but CLI has no trusted public keys configured", role)
	}
	if err := VerifyDetached(payload, sig, keys, role); err != nil {
		return nil, err
	}

	if fresh.IndexVersion > 0 {
		_ = store.Save(versionStoreName, fresh.IndexVersion)
	}

	return &ArtifactCheck{
		Verified:  true,
		KeyID:     sig.KeyID,
		Freshness: fresh,
	}, nil
}
