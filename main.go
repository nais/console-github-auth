package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v52/github"
	log "github.com/sirupsen/logrus"
)

const (
	acceptHeader = "application/vnd.github.v3+json"
	apiBaseURL   = "https://api.github.com"
)

type State struct {
	appID                    int64
	signer                   ghinstallation.Signer
	installationTokenOptions *github.InstallationTokenOptions
	baseURL                  string
}

// accessToken is an installation access token response from GitHub
type accessToken struct {
	Token        string                         `json:"token"`
	ExpiresAt    time.Time                      `json:"expires_at"`
	Permissions  github.InstallationPermissions `json:"permissions,omitempty"`
	Repositories []github.Repository            `json:"repositories,omitempty"`
}

func (s *State) getAppToken() (string, error) {
	iss := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	exp := iss.Add(2 * time.Minute)
	claims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(iss),
		ExpiresAt: jwt.NewNumericDate(exp),
		Issuer:    strconv.FormatInt(s.appID, 10),
	}

	ss, err := s.signer.Sign(claims)
	if err != nil {
		return "", fmt.Errorf("could not sign jwt: %s", err)
	}

	return ss, nil
}

func main() {
	keyPem, err := os.ReadFile("key.pem")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyPem)
	if err != nil {
		log.Errorf("could not parse private key: %v", err)
		os.Exit(1)
	}

	state := State{
		signer:  ghinstallation.NewRSASigner(jwt.SigningMethodRS256, key),
		appID:   293384,
		baseURL: apiBaseURL,
		installationTokenOptions: &github.InstallationTokenOptions{
			RepositoryIDs: []int64{},
			Repositories:  []string{},
			Permissions:   &github.InstallationPermissions{},
		},
	}

	http.HandleFunc("/getToken", func(w http.ResponseWriter, r *http.Request) {
		token, err := state.accessToken(r.Context(), 1234)
		if err != nil {
			fmt.Fprintf(w, "access token error: %v", err)
			return
		}

		responseBody := &bytes.Buffer{}
		if err := json.NewEncoder(responseBody).Encode(token); err != nil {
			fmt.Fprintf(w, "encode error: %v", err)
			return
		}

		fmt.Fprintf(w, responseBody.String())
	})

	if err := http.ListenAndServe(":8080", nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("error stopping server: %v", err)
		os.Exit(1)
	}
	log.Info("shut down")
}

func (s *State) accessToken(ctx context.Context, installationID int64) (*accessToken, error) {
	body := &bytes.Buffer{}

	err := json.NewEncoder(body).Encode(s.installationTokenOptions)
	if err != nil {
		return nil, fmt.Errorf("could not convert installation token parameters into json: %v", err)
	}

	requestURL := fmt.Sprintf("%s/app/installations/%v/access_tokens", strings.TrimRight(s.baseURL, "/"), installationID)
	req, err := http.NewRequest("POST", requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	appAccessToken, err := s.getAppToken()
	if err != nil {
		return nil, fmt.Errorf("get app token: %w", err)
	}
	// Set Content and Accept headers.
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+appAccessToken)
	}
	req.Header.Set("Accept", acceptHeader)

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get access_tokens from GitHub API for installation ID %v: %w", installationID, err)
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("received non 2xx response status %q when fetching %v", resp.Status, req.URL)
	}

	t := &accessToken{}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("decode access token: %w", err)
	}

	if err := resp.Body.Close(); err != nil {
		log.Errorf("close response body: %v", err)
	}

	return t, nil
}
