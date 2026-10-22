package trust

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
)

type Config struct {
	CacheDir      string
	DiscoveryURL  string
	SourceBase    string
	HTTPClient    *http.Client
	Now           func() time.Time
	MetadataHosts []string
	RecoveryHosts []string
	BootstrapRoot []byte
	RecoveryKeys  []Key
	RecoveryRole  Role
	AllowTestHTTP bool
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	if cfg.DiscoveryURL == "" {
		cfg.DiscoveryURL = defaultDiscoveryURL()
	}
	if strings.TrimSpace(cfg.SourceBase) == "" {
		cfg.SourceBase = defaultSourceBase()
	}
	cfg.SourceBase = strings.TrimRight(strings.TrimSpace(cfg.SourceBase), "/")
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if len(cfg.MetadataHosts) == 0 {
		cfg.MetadataHosts = append([]string(nil), MetadataHostAllowlist...)
	}
	if len(cfg.RecoveryHosts) == 0 {
		cfg.RecoveryHosts = append([]string(nil), RecoveryHostAllowlist...)
	}
	if host := sourceHost(cfg.SourceBase); host != "" {
		cfg.MetadataHosts = appendHost(cfg.MetadataHosts, host)
		if !directRepositoryHost(host, cfg.DiscoveryURL) {
			cfg.RecoveryHosts = appendHost(cfg.RecoveryHosts, host)
		}
	}
	if len(cfg.BootstrapRoot) == 0 {
		cfg.BootstrapRoot = builtinMaterial(
			BuiltinRootEnvelope, BuiltinRootEnvelopeBase64,
		)
	}
	if len(cfg.RecoveryKeys) == 0 {
		_ = strictJSON(
			builtinMaterial(BuiltinRecoveryKeysJSON, BuiltinRecoveryKeysBase64),
			&cfg.RecoveryKeys,
		)
	}
	if cfg.RecoveryRole.Threshold == 0 {
		_ = strictJSON(
			builtinMaterial(BuiltinRecoveryRoleJSON, BuiltinRecoveryRoleBase64),
			&cfg.RecoveryRole,
		)
	}
	if cfg.CacheDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cfg.CacheDir = filepath.Join(home, ".aliyun", "trust")
		}
	}
	return &Client{cfg: cfg}
}

func builtinMaterial(plain, encoded string) []byte {
	if strings.TrimSpace(encoded) != "" {
		if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
			return decoded
		}
		return nil
	}
	return []byte(plain)
}

func DiscoveryURLForSite(site string) string {
	if pluginsettings.NormalizeSite(site) == pluginsettings.SiteIntl {
		return DiscoveryURLIntl
	}
	return DiscoveryURLCN
}

func defaultDiscoveryURL() string {
	var settings *pluginsettings.PluginSettings
	if home, err := os.UserHomeDir(); err == nil {
		if loaded, loadErr := pluginsettings.Load(filepath.Join(home, ".aliyun")); loadErr == nil {
			settings = loaded
		}
	}
	return DiscoveryURLForSite(pluginsettings.EffectiveSite(settings))
}

func defaultSourceBase() string {
	var settings *pluginsettings.PluginSettings
	if home, err := os.UserHomeDir(); err == nil {
		if loaded, loadErr := pluginsettings.Load(filepath.Join(home, ".aliyun")); loadErr == nil {
			settings = loaded
		}
	}
	if source := pluginsettings.EffectiveSourceBase(settings); source != "" {
		return source
	}
	return "https://aliyuncli.alicdn.com/plugins-v2"
}

