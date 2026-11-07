// Package coverage generates docs/COVERAGE.md from markers beside the code it
// maps, so the map from each rule to its page, its implementation and its test
// is one copy of what the markers say rather than a second one typed by hand.
//
// The package has no non-test code.
//
// # Markers
//
// An implementation carries one line per rule it implements, naming the rule,
// its printed page and the document section the rule is listed under:
//
//	// Implements: Six 2D rolls, in order [1:4] in Characteristics and the record.
//
// A test that holds a rule carries one line naming it:
//
//	// Holds: Six 2D rolls, in order.
//
// The bracket is the rule's authority, from a closed set: a page cite, which
// opens with book:page and may name readings, "1:5; 3:22 (E011)"; a reading
// alone, "(E011)", that ERRATA.md has a heading for; "ADR 0001", naming a file
// in docs/adr; or a "## " heading or FRn requirement of docs/PRD.md, which
// governs nothing and keeps its labels as names. The PRD is never edited, so a
// label withdrawn from it is named by the ADR that withdrew it, on a line
// "Withdraws from `docs/PRD.md`: Determinism.", and is refused as a cite. The
// set is read from those documents, and a cite outside it fails the
// generator, naming the rule.
//
// A marker runs on to the following comment lines until a blank comment line,
// the next marker or the end of the comment, so a long one wraps like prose.
// The rows are joined on the rule's name, which is therefore unique across the
// document. A rule some implementation names and no test holds, or that a test
// holds and no implementation names, fails the generator naming the rule; so do
// two implementations that disagree on the rule's page or section.
//
// A marker is read from four kinds of place, and from nowhere else — one found
// anywhere else fails the generator rather than being silently dropped:
//
//   - the doc comment of a function, method, type, constant or variable, which
//     names it as `dir.Name` or `dir.Type.Method`, dir relative to the module;
//   - a package's doc comment, which names the package, for a rule a whole
//     package carries (the alphabets' spellings are every alphabet's, and a
//     gate's rule is its package's, so a gate does not cite itself);
//   - the comment above an entry of chargen's fixture roster, which names the
//     golden `name` that fixture generates — a golden is a test, and the roster
//     is where it is declared;
//   - a `$comment` in docs/character.schema.json, which names the schema, for
//     the rules the schema itself implements. JSON has no other comment.
//
// Only a test holds a rule: a `Holds:` is refused anywhere but a Test function
// in a test file or a golden, and a golden implements nothing.
//
// # The document
//
// Only the tables are generated. The preamble, the section headings and the
// prose under them are the document's own, hand-written and carried through as
// they stand: a section is a `## ` heading, a marker names one, and the
// section's table is written where its old one was, or at the section's end
// if it had none. A marker naming a heading the document lacks fails, naming
// the section, and so does a section with a table no marker writes, which the
// rewrite would otherwise drop. Within a section the rows run in page order, Book 1 then 2 then
// 3, and rows cited to something other than a page follow them.
//
// The test rewrites the document under -regenerate and otherwise fails when
// the committed document differs from what the markers generate, the way the
// goldens and the schema's examples work.
//
// Implements: Every rule here names an implementation and a test that exist
// [Documents] in The record.
package coverage_test

