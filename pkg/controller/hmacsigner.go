package controller

import (
	"github.com/ServerPlace/iac-controller/pkg/hmac"
)

type SignerRemote struct {
	secretKey string
}

func NewSigner(secret string) *SignerRemote {
	return &SignerRemote{
		secretKey: secret,
	}
}

func (s *SignerRemote) Sign(object hmac.Signable) (string, error) {
	sig, err := hmac.Sign([]byte(s.secretKey), object)
	return string(sig), err
}
