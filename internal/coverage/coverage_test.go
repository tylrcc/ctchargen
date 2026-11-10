// Package coverage holds the markers beside the code to each other: every rule
// an implementation names is held by a test, every rule a test holds is named
// by an implementation, and every cite names an authority that exists. It
// writes nothing; the map from each rule to its page, its implementation and
// its test is what the markers say, and -print shows it.
//
// The package has no non-test code.
//
// # Markers
//
// An implementation carries one line per rule it implements, naming the rule
// and its printed page:
//
//	// Implements: Six 2D rolls, in order [1:4].
//
// A test that holds a rule carries one line naming it:
//
//	// Holds: Six 2D rolls, in order.
//
// The bracket is the rule's authority, from a closed set: a page cite, pages
// or ranges (an en dash) joined by ", " or "; ", the first with its book,
// then any readings in one pair of parentheses, naming only Books 1 to 3 and
// readings rules declares, "1:5; 3:22 (E011)"; a reading alone, "(E011)",
// that it declares; "ADR 0001", naming a file in docs/adr; or one of the
// promise labels below, for a promise of the tool's own. The readings are
// read from rules' values and the ADRs from their files, and a cite outside
// the set fails the check, naming the rule.
//
// A marker runs on to the following comment lines until a blank comment line,
// the next marker or the end of the comment, so a long one wraps like prose.
// The markers are joined on the rule's name, which is therefore unique across
// the tree. A rule some implementation names and no test holds, or that a test
// holds and no implementation names, fails the check naming the rule; so do
// two implementations that disagree on the rule's page.
//
// A marker is read from three kinds of place, and from nowhere else — one found
// anywhere else fails the check rather than being silently dropped:
//
//   - the doc comment of a function, method, type, constant or variable, which
//     names it as `dir.Name` or `dir.Type.Method`, dir relative to the module;
//   - a package's doc comment, which names the package, where
//     docs/CODING_STANDARDS.md puts a promise no single declaration carries;
//   - the comment above an entry of chargen's fixture roster, which names the
//     golden `name` that fixture generates — a golden is a test, and the roster
//     is where it is declared.
//
// Only a test holds a rule: a `Holds:` is refused anywhere but a Test function
// in a test file or a golden, and a golden implements nothing.
//
// # The table
//
// Nothing is committed. Under -print the map is written as one Markdown table:
// rule, authority, implementation and test, in page order, Book 1 then 2 then
// 3, and rows cited to something other than a page after them. Run it from
// this package's directory, where a passing test's output is not held back:
//
//	go test -run TestTheCoverageMapHolds -print
//
// Implements: Every rule a marker names has an implementation and a test that
// exist [Documents].
package coverage_test

import (
	"cmp"
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

	"github.com/philoserf/ctchargen/internal/rules"
)

//nolint:gochecknoglobals // the flag package has no other shape.
var printTable = flag.Bool("print", false, "write the coverage table to standard output")

// The module root, relative to this package, and the files the check reads
// beneath it.
const (
	root = "../.."

	// The documents a cite that is not a page may name.
	adrDir = "docs/adr"

	// rosterDir and rosterVar are where chargen's fixture roster is declared.
	rosterDir = "internal/chargen"
	rosterVar = "fixtures"
)

// errMarker is every way the markers can fail to hold each other.
var errMarker = errors.New("coverage markers")

var (
	implementsMarker = regexp.MustCompile(`^Implements: (.+) \[([^\]]+)\]\.$`)
	holdsMarker      = regexp.MustCompile(`^Holds: (.+)\.$`)

	// pageCite is a page cite's leading book and page, "1:4" or "3:22".
	pageCite = regexp.MustCompile(`^(\d+):(\d+)`)

	// citedBook is every book a page cite names, "1:5; 3:22" naming 1 and 3,
	// and citedReading every reading it names, "(E006, E007)" naming two.
	citedBook    = regexp.MustCompile(`(\d+):\d+`)
	citedReading = regexp.MustCompile(`E\d{3}`)

	// pageCiteShape is the whole of a page cite as the tree writes one: pages
	// or ranges (an en dash, never a hyphen), the first with its book and any
	// later one with or without, joined by ", " or "; ", then any readings in
	// one pair of parentheses:
	// "1:5, 10", "1:4, 7–9 (E010)", "1:5; 3:22 (E011)", "1:9 (E006, E007)".
	pageCiteShape = regexp.MustCompile(
		`^\d+:\d+(?:–\d+)?(?:(?:, |; )(?:\d+:)?\d+(?:–\d+)?)*(?: \(E\d{3}(?:, E\d{3})*\))?$`,
	)

	// The label an ADR cite is drawn from, as its file is named.
	adrFile = regexp.MustCompile(`^(\d{4})-.+\.md$`)
)