import (
	"cmp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

//nolint:gochecknoglobals // the flag package has no other shape.
var regenerate = flag.Bool("regenerate", false, "rewrite docs/COVERAGE.md from the markers")

// The module root, relative to this package, and the files the generator
// reads and writes beneath it.
const (
	root     = "../.."
	document = "docs/COVERAGE.md"
	schema   = "docs/character.schema.json"

	// The documents a cite that is not a page may name.
	prd      = "docs/PRD.md"
	adrDir   = "docs/adr"
	readings = "internal/rules/data/ERRATA.md"

	// rosterDir and rosterVar are where chargen's fixture roster is declared.
	rosterDir = "internal/chargen"
	rosterVar = "fixtures"
)

// errMarker is every way the markers can fail to make a document.
var errMarker = errors.New("coverage markers")

var (
	implementsMarker = regexp.MustCompile(`^Implements: (.+) \[([^\]]+)\] in (.+)\.$`)
	holdsMarker      = regexp.MustCompile(`^Holds: (.+)\.$`)

	// pageCite is a page cite's leading book and page, "1:4" or "3:22".
	pageCite = regexp.MustCompile(`^(\d+):(\d+)`)

	// The labels the other cites are drawn from, as each document writes
	// them: a PRD heading or requirement, an ADR's file, a reading's heading.
	prdHeading     = regexp.MustCompile(`(?m)^## (.+)$`)
	prdRequirement = regexp.MustCompile(`(?m)^\*\*(FR\d+) — `)
	adrFile        = regexp.MustCompile(`^(\d{4})-.+\.md$`)
	readingHeading = regexp.MustCompile(`(?m)^### (E\d{3}) — `)

	// adrWithdraws is an ADR's line naming the PRD labels it withdrew.
	adrWithdraws = regexp.MustCompile("(?m)^Withdraws from `docs/PRD.md`: (.+)\\.$")
)

// implementation is one Implements marker and the declaration carrying it.
type implementation struct {
	rule, page, section, site string
}

// holding is one Holds marker and the test or golden carrying it.
type holding struct {
	rule, site string
	golden     bool
}

// markers is everything the tree says about coverage.
type markers struct {
	implementations []implementation
	holdings        []holding
}

// row is one generated row of a section's table.
type row struct {
	rule, page, section string
	implementations     []string
	tests, goldens      []string
}

// The committed document is what the markers generate.
func TestTheCoverageMapIsGenerated(t *testing.T) {
	t.Parallel()

	found, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}

	known, err := readAuthorities(root)
	if err != nil {
		t.Fatal(err)
	}

	err = known.check(found)
	if err != nil {
		t.Fatal(err)
	}

	committed := readDocument(t)

	generated, err := generate(committed, found)
	if err != nil {
		t.Fatal(err)
	}

	if *regenerate {
		// Written beside the document and renamed over it, so the tests
		// reading it in parallel see the old one or the new one, never half.
		path := filepath.Join(root, document)

		err := os.WriteFile(path+".new", []byte(generated), 0o600)
		if err == nil {
			err = os.Rename(path+".new", path)
		}

		if err != nil {
			t.Fatal(err)
		}

		return
	}

	if generated != committed {
		t.Errorf("%s differs from what the markers generate, first at:\n  committed: %s\n  generated: %s\n"+
			"Edit the markers, not the table, then regenerate and read the diff:\n"+
			"  go test ./internal/coverage -regenerate", document, firstDifference(committed, generated),
			firstDifference(generated, committed))
	}
}

// A marker removed from either side fails the generator, naming the rule.
//
// Holds: Every rule here names an implementation and a test that exist.
func TestAMarkerRemovedFailsNamingTheRule(t *testing.T) {
	t.Parallel()

	found, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}

	committed := readDocument(t)

	victim := found.implementations[0].rule

	for name, breaks := range map[string]func(markers) markers{
		"from the implementation": func(m markers) markers {
			m.implementations = slices.DeleteFunc(slices.Clone(m.implementations),
				func(i implementation) bool { return i.rule == victim })

			return m
		},
		"from the test": func(m markers) markers {
			m.holdings = slices.DeleteFunc(slices.Clone(m.holdings),
				func(h holding) bool { return h.rule == victim })

			return m
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := generate(committed, breaks(found))
			refused(t, err, victim)
		})
	}
}

// A cite that is not a page names an authority that exists: a reading, an ADR,
// or a heading or requirement of the PRD, whose labels stay citable though it
// governs nothing (#232), unless an ADR withdrew that label. Anything else
// fails, naming the rule.
func TestACiteNamesAnAuthority(t *testing.T) {
	t.Parallel()

	found, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}

	known, err := readAuthorities(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, cite := range []string{"(E011)", "ADR 0001", "ADR 0002", "Domain model", "FR8"} {
		if !known.cites[cite] {
			t.Errorf("[%s] names an authority that exists, and is refused", cite)
		}
	}

	victim := found.implementations[0]

	if known.withdrawn["Determinism"] != "ADR 0001" {
		t.Errorf("ADR 0001 withdrew [Determinism], and the generator does not know it")
	}

	for _, cite := range []string{"Nonsense", "ADR 0099", "(E099)", "FR99", "Domain", "Determinism"} {
		t.Run(cite, func(t *testing.T) {
			t.Parallel()

			implementations := slices.Clone(found.implementations)

			implementations[0].page = cite

			broken := markers{implementations: implementations, holdings: found.holdings}

			refused(t, known.check(broken), victim.rule)
		})
	}
}

