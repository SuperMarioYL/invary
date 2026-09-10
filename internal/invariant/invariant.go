// Package invariant defines Invary's core primitive: the per-model invariant
// breach, produced by a differential oracle whose ground truth is the user's
// own designated contracts.
//
// An Invariant is a machine-checkable contract over a model's tool-call output
// (e.g. "arguments parses as JSON", "all schema-required fields are present").
// Eval runs a set of invariants against one captured tool-call trace and
// reports per-invariant pass/fail with the minimal failing evidence.
//
// This package is intentionally free of any network call: it operates on saved
// traces so the invariant-evaluation primitive can be developed, tested and
// demonstrated in isolation from provider latency or rate limits.
package invariant

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Severity classifies how hard an invariant breach is.
const (
	SeverityError = "error" // a hard contract violation
	SeverityWarn  = "warn"  // a soft, advisory contract deviation
)

// Identifier is the canonical name of a built-in invariant.
const (
	IDValidJSON      = "valid_json"
	IDRequiredFields = "required_fields"
	IDArgTypes       = "arg_types"
	IDNoSchemaDrift  = "no_schema_drift"
)

// Invariant is one user-designated contract check.
type Invariant struct {
	ID       string         `json:"id"       yaml:"id"`
	Severity string         `json:"severity" yaml:"severity"` // "error" | "warn"
	Spec     map[string]any `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// Result is the outcome of evaluating one Invariant against one tool-call trace.
type Result struct {
	Invariant string `json:"invariant"`
	Severity  string `json:"severity"`
	Pass      bool   `json:"pass"`
	// Evidence is the minimal failing trace (empty when Pass is true).
	Evidence string `json:"evidence,omitempty"`
}

// Breach is a per-provider invariant failure. It is the first-class object the
// differential oracle emits: the killer flag is Silent, true when at least one
// *other* provider passed the same invariant — i.e. the model broke a contract
// its peers kept.
type Breach struct {
	Provider  string `json:"provider"`
	Invariant string `json:"invariant"`
	Evidence  string `json:"evidence"`
	Silent    bool   `json:"silent"`
}

// DifferentialReport is the cross-model matrix (milestone m2). It is defined
// here because it is the canonical shape of the core primitive; the m2 diff
// runner populates it.
type DifferentialReport struct {
	Invariants []Invariant                `json:"invariants"`
	Results    map[string]map[string]bool `json:"results"` // provider -> invariantID -> pass
	Breaches   []Breach                   `json:"breaches"`
}

// ToolCall is the minimal tool-call shape extracted from a provider trace. The
// Arguments field is the raw, provider-emitted string exactly as it appeared
// in the chat-completions response (before any normalization).
type ToolCall struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"` // typically "function"
	Name      string `json:"name"`           // function.name
	Arguments string `json:"arguments"`      // function.arguments (raw JSON string)
}

// ParamSpec is the declared contract for one schema parameter.
type ParamSpec struct {
	Type string `json:"type"` // string|number|integer|boolean|array|object
}

// Schema is the minimal function schema the invariant checks reason about.
type Schema struct {
	Name       string               `json:"name"`
	Properties map[string]ParamSpec `json:"properties"`
	Required   []string             `json:"required"`
}

// Builtins returns the four v0.1 invariants in evaluation order. The set is
// hand-rolled (no schema-validator dependency) and is the ground-truth spec a
// trace is checked against.
func Builtins() []Invariant {
	return []Invariant{
		{ID: IDValidJSON, Severity: SeverityError},
		{ID: IDRequiredFields, Severity: SeverityError},
		{ID: IDArgTypes, Severity: SeverityError},
		{ID: IDNoSchemaDrift, Severity: SeverityWarn},
	}
}

// Eval runs the built-in invariant set against a single captured tool call and
// returns one Result per invariant, in Builtins() order. Each check is
// self-contained: a failure in one (e.g. arguments not valid JSON) does not
// short-circuit the others, so a dev sees every contract that the trace
// violates rather than only the first.
func Eval(tc ToolCall, sch Schema) []Result {
	ins := Builtins()
	results := make([]Result, len(ins))
	argsObj, argsErr, argsKind := parseArgumentsObject(tc.Arguments)
	for i, in := range ins {
		r := Result{Invariant: in.ID, Severity: in.Severity}
		switch in.ID {
		case IDValidJSON:
			r.Pass, r.Evidence = checkValidJSON(tc, argsErr)
		case IDRequiredFields:
			r.Pass, r.Evidence = checkRequiredFields(sch, argsObj, argsErr, argsKind)
		case IDArgTypes:
			r.Pass, r.Evidence = checkArgTypes(sch, argsObj, argsErr, argsKind)
		case IDNoSchemaDrift:
			r.Pass, r.Evidence = checkNoSchemaDrift(sch, argsObj, argsErr, argsKind)
		}
		if r.Pass {
			r.Evidence = "" // keep pass results clean
		}
		results[i] = r
	}
	return results
}

// Summary returns a one-line tally of an Eval result set, e.g. "3 pass / 1 fail".
func Summary(results []Result) (pass, fail int) {
	for _, r := range results {
		if r.Pass {
			pass++
		} else {
			fail++
		}
	}
	return pass, fail
}

