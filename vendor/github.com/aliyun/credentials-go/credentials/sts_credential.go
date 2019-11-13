package credentials

// StsTokenCredential is a kind of credentials
type StsTokenCredential struct {
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string
}

func newStsTokenCredential(accessKeyID, accessKeySecret, securityToken string) *StsTokenCredential {
	return &StsTokenCredential{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		SecurityToken:   securityToken,
	}
}

// GetAccessKeyID reutrns  StsTokenCredential's AccessKeyID
func (s *StsTokenCredential) GetAccessKeyID() (string, error) {
	return s.AccessKeyID, nil
}

// GetAccessSecret reutrns  StsTokenCredential's AccessKeySecret
func (s *StsTokenCredential) GetAccessSecret() (string, error) {
	return s.AccessKeySecret, nil
}

// GetSecurityToken reutrns  StsTokenCredential's SecurityToken
func (s *StsTokenCredential) GetSecurityToken() (string, error) {
	return s.SecurityToken, nil
}

// GetBearerToken is useless StsTokenCredential
func (s *StsTokenCredential) GetBearerToken() string {
	return ""
}

// GetType reutrns  StsTokenCredential's type
func (s *StsTokenCredential) GetType() string {
	return "sts"
}
