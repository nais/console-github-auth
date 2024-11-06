package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/google/go-github/v66/github"
	"github.com/nais/console-github-auth/internal/github_app"
	"github.com/sirupsen/logrus"
)

const (
	exitCodeSuccess = iota
	exitCodeGitHubPrivateKeyError
	exitCodeGitHubAppIDError
	exitCodeGitHubAppClientError
	exitCodeListenError
	exitCodeHttpServerError
)

var (
	port                 = os.Getenv("PORT")
	githubOrg            = os.Getenv("GITHUB_ORG")
	githubAppIDString    = os.Getenv("GITHUB_APP_ID")
	githubPrivateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH")
)

func main() {
	ctx := context.Background()
	log := logrus.New()

	githubPrivateKey, err := os.ReadFile(githubPrivateKeyPath)
	if err != nil {
		log.WithError(err).WithField("path", githubPrivateKeyPath).Error("could not read GitHub private key")
		os.Exit(exitCodeGitHubPrivateKeyError)
	}

	githubAppID, err := strconv.Atoi(githubAppIDString)
	if err != nil {
		log.WithError(err).Errorf("could not parse GitHub app ID")
		os.Exit(exitCodeGitHubAppIDError)
	}

	httpClient, err := github_app.New(int64(githubAppID), githubPrivateKey)
	if err != nil {
		log.WithError(err).Errorf("create GitHub HTTP client")
		os.Exit(exitCodeGitHubAppClientError)
	}

	githubClient := github.NewClient(httpClient)

	appInstallation, err := getAppInstallation(ctx, githubClient, githubOrg)
	if err != nil {
		log.WithError(err).WithField("github_org", githubOrg).Warnf("no GitHub installation found for org")
	} else {
		log.WithField("installation_id", appInstallation.GetID()).Infof("ready to serve tokens for installation")
	}

	http.HandleFunc("/createInstallationToken", func(w http.ResponseWriter, r *http.Request) {
		if appInstallation == nil {
			appInstallation, err = getAppInstallation(ctx, githubClient, githubOrg)
			if err != nil {
				log.WithError(err).WithField("github_org", githubOrg).Warnf("no GitHub installation found for org - aborting token creation")
				_, _ = fmt.Fprintf(w, "no GitHub installation found. Please install the nais/console app in your GitHub org.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		token, _, err := githubClient.Apps.CreateInstallationToken(r.Context(), appInstallation.GetID(), nil)
		if err != nil {
			log.WithError(err).Errorf("create installation token")
			_, _ = fmt.Fprintf(w, "installation token error: %v", err)
			return
		}

		if err := json.NewEncoder(w).Encode(token); err != nil {
			err := fmt.Errorf("encode token: %v", err)
			log.WithError(err).Errorf("encode token")
			if _, err := fmt.Fprint(w, err.Error()); err != nil {
				log.WithError(err).Errorf("write error to client")
			}
			return
		}
	})

	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.WithError(err).WithField("port", port).Errorf("create listener")
		os.Exit(exitCodeListenError)
	}

	log.WithField("port", l.Addr().String()).Infof("listening")
	if err := http.Serve(l, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.WithError(err).Errorf("error stopping server")
		os.Exit(exitCodeHttpServerError)
	}

	log.Info("successful shut down")
	os.Exit(exitCodeSuccess)
}

func getAppInstallation(ctx context.Context, client *github.Client, organization string) (*github.Installation, error) {
	appInstallation, _, err := client.Apps.FindOrganizationInstallation(ctx, organization)
	return appInstallation, err
}
