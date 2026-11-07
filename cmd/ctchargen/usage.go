package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// commandList is the subcommands in the order run's switch takes them. Both
// messages that name them read it here, so the two cannot come to disagree
// about the order again.
const commandList = "new|batch|render|version"

// The usage line of each subcommand: what it takes that is not a flag.
//
// Each is the shape alone: errUsage prints the word "usage" ahead of an
// error, and writeHelp prints "usage: ctchargen " ahead of the help.
//
// None of them enumerates its flags. An enumeration typed beside a flag set
// is a line that goes stale the first time a flag is added; the list
// writeFlags prints from the set itself is what carries the detail, and it
// cannot go stale.
const (
	newUsage     = "new [flags]"
	batchUsage   = "batch --auto --count N [flags]"
	renderUsage  = "render [flags] <character.json>"
	versionUsage = "version"
)

// topLevelUsage answers `ctchargen --help`: what the commands are, and where
// to ask about one of them.
const topLevelUsage = "usage: ctchargen <command> [flags]\n" +
	"\n" +
	"  new      generate one character, asking at every choice unless --auto\n" +
	"  batch    generate many characters from one base seed, under --auto\n" +
	"  render   write a record saved earlier as a sheet or as a transcript\n" +
	"  version  write the build\n" +
	"\n" +
	"Run `ctchargen <command> --help` for that command's flags.\n"

// The three strategy flags, named once, because `new` and `batch` each
// define all three and a name typed six times is a name that can be typed
// wrong once.
const (
	careerFlag = "career"
	skillsFlag = "skills"
	musterFlag = "muster"
)

// seedFlag is named once too, because `new` and `batch` each define it and
// then ask the set whether it was given.
const seedFlag = "seed"

// answersFlag is named here because the flag is defined in one file and the
// usage errors that name it are in another, and a flag the tool refuses under
// one spelling and reads under another would be worse than none.
const answersFlag = "answers"

// shortFlag is the length of a flag name written with one dash. There is one
// of them, `-o`, and it is written that way everywhere else.
const shortFlag = 1

// writeTopLevelHelp answers --help before a subcommand is named.
func writeTopLevelHelp(out io.Writer) error {
	return writeUsageText(out, topLevelUsage)
}

// writeVersionUsage answers `ctchargen version --help`. `version` has no flag
// set, so it cannot go through writeHelp: that would print the usage line, a
// blank line, and then nothing at all.
func writeVersionUsage(out io.Writer) error {
	return writeUsageText(out, "usage: ctchargen "+versionUsage+"\n")
}

// writeUsageText writes a usage that is already composed.
func writeUsageText(out io.Writer, text string) error {
	_, err := io.WriteString(out, text)
	if err != nil {
		return fmt.Errorf("writing the usage: %w", err)
	}

	return nil
}

// writeHelp answers --help on a subcommand: its usage line, then its flags.
//
// It writes to out and returns nil, so that help asked for leaves by the
// same door as the record and exits 0 - not to `asking`, which two of the
// three subcommands are never handed, and which would put `new --help | less`
// on the wrong stream. It cannot reach a `-o` file: parsing happens before
// openDestination, and a run that comes here never opens one.
//
// A word after a help flag is refused, not discarded, the same as everywhere
// else a command meets an unexpected word (#69): `new --help junk`, `batch
// --help junk`, `render --help junk` and `version --help junk` refuse it as
// `ctchargen help junk` does (#84). The flag package returns ErrHelp before a
// command reaches its own NArg check, so the word has to be looked at here.
//
// Nobody loses a character to it. It is consistency, and the kind of thing a
// referee finds by typing.
func writeHelp(out io.Writer, line string, flags *flag.FlagSet) error {
	if flags.NArg() > 0 {
		return fmt.Errorf("%w: %s; help takes no arguments, and was given %q",
			errUsage, line, flags.Arg(0))
	}

	_, err := fmt.Fprintf(out, "usage: ctchargen %s\n\n", line)
	if err != nil {
		return fmt.Errorf("writing the usage: %w", err)
	}

	return writeFlags(out, flags)
}

// helpLine is one flag as the help prints it: the name and kind on the left,
// the description on the right.
type helpLine struct{ named, description string }

// writeFlags lists a command's flags, with their descriptions and defaults.
//
// It exists rather than flag.PrintDefaults because PrintDefaults writes one
// dash and everything else here writes two: the README, the error messages,
// and the referee's own transcript. The defaults are printed because a
// strategy flag left alone still chooses, and the reader who could not see
// which one it chose is the one who reported this.
//
// The description column is measured from the set rather than typed as a
// constant, so a flag with a longer name than any here arrives aligned
// instead of pushing one line out of the column nothing checks.
func writeFlags(out io.Writer, flags *flag.FlagSet) error {
	var (
		lines []helpLine
		width int
	)

	flags.VisitAll(func(each *flag.Flag) {
		kind, description := flag.UnquoteUsage(each)

		named := "  " + dashed(each.Name)
		if kind != "" {
			named += " " + kind
		}

		if !zeroDefault(each.DefValue) {
			description += " (default " + each.DefValue + ")"
		}

		// A column one wider than the longest name, so that even the
		// longest line has two spaces before its description rather than
		// running into it.
		width = max(width, len(named)+1)
		lines = append(lines, helpLine{named: named, description: description})
	})

	for _, line := range lines {
		_, err := fmt.Fprintf(out, "%-*s %s\n", width, line.named, line.description)
		if err != nil {
			return fmt.Errorf("writing the flags: %w", err)
		}
	}

	return nil
}

// dashed writes a flag the way this command line writes it: -o, --seed. The
// flag package accepts either spelling for either name; this is about what
// the reader is shown.
func dashed(name string) string {
	if len(name) == shortFlag {
		return "-" + name
	}

	return "--" + name
}

// zeroDefault reports a default not worth printing. The flag package's own
// isZeroValue is unexported; these three cover every flag this command line
// has, and a flag whose default is worth reading is never one of them.
func zeroDefault(value string) bool {
	return value == "" || value == "0" || value == "false"
}

// serviceChoices is the --service flag's own description: the six of p. 10,
// in the book's order, spelled the way they are typed.
func serviceChoices() string {
	names := make([]string, 0, len(traveller.ServiceNames))

	for _, service := range traveller.ServiceNames {
		names = append(names, strings.ToLower(service.String()))
	}

	return strings.Join(names, ", ")
}
