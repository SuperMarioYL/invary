package invariant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: build a Schema the way the checks expect it.
func schemaFor(properties map[string]string, required ...string) Schema {
	props := make(map[string]ParamSpec, len(properties))
	for k, t := range properties {
		props[k] = ParamSpec{Type: t}
	}
	return Schema{Name: "get_weather", Properties: props, Required: required}
}

// helper: run the named invariant's check and report (pass, evidence).
func checkByID(id string, tc ToolCall, sch Schema) (bool, string) {
	argsObj, argsErr, kind := parseArgumentsObject(tc.Arguments)
	switch id {
	case IDValidJSON:
		return checkValidJSON(tc, argsErr)
	case IDRequiredFields:
		return checkRequiredFields(sch, argsObj, argsErr, kind)
	case IDArgTypes:
		return checkArgTypes(sch, argsObj, argsErr, kind)
	case IDNoSchemaDrift:
		return checkNoSchemaDrift(sch, argsObj, argsErr, kind)
	}
	return false, "unknown invariant"
}

func TestValidJSON(t *testing.T) {
	t.Run("object parses", func(t *testing.T) {
		pass, ev := checkByID(IDValidJSON, ToolCall{Arguments: `{"location":"Tokyo"}`}, Schema{})
		if !pass || ev != "" {
			t.Fatalf("expected pass, got pass=%v ev=%q", pass, ev)
		}
	})
	t.Run("single quoted pseudo json fails", func(t *testing.T) {
		pass, ev := checkByID(IDValidJSON, ToolCall{Arguments: `{'location':'Tokyo'}`}, Schema{})
		if pass {
			t.Fatalf("expected fail for invalid JSON, got pass; ev=%q", ev)
		}
		if ev == "" {
			t.Fatalf("expected non-empty evidence")
		}
	})
	t.Run("bareword fails", func(t *testing.T) {
		pass, _ := checkByID(IDValidJSON, ToolCall{Arguments: `Tokyo`}, Schema{})
		if pass {
			t.Fatalf("expected fail for bareword")
		}
	})
}

func TestRequiredFields(t *testing.T) {
	sch := schemaFor(map[string]string{"location": "string", "unit": "string"}, "location", "unit")
	t.Run("all required present", func(t *testing.T) {
		pass, ev := checkByID(IDRequiredFields, ToolCall{Arguments: `{"location":"Tokyo","unit":"celsius"}`}, sch)
		if !pass || ev != "" {
			t.Fatalf("expected pass, got pass=%v ev=%q", pass, ev)
		}
	})
	t.Run("missing one required", func(t *testing.T) {
		pass, ev := checkByID(IDRequiredFields, ToolCall{Arguments: `{"location":"Tokyo"}`}, sch)
		if pass {
			t.Fatalf("expected fail when unit missing")
		}
		if want := "missing required field(s): unit"; ev != want {
			t.Fatalf("evidence = %q, want %q", ev, want)
		}
	})
	t.Run("no required declared passes on empty", func(t *testing.T) {
		sch := schemaFor(map[string]string{"location": "string"})
		pass, _ := checkByID(IDRequiredFields, ToolCall{Arguments: `{}`}, sch)
		if !pass {
			t.Fatalf("expected pass when no required fields")
		}
	})
}

