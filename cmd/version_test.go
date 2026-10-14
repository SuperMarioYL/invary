package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionLockstep asserts every version surface reports the same version:
// the repo VERSION file, the cmd.Version constant (ldflags-overridable), the
// rendered `invary --version` output, the website's meta.content_version, and
// a CHANGELOG.md section for the release.
func TestVersionLockstep(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	fileVersion := strings.TrimSpace(string(raw))
	if fileVersion == "" {
		t.Fatalf("VERSION file is empty")
	}
	if fileVersion != Version {
		t.Fatalf("version drift: VERSION file = %q, cmd.Version = %q", fileVersion, Version)
	}

	// The website's content_version participates in the same lockstep so a
	// release that bumps the CLI but not the site fails here.
	siteRaw, err := os.ReadFile(filepath.Join("..", "web", "site.json"))
	if err != nil {
		t.Fatalf("read web/site.json: %v", err)
	}
	var site struct {
		Meta struct {
			ContentVersion string `json:"content_version"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(siteRaw, &site); err != nil {
		t.Fatalf("parse web/site.json: %v", err)
	}
	if site.Meta.ContentVersion != Version {
		t.Fatalf("version drift: web/site.json meta.content_version = %q, cmd.Version = %q", site.Meta.ContentVersion, Version)
	}

	// The changelog must carry a section for the shipped version.
	changelogRaw, err := os.ReadFile(filepath.Join("..", "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	if !strings.Contains(string(changelogRaw), "## ["+Version+"]") {
		t.Fatalf("CHANGELOG.md has no ## [%s] section", Version)
	}

	var out strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("invary --version failed: %v", err)
	}
	if want := "invary version " + Version + "\n"; out.String() != want {
		t.Fatalf("--version output = %q, want %q", out.String(), want)
	}
}