func (c *Client) FetchArtifact(ctx context.Context, purpose, roleName string) (*VerifiedArtifact, error) {
	if len(c.cfg.BootstrapRoot) == 0 {
		return nil, fmt.Errorf("artifact trust bootstrap root is not configured")
	}
	now := c.cfg.Now().UTC()
	discovery, err := c.loadDiscovery(ctx, purpose, now)
	if err != nil {
		return nil, err
	}
	authority, err := checkedOrigin(discovery.Authority, c.cfg.AllowTestHTTP)
	if err != nil {
		return nil, fmt.Errorf("invalid discovery authority: %w", err)
	}
	if err := c.validateURL(discovery.RootMetadataURI, c.cfg.MetadataHosts); err != nil {
		return nil, err
	}
	if err := c.validateURL(discovery.ArtifactKeysURI, c.cfg.MetadataHosts); err != nil {
		return nil, err
	}
	if c.directRepositoryTrust() {
		discovery.TimestampURI = c.repositoryTimestampURL()
	}
	if err := c.validateURL(discovery.TimestampURI, c.cfg.MetadataHosts); err != nil {
		return nil, err
	}

	root, err := c.bootstrapRoot(now)
	if err != nil {
		return nil, err
	}
	origin := ""
	for _, candidate := range discovery.OfficialArtifactOrigins {
		if _, e := checkedOrigin(candidate, c.cfg.AllowTestHTTP); e == nil {
			origin = candidate
			break
		}
	}
	if origin == "" {
		return nil, fmt.Errorf("discovery has no valid official artifact origin")
	}
	cacheDir := filepath.Join(
		c.cfg.CacheDir, digest(authority), digest(c.cfg.SourceBase), digest(origin),
	)
	state := c.readState(cacheDir)
	deny := make(map[string]bool)
	for _, id := range state.DenyKeyIDs {
		deny[id] = true
	}
	if state.RootVersion > root.Version {
		root, err = c.loadPersistedRoot(cacheDir, root, state, now, deny)
		if err != nil {
			return nil, fmt.Errorf("load persisted root: %w", err)
		}
	}
	if discovery.RecoveryPatchURI != "" {
		if err := c.validateURL(discovery.RecoveryPatchURI, c.cfg.RecoveryHosts); err != nil {
			return nil, err
		}
		root, err = c.CheckAndApplyRecovery(ctx, discovery.RecoveryPatchURI, cacheDir, root, &state)
		if err != nil {
			return nil, fmt.Errorf("apply recovery: %w", err)
		}
		deny = make(map[string]bool)
		for _, id := range state.DenyKeyIDs {
			deny[id] = true
		}
	}
	root, err = c.updateRoots(ctx, discovery.RootMetadataURI, cacheDir, root, now, deny)
	if err != nil {
		return nil, err
	}
	state.RootVersion = root.Version

	keysCache := filepath.Join(cacheDir, "artifact-keys.json")
	keysBytes, keysRemote, err := c.fetchWithLKG(ctx, discovery.ArtifactKeysURI, keysCache, c.cfg.MetadataHosts)
	if err != nil {
		return nil, err
	}
	keys, err := VerifyArtifactKeys(keysBytes, root, state.KeysVersion, now, deny)
	if err != nil && keysRemote {
		if cached, readErr := os.ReadFile(keysCache); readErr == nil {
			if cachedKeys, verifyErr := VerifyArtifactKeys(cached, root, state.KeysVersion, now, deny); verifyErr == nil {
				keysBytes, keys, keysRemote, err = cached, cachedKeys, false, nil
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("verify artifact keys: %w", err)
	}
	if keysRemote {
		if err := atomicWrite(keysCache, keysBytes, 0600); err != nil {
			return nil, err
		}
	}
	state.KeysVersion = keys.Version

	timestampCache := filepath.Join(cacheDir, "timestamp.json")
	timestampBytes, timestampRemote, err := c.fetchWithLKG(ctx, discovery.TimestampURI, timestampCache, c.cfg.MetadataHosts)
	if err != nil {
		return nil, err
	}
	timestamp, _, err := VerifyTyped[Timestamp](timestampBytes, keys, "timestamp", "timestamp", now, deny)
	timestampValid := func(value Timestamp, verifyErr error) bool {
		return verifyErr == nil && value.Type == "timestamp" && value.SchemaVersion == SchemaVersion &&
			value.Authority == root.Authority && value.Version >= state.TimestampVersion && now.Before(value.ExpiresAt)
	}
	if !timestampValid(timestamp, err) && timestampRemote {
		if cached, readErr := os.ReadFile(timestampCache); readErr == nil {
			if cachedTimestamp, _, verifyErr := VerifyTyped[Timestamp](cached, keys, "timestamp", "timestamp", now, deny); timestampValid(cachedTimestamp, verifyErr) {
				timestampBytes, timestamp, timestampRemote, err = cached, cachedTimestamp, false, nil
			}
		}
	}
	if !timestampValid(timestamp, err) {
		return nil, fmt.Errorf("verify timestamp: invalid, expired, or rolled back")
	}
	if timestampRemote {
		if err := atomicWrite(timestampCache, timestampBytes, 0600); err != nil {
			return nil, err
		}
	}
	state.TimestampVersion = timestamp.Version
	descriptor, ok := timestamp.Artifacts[purpose]
	if !ok {
		return nil, fmt.Errorf("timestamp has no %s descriptor", purpose)
	}
	if state.ArtifactVersions == nil {
		state.ArtifactVersions = make(map[string]uint64)
	}
	minimumVersion := state.ArtifactVersions[purpose]
	if purpose == PurposePluginIndex && state.IndexVersion > minimumVersion {
		minimumVersion = state.IndexVersion
	}
	if descriptor.Version < minimumVersion {
		return nil, fmt.Errorf("artifact version rollback")
	}
	artifactHosts := discovery.OfficialArtifactOrigins
	if c.directRepositoryTrust() {
		origin, originErr := sourceOrigin(c.cfg.SourceBase)
		if originErr != nil {
			return nil, originErr
		}
		artifactHosts = []string{origin}
	}
	if err := c.validateURL(descriptor.URL, artifactHosts); err != nil {
		return nil, err
	}
	artifactCache := filepath.Join(cacheDir, purpose+".envelope.json")
	envelopeBytes, envelopeRemote, err := c.fetchWithLKG(
		ctx, descriptor.URL, artifactCache, artifactHosts,
	)
	if err != nil {
		return nil, err
	}
	payload, ids, err := c.verifyArtifactEnvelope(envelopeBytes, keys, roleName, purpose, now, deny)
	if err != nil {
		return nil, err
	}
	if err := VerifyDescriptor(payload, descriptor, now); err != nil {
		return nil, err
	}
	state.ArtifactVersions[purpose] = descriptor.Version
	if purpose == PurposePluginIndex {
		state.IndexVersion = descriptor.Version
	}
	if envelopeRemote {
		if err := atomicWrite(artifactCache, envelopeBytes, 0600); err != nil {
			return nil, err
		}
	}
	if err := atomicWrite(filepath.Join(cacheDir, "trust-state.json"), mustJSON(state), 0600); err != nil {
		return nil, err
	}
	return &VerifiedArtifact{Bytes: payload, Descriptor: descriptor, KeyIDs: ids, SignedAt: now}, nil
}

func (c *Client) loadDiscovery(ctx context.Context, purpose string, now time.Time) (Discovery, error) {
	var discovery Discovery
	discoveryURL, err := c.discoveryURL()
	if err != nil {
		return discovery, err
	}
	cache := filepath.Join(c.cfg.CacheDir, "discovery", digest(discoveryURL)+".json")
	data, remote, err := c.fetchWithLKG(ctx, discoveryURL, cache, c.cfg.MetadataHosts)
	if err != nil {
		return discovery, fmt.Errorf("fetch discovery: %w", err)
	}
	validate := func(raw []byte) error {
		if err := strictJSON(raw, &discovery); err != nil {
			return fmt.Errorf("decode discovery: %w", err)
		}
		if discovery.SchemaVersion != SchemaVersion ||
			discovery.SourceBase != c.advertisedSourceBase() ||
			!now.Before(discovery.ExpiresAt) ||
			discovery.CacheMaxAgeSeconds <= 0 ||
			!contains(discovery.SupportedSignatureAlgorithms, AlgorithmEd25519) ||
			!contains(discovery.SupportedArtifactTypes, purpose) {
			return fmt.Errorf("invalid or expired discovery")
		}
		return nil
	}
	if err := validate(data); err != nil {
		if !remote {
			return discovery, err
		}
		cached, readErr := os.ReadFile(cache)
		if readErr != nil {
			return discovery, err
		}
		if cachedErr := validate(cached); cachedErr != nil {
			return discovery, err
		}
		return discovery, nil
	}
	if remote {
		if err := atomicWrite(cache, data, 0600); err != nil {
			return discovery, err
		}
	}
	return discovery, nil
}

func (c *Client) discoveryURL() (string, error) {
	source, err := url.Parse(c.cfg.SourceBase)
	if err != nil || source.Host == "" || source.User != nil ||
		source.RawQuery != "" || source.Fragment != "" ||
		(source.Scheme != "https" && !(c.cfg.AllowTestHTTP && source.Scheme == "http")) {
		return "", fmt.Errorf("invalid plugin source-base")
	}
	discovery, err := url.Parse(c.cfg.DiscoveryURL)
	if err != nil {
		return "", fmt.Errorf("invalid discovery URL: %w", err)
	}
	query := discovery.Query()
	query.Set("source_base", c.advertisedSourceBase())
	discovery.RawQuery = query.Encode()
	return discovery.String(), nil
}

func (c *Client) directRepositoryTrust() bool {
	return directRepositoryHost(sourceHost(c.cfg.SourceBase), c.cfg.DiscoveryURL)
}

func (c *Client) advertisedSourceBase() string {
	if c.directRepositoryTrust() {
		return "https://aliyuncli.alicdn.com/plugins-v2"
	}
	return c.cfg.SourceBase
}

func (c *Client) repositoryTimestampURL() string {
	return strings.TrimRight(c.cfg.SourceBase, "/") + "/trust/timestamp.json"
}

func directRepositoryHost(sourceHostname, discoveryURL string) bool {
	sourceHostname = strings.ToLower(strings.TrimSuffix(sourceHostname, "."))
	if sourceHostname == "" || sourceHostname == "aliyuncli.alicdn.com" {
		return false
	}
	return sourceHostname != sourceHost(discoveryURL)
}

func sourceOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid plugin source-base")
	}
	return u.Scheme + "://" + u.Host, nil
}

