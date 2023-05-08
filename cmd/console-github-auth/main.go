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

	"github.com/google/go-github/v52/github"
	"github.com/nais/console-github-auth/internal/github_app"
	log "github.com/sirupsen/logrus"
)

var (
	port                 = os.Getenv("PORT")
	githubOrg            = os.Getenv("GITHUB_ORG")
	githubAppIDString    = os.Getenv("GITHUB_APP_ID")
	githubPrivateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH")
)

func getAppInstallation(ctx context.Context, client *github.Client, organization string) (*github.Installation, error) {
	appInstallation, _, err := client.Apps.FindOrganizationInstallation(ctx, organization)
	return appInstallation, err
}

func main() {
	ctx := context.Background()

	githubPrivateKey, err := os.ReadFile(githubPrivateKeyPath)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	githubAppID, err := strconv.Atoi(githubAppIDString)
	if err != nil {
		log.Errorf("could not parse app id: %v", err)
		os.Exit(1)
	}

	appClient, err := github_app.New(int64(githubAppID), githubPrivateKey)
	if err != nil {
		log.Errorf("could get GitHub app client: %v", err)
		os.Exit(1)
	}

	githubClient := github.NewClient(appClient)

	appInstallation, err := getAppInstallation(ctx, githubClient, githubOrg)
	if err != nil {
		log.Warnf("no GitHub installation found for org %v (yet). err: %v", githubOrg, err)
	} else {
		log.Infof("ready to serve tokens for installation: %v", appInstallation.GetID())
	}

	http.HandleFunc("/createInstallationToken", func(w http.ResponseWriter, r *http.Request) {
		if appInstallation == nil {
			appInstallation, err = getAppInstallation(ctx, githubClient, githubOrg)
			if err != nil {
				log.Warnf("no GitHub installation found for org %v - aborting token creation. err: %v", githubOrg, err)
				fmt.Fprintf(w, "no GitHub installation found. Please install the nais/console app in your GitHub org.")
				return
			}
		}

		token, _, err := githubClient.Apps.CreateInstallationToken(r.Context(), appInstallation.GetID(), nil)
		if err != nil {
			fmt.Fprintf(w, "installation token error: %v", err)
			return
		}

		if err := json.NewEncoder(w).Encode(token); err != nil {
			err := fmt.Errorf("encode token: %v", err)
			log.Errorf(err.Error())

			if _, err := fmt.Fprintf(w, err.Error()); err != nil {
				log.Errorf("write error to client: %v", err)
			}
			return
		}
	})

	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error("create listener: %v", err)
	}
	log.Infof("listening on: %v", l.Addr().String())
	if err := http.Serve(l, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("error stopping server: %v", err)
		os.Exit(1)
	}

	log.Info("shut down")
}