// promises are the labels a promise of the tool's own cites: the topic it
// belongs to, from this closed list (#391). They were once the sections of
// the v1 contract each promise was first stated under (docs/PRD.md as of
// ec7289f, in the history); promises have been written since, so a label is
// what a promise is about, not where it was first said. The coverage check
// refuses a label not on the list.
//
//   - Authority: where the rules come from, and which pages govern.
//   - Choices: the choice points, and who answers them.
//   - Command line: what the commands take, refuse, write and say.
//   - Documents: what the documents and generated examples promise.
//   - Domain: the domain's values and the sums that say "exactly one of".
//   - Generation record: the record of every die and choice, and the live
//     view of it.
//   - Sheet: what a note tells of the character - its properties, its sheet,
//     its summary and the service history behind them.
//   - Transcription: the transcribed tables and their lift.
//
//nolint:gochecknoglobals // a fixed list, and Go has no const slice.
var promises = []string{
	"Authority", "Choices", "Command line", "Documents",
	"Domain", "Generation record", "Sheet", "Transcription",
}

// implementation is one Implements marker and the declaration carrying it.
type implementation struct {
	rule, page, site string
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

// row is one rule of the map: its page, what implements it and what holds it.
type row struct {
	rule, page      string
	implementations []string
	tests, goldens  []string
}

// Every rule is implemented and held, and every cite names an authority that
// exists. Under -print, the map is written as a table.
func TestTheCoverageMapHolds(t *testing.T) {
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

	rows, err := mapped(found)
	if err != nil {
		t.Fatal(err)
	}

	if *printTable {
		_, err := os.Stdout.WriteString(strings.Join(table(rows), "\n") + "\n")
		if err != nil {
			t.Fatal(err)
		}
	}
}

// A marker removed from either side fails the check, naming the rule.
//
// Holds: Every rule a marker names has an implementation and a test that
// exist.
func TestAMarkerRemovedFailsNamingTheRule(t *testing.T) {
	t.Parallel()

	found, err := collect(root)
	if err != nil {
		t.Fatal(err)
	}

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

			_, err := mapped(breaks(found))
			refused(t, err, victim)
		})
	}
}

// A cite that is not a page names an authority that exists: a reading, an ADR,
// or a promise label. Anything else fails, naming the rule.
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

	for _, cite := range []string{"(E011)", "ADR 0001", "ADR 0002", "Domain", "Choices"} {
		if !known.cites[cite] {
			t.Errorf("[%s] names an authority that exists, and is refused", cite)
		}
	}

	victim := found.implementations[0]

	// Every form a page cite takes is accepted when what it names exists.
	for _, cite := range []string{
		"1:5", "1:2–3", "1:11–13", "1:5, 10", "1:4, 7–9 (E010)", "1:5, 1:7–8",
		"1:5; 3:22 (E011)", "2:18–19", "1:9 (E006, E007)",
	} {
		err := known.check(withFirstCite(t, found, cite))
		if err != nil {
			t.Errorf("[%s] is a page cite naming what exists, and is refused: %v", cite, err)
		}
	}

	// A page cite is checked too: its shape, every book it names is in
	// authority, and every reading it names is declared.
	for _, cite := range []string{
		"Nonsense", "ADR 0099", "(E099)", "FR9", "Domain model", "CLI sketch", "Determinism",
		"1:5 (E099)", "4:5", "1:5; 4:22", "1:5; 3:22 (E099)", "0:5",
		"1:9 (E006, E099)", "1:9 E099", "1:9 (e099)", "1:9 (e006)",
		"1:5 garbage", "1:5; Book 4 p. 5", "1:5 (E99)", "1:5 (E0999)", "1:5,6",
		"1:5 (E006,E007)", "1:11-13", "1:5, 7-9",
	} {
		t.Run(cite, func(t *testing.T) {
			t.Parallel()

			refused(t, known.check(withFirstCite(t, found, cite)), victim.rule)
		})
	}
}

