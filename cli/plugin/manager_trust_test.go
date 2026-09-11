package plugin

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIndex_SignedHappyPath(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	trust.SetVerifyKeys([]trust.VerifyKey{{
		KeyID:     "plugins-test",
		PublicKey: pub,
		Roles:     []string{trust.RolePlugins},
	}})
	t.Cleanup(trust.ResetVerifyKeys)

	indexObj := Index{
		IndexVersion: 10,
		ExpiresAt:    "2099-01-01T00:00:00Z",
		Plugins:      []PluginInfo{{Name: "demo", Versions: map[string]VersionInfo{}}},
	}
	payload, err := json.Marshal(indexObj)
	require.NoError(t, err)
	sig, err := trust.Sign("plugins-test", priv, payload, time.Now())
	require.NoError(t, err)
	sigBytes, err := trust.MarshalSignature(sig)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/plugin_pkg_index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/plugin_pkg_index.json.sig", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(sigBytes)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	root := t.TempDir()
	mgr := &Manager{
		rootDir:  root,
		indexURL: srv.URL + "/plugin_pkg_index.json",
		trustDir: filepath.Join(root, "trust"),
	}
	got, err := mgr.GetIndex()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "demo", got.Plugins[0].Name)
	assert.EqualValues(t, 10, got.IndexVersion)
}

func TestGetIndex_BadSignatureFails(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	_, otherPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	trust.SetVerifyKeys([]trust.VerifyKey{{
		KeyID:     "plugins-test",
		PublicKey: pub,
		Roles:     []string{trust.RolePlugins},
	}})
	t.Cleanup(trust.ResetVerifyKeys)

	payload := []byte(`{"index_version":1,"expires_at":"2099-01-01T00:00:00Z","plugins":[]}`)
	sig, err := trust.Sign("plugins-test", otherPriv, payload, time.Now())
	require.NoError(t, err)
	sigBytes, _ := trust.MarshalSignature(sig)

	mux := http.NewServeMux()
	mux.HandleFunc("/plugin_pkg_index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/plugin_pkg_index.json.sig", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(sigBytes)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mgr := &Manager{
		rootDir:  t.TempDir(),
		indexURL: srv.URL + "/plugin_pkg_index.json",
		trustDir: t.TempDir(),
	}
	_, err = mgr.GetIndex()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func TestGetIndex_EnforceUnsignedFails(t *testing.T) {
	t.Setenv(trust.EnvTrustEnforce, "1")
	payload := []byte(`{"plugins":[]}`)
	mux := http.NewServeMux()
	mux.HandleFunc("/plugin_pkg_index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mgr := &Manager{
		rootDir:  t.TempDir(),
		indexURL: srv.URL + "/plugin_pkg_index.json",
		trustDir: t.TempDir(),
	}
	_, err := mgr.GetIndex()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing signature")
}
