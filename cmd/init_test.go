package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbeddedTemplatesMatchExamples guards the cmd/templates/ copies against
// drift from examples/ (the user-visible canonical files).
func TestEmbeddedTemplatesMatchExamples(t *testing.T) {
	for _, pair := range []struct{ embedded, example string }{
		{invariantsTemplate, filepath.Join("..", "examples", "invariants.yaml")},
		{toolSchemaTemplate, filepath.Join("..", "examples", "tool-schema.json")},
	} {
		want, err := os.ReadFile(pair.example)
		if err != nil {
			t.Fatalf("read %s: %v", pair.example, err)
		}
		if pair.embedded != string(want) {
			t.Fatalf("embedded template for %s drifted from %s — copy the file again", pair.example, pair.example)
		}
	}
}

// TestInitWritesTemplates runs the real init command in an empty temp dir.
func TestInitWritesTemplates(t *testing.T) {
	// Resolve the canonical examples/ contents before chdir moves the cwd.
	wantInvariants, err := os.ReadFile(filepath.Join("..", "examples", "invariants.yaml"))
	if err != nil {
		t.Fatalf("read examples/invariants.yaml: %v", err)
	}
	wantSchema, err := os.ReadFile(filepath.Join("..", "examples", "tool-schema.json"))
	if err != nil {
		t.Fatalf("read examples/tool-schema.json: %v", err)
	}

	t.Chdir(t.TempDir())

	var out strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("invary init failed: %v", err)
	}

	for name, want := range map[string][]byte{
		"invariants.yaml":  wantInvariants,
		"tool-schema.json": wantSchema,
	} {
		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("init should write %s: %v", name, err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s content differs from examples/", name)
		}
	}
	if !strings.Contains(out.String(), "wrote invariants.yaml") || !strings.Contains(out.String(), "wrote tool-schema.json") {
		t.Fatalf("init output should name both written files, got:\n%s", out.String())
	}
}

// TestInitRefusesOverwrite pins the never-overwrite contract: a failed init
// must leave the tree untouched.
func TestInitRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	marker := []byte("# do not clobber\n")
	if err := os.WriteFile("invariants.yaml", marker, 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("init must fail when invariants.yaml already exists")
	}
	if !strings.Contains(err.Error(), "invariants.yaml already exists") {
		t.Fatalf("error should name the existing file, got %q", err)
	}

	got, readErr := os.ReadFile("invariants.yaml")
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(marker) {
		t.Fatalf("existing invariants.yaml must be untouched, got %q", got)
	}
	if _, statErr := os.Stat("tool-schema.json"); statErr == nil {
		t.Fatalf("init must not write tool-schema.json when it refuses")
	}
}
