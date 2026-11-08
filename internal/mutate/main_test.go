package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const source = "m.go"

// args is a mutate command line that changes old to replacement in source and
// runs the module's tests.
func args(old, replacement string) []string {
	return []string{"-file", source, "-old", old, "-new", replacement, "go", "test", "./..."}
}

// A module of one function and one test of it, in a git repository of its
// own, so mutate has a tree to copy and a test to run.
func module(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	for name, text := range map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.24\n",
		source:   "package m\n\n// One is one.\nfunc One() int { return 1 }\n",
		"m_test.go": "package m\n\nimport \"testing\"\n\nfunc TestOne(t *testing.T) {\n" +
			"\tif One() != 1 {\n\t\tt.Fatal(\"One is not one\")\n\t}\n}\n",
	} {
		err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}

	git := exec.CommandContext(t.Context(), "git", "init", "-q")

	git.Dir = root

	out, err := git.CombinedOutput()
	if err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	return root
}

// A mutation the test catches is reported killed, one it misses is reported
// as surviving, and the tree mutate was run from is left exactly as it was.
func TestAMutationIsKilledOrSurvivesInACopy(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		old, replacement, says string
		survives               bool
	}{
		"a change the test catches": {old: "return 1", replacement: "return 2", says: "KILLED"},
		"a change it cannot see":    {old: "One is one.", replacement: "One is uno.", says: "SURVIVED", survives: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := module(t)

			var out strings.Builder

			err := run(t.Context(), args(tc.old, tc.replacement), root, &out)

			if tc.survives != errors.Is(err, errSurvived) {
				t.Errorf("survived = %v, want %v (error %v)", errors.Is(err, errSurvived), tc.survives, err)
			}

			if !strings.Contains(out.String(), tc.says) {
				t.Errorf("the verdict does not say %s:\n%s", tc.says, out.String())
			}

			kept, err := os.ReadFile(filepath.Join(root, source))
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(string(kept), "return 1") || !strings.Contains(string(kept), "One is one.") {
				t.Errorf("the tree mutate ran from was changed:\n%s", kept)
			}
		})
	}
}

// Text that occurs more than once, or not at all, is refused rather than
// replaced somewhere it was not meant for.
func TestAMutationMustLandOnce(t *testing.T) {
	t.Parallel()

	for _, old := range []string{"One", "nowhere"} {
		err := run(t.Context(), args(old, "x"), module(t), &strings.Builder{})
		if !errors.Is(err, errNotUnique) {
			t.Errorf("-old %q: %v, want the text refused as not unique", old, err)
		}
	}
}

// A repository nested inside the tree - an agent's worktree, say - is listed
// by git as a directory and is not part of the module, so it is passed over
// rather than failing the copy.
func TestANestedRepositoryIsPassedOver(t *testing.T) {
	t.Parallel()

	root := module(t)
	nested := filepath.Join(root, "worktrees", "other")

	err := os.MkdirAll(nested, 0o700)
	if err != nil {
		t.Fatal(err)
	}

	git := exec.CommandContext(t.Context(), "git", "init", "-q")

	git.Dir = nested

	out, err := git.CombinedOutput()
	if err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	err = run(t.Context(), args("return 1", "return 2"), root, &strings.Builder{})
	if err != nil {
		t.Errorf("a nested repository failed the run: %v", err)
	}
}
