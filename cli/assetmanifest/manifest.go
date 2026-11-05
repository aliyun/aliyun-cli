package assetmanifest

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
	"golang.org/x/mod/semver"
)

const (
	FileName          = "cli-assets-v1.json"
	Format            = "aliyun-cli-assets v1"
	Purpose           = "external-cli"
	SourceEnv         = "ALIBABA_CLOUD_CLI_ASSET_MANIFEST_URL"
	maxManifestBytes  = 2 * 1024 * 1024
	maxSignatureBytes = 4096
	maxStateBytes     = 64 * 1024
)

type Entry struct {
	Product    string `json:"product"`
	Purpose    string `json:"purpose"`
	Version    string `json:"version"`
	Platform   string `json:"platform"`
	Asset      string `json:"asset"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
	Size       int64  `json:"size"`
	MinVersion string `json:"min_version"`
}

type Manifest struct {
	Format    string    `json:"format"`
	Sequence  uint64    `json:"sequence"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Assets    []Entry   `json:"assets"`
}

type Request struct {
	Product, Purpose, Version, Platform, Asset, URL, CurrentVersion string
}

func Parse(data []byte) (Manifest, error) {
	var m Manifest
	if len(data) > maxManifestBytes {
		return m, fmt.Errorf("asset manifest exceeds %d bytes", maxManifestBytes)
	}
	if err := strictJSON(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("asset manifest JSON: %w", err)
	}
	if m.Format != Format || m.Sequence == 0 || len(m.Assets) == 0 {
		return Manifest{}, fmt.Errorf("invalid asset manifest format, sequence or empty assets")
	}
	if m.IssuedAt.IsZero() || m.ExpiresAt.IsZero() || !m.ExpiresAt.After(m.IssuedAt) {
		return Manifest{}, fmt.Errorf("invalid asset manifest validity dates")
	}
	seen := make(map[[5]string]bool, len(m.Assets))
	for i, e := range m.Assets {
		if err := validateEntry(e); err != nil {
			return Manifest{}, fmt.Errorf("asset manifest entry %d: %w", i, err)
		}
		id := [5]string{e.Product, e.Purpose, e.Version, e.Platform, e.Asset}
		if seen[id] {
			return Manifest{}, fmt.Errorf("duplicate asset manifest identity at entry %d", i)
		}
		seen[id] = true
	}
	return m, nil
}

func Encode(m Manifest) ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if _, err := Parse(data); err != nil {
		return nil, err
	}
	return data, nil
}

func validateEntry(e Entry) error {
	if !safeToken(e.Product) || !safeToken(e.Purpose) || !safeToken(e.Platform) {
		return fmt.Errorf("invalid product, purpose or platform token")
	}
	if e.Asset == "" || e.Asset == "." || strings.TrimSpace(e.Asset) != e.Asset || strings.ContainsAny(e.Asset, "/\\") || strings.Contains(e.Asset, "..") || strings.ContainsFunc(e.Asset, unicode.IsControl) {
		return fmt.Errorf("invalid asset basename")
	}
	if _, err := absoluteURL(e.URL, false); err != nil {
		return err
	}
	if !validDigest(e.SHA256) || e.Size <= 0 || e.Size == math.MaxInt64 {
		return fmt.Errorf("invalid asset SHA-256 or size")
	}
	version, err := normalizedVersion(e.Version)
	if err != nil {
		return err
	}
	minimum, err := normalizedVersion(e.MinVersion)
	if err != nil {
		return err
	}
	if semver.Compare(version, minimum) < 0 {
		return fmt.Errorf("asset version is below min_version")
	}
	return nil
}

func safeToken(s string) bool {
	if s == "" || strings.Contains(s, "..") {
		return false
	}
	for i, c := range s {
		alnum := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
		if !alnum && (i == 0 || c != '-' && c != '_' && c != '.') {
			return false
		}
	}
	return true
}

func normalizedVersion(s string) (string, error) {
	v := "v" + strings.TrimPrefix(s, "v")
	core := strings.SplitN(strings.SplitN(v, "+", 2)[0], "-", 2)[0]
	if !semver.IsValid(v) || len(strings.Split(core, ".")) != 3 {
		return "", fmt.Errorf("invalid semantic version %q", s)
	}
	return v, nil
}

func validDigest(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == sha256.Size
}