// A hand edit to a table does not survive generation, and a hand edit to the
// prose does: the tables are the markers', the prose is the document's.
func TestOnlyTheTablesAreGenerated(t *testing.T) {
	t.Parallel()

	found, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}

	committed := readDocument(t)

	rule := found.implementations[0].rule
	edited := strings.Replace(committed, "| "+rule+" ", "| "+rule+"!", 1)

	if edited == committed {
		t.Fatalf("the committed document has no row %q to edit", rule)
	}

	generated, err := generate(edited, found)
	if err != nil {
		t.Fatal(err)
	}

	if generated != committed {
		t.Errorf("a hand edit to a row survived generation")
	}

	prose := strings.Replace(committed, "## What the engine walks\n", "## What the engine walks\n\nA note.\n", 1)

	generated, err = generate(prose, found)
	if err != nil {
		t.Fatal(err)
	}

	if generated != prose {
		t.Errorf("a hand edit to the prose did not survive generation")
	}
}

// Markers that cannot make a document fail, naming what is wrong.
func TestMarkersThatCannotMakeADocument(t *testing.T) {
	t.Parallel()

	for name, tc := range documentBreakages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := generate(tc.doc, tc.found)
			refused(t, err, tc.mentions)
		})
	}
}

// The fixtures the breakage tables share.
const (
	aDoc   = "# Title\n\n## Alpha\n\n## Beta\n"
	aPage  = "1:4"
	alpha  = "Alpha"
	aRule  = "A rule"
	anImpl = "pkg.F"
	aFile  = "p/p.go"
)

// aHolding is a test holding aRule.
//
//nolint:gochecknoglobals // a test fixture, and Go has no const slice.
var aHolding = []holding{{aRule, "pkg.TestF", false}}

// documentBreakages is each set of markers that cannot make a document, and
// what the failure must name.
//
//nolint:gochecknoglobals // a test table too long for its function.
var documentBreakages = map[string]struct {
	doc      string
	found    markers
	mentions string
}{
	"a section the document lacks": {
		aDoc,
		markers{
			implementations: []implementation{{aRule, aPage, "Gamma", anImpl}},
			holdings:        aHolding,
		},
		"Gamma",
	},
	"a table no marker writes": {
		aDoc + "\n| Rule |\n| ---- |\n| Kept |\n",
		markers{
			implementations: []implementation{{aRule, aPage, alpha, anImpl}},
			holdings:        aHolding,
		},
		"Beta",
	},
	"two pages for one rule": {
		aDoc,
		markers{
			implementations: []implementation{
				{aRule, aPage, alpha, anImpl},
				{aRule, "1:5", alpha, "pkg.G"},
			},
			holdings: aHolding,
		},
		aRule,
	},
	"two sections for one rule": {
		aDoc,
		markers{
			implementations: []implementation{
				{aRule, aPage, alpha, anImpl},
				{aRule, aPage, "Beta", "pkg.G"},
			},
			holdings: aHolding,
		},
		aRule,
	},
}

// A marker that does not parse, or sits where no name can be given to it,
// fails, naming what it says.
func TestAMarkerOutOfPlaceOrOutOfShape(t *testing.T) {
	t.Parallel()

	for name, tc := range markerBreakages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			for path, text := range map[string]string{tc.path: tc.source} {
				full := filepath.Join(dir, filepath.FromSlash(path))

				err := os.MkdirAll(filepath.Dir(full), 0o700)
				if err != nil {
					t.Fatal(err)
				}

				err = os.WriteFile(full, []byte(text), 0o600)
				if err != nil {
					t.Fatal(err)
				}
			}

			_, err := collect(dir)
			refused(t, err, tc.mentions)
		})
	}
}