func TestArgTypes(t *testing.T) {
	sch := schemaFor(map[string]string{
		"location": "string",
		"count":    "integer",
		"temp":     "number",
		"active":   "boolean",
		"tags":     "array",
		"meta":     "object",
	}, "location")
	t.Run("all types match", func(t *testing.T) {
		args := `{"location":"Tokyo","count":3,"temp":36.5,"active":true,"tags":["a"],"meta":{"k":"v"}}`
		pass, ev := checkByID(IDArgTypes, ToolCall{Arguments: args}, sch)
		if !pass || ev != "" {
			t.Fatalf("expected pass, got pass=%v ev=%q", pass, ev)
		}
	})
	t.Run("integer accepts whole number", func(t *testing.T) {
		pass, _ := checkByID(IDArgTypes, ToolCall{Arguments: `{"location":"x","count":3}`}, sch)
		if !pass {
			t.Fatalf("integer should accept whole number 3")
		}
	})
	t.Run("integer rejects fractional", func(t *testing.T) {
		pass, ev := checkByID(IDArgTypes, ToolCall{Arguments: `{"location":"x","count":3.5}`}, sch)
		if pass {
			t.Fatalf("integer should reject 3.5")
		}
		if ev == "" {
			t.Fatalf("expected evidence for type mismatch")
		}
	})
	t.Run("string vs number mismatch", func(t *testing.T) {
		pass, ev := checkByID(IDArgTypes, ToolCall{Arguments: `{"location":42}`}, sch)
		if pass {
			t.Fatalf("location declared string, got number -> mismatch")
		}
		if want := "argument type mismatch: location: expected string, got number"; ev != want {
			t.Fatalf("evidence = %q, want %q", ev, want)
		}
	})
	t.Run("undeclared key ignored by arg_types", func(t *testing.T) {
		// timezone is not in schema; arg_types must not flag it (no_schema_drift does).
		pass, _ := checkByID(IDArgTypes, ToolCall{Arguments: `{"location":"Tokyo","timezone":"Asia/Tokyo"}`}, sch)
		if !pass {
			t.Fatalf("undeclared key should be ignored by arg_types")
		}
	})
}

func TestNoSchemaDrift(t *testing.T) {
	sch := schemaFor(map[string]string{"location": "string", "unit": "string"}, "location")
	t.Run("declared keys only", func(t *testing.T) {
		pass, ev := checkByID(IDNoSchemaDrift, ToolCall{Arguments: `{"location":"Tokyo","unit":"celsius"}`}, sch)
		if !pass || ev != "" {
			t.Fatalf("expected pass, got pass=%v ev=%q", pass, ev)
		}
	})
	t.Run("extra undeclared key fails", func(t *testing.T) {
		pass, ev := checkByID(IDNoSchemaDrift, ToolCall{Arguments: `{"location":"Tokyo","timezone":"Asia/Tokyo"}`}, sch)
		if pass {
			t.Fatalf("expected drift failure for timezone")
		}
		if want := "undeclared argument field(s): timezone"; ev != want {
			t.Fatalf("evidence = %q, want %q", ev, want)
		}
	})
}

func TestEvalConforming(t *testing.T) {
	sch := schemaFor(map[string]string{"location": "string", "unit": "string"}, "location")
	tc := ToolCall{Name: "get_weather", Arguments: `{"location":"Tokyo","unit":"celsius"}`}
	results := Eval(tc, sch)
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Pass {
			t.Errorf("expected %s to pass, got ev=%q", r.Invariant, r.Evidence)
		}
		if r.Pass && r.Evidence != "" {
			t.Errorf("pass result %s should have empty evidence, got %q", r.Invariant, r.Evidence)
		}
	}
	pass, fail := Summary(results)
	if pass != 4 || fail != 0 {
		t.Fatalf("summary = %d pass / %d fail, want 4/0", pass, fail)
	}
}

func TestEvalBadJSONCascades(t *testing.T) {
	sch := schemaFor(map[string]string{"location": "string"}, "location")
	tc := ToolCall{Arguments: `{'location':'Tokyo'}`}
	results := Eval(tc, sch)
	rByID := map[string]Result{}
	for _, r := range results {
		rByID[r.Invariant] = r
	}
	// valid_json is the root cause; structural checks root-cause to it.
	if rByID[IDValidJSON].Pass {
		t.Fatalf("valid_json should fail on single-quoted JSON")
	}
	for _, id := range []string{IDRequiredFields, IDArgTypes, IDNoSchemaDrift} {
		if rByID[id].Pass {
			t.Errorf("%s should be blocked (not pass) when arguments is invalid JSON", id)
		}
		if rByID[id].Evidence == "" {
			t.Errorf("%s should carry blocking evidence", id)
		}
	}
	pass, fail := Summary(results)
	if pass != 0 || fail != 4 {
		t.Fatalf("summary = %d pass / %d fail, want 0/4", pass, fail)
	}
}

