package common

type Credential struct {
	SecretId  string
	SecretKey string
	Token     string
}

func NewCredential(secretId, secretKey string) *Credential {
	return &Credential{
		SecretId:  secretId,
		SecretKey: secretKey,
	}
}

func NewTokenCredential(secretId, secretKey, token string) *Credential {
	return &Credential{
		SecretId:  secretId,
		SecretKey: secretKey,
		Token:     token,
	}
}

func (c *Credential) GetCredentialParams() map[string]string {
	p := map[string]string{
		"SecretId": c.SecretId,
	}
	if c.Token != "" {
		p["Token"] = c.Token
	}
	return p
}
