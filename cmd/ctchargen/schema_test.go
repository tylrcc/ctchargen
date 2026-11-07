package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// What the command writes is held to the schema that describes it.
//
// `render`'s schema test validates the goldens and the two documented
// examples, which are what the engine produces. Nothing validated what comes
// out of the CLI, and the two are not the same document: the command stamps
// `build` from debug.ReadBuildInfo and fills `name` from a flag, neither of
// which any golden carries. A guided run and a batch member are further
// paths of their own.
//
// The gap was not theoretical. Until #24 a guided `new --career dawdle`
// was accepted and written into the record, against the enum the schema
// prints - and CI could not see it, because only goldens were validated.
//
// The schema is compiled here rather than shared with `render`'s test: a
// package both could import would have to be a non-test package, which would
// make jsonschema a build dependency of the module instead of a test-only
// one. Two calls to a schema compiler cannot disagree about a schema, so the
// duplication is plumbing rather than a second definition of anything.
const schemaPath = "../../docs/character.schema.json"

func schema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	text, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("reading the schema: %v", err)
	}

	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(text))
	if err != nil {
		t.Fatalf("the schema is not JSON: %v", err)
	}

	compiler := jsonschema.NewCompiler()

	added := compiler.AddResource(schemaPath, doc)
	if added != nil {
		t.Fatalf("adding the schema: %v", added)
	}

	compiled, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("the schema does not compile: %v", err)
	}

	return compiled
}

func mustMatch(t *testing.T, compiled *jsonschema.Schema, what, text string) {
	t.Helper()

	var record any

	err := json.Unmarshal([]byte(text), &record)
	if err != nil {
		t.Fatalf("%s is not JSON: %v", what, err)
	}

	invalid := compiled.Validate(record)
	if invalid != nil {
		t.Errorf("%s does not match the schema:\n%v", what, invalid)
	}
}

// Every shape of record the command writes matches the schema.
func TestWhatTheCommandWritesMatchesTheSchema(t *testing.T) {
	t.Parallel()

	compiled := schema(t)

	for name, tc := range map[string]struct {
		args []string
		in   string
		dice func(*testing.T) source
	}{
		// The automatic path, which is what a batch member and most runs are.
		"an automatic run": {
			args: []string{cmdNew, flagAuto, flagService, navy},
		},
		// A name from the flag, which no golden carries.
		"a named character": {
			args: []string{cmdNew, flagAuto, flagService, other, flagName, "Alexander Jamison"},
		},
		// A death, whose record carries a departure the living do not.
		"a character killed in service": {
			args: []string{cmdNew, flagAuto, flagService, other},
			dice: otherKilled.roller,
		},
		// The guided path, whose choices are recorded as the player's.
		"a guided run": {
			args: []string{cmdNew, flagService, other},
			in:   strings.Repeat("1\n", 300),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var dice source

			if tc.dice != nil {
				dice = tc.dice(t)
			}

			out, _ := mustRun(t, tc.args, strings.NewReader(tc.in), withDice(dice))

			mustMatch(t, compiled, name, out)
		})
	}
}

// Every line of a batch is a record in its own right, and each is validated.
func TestEveryBatchMemberMatchesTheSchema(t *testing.T) {
	t.Parallel()

	compiled := schema(t)

	out, _ := mustRun(t, []string{cmdBatch, flagCount, "5", flagAuto, flagService, merchants}, nil)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		t.Fatalf("the batch wrote %d lines, want 5", len(lines))
	}

	for i, line := range lines {
		mustMatch(t, compiled, "batch member "+strconv.Itoa(i), line)
	}
}

// This build writes shape 1.
//
// The schema said `"const": 1` until #113, and that one word was doing two
// jobs: describing what a record may carry, and pinning what this build
// writes. The freeze needs the first of those loose - a reader holding the
// v1.0.0 schema must accept a shape number from a later build, or opening the
// root object delivers nothing, because the record would be refused before its
// unknown field was ever reached. So the schema says `"minimum": 1` now, and
// the pinning that used to be a side effect of the const is this test.
func TestThisBuildWritesShapeOne(t *testing.T) {
	t.Parallel()

	out, _ := mustRun(t, []string{cmdNew, flagAuto, flagService, navy}, nil)

	// Into a field of the right type rather than a map, whose numbers are
	// float64 and would compare against an untyped 1 for the wrong reason.
	var written struct {
		Record int `json:"record"`
	}

	err := json.Unmarshal([]byte(out), &written)
	if err != nil {
		t.Fatalf("what the command wrote is not JSON: %v", err)
	}

	if written.Record != 1 {
		t.Errorf("this build writes record shape %d; it writes 1 until v1.0.0 (#96)", written.Record)
	}
}
