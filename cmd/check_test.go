package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestErrorPrintedOnceByRoot pins the single-error-print contract: cobra is
// silenced (SilenceErrors on rootCmd) so Execute() prints the one canonical
// line. In v0.1.0 cobra printed "Error: …" AND the executor printed the raw
// error again — every CLI error appeared twice on stderr.
func TestErrorPrintedOnceByRoot(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Fatalf("rootCmd.SilenceErrors must be true so cobra does not duplicate the executor's error line")
	}
	var errBuf strings.Builder
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"check", "--trace", "/nope/missing-trace.json", "--schema", "/nope/missing-schema.json"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected an error for a missing trace file")
	}
	if !strings.Contains(err.Error(), "missing-trace.json") {
		t.Fatalf("error should name the trace file, got %q", err)
	}
	if strings.Contains(errBuf.String(), "Error:") {
		t.Fatalf("cobra must not print its own error line on top of Execute()'s; stderr buffer = %q", errBuf.String())
	}
}

func TestTruncatePreviewShortUnchanged(t *testing.T) {
	for _, s := range []string{``, `{"location":"Tokyo"}`, strings.Repeat("x", 80)} {
		if got := truncatePreview(s, 80); got != s {
			t.Fatalf("truncatePreview(%q, 80) = %q, want unchanged", s, got)
		}
	}
}

func TestTruncatePreviewLongASCII(t *testing.T) {
	got := truncatePreview(strings.Repeat("x", 200), 80)
	if want := strings.Repeat("x", 77) + "..."; got != want {
		t.Fatalf("ascii truncation = %q, want %q", got, want)
	}
}

// TestTruncatePreviewLongCJKStaysValidUTF8 reproduces the v0.1.0 bug: a
// byte-wise cut at 77 split a multi-byte rune and printed invalid UTF-8.
func TestTruncatePreviewLongCJKStaysValidUTF8(t *testing.T) {
	// "BBBBBBBBBBBBBBBBBBBBBBBBB" is 25 bytes, then each 北京 is 6 bytes:
	// the 77-byte cut lands mid-rune.
	s := `{"location": "` + strings.Repeat("B", 12) + strings.Repeat("北京", 30) + `"}`
	if len(s) <= 80 {
		t.Fatalf("test string must exceed 80 bytes, got %d", len(s))
	}
	got := truncatePreview(s, 80)
	if !utf8.ValidString(got) {
		t.Fatalf("truncated preview is not valid UTF-8: %q", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("truncated preview should end with ..., got %q", got)
	}
	if len(got) > 80 {
		t.Fatalf("truncated preview exceeds 80 bytes: %d", len(got))
	}
}

// TestCheckInvalidJSONReportsSyntaxNotShape pins the v0.3.0 diagnosis fix at
// the CLI boundary: a trace file that is not valid JSON must produce the
// syntax error, not v0.2.0's false "trace has no tool_calls" shape diagnosis.
func TestCheckInvalidJSONReportsSyntaxNotShape(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "truncated.json")
	if err := os.WriteFile(tracePath, []byte(`{"choices":[{"mess`), 0o644); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetArgs([]string{"check", "--trace", tracePath, "--schema", filepath.Join("..", "examples", "tool-schema.json")})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected an error for a truncated (invalid JSON) trace")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("error should name the JSON syntax failure, got %q", err)
	}
	if strings.Contains(err.Error(), "no tool_calls") {
		t.Fatalf("error must not misreport broken syntax as a shape problem, got %q", err)
	}
}

// TestCheckReportValidUTF8OnCJKArgs runs the real check command on a trace
// whose arguments exceed the preview cap with CJK content and asserts the
// whole report is valid UTF-8 (v0.1.0 emitted a lone 0xE4 byte).
func TestCheckReportValidUTF8OnCJKArgs(t *testing.T) {
	trace := `{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"location\": \"` +
		strings.Repeat("B", 12) + strings.Repeat("北京", 30) + `\", \"unit\": \"celsius\"}"}}`
	tracePath := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(tracePath, []byte(trace), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"check", "--trace", tracePath, "--schema", filepath.Join("..", "examples", "tool-schema.json")})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("conforming CJK trace must pass all invariants: %v", err)
	}
	report := out.String()
	if !utf8.ValidString(report) {
		t.Fatalf("check report contains invalid UTF-8:\n%q", report)
	}
	if !strings.Contains(report, "4 pass / 0 fail") {
		t.Fatalf("expected 4 pass / 0 fail for a conforming trace, got:\n%s", report)
	}
}

