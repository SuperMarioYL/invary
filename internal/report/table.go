// Package report renders invariant evaluation results. In m2 it prints the
// cross-model differential matrix with silent-breaker flags; m1 ships the
// package skeleton only (the 'invary check' command formats its own
// single-trace table inline).
package report

import (
	"fmt"

	"github.com/SuperMarioYL/invary/internal/invariant"
)

// RenderDifferential renders the cross-model invariant matrix + silent-breaker
// flags to a printable table. m1 ships this as a stub; the live renderer lands
// in milestone m2 alongside the differential runner.
func RenderDifferential(r invariant.DifferentialReport) (string, error) {
	return "", fmt.Errorf("differential report rendering is not implemented until milestone m2")
}