// markerBreakages is each file that carries a marker it must not, at the
// path it is written to, and what the failure must name.
//
//nolint:gochecknoglobals // a test table too long for its function.
var markerBreakages = map[string]struct {
	path, source, mentions string
}{
	"a page with no brackets": {
		aFile,
		"package p\n\n// F does.\n//\n// Implements: A rule 1:4 in Alpha.\nfunc F() {}\n",
		"A rule 1:4 in Alpha",
	},
	"no closing period": {
		"p/p_test.go",
		"package p\n\n// TestF holds.\n//\n// Holds: A rule\nfunc TestF() {}\n",
		aRule,
	},
	"inside a function body": {
		aFile,
		"package p\n\nfunc F() {\n\t// Holds: A rule.\n\t_ = 1\n}\n",
		aRule,
	},
	"on a parenthesized declaration rather than its spec": {
		aFile,
		"package p\n\n// Implements: A rule [1:4] in Alpha.\nconst (\n\tA = 1\n\tB = 2\n)\n",
		aRule,
	},
	"a hold on something that is not a test": {
		aFile,
		"package p\n\n// TestF is not in a test file.\n//\n// Holds: A rule.\nfunc TestF() {}\n",
		aRule,
	},
	"a hold on a test helper": {
		"p/p_test.go",
		"package p\n\n// helper helps.\n//\n// Holds: A rule.\nfunc helper() {}\n",
		aRule,
	},
	"a golden that implements": {
		rosterDir + "/f_test.go",
		"package p\n\nvar fixtures = []fixture{\n\t// Implements: A rule [1:4] in Alpha.\n\t{name: \"g\"},\n}\n",
		aRule,
	},
	"a schema comment out of shape": {
		schema,
		`{"$comment": "Implements: A rule 1:4 in Alpha."}`,
		"A rule 1:4",
	},
}

// A marker wraps onto the lines after it, and stops at a blank line or the
// next marker.
func TestAMarkerWraps(t *testing.T) {
	t.Parallel()

	got, err := parseComment([]string{
		"F does a thing.",
		"",
		"Implements: A long rule that",
		"wraps [1:4, 9 (E006)] in Alpha.",
		"Holds: Another.",
		"",
		"More prose.",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"Implements: A long rule that wraps [1:4, 9 (E006)] in Alpha.", "Holds: Another."}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func refused(t *testing.T, err error, mentions string) {
	t.Helper()

	switch {
	case err == nil:
		t.Errorf("generated without complaint")
	case !strings.Contains(err.Error(), mentions):
		t.Errorf("error %q does not name %q", err, mentions)
	case !errors.Is(err, errMarker):
		t.Errorf("error %q does not wrap errMarker", err)
	}
}

func readDocument(t *testing.T) string {
	t.Helper()

	text, err := os.ReadFile(filepath.Join(root, document))
	if err != nil {
		t.Fatal(err)
	}

	return string(text)
}

// firstDifference is the first line of a that b does not have at the same
// place.
func firstDifference(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i, line := range al {
		if i >= len(bl) || bl[i] != line {
			return line
		}
	}

	return "(end of document)"
}

// collect reads every marker under dir: Go source and the schema.
func collect(dir string) (markers, error) {
	var (
		found markers
		errs  []error
	)

	fset := token.NewFileSet()

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if path != dir && (entry.Name() == "testdata" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}

			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("%w: %w", errMarker, err)
		}

		switch {
		case strings.HasSuffix(path, ".go"):
			errs = append(errs, collectGo(fset, path, filepath.ToSlash(filepath.Dir(rel)), &found))
		case filepath.ToSlash(rel) == schema:
			errs = append(errs, collectSchema(path, &found))
		}

		return nil
	})
	if err != nil {
		return markers{}, fmt.Errorf("%w: walking %s: %w", errMarker, dir, err)
	}

	return found, errors.Join(errs...)
}