func (c *Client) loadPersistedRoot(cacheDir string, root Root, state State, now time.Time, deny map[string]bool) (Root, error) {
	if state.RecoveryEpoch > 0 {
		data, err := os.ReadFile(filepath.Join(cacheDir, "recovery-patch.json"))
		if err != nil {
			return root, err
		}
		if len(c.cfg.RecoveryKeys) == 0 || c.cfg.RecoveryRole.Threshold == 0 {
			return root, fmt.Errorf("recovery trust anchor is not configured")
		}
		payload, _, err := VerifyEnvelope(data, c.cfg.RecoveryKeys, c.cfg.RecoveryRole, "recovery", now, nil)
		if err != nil {
			return root, err
		}
		var patch RecoveryPatch
		if err := strictJSON(payload, &patch); err != nil {
			return root, err
		}
		if patch.Type != "recovery" || patch.SchemaVersion != SchemaVersion ||
			patch.RecoveryEpoch != state.RecoveryEpoch {
			return root, fmt.Errorf("persisted recovery patch does not match trust state")
		}
		replacement, err := c.parseTrustedRoot([]byte(patch.NewBootstrapRoot), now, deny)
		if err != nil {
			return root, err
		}
		if replacement.Authority != root.Authority || replacement.Version != state.RootVersion {
			return root, fmt.Errorf("persisted recovery root does not match trust state")
		}
		return replacement, nil
	}

	for version := root.Version + 1; version <= state.RootVersion; version++ {
		data, err := os.ReadFile(filepath.Join(cacheDir, fmt.Sprintf("%d.root.json", version)))
		if err != nil {
			return root, err
		}
		next, err := VerifyRootUpdate(data, root, now, deny)
		if err != nil {
			return root, err
		}
		root = next
	}
	return root, nil
}

