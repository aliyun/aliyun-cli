package main

import (
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

func TestRunUsage(t *testing.T) {
	if code := run([]string{"publishsign"}); code != 2 {
		t.Fatalf("missing command exit = %d", code)
	}
	if code := run([]string{"publishsign", "nope"}); code != 2 {
		t.Fatalf("unknown command exit = %d", code)
	}
	if code := run([]string{"publishsign", "keygen", "--nope"}); code != 1 {
		t.Fatalf("bad flag exit = %d", code)
	}
	if code := run([]string{"publishsign", "sign", "--nope"}); code != 1 {
		t.Fatalf("bad sign flag exit = %d", code)
	}
	if code := run([]string{"publishsign", "verify", "--nope"}); code != 1 {
		t.Fatalf("bad verify flag exit = %d", code)
	}
}

func TestKeygenSignAndVerify(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "release.key")
	if code := run([]string{"publishsign", "keygen"}); code != 1 {
		t.Fatalf("keygen without output exit = %d", code)
	}
	if code := run([]string{"publishsign", "keygen", "--secret-out", filepath.Join(dir, "missing", "key")}); code != 1 {
		t.Fatalf("keygen unwritable exit = %d", code)
	}
	if code := run([]string{"publishsign", "keygen", "--secret-out", secret}); code != 0 {
		t.Fatal("keygen failed")
	}
	if code := run([]string{"publishsign", "keygen", "--secret-out", secret}); code != 1 {
		t.Fatal("keygen overwrote an existing secret")
	}
	info, err := os.Stat(secret)
	if err != nil {
		t.Fatal(err)
	}
	// Windows does not store the Unix mode passed to WriteFile.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("secret mode = %o", info.Mode().Perm())
	}

	raw, err := os.ReadFile(secret)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := publishsig.ParseSecret(raw)
	if err != nil {
		t.Fatal(err)
	}
	pub := publishsig.FormatPublicKey(priv.Public().(ed25519.PublicKey))

	asset := filepath.Join(dir, "aliyun-cli-linux-1.2.3-amd64.tgz")
	if err := os.WriteFile(asset, []byte("archive-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"publishsign", "sign"}); code != 1 {
		t.Fatalf("sign without flags exit = %d", code)
	}
	if code := run([]string{"publishsign", "sign", "--secret-key-file", filepath.Join(dir, "missing.key"), "--asset", asset}); code != 1 {
		t.Fatal("sign accepted a missing secret")
	}
	if code := run([]string{"publishsign", "sign", "--secret-key-file", asset, "--asset", asset}); code != 1 {
		t.Fatal("sign accepted a secret that is not a key")
	}
	if code := run([]string{"publishsign", "sign", "--secret-key-file", secret, "--asset", filepath.Join(dir, "missing.tgz")}); code != 1 {
		t.Fatal("sign accepted a missing asset")
	}
	if code := run([]string{"publishsign", "sign", "--secret-key-file", secret, "--asset", asset}); code != 0 {
		t.Fatal("sign failed")
	}

	if code := run([]string{"publishsign", "verify"}); code != 1 {
		t.Fatalf("verify without flags exit = %d", code)
	}
	if code := run([]string{"publishsign", "verify", "--public-key", "@@@@", "--asset", asset}); code != 1 {
		t.Fatal("verify accepted a bad public key")
	}
	if code := run([]string{"publishsign", "verify", "--public-key", pub, "--asset", filepath.Join(dir, "missing.tgz")}); code != 1 {
		t.Fatal("verify accepted a missing signature")
	}
	if code := run([]string{"publishsign", "verify", "--asset", asset}); code != 1 {
		t.Fatal("embedded key accepted a signature from the test key")
	}
	if code := run([]string{"publishsign", "verify", "--public-key", pub, "--asset", asset}); code != 0 {
		t.Fatal("verify failed")
	}

	os.Remove(asset)
	if code := run([]string{"publishsign", "verify", "--public-key", pub, "--asset", asset}); code != 1 {
		t.Fatal("verify accepted a deleted asset")
	}
}

func TestMainAndKeygenFailures(t *testing.T) {
	origExit := osExit
	origArgs := os.Args
	defer func() {
		osExit = origExit
		os.Args = origArgs
	}()
	var code int
	osExit = func(c int) { code = c }
	os.Args = []string{"publishsign"}
	main()
	if code != 2 {
		t.Fatalf("exit = %d", code)
	}

	origGenerate := generateKeyPair
	generateKeyPair = func() (ed25519.PublicKey, ed25519.PrivateKey, error) {
		return nil, nil, errors.New("entropy failed")
	}
	defer func() { generateKeyPair = origGenerate }()
	if code := run([]string{"publishsign", "keygen", "--secret-out", filepath.Join(t.TempDir(), "key")}); code != 1 {
		t.Fatalf("keygen exit = %d", code)
	}
}
