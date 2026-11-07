package coverage_test

import (
	"flag"
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// regenerateAPI is its own flag, not the coverage map's -regenerate: a new
// exported name is a promise, taken on purpose, and rewriting the coverage map
// must not take one in passing.
var regenerateAPI = flag.Bool("regenerate-api", false, "rewrite the exported API baseline")

// apiBaseline is the exported Go API v1.0.0 ships, one declaration a line.
const apiBaseline = "testdata/api.txt"

// The packages whose exported names the tag freezes, in the ordinary Go sense
// (#126). cmd/ctchargen is a command and internal/coverage has no non-test
// code, so neither has an API to freeze.
var apiPackages = []string{
	"github.com/philoserf/ctchargen/traveller",
	"github.com/philoserf/ctchargen/dice",
	"github.com/philoserf/ctchargen/rules",
	"github.com/philoserf/ctchargen/chargen",
	"github.com/philoserf/ctchargen/render",
}

// The exported API is the baseline, in both directions.
//
// A line that has gone is a name removed or a signature changed, which breaks
// a caller. A line that has arrived is a name exported, which is a promise
// made; it is allowed under Go's rules, but only on purpose, so it is
// regenerated rather than absorbed:
//
//	go test ./internal/coverage -regenerate-api
//
// then read the diff. Nothing else held this - six exported names nobody used
// could leave and every test stayed green (#130).
func TestTheExportedAPIIsTheBaseline(t *testing.T) {
	t.Parallel()

	lines := make([]string, 0, len(apiPackages))
	for _, path := range apiPackages {
		lines = append(lines, exported(t, path)...)
	}

	current := strings.Join(lines, "\n") + "\n"

	if *regenerateAPI {
		err := os.WriteFile(apiBaseline, []byte(current), 0o600)
		if err != nil {
			t.Fatalf("writing %s: %v", apiBaseline, err)
		}

		return
	}

	text, err := os.ReadFile(filepath.Clean(apiBaseline))
	if err != nil {
		t.Fatalf("reading %s: %v", apiBaseline, err)
	}

	baseline := strings.Split(strings.TrimSuffix(string(text), "\n"), "\n")

	for _, line := range baseline {
		if !slices.Contains(lines, line) {
			t.Errorf("gone from the exported API: %s", line)
		}
	}

	for _, line := range lines {
		if !slices.Contains(baseline, line) {
			t.Errorf("new to the exported API, and not in %s: %s", apiBaseline, line)
		}
	}
}

// exported is every exported declaration of a package, sorted: each object at
// package scope, each exported field of an exported struct, and each exported
// method of an exported type. Unexported fields stay out, so the baseline does
// not move when a package's insides do.
func exported(t *testing.T, path string) []string {
	t.Helper()

	source := importer.ForCompiler(token.NewFileSet(), "source", nil)
	if source == nil {
		t.Fatal("this toolchain has no source importer")
	}

	pkg, err := source.Import(path)
	if err != nil || pkg == nil {
		t.Fatalf("importing %s: %v", path, err)
	}

	qualify := func(other *types.Package) string { return other.Name() }

	scope := pkg.Scope()
	lines := make([]string, 0, scope.Len())

	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}

		named, isType := obj.(*types.TypeName)
		if !isType {
			lines = append(lines, pkg.Name()+": "+types.ObjectString(obj, qualify))

			continue
		}

		lines = append(lines, typeLines(pkg.Name(), named, qualify)...)
	}

	slices.Sort(lines)

	return lines
}

// typeLines is an exported type's declaration without its struct body, and
// then its exported fields and methods a line apiece.
func typeLines(pkg string, named *types.TypeName, qualify types.Qualifier) []string {
	prefix := pkg + ": " + named.Name()

	under := named.Type().Underlying()

	strct, isStruct := under.(*types.Struct)
	if !isStruct {
		declared := pkg + ": type " + named.Name() + " " + types.TypeString(under, qualify)

		return append([]string{declared}, methodLines(prefix, named, qualify)...)
	}

	lines := []string{pkg + ": type " + named.Name() + " struct"}

	for field := range strct.Fields() {
		if field.Exported() {
			lines = append(lines, prefix+"."+field.Name()+" "+types.TypeString(field.Type(), qualify))
		}
	}

	return append(lines, methodLines(prefix, named, qualify)...)
}

func methodLines(prefix string, named *types.TypeName, qualify types.Qualifier) []string {
	methods := types.NewMethodSet(types.NewPointer(named.Type()))
	lines := make([]string, 0, methods.Len())

	for selection := range methods.Methods() {
		method := selection.Obj()
		if method.Exported() {
			lines = append(lines, prefix+"."+method.Name()+" "+types.TypeString(method.Type(), qualify))
		}
	}

	return lines
}