func (c *Client) verifyArtifactEnvelope(data []byte, keys ArtifactKeys, roleName, purpose string, now time.Time, deny map[string]bool) ([]byte, []string, error) {
	role, ok := keys.Roles[roleName]
	if !ok {
		return nil, nil, fmt.Errorf("%s role is missing", roleName)
	}
	return VerifyEnvelope(data, keys.Keys, role, purpose, now, deny)
}

func (c *Client) bootstrapRoot(now time.Time) (Root, error) {
	envelope, payload, err := DecodeEnvelope(c.cfg.BootstrapRoot)
	if err != nil {
		return Root{}, fmt.Errorf("invalid bootstrap root: %w", err)
	}
	var root Root
	if err := strictJSON(payload, &root); err != nil {
		return root, err
	}
	role, ok := root.Roles["root"]
	if !ok {
		return root, fmt.Errorf("bootstrap root role is missing")
	}
	if _, _, err := VerifyEnvelope(mustJSON(envelope), root.Keys, role, "", now, nil); err != nil {
		return root, fmt.Errorf("bootstrap root self-signature: %w", err)
	}
	if root.Type != "root" || root.SchemaVersion != SchemaVersion || root.Version == 0 || !now.Before(root.ExpiresAt) {
		return root, fmt.Errorf("invalid or expired bootstrap root")
	}
	return root, nil
}