// withFirstCite is the tree's markers with the first implementation's cite
// replaced, the rest untouched.
func withFirstCite(t *testing.T, found markers, cite string) markers {
	t.Helper()

	implementations := slices.Clone(found.implementations)
	if len(implementations) == 0 {
		t.Fatal("the tree has no Implements: marker to re-cite")
	}

	implementations[0].page = cite

	return markers{implementations: implementations, holdings: found.holdings}
}

// The table is the map in page order, Book 1 then 2 then 3 and by page within
// a book, with every row cited to something other than a page after them, and
// a row's tests ahead of its goldens, each list sorted. The markers here are
// made up, so the table they must make is typed rather than computed.
func TestTheTableIsInPageOrder(t *testing.T) {
	t.Parallel()

	const (
		book2   = "Book 2"
		book1p4 = "Book 1, p. 4"
	)

	// Declared out of order, as markers are found across the tree.
	found := markers{
		implementations: []implementation{
			{"Promise", "Command line", "cmd.F"},
			{"Book 3", "3:5", "pkg.C"},
			{"Book 1, p. 25", "1:25", "pkg.B"},
			{book2, "2:18–19", "pkg.D"},
			{book1p4, "1:4 (E010)", "pkg.E"},
			{book1p4, "1:4 (E010)", "pkg.A"},
		},
		holdings: []holding{
			{"Promise", "cmd.TestF", false},
			{"Book 3", "g", true},
			{"Book 1, p. 25", "pkg.TestB", false},
			{book2, "h", true},
			{book2, "pkg.TestD", false},
			{book2, "g", true},
			{book1p4, "pkg.TestA", false},
		},
	}

	rows, err := mapped(found)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"| Rule | Authority | Implementation | Test |",
		"| --- | --- | --- | --- |",
		"| Book 1, p. 4 | 1:4 (E010) | `pkg.A`, `pkg.E` | `pkg.TestA` |",
		"| Book 1, p. 25 | 1:25 | `pkg.B` | `pkg.TestB` |",
		"| Book 2 | 2:18–19 | `pkg.D` | `pkg.TestD`, goldens `g`, `h` |",
		"| Book 3 | 3:5 | `pkg.C` | golden `g` |",
		"| Promise | Command line | `cmd.F` | `cmd.TestF` |",
	}

	if got := table(rows); !slices.Equal(got, want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// Markers that disagree fail, naming what is wrong.
func TestMarkersThatCannotMakeAMap(t *testing.T) {
	t.Parallel()

	for name, tc := range mapBreakages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := mapped(tc.found)
			refused(t, err, tc.mentions)
		})
	}
}

// The fixtures the breakage tables share.
const (
	aPage  = "1:4"
	aRule  = "A rule"
	anImpl = "pkg.F"
	aFile  = "p/p.go"
)

// aHolding is a test holding aRule.
//
//nolint:gochecknoglobals // a test fixture, and Go has no const slice.
var aHolding = []holding{{aRule, "pkg.TestF", false}}

