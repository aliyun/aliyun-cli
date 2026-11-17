// Package publishsig signs and verifies CLI release assets.
//
// The signed message binds the publisher key, the exact asset file name
// (version and platform), and the SHA-256 of the asset bytes. Checking the
// file name on the client rejects a valid signature copied onto another
// platform or version.
package publishsig

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	formatID     = "aliyun-cli-publish-signature v1"
	secretHeader = "aliyun-cli-publish-secret-key v1"
	maxSigBytes  = 4096
)

// SignatureName is the object published next to an asset.
func SignatureName(assetName string) string {
	return assetName + ".minisig"
}

// TrustedPublicKeys returns the keys embedded in this CLI.
func TrustedPublicKeys() []ed25519.PublicKey {
	keys, err := ParsePublicKeys(embeddedPublicKeys)
	if err != nil {
		panic("publishsig: embedded publisher key: " + err.Error())
	}
	return keys
}

// ParsePublicKeys decodes base64 raw Ed25519 public keys.
func ParsePublicKeys(encoded []string) ([]ed25519.PublicKey, error) {
	if len(encoded) == 0 {
		return nil, fmt.Errorf("no publisher keys")
	}
	keys := make([]ed25519.PublicKey, 0, len(encoded))
	for _, item := range encoded {
		key, err := ParsePublicKey(item)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// FormatPublicKey encodes a raw Ed25519 public key for keys.go and the CLI.
func FormatPublicKey(pub ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(pub)
}

// ParsePublicKey decodes one base64 raw Ed25519 public key.
func ParsePublicKey(encoded string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("publisher key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("publisher key: want %d bytes, got %d", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

// PublicKeyID is the short id written into a signature file.
func PublicKeyID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:8])
}

// Generate creates a new publisher key pair.
func Generate() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// MarshalSecret encodes a private key for the release secret store.
func MarshalSecret(priv ed25519.PrivateKey) []byte {
	body := secretHeader + "\n" + base64.StdEncoding.EncodeToString(priv) + "\n"
	return []byte(body)
}

// ParseSecret reads a release secret. A bare base64 line is accepted so a
// store that strips the header still works.
func ParseSecret(raw []byte) (ed25519.PrivateKey, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return nil, fmt.Errorf("empty publisher secret")
	}
	lines := strings.Split(text, "\n")
	encoded := strings.TrimSpace(lines[0])
	if encoded == secretHeader {
		if len(lines) != 2 {
			return nil, fmt.Errorf("publisher secret: want header and key")
		}
		encoded = strings.TrimSpace(lines[1])
	} else if len(lines) != 1 {
		return nil, fmt.Errorf("publisher secret: unrecognized format")
	}
	seed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("publisher secret: %w", err)
	}
	if len(seed) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("publisher secret: want %d bytes, got %d", ed25519.PrivateKeySize, len(seed))
	}
	return ed25519.PrivateKey(seed), nil
}

// Sign hashes r and returns the signature document for assetName.
func Sign(priv ed25519.PrivateKey, assetName string, r io.Reader) ([]byte, error) {
	if err := cleanAssetName(assetName); err != nil {
		return nil, err
	}
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("publisher secret: invalid key")
	}
	digest, err := hashHex(r)
	if err != nil {
		return nil, err
	}
	pub := priv.Public().(ed25519.PublicKey)
	keyID := PublicKeyID(pub)
	msg := signedMessage(assetName, keyID, digest)
	sig := ed25519.Sign(priv, msg)
	doc := formatID + "\n" +
		"asset: " + assetName + "\n" +
		"key: " + keyID + "\n" +
		"signature: " + base64.StdEncoding.EncodeToString(sig) + "\n"
	return []byte(doc), nil
}

// fileStat reads the opened asset. Tests replace it.
var fileStat = func(f *os.File) (os.FileInfo, error) { return f.Stat() }

// SignFile signs the file at path. The asset name is the base name of path.
func SignFile(priv ed25519.PrivateKey, path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := fileStat(f)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return nil, fmt.Errorf("asset is missing or empty")
	}
	return Sign(priv, info.Name(), f)
}

// Verify checks that sigText was produced by one of keys for assetName and
// the bytes read from r.
func Verify(keys []ed25519.PublicKey, assetName string, r io.Reader, sigText []byte) error {
	if err := cleanAssetName(assetName); err != nil {
		return err
	}
	if len(keys) == 0 {
		return fmt.Errorf("no trusted publisher keys")
	}
	if len(sigText) == 0 || len(sigText) > maxSigBytes {
		return fmt.Errorf("publisher signature is missing or too large")
	}
	parsed, err := parseSignature(sigText)
	if err != nil {
		return err
	}
	if parsed.asset != assetName {
		return fmt.Errorf("publisher signature is for %q", parsed.asset)
	}
	pub, err := findKey(keys, parsed.keyID)
	if err != nil {
		return err
	}
	digest, err := hashHex(r)
	if err != nil {
		return err
	}
	msg := signedMessage(assetName, parsed.keyID, digest)
	raw, err := base64.StdEncoding.DecodeString(parsed.signature)
	if err != nil {
		return fmt.Errorf("publisher signature: %w", err)
	}
	if !ed25519.Verify(pub, msg, raw) {
		return fmt.Errorf("publisher signature does not match %s", assetName)
	}
	return nil
}

// VerifyFile checks the asset on disk.
func VerifyFile(keys []ed25519.PublicKey, assetName, path string, sigText []byte) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Verify(keys, assetName, f, sigText)
}

type signatureDoc struct {
	asset     string
	keyID     string
	signature string
}

func parseSignature(raw []byte) (signatureDoc, error) {
	text := strings.TrimSuffix(string(raw), "\n")
	lines := strings.Split(text, "\n")
	if len(lines) != 4 || lines[0] != formatID {
		return signatureDoc{}, fmt.Errorf("publisher signature: unrecognized document")
	}
	asset, ok := strings.CutPrefix(lines[1], "asset: ")
	if !ok || asset == "" || strings.TrimSpace(asset) != asset {
		return signatureDoc{}, fmt.Errorf("publisher signature: missing asset")
	}
	keyID, ok := strings.CutPrefix(lines[2], "key: ")
	if !ok || keyID == "" {
		return signatureDoc{}, fmt.Errorf("publisher signature: missing key")
	}
	sig, ok := strings.CutPrefix(lines[3], "signature: ")
	if !ok || sig == "" {
		return signatureDoc{}, fmt.Errorf("publisher signature: missing signature")
	}
	if err := cleanAssetName(asset); err != nil {
		return signatureDoc{}, err
	}
	return signatureDoc{asset: asset, keyID: keyID, signature: sig}, nil
}

func findKey(keys []ed25519.PublicKey, keyID string) (ed25519.PublicKey, error) {
	for _, key := range keys {
		if PublicKeyID(key) == keyID {
			return key, nil
		}
	}
	return nil, fmt.Errorf("publisher key %s is not trusted", keyID)
}

func signedMessage(assetName, keyID, digest string) []byte {
	return []byte(formatID + "\n" +
		"asset: " + assetName + "\n" +
		"key: " + keyID + "\n" +
		"sha256: " + digest + "\n")
}

func hashHex(r io.Reader) (string, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("asset is missing or empty")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func cleanAssetName(name string) error {
	if name == "" || name != strings.TrimSpace(name) {
		return fmt.Errorf("invalid asset name")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid asset name")
	}
	if strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("invalid asset name")
	}
	return nil
}