func (c *Client) parseTrustedRoot(data []byte, now time.Time, deny map[string]bool) (Root, error) {
	_, payload, err := DecodeEnvelope(data)
	if err != nil {
		return Root{}, err
	}
	var root Root
	if err := strictJSON(payload, &root); err != nil {
		return root, err
	}
	role := root.Roles["root"]
	if _, _, err := VerifyEnvelope(data, root.Keys, role, "", now, deny); err != nil {
		return root, err
	}
	if root.Type != "root" || root.SchemaVersion != SchemaVersion || root.Version == 0 || !now.Before(root.ExpiresAt) {
		return root, fmt.Errorf("invalid or expired trusted root")
	}
	return root, nil
}

func (c *Client) updateRoots(ctx context.Context, base, cacheDir string, root Root, now time.Time, deny map[string]bool) (Root, error) {
	for {
		nextURL := strings.TrimRight(base, "/") + fmt.Sprintf("/%d.root.json", root.Version+1)
		data, status, err := c.fetchStatus(ctx, nextURL, c.cfg.MetadataHosts)
		if err != nil {
			return root, nil
		}
		if status == http.StatusNotFound {
			return root, nil
		}
		if status != http.StatusOK {
			return root, nil
		}
		next, err := VerifyRootUpdate(data, root, now, deny)
		if err != nil {
			return root, nil
		}
		if err := atomicWrite(filepath.Join(cacheDir, fmt.Sprintf("%d.root.json", next.Version)), data, 0600); err != nil {
			return root, err
		}
		root = next
	}
}