// mapBreakages is each set of markers that cannot make a map, and what the
// failure must name.
//
//nolint:gochecknoglobals // a test table too long for its function.
var mapBreakages = map[string]struct {
	found    markers
	mentions string
}{
	"two pages for one rule": {
		markers{
			implementations: []implementation{
				{aRule, aPage, anImpl},
				{aRule, "1:5", "pkg.G"},
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
		"package p\n\n// F does.\n//\n// Implements: A rule 1:4.\nfunc F() {}\n",
		"A rule 1:4",
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
		"package p\n\n// Implements: A rule [1:4].\nconst (\n\tA = 1\n\tB = 2\n)\n",
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
		"package p\n\nvar fixtures = []fixture{\n\t// Implements: A rule [1:4].\n\t{name: \"g\"},\n}\n",
		aRule,
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
		"wraps [1:4, 9 (E006)].",
		"Holds: Another.",
		"",
		"More prose.",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"Implements: A long rule that wraps [1:4, 9 (E006)].", "Holds: Another."}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func refused(t *testing.T, err error, mentions string) {
	t.Helper()

	switch {
	case err == nil:
		t.Errorf("accepted without complaint")
	case !strings.Contains(err.Error(), mentions):
		t.Errorf("error %q does not name %q", err, mentions)
	case !errors.Is(err, errMarker):
		t.Errorf("error %q does not wrap errMarker", err)
	}
}

// collect reads every marker under dir, in its Go source.
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

		if strings.HasSuffix(path, ".go") {
			errs = append(errs, collectGo(fset, path, filepath.ToSlash(filepath.Dir(rel)), &found))
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
			return nil, fmt.Errorf("%w: %q is neither \"Implements: rule [page].\" nor \"Holds: rule.\"",
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
			rule: match[1], page: match[2], site: where.label,
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

// mapped is every rule's row, joined on the rule's name, in page order. It
// refuses a rule with only one side, and implementations that disagree about
// a rule's page.
func mapped(found markers) ([]row, error) {
	rules, errs := joined(found)

	var rows []row

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

		rows = append(rows, existing)
	}

	slices.SortStableFunc(rows, byPage)

	return rows, errors.Join(errs...)
}

// joined is every rule's row, keyed by the rule, with whatever disagreed or
// was held by nothing.
func joined(found markers) (map[string]row, []error) {
	rules := make(map[string]row)

	var errs []error

	for _, impl := range found.implementations {
		existing, ok := rules[impl.rule]
		if !ok {
			existing = row{rule: impl.rule, page: impl.page}
		}

		if existing.page != impl.page {
			errs = append(errs, fmt.Errorf("%w: %q: %s cites [%s], another implementation [%s]",
				errMarker, impl.rule, impl.site, impl.page, existing.page))
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
// exists: the promise labels, the ADRs read from their files, and the
// readings rules declares.
type authorities struct {
	cites map[string]bool
}

func readAuthorities(dir string) (authorities, error) {
	known := authorities{cites: make(map[string]bool)}

	for _, label := range promises {
		known.cites[label] = true
	}

	err := known.readADRs(dir)
	if err != nil {
		return authorities{}, err
	}

	for _, r := range rules.Readings() {
		known.cites["("+r.ID.String()+")"] = true
	}

	return known, nil
}

// readADRs adds each ADR in docs/adr as a cite.
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

		known.cites["ADR "+match[1]] = true
	}

	return nil
}

// check refuses every implementation whose cite is not a page cite naming
// only what exists, or an authority that exists and is in force. A page cite
// is checked for what it names: every book is in authority and every reading
// has a heading.
func (known authorities) check(found markers) error {
	var errs []error

	for _, impl := range found.implementations {
		if pageCite.MatchString(impl.page) {
			errs = append(errs, known.checkPage(impl)...)

			continue
		}

		if !known.cites[impl.page] {
			errs = append(errs, fmt.Errorf(
				"%w: %q: %s cites [%s], which is no page, reading, ADR, or promise label",
				errMarker, impl.rule, impl.site, impl.page,
			))
		}
	}

	return errors.Join(errs...)
}

// checkPage refuses a page cite that is not in pageCiteShape, or that names a
// book outside authority or a reading rules does not declare.
func (known authorities) checkPage(impl implementation) []error {
	if !pageCiteShape.MatchString(impl.page) {
		return []error{fmt.Errorf("%w: %q: %s cites [%s], which is no page cite's shape",
			errMarker, impl.rule, impl.site, impl.page)}
	}

	var errs []error

	for _, match := range citedBook.FindAllStringSubmatch(impl.page, -1) {
		// Book 1, and Books 2 and 3 where Book 1 points at them (CLAUDE.md,
		// Authority).
		if !slices.Contains([]string{"1", "2", "3"}, match[1]) {
			errs = append(errs, fmt.Errorf("%w: %q: %s cites [%s], and Book %s is not in authority",
				errMarker, impl.rule, impl.site, impl.page, match[1]))
		}
	}

	for _, reading := range citedReading.FindAllString(impl.page, -1) {
		if !known.cites["("+reading+")"] {
			errs = append(errs, fmt.Errorf("%w: %q: %s cites [%s], and no reading is %s",
				errMarker, impl.rule, impl.site, impl.page, reading))
		}
	}

	return errs
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

// table is the map as a Markdown table, one row to a line and unpadded.
func table(rows []row) []string {
	out := make([]string, 0, 2+len(rows))

	out = append(out, "| Rule | Authority | Implementation | Test |", "| --- | --- | --- | --- |")

	for _, r := range rows {
		out = append(out, "| "+strings.Join([]string{r.rule, r.page, ticked(r.implementations), tests(r)}, " | ")+" |")
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
