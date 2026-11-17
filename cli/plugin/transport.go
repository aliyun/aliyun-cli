package plugin

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(strings.ToLower(host)), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// validatePluginTransportURL accepts HTTPS only. HTTP is refused, including
// loopback. go test still allows loopback HTTP so httptest fixtures can run;
// that switch is not a CLI option.
func (m *Manager) validatePluginTransportURL(raw string) error {
	return validatePluginTransport(raw, testing.Testing())
}

func validatePluginTransport(raw string, allowLoopbackHTTP bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || u.Host == "" {
		return fmt.Errorf("plugin URL must use https://")
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if allowLoopbackHTTP && isLoopbackHost(u.Hostname()) {
			return nil
		}
		return fmt.Errorf("refusing insecure HTTP URL %s: plugin sources and packages must use HTTPS", safePluginPackageURL(u))
	default:
		return fmt.Errorf("plugin URL must use https://")
	}
}

func httpsDowngradedToHTTP(from, to *url.URL) bool {
	if from == nil || to == nil {
		return false
	}
	return strings.EqualFold(from.Scheme, "https") && strings.EqualFold(to.Scheme, "http")
}

func (m *Manager) checkPluginRedirect(req *http.Request, via []*http.Request) error {
	if req == nil || req.URL == nil {
		return fmt.Errorf("invalid plugin redirect")
	}
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if err := m.validatePluginTransportURL(req.URL.String()); err != nil {
		return err
	}
	if len(via) > 0 && via[0] != nil && httpsDowngradedToHTTP(via[0].URL, req.URL) {
		return fmt.Errorf("refusing redirect that downgrades HTTPS to HTTP (%s)", safePluginPackageURL(req.URL))
	}
	return nil
}

func (m *Manager) acceptPluginTransport(raw string) error {
	return m.validatePluginTransportURL(raw)
}

func normalizeSHA256Checksum(expected string) (string, error) {
	expected = strings.TrimSpace(strings.ToLower(expected))
	expected = strings.TrimPrefix(expected, "sha256:")
	if len(expected) != hex.EncodedLen(32) {
		return "", fmt.Errorf("checksum must be a 64-character SHA-256 hex digest")
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return "", fmt.Errorf("checksum must be a 64-character SHA-256 hex digest")
	}
	return expected, nil
}

func verifySHA256File(path, expected string) error {
	want, err := normalizeSHA256Checksum(expected)
	if err != nil {
		return err
	}
	actual, err := calculateSHA256(path)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	if actual != want {
		return fmt.Errorf(
			"checksum verification failed\nExpected: %s\nActual:   %s\nThe package may be corrupted or tampered with",
			want, actual,
		)
	}
	return nil
}

func reportPackageInstall(ctx *cli.Context, name, version string, verified bool) {
	if verified {
		cli.Printf(ctx.Stdout(), "Plugin %s %s installed successfully!\nSHA-256 checksum verified.\n", name, version)
		return
	}
	cli.Printf(ctx.Stdout(), "Plugin %s %s installed successfully!\nChecksum was not provided; package integrity was not checked.\n", name, version)
}
