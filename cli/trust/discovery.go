package trust

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// Fixed Discovery entry for Design B+.
const DefaultDiscoveryURL = "https://api.aliyun.com/.well-known/aliyun-cli-trust.json"

// BuiltinHostAllowlist restricts Discovery-returned metadata hosts.
var BuiltinHostAllowlist = []string{
	"api.aliyun.com",
	"aliyuncli.alicdn.com",
	"pre-cli.aliyun-inc.com",
}

// DiscoveryDocument is the unsigned HTTPS discovery document (location only).
type DiscoveryDocument struct {
	SchemaVersion               int      `json:"schema_version"`
	Authority                   string   `json:"authority"`
	RootMetadataURI             string   `json:"root_metadata_uri"`
	ArtifactKeysURI             string   `json:"artifact_keys_uri"`
	TimestampURI                string   `json:"timestamp_uri"`
	RecoveryPatchURI            string   `json:"recovery_patch_uri,omitempty"`
	OfficialArtifactOrigins     []string `json:"official_artifact_origins"`
	SupportedSignatureAlgorithms []string `json:"supported_signature_algorithms"`
	SupportedArtifactTypes      []string `json:"supported_artifact_types"`
	CacheMaxAgeSeconds          int64    `json:"cache_max_age_seconds,omitempty"`
	ExpiresAt                   string   `json:"expires_at,omitempty"`
}

// ParseDiscovery parses and validates a Discovery document.
func ParseDiscovery(data []byte, now time.Time) (*DiscoveryDocument, error) {
	var doc DiscoveryDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse discovery: %w", err)
	}
	if doc.SchemaVersion != 0 && doc.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported discovery schema_version %d", doc.SchemaVersion)
	}
	if strings.TrimSpace(doc.RootMetadataURI) == "" {
		return nil, fmt.Errorf("discovery missing root_metadata_uri")
	}
	if strings.TrimSpace(doc.ArtifactKeysURI) == "" {
		return nil, fmt.Errorf("discovery missing artifact_keys_uri")
	}
	if strings.TrimSpace(doc.ExpiresAt) != "" {
		exp, err := time.Parse(time.RFC3339, doc.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("discovery expires_at: %w", err)
		}
		if now.After(exp) {
			return nil, fmt.Errorf("discovery expired at %s", exp.UTC().Format(time.RFC3339))
		}
	}
	for _, u := range []string{doc.RootMetadataURI, doc.ArtifactKeysURI, doc.TimestampURI, doc.RecoveryPatchURI} {
		if strings.TrimSpace(u) == "" {
			continue
		}
		if err := ValidateTrustURL(u, BuiltinHostAllowlist); err != nil {
			return nil, err
		}
	}
	for _, o := range doc.OfficialArtifactOrigins {
		if err := ValidateTrustURL(o, BuiltinHostAllowlist); err != nil {
			return nil, fmt.Errorf("official_artifact_origins: %w", err)
		}
	}
	return &doc, nil
}

// ValidateTrustURL enforces HTTPS + host allowlist + no IP/localhost/private.
func ValidateTrustURL(raw string, allowlist []string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", raw, err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("url must be https: %s", raw)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("forbidden host %q", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		return fmt.Errorf("ip hosts are not allowed: %s", host)
	}
	allowed := false
	for _, a := range allowlist {
		a = strings.ToLower(strings.TrimSpace(a))
		if host == a || strings.HasSuffix(host, "."+a) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("host %q not in trust allowlist", host)
	}
	return nil
}

// RootsBaseFromDiscovery returns the roots/ directory URL from Discovery.
func RootsBaseFromDiscovery(doc *DiscoveryDocument) string {
	if doc == nil {
		return ""
	}
	u := strings.TrimSpace(doc.RootMetadataURI)
	if strings.HasSuffix(u, ".json") {
		// If a single file was given, use its directory.
		if i := strings.LastIndex(u, "/"); i >= 0 {
			return u[:i+1]
		}
	}
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u
}