// collectGo reads one Go file's markers, and refuses any in a comment that is
// not a place a marker can be.
func collectGo(fset *token.FileSet, path, pkg string, found *markers) error {
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("%w: %w", errMarker, err)
	}

	sites := make(map[*ast.CommentGroup]site)

	if file.Doc != nil {
		sites[file.Doc] = site{label: pkg}
	}

	testFile := strings.HasSuffix(path, "_test.go")

	for _, decl := range file.Decls {
		declarationSites(decl, pkg, testFile, sites)

		if pkg == rosterDir {
			rosterSites(fset, file, decl, sites)
		}
	}

	var errs []error

	for _, group := range file.Comments {
		lines := commentLines(group)

		parsed, err := parseComment(lines)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", fset.Position(group.Pos()), err))

			continue
		}

		if len(parsed) == 0 {
			continue
		}

		at, ok := sites[group]
		if !ok {
			errs = append(errs, fmt.Errorf("%w: %s: %q is on nothing a marker can name",
				errMarker, fset.Position(group.Pos()), parsed[0]))

			continue
		}

		for _, marker := range parsed {
			errs = append(errs, found.add(marker, at))
		}
	}

	return errors.Join(errs...)
}

// site is what a marker's comment names, and whether it is a test: a
// golden, or a Test function in a test file. Only a test holds a rule.
type site struct {
	label        string
	golden, test bool
}

// declarationSites names the doc comment of a declaration, and of each spec
// in a parenthesized one.
func declarationSites(decl ast.Decl, pkg string, testFile bool, sites map[*ast.CommentGroup]site) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		if decl.Doc == nil {
			return
		}

		name := decl.Name.Name
		if decl.Recv != nil && len(decl.Recv.List) > 0 {
			name = receiver(decl.Recv.List[0].Type) + "." + name
		}

		sites[decl.Doc] = site{
			label: pkg + "." + name,
			test:  testFile && decl.Recv == nil && strings.HasPrefix(decl.Name.Name, "Test"),
		}
	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			name, doc := specName(spec)
			if !decl.Lparen.IsValid() {
				doc = decl.Doc
			}

			if doc != nil && name != "" {
				sites[doc] = site{label: pkg + "." + name}
			}
		}
	}
}

func receiver(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.StarExpr:
		return receiver(expr.X)
	case *ast.IndexExpr:
		return receiver(expr.X)
	case *ast.IndexListExpr:
		return receiver(expr.X)
	case *ast.Ident:
		return expr.Name
	default:
		return ""
	}
}

func specName(spec ast.Spec) (string, *ast.CommentGroup) {
	switch spec := spec.(type) {
	case *ast.TypeSpec:
		return spec.Name.Name, spec.Doc
	case *ast.ValueSpec:
		return spec.Names[0].Name, spec.Doc
	default:
		return "", nil
	}
}

// rosterSites names the comment directly above each entry of the fixture
// roster after the golden that entry generates.
func rosterSites(fset *token.FileSet, file *ast.File, decl ast.Decl, sites map[*ast.CommentGroup]site) {
	gen, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	for _, spec := range gen.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok || value.Names[0].Name != rosterVar || len(value.Values) != 1 {
			continue
		}

		roster, ok := value.Values[0].(*ast.CompositeLit)
		if !ok {
			continue
		}

		for _, entry := range roster.Elts {
			name := fixtureName(entry)
			if name == "" {
				continue
			}

			line := fset.Position(entry.Pos()).Line

			for _, group := range file.Comments {
				if fset.Position(group.End()).Line == line-1 {
					sites[group] = site{label: name, golden: true, test: true}
				}
			}
		}
	}
}

// fixtureName is the string a roster entry gives its name field.
func fixtureName(entry ast.Expr) string {
	literal, ok := entry.(*ast.CompositeLit)
	if !ok {
		return ""
	}

	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key, ok := field.Key.(*ast.Ident)
		if !ok || key.Name != "name" {
			continue
		}

		value, ok := field.Value.(*ast.BasicLit)
		if !ok {
			return ""
		}

		name, err := strconv.Unquote(value.Value)
		if err != nil {
			return ""
		}

		return name
	}

	return ""
}

// commentLines is a comment group's text a line at a time, without the
// comment markers. A block comment is one line per line it spans.
func commentLines(group *ast.CommentGroup) []string {
	var lines []string

	for _, comment := range group.List {
		text := comment.Text
		if body, ok := strings.CutPrefix(text, "//"); ok {
			lines = append(lines, strings.TrimPrefix(body, " "))

			continue
		}

		text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
		lines = append(lines, strings.Split(text, "\n")...)
	}

	return lines
}

