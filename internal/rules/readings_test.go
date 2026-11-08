package rules //nolint:testpackage // the lift is tested on mutated text, which only the unexported lift takes

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

func committedReadings(t *testing.T) string {
	t.Helper()

	text, err := files.ReadFile(readingsFile)
	if err != nil {
		t.Fatalf("reading %s: %v", readingsFile, err)
	}

	return string(text)
}

// Every erratum has its reading, in its place, and a reading carries the
// document's own prose. The quotes below are typed from the document, not
// taken from the values, so a field left empty or swapped with its neighbour
// fails here.
func TestTheReadings(t *testing.T) {
	t.Parallel()

	for _, e := range traveller.Errata {
		t.Run(e.String(), func(t *testing.T) {
			t.Parallel()

			r := ReadingOf(e)
			if r.ID != e {
				t.Errorf("declared as %v", r.ID)
			}

			for field, text := range map[string]string{
				"title": r.Title, "pages": r.Pages, "what the page says": r.PageSays,
				"gap": r.Gap, "reading": r.Reading, "why": r.Why, "stamped on": r.StampedOn,
			} {
				if strings.TrimSpace(text) == "" {
					t.Errorf("%s is empty", field)
				}
			}
		})
	}

	e001 := ReadingOf(traveller.E001)

	for field, tc := range map[string]struct{ got, prefix, suffix string }{
		"title":   {e001.Title, "The draft is _may_", "not _must_"},
		"pages":   {e001.Pages, "Book 1 p. 5", "Draft paragraphs."},
		"says":    {e001.PageSays, "Enlistment: \"Only one", "submit\nto the draft.\""},
		"gap":     {e001.Gap, "The same page", "has to lose."},
		"reading": {e001.Reading, "_May._", "benefits."},
		"why":     {e001.Why, "The Enlistment paragraph", "not\nrequired)."},
		"stamped": {e001.StampedOn, "Records where the enlistment throw", "and those that declined."},
	} {
		if !strings.HasPrefix(tc.got, tc.prefix) || !strings.HasSuffix(tc.got, tc.suffix) {
			t.Errorf("E001 %s is %q, want it to run from %q to %q", field, tc.got, tc.prefix, tc.suffix)
		}
	}

	// A field runs across paragraphs to the next label, and the entry ends at
	// its rule: E015's rationale has a second paragraph, and its stamped-on
	// prose a trailing one, and neither carries the separator.
	e015 := ReadingOf(traveller.E015)
	if !strings.HasSuffix(e015.Why, "where the exposition and the example disagree, the exposition governs.") {
		t.Errorf("E015 why lost its second paragraph: %q", e015.Why)
	}

	if !strings.HasSuffix(e015.StampedOn, "a **Low** Passage where p. 25 claims a\nMiddle one.") {
		t.Errorf("E015 stamped on lost its last paragraph or kept the rule: %q", e015.StampedOn)
	}

	// P. 23's box prints "Page." for an entry with one page.
	if got := ReadingOf(traveller.E005).Pages; !strings.HasPrefix(got, "Book 1 p. 23") {
		t.Errorf("E005 pages is %q", got)
	}
}

// Each Erratum constant's comment is its id and its entry's title, as the
// heading prints it, closed with the period a doc comment ends in: "E001 —
// The draft is _may_, not _must_." and nothing more, above the constant or
// beside it. A paraphrase of the reading beside the constant was a second
// statement of it that nothing held, and E004's drifted from the document
// (#233, #234). The title is compared verbatim, Markdown emphasis included,
// so the comment is the heading and not a rendering of it.
func TestEachErratumIsCommentedWithItsTitle(t *testing.T) {
	t.Parallel()

	const source = "../traveller/erratum.go"

	file, err := parser.ParseFile(token.NewFileSet(), source, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing %s: %v", source, err)
	}

	commented := make(map[string]string)

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range value.Names {
				commented[name.Name] = strings.Join(strings.Fields(value.Doc.Text()+" "+value.Comment.Text()), " ")
			}
		}
	}

	for _, e := range traveller.Errata {
		want := e.String() + " — " + ReadingOf(e).Title + "."

		got, ok := commented[e.String()]

		switch {
		case !ok:
			t.Errorf("%v: no constant of that name in %s", e, source)
		case got != want:
			t.Errorf("%v is commented %q, and its entry is titled %q", e, got, want)
		}
	}
}