// twoCallTrace writes a chat-completions response with two tool calls: the
// first conforms to examples/tool-schema.json, the second's arguments are not
// JSON. v0.3.0 scored only the first call and reported 4 pass / 0 fail.
func twoCallTrace(t *testing.T) string {
	t.Helper()
	trace := `{"choices":[{"message":{"role":"assistant","tool_calls":[` +
		`{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Tokyo\"}"}},` +
		`{"id":"call_2","type":"function","function":{"name":"get_weather","arguments":"NOT JSON AT ALL"}}` +
		`]},"finish_reason":"tool_calls"}]}`
	path := filepath.Join(t.TempDir(), "two-calls.json")
	if err := os.WriteFile(path, []byte(trace), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runCheckCommand(t *testing.T, tracePath string) (string, error) {
	t.Helper()
	var out strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"check", "--trace", tracePath, "--schema", filepath.Join("..", "examples", "tool-schema.json")})
	err := rootCmd.Execute()
	return out.String(), err
}

// TestCheckEvaluatesEveryToolCall pins the v0.3.0 false PASS: a response whose
// second tool call violates the contract must fail the gate and be attributed
// to that call, not be silently dropped.
func TestCheckEvaluatesEveryToolCall(t *testing.T) {
	report, err := runCheckCommand(t, twoCallTrace(t))
	if !errors.Is(err, errContractBreach) {
		t.Fatalf("expected errContractBreach for a trace with a broken second call, got %v", err)
	}
	if !strings.Contains(report, "call 1/2") || !strings.Contains(report, "call 2/2") {
		t.Fatalf("report must label both calls, got:\n%s", report)
	}
	if !strings.Contains(report, "4 pass / 0 fail") || !strings.Contains(report, "0 pass / 4 fail") {
		t.Fatalf("report must carry each call's tally, got:\n%s", report)
	}
	if !strings.Contains(report, "1 of 2 tool calls breached a contract") {
		t.Fatalf("report must summarise how many calls breached, got:\n%s", report)
	}
	if !strings.Contains(report, "valid_json       error    ✗ FAIL") {
		t.Fatalf("the second call's valid_json row must show the failure, got:\n%s", report)
	}
}

// TestCheckAllConformingMultiCallPasses keeps the fix from over-reaching: a
// trace whose calls all conform still exits clean.
func TestCheckAllConformingMultiCallPasses(t *testing.T) {
	trace := `{"choices":[{"message":{"role":"assistant","tool_calls":[` +
		`{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Tokyo\"}"}},` +
		`{"id":"call_2","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Osaka\",\"unit\":\"celsius\"}"}}` +
		`]},"finish_reason":"tool_calls"}]}`
	path := filepath.Join(t.TempDir(), "two-ok.json")
	if err := os.WriteFile(path, []byte(trace), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := runCheckCommand(t, path)
	if err != nil {
		t.Fatalf("conforming two-call trace must pass, got %v\n%s", err, report)
	}
	if !strings.Contains(report, "0 of 2 tool calls breached a contract") {
		t.Fatalf("expected a clean breach summary, got:\n%s", report)
	}
}

// TestCheckSingleCallOutputUnchanged guards the shipped demo and README table:
// one call keeps the original layout with no call header.
func TestCheckSingleCallOutputUnchanged(t *testing.T) {
	report, err := runCheckCommand(t, filepath.Join("..", "examples", "sample_deepseek_output.json"))
	if err != nil {
		t.Fatalf("conforming single-call trace must pass: %v", err)
	}
	if strings.Contains(report, "call 1/") || strings.Contains(report, "breached a contract") {
		t.Fatalf("single-call report must not gain multi-call lines, got:\n%s", report)
	}
	if !strings.HasSuffix(report, "4 pass / 0 fail\n") {
		t.Fatalf("single-call report must end with the tally, got:\n%s", report)
	}
}

// TestCheckBreachExitContract pins the exit-code contract Execute() relies on:
// a breach is reported through the returned error, not by exiting mid-command.
func TestCheckBreachExitContract(t *testing.T) {
	report, err := runCheckCommand(t, filepath.Join("..", "examples", "sample_glm_output.json"))
	if !errors.Is(err, errContractBreach) {
		t.Fatalf("expected errContractBreach for the drifting fixture, got %v", err)
	}
	if !strings.Contains(report, "3 pass / 1 fail") {
		t.Fatalf("report must be written before the breach error, got:\n%s", report)
	}
	if !rootCmd.SilenceErrors {
		t.Fatalf("rootCmd.SilenceErrors must stay true so Execute() prints at most one error line")
	}
}

// TestDocsDescribeAllToolCallEvaluation keeps the shipped docs in step with the
// m11 behaviour change: check no longer evaluates only the first tool call.
func TestDocsDescribeAllToolCallEvaluation(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "web", "site.json"),
		filepath.Join("..", "README.md"),
		filepath.Join("..", "README.en.md"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, stale := range []string{
			"only the first tool call",
			"只检查响应中的第一个工具调用",
			"selects the first tool call",
			"选择第一个工具调用",
		} {
			if strings.Contains(text, stale) {
				t.Errorf("%s still describes the single-call behaviour (%q)", path, stale)
			}
		}
	}
}