func TestEvalSchemaDriftOnly(t *testing.T) {
	sch := schemaFor(map[string]string{"location": "string", "unit": "string"}, "location")
	tc := ToolCall{Arguments: `{"location":"Tokyo","timezone":"Asia/Tokyo"}`}
	results := Eval(tc, sch)
	rByID := map[string]Result{}
	for _, r := range results {
		rByID[r.Invariant] = r
	}
	for _, id := range []string{IDValidJSON, IDRequiredFields, IDArgTypes} {
		if !rByID[id].Pass {
			t.Errorf("%s should pass on a valid object with an extra field", id)
		}
	}
	if rByID[IDNoSchemaDrift].Pass {
		t.Errorf("no_schema_drift should fail on undeclared timezone")
	}
}

func TestParseSchemaWrapped(t *testing.T) {
	data := []byte(`{
		"type":"function",
		"function":{
			"name":"get_weather",
			"parameters":{
				"type":"object",
				"properties":{"location":{"type":"string"},"unit":{"type":"string","enum":["celsius","fahrenheit"]}},
				"required":["location"]
			}
		}
	}`)
	sch, err := ParseSchema(data)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}
	if sch.Name != "get_weather" {
		t.Errorf("name = %q", sch.Name)
	}
	if len(sch.Required) != 1 || sch.Required[0] != "location" {
		t.Errorf("required = %v", sch.Required)
	}
	if sch.Properties["location"].Type != "string" {
		t.Errorf("location type = %q", sch.Properties["location"].Type)
	}
}

func TestParseSchemaBareFunction(t *testing.T) {
	data := []byte(`{"name":"f","parameters":{"properties":{"a":{"type":"integer"}},"required":["a"]}}`)
	sch, err := ParseSchema(data)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}
	if sch.Properties["a"].Type != "integer" {
		t.Errorf("a type = %q", sch.Properties["a"].Type)
	}
}

