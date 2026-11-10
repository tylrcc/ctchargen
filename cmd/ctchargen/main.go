// Command ctchargen generates rules-accurate Classic Traveller characters
// from Book 1 pp. 4-25, as held in the FFE reprint of the (c) 1977 text.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// errUsage reports a command line the tool cannot act on.
var errUsage = errors.New("usage")

// errSearch reports a run that was asked for correctly and could not deliver
// it. The command line was well formed; what failed was the search, and a
// reader told "usage" goes looking for a flag he typed wrong.
//
// The difference is in the message, not the exit status: every failure exits
// 1, which is what the README promises (#121).
var errSearch = errors.New("search")

func main() {
	err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ctchargen:", err)
	}

	os.Exit(exitStatus(err))
}

// exitStatus is 0 for a run that did what it was asked and 1 for one that did
// not. Those are the two the README promises (#121), and this is the one
// place that decides them.
func exitStatus(err error) int {
	if err == nil {
		return 0
	}

	return 1
}

// A source is where a run's dice come from, one roller per character. Nil is
// the engine's own: an unseeded stream, so no two characters are alike and
// none can be brought back (ADR 0001). A test hands in a scripted one through
// withDice, so that a run walks a known path; nothing else does.
type source func() chargen.Roller

// options is everything the command adds to a generation: the build that
// writes the note, and a roller of the source's own when one is given.
// Every generation the command makes takes these, so the build has one place
// to come from rather than a call at each generating site (#282).
// TestRunWritesTheNote holds a new note and a batch member to carrying it.
//
// Implements: The note names its ruleset and the build that wrote it [ADR
// 0002].
func (s source) options() []chargen.Option {
	options := []chargen.Option{chargen.WithBuild(build())}

	if s != nil {
		options = append(options, chargen.WithRoller(s()))
	}

	return options
}

// An option is a setting a caller of run may change. There is one.
type option func(*settings)

type settings struct{ dice source }

// withDice rolls every character of the run from the given source.
func withDice(dice source) option {
	return func(s *settings) { s.dice = dice }
}

// run takes the two output channels apart. `out` is the data channel - the
// note - and `asking` is where the tool talks to the person driving it.
// Interactive mode writes its questions to the second, so that
// `ctchargen new > kira.md` saves a note and not a conversation.
//
// Implements: The questions go to stderr, not into the note [Command line].
func run(args []string, in io.Reader, out, asking io.Writer, options ...option) error {
	var chosen settings

	for _, apply := range options {
		apply(&chosen)
	}

	if len(args) == 0 {
		return fmt.Errorf("%w: ctchargen %s; run `ctchargen --help`", errUsage, commandList)
	}

	// Help asked for is not a misuse: it leaves by out and exits 0. A bare
	// `ctchargen` stays an error, because nothing was asked for at all.
	// There is no `help <command>`; --help on the command is the one way in,
	// so there is one place a command's flags are described - and a word
	// after it is refused rather than dropped, as `new` refuses one.
	if args[0] == "help" || isHelpFlag(args[0]) {
		if len(args) > 1 {
			return fmt.Errorf("%w: %s; run `ctchargen <command> --help` for one command's flags",
				errUsage, helpTakesNoArguments(args[1]))
		}

		return writeTopLevelHelp(out)
	}

	switch args[0] {
	case "new":
		return newCharacter(args[1:], in, out, asking, chosen.dice)
	case "batch":
		return batch(args[1:], out, asking, chosen.dice)
	case "readings":
		return readings(args[1:], out)
	case "version":
		return version(args[1:], out)
	default:
		return fmt.Errorf("%w: unknown command %q; want %s; run `ctchargen --help`",
			errUsage, args[0], commandList)
	}
}

// version writes the build.
//
// It takes no flags, so --help is answered by answerHelp rather than by a
// flag set: the top-level help promises that every command answers --help,
// and version is one of those it lists. Anything else is refused rather than ignored,
// which is what the others do with a word they were not expecting.
func version(args []string, out io.Writer) error {
	if asked, err := answerHelp(args, out, versionUsage); asked {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("%w: %s; it takes no arguments, and was given %q",
			errUsage, versionUsage, args[0])
	}

	stamp := build()
	if stamp == "" {
		return fmt.Errorf("%w: no build information is embedded", errUsage)
	}

	_, err := fmt.Fprintln(out, stamp)
	if err != nil {
		return fmt.Errorf("writing the version: %w", err)
	}

	return nil
}

// build names this build, as the version subcommand reports it, or nothing
// when no build information is embedded.
func build() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	return info.Main.Path + " " + buildVersion(info)
}

func buildVersion(info *debug.BuildInfo) string {
	if info.Main.Version != "" {
		return info.Main.Version
	}

	return "(devel)"
}

// isHelpFlag reports the spellings of --help the flag package itself answers,
// for what has no flag set to answer them: the top level, version and
// readings.
func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "-help" || arg == "--help"
}
