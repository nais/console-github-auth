package github_app

import (
	"crypto/rsa"

	"github.com/golang-jwt/jwt/v4"
)

type Signer interface {
	Sign(jwt.Claims) (string, error)
}

type RSASigner struct {
	key *rsa.PrivateKey
}

func (s *RSASigner) Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.key)
}
