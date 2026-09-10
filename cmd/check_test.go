package cmd

import (
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