// The readings are the readings document, field by field and byte for byte as
// its parser lifts it, while both exist: the values were generated once from the document, and
// this is what says the generation lost nothing, not a backtick, not an
// italic, not a line break.
func TestTheReadingsAreTheDocument(t *testing.T) {
	t.Parallel()

	lifted, err := parseReadings(committedReadings(t))
	if err != nil {
		t.Fatalf("lifting %s: %v", readingsFile, err)
	}

	for i, declared := range Readings() {
		document := lifted[i]

		for field, pair := range map[string][2]string{
			"id":    {declared.ID.String(), document.ID.String()},
			"title": {declared.Title, document.Title}, "pages": {declared.Pages, document.Pages},
			"what the page says": {declared.PageSays, document.PageSays}, "gap": {declared.Gap, document.Gap},
			"reading": {declared.Reading, document.Reading}, "why": {declared.Why, document.Why},
			"stamped on": {declared.StampedOn, document.StampedOn},
		} {
			if pair[0] != pair[1] {
				t.Errorf("%v %s is declared\n%q\nand %s has\n%q", document.ID, field, pair[0], readingsFile, pair[1])
			}
		}
	}
}

// A document that will not lift fails, and says which reading.
//
// Each case starts from the committed document and breaks one thing, so
// what is under test is the guard and not a fixture that resembles it.
func TestMalformedReadings(t *testing.T) {
	t.Parallel()

	committed := committedReadings(t)

	for name, tc := range readingBreakages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := parseReadings(tc.breaks(committed))

			switch {
			case err == nil:
				t.Errorf("lifted without complaint")
			case !strings.Contains(err.Error(), tc.mentions):
				t.Errorf("error %q does not name %s", err, tc.mentions)
			case !errors.Is(err, errMalformed):
				t.Errorf("error %q does not wrap errMalformed", err)
			}
		})
	}
}

// lastEntry is E015's entry, the last before the determinate findings.
func lastEntry(s string) string {
	return s[strings.Index(s, "### E015 —"):strings.Index(s, "## Checked and found determinate")]
}

// cut removes the document from one marker up to another.
func cut(from, to string) func(string) string {
	return func(s string) string {
		start := strings.Index(s, from)
		end := strings.Index(s, to)

		return s[:start] + s[end:]
	}
}

// readingBreakages is each way to break the committed document, and the id
// the failure must name.
var readingBreakages = map[string]struct {
	breaks   func(string) string
	mentions string
}{
	"an entry removed by hand": {
		cut("### E013 —", "### E014 —"),
		"E013",
	},
	"an id misspelled": {
		func(s string) string { return strings.Replace(s, "### E007 —", "### E070 —", 1) },
		"E070",
	},
	"a misspelling leaves its constant without an entry": {
		func(s string) string { return strings.Replace(s, "### E007 —", "### E070 —", 1) },
		"E007",
	},
	"an entry with no constant": {
		func(s string) string {
			return strings.Replace(s, "## Checked and found determinate",
				strings.Replace(lastEntry(s), "### E015 —", "### E016 —", 1)+"## Checked and found determinate", 1)
		},
		"E016",
	},
	"an entry twice": {
		func(s string) string {
			return strings.Replace(s, "## Checked and found determinate",
				lastEntry(s)+"## Checked and found determinate", 1)
		},
		"E015",
	},
	"a heading not in the entry form": {
		func(s string) string { return strings.Replace(s, "### E004 —", "### E004:", 1) },
		"E004",
	},
	"a label missing": {
		func(s string) string {
			i := strings.Index(s, "### E009 —")

			return s[:i] + strings.Replace(s[i:], "**The gap.**", "The gap.", 1)
		},
		"E009",
	},
	"labels out of order": {
		func(s string) string {
			i := strings.Index(s, "### E002 —")

			return s[:i] + strings.Replace(strings.Replace(s[i:], "**Why.**", "**WHY**", 1),
				"**Reading.**", "**Why.**", 1)
		},
		"E002",
	},
	"a label left empty": {
		func(s string) string {
			i := strings.Index(s, "### E010 —")
			j := i + strings.Index(s[i:], "**Stamped on.**")
			k := j + strings.Index(s[j:], "\n---")

			return s[:j] + "**Stamped on.**\n" + s[k:]
		},
		"E010",
	},
	"prose between the heading and the first label": {
		func(s string) string {
			return strings.Replace(s, "### E006 — Where the aging round sits within the term\n",
				"### E006 — Where the aging round sits within the term\n\nA stray line.\n", 1)
		},
		"E006",
	},
	"no entries at all": {
		func(s string) string {
			before, _, _ := strings.Cut(s, "### E001 —")

			return before
		},
		"E001",
	},
}
