package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v93/github"
)

func TestGetAppInstallation(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/orgs/test-org/installation" {
					t.Errorf("request = %s %s, want GET /orgs/test-org/installation", r.Method, r.URL.Path)
				}
				w.WriteHeader(status)
				if status == http.StatusOK {
					_, _ = fmt.Fprint(w, `{"id":42}`)
				} else {
					_, _ = fmt.Fprint(w, `{"message":"installation lookup failed"}`)
				}
			}))
			defer server.Close()

			baseURL := server.URL + "/"
			client, err := github.NewClient(github.WithURLs(&baseURL, nil))
			if err != nil {
				t.Fatal(err)
			}
			installation, err := getAppInstallation(t.Context(), client, "test-org")
			if status != http.StatusOK {
				if err == nil {
					t.Fatal("expected GitHub API error")
				}
				if installation != nil {
					t.Errorf("installation = %v, want nil on error", installation)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if installation == nil || installation.GetID() != 42 {
				t.Errorf("installation = %v, want ID 42", installation)
			}
		})
	}
}
