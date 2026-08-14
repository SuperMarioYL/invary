// Package diff drives the four providers with one prompt + tool schema and
// reduces their tool-call outputs into a differential report with
// silent-breaker flags. This is the m2 wedge — the table that makes a dev
// star — so m1 ships only the package skeleton.
package diff

import (
	"fmt"

	"github.com/SuperMarioYL/invary/internal/invariant"
)

// Runner orchestrates the differential evaluation: loop the four providers,
// collect each tool-call output, run the m1 invariant evaluator on each, and
// flag the silent breakers (an invariant that passes on >=1 provider and
// fails on >=1 other).
type Runner struct{}

// Run executes the differential evaluation for the given prompt + schema and
// returns the cross-model matrix. m1 ships this as a stub; the live
// implementation (model.Client calls + breach detection) lands in milestone m2.
func (Runner) Run(prompt, schemaPath string) (invariant.DifferentialReport, error) {
	return invariant.DifferentialReport{},
		fmt.Errorf("invary run is not available in this build — milestone m2 wires the live differential run across DeepSeek/Qwen/Kimi/GLM. Use 'invary check --trace <file> --schema <file>' to evaluate a single captured trace")
}