func absoluteURL(s string, source bool) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid asset URL: %w", err)
	}
	if u.Opaque != "" || u.Hostname() == "" || u.User != nil || strings.Contains(s, "#") || (u.Scheme != "https" && (source || u.Scheme != "http")) {
		return nil, fmt.Errorf("asset URL must be absolute %s without userinfo or fragment", map[bool]string{true: "HTTPS", false: "HTTP(S)"}[source])
	}
	if source && (u.RawQuery != "" || u.ForceQuery || strings.HasSuffix(u.Path, "/") || path.Base(u.Path) != FileName) {
		return nil, fmt.Errorf("manifest source must have no query and basename %s", FileName)
	}
	return u, nil
}

func strictJSON(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := checkValue(d, reflect.TypeOf(out).Elem(), 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON data: %v", err)
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(out)
}

func checkValue(d *json.Decoder, typ reflect.Type, depth int) error {
	if depth > 32 {
		return fmt.Errorf("JSON nesting too deep")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		if depth == 0 {
			return fmt.Errorf("expected JSON object")
		}
		return nil
	}
	switch delim {
	case '{':
		if typ.Kind() != reflect.Struct || typ == reflect.TypeOf(time.Time{}) {
			return fmt.Errorf("unexpected JSON object")
		}
		fields := make(map[string]reflect.Type, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fields[f.Tag.Get("json")] = f.Type
		}
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("invalid JSON object key")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON object key %q", name)
			}
			seen[name] = true
			field, ok := fields[name]
			if !ok {
				return fmt.Errorf("unknown JSON field %q", name)
			}
			if err := checkValue(d, field, depth+1); err != nil {
				return err
			}
		}
	case '[':
		if typ.Kind() != reflect.Slice {
			return fmt.Errorf("unexpected JSON array")
		}
		for d.More() {
			if err := checkValue(d, typ.Elem(), depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
	_, err = d.Token()
	return err
}

type Client struct {
	SourceURL  string
	HTTPClient *http.Client
	Keys       []ed25519.PublicKey
	Now        func() time.Time
}

func NewClient() *Client {
	return &Client{SourceURL: os.Getenv(SourceEnv), HTTPClient: &http.Client{Timeout: 30 * time.Second}, Keys: publishsig.TrustedPublicKeys(), Now: time.Now}
}

type VerifiedManifest struct {
	manifest Manifest
	digest   string
	now      func() time.Time
}

func (c *Client) Fetch() (*VerifiedManifest, error) {
	if c == nil || c.SourceURL == "" {
		return nil, fmt.Errorf("set %s to the explicit HTTPS URL of %s", SourceEnv, FileName)
	}
	u, err := absoluteURL(c.SourceURL, true)
	if err != nil {
		return nil, err
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	// Copy the caller's client so redirect policy and timeout cannot weaken the source boundary.
	client := *hc
	if client.Timeout <= 0 || client.Timeout > 30*time.Second {
		client.Timeout = 30 * time.Second
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if _, err := absoluteURL(req.URL.String(), false); err != nil || req.URL.Scheme != "https" {
			return fmt.Errorf("unsafe manifest redirect")
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many manifest redirects")
		}
		if hc.CheckRedirect != nil {
			return hc.CheckRedirect(req, via)
		}
		return nil
	}
	data, err := fetchBytes(&client, u.String(), maxManifestBytes)
	if err != nil {
		return nil, err
	}
	u.Path += ".minisig"
	if u.RawPath != "" {
		u.RawPath += ".minisig"
	}
	signature, err := fetchBytes(&client, u.String(), maxSignatureBytes)
	if err != nil {
		return nil, err
	}
	if err := publishsig.Verify(c.Keys, FileName, bytes.NewReader(data), signature); err != nil {
		return nil, fmt.Errorf("asset manifest signature: %w", err)
	}
	m, err := Parse(data)
	if err != nil {
		return nil, err
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	digest := sha256.Sum256(data)
	verified := &VerifiedManifest{manifest: m, digest: hex.EncodeToString(digest[:]), now: now}
	if err := verified.validNow(); err != nil {
		return nil, err
	}
	return verified, nil
}

func fetchBytes(c *http.Client, source string, limit int64) ([]byte, error) {
	response, err := c.Get(source)
	if err != nil {
		return nil, fmt.Errorf("fetch asset manifest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch asset manifest: HTTP %d", response.StatusCode)
	}
	return boundedRead(response.Body, limit)
}

func boundedRead(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("document exceeds %d bytes", limit)
	}
	return data, nil
}

func (m *VerifiedManifest) validNow() error {
	if m == nil || m.now == nil || !validDigest(m.digest) {
		return fmt.Errorf("manifest was not verified")
	}
	now := m.now()
	if now.Before(m.manifest.IssuedAt) || !now.Before(m.manifest.ExpiresAt) {
		return fmt.Errorf("asset manifest is not yet valid or expired")
	}
	return nil
}

type authorizationState struct {
	Product                  string `json:"product"`
	Purpose                  string `json:"purpose"`
	Platform                 string `json:"platform"`
	HighestSequence          uint64 `json:"highest_sequence"`
	ManifestDigest           string `json:"manifest_digest"`
	HighestAuthorizedVersion string `json:"highest_authorized_version"`
}

func (m *VerifiedManifest) Authorize(r Request, statePath string) (Entry, error) {
	if strings.TrimSpace(statePath) == "" {
		return Entry{}, fmt.Errorf("authorization state path is required")
	}
	if err := m.validNow(); err != nil {
		return Entry{}, err
	}
	var candidate *Entry
	for i := range m.manifest.Assets {
		e := &m.manifest.Assets[i]
		if e.Product == r.Product && e.Purpose == r.Purpose && e.Version == r.Version && e.Platform == r.Platform && e.Asset == r.Asset && e.URL == r.URL {
			candidate = e
			break
		}
	}
	if candidate == nil {
		return Entry{}, fmt.Errorf("requested asset identity or URL is not authorized")
	}
	version, _ := normalizedVersion(candidate.Version)
	if r.CurrentVersion != "" {
		current, err := normalizedVersion(r.CurrentVersion)
		if err != nil {
			return Entry{}, err
		}
		if semver.Compare(version, current) < 0 {
			return Entry{}, fmt.Errorf("asset would downgrade the current version")
		}
	}
	state, err := readState(statePath)
	if err != nil {
		return Entry{}, fmt.Errorf("read authorization state: %w", err)
	}
	if state.HighestSequence != 0 {
		if state.Product != r.Product || state.Purpose != r.Purpose || state.Platform != r.Platform {
			return Entry{}, fmt.Errorf("authorization state identity mismatch")
		}
		if m.manifest.Sequence < state.HighestSequence || m.manifest.Sequence == state.HighestSequence && m.digest != state.ManifestDigest {
			return Entry{}, fmt.Errorf("manifest sequence rollback or equivocation")
		}
		highest, _ := normalizedVersion(state.HighestAuthorizedVersion)
		if semver.Compare(version, highest) < 0 {
			return Entry{}, fmt.Errorf("asset would downgrade the highest authorized version")
		}
	}
	next := authorizationState{Product: r.Product, Purpose: r.Purpose, Platform: r.Platform, HighestSequence: m.manifest.Sequence, ManifestDigest: m.digest, HighestAuthorizedVersion: candidate.Version}
	if err := writeState(statePath, next); err != nil {
		return Entry{}, fmt.Errorf("persist authorization state: %w", err)
	}
	return *candidate, nil
}

func readState(statePath string) (authorizationState, error) {
	var state authorizationState
	info, err := os.Lstat(statePath)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if !info.Mode().IsRegular() {
		return state, fmt.Errorf("authorization state is not a regular file")
	}
	f, err := os.Open(statePath)
	if err != nil {
		return state, err
	}
	defer f.Close()
	data, err := boundedRead(f, maxStateBytes)
	if err != nil {
		return state, err
	}
	if err := strictJSON(data, &state); err != nil {
		return state, err
	}
	if !safeToken(state.Product) || !safeToken(state.Purpose) || !safeToken(state.Platform) || state.HighestSequence == 0 || !validDigest(state.ManifestDigest) {
		return state, fmt.Errorf("invalid authorization state")
	}
	if _, err := normalizedVersion(state.HighestAuthorizedVersion); err != nil {
		return state, err
	}
	return state, nil
}

func writeState(statePath string, state authorizationState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(statePath), ".assetmanifest-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), statePath)
}

func VerifyContent(entry Entry, r io.Reader) error {
	if entry.Size <= 0 || entry.Size == math.MaxInt64 || !validDigest(entry.SHA256) {
		return fmt.Errorf("invalid asset size or SHA-256")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(r, entry.Size+1))
	if err != nil {
		return fmt.Errorf("read asset content: %w", err)
	}
	if n != entry.Size {
		return fmt.Errorf("asset size mismatch: got %d, want %d", n, entry.Size)
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), entry.SHA256) {
		return fmt.Errorf("asset SHA-256 mismatch")
	}
	return nil
}
