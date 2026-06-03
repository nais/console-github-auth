package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/nais/console-github-auth/internal/github_app"
)

var (
	port                 = os.Getenv("PORT")
	githubOrg            = os.Getenv("GITHUB_ORG")
	githubAppIDString    = os.Getenv("GITHUB_APP_ID")
	githubPrivateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH")
)

func main() {
	ctx := context.Background()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	githubPrivateKey, err := os.ReadFile(githubPrivateKeyPath) // #nosec G304 -- path from trusted env var
	if err != nil {
		log.
			With("error", err, "path", githubPrivateKeyPath).
			Error("could not read GitHub private key")
		os.Exit(1)
	}

	githubAppID, err := strconv.Atoi(githubAppIDString)
	if err != nil {
		log.
			With("error", err).
			Error("could not parse GitHub app ID")
		os.Exit(1)
	}

	httpClient, err := github_app.New(int64(githubAppID), githubPrivateKey)
	if err != nil {
		log.
			With("error", err).
			Error("create GitHub HTTP client")
		os.Exit(1)
	}

	githubClient := github.NewClient(httpClient)

	appInstallation, err := getAppInstallation(ctx, githubClient, githubOrg)
	if err != nil {
		log.
			With("error", err, "github_org", githubOrg).
			Warn("no GitHub installation found for org")
	} else {
		log.
			With("installation_id", appInstallation.GetID()).
			Info("ready to serve tokens for installation")
	}

	http.HandleFunc("/createInstallationToken", func(w http.ResponseWriter, r *http.Request) {
		if appInstallation == nil {
			appInstallation, err = getAppInstallation(ctx, githubClient, githubOrg)
			if err != nil {
				log.
					With("error", err, "github_org", githubOrg).
					Warn("no GitHub installation found for org - aborting token creation")
				_, _ = fmt.Fprintf(w, "no GitHub installation found. Please install the nais/console app in your GitHub org.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		token, _, err := githubClient.Apps.CreateInstallationToken(r.Context(), appInstallation.GetID(), nil)
		if err != nil {
			log.
				With("error", err).
				Error("create installation token")
			_, _ = fmt.Fprintf(w, "installation token error: %v", err)
			return
		}

		if err := json.NewEncoder(w).Encode(token); err != nil {
			err := fmt.Errorf("encode token: %v", err)
			log.
				With("error", err).
				Error("encode token")
			if _, err := fmt.Fprint(w, err.Error()); err != nil {
				log.
					With("error", err).
					Error("write error to client")
			}
			return
		}
	})

	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.
			With("error", err, "port", port).
			Error("create listener")
		os.Exit(1)
	}

	log.Info("listening", "port", l.Addr().String())
	srv := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.
			With("error", err).
			Error("error stopping server")
		os.Exit(1)
	}

	log.Info("successful shut down")
}

func getAppInstallation(ctx context.Context, client *github.Client, organization string) (*github.Installation, error) {
	appInstallation, _, err := client.Apps.FindOrganizationInstallation(ctx, organization)
	return appInstallation, err
}
