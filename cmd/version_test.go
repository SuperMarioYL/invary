package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionLockstep asserts every version surface reports the same version:
// the repo VERSION file, the cmd.Version constant (ldflags-overridable), and
// the rendered `invary --version` output.
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