// CheckAndApplyRecovery checks the independently signed recovery channel before
// a trust-changing operation. Root keys are deliberately not consulted here.
func (c *Client) CheckAndApplyRecovery(ctx context.Context, recoveryURL, cacheDir string, root Root, state *State) (Root, error) {
	data, status, err := c.fetchStatus(ctx, recoveryURL, c.cfg.RecoveryHosts)
	if err != nil {
		return root, err
	}
	if status == http.StatusNotFound || status == http.StatusNoContent {
		return root, nil
	}
	if status != http.StatusOK {
		return root, fmt.Errorf("recovery endpoint returned HTTP %d", status)
	}
	if len(c.cfg.RecoveryKeys) == 0 || c.cfg.RecoveryRole.Threshold == 0 {
		return root, fmt.Errorf("recovery trust anchor is not configured")
	}
	now := c.cfg.Now().UTC()
	payload, _, err := VerifyEnvelope(data, c.cfg.RecoveryKeys, c.cfg.RecoveryRole, "recovery", now, nil)
	if err != nil {
		return root, err
	}
	var patch RecoveryPatch
	if err := strictJSON(payload, &patch); err != nil {
		return root, err
	}
	if patch.RecoveryEpoch < state.RecoveryEpoch {
		return root, fmt.Errorf("recovery epoch rollback")
	}
	if patch.RecoveryEpoch == state.RecoveryEpoch {
		return root, nil
	}
	if patch.Type != "recovery" || patch.SchemaVersion != SchemaVersion ||
		patch.PatchEpoch < state.PatchEpoch ||
		now.Before(patch.NotBefore) || !now.Before(patch.ExpiresAt) ||
		root.Version < patch.AffectedRootMin || root.Version > patch.AffectedRootMax {
		return root, fmt.Errorf("invalid, expired, or rolled-back recovery patch")
	}
	replacementBytes := []byte(patch.NewBootstrapRoot)
	replacement, err := c.parseTrustedRoot(replacementBytes, now, nil)
	if err != nil {
		return root, fmt.Errorf("invalid replacement root: %w", err)
	}
	if replacement.Authority != root.Authority {
		return root, fmt.Errorf("replacement root authority mismatch")
	}
	for _, key := range replacement.Keys {
		if contains(patch.DenyKeyIDs, key.KeyID) {
			return root, fmt.Errorf("replacement root contains denied key %q", key.KeyID)
		}
	}

	deny := make(map[string]bool, len(state.DenyKeyIDs)+len(patch.DenyKeyIDs))
	for _, id := range state.DenyKeyIDs {
		deny[id] = true
	}
	for _, id := range patch.DenyKeyIDs {
		deny[id] = true
	}
	state.DenyKeyIDs = state.DenyKeyIDs[:0]
	for id := range deny {
		state.DenyKeyIDs = append(state.DenyKeyIDs, id)
	}
	sort.Strings(state.DenyKeyIDs)
	state.RecoveryEpoch = patch.RecoveryEpoch
	state.PatchEpoch = patch.PatchEpoch
	state.RootVersion = replacement.Version
	state.KeysVersion = 0
	state.TimestampVersion = 0
	state.IndexVersion = 0
	state.ArtifactVersions = nil

	authorityCacheDir := filepath.Dir(cacheDir)
	for _, rawOrigin := range patch.ClearOrigins {
		origin, err := checkedOrigin(rawOrigin, c.cfg.AllowTestHTTP)
		if err != nil {
			return root, fmt.Errorf("invalid recovery clear origin: %w", err)
		}
		if err := os.RemoveAll(filepath.Join(authorityCacheDir, digest(origin))); err != nil {
			return root, err
		}
	}
	for _, name := range []string{
		"artifact-keys.json", "timestamp.json",
		PurposePluginIndex + ".envelope.json",
		PurposeUpgrade + ".envelope.json",
		PurposeSecurityPatch + ".envelope.json",
	} {
		if err := os.Remove(filepath.Join(cacheDir, name)); err != nil && !os.IsNotExist(err) {
			return root, err
		}
	}
	if err := atomicWrite(filepath.Join(cacheDir, fmt.Sprintf("%d.root.json", replacement.Version)), replacementBytes, 0600); err != nil {
		return root, err
	}
	if err := atomicWrite(filepath.Join(cacheDir, "recovery-patch.json"), data, 0600); err != nil {
		return root, err
	}
	if err := atomicWrite(filepath.Join(cacheDir, "trust-state.json"), mustJSON(state), 0600); err != nil {
		return root, err
	}
	return replacement, nil
}

