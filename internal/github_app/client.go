package github_app

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	acceptHeader = "application/vnd.github.v3+json"
)

type transport struct {
	appID  int64
	signer Signer
}

func New(appID int64, privateKeyPem []byte) (*http.Client, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPem)
	if err != nil {
		return nil, fmt.Errorf("could not parse private key: %w", err)
	}

	return &http.Client{
		Transport: &transport{
			appID:  appID,
			signer: &RSASigner{key: privateKey},
		},
	}, nil
}

func (g *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	appAccessToken, err := g.appToken()
	if err != nil {
		return nil, fmt.Errorf("get app token: %w", err)
	}

	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Authorization", "Bearer "+appAccessToken)
	req.Header.Set("Accept", acceptHeader)

	return http.DefaultTransport.RoundTrip(req)
}

func (g *transport) appToken() (string, error) {
	iss := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	exp := iss.Add(2 * time.Minute)
	claims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(iss),
		ExpiresAt: jwt.NewNumericDate(exp),
		Issuer:    strconv.FormatInt(g.appID, 10),
	}

	ss, err := g.signer.Sign(claims)
	if err != nil {
		return "", fmt.Errorf("could not sign jwt: %s", err)
	}

	return ss, nil
}