// parseComment finds the markers in a comment's lines, each joined with the
// lines it wraps onto, and checks each is in one of the two shapes.
func parseComment(lines []string) ([]string, error) {
	var found []string

	current := make([]string, 0, 1)
	started := false

	flush := func() {
		if started {
			found = append(found, strings.Join(current, " "))
		}

		current, started = current[:0], false
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "Implements: "), strings.HasPrefix(line, "Holds: "):
			flush()

			current, started = append(current, line), true
		case line == "":
			flush()
		case started:
			current = append(current, line)
		}
	}

	flush()

	for _, marker := range found {
		if !implementsMarker.MatchString(marker) && !holdsMarker.MatchString(marker) {
			return nil, fmt.Errorf("%w: %q is neither \"Implements: rule [page] in section.\" nor \"Holds: rule.\"",
				errMarker, marker)
		}
	}

	return found, nil
}

// add records one parsed marker against what carries it.
func (m *markers) add(marker string, where site) error {
	if match := implementsMarker.FindStringSubmatch(marker); match != nil {
		if where.golden {
			return fmt.Errorf("%w: golden %s implements %q; a golden holds a rule and implements none",
				errMarker, where.label, match[1])
		}

		m.implementations = append(m.implementations, implementation{
			rule: match[1], page: match[2], section: match[3], site: where.label,
		})

		return nil
	}

	match := holdsMarker.FindStringSubmatch(marker)
	if match == nil {
		return fmt.Errorf("%w: %q is not a marker", errMarker, marker)
	}

	if !where.test {
		return fmt.Errorf("%w: %s holds %q and is not a test; a rule is held by a Test function or a golden",
			errMarker, where.label, match[1])
	}

	m.holdings = append(m.holdings, holding{rule: match[1], site: where.label, golden: where.golden})

	return nil
}

// collectSchema reads the markers in the schema's $comment keywords.
func collectSchema(path string, found *markers) error {
	text, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%w: %w", errMarker, err)
	}

	var document any

	err = json.Unmarshal(text, &document)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", errMarker, schema, err)
	}

	var errs []error

	for _, comment := range schemaComments(document) {
		parsed, err := parseComment([]string{comment})
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", schema, err))

			continue
		}

		for _, marker := range parsed {
			errs = append(errs, found.add(marker, site{label: schema}))
		}
	}

	return errors.Join(errs...)
}

// schemaComments is every $comment in a decoded JSON document.
func schemaComments(value any) []string {
	var comments []string

	switch value := value.(type) {
	case map[string]any:
		if comment, ok := value["$comment"].(string); ok {
			comments = append(comments, comment)
		}

		for _, child := range value {
			comments = append(comments, schemaComments(child)...)
		}
	case []any:
		for _, child := range value {
			comments = append(comments, schemaComments(child)...)
		}
	}

	return comments
}

// generate is the document with every section's table rewritten from the
// markers.
func generate(doc string, found markers) (string, error) {
	sections, err := join(found)
	if err != nil {
		return "", err
	}

	headings := make(map[string]bool)

	var (
		errs    []error
		section string
	)

	for line := range strings.SplitSeq(doc, "\n") {
		if heading, ok := strings.CutPrefix(line, "## "); ok {
			section = heading
			headings[heading] = true
		}

		// A table no marker writes would be dropped by the rewrite, so it
		// is refused rather than lost: a table here is the markers'.
		if strings.HasPrefix(line, "| ") && len(sections[section]) == 0 && !headings["!"+section] {
			headings["!"+section] = true

			errs = append(errs, fmt.Errorf("%w: section %q has a table and no marker writes a row of it",
				errMarker, section))
		}
	}

	for _, section := range slices.Sorted(maps.Keys(sections)) {
		rows := sections[section]
		if !headings[section] && len(rows) > 0 {
			errs = append(errs, fmt.Errorf("%w: %q is listed under %q, which is not a section of %s",
				errMarker, rows[0].rule, section, document))
		}
	}

	if len(errs) > 0 {
		return "", errors.Join(errs...)
	}

	return rewrite(doc, sections), nil
}

