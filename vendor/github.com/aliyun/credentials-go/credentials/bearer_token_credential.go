package credentials

// BearerTokenCredential is a kind of credential
type BearerTokenCredential struct {
	BearerToken string
}

// newBearerTokenCredential return a BearerTokenCredential object
func newBearerTokenCredential(token string) *BearerTokenCredential {
	return &BearerTokenCredential{
		BearerToken: token,
	}
}

// GetAccessKeyID is useless for BearerTokenCredential
func (b *BearerTokenCredential) GetAccessKeyID() (string, error) {
	return "", nil
}

// GetAccessSecret is useless for BearerTokenCredential
func (b *BearerTokenCredential) GetAccessSecret() (string, error) {
	return "", nil
}

// GetSecurityToken is useless for BearerTokenCredential
func (b *BearerTokenCredential) GetSecurityToken() (string, error) {
	return "", nil
}

// GetBearerToken reutrns  BearerTokenCredential's BearerToken
func (b *BearerTokenCredential) GetBearerToken() string {
	return b.BearerToken
}

// GetType reutrns  BearerTokenCredential's type
func (b *BearerTokenCredential) GetType() string {
	return "bearer"
}
