package traveller_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// Errata lists every Erratum constant, in order, numbered from 1 (#333). The
// constants are read from the package's own source, every file of it, and
// only those of type Erratum: a const block's later names take the type of
// the name above them, as Go does.
//
// Holds: Every erratum constant is in Errata, in order.
func TestErrataListsEveryConstant(t *testing.T) {
	t.Parallel()

	declared := erratumConstants(t)

	listed := make([]string, len(traveller.Errata))

	for i, e := range traveller.Errata {
		listed[i] = e.String()

		if int(e) != i+1 {
			t.Errorf("Errata[%d] is %v, numbered %d; the list runs from 1 in order", i, e, int(e))
		}
	}

	for _, name := range declared {
		if !slices.Contains(listed, name) {
			t.Errorf("%s is declared an Erratum and is not in Errata", name)
		}
	}

	for _, name := range listed {
		if !slices.Contains(declared, name) {
			t.Errorf("Errata lists %s, which is declared nowhere", name)
		}
	}

	if !slices.Equal(declared, listed) {
		t.Errorf("the constants run %v and Errata %v; they are listed in the order declared", declared, listed)
	}
}

// erratumConstants is every constant of type Erratum the package declares,
// in source order within each file, the files in name order.
func erratumConstants(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	var names []string

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}

			names = append(names, typedErratum(gen)...)
		}
	}

	return names
}

// typedErratum is the names a const declaration gives type Erratum. A spec
// with neither a type nor a value repeats the one above it, type included.
func typedErratum(gen *ast.GenDecl) []string {
	var (
		names    []string
		repeated bool
	)

	for _, spec := range gen.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		switch {
		case value.Type != nil:
			ident, isIdent := value.Type.(*ast.Ident)

			repeated = isIdent && ident.Name == "Erratum"
		case len(value.Values) > 0:
			repeated = false
		}

		if repeated {
			for _, name := range value.Names {
				names = append(names, name.Name)
			}
		}
	}

	return names
}