// join is the rows of every section, joined on the rule's name. It refuses a
// rule with only one side, and implementations that disagree about a rule.
func join(found markers) (map[string][]row, error) {
	rules, errs := joined(found)
	sections := make(map[string][]row)

	for _, rule := range slices.Sorted(maps.Keys(rules)) {
		existing := rules[rule]
		if len(existing.tests)+len(existing.goldens) == 0 {
			errs = append(errs, fmt.Errorf("%w: %q is implemented by %s and held by no test",
				errMarker, rule, strings.Join(existing.implementations, ", ")))

			continue
		}

		for _, list := range [][]string{existing.implementations, existing.tests, existing.goldens} {
			slices.Sort(list)
		}

		sections[existing.section] = append(sections[existing.section], existing)
	}

	for _, rows := range sections {
		slices.SortStableFunc(rows, byPage)
	}

	return sections, errors.Join(errs...)
}

// joined is every rule's row, keyed by the rule, with whatever disagreed or
// was held by nothing.
func joined(found markers) (map[string]row, []error) {
	rules := make(map[string]row)

	var errs []error

	for _, impl := range found.implementations {
		existing, ok := rules[impl.rule]
		if !ok {
			existing = row{rule: impl.rule, page: impl.page, section: impl.section}
		}

		if existing.page != impl.page || existing.section != impl.section {
			errs = append(errs, fmt.Errorf("%w: %q: %s cites [%s] in %s, another implementation [%s] in %s",
				errMarker, impl.rule, impl.site, impl.page, impl.section, existing.page, existing.section))
		}

		existing.implementations = append(existing.implementations, impl.site)
		rules[impl.rule] = existing
	}

	for _, held := range found.holdings {
		existing, ok := rules[held.rule]
		if !ok {
			errs = append(errs, fmt.Errorf("%w: %q is held by %s and implemented by nothing", errMarker, held.rule, held.site))

			continue
		}

		if held.golden {
			existing.goldens = append(existing.goldens, held.site)
		} else {
			existing.tests = append(existing.tests, held.site)
		}

		rules[held.rule] = existing
	}

	return rules, errs
}

// authorities is every cite that is not a page and names something that
// exists, and every PRD label an ADR withdrew, with the ADR that did, read
// from the documents themselves rather than listed here.
type authorities struct {
	cites     map[string]bool
	withdrawn map[string]string
}

func readAuthorities(dir string) (authorities, error) {
	known := authorities{cites: make(map[string]bool), withdrawn: make(map[string]string)}

	text, err := os.ReadFile(filepath.Join(dir, prd))
	if err != nil {
		return authorities{}, fmt.Errorf("%w: %w", errMarker, err)
	}

	for _, match := range prdHeading.FindAllStringSubmatch(string(text), -1) {
		known.cites[match[1]] = true
	}

	for _, match := range prdRequirement.FindAllStringSubmatch(string(text), -1) {
		known.cites[match[1]] = true
	}

	err = known.readADRs(dir)
	if err != nil {
		return authorities{}, err
	}

	text, err = os.ReadFile(filepath.Join(dir, readings))
	if err != nil {
		return authorities{}, fmt.Errorf("%w: %w", errMarker, err)
	}

	for _, match := range readingHeading.FindAllStringSubmatch(string(text), -1) {
		known.cites["("+match[1]+")"] = true
	}

	var withdrawals []error

	for _, label := range slices.Sorted(maps.Keys(known.withdrawn)) {
		if !known.cites[label] {
			withdrawals = append(withdrawals, fmt.Errorf("%w: %s withdraws [%s], which is no PRD heading or requirement",
				errMarker, known.withdrawn[label], label))
		}
	}

	return known, errors.Join(withdrawals...)
}

// readADRs adds each ADR in docs/adr as a cite, and each PRD label one says
// it withdrew.
func (known authorities) readADRs(dir string) error {
	entries, err := os.ReadDir(filepath.Join(dir, adrDir))
	if err != nil {
		return fmt.Errorf("%w: %w", errMarker, err)
	}

	for _, entry := range entries {
		match := adrFile.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}

		adr := "ADR " + match[1]

		known.cites[adr] = true

		record, err := os.ReadFile(filepath.Join(dir, adrDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("%w: %w", errMarker, err)
		}

		for _, line := range adrWithdraws.FindAllStringSubmatch(string(record), -1) {
			for label := range strings.SplitSeq(line[1], ", ") {
				known.withdrawn[label] = adr
			}
		}
	}

	return nil
}