func TestParseTraceShapes(t *testing.T) {
	t.Run("full chat completion", func(t *testing.T) {
		data := []byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Tokyo\"}"}}]}}]}`)
		calls, err := ParseTrace(data)
		if err != nil {
			t.Fatalf("ParseTrace: %v", err)
		}
		if len(calls) != 1 || calls[0].Name != "get_weather" {
			t.Fatalf("calls = %+v", calls)
		}
		// arguments must be unwrapped one layer to the inner JSON.
		if calls[0].Arguments != `{"location":"Tokyo"}` {
			t.Errorf("arguments = %q, want unwrapped inner JSON", calls[0].Arguments)
		}
	})
	t.Run("bare tool call object", func(t *testing.T) {
		data := []byte(`{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Tokyo\"}"}}`)
		calls, err := ParseTrace(data)
		if err != nil {
			t.Fatalf("ParseTrace: %v", err)
		}
		if len(calls) != 1 || calls[0].Name != "get_weather" {
			t.Fatalf("calls = %+v", calls)
		}
	})
	t.Run("minimal arguments object", func(t *testing.T) {
		data := []byte(`{"arguments":{"location":"Tokyo"}}`)
		calls, err := ParseTrace(data)
		if err != nil {
			t.Fatalf("ParseTrace: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("calls = %+v", calls)
		}
		// arguments is a raw JSON object here (not a doubly-encoded string);
		// the unwrap falls back to the raw bytes, which still parse as an object.
		if calls[0].Arguments == "" {
			t.Errorf("arguments should not be empty")
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(calls[0].Arguments), &obj); err != nil {
			t.Errorf("minimal arguments should still parse: %v (got %q)", err, calls[0].Arguments)
		}
	})
	t.Run("minimal wire-string arguments unwrap like tool_calls", func(t *testing.T) {
		// The OpenAI wire shape encodes arguments as a JSON string whose value
		// is the args JSON; the minimal shape must normalize identically to
		// the tool_calls shape. v0.1.0 kept the quoted raw bytes and the
		// structural checks then false-failed with "parsed to string".
		data := []byte(`{"arguments":"{\"location\":\"Tokyo\"}"}`)
		calls, err := ParseTrace(data)
		if err != nil {
			t.Fatalf("ParseTrace: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("calls = %+v", calls)
		}
		if calls[0].Arguments != `{"location":"Tokyo"}` {
			t.Fatalf("arguments = %q, want unwrapped inner JSON {\"location\":\"Tokyo\"}", calls[0].Arguments)
		}
	})
	t.Run("empty trace errors", func(t *testing.T) {
		if _, err := ParseTrace([]byte{}); err == nil {
			t.Fatalf("expected error on empty trace")
		}
	})
	t.Run("no tool calls errors", func(t *testing.T) {
		if _, err := ParseTrace([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)); err == nil {
			t.Fatalf("expected error when no tool_calls present")
		}
	})
}

// TestEvalMinimalStringArgumentsConforms pins the v0.1.0 false-breach fix: a
// conforming arguments set saved in the minimal wire-string shape evaluated to
// 1 pass / 3 fail; it must evaluate identically to any other accepted shape.
func TestEvalMinimalStringArgumentsConforms(t *testing.T) {
	sch := schemaFor(map[string]string{"location": "string", "unit": "string"}, "location", "unit")
	data := []byte(`{"arguments":"{\"location\":\"Tokyo\",\"unit\":\"celsius\"}"}`)
	calls, err := ParseTrace(data)
	if err != nil {
		t.Fatalf("ParseTrace: %v", err)
	}
	results := Eval(calls[0], sch)
	pass, fail := Summary(results)
	if pass != 4 || fail != 0 {
		var b strings.Builder
		for _, r := range results {
			fmt.Fprintf(&b, "%s pass=%v ev=%q; ", r.Invariant, r.Pass, r.Evidence)
		}
		t.Fatalf("summary = %d pass / %d fail, want 4/0 (results: %s)", pass, fail, b.String())
	}
}

// TestExampleFiles exercises the shipped examples end-to-end through the same
// parse + Eval path the CLI uses, pinning the differential story to reality.
func TestExampleFiles(t *testing.T) {
	root := exampleRoot(t)
	schemaBytes, err := os.ReadFile(filepath.Join(root, "examples", "tool-schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	sch, err := ParseSchema(schemaBytes)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}

	cases := []struct {
		trace    string
		wantPass int
		wantFail int
	}{
		{"sample_deepseek_output.json", 4, 0}, // conforming baseline
		{"sample_qwen_output.json", 4, 0},     // conforming (required-only)
		{"sample_kimi_output.json", 0, 4},     // invalid JSON -> all blocked
		{"sample_glm_output.json", 3, 1},      // schema drift -> only drift fails
	}
	for _, c := range cases {
		t.Run(c.trace, func(t *testing.T) {
			tb, err := os.ReadFile(filepath.Join(root, "examples", c.trace))
			if err != nil {
				t.Fatalf("read trace: %v", err)
			}
			calls, err := ParseTrace(tb)
			if err != nil {
				t.Fatalf("ParseTrace: %v", err)
			}
			if len(calls) != 1 {
				t.Fatalf("expected 1 tool call, got %d", len(calls))
			}
			results := Eval(calls[0], sch)
			pass, fail := Summary(results)
			if pass != c.wantPass || fail != c.wantFail {
				t.Fatalf("%s: summary = %d pass / %d fail, want %d/%d\nresults: %+v",
					c.trace, pass, fail, c.wantPass, c.wantFail, results)
			}
		})
	}
}

// exampleRoot locates the repo examples/ dir relative to the test file.
func exampleRoot(t *testing.T) string {
	t.Helper()
	// test binary runs from the package dir; walk up to the repo root.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "examples", "tool-schema.json")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate examples/ relative to test")
	return ""
}
