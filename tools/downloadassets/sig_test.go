package downloadassets

import (
	"crypto/ed25519"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

var (
	testPub  ed25519.PublicKey
	testPriv ed25519.PrivateKey
)

func TestMain(m *testing.M) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	testPub, testPriv = pub, priv
	verifyReleaseAsset = func(assetName, assetPath string, sig []byte) error {
		return publishsig.VerifyFile([]ed25519.PublicKey{testPub}, assetName, assetPath, sig)
	}
	os.Exit(m.Run())
}

func signedHandler(bodyOf func(name string) []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if strings.HasSuffix(name, ".minisig") {
			asset := strings.TrimSuffix(name, ".minisig")
			doc, err := publishsig.Sign(testPriv, asset, strings.NewReader(string(bodyOf(asset))))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(doc)
			return
		}
		_, _ = w.Write(bodyOf(name))
	}
}
