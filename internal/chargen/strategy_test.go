package chargen_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// schemaPath is the record's published schema, which names the strategies an
// auto record's policy may carry.
const schemaPath = "../../docs/character.schema.json"

// The schema's three strategy enums, under `policy`, name exactly the strategies the code
// takes, in both directions.
//
// The point enum is held through the goldens, because every choice point
// must be reached by one. A strategy has no such chain: one added without a
// golden that uses it would leave the schema refusing records the tool
// writes, with every test green (#177).
func TestTheSchemaNamesEveryStrategy(t *testing.T) {
	t.Parallel()

	text, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("reading the schema: %v", err)
	}

	var document struct {
		Properties struct {
			Policy struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			} `json:"policy"`
		} `json:"properties"`
	}

	err = json.Unmarshal(text, &document)
	if err != nil {
		t.Fatalf("the schema is not JSON: %v", err)
	}

	for field, choices := range map[string]string{
		"career": chargen.CareerChoices(),
		"skills": chargen.SkillsChoices(),
		"muster": chargen.MusterChoices(),
	} {
		inSchema := document.Properties.Policy.Properties[field].Enum
		if len(inSchema) == 0 {
			t.Errorf("the schema's policy.%s has no enum, so this checks nothing", field)
		}

		sameNames(t, strings.Split(choices, ", "), "a --"+field+" strategy",
			inSchema, "in the schema's policy."+field+" enum")
	}
}
