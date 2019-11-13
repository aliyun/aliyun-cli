package credentials

// AccessKeyCredential is a kind of credential
type AccessKeyCredential struct {
	AccessKeyID     string
	AccessKeySecret string
}

func newAccessKeyCredential(accessKeyID, accessKeySecret string) *AccessKeyCredential {
	return &AccessKeyCredential{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
	}
}

// GetAccessKeyID reutrns  AccessKeyCreential's AccessKeyID
func (a *AccessKeyCredential) GetAccessKeyID() (string, error) {
	return a.AccessKeyID, nil
}

// GetAccessSecret reutrns  AccessKeyCreential's AccessKeySecret
func (a *AccessKeyCredential) GetAccessSecret() (string, error) {
	return a.AccessKeySecret, nil
}

// GetSecurityToken is useless for AccessKeyCreential
func (a *AccessKeyCredential) GetSecurityToken() (string, error) {
	return "", nil
}

// GetBearerToken is useless for AccessKeyCreential
func (a *AccessKeyCredential) GetBearerToken() string {
	return ""
}

// GetType reutrns  AccessKeyCreential's type
func (a *AccessKeyCredential) GetType() string {
	return "access_key"
}
