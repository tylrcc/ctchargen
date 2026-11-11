package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// characterFlags are the flags both generating subcommands take: what a
// character is generated from, as opposed to what is done with it afterwards.
//
// Each command gives them its own descriptions, which legitimately differ;
// the names, types and defaults are declared once, in declareCharacterFlags,
// so the two cannot come to disagree about them. The strategies' defaults are
// read there from chargen.DefaultPolicy, so the help cannot disagree with what
// an auto run applies (#367). They are carried as one
// value from there on, because four adjacent strings in a signature are four
// strings that can be transposed, and a transposed strategy is one the record
// then reports as the one the operator asked for (#132).
type characterFlags struct {
	set                             *flag.FlagSet
	service, career, skills, muster *string
}

// characterHelp is what one command says about each of its characterFlags.
type characterHelp struct {
	service, career, skills, muster string
}

func declareCharacterFlags(set *flag.FlagSet, help characterHelp) characterFlags {
	defaults := chargen.DefaultPolicy()

	return characterFlags{
		set:     set,
		service: set.String("service", "", help.service+serviceChoices()),
		career:  set.String(careerFlag, defaults.Career.String(), help.career+chargen.CareerChoices()),
		skills:  set.String(skillsFlag, defaults.Skills.String(), help.skills+chargen.SkillsChoices()),
		muster:  set.String(musterFlag, defaults.Muster.String(), help.muster+chargen.MusterChoices()),
	}
}

// newFlagSet is new's flag set, declared apart from running the command so a
// test can walk the set it declares.
type newFlagSet struct {
	characterFlags

	auto, force *bool
	name        *string
	output      *string
}

func declareNewFlags() newFlagSet {
	set := flag.NewFlagSet("new", flag.ContinueOnError)
	set.SetOutput(io.Discard)

	flags := newFlagSet{
		characterFlags: declareCharacterFlags(set, characterHelp{
			service: "attempt enlistment in this service: ",
			career:  "the --auto career `strategy`: ",
			skills:  "the --auto skills `strategy`: ",
			muster:  "the --auto mustering out `strategy`: ",
		}),
		auto:   set.Bool("auto", false, "decide by the policy at every choice, rather than asking"),
		name:   set.String("name", "", "the character's name"),
		output: set.String("o", "", "write to this `file` rather than to standard output, which - also names"),
		force:  set.Bool("force", false, "overwrite the output file in place if it already exists"),
	}

	strictly(set)

	return flags
}

// newHelp is what new's help says in its own words (#563): a guided run asks
// for the service where an auto run draws it, and a death is written.
var newHelp = commandHelp{
	service: "--service is an enlistment attempt, not a guarantee: a failed throw is the\n" +
		"draft. Without it the player picks the service, or --auto draws it.\n",
	death: "A character can die in service; new writes the note all the same, and exits 0.\n",
}

// newCharacter generates one character and writes its note.
//
// --force overwrites an -o file, so without one, no -o or -o -, it has
// nothing to do and is refused rather than ignored (#473).
//
// Implements: --force without an -o file is refused [Command line].
func newCharacter(args []string, in io.Reader, out, asking io.Writer, dice source) error {
	flags := declareNewFlags()

	err := flags.set.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, newUsage, flags.set, newHelp)
	case err != nil:
		return fmt.Errorf("%w: %s; run `ctchargen new --help`", errUsage, flagMistake(err))
	}

	// `new` takes no positional argument, and a stray word is a typo that
	// would otherwise become an interactive session nobody asked for.
	// The flag package stops at the first non-flag word, so `new foo --auto`
	// leaves --auto unparsed as well: refusing the word catches both.
	if flags.set.NArg() > 0 {
		return fmt.Errorf("%w: %s; it takes no arguments, and was given %q",
			errUsage, newUsage, flags.set.Arg(0))
	}

	inputs, policy, err := flags.inputs()
	if err != nil {
		return err
	}

	err = checkName(*flags.name)
	if err != nil {
		return err
	}

	err = strategiesWithoutAuto(flags.set, *flags.auto)
	if err != nil {
		return err
	}

	inputs.Name = *flags.name

	output := toFile(*flags.output)
	if *flags.force && output == "" {
		return fmt.Errorf("%w: --force overwrites an -o file, and no -o file was given", errUsage)
	}

	return writeCharacter(inputs, policy, newRendering{
		auto:   *flags.auto,
		output: output, force: *flags.force,
	}, in, out, asking, dice)
}

// toFile is the file an -o names, or nothing where it names standard output:
// absent, or "-", the convention for it (#467).
//
// Implements: -o - is standard output [Command line].
func toFile(output string) string {
	if output == "-" {
		return ""
	}

	return output
}

