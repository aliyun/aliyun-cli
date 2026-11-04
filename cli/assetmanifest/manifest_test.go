package assetmanifest

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

var testNow = time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)

func fixture() Manifest {
	sum := sha256.Sum256([]byte("asset"))
	return Manifest{Format: Format, Sequence: 10, IssuedAt: testNow.Add(-time.Hour), ExpiresAt: testNow.Add(time.Hour), Assets: []Entry{{Product: "bl", Purpose: Purpose, Version: "1.2.3", Platform: "darwin-arm64", Asset: "bl-1.2.3-darwin-arm64.tgz", URL: "http://fixture.invalid/bl.tgz", SHA256: hex.EncodeToString(sum[:]), Size: 5, MinVersion: "v1.0.0"}}}
}

func request(e Entry) Request {
	return Request{Product: e.Product, Purpose: e.Purpose, Version: e.Version, Platform: e.Platform, Asset: e.Asset, URL: e.URL}
}

func raw(t *testing.T, m Manifest) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSchemaRoundTrip(t *testing.T) {
	if FileName != "cli-assets-v1.json" || Format != "aliyun-cli-assets v1" || Purpose != "external-cli" || SourceEnv != "ALIBABA_CLOUD_CLI_ASSET_MANIFEST_URL" {
		t.Fatal("API constants changed")
	}
	m := fixture()
	b, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(b)
	if err != nil || !reflect.DeepEqual(got, m) {
		t.Fatalf("roundtrip: %v, %+v", err, got)
	}
	m.Assets[0].Product = "dgate"
	m.Assets[0].Version = "v2.0.0-rc.1+build.2"
	m.Assets[0].MinVersion = "2.0.0-beta.1"
	if _, err := Encode(m); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaRejectsInvalidEntries(t *testing.T) {
	cases := map[string]func(*Entry){
		"empty product": func(e *Entry) { e.Product = "" }, "space purpose": func(e *Entry) { e.Purpose = " external-cli" }, "unsafe platform": func(e *Entry) { e.Platform = "../darwin" },
		"token control": func(e *Entry) { e.Product = "b\nl" }, "token unicode": func(e *Entry) { e.Product = "中文" }, "token symbol": func(e *Entry) { e.Product = "b:l" },
		"asset empty": func(e *Entry) { e.Asset = "" }, "asset dot": func(e *Entry) { e.Asset = "." }, "asset slash": func(e *Entry) { e.Asset = "dir/a" }, "asset backslash": func(e *Entry) { e.Asset = `dir\a` }, "asset dots": func(e *Entry) { e.Asset = "a..tgz" }, "asset control": func(e *Entry) { e.Asset = "a\x00" }, "asset whitespace": func(e *Entry) { e.Asset = " a" },
		"url relative": func(e *Entry) { e.URL = "/a" }, "url scheme": func(e *Entry) { e.URL = "ftp://example.invalid/a" }, "url userinfo": func(e *Entry) { e.URL = "https://user:password@example.invalid/a" }, "url fragment": func(e *Entry) { e.URL = "https://example.invalid/a#x" }, "url host": func(e *Entry) { e.URL = "https:///a" }, "url malformed": func(e *Entry) { e.URL = "https://[" }, "url opaque": func(e *Entry) { e.URL = "https:example.invalid" },
		"sha length": func(e *Entry) { e.SHA256 = "ab" }, "sha hex": func(e *Entry) { e.SHA256 = strings.Repeat("z", 64) },
		"size zero": func(e *Entry) { e.Size = 0 }, "size negative": func(e *Entry) { e.Size = -1 }, "size overflow": func(e *Entry) { e.Size = math.MaxInt64 },
		"version invalid": func(e *Entry) { e.Version = "latest" }, "version short": func(e *Entry) { e.Version = "1.2" }, "version leadingzero": func(e *Entry) { e.Version = "01.2.3" }, "minimum empty": func(e *Entry) { e.MinVersion = "" }, "minimum invalid": func(e *Entry) { e.MinVersion = "bad" }, "below minimum": func(e *Entry) { e.MinVersion = "1.2.4" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			m := fixture()
			mutate(&m.Assets[0])
			if _, err := Parse(raw(t, m)); err == nil {
				t.Fatal("invalid entry accepted")
			}
			if _, err := Encode(m); err == nil {
				t.Fatal("invalid entry encoded")
			}
		})
	}
	for name, mutate := range map[string]func(*Manifest){"format": func(m *Manifest) { m.Format = "wrong" }, "sequence": func(m *Manifest) { m.Sequence = 0 }, "no entries": func(m *Manifest) { m.Assets = nil }, "no issued": func(m *Manifest) { m.IssuedAt = time.Time{} }, "no expires": func(m *Manifest) { m.ExpiresAt = time.Time{} }, "equal dates": func(m *Manifest) { m.ExpiresAt = m.IssuedAt }, "past expiry": func(m *Manifest) { m.ExpiresAt = m.IssuedAt.Add(-time.Second) }, "duplicate identity": func(m *Manifest) { m.Assets = append(m.Assets, m.Assets[0]); m.Assets[1].URL += "?other" }} {
		t.Run(name, func(t *testing.T) {
			m := fixture()
			mutate(&m)
			if _, err := Parse(raw(t, m)); err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}

func TestStrictJSON(t *testing.T) {
	b := string(raw(t, fixture()))
	for name, text := range map[string]string{
		"empty": "", "syntax": "{", "array": "[]", "null": "null", "trailing": b + " {}", "garbage": b + "!", "unknown": strings.Replace(b, `"sequence":10`, `"sequence":10,"surprise":true`, 1),
		"nested unknown": strings.Replace(b, `"size":5`, `"size":5,"surprise":true`, 1), "root duplicate": strings.Replace(b, `"sequence":10`, `"sequence":10,"sequence":11`, 1),
		"escaped duplicate": strings.Replace(b, `"size":5`, `"size":5,"s\u0069ze":6`, 1), "nested duplicate": strings.Replace(b, `"product":"bl"`, `"product":"bl","product":"dgate"`, 1),
		"duplicate unknown object": strings.Replace(b, `"size":5`, `"size":5,"x":{"a":1,"a":2}`, 1),
		"missing date":             strings.Replace(b, `"issued_at":"2026-11-03T23:00:00Z",`, "", 1), "date null": strings.Replace(b, `"expires_at":"2026-11-04T01:00:00Z"`, `"expires_at":null`, 1), "bad date": strings.Replace(b, `2026-11-04T01:00:00Z`, `not-a-date`, 1), "wrong type": strings.Replace(b, `"sequence":10`, `"sequence":"10"`, 1),
		"too large": strings.Repeat(" ", 2*1024*1024+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(text)); err == nil {
				t.Fatal("bad JSON accepted")
			}
		})
	}
	padded := append(raw(t, fixture()), bytes.Repeat([]byte(" "), 2*1024*1024-len(b))...)
	if _, err := Parse(padded); err != nil {
		t.Fatal("exact size boundary:", err)
	}
}

func keys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := publishsig.Generate()
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func serve(t *testing.T, m Manifest) (*Client, *[]byte, *[]byte, *int, *int) {
	t.Helper()
	pub, priv := keys(t)
	data := raw(t, m)
	sig, err := publishsig.Sign(priv, FileName, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	bodyStatus, sigStatus := 200, 200
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Error("not GET")
		}
		switch r.URL.Path {
		case "/releases/" + FileName:
			w.WriteHeader(bodyStatus)
			_, _ = w.Write(data)
		case "/releases/" + FileName + ".minisig":
			w.WriteHeader(sigStatus)
			_, _ = w.Write(sig)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return &Client{SourceURL: srv.URL + "/releases/" + FileName, HTTPClient: srv.Client(), Keys: []ed25519.PublicKey{pub}, Now: func() time.Time { return testNow }}, &data, &sig, &bodyStatus, &sigStatus
}

func verified(t *testing.T, m Manifest) *VerifiedManifest {
	t.Helper()
	c, _, _, _, _ := serve(t, m)
	v, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestClientDefaultsAndSource(t *testing.T) {
	t.Setenv(SourceEnv, "")
	c := NewClient()
	if c.SourceURL != "" || c.HTTPClient == nil || c.HTTPClient.Timeout <= 0 || c.Now == nil || !reflect.DeepEqual(c.Keys, publishsig.TrustedPublicKeys()) {
		t.Fatal("unsafe defaults")
	}
	if _, err := c.Fetch(); err == nil || !strings.Contains(err.Error(), SourceEnv) {
		t.Fatalf("missing source error: %v", err)
	}
	t.Setenv(SourceEnv, "https://fixture.invalid/"+FileName)
	if NewClient().SourceURL != os.Getenv(SourceEnv) {
		t.Fatal("source env ignored")
	}
	for _, s := range []string{"http://fixture.invalid/" + FileName, "/" + FileName, "https://user@fixture.invalid/" + FileName, "https://fixture.invalid/" + FileName + "#x", "https://fixture.invalid/" + FileName + "?x=1", "https://fixture.invalid/" + FileName + "?", "https://fixture.invalid/other.json", "https:///" + FileName, "https://[", "https:opaque"} {
		t.Run(s, func(t *testing.T) {
			c.SourceURL = s
			if _, err := c.Fetch(); err == nil {
				t.Fatal("invalid source accepted")
			}
		})
	}
}

func TestFetchRejectsUntrustedInvalidAndExpired(t *testing.T) {
	for _, name := range []string{"missing signature", "oversized signature", "tamper", "rename", "wrong key", "no keys", "malformed signed JSON", "oversized manifest", "body status", "signature status", "future", "expired", "exact expiry"} {
		t.Run(name, func(t *testing.T) {
			m := fixture()
			if name == "future" {
				m.IssuedAt = testNow.Add(time.Second)
			}
			if name == "expired" || name == "exact expiry" {
				m.ExpiresAt = testNow
				if name == "expired" {
					m.ExpiresAt = testNow.Add(-time.Second)
				}
			}
			c, b, s, bs, ss := serve(t, m)
			switch name {
			case "missing signature":
				*s = nil
			case "oversized signature":
				*s = bytes.Repeat([]byte("x"), 4097)
			case "tamper":
				*b = append(*b, ' ')
			case "rename":
				*s = bytes.Replace(*s, []byte(FileName), []byte("renamed.json"), 1)
			case "wrong key":
				pub, _ := keys(t)
				c.Keys = []ed25519.PublicKey{pub}
			case "no keys":
				c.Keys = nil
			case "malformed signed JSON":
				_, priv := keys(t)
				c.Keys = []ed25519.PublicKey{priv.Public().(ed25519.PublicKey)}
				*b = []byte(`{"format":"bad"}`)
				var err error
				*s, err = publishsig.Sign(priv, FileName, bytes.NewReader(*b))
				if err != nil {
					t.Fatal(err)
				}
			case "oversized manifest":
				*b = bytes.Repeat([]byte(" "), 2*1024*1024+1)
			case "body status":
				*bs = 404
			case "signature status":
				*ss = 503
			}
			if _, err := c.Fetch(); err == nil {
				t.Fatal("unsafe fetch accepted")
			}
		})
	}
	c, _, _, _, _ := serve(t, fixture())
	c.Now = func() time.Time { return fixture().IssuedAt }
	if _, err := c.Fetch(); err != nil {
		t.Fatal("issued boundary:", err)
	}
	ancient := fixture()
	ancient.IssuedAt = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	ancient.ExpiresAt = ancient.IssuedAt.Add(time.Hour)
	c, _, _, _, _ = serve(t, ancient)
	c.Now = nil
	if _, err := c.Fetch(); err == nil {
		t.Fatal("default clock accepted expired manifest")
	}
}

func TestAuthorizeIdentityAndVersion(t *testing.T) {
	v := verified(t, fixture())
	e := fixture().Assets[0]
	r := request(e)
	for name, mutate := range map[string]func(*Request){"product": func(r *Request) { r.Product = "dgate" }, "purpose": func(r *Request) { r.Purpose = "host" }, "version": func(r *Request) { r.Version = "v1.2.3" }, "platform": func(r *Request) { r.Platform = "linux-amd64" }, "asset": func(r *Request) { r.Asset = "other.tgz" }, "URL": func(r *Request) { r.URL += "?x" }, "current newer": func(r *Request) { r.CurrentVersion = "1.2.4" }, "current invalid": func(r *Request) { r.CurrentVersion = "bad" }} {
		t.Run(name, func(t *testing.T) {
			r := r
			mutate(&r)
			p := filepath.Join(t.TempDir(), "state.json")
			if _, err := v.Authorize(r, p); err == nil {
				t.Fatal("unsafe authorization")
			}
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Fatal("rejected authorization persisted")
			}
		})
	}
	if _, err := v.Authorize(r, ""); err == nil {
		t.Fatal("empty state path bypass")
	}
	for _, current := range []string{"", "v1.2.3", "1.0.0"} {
		r.CurrentVersion = current
		p := filepath.Join(t.TempDir(), "state.json")
		got, err := v.Authorize(r, p)
		if err != nil || got != e {
			t.Fatalf("authorize: %+v %v", got, err)
		}
		info, err := os.Stat(p)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("state not private")
		}
		if _, err := v.Authorize(r, p); err != nil {
			t.Fatal("identical retry:", err)
		}
		names, err := os.ReadDir(filepath.Dir(p))
		if err != nil || len(names) != 1 {
			t.Fatal("temporary state leak")
		}
	}
}

func TestStateRollbackEquivocationAndDowngrade(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	m := fixture()
	v := verified(t, m)
	if _, err := v.Authorize(request(m.Assets[0]), p); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"lower sequence", "equal equivocation", "entry downgrade", "binding product", "binding purpose", "binding platform"} {
		t.Run(name, func(t *testing.T) {
			n := fixture()
			switch name {
			case "lower sequence":
				n.Sequence--
			case "equal equivocation":
				n.ExpiresAt = n.ExpiresAt.Add(time.Second)
			case "entry downgrade":
				n.Sequence++
				n.Assets[0].Version = "1.1.0"
			case "binding product":
				n.Sequence++
				n.Assets[0].Product = "dgate"
			case "binding purpose":
				n.Sequence++
				n.Assets[0].Purpose = "other"
			case "binding platform":
				n.Sequence++
				n.Assets[0].Platform = "linux-amd64"
			}
			v := verified(t, n)
			if _, err := v.Authorize(request(n.Assets[0]), p); err == nil {
				t.Fatal("rollback or cross-binding accepted")
			}
			after, err := os.ReadFile(p)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("old state overwritten")
			}
		})
	}
	m.Sequence++
	m.Assets[0].Version = "1.3.0"
	v = verified(t, m)
	if _, err := v.Authorize(request(m.Assets[0]), p); err != nil {
		t.Fatal("upgrade:", err)
	}
	m.Sequence++
	m.Assets[0].Version = "v1.3.0"
	v = verified(t, m)
	if _, err := v.Authorize(request(m.Assets[0]), p); err != nil {
		t.Fatal("equivalent semver:", err)
	}
}