// check refuses every implementation whose cite is neither a page nor an
// authority that exists and is in force. A page cite's form is the page's own
// business.
func (known authorities) check(found markers) error {
	var errs []error

	for _, impl := range found.implementations {
		if pageCite.MatchString(impl.page) {
			continue
		}

		switch by, withdrawn := known.withdrawn[impl.page]; {
		case withdrawn:
			errs = append(errs, fmt.Errorf("%w: %q: %s cites [%s], which %s withdrew",
				errMarker, impl.rule, impl.site, impl.page, by))
		case !known.cites[impl.page]:
			errs = append(errs, fmt.Errorf(
				"%w: %q: %s cites [%s], which is no page, reading, ADR, or PRD heading or requirement",
				errMarker, impl.rule, impl.site, impl.page,
			))
		}
	}

	return errors.Join(errs...)
}

// byPage orders rows Book 1 before 2 before 3, by page within a book, and
// rows cited to something other than a page after every one that is.
func byPage(a, b row) int {
	return cmp.Or(cmp.Compare(pageKey(a.page), pageKey(b.page)), cmp.Compare(a.rule, b.rule))
}

func pageKey(page string) int {
	match := pageCite.FindStringSubmatch(page)
	if match == nil {
		return 1 << 20
	}

	book, _ := strconv.Atoi(match[1])
	number, _ := strconv.Atoi(match[2])

	return book*1000 + number
}

// rewrite replaces each section's table with the generated one, and adds one
// at the end of a section that had none.
func rewrite(doc string, sections map[string][]row) string {
	var (
		out     []string
		section string
		written = make(map[string]bool)
	)

	lines := strings.Split(doc, "\n")

	finish := func() {
		if section == "" || written[section] || len(sections[section]) == 0 {
			return
		}

		for len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}

		out = append(out, "")
		out = append(out, table(sections[section])...)
		out = append(out, "")
		written[section] = true
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if heading, ok := strings.CutPrefix(line, "## "); ok {
			finish()

			section = heading
		}

		if strings.HasPrefix(line, "|") {
			for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "|") {
				i++
			}

			if !written[section] && len(sections[section]) > 0 {
				out = append(out, table(sections[section])...)
				written[section] = true
			}

			continue
		}

		out = append(out, line)
	}

	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	finish()

	return strings.Join(out, "\n") + "\n"
}

// table is a section's rows as a Markdown table, padded the way the
// repository's formatter pads one.
func table(rows []row) []string {
	cells := make([][]string, 0, 1+len(rows))

	cells = append(cells, []string{"Rule", "Page", "Implementation", "Test"})

	for _, r := range rows {
		cells = append(cells, []string{r.rule, r.page, ticked(r.implementations), tests(r)})
	}

	widths := make([]int, len(cells[0]))

	for _, line := range cells {
		for i, cell := range line {
			widths[i] = max(widths[i], utf8.RuneCountInString(cell))
		}
	}

	format := func(line []string) string {
		padded := make([]string, len(line))
		for i, cell := range line {
			padded[i] = cell + strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell))
		}

		return "| " + strings.Join(padded, " | ") + " |"
	}

	separator := make([]string, len(widths))
	for i, width := range widths {
		separator[i] = strings.Repeat("-", width)
	}

	out := make([]string, 0, 1+len(cells))

	out = append(out, format(cells[0]), format(separator))

	for _, line := range cells[1:] {
		out = append(out, format(line))
	}

	return out
}

func ticked(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "`" + name + "`"
	}

	return strings.Join(quoted, ", ")
}

// tests is a row's test cell: its tests, then its goldens.
func tests(r row) string {
	parts := []string{}
	if len(r.tests) > 0 {
		parts = append(parts, ticked(r.tests))
	}

	switch len(r.goldens) {
	case 0:
	case 1:
		parts = append(parts, "golden "+ticked(r.goldens))
	default:
		parts = append(parts, "goldens "+ticked(r.goldens))
	}

	return strings.Join(parts, ", ")
}
