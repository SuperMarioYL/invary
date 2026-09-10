package cmd

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// The starter files invary init writes are embedded (a go-installed binary has
// no examples/ directory next to it). They are byte-copies of examples/ and
// kept identical by a drift test.
const (
	invariantsYAMLName = "invariants.yaml"
	toolSchemaJSONName = "tool-schema.json"
)

//go:embed templates/invariants.yaml
var invariantsTemplate string

//go:embed templates/tool-schema.json
var toolSchemaTemplate string

// initCmd implements `invary init`: scaffold an invariants.yaml and a starter
// tool schema into the working directory, so a custom invariant workflow
// starts from a working example. This is the m3 milestone's offline core.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write an invariants.yaml + tool-schema.json starter set",
	Long: `'invary init' writes an invariants.yaml (the user-designated
contract set) and a tool-schema.json starter into the current directory, so a
custom invariant workflow starts from a working example. It never overwrites:
if either file already exists, init fails without touching anything.

The starter files are the same ones shipped in the repo's examples/ directory.
'invary check' always runs the built-in four invariants without any config
file; the written invariants.yaml documents the DSL for customizing the set.`,
	Example:      "  invary init",
	SilenceUsage: true,
	RunE:         runInit,
}

func runInit(cmd *cobra.Command, _ []string) error {
	files := []struct{ name, content string }{
		{invariantsYAMLName, invariantsTemplate},
		{toolSchemaJSONName, toolSchemaTemplate},
	}
	// Refuse before writing anything so a failed init leaves the tree untouched.
	for _, f := range files {
		if _, err := os.Stat(f.name); err == nil {
			return fmt.Errorf("%s already exists — invary init will not overwrite it; remove the file or run in an empty directory", f.name)
		}
	}
	out := cmd.OutOrStdout()
	for _, f := range files {
		if err := os.WriteFile(f.name, []byte(f.content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", f.name, err)
		}
		fmt.Fprintf(out, "wrote %s\n", f.name)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next: capture a tool-call trace from your provider, then run")
	fmt.Fprintf(out, "  invary check --trace <trace.json> --schema %s\n", toolSchemaJSONName)
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