func TestStateCorruptionAndPersistenceFailure(t *testing.T) {
	v := verified(t, fixture())
	r := request(fixture().Assets[0])
	p := filepath.Join(t.TempDir(), "state.json")
	if _, err := v.Authorize(r, p); err != nil {
		t.Fatal(err)
	}
	valid, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(valid, &obj); err != nil {
		t.Fatal(err)
	}
	malformed := []string{"", "null", "[]", "{", string(valid) + " {}", strings.Repeat(" ", 65537), strings.Replace(string(valid), `"product":"bl"`, `"product":"bl","product":"bl"`, 1)}
	for key, val := range map[string]any{"highest_sequence": float64(0), "manifest_digest": "bad", "highest_authorized_version": "bad", "product": "", "purpose": "", "platform": "", "unknown": true} {
		copyObj := map[string]any{}
		for k, v := range obj {
			copyObj[k] = v
		}
		copyObj[key] = val
		b, err := json.Marshal(copyObj)
		if err != nil {
			t.Fatal(err)
		}
		malformed = append(malformed, string(b))
	}
	for i, text := range malformed {
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := v.Authorize(r, p); err == nil {
			t.Fatalf("corrupt state %d accepted: %s", i, text)
		}
		after, err := os.ReadFile(p)
		if err != nil || string(after) != text {
			t.Fatal("corrupt state overwritten")
		}
	}
	dir := t.TempDir()
	if _, err := v.Authorize(r, dir); err == nil {
		t.Fatal("directory state accepted")
	}
	loop := filepath.Join(t.TempDir(), "loop")
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Authorize(r, loop); err == nil {
		t.Fatal("unreadable state accepted")
	}
	missing := filepath.Join(t.TempDir(), "missing", "state.json")
	if _, err := v.Authorize(r, missing); err == nil {
		t.Fatal("persistence failure ignored")
	}
}