// strategiesWithoutAuto refuses a strategy flag on a guided run. The
// strategies are the policy's, and on a guided run the player answers every
// choice the policy would have: a strategy accepted there would decide
// nothing and appear nowhere in the record. The check is on what the operator
// set, not on the values, because every strategy flag has a default.
//
// Implements: A strategy given without --auto is refused, not ignored
// [Command line].
func strategiesWithoutAuto(set *flag.FlagSet, auto bool) error {
	if auto {
		return nil
	}

	var given []string

	set.Visit(func(f *flag.Flag) {
		switch f.Name {
		case careerFlag, skillsFlag, musterFlag:
			given = append(given, "--"+f.Name)
		}
	})

	if len(given) > 0 {
		return fmt.Errorf("%w: %s %s the --auto policy's; without --auto the player answers every choice",
			errUsage, strings.Join(given, " and "), isOrAre(len(given)))
	}

	return nil
}

func isOrAre(n int) string {
	if n == 1 {
		return "is"
	}

	return "are"
}

// newRendering is what the flags say to do with the character once it exists,
// gathered so that newCharacter reads as the four steps it takes rather than
// as its whole flag set threaded through them.
type newRendering struct {
	auto, force bool
	output      string
}

func writeCharacter(inputs chargen.Inputs, policy chargen.Policy, how newRendering,
	in io.Reader, out, asking io.Writer, dice source,
) error {
	err := refusedBefore(how.output, how.force)
	if err != nil {
		return err
	}

	character, err := generate(inputs, policy, how.auto, in, asking, dice)
	if err != nil {
		return err
	}

	text := render.Note(character)

	where, err := openDestination(out, how.output, how.force)
	if err != nil {
		return err
	}

	return where.write(text)
}

// inputs is what the shared flags say to generate - all but the name, which
// is new's alone (#406) - and the policy the strategy flags name.
//
// This is the boundary the strategies are parsed at, and the only one: past
// here a Career is a Career, and no part of the engine asks again whether it
// is one of three. That is why chargen has no Validate any more (#41). The
// strategies go into the policy and nowhere else, so what an auto run's
// record credits to the policy is what those strategies decided.
func (f characterFlags) inputs() (chargen.Inputs, chargen.Policy, error) {
	policy, err := chargen.ParsePolicy(*f.career, *f.skills, *f.muster)
	if err != nil {
		return chargen.Inputs{}, policy, fmt.Errorf("%w: %w", errUsage, err)
	}

	var in chargen.Inputs

	if *f.service == "" {
		return in, policy, nil
	}

	for _, known := range traveller.ServiceNames {
		if strings.EqualFold(known.String(), *f.service) {
			in.Forced = chargen.ForceService(known)

			return in, policy, nil
		}
	}

	return in, policy, fmt.Errorf("%w: no service is called %q", errUsage, *f.service)
}

// generate runs the procedure as an auto run under the policy, or as a
// guided run that asks the player.
//
// Interactive mode watches the record as it is written, so the throws and
// their consequences reach the player between his questions; --auto passes
// no observer, because nobody is reading.
func generate(inputs chargen.Inputs, policy chargen.Policy, auto bool,
	in io.Reader, asking io.Writer, dice source,
) (*chargen.Character, error) {
	if auto {
		character, err := chargen.Auto(inputs, policy, dice.options()...)
		if err != nil {
			return nil, fmt.Errorf("generating: %w", err)
		}

		return character, nil
	}

	asked := newPlayer(in, asking)

	character, err := chargen.Guided(inputs, asked, append(dice.options(),
		chargen.WithObserver(asked.watch))...)
	if err != nil {
		return nil, fmt.Errorf("generating: %w", err)
	}

	return character, nil
}

// checkName refuses a --name the note cannot carry as typed (#128). A name is
// one line: the note prints it as its heading, so a line break in it starts
// sections the character does not have, and markdown ends a line at a
// carriage return too. And it is text: the note writes it into its
// properties as a YAML string, which reads a byte that is not UTF-8 back as
// another letter, so the name a vault saw would not be the one typed.
func checkName(name string) error {
	if strings.ContainsAny(name, "\n\r") {
		return fmt.Errorf("%w: --name %q is more than one line", errUsage, name)
	}

	if !utf8.ValidString(name) {
		return fmt.Errorf("%w: --name %q is not UTF-8 text", errUsage, name)
	}

	return nil
}

// errNoFolder reports an -o whose folder is missing, or is not a folder.
var errNoFolder = errors.New("is not a folder that exists")

// refusedBefore refuses an -o the note could never be written to, before
// anyone is generated (#424): a path whose folder is missing or is not a
// folder, a path already there without --force, and a directory, which
// --force cannot overwrite either. Refused only at the end, a guided session
// lost every answer the player had given. Without --force, the exclusive open
// that writes the note still refuses a file that appears during the session.
// Anything else in the way is the operating system's to refuse, at the open.
//
// None of these is a usage error: the command line was well formed, and what
// stands in the way is on disk (#472).
//
// Implements: A guided session onto an existing -o is refused before it asks
// [Command line].
func refusedBefore(path string, force bool) error {
	if path == "" {
		return nil
	}

	folder := filepath.Dir(path)
	if !isDirectory(folder) {
		return fmt.Errorf("%s %w, so -o %s cannot be written", folder, errNoFolder, path)
	}

	_, err := os.Lstat(path)
	if err == nil && (!force || isDirectory(path)) {
		// inTheWay names a directory as one, with no --force advice (#254).
		return inTheWay(path)
	}

	return nil
}
