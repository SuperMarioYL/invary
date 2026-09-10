package cmd

import (
	"github.com/SuperMarioYL/invary/internal/diff"
	"github.com/spf13/cobra"
)

// runCmd implements `invary run`: drive the same prompt + tool schema across
// DeepSeek, Qwen, Kimi and GLM via the hand-rolled OpenAI-compatible client,
// then print the differential table with silent-breaker flags. This is the m2
// wedge; m1 ships the command skeleton so the entry point and flag surface are
// in place.
var runCmd = &cobra.Command{
	Use:   "run --prompt <text> --schema <file>",
	Short: "Run one prompt + tool across the four CN providers (milestone m2)",
	Long: `'invary run' sends the same prompt + tool schema to DeepSeek, Qwen,
Kimi and GLM in parallel, evaluates the four contract invariants on each
tool-call output, and prints the differential matrix with silent-breaker
flags — the model that broke a contract its peers kept.

This command is milestone m2 work and is not yet wired to the providers in
this build. Use 'invary check' to evaluate a single captured trace against
the invariants today.`,
	Example:      "  invary run --prompt \"What's the weather in Tokyo?\" --schema examples/tool-schema.json",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Flags are accepted so the documented surface is stable; the live
		// differential run lands in m2.
		_, err := diff.Runner{}.Run(runPrompt, runSchemaPath)
		return err
	},
}

var (
	runPrompt     string
	runSchemaPath string
)

func init() {
	runCmd.Flags().StringVar(&runPrompt, "prompt", "", "prompt text sent to every provider")
	runCmd.Flags().StringVar(&runSchemaPath, "schema", "", "path to an OpenAI-compatible tool/function schema JSON")
	rootCmd.AddCommand(runCmd)
}