func TestAuthorizeRechecksTime(t *testing.T) {
	now := testNow
	c, _, _, _, _ := serve(t, fixture())
	c.Now = func() time.Time { return now }
	v, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	for _, clock := range []time.Time{fixture().IssuedAt.Add(-time.Second), fixture().ExpiresAt} {
		now = clock
		p := filepath.Join(t.TempDir(), "state.json")
		if _, err := v.Authorize(request(fixture().Assets[0]), p); err == nil {
			t.Fatal("stale authorization accepted")
		}
	}
	var zero VerifiedManifest
	if _, err := zero.Authorize(request(fixture().Assets[0]), filepath.Join(t.TempDir(), "state.json")); err == nil {
		t.Fatal("zero verified value accepted")
	}
}

func TestSourceRejectsTrailingSlash(t *testing.T) {
	c, _, _, _, _ := serve(t, fixture())
	c.SourceURL += "/"
	if _, err := c.Fetch(); err == nil || !strings.Contains(err.Error(), "basename") {
		t.Fatalf("trailing slash source was not rejected before GET: %v", err)
	}
}

func TestSignatureURLPreservesEscapedPath(t *testing.T) {
	pub, priv := keys(t)
	data := raw(t, fixture())
	signature, err := publishsig.Sign(priv, FileName, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := "/releases%2Fstable/" + FileName
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case sourcePath:
			_, _ = w.Write(data)
		case sourcePath + ".minisig":
			_, _ = w.Write(signature)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := &Client{SourceURL: srv.URL + sourcePath, HTTPClient: srv.Client(), Keys: []ed25519.PublicKey{pub}, Now: func() time.Time { return testNow }}
	if _, err := c.Fetch(); err != nil {
		t.Fatal("escaped signature source:", err)
	}
}

func TestAdditionalSchemaBoundaries(t *testing.T) {
	m := fixture()
	m.IssuedAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := Encode(m); err == nil {
		t.Fatal("marshal failure ignored")
	}
	b := string(raw(t, fixture()))
	for _, text := range []string{
		strings.Replace(b, `"format"`, `"Format"`, 1),
		strings.Replace(b, `"product":"bl"`, `"product":{}`, 1),
		strings.Replace(b, `"size":5`, `"size":[]`, 1),
		strings.Replace(b, `"assets":[{`, `"assets":[{`, 1)[:len(b)-2],
		strings.Replace(b, `"size":5`, `"size":5.5`, 1),
	} {
		if _, err := Parse([]byte(text)); err == nil {
			t.Fatalf("invalid shape accepted: %s", text)
		}
	}
	m = fixture()
	m.Sequence = math.MaxUint64
	m.Assets[0].Size = math.MaxInt64 - 1
	m.Assets[0].SHA256 = strings.ToUpper(m.Assets[0].SHA256)
	m.Assets[0].URL = "https://fixture.invalid/a?signature=test"
	if _, err := Encode(m); err != nil {
		t.Fatal("valid upper boundaries:", err)
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchTransportFailuresAndBounds(t *testing.T) {
	for _, name := range []string{"transport", "manifest read", "signature read", "signature exact bound", "verify before parse"} {
		t.Run(name, func(t *testing.T) {
			c, body, signature, _, _ := serve(t, fixture())
			calls := 0
			c.HTTPClient = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if name == "transport" {
					return nil, errors.New("transport failure")
				}
				var reader io.Reader = bytes.NewReader(*body)
				if strings.HasSuffix(r.URL.Path, ".minisig") {
					reader = bytes.NewReader(*signature)
				}
				switch name {
				case "manifest read":
					reader = failingReader{}
				case "signature read":
					if calls == 2 {
						reader = failingReader{}
					}
				case "signature exact bound":
					if calls == 2 {
						reader = strings.NewReader(strings.Repeat("x", 4096))
					}
				case "verify before parse":
					if calls == 1 {
						reader = strings.NewReader("{")
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(reader), Header: make(http.Header), Request: r}, nil
			})}
			_, err := c.Fetch()
			if err == nil {
				t.Fatal("transport failure accepted")
			}
			if name == "verify before parse" && !strings.Contains(err.Error(), "signature") {
				t.Fatalf("parsed before signature: %v", err)
			}
			if name == "signature exact bound" && strings.Contains(err.Error(), "exceeds") {
				t.Fatalf("exact bound wrongly rejected by fetch: %v", err)
			}
			if c.HTTPClient.Timeout != 0 || c.HTTPClient.CheckRedirect != nil {
				t.Fatal("caller client mutated")
			}
		})
	}
	var nilClient *Client
	if _, err := nilClient.Fetch(); err == nil {
		t.Fatal("nil client accepted")
	}
	var nilManifest *VerifiedManifest
	if _, err := nilManifest.Authorize(Request{}, filepath.Join(t.TempDir(), "state.json")); err == nil {
		t.Fatal("nil manifest accepted")
	}
}

func TestHTTPSRedirectPolicy(t *testing.T) {
	pub, priv := keys(t)
	data := raw(t, fixture())
	signature, err := publishsig.Sign(priv, FileName, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"safe", "HTTP", "userinfo", "loop", "caller policy"} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, ".minisig") {
					_, _ = w.Write(signature)
					return
				}
				if strings.HasPrefix(r.URL.Path, "/final/") {
					_, _ = w.Write(data)
					return
				}
				target := "/final/" + FileName
				switch name {
				case "HTTP":
					target = "http://fixture.invalid/" + FileName
				case "userinfo":
					target = "https://user@fixture.invalid/" + FileName
				case "loop":
					target = r.URL.Path
				}
				http.Redirect(w, r, target, http.StatusFound)
			}))
			defer srv.Close()
			c := &Client{SourceURL: srv.URL + "/start/" + FileName, HTTPClient: srv.Client(), Keys: []ed25519.PublicKey{pub}, Now: func() time.Time { return testNow }}
			if name == "caller policy" {
				c.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("caller disallows redirect") }
			}
			_, err := c.Fetch()
			if (name == "safe") != (err == nil) {
				t.Fatalf("redirect result: %v", err)
			}
		})
	}
}

