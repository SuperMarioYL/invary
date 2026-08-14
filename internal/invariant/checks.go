package invariant

import (
	"fmt"
	"sort"
	"strings"
)

// checkValidJSON: the arguments string parses as JSON. This is the syntax gate;
// a trace that is not even valid JSON cannot meaningfully be checked for
// structure, and the structural checks below root-cause to this same failure.
func checkValidJSON(tc ToolCall, parseErr error) (bool, string) {
	if parseErr != nil {
		return false, fmt.Sprintf("arguments is not valid JSON: %v", parseErr)
	}
	return true, ""
}

// checkRequiredFields: every schema-declared required parameter is present in
// the parsed arguments object. Missing keys are listed in declaration order.
func checkRequiredFields(sch Schema, obj map[string]any, parseErr error, kind string) (bool, string) {
	if blocked := blockedEvidence(parseErr, kind); blocked != "" {
		return false, blocked
	}
	var missing []string
	for _, req := range sch.Required {
		if _, ok := obj[req]; !ok {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return false, fmt.Sprintf("missing required field(s): %s", strings.Join(missing, ", "))
	}
	return true, ""
}

// checkArgTypes: each argument present matches its schema-declared type. JSON
// numbers decode to float64, so "number" and "integer" both accept float64 (an
// integer is reported only when a number has a non-zero fractional part). A
// property declared without a type skips type checking (only presence/drift
// apply), mirroring JSON Schema's lenient default.
func checkArgTypes(sch Schema, obj map[string]any, parseErr error, kind string) (bool, string) {
	if blocked := blockedEvidence(parseErr, kind); blocked != "" {
		return false, blocked
	}
	// Iterate argument keys in sorted order for deterministic evidence.
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var mismatches []string
	for _, k := range keys {
		spec, declared := sch.Properties[k]
		if !declared {
			continue // undeclared keys are no_schema_drift's concern, not arg_types'
		}
		if spec.Type == "" {
			continue // no type declared -> not a type constraint
		}
		got, _ := goJSONType(obj[k])
		if !typeMatches(spec.Type, got, obj[k]) {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected %s, got %s", k, spec.Type, got))
		}
	}
	if len(mismatches) > 0 {
		return false, fmt.Sprintf("argument type mismatch: %s", strings.Join(mismatches, "; "))
	}
	return true, ""
}

// checkNoSchemaDrift: no argument key exists that the schema does not declare.
// Extra keys are the quietest form of contract drift (a model invents a field
// the caller never documented), so this is the one Warn-severity invariant.
func checkNoSchemaDrift(sch Schema, obj map[string]any, parseErr error, kind string) (bool, string) {
	if blocked := blockedEvidence(parseErr, kind); blocked != "" {
		return false, blocked
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var extra []string
	for _, k := range keys {
		if _, declared := sch.Properties[k]; !declared {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		return false, fmt.Sprintf("undeclared argument field(s): %s", strings.Join(extra, ", "))
	}
	return true, ""
}

// blockedEvidence reports why a structural check could not run: the arguments
// were not a parseable JSON object. Returning this (instead of silently
// passing) keeps the per-invariant table honest about the root cause.
func blockedEvidence(parseErr error, kind string) string {
	if parseErr != nil {
		return "arguments is not valid JSON (see valid_json)"
	}
	if kind != "object" {
		return fmt.Sprintf("arguments parsed to %s, expected a JSON object (see valid_json)", kind)
	}
	return ""
}

// goJSONType maps a decoded JSON value to its schema type name.
func goJSONType(v any) (string, bool) {
	switch v.(type) {
	case string:
		return "string", true
	case bool:
		return "boolean", true
	case float64:
		return "number", true
	case []any:
		return "array", true
	case map[string]any:
		return "object", true
	case nil:
		return "null", true
	default:
		return fmt.Sprintf("%T", v), false
	}
}

// typeMatches reconciles a JSON-Schema-declared type with the decoded Go value.
func typeMatches(declared, got string, v any) bool {
	switch declared {
	case got:
		return true
	case "integer":
		// JSON has no integer type; accept any number that is whole.
		if got == "number" {
			if f, ok := v.(float64); ok {
				return f == float64(int64(f))
			}
		}
		return false
	case "number":
		return got == "number" // integer (whole number) also accepted
	case "array":
		return got == "array"
	case "object":
		return got == "object"
	case "string":
		return got == "string"
	case "boolean":
		return got == "boolean"
	default:
		// Unknown declared type: do not fail on type (lenient, like JSON Schema).
		return true
	}
}

// (evidence helpers intentionally minimal — see check functions above)
