package config

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/util"
)

// Manual login uses the existing registered redirect URI without binding it.
// The user transports the callback URL from the browser back to this process.
func startManualOauthFlow(w io.Writer, cp *Profile) error {
	if cp.OAuthSiteType != "CN" && cp.OAuthSiteType != "INTL" {
		return fmt.Errorf("invalid OAuth site type: %s, only support CN or INTL", cp.OAuthSiteType)
	}
	const redirectURI = "http://127.0.0.1:12345/cli/callback"
	state := util.RandStringBytesMaskImprSrc(16).(string)
	verifier := generateCodeVerifier()
	query := url.Values{
		"response_type": {"code"}, "client_id": {oauthClientMap[cp.OAuthSiteType]},
		"redirect_uri": {redirectURI}, "state": {state},
		"code_challenge": {generateCodeChallenge(verifier)}, "code_challenge_method": {"S256"},
	}
	authURL := signInMap[cp.OAuthSiteType] + "/oauth2/v1/auth?" + query.Encode()
	if _, err := fmt.Fprintf(w, `Open this link in a browser on your computer or phone:
%s

Sign in. The browser will redirect to a local address starting with:
%s
The page may fail to load. This is expected.

Paste the entire redirected URL from your browser's address bar below.
Example: %s?code=abc123xyz&state=example-state

`, authURL, redirectURI, redirectURI); err != nil {
		return err
	}
	for {
		if _, err := fmt.Fprint(w, "Callback URL (Ctrl+C to cancel): "); err != nil {
			return err
		}
		line, err := readOAuthCallbackLine(stdin)
		if err != nil {
			return fmt.Errorf("cannot read OAuth callback URL: %w", err)
		}
		code, denied, err := parseOAuthCallbackURL(strings.TrimSpace(line), redirectURI, state)
		if err != nil {
			if denied {
				return err
			}
			if _, writeErr := fmt.Fprintf(w, "%s. Please paste the complete callback URL for this login.\n", err); writeErr != nil {
				return writeErr
			}
			continue
		}
		return exchangeOAuthCode(context.Background(), cp, code, verifier, redirectURI)
	}
}

// Do not read ahead: subsequent configure prompts share the same input stream.
func readOAuthCallbackLine(r io.Reader) (string, error) {
	var line strings.Builder
	var b [1]byte
	for line.Len() < 16*1024 {
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			if err == io.EOF && line.Len() > 0 {
				return line.String(), nil
			}
			return "", err
		}
		if b[0] == '\n' {
			return line.String(), nil
		}
		line.WriteByte(b[0])
	}
	return "", fmt.Errorf("callback URL is too long")
}

func parseOAuthCallbackURL(raw, redirectURI, state string) (code string, denied bool, err error) {
	u, parseErr := url.Parse(raw)
	expected, _ := url.Parse(redirectURI)
	if parseErr != nil || u.Scheme != expected.Scheme || u.Host != expected.Host ||
		u.Path != expected.Path || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return "", false, fmt.Errorf("invalid OAuth callback address")
	}
	query, parseErr := url.ParseQuery(u.RawQuery)
	if parseErr != nil {
		return "", false, fmt.Errorf("invalid OAuth callback parameters")
	}
	if len(query["state"]) != 1 || query.Get("state") != state {
		return "", false, fmt.Errorf("invalid state")
	}
	if len(query["error"]) > 0 {
		return "", true, fmt.Errorf("OAuth authorization was denied or failed; restart login to try again")
	}
	if len(query["code"]) != 1 || strings.TrimSpace(query.Get("code")) == "" {
		return "", false, fmt.Errorf("missing or duplicate authorization code")
	}
	return query.Get("code"), false, nil
}
