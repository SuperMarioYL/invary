package cmd

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/SuperMarioYL/invary/internal/invariant"
	"github.com/spf13/cobra"
)

// checkCmd implements `invary check`: read one captured tool-call trace and one
// tool schema, run the four built-in contract invariants, print pass/fail with
// the minimal failing evidence. This is the m1 milestone — it isolates the
// invariant-evaluation primitive from any network call.
var checkCmd = &cobra.Command{
	Use:   "check --trace <file> --schema <file>",
	Short: "Evaluate the contract invariants on one captured tool-call trace",
	Long: `'invary check' runs the four built-in invariants (valid_json,
required_fields, arg_types, no_schema_drift) against a single captured
tool-call trace and prints which pass or fail with the minimal failing
evidence. It makes no network call: feed it a saved provider response so
the invariant primitive can be developed and demonstrated in isolation.

The trace may be a full chat-completions response, a bare assistant message
with tool_calls, a bare tool_call object, or a minimal {"arguments":…}
object. When the response carries multiple tool calls, the first is
evaluated.

Exit code is 0 only when every invariant passes; any breach (error or warn)
yields a non-zero exit so 'invary check' can gate a pipeline.`,
	Example: `  invary check --trace examples/sample_kimi_output.json --schema examples/tool-schema.json
  invary check --trace examples/sample_glm_output.json --schema examples/tool-schema.json`,
	SilenceUsage: true,
	RunE:         runCheck,
}

var (
	checkTracePath  string
	checkSchemaPath string
)

func init() {
	checkCmd.Flags().StringVar(&checkTracePath, "trace", "", "path to a captured tool-call trace JSON (required)")
	checkCmd.Flags().StringVar(&checkSchemaPath, "schema", "", "path to an OpenAI-compatible tool/function schema JSON (required)")
	_ = checkCmd.MarkFlagRequired("trace")
	_ = checkCmd.MarkFlagRequired("schema")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, _ []string) error {
	traceBytes, err := os.ReadFile(checkTracePath)
	if err != nil {
		return fmt.Errorf("read trace %q: %w", checkTracePath, err)
	}
	schemaBytes, err := os.ReadFile(checkSchemaPath)
	if err != nil {
		return fmt.Errorf("read schema %q: %w", checkSchemaPath, err)
	}

	sch, err := invariant.ParseSchema(schemaBytes)
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	calls, err := invariant.ParseTrace(traceBytes)
	if err != nil {
		return fmt.Errorf("trace: %w", err)
	}
	if len(calls) == 0 {
		return fmt.Errorf("trace contains no tool call to evaluate")
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "trace:   %s\n", checkTracePath)
	fmt.Fprintf(out, "schema:  %s\n", checkSchemaPath)

	// Every tool call in the trace is a contract candidate: gating on only the
	// first one let a response with a conforming first call and a broken second
	// call report 4 pass / 0 fail and exit 0.
	breaching := 0
	for i, tc := range calls {
		results := invariant.Eval(tc, sch)
		printCallReport(out, i, len(calls), tc, sch, results)
		if _, fail := invariant.Summary(results); fail > 0 {
			breaching++
		}
	}
	if len(calls) > 1 {
		fmt.Fprintf(out, "%d of %d tool calls breached a contract\n", breaching, len(calls))
	}
	if breaching > 0 {
		// Non-zero exit on any breach so 'invary check' can gate a pipeline.
		return errContractBreach
	}
	return nil
}

// truncatePreview caps s at maxBytes total, cutting on a UTF-8 rune boundary
// and appending "..." for whatever was dropped. A byte-wise cut would split a
// multi-byte rune (e.g. a Chinese city name) and print invalid UTF-8.
func truncatePreview(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes - len("...")
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut-- // back off to the start of the rune we would have split
	}
	return s[:cut] + "..."
}

// printCallReport writes one call's tool/args header, per-invariant table and
// tally. A trace with several calls labels each section so a breach can be
// attributed to the call that produced it; a single-call trace keeps the
// original report layout byte-for-byte.
func printCallReport(out io.Writer, index, total int, tc invariant.ToolCall, sch invariant.Schema, results []invariant.Result) {
	if total > 1 {
		if index > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "call %d/%d\n", index+1, total)
	}
	if tc.Name != "" {
		fmt.Fprintf(out, "tool:    %s\n", tc.Name)
	} else if sch.Name != "" {
		fmt.Fprintf(out, "tool:    %s\n", sch.Name)
	}
	if tc.Arguments != "" {
		fmt.Fprintf(out, "args:    %s\n", truncatePreview(tc.Arguments, 80))
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%-16s %-8s %-6s %s\n", "INVARIANT", "SEV", "STATE", "EVIDENCE")
	for _, r := range results {
		state := "✓ pass"
		if !r.Pass {
			state = "✗ FAIL"
		}
		ev := r.Evidence
		if ev == "" {
			ev = "-"
		}
		fmt.Fprintf(out, "%-16s %-8s %-6s %s\n", r.Invariant, r.Severity, state, ev)
	}
	fmt.Fprintln(out)
	pass, fail := invariant.Summary(results)
	fmt.Fprintf(out, "%d pass / %d fail\n", pass, fail)
}
