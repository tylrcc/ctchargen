// Package walkthrough holds WALKTHROUGH.md's transcribed commands to what they
// print, so a change that alters a command's output cannot leave the
// walkthrough quoting the old one.
//
// The package has no non-test code.
//
// # Transcripts and samples
//
// A line "A transcript of `cmd`, run while writing:" followed by a fenced
// block is a claim that cmd prints the block, and cmd is re-run from the
// module root and its output compared exactly. A line "A sample of `cmd`,
// run while writing:" is a run whose output depends on the dice, which are
// random (ADR 0001): cmd is re-run and must succeed and print something, so a
// flag that no longer exists fails, but its output is not compared.
//
// Under -regenerate, every transcript's block is rewritten with what its
// command prints, and samples are left as they are: a sample's prose
// describes the run it shows, so a fresh run would leave the prose wrong.
//
// Every command runs under bash with pipefail, with a ctchargen built from
// this tree first on PATH. A line that says "A transcript of" and names no
// command in backticks fails, since it claims what nothing can check.
package walkthrough_test

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

//nolint:gochecknoglobals // the flag package has no other shape.
var regenerate = flag.Bool("regenerate", false, "rewrite WALKTHROUGH.md's transcripts from their commands")

const (
	root     = "../.."
	document = "WALKTHROUGH.md"

	// nested is set in the environment of every command run here, so the
	// walkthrough's own `go test ./...` transcript does not run this test
	// again inside itself.
	nested = "CTCHARGEN_WALKTHROUGH_NESTED"
)

var (
	transcribed = regexp.MustCompile("^A (transcript|sample) of `(.+)`, run while writing:$")
	claimed     = regexp.MustCompile(`^A (transcript|sample) of `)
)

// claim is one transcribed command and the block it is said to print, with
// where that block's body sits among the document's lines.
type claim struct {
	line       int
	sample     bool
	command    string
	printed    string
	start, end int
}

// Every transcript in the walkthrough is what its command prints today, and
// every sample still runs.
func TestEachTranscriptIsWhatItsCommandPrints(t *testing.T) {
	t.Parallel()

	if os.Getenv(nested) != "" {
		t.Skip("run by the walkthrough's own go test transcript")
	}

	lines, claims := readClaims(t)
	if len(claims) == 0 {
		t.Fatalf("%s transcribes no command, so this checks nothing", document)
	}

	bin := buildCtchargen(t)

	if *regenerate {
		rewrite(t, bin, lines, claims)

		return
	}

	for _, c := range claims {
		out, err := run(t, bin, c.command)

		switch {
		case c.sample && err != nil:
			t.Errorf("%s:%d: the sample `%s` no longer runs: %v\n%s", document, c.line, c.command, err, out)
		case c.sample && strings.TrimSpace(out) == "":
			t.Errorf("%s:%d: the sample `%s` prints nothing", document, c.line, c.command)
		case !c.sample && strings.TrimRight(out, "\n") != c.printed:
			t.Errorf("%s:%d: `%s` no longer prints what the walkthrough transcribes.\n"+
				"Re-run it and replace the block with its output.\n--- transcribed\n%s\n--- printed\n%s",
				document, c.line, c.command, c.printed, strings.TrimRight(out, "\n"))
		}
	}
}

// readClaims is every transcript and sample in the walkthrough, with the
// fenced block that follows each.
func readClaims(t *testing.T) ([]string, []claim) {
	t.Helper()

	text, err := os.ReadFile(filepath.Join(root, document))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(text), "\n")

	var claims []claim

	for i, line := range lines {
		if !claimed.MatchString(line) {
			continue
		}

		match := transcribed.FindStringSubmatch(line)
		if match == nil {
			t.Errorf("%s:%d: %q claims a transcript and names no command to run", document, i+1, line)

			continue
		}

		start, end, ok := blockAfter(lines, i+1)
		if !ok {
			t.Errorf("%s:%d: `%s` is followed by no fenced block", document, i+1, match[2])

			continue
		}

		claims = append(claims, claim{
			line: i + 1, sample: match[1] == "sample", command: match[2],
			printed: strings.Join(lines[start:end], "\n"), start: start, end: end,
		})
	}

	return lines, claims
}

// blockAfter is where the body of the first fenced block at or after from
// starts and ends, as line indexes; the block must open within two lines.
func blockAfter(lines []string, from int) (int, int, bool) {
	for i := from; i < len(lines) && i <= from+2; i++ {
		if !strings.HasPrefix(lines[i], "```") {
			continue
		}

		for j := i + 1; j < len(lines); j++ {
			if lines[j] == "```" {
				return i + 1, j, true
			}
		}

		return 0, 0, false
	}

	return 0, 0, false
}

// rewrite replaces each transcript's block with what its command prints,
// from the last block to the first so earlier indexes stay true.
func rewrite(t *testing.T, bin string, lines []string, claims []claim) {
	t.Helper()

	for _, c := range slices.Backward(claims) {
		if c.sample {
			continue
		}

		out, err := run(t, bin, c.command)
		if err != nil {
			t.Fatalf("%s:%d: `%s` failed: %v\n%s", document, c.line, c.command, err, out)
		}

		printed := strings.Split(strings.TrimRight(out, "\n"), "\n")

		lines = slices.Replace(lines, c.start, c.end, printed...)
	}

	path := filepath.Join(root, document)

	err := os.WriteFile(path+".new", []byte(strings.Join(lines, "\n")), 0o600)
	if err == nil {
		err = os.Rename(path+".new", path)
	}

	if err != nil {
		t.Fatal(err)
	}
}

// buildCtchargen builds the command from this tree into a directory of its
// own, which run puts first on PATH.
func buildCtchargen(t *testing.T) string {
	t.Helper()

	bin := t.TempDir()

	//nolint:gosec // the output path is this test's own temporary directory
	build := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(bin, "ctchargen"), "./cmd/ctchargen")

	build.Dir = root

	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("building ctchargen: %v\n%s", err, out)
	}

	return bin
}

// run is command's standard output and standard error together, as a person
// running it in a terminal would read them.
func run(t *testing.T, bin, command string) (string, error) {
	t.Helper()

	// The command is the walkthrough's own text, run as the walkthrough says
	// it was: running it is the whole point of this test.
	shell := exec.CommandContext(t.Context(), "bash", "-c", "set -o pipefail; "+command) //nolint:gosec // see above

	shell.Dir = root

	shell.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		nested+"=1",
	)

	var out bytes.Buffer

	shell.Stdout = &out
	shell.Stderr = &out

	err := shell.Run()

	return out.String(), err
}