func TestStateIOFailuresRetainOldState(t *testing.T) {
	v := verified(t, fixture())
	r := request(fixture().Assets[0])
	dir := t.TempDir()
	p := filepath.Join(dir, "state.json")
	if _, err := v.Authorize(r, p); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() != 0 {
		if err := os.Chmod(p, 0000); err != nil {
			t.Fatal(err)
		}
		_, err := v.Authorize(r, p)
		if restore := os.Chmod(p, 0600); restore != nil {
			t.Fatal(restore)
		}
		if err == nil {
			t.Fatal("unreadable state ignored")
		}
		if err := os.Chmod(dir, 0500); err != nil {
			t.Fatal(err)
		}
		_, err = v.Authorize(r, p)
		if restore := os.Chmod(dir, 0700); restore != nil {
			t.Fatal(restore)
		}
		if err == nil {
			t.Fatal("state persistence failure ignored")
		}
		if err := os.Chmod(dir, 0000); err != nil {
			t.Fatal(err)
		}
		_, err = v.Authorize(r, p)
		if restore := os.Chmod(dir, 0700); restore != nil {
			t.Fatal(restore)
		}
		if err == nil {
			t.Fatal("unreadable parent ignored")
		}
	}
	after, err := os.ReadFile(p)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("failed persistence changed old state")
	}
	if err := writeState(dir, authorizationState{}); err == nil {
		t.Fatal("rename over directory accepted")
	}
	names, err := os.ReadDir(dir)
	if err != nil || len(names) != 1 {
		t.Fatal("failed rename left temporary files")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failure") }

func TestVerifyContent(t *testing.T) {
	e := fixture().Assets[0]
	for name, reader := range map[string]io.Reader{"valid": strings.NewReader("asset"), "short": strings.NewReader("asse"), "long": strings.NewReader("asset!"), "wrong digest": strings.NewReader("other"), "read failure": failingReader{}} {
		t.Run(name, func(t *testing.T) {
			err := VerifyContent(e, reader)
			if (name == "valid") != (err == nil) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	for _, size := range []int64{0, -1, math.MaxInt64} {
		e.Size = size
		if err := VerifyContent(e, strings.NewReader("asset")); err == nil {
			t.Fatal("unsafe size")
		}
	}
	e = fixture().Assets[0]
	e.SHA256 = "bad"
	if err := VerifyContent(e, strings.NewReader("asset")); err == nil {
		t.Fatal("bad digest accepted")
	}
	e = fixture().Assets[0]
	e.SHA256 = strings.ToUpper(e.SHA256)
	if err := VerifyContent(e, strings.NewReader("asset")); err != nil {
		t.Fatal("uppercase hex:", err)
	}
}
