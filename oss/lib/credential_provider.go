package lib

import (
	"fmt"
	"sync"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/credentials-go/credentials"
)

type hostCredentialSource interface {
	GetCredential() (*credentials.CredentialModel, error)
}
type credentialSnapshot struct{ id, secret, token string }

func (c credentialSnapshot) GetAccessKeyID() string     { return c.id }
func (c credentialSnapshot) GetAccessKeySecret() string { return c.secret }
func (c credentialSnapshot) GetSecurityToken() string   { return c.token }

// Serialize host provider access across multipart workers; one complete model
// supplies all three signing fields. Refresh errors abort before request dispatch.
type hostOSSProvider struct {
	mu      sync.Mutex
	source  hostCredentialSource
	current credentialSnapshot
	secrets []string
}
type credentialRefreshError struct{ err error }

func (e *credentialRefreshError) Error() string {
	return fmt.Sprintf("OSS credential refresh failed: %v", e.err)
}
func (e *credentialRefreshError) Unwrap() error { return e.err }

var bridgeCredentialProvider *hostOSSProvider
var _ oss.CredentialsProviderE = (*hostOSSProvider)(nil)

func (p *hostOSSProvider) GetCredentials() oss.Credentials {
	// SDK URL signing has no error channel. It uses the initial, already resolved
	// snapshot; network requests use GetCredentialsE and never fall back on error.
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}
func (p *hostOSSProvider) GetCredentialsE() (oss.Credentials, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	model, err := p.source.GetCredential()
	if err != nil {
		return nil, &credentialRefreshError{err}
	}
	if model == nil || model.AccessKeyId == nil || model.AccessKeySecret == nil || *model.AccessKeyId == "" || *model.AccessKeySecret == "" {
		return nil, fmt.Errorf("OSS credential provider returned incomplete credentials")
	}
	next := credentialSnapshot{id: *model.AccessKeyId, secret: *model.AccessKeySecret}
	if model.SecurityToken != nil {
		next.token = *model.SecurityToken
	}
	for _, s := range []string{next.id, next.secret, next.token} {
		if s == "" {
			continue
		}
		found := false
		for _, old := range p.secrets {
			if old == s {
				found = true
				break
			}
		}
		if !found {
			p.secrets = append(p.secrets, s)
		}
	}
	p.current = next
	return next, nil
}
func (p *hostOSSProvider) redact(message string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return redactOSSValues(message, p.secrets)
}