// parseArgumentsObject parses the raw arguments string into a JSON object map.
// It returns the object (or nil), a parse error (or nil), and the JSON kind
// ("object", "string", "number", ...) so each structural check can report a
// precise, root-caused failure instead of silently no-op-ing.
func parseArgumentsObject(raw string) (obj map[string]any, parseErr error, kind string) {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err, ""
	}
	switch t := v.(type) {
	case map[string]any:
		return t, nil, "object"
	case []any:
		return nil, nil, "array"
	case string:
		return nil, nil, "string"
	case bool:
		return nil, nil, "boolean"
	case float64:
		return nil, nil, "number"
	case nil:
		return nil, nil, "null"
	default:
		return nil, nil, fmt.Sprintf("%T", v)
	}
}

// ParseTrace extracts tool calls from a captured provider response. It accepts
// the shapes a dev is likely to save:
//   - a full chat-completions response (choices[].message.tool_calls),
//   - a bare assistant message ({role, tool_calls}),
//   - a bare tool_call object ({id, type, function:{name, arguments}}),
//   - a minimal {arguments} object.
//
// Tool calls are returned in document order.
func ParseTrace(data []byte) ([]ToolCall, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("trace is empty")
	}

	// Full chat-completions response.
	var resp struct {
		Choices []struct {
			Message struct {
				Role      string        `json:"role"`
				ToolCalls []rawToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &resp); err == nil && len(resp.Choices) > 0 {
		var out []ToolCall
		for _, c := range resp.Choices {
			for _, tc := range c.Message.ToolCalls {
				out = append(out, tc.toToolCall())
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}

	// Bare assistant message with tool_calls.
	var msg struct {
		ToolCalls []rawToolCall `json:"tool_calls"`
	}
	if err := json.Unmarshal(data, &msg); err == nil && len(msg.ToolCalls) > 0 {
		out := make([]ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			out[i] = tc.toToolCall()
		}
		return out, nil
	}

	// Bare tool_call object.
	var single rawToolCall
	if err := json.Unmarshal(data, &single); err == nil && single.Function.Arguments != nil {
		return []ToolCall{single.toToolCall()}, nil
	}

	// Minimal {arguments} object.
	var minimal struct {
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(data, &minimal); err == nil && len(minimal.Arguments) > 0 {
		return []ToolCall{{Arguments: unwrapArguments(minimal.Arguments)}}, nil
	}

	return nil, fmt.Errorf("trace has no tool_calls: expected a chat-completions response, a tool_calls message, or a tool_call object")
}

// rawToolCall is the wire shape of an OpenAI-compatible tool_call.
type rawToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

func (r rawToolCall) toToolCall() ToolCall {
	return ToolCall{
		ID:        r.ID,
		Type:      r.Type,
		Name:      r.Function.Name,
		Arguments: unwrapArguments(r.Function.Arguments),
	}
}

// unwrapArguments normalizes the raw wire value of a tool call's arguments.
// The OpenAI shape encodes arguments as a JSON string whose VALUE is itself
// the args JSON ("arguments": "{\"location\":\"Tokyo\"}"); unwrap one layer so
// every accepted trace shape reasons about the model's actual output rather
// than a doubly-encoded string. If unwrap fails the raw bytes are kept, and
// every check will root-cause to valid_json — exactly the silent breaker case.
func unwrapArguments(raw json.RawMessage) string {
	var inner string
	if err := json.Unmarshal(raw, &inner); err == nil {
		return inner
	}
	return string(raw)
}

// ParseSchema parses an OpenAI-compatible function/tool schema into the minimal
// Schema the checks reason about. The "type":"function" wrapper is optional.
func ParseSchema(data []byte) (Schema, error) {
	if len(data) == 0 {
		return Schema{}, fmt.Errorf("schema is empty")
	}

	// Accept {"type":"function","function":{...}} or a bare function object.
	var wrapped struct {
		Function json.RawMessage `json:"function"`
	}
	var fnBytes json.RawMessage = data
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Function) > 0 {
		fnBytes = wrapped.Function
	}

	var fn struct {
		Name       string `json:"name"`
		Parameters struct {
			Properties map[string]json.RawMessage `json:"properties"`
			Required   []string                   `json:"required"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(fnBytes, &fn); err != nil {
		return Schema{}, fmt.Errorf("parse schema: %w", err)
	}

	sch := Schema{
		Name:       fn.Name,
		Required:   fn.Parameters.Required,
		Properties: make(map[string]ParamSpec, len(fn.Parameters.Properties)),
	}
	// Sort property names for deterministic iteration in the checks.
	for _, name := range sortedKeys(fn.Parameters.Properties) {
		raw := fn.Parameters.Properties[name]
		var ps ParamSpec
		if err := json.Unmarshal(raw, &ps); err != nil {
			// Property without an explicit type is still a declared property
			// (it just won't constrain arg_types).
			ps = ParamSpec{Type: ""}
		}
		sch.Properties[name] = ps
	}
	return sch, nil
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
