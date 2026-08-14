package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is the semantic version of the invary binary. It is overridable at
// build time via -ldflags "-X github.com/SuperMarioYL/invary/cmd.Version=…"
// and otherwise tracks the repo VERSION file.
var Version = "0.1.0"

const rootLong = `Invary — 国产大模型工具调用契约差分测试器

同一个 prompt + 工具 schema 跑过 DeepSeek / Qwen / Kimi / GLM，Invary 报告
哪个模型静默地违反了其他模型都遵守的工具调用契约——把模型替换从盲赌
变成可度量的决策。

Run 'invary check' to evaluate the built-in contract invariants against one
captured tool-call trace, or 'invary run' (milestone m2) to drive the four
providers live and print the differential table.`

// rootCmd is the invary entry point.
var rootCmd = &cobra.Command{
	Use:   "invary",
	Short: "Flag which CN model silently breaks a tool-call contract",
	Long:  rootLong,
	// root has no Run of its own; subcommands carry the behavior.
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("invary version %s\n", Version))
}

// Execute runs the root command and translates a non-nil error into a
// non-zero process exit code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