func (c *Client) fetchWithLKG(ctx context.Context, rawURL, cache string, hosts []string) ([]byte, bool, error) {
	data, err := c.fetch(ctx, rawURL, hosts)
	if err == nil {
		return data, true, nil
	}
	cached, cacheErr := os.ReadFile(cache)
	if cacheErr != nil {
		return nil, false, err
	}
	return cached, false, nil
}

func (c *Client) fetch(ctx context.Context, rawURL string, hosts []string) ([]byte, error) {
	data, status, err := c.fetchStatus(ctx, rawURL, hosts)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", status)
	}
	return data, nil
}

func (c *Client) fetchStatus(ctx context.Context, rawURL string, hosts []string) ([]byte, int, error) {
	if err := c.validateURL(rawURL, hosts); err != nil {
		return nil, 0, err
	}
	client := *c.cfg.HTTPClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return c.validateURL(req.URL.String(), hosts)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, (16<<20)+1))
	if err == nil && len(data) > 16<<20 {
		err = fmt.Errorf("trust response exceeds size limit")
	}
	return data, resp.StatusCode, err
}

func (c *Client) validateURL(raw string, hosts []string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("invalid trust URL")
	}
	if u.RawQuery != "" {
		query := u.Query()
		values, ok := query["source_base"]
		if !ok || len(query) != 1 || len(values) != 1 || values[0] != c.cfg.SourceBase {
			return fmt.Errorf("invalid trust URL query")
		}
	}
	if u.Scheme != "https" && !(c.cfg.AllowTestHTTP && u.Scheme == "http" && isLoopback(u.Hostname())) {
		return fmt.Errorf("trust URL must use HTTPS")
	}
	if net.ParseIP(u.Hostname()) != nil || isLoopback(u.Hostname()) || !hostAllowed(u.Hostname(), hosts) {
		if !(c.cfg.AllowTestHTTP && isLoopback(u.Hostname())) {
			return fmt.Errorf("trust URL host is not allowed")
		}
	}
	return nil
}

func checkedOrigin(raw string, allowTestHTTP bool) (string, error) {
	u, err := url.Parse(raw)
	testOrigin := allowTestHTTP && u != nil && u.Scheme == "http" && isLoopback(u.Hostname())
	if err != nil || (u.Scheme != "https" && !testOrigin) || u.Host == "" || u.User != nil || u.Path != "" ||
		u.RawQuery != "" || u.Fragment != "" || (!testOrigin && (net.ParseIP(u.Hostname()) != nil || isLoopback(u.Hostname()))) {
		return "", fmt.Errorf("origin must be a public HTTPS origin")
	}
	return u.Scheme + "://" + u.Host, nil
}

func sourceHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
}

func appendHost(hosts []string, host string) []string {
	if host == "" || hostAllowed(host, hosts) {
		return hosts
	}
	return append(hosts, host)
}

func hostAllowed(host string, values []string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, raw := range values {
		u, err := url.Parse(raw)
		candidate := raw
		if err == nil && u.Hostname() != "" {
			candidate = u.Hostname()
		}
		if host == strings.ToLower(strings.TrimSuffix(candidate, ".")) {
			return true
		}
	}
	return false
}

func isLoopback(host string) bool {
	return strings.EqualFold(host, "localhost") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}
func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }

func (c *Client) readState(dir string) State {
	var state State
	if data, err := os.ReadFile(filepath.Join(dir, "trust-state.json")); err == nil {
		_ = strictJSON(data, &state)
	}
	return state
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		return err
	}
	return os.Rename(name, path)
}
