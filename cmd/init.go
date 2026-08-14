package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// initCmd implements `invary init`: scaffold an invariants.yaml and copy the
// example tool schema into the working tree, so a fresh-clone user can go from
// 'go install' to 'invary run' in under five minutes. This is milestone m3
// polish; m1 ships the command skeleton only.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold an invariants.yaml + copy the example schema (milestone m3)",
	Long: `'invary init' writes an invariants.yaml (the user-designated
contract set) into the current directory and copies examples/tool-schema.json
beside it, so a custom invariant workflow starts from a working example.

This command is milestone m3 work and is not yet implemented in this build.
The built-in four invariants are always available via 'invary check' without
any config file.`,
	SilenceUsage: true,
	RunE: func(*cobra.Command, []string) error {
		return fmt.Errorf("invary init is not available in this build — milestone m3 will write invariants.yaml and copy the example schema")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
