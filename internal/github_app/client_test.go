package github_app

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewRejectsInvalidKey(t *testing.T) {
	for _, key := range []string{
		"",
		"not a PEM key",
		string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("invalid")})),
	} {
		if _, err := New(123, []byte(key)); err == nil {
			t.Errorf("New accepted invalid key %q", key)
		}
	}
}

func TestClientAuthenticatesRequests(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	client, err := New(123, keyPEM)
	if err != nil {
		t.Fatal(err)
	}

	earliestIssuedAt := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != acceptHeader {
			t.Errorf("Accept = %q, want %q", got, acceptHeader)
		}
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Authorization is missing Bearer prefix")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		claims := new(jwt.RegisteredClaims)
		_, err := jwt.ParseWithClaims(strings.TrimPrefix(auth, "Bearer "), claims, func(_ *jwt.Token) (any, error) {
			return &key.PublicKey, nil
		}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer("123"), jwt.WithIssuedAt(), jwt.WithExpirationRequired())
		if err != nil {
			t.Errorf("invalid app JWT: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if claims.IssuedAt == nil || claims.ExpiresAt == nil {
			t.Error("JWT must include issued-at and expiry")
			return
		}
		latestIssuedAt := time.Now().Add(-30 * time.Second).Truncate(time.Second)
		if claims.IssuedAt.Before(earliestIssuedAt) || claims.IssuedAt.After(latestIssuedAt) {
			t.Errorf("JWT issued at %s, want between %s and %s", claims.IssuedAt.Time, earliestIssuedAt, latestIssuedAt)
		}
		if lifetime := claims.ExpiresAt.Sub(claims.IssuedAt.Time); lifetime != 2*time.Minute {
			t.Errorf("JWT lifetime = %s, want 2m", lifetime)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			var body io.Reader
			if method == http.MethodPost {
				body = strings.NewReader(`{}`)
			}
			req, err := http.NewRequestWithContext(t.Context(), method, server.URL, body)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Error(err)
				}
			}()
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
			}
		})
	}
}
