package publishsig

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSignVerifyRoundTripAndRejections(t *testing.T) {
	pub, priv, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	keys := []ed25519.PublicKey{pub}
	body := []byte("release-bytes")
	name := "aliyun-cli-linux-3.5.2-amd64.tgz"
	doc, err := Sign(priv, name, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(keys, name, bytes.NewReader(body), doc); err != nil {
		t.Fatal(err)
	}

	tampered := append([]byte(nil), body...)
	tampered[0] ^= 0xff
	if err := Verify(keys, name, bytes.NewReader(tampered), doc); err == nil {
		t.Fatal("tampered asset was accepted")
	}

	other := "aliyun-cli-linux-3.5.2-arm64.tgz"
	if err := Verify(keys, other, bytes.NewReader(body), doc); err == nil {
		t.Fatal("signature for another asset name was accepted")
	}

	_, otherPriv, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	forged, err := Sign(otherPriv, name, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(keys, name, bytes.NewReader(body), forged); err == nil || !strings.Contains(err.Error(), "not trusted") {
		t.Fatalf("untrusted key error = %v", err)
	}

	swapped := bytes.Replace(doc, []byte("asset: "+name), []byte("asset: "+other), 1)
	if err := Verify(keys, other, bytes.NewReader(body), swapped); err == nil {
		t.Fatal("rewritten asset line was accepted")
	}

	if err := Verify(nil, name, bytes.NewReader(body), doc); err == nil {
		t.Fatal("empty trust store was accepted")
	}
	if err := Verify(keys, name, bytes.NewReader(body), nil); err == nil {
		t.Fatal("missing signature was accepted")
	}
	if err := Verify(keys, name, bytes.NewReader(body), bytes.Repeat([]byte("a"), maxSigBytes+1)); err == nil {
		t.Fatal("oversized signature was accepted")
	}
	if _, err := Sign(priv, "../aliyun.tgz", bytes.NewReader(body)); err == nil {
		t.Fatal("path asset name was signed")
	}
	if _, err := Sign(priv, name, bytes.NewReader(nil)); err == nil {
		t.Fatal("empty asset was signed")
	}
}

func TestSecretRoundTripAndFileSign(t *testing.T) {
	pub, priv, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseSecret(MarshalSecret(priv))
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(parsed.Public()) {
		t.Fatal("secret public key mismatch")
	}
	bare := []byte(strings.Split(strings.TrimSpace(string(MarshalSecret(priv))), "\n")[1])
	if _, err := ParseSecret(bare); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSecret(nil); err == nil {
		t.Fatal("expected empty secret to fail")
	}
	if _, err := ParseSecret([]byte("not base64")); err == nil {
		t.Fatal("expected bad secret to fail")
	}
	if _, err := ParseSecret([]byte(secretHeader + "\n")); err == nil {
		t.Fatal("expected incomplete secret to fail")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "aliyun-cli-macosx-1.2.3-arm64.tgz")
	if err := os.WriteFile(path, []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := SignFile(parsed, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyFile([]ed25519.PublicKey{pub}, "aliyun-cli-macosx-1.2.3-arm64.tgz", path, doc); err != nil {
		t.Fatal(err)
	}
	if _, err := SignFile(parsed, dir); err == nil {
		t.Fatal("expected directory to be rejected")
	}
}

func TestEmbeddedPublisherKey(t *testing.T) {
	keys := TrustedPublicKeys()
	if len(keys) != 1 || len(keys[0]) != ed25519.PublicKeySize {
		t.Fatalf("embedded keys = %d", len(keys))
	}
	if _, err := ParsePublicKey("@@@@"); err == nil {
		t.Fatal("expected invalid public key to fail")
	}
	if _, err := ParsePublicKey(""); err == nil {
		t.Fatal("expected empty public key to fail")
	}
	if _, err := ParsePublicKeys(nil); err == nil {
		t.Fatal("expected empty key list to fail")
	}
}

func TestFormatAndMalformedDocuments(t *testing.T) {
	if SignatureName("a.tgz") != "a.tgz.minisig" {
		t.Fatal(SignatureName("a.tgz"))
	}
	pub, priv, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if FormatPublicKey(pub) == "" {
		t.Fatal("empty public key encoding")
	}
	keys := []ed25519.PublicKey{pub}
	body := []byte("x")
	doc, err := Sign(priv, "a.tgz", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	badSig := bytes.Replace(doc, []byte("signature: "), []byte("signature: !!!"), 1)
	if err := Verify(keys, "a.tgz", bytes.NewReader(body), badSig); err == nil {
		t.Fatal("invalid signature encoding was accepted")
	}
	for _, raw := range []string{
		"nope\nasset: a.tgz\nkey: ab\nsignature: AA\n",
		formatID + "\nasset:\nkey: ab\nsignature: AA\n",
		formatID + "\nno-asset\nkey: ab\nsignature: AA\n",
		formatID + "\nasset: a.tgz\nkey:\nsignature: AA\n",
		formatID + "\nasset: a.tgz\nno-key\nsignature: AA\n",
		formatID + "\nasset: a.tgz\nkey: ab\nsignature:\n",
		formatID + "\nasset: a.tgz\nkey: ab\nno-signature\n",
		formatID + "\nasset: ../a.tgz\nkey: ab\nsignature: AA\n",
	} {
		if err := Verify(keys, "a.tgz", bytes.NewReader(body), []byte(raw)); err == nil {
			t.Fatalf("accepted malformed signature %q", raw)
		}
	}
	for _, name := range []string{" a.tgz", `a\b.tgz`, "a\nb.tgz", "a/b.tgz"} {
		if _, err := Sign(priv, name, bytes.NewReader(body)); err == nil {
			t.Fatalf("signed invalid name %q", name)
		}
	}
	if _, err := Sign(ed25519.PrivateKey{1, 2, 3}, "a.tgz", bytes.NewReader(body)); err == nil {
		t.Fatal("short private key was accepted")
	}
	if _, err := ParseSecret([]byte("@@@")); err == nil {
		t.Fatal("expected undecodable secret to fail")
	}
	short := base64.StdEncoding.EncodeToString([]byte("too-short"))
	if _, err := ParseSecret([]byte(short)); err == nil {
		t.Fatal("expected short secret to fail")
	}
	if _, err := ParseSecret([]byte("one\ntwo\nthree")); err == nil {
		t.Fatal("expected extra secret lines to fail")
	}
	dir := t.TempDir()
	if _, err := SignFile(priv, filepath.Join(dir, "missing.tgz")); err == nil {
		t.Fatal("expected missing file to fail")
	}
	empty := filepath.Join(dir, "empty.tgz")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SignFile(priv, empty); err == nil {
		t.Fatal("expected empty file to fail")
	}
	if err := VerifyFile(keys, "a.tgz", filepath.Join(dir, "missing.tgz"), doc); err == nil {
		t.Fatal("expected missing verify target to fail")
	}
}
