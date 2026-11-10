# Walkthrough

> Release snapshot for `v1.0.0-rc.5`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots"). `THEORY.md` explains why the system is shaped this way; this
> document follows one character through the code.

## Overview

`ctchargen` is a Go command that generates Classic Traveller characters by
walking Book 1's procedure (pp. 4–25 of the 1977 text). It writes each one as
a markdown **note** (ADR 0006): properties a vault can query, a heading and a
summary a referee can read aloud, the sheet, and the **generation record** of
every die thrown and every choice made. The note ends with the record. It is
the only way back to a character, and nothing reads one back. There are three
subcommands:

- `new`: one character, answered by the policy (an **auto run**, `--auto`) or
  by a person at the terminal (a **guided run**);
- `batch`: many auto runs, each written as its own note into a folder of the
  batch's own, or one after another to standard output;
- `version`: the build.

There are no dependencies beyond the standard library. The rules tables are Go
literals typed as the page prints them, and they are **lifted** into the
domain's values once per process (ADR 0005). The **readings**, the decisions
taken where the page is silent, are Go values compiled in beside them. They
are the authority's record, not the referee's (ADR 0007). The engine stamps
each erratum on the character it governed, and the tests hold every stamp to
its reading's condition, but no note names an erratum.

Hold one idea through everything below: the generation record is the
character. Every outcome that a sheet reads carries a typed `traveller.Fact`.
Everything the note says about the character is folded out of those facts:
the UPP, the skills, the possessions, the service record, the age, and how the
service ended. While the engine runs, it keeps its own working copy of the
character, because every DM reads it. The finished `Character` holds only its
record and what it was given.

## Architecture

```text
cmd/ctchargen/        the subcommands; stdout carries the note, stderr the conversation
internal/traveller/   the domain: alphabets, values, the sums, events and facts (imports nothing here)
internal/rules/       every table of pp. 4–25, transcribed in Go and lifted once, and the readings as values
internal/dice/        an unseeded stream of dice
internal/chargen/     the procedure, the Decider it asks, the auto run's Policy, and the folds that read the record
internal/render/      the note: properties, summary, sheet, generation record
internal/scripted/    test support: the strict roller that fixtures and walks play
internal/coverage/    test code only: holds the Implements: and Holds: markers to each other
internal/mutate/      a tool: runs a command against a scratch copy with one edit made
```

The imports point one way. `traveller` and `dice` import none of the others.
`rules` imports `traveller`, and `chargen` imports `traveller`, `rules` and
`dice`. `render` imports `chargen` for the character and `traveller` for its
values. It does not import `rules`, since a note names no reading, and a
depguard rule forbids that import. Data flows in a line. The flags become
`chargen.Inputs` and a `Policy`, or a person answers instead. `chargen` walks
the procedure against `rules` and a roller, and hands back a
`*chargen.Character` whose record is the whole of it. `render` folds that
record into the note.

This document follows the character in the golden
`internal/chargen/testdata/drafted-then-commissioned.md`: the run that
`ctchargen new --auto --service navy` makes under the default policy, played
from a scripted roller. The Navy rejects the enlistment and the draft puts the
character in the Merchants. The character is commissioned and promoted in the
second term. After the fourth term the reenlistment throw fails, and the
character is denied reenlistment and musters out with cash and two Low
Passages.

## 1. The command

`main` hands everything to `run`, and maps the result to the exit status the
README promises: 0 or 1.

`cmd/ctchargen/main.go` — `main`

```go
func main() {
	err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ctchargen:", err)
	}

	os.Exit(exitStatus(err))
}
```

`run` keeps the two output channels apart. `out` carries the note, and
`asking` is where the tool talks to a person, so `ctchargen new > kira.md`
saves a note and not a conversation. After answering help, `run` dispatches on
the subcommand. The `readings` subcommand is gone (#460, ADR 0007), so
`readings` is now an unknown command like any other.

`cmd/ctchargen/main.go` — `run`

```go
	switch args[0] {
	case "new":
		return newCharacter(args[1:], in, out, asking, chosen.dice)
	case "batch":
		return batch(args[1:], out, asking, chosen.dice)
	case "version":
		return version(args[1:], out)
	default:
		return fmt.Errorf("%w: unknown command %q; want %s; run `ctchargen --help`",
			errUsage, args[0], commandList)
	}
```

`chosen.dice` is a `source`. In a real run it is nil, and the engine rolls its
own unseeded stream. It is a scripted roller only when a test passes
`withDice`. Every generation the command makes takes the same options, so the
build has one place to come from (ADR 0002).

`cmd/ctchargen/main.go` — `source.options`

```go
func (s source) options() []chargen.Option {
	options := []chargen.Option{chargen.WithBuild(noteBuild())}

	if s != nil {
		options = append(options, chargen.WithRoller(s()))
	}

	return options
}
```

`noteBuild` is the version alone, which the note writes as its `ctchargen`
property. `version` prints `build`, which puts the module path in front of
the same version (#476).

`cmd/ctchargen/main.go` — `noteBuild`

```go
func noteBuild() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	return buildVersion(info)
}
```

Errors fall into two kinds, and only the message tells them apart, since
both exit 1. `errUsage` is a mistake in the command line: an unknown flag, a
value the flag cannot take, or a word where none belongs. A refusal of
something found at run time is not one: a file already in the way, a folder
that is missing or not empty, a failed write, or a batch's search that gives
up (#472, #508).

## 2. `new`: from flags to a generation

`new` and `batch` share the flags that a character is generated from:
`--service` and the three strategies. These are declared once, in
`declareCharacterFlags`, which reads the strategy defaults from
`chargen.DefaultPolicy`, so the help cannot disagree with what an auto run
applies. `new` adds `--auto`, `--name`, `-o` and `--force`. Then `strictly`
wraps every flag in the set.

`cmd/ctchargen/new.go` — `declareNewFlags`

```go
		output: set.String("o", "", "write to this `file` rather than to standard output, which - also names"),
		force:  set.Bool("force", false, "overwrite the output file in place if it already exists"),
	}

	strictly(set)
```

The wrapper is new since rc.4 (#466, #473). A flag given twice, or given an
empty value, used to be read one way without a word: the flag package took
the last value, an empty `--service` left the service to the policy, and an
empty `-o` wrote to standard output. Now each flag can be set once, and only
to something. `strictBool` keeps a switch such as
`--auto` a switch, which needs no value after it.

`cmd/ctchargen/usage.go` — `strictValue.Set`

```go
func (v *strictValue) Set(value string) error {
	switch {
	case v.given:
		return errGivenTwice
	case value == "":
		return errEmptyValue
	}

	v.given = true
```

Once every flag is wrapped, the flag package would call every kind "value",
because it names a kind from the value's concrete type. `kindOf` reads the
kind from what the wrapped value holds instead. A word that a description
puts in backquotes still names the kind, which is how the help shows
`-o file` for `new` and `-o folder` for `batch` (#517).

`cmd/ctchargen/usage.go` — `kindOf`

```go
func kindOf(declared *flag.Flag, named string) string {
	getter, ok := declared.Value.(flag.Getter)
	if named != "value" || !ok {
		return named
	}

	return fmt.Sprintf("%T", getter.Get())
}
```

The strategies are parsed at this boundary and nowhere else. A `--service`
word becomes a `chargen.ForcedService`, one value that is either a service or
nothing, so the zero `ServiceName`, which is Navy, cannot stand in for "none".

`cmd/ctchargen/new.go` — `characterFlags.inputs`

```go
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
```

Before anyone is asked anything, `newCharacter` refuses what it can. It
refuses a stray positional word. It refuses a `--name` that is not one line of
UTF-8, since the note writes the name as a heading and as a YAML string. And
it refuses a strategy flag without `--auto`, which would decide nothing on a
guided run. Then it reads `-o`. A `-` names standard output, the usual
convention (#467). `--force` with no file to overwrite is now refused rather
than ignored (#473).

`cmd/ctchargen/new.go` — `newCharacter`

```go
	output := toFile(*flags.output)
	if *flags.force && output == "" {
		return fmt.Errorf("%w: --force overwrites an -o file, and no -o file was given", errUsage)
	}
```

`cmd/ctchargen/new.go` — `toFile`

```go
func toFile(output string) string {
	if output == "-" {
		return ""
	}

	return output
}
```

What is left is four steps: refuse a path the note could never be written to,
generate, render, and write.

`cmd/ctchargen/new.go` — `writeCharacter`

```go
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
```

`refusedBefore` checks the path before a guided session asks its first
question (#424). The path's folder must exist. The path itself must be absent
unless `--force` is given, and it must never be a directory. What it refuses
is on disk, so it is not a usage error (#472).

`cmd/ctchargen/new.go` — `refusedBefore`

```go
	folder := filepath.Dir(path)
	if !isDirectory(folder) {
		return fmt.Errorf("%s %w, so -o %s cannot be written", folder, errNoFolder, path)
	}

	_, err := os.Lstat(path)
	if err == nil && (!force || isDirectory(path)) {
		// inTheWay names a directory as one, with no --force advice (#254).
		return inTheWay(path)
	}
```

The check is not the last word. Without `--force`, the note is written
through an exclusive create, which refuses a file that appeared during the
session. With `--force`, the file is opened once and truncated, then written
in place, as the shell's `>` writes (#469). A symlink is followed and the
file keeps its mode, with no code of the tool's own to do it.

`cmd/ctchargen/output.go` — `openDestination`

```go
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
```

A write that fails without `--force` removes the note. The exclusive create
made the file this run's own, and a truncated note left under its name would
make the rerun refuse the name (#468).

`cmd/ctchargen/output.go` — `destination.write`

```go
	if d.exclusive {
		_ = os.Remove(d.path)
	}
```

The following is a transcript of commands run against a build of this tree,
from a scratch folder (nothing re-runs it). Every refusal exits 1, and only a
command-line mistake says "usage".

```text
$ ctchargen new --auto -o crew/x/y.md
ctchargen: crew/x is not a folder that exists, so -o crew/x/y.md cannot be written
$ touch kira.md; ctchargen new -o kira.md
ctchargen: kira.md already exists; pass --force to overwrite it
$ ctchargen new --auto --force -o -
ctchargen: usage: --force overwrites an -o file, and no -o file was given
$ ctchargen new --auto --service navy --service army
ctchargen: usage: invalid value "army" for flag --service: given twice; run `ctchargen new --help`
$ ctchargen new --career retire
ctchargen: usage: --career is the --auto policy's; without --auto the player answers every choice
$ ctchargen new --auto --readings
ctchargen: usage: flag provided but not defined: --readings; run `ctchargen new --help`
```

The second command is a guided run, refused before its first question.

`generate` is where the two ways in part. An auto run hands the policy to the
engine. A guided run builds a `player`, which also watches the record as it is
written, so the throws reach the person between questions.

`cmd/ctchargen/new.go` — `generate`

```go
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
```

## 3. The engine's two ways in

`chargen` has two exported entry points, and which one was called is fixed
before anything is asked. The `*Policy` is the whole of what marks an auto
run. `Guided` passes nil, so the record can never credit a choice to a policy
that did not answer.

`internal/chargen/generate.go` — `Auto`

```go
func Auto(in Inputs, policy Policy, options ...Option) (*Character, error) {
	// The pointer is what marks an auto run: Guided passes nil.
	return generate(in, policy, &policy, options)
}
```

`internal/chargen/generate.go` — `Guided`

```go
func Guided(in Inputs, decider Decider, options ...Option) (*Character, error) {
	return generate(in, decider, nil, options)
}
```

`generate` loads the rules, builds a `run`, and applies the options. It wraps
the decider in the `logging` gate (section 6), then walks the procedure. When
the walk ends, the character is handed its events and its stamped errata, and
nothing else. No step copies a profile, a skill list or a career onto it.

`internal/chargen/generate.go` — `generate`

```go
	r := &run{
		tables: tables,
		roll:   dice.New(),
		record: newRecord(),
		char:   &Character{Inputs: in, Policy: policy},
		atAge:  traveller.NewAge(traveller.StartingAge),
	}

	for _, option := range options {
		option(r)
	}

	r.decide = logging{to: decider, into: r.record}

	err = r.generate()
	if err != nil {
		return nil, err
	}

	r.char.Events = r.record.events
	r.char.Errata = r.record.stamped()

	return r.char, nil
```

The character lives in the `run` while it is made. The fields of `run` are
working state, and their comments say so. Each change to them is recorded as
a fact carrying the value it left, and the finished character is read from
those facts (#441).

`internal/chargen/generate.go` — `run`

```go
	// enlistment, terms, rank, atAge and departure are the career so far,
	// which the commission and promotion throws, mustering out, the pension
	// and the check for death read mid-run, and from which each age is
	// advanced. They are working state, as profile is: each change to them is
	// recorded as a fact carrying the value it left, or for terms as the step
	// that opens one, and the finished Character's are read from the record
	// (#441).
	enlistment traveller.Enlistment
	terms      int
	rank       traveller.Rank
	atAge      traveller.Age
	departure  traveller.Departure
```

`run.generate` is the procedure's outline. It rolls the six characteristics
and enlists. Then the character either stops, as a civilian assessed for a
title, or serves. A character that lives musters out and takes the pension,
and every character is assessed for a title at the end.

`internal/chargen/generate.go` — `run.generate`

```go
func (r *run) generate() error {
	r.rollProfile()

	err := r.enlist()
	if err != nil {
		return err
	}

	_, served := servedIn(r.enlistment)
	if !served {
		return r.assessTitle()
	}

	err = r.serve()
	if err != nil {
		return err
	}

	if !r.dead() {
		err := r.musterOut()
		if err != nil {
			return err
		}

		r.pension()
	}

	return r.assessTitle()
}
```

## 4. The record as it is written

The generation record is a `record`: an append-only list of
`traveller.Event`, and the set of errata that governed the run. `Event` is a
sealed sum of five kinds: a step entered, a throw against a target, a roll
with nothing to meet, a choice, and an outcome. Each event is numbered by its
place in the list, and each is shown to a watcher, if there is one, as it is
appended.

`internal/chargen/record.go` — `record`

```go
type record struct {
	events []traveller.Event
	errata map[traveller.Erratum]bool

	// watch is called with each event as it is appended, or nil when nobody
	// is reading. It is how interactive mode shows the procedure between
	// its questions. It is handed the event the record holds, so a write
	// through the event's slices reaches this record and no other.
	watch func(traveller.Event)
}
```

An outcome is where a fact rides. `record.fact` writes an `OutcomeEvent`
whose prose is the fact's own `String`, so the line a referee reads and the
value a fold reads cannot say different things (#412). The errata it is handed
go into the record's set and nowhere else. An outcome carries no errata since
ADR 0007, because no note names them.

`internal/chargen/record.go` — `record.fact`

```go
func (r *record) fact(because int, errata []traveller.Erratum, fact traveller.Fact) {
	for _, e := range errata {
		r.errata[e] = true
	}

	r.add(traveller.OutcomeEvent{
		Seq: r.next(), Because: because, Description: fact.String(), Fact: fact,
	})
}
```

Its sibling `record.outcomef` writes an outcome with prose and no fact. It is
for outcomes that nothing is read from: "survived term 1", "not promoted this
term", an aging throw that holds, a second Travellers' Aid wasted, a second
scout ship lost, a dash row. Both stamp the errata they are handed. Later,
`stamped` returns every erratum that any outcome stamped, in id order, and
that list becomes `Character.Errata`.

`traveller.Fact` is a sealed sum like the others: a `Fold` over a `FactCases`
interface with one method per kind. A new kind of fact stops every fold
compiling until the fold handles it.

`internal/traveller/fact.go` — `FactCases`

```go
type FactCases interface {
	Applied(f AppliedFact) error
	WeaponPicked(f WeaponPickedFact) error
	Raised(f RaisedFact) error
	Aged(f AgedFact) error
	Recovered(f RecoveredFact) error
	Enlisted(f EnlistedFact) error
	Rolled(f RolledFact) error
...
	Started(f StartedFact) error
	Drafted(f DraftedFact) error
	Declined(f DeclinedFact) error
	Died(f DiedFact) error
	DiedOfCrisis(f DiedOfCrisisFact) error
	Retained(f RetainedFact) error
	Reenlisted(f ReenlistedFact) error
	Retired(f RetiredFact) error
	DeniedReenlistment(f DeniedReenlistmentFact) error
	Discharged(f DischargedFact) error
}
```

There are twenty-six. Every one follows the same pattern. The fact carries
the value the engine reached, not the change that reached it, so a fold only
assigns. A skill fact carries the level the skill reached, a cash fact the
running total, and a reenlistment fact the age at which the term left the
character. `DeniedReenlistmentFact` was `ForcedOutFact` until #533: the page's
own word is "reenlistment has been denied" (p. 6, #485).

## 5. The procedure, one character through it

### The six rolls

`rollProfile` throws two dice per characteristic, in p. 4's order. Each
characteristic is a `RollEvent` and a `RolledFact`. After the six, a
`StartedFact` writes the rolled UPP and the age every character starts at.

`internal/chargen/generate.go` — `run.rollProfile`

```go
func (r *run) rollProfile() {
	r.record.step("characteristics", "p. 4")

	for _, c := range traveller.Characteristics {
		first, second := twoDice(r.roll)

		r.profile[c] = first + second

		if c == traveller.SocialStanding {
			r.rolledSocial = r.profile[c]
		}

		seq := r.record.dice(c.String(), first, second)
		r.record.fact(seq, nil, traveller.RolledFact{Characteristic: c, Value: r.profile[c]})
	}

	r.record.fact(0, nil, traveller.StartedFact{Profile: r.profile, Age: r.atAge})
}
```

In the note, each outcome names the event that caused it as `(from step N)`.
The form was `(from N)` until #487, and it read like a characteristic that had
been N.

`internal/chargen/testdata/drafted-then-commissioned.md` — `characteristics`

```markdown
**1. characteristics (p. 4)**

2. Strength: rolled 6+1
3. Strength 7 (from step 2)
4. Dexterity: rolled 5+1
5. Dexterity 6 (from step 4)
6. Endurance: rolled 2+2
7. Endurance 4 (from step 6)
8. Intelligence: rolled 5+2
9. Intelligence 7 (from step 8)
10. Education: rolled 3+4
11. Education 7 (from step 10)
12. Social Standing: rolled 4+3
13. Social Standing 7 (from step 12)
14. UPP 764777 at age 18
```

`rolledSocial` is kept beside the profile because E011, the title reading,
turns on whether the Social Standing at the end differs from the one rolled.

### Enlistment and the draft

`--service navy` forced the attempt, so `chooseService` returns the Navy
without asking. Unforced, it would build one `EnlistmentOffer` per service,
each carrying its odds, and ask the decider. On an auto run the decider draws
the service (#420, section 6). The throw is made either way, and its verdict
is an `EnlistedFact`, accepted or not. The service attempted is read from
that fact afterwards (#397).

`internal/chargen/enlist.go` — `run.enlist`

```go
	throw := r.against(service.Enlistment)
	seq := r.record.throw("enlistment, "+name.String(), throw)

	if throw.succeeded {
		r.record.fact(seq, nil, traveller.EnlistedFact{Service: name, Accepted: true})

		return r.join(traveller.Enlisted{Service: name})
	}

	r.record.fact(seq, nil, traveller.EnlistedFact{Service: name})

	return r.draft()
```

Every throw that the page prints with characteristic DMs is made through
`against`. Since #487, `against` keeps beside the DM which prerequisites
earned it, so the record can show where the number came from.

`internal/chargen/serve.go` — `run.against`

```go
func (r *run) against(printed rules.Throw) throw {
	made := roll(r.roll, printed.Target, printed.Modifier(r.profile))

	made.from = printed.Met(r.profile)

	return made
}
```

The roll of 5 against the Navy's 8+ misses, and the character's
characteristics earn no Navy DM. P. 5 prints the draft both ways, "may" and
"must". E001 reads "may", so the draft is a question.

`internal/chargen/enlist.go` — `run.draft`

```go
	submit, err := r.decide.SubmitToDraft()
	if err != nil {
		return fmt.Errorf("submitting to the draft: %w", err)
	}

	if !submit {
		r.enlistment = traveller.DeclinedTheDraft{}
		r.record.fact(0, []traveller.Erratum{traveller.E001}, traveller.DeclinedFact{})

		return nil
	}

	face := r.roll.Die()

	name, err := r.tables.Draft(face)
	if err != nil {
		return fmt.Errorf("the draft: %w", err)
	}

	seq := r.record.dice("draft", face)
	r.record.fact(seq, []traveller.Erratum{traveller.E001}, traveller.DraftedFact{Service: name})

	return r.join(traveller.Drafted{Service: name})
```

E001 is stamped on the character, but no line of the note shows it. The record
reads "rejected by the Navy", with no pronoun: a character has no sex in these
rules (p. 8), and no note names one (#459).

`internal/chargen/testdata/drafted-then-commissioned.md` — `enlistment`

```markdown
**15. enlistment (pp. 5, 10)**

16. enlistment, Navy: rolled 2+3 = 5 against 8+, missed
17. rejected by the Navy (from step 16)
18. submit to the draft (p. 5): policy chose yes from yes, no
19. draft: rolled 5
20. drafted into the Merchants (from step 19)
```

`join` enters the service. It takes what p. 23's Rank and Service Skills box
grants on entering, once (E005), and applies each grant as an `AppliedFact`
whose source is `OnEntering`. The box prints nothing for the Merchants, so no
step appears here. A Marine would get Cutlass-1 under a "rank and service
skills" step.

### A term

`serve` runs terms until one ends the service. `term` follows the
exposition's order (E002): survival, commission, promotion, skills,
reenlistment, and then the aging round (E006). It opens with `record.opens`,
a step that carries the term's number in `StepEvent.Opens`. That field is how
the finished character counts its terms, the fatal one included.

`internal/chargen/serve.go` — `run.term`

```go
func (r *run) term(term traveller.Term) (bool, error) {
	r.record.opens(term, "pp. 5-7")

	r.terms = int(term)

	if !r.survive(term) {
		return true, nil
	}
...
	r.grantEligibility(term)

	err = r.trainSkills()
	if err != nil {
		return false, err
	}
...
	r.atAge = r.atAge.PlusYears(traveller.Years)

	intent, err := r.reenlist(term)
	if err != nil {
		return false, err
	}

	err = r.agingRound(term)
	if err != nil {
		return false, err
	}

	if r.dead() {
		return true, nil
	}

	if intent == traveller.Continue {
		return false, nil
	}

	return true, nil
}
```

The term's four years are counted before the reenlistment throw, whose
outcome carries the age. A failed survival throw is death (p. 5). The fatal
term's four years are counted as well (E004), and the `DiedFact` carries both
the term and the age.

`internal/chargen/serve.go` — `run.survive`

```go
	r.atAge = r.atAge.PlusYears(traveller.Years)
	r.departure = traveller.KilledBySurvivalThrow{}
	r.record.fact(seq, []traveller.Erratum{traveller.E004}, traveller.DiedFact{Term: term, Age: r.atAge})
```

The draftee survives. In term 1, `commission` asks nothing, because p. 5 bars
a draftee from a commission in the first term only. `r.drafted()` reads that
off the enlistment.

`internal/chargen/rank.go` — `run.commission`

```go
	throw, ok := r.service.Commission()
	if !ok || r.rank.Commissioned() || (r.drafted() && term == 1) {
		return nil
	}
```

In term 2 the draftee is asked about a commission, makes the throw, and is
commissioned. The promotion comes in the same term, which p. 6 allows. Each
question shows the decider its `Odds`: the target, the DM, and the
prerequisites that earn it (#482). `raise` sets the working rank and records a
`RaisedFact` with the rank's number and its name. It applies any rank-keyed
grants, each sourced `AtRank`, and tallies the skill eligibility the rank
carries under how the rank came. That tally is how a guided run's skills
question can say why the term has the rolls it has (#486).

`internal/chargen/rank.go` — `run.raise`

```go
func (r *run) raise(seq int, conferred traveller.Rank, how traveller.Advancement) error {
	r.rank = conferred

	switch how {
	case traveller.Commissioned:
		r.eligible.Commission += r.tables.Eligibility().OnCommission
	case traveller.Promoted:
		r.eligible.Promotion += r.tables.Eligibility().OnPromotion
	}

	r.record.fact(seq, nil, traveller.RaisedFact{How: how, Rank: conferred, Name: r.rankName()})
```

At the top of a service's rank column no promotion throw is made at all
(E013). The outcome says so, and the decider is not asked.

### Skills

`grantEligibility` adds the term's own rolls to the tally: two for the first
term and one after. The commission and promotion have already added theirs.
`trainSkills` takes the tally whole, clears it, and spends it one roll at a
time, telling each roll which of how many it is.

`internal/chargen/serve.go` — `run.trainSkills`

```go
	earned := r.eligible

	r.eligible = traveller.Earned{}

	if earned.Rolls() == 0 {
		return nil
	}

	r.record.step("skills and training", "pp. 6, 11")

	for n := 1; n <= earned.Rolls(); n++ {
		err := r.trainOnce(traveller.SkillRoll{N: n, Of: earned})
```

`trainOnce` asks for the table before the die, as p. 11 requires. Each offer
carries its table, how many results the character has already taken off it,
and its six results from the service's column. The policy reads the count,
and a person reads the results.

`internal/chargen/serve.go` — `run.trainOnce`

```go
	offered := make([]traveller.SkillTableOffer, 0, len(tables))
	for _, table := range tables {
		offered = append(offered, traveller.SkillTableOffer{
			Table:   table,
			Taken:   r.trained[table],
			Results: slices.Clone(r.service.Skills[table][:]),
		})
	}

	table, err := r.decide.SkillTable(offered, roll)
	if err != nil {
		return fmt.Errorf("designating a skills table: %w", err)
	}
...
	face := r.roll.Die()

	result, err := r.service.Result(table, face)
	if err != nil {
		return fmt.Errorf("consulting %v: %w", table, err)
	}

	seq := r.record.dice(table.String(), face)

	return r.apply(result, seq, []traveller.Erratum{traveller.E002}, traveller.InTerm)
```

`apply` folds the `TableResult` sum, p. 12's three kinds, through
`applyResult`. A skill raises the working list and records the level it
reached. A characteristic change records both ends and the table's own delta,
so a change that the floor or ceiling held is not read as a change of nothing
(#409).

`internal/chargen/enlist.go` — `applyResult.Skill`

```go
func (a *applyResult) Skill(name traveller.SkillName) error {
	level := a.run.addSkill(name)
	a.run.record.fact(a.because, a.errata, traveller.AppliedFact{
		Gain: traveller.SkillGain{Skill: traveller.Skill{Name: name, Level: level}}, Source: a.source,
	})

	return nil
}
```

A Blade Cbt or Gun Cbt result names a weapon at once. The list is the page's.
The decider is handed a `Vary` to draw from, which is the run's own dice and
nothing more, because the page gives no basis to choose.

`internal/chargen/enlist.go` — `applyResult.WeaponPick`

```go
	chosen, err := a.run.decide.Weapon(category, list, varyOnly{a.run.roll})
	if err != nil {
		return fmt.Errorf("choosing a weapon: %w", err)
	}

	level := a.run.addSkill(traveller.SkillName(chosen))
	a.run.record.fact(a.because, a.errata, traveller.WeaponPickedFact{
		Category: category, Skill: traveller.Skill{Name: traveller.SkillName(chosen), Level: level}, Source: a.source,
	})
```

Here is term 2 in the note. Each DM names the prerequisite that earned it.
The Merchants' survival DM is p. 10's "Intel 7+", written in the domain's
words. The three tables follow the policy's rule of the least-trained table
first. After term 1, nothing has been taken off Personal Development, so it
comes first. Then all three stand at one each, and the skills ranking
picks Advanced Education. Then Service Skills and Personal Development tie at
one, and the ranking takes Service Skills.

`internal/chargen/testdata/drafted-then-commissioned.md` — `term 2`

```markdown
**34. term 2 (pp. 5-7)**

35. survival: rolled 4+3 +2 (Intelligence 7+) = 9 against 5+, made
36. survived term 2 (from step 35)
37. attempt a commission (p. 6): policy chose yes from yes, no
38. commission: rolled 2+5 +1 (Intelligence 6+) = 8 against 4+, made
39. commissioned: 4th Officer (from step 38)
40. attempt promotion (p. 6): policy chose yes from yes, no
41. promotion: rolled 5+6 = 11 against 10+, made
42. promoted: 3rd Officer (from step 41)

**43. skills and training (pp. 6, 11)**

44. acquired skills table to consult (pp. 6, 11): policy chose Personal Development Table from Personal Development Table, Service Skills Table, Advanced Education Table
45. Personal Development Table: rolled 1
46. Strength +1, 7 to 8 (from step 45)
47. acquired skills table to consult (pp. 6, 11): policy chose Advanced Education Table from Personal Development Table, Service Skills Table, Advanced Education Table
48. Advanced Education Table: rolled 2
49. Mechanical-2 (from step 48)
50. acquired skills table to consult (pp. 6, 11): policy chose Service Skills Table from Personal Development Table, Service Skills Table, Advanced Education Table
51. Service Skills Table: rolled 5
52. Electronic-1 (from step 51)
```

The character's Education is 7, so the fourth table (Education 8+) is not
offered. `offeredTables` tests the current value at each designation. Each of
these outcomes once carried `[E002]`, and since #524 none does.

### Reenlistment and leaving

The reenlistment throw is made every term, before the intent is asked,
because a 12 and a failure both settle the question, and a question with one
answer is not asked. A 12 exactly is a `RetainedFact`. A failure departs. A
success asks the intent, where more than one intent is legal.

`internal/chargen/reenlist.go` — `run.reenlist`

```go
	attempt := roll(r.roll, r.service.Reenlist, 0)
	seq := r.record.throw("reenlistment", attempt)

	if attempt.dice[0]+attempt.dice[1] == forcedByTwelve {
...
		r.record.fact(seq, errata, traveller.RetainedFact{Age: r.atAge})

		return traveller.Continue, nil
	}

	if !attempt.succeeded {
		r.depart(seq, term, true)

		return traveller.Discharge, nil
	}

	return r.intent(seq, term)
```

`depart` decides the departure by the term count, as p. 21 does. At or past
the fifth term, leaving is a retirement whatever brought it on. Below the
fifth term, leaving is a denied reenlistment if the throw refused the
character, and a discharge if the character chose to go. Each of the five
reenlistment facts carries the age at which the term left the character, and
that is how the finished character's age is read.

`internal/chargen/reenlist.go` — `run.depart`

```go
	switch {
	case term >= firstRetiringTerm:
		r.departure = traveller.Retired{}
		r.record.fact(seq, nil, traveller.RetiredFact{After: term, Age: r.atAge, Denied: denied})
	case denied:
		r.departure = traveller.DeniedReenlistment{}
		r.record.fact(seq, nil, traveller.DeniedReenlistmentFact{After: term, Age: r.atAge})
	default:
		r.departure = traveller.Discharged{}
		r.record.fact(seq, nil, traveller.DischargedFact{After: term, Age: r.atAge})
	}
```

### Aging

From the fourth term on, the aging round closes the term (E006), and its
saving throws are made in the table's row order (E007). A failed throw records
an `AgedFact` with both ends and the table's reduction. A characteristic can
be reduced to zero only because E010 opens p. 4's floor to 0 under aging, and
one that reaches zero goes straight to a medical crisis. Surviving the crisis
records a `RecoveredFact`, which carries the months of recovery and the age
they leave the character at.

`internal/chargen/aging.go` — `run.age`

```go
	before := r.profile[characteristic]

	r.profile = r.profile.AgeReduce(characteristic, reduction)

	after := r.profile[characteristic]

	errata := r.agingErrata(term)
	if before-reduction < traveller.MinCharacteristic {
		// Below 1 is where the two readings of p. 4's floor diverge: the
		// ordinary floor would have stopped here and no crisis would
		// follow, and E010 is what carries it to 0 instead.
		errata = append(errata, traveller.E010)
	}

	r.record.fact(seq, errata, traveller.AgedFact{
		Change: traveller.Change{Characteristic: characteristic, From: before, To: after, Delta: -reduction},
	})
```

The character's fourth term ends both ways at once: the reenlistment throw
fails, and the character is old enough for a round. The departure line uses
the page's word.

`internal/chargen/testdata/drafted-then-commissioned.md` — `term 4`

```markdown
80. reenlistment: rolled 1+2 = 3 against 4+, missed
81. left the service after term 4: reenlistment denied (from step 80)

**82. aging, end of term 4 (pp. 7-9)**

83. aging, Strength: rolled 5+1 = 6 against 8+, missed
84. Strength -1, 8 to 7 (from step 83)
85. aging, Dexterity: rolled 4+1 = 5 against 7+, missed
86. Dexterity -1, 6 to 5 (from step 85)
87. aging, Endurance: rolled 3+6 = 9 against 8+, made
88. Endurance holds (from step 87)
```

### Mustering out, the pension and the title

`musterOut` reads the number of rolls off the terms and the rank. For each
roll it asks for a table, unless three rolls have already gone on Table 2. In
that case p. 9 leaves only Table 1, and nothing is asked.

`internal/chargen/muster.go` — `run.chooseMusterTable`

```go
func (r *run) chooseMusterTable(onTableTwo int) (traveller.MusterTable, error) {
	if onTableTwo >= maxRollsOnTableTwo {
		return traveller.TableOne, nil
	}
```

`benefit` offers the +1 that a table allows where the character has earned
it, rolls one die, and reads the row. Table 2 is cash, and the `CashFact`
carries the running total. Table 1 is folded through `applyBenefit`, whose
cases record a `PassageFact`, a `TravellersAidFact`, a `WeaponReceivedFact`
or `ExpertiseFact`, a characteristic change sourced `AtMusteringOut`, or a
ship.

`internal/chargen/muster.go` — `run.benefit`

```go
	if table == traveller.TableTwo {
		r.benefits.Cash += cash
		r.record.fact(seq, nil, traveller.CashFact{Cash: cash, Total: r.benefits.Cash})

		return nil
	}

	applied := &applyBenefit{run: r, because: seq}

	err = benefit.Fold(applied)
```

The fact says whether a result is a repeat, not a fold. `receiveShip`
records a `ShipReceivedFact` for the first ship of its kind. A second Free
Trader is a `FreeTraderAgainFact`, which carries the ship as it now stands:
ten years older, and ten years further paid down. A second scout ship is lost
and carries no fact. Since rc.4 a Free Trader can be received past the fifth
time (E016, #464). P. 23 counts five receipts to own the ship free and clear,
and says nothing of a sixth. E016 ages the ship and leaves the payment term
at zero.

`internal/chargen/muster.go` — `run.receiveShipAgain`

```go
	var errata []traveller.Erratum

	if ship.PaymentYears == 0 {
		errata = []traveller.Erratum{traveller.E016}
	}

	ship.Years += freeTraderYearsPerRoll

	ship.PaymentYears = max(0, ship.PaymentYears-freeTraderYearsPerRoll)

	r.record.fact(because, errata, traveller.FreeTraderAgainFact{Ship: *ship})
```

The golden `merchants-sixth-free-trader` reaches that receipt. The last line
below is the one that E016 governs, and like every line it names no erratum.

`internal/chargen/testdata/merchants-sixth-free-trader.md` — `mustering out`

```markdown
189. mustering out table to consult (pp. 7, 9): policy chose Table 1, Material Benefits from Table 1, Material Benefits, Table 2, Cash Allowances
190. add +1 on Table 1 for rank 5 or 6 (pp. 7, 9): policy chose yes from yes, no
191. Table 1, Material Benefits: rolled 6 +1 = 7
192. the Free Trader again: 40 years old, with 0 years of payments left (from step 191)
193. mustering out table to consult (pp. 7, 9): policy chose Table 1, Material Benefits from Table 1, Material Benefits, Table 2, Cash Allowances
194. add +1 on Table 1 for rank 5 or 6 (pp. 7, 9): policy chose yes from yes, no
195. Table 1, Material Benefits: rolled 6 +1 = 7
196. the Free Trader again: 50 years old, with 0 years of payments left (from step 195)
```

Our character's mustering out has five rolls: four for the terms and one for
the rank. The heading says "rolls" by count, and a single roll reads "1 roll"
since #490.

`internal/chargen/testdata/drafted-then-commissioned.md` — `mustering out`

```markdown
**89. mustering out, 5 rolls (pp. 7, 9, 21-23)**

90. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
91. Table 2, Cash Allowances: rolled 4
92. CR 20000, bringing the total to CR 20000 (from step 91)
93. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
94. Table 2, Cash Allowances: rolled 4
95. CR 20000, bringing the total to CR 40000 (from step 94)
96. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
97. Table 2, Cash Allowances: rolled 3
98. CR 10000, bringing the total to CR 50000 (from step 97)
99. Table 1, Material Benefits: rolled 6
100. a Low Passage (from step 99)
101. Table 1, Material Benefits: rolled 6
102. a Low Passage (from step 101)
```

The fourth and fifth rolls ask nothing. The cash strategy put three rolls on
Table 2, so Table 1 is all that is left. The Merchants pay a pension, but the
character left after four terms, so `pension` finds no row in p. 21's table
and records nothing. The final Social Standing is 7, as rolled, so
`assessTitle` neither offers a title nor stamps E011, and the record ends
there. A character that is alive at the end and eligible is asked
`AssumeTitle`, and the answer is a `TitledFact`. A dead character is assessed
and not asked (E011).

## 6. A question, through the gate

Every question reaches the decider through `logging`, which `generate` wraps
around whatever decider it was given. `logging` implements every method of
`Decider`, one per choice point, so a choice point cannot reach the engine
without reaching the record. Each method names the **offer** and refuses a
question with only one legal answer. It hands the decider a copy of what is
offered, which the decider owns, and it records the answer only if that
answer was offered.

`internal/chargen/logging.go` — `logging.Service`

```go
func (l logging) Service(from []traveller.EnlistmentOffer, vary Vary) (traveller.ServiceName, error) {
	offer := names(from, func(o traveller.EnlistmentOffer) string { return o.Service.String() })

	err := asked(traveller.ChoiceService, offer)
	if err != nil {
		return 0, err
	}

	chosen, err := l.to.Service(own(from), vary)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceService, offer, chosen.String())
}
```

`internal/chargen/logging.go` — `logging.record`

```go
func (l logging) record(point traveller.ChoicePoint, from []string, chosen string) error {
	if !slices.Contains(from, chosen) {
		return fmt.Errorf("%w: %v offered %v, and was answered %q",
			errNotOffered, point, from, chosen)
	}

	l.into.choice(point, from, chosen)

	return nil
}
```

On an auto run the decider is a `chargen.Policy`: three strategies, each over
a group of choice points. The policy draws the service (#420). Taking the
likeliest enlistment, as the policy did before #420, had made a referee's
batch nearly all Army. Before #533 the note ended with lines naming the
policy's version and strategies. Those lines are gone, and each choice's line
now says who made it, `policy chose` or `player chose`, as the doc comment
on `DefaultPolicy` says (#537).

`internal/chargen/policy.go` — `Policy.Service`

```go
func (p Policy) Service(from []traveller.EnlistmentOffer, vary Vary) (traveller.ServiceName, error) {
	return from[vary.Among(len(from))].Service, nil
}
```

`Vary` is a `Roller` with everything but `Among` taken off it. It is wrapped
in a value (`varyOnly`), so no type assertion can hand the decider the dice
back. A die thrown by a decider would reach no event, and the record must
carry every die. The other draw is `Weapon`. Every other answer is a ranking
over what is offered, or a fixed yes or no. Our character's `--service navy`
suppressed `Service`, so the one draw was the weapon at event 65.

On a guided run the decider is the command's `player`. It asks only what the
engine asked, with exactly the alternatives offered. It asks again rather
than guess at an answer it cannot read, and since #461 the prompt carries
the range it accepts.

`cmd/ctchargen/interactive.go` — `player.choose`

```go
		typed := strings.TrimSpace(p.in.Text())

		chosen, err := strconv.Atoi(typed)
		if err == nil && chosen >= 1 && chosen <= len(options) {
			p.sayf("\n")

			p.given = append(p.given, chosen)

			return chosen - 1, nil
		}

		if typed == "" {
			p.sayf("  %s\n", nudge)

			continue
		}

		p.sayf("  %q is refused; %s\n", typed, nudge)
```

A question that leads to a throw shows that throw before the answer: the
target, the DM, and what earns it (#487). The `Odds` that the engine computes
are what the decider is handed, so a person and the policy see the same
numbers.

`cmd/ctchargen/interactive.go` — `oddsText`

```go
func oddsText(odds traveller.Odds) string {
	text := fmt.Sprintf("%v, DM %+d", odds.Target, odds.DM)

	if len(odds.From) == 0 {
		return text
	}

	return text + " for " + traveller.Prerequisites(odds.From)
}
```

The following transcript is of a guided session against a build of this
tree, given one answer and then the end of its input. It shows the end of the
characteristics and everything after. The dice are the engine's own, so
another run asks other questions. Everything here goes to standard error.

```text
$ printf '1\n' | ctchargen new -o guided.md
...
14. UPP BA8679 at age 18

15. enlistment (pp. 5, 10)


Which service will you try to enlist in? (p. 10)
  1) Navy (enlistment 8+, DM +0)
  2) Marines (enlistment 9+, DM +2 for Strength 8+)
  3) Army (enlistment 5+, DM +3 for Dexterity 6+, Endurance 5+)
  4) Scouts (enlistment 7+, DM +3 for Intelligence 6+, Strength 8+)
  5) Merchants (enlistment 7+, DM +3 for Strength 7+, Intelligence 6+)
  6) Other (enlistment 3+, DM +0)
answer 1-6>
16. you chose Navy
17. enlistment, Navy: rolled 5+4 = 9 against 8+, made
18. enlisted in the Navy (from step 17)

19. term 1 (pp. 5-7)

20. survival: rolled 4+3 = 7 against 5+, made
21. survived term 1 (from step 20)

Attempt a commission this term? (commission 10+, DM +1 for Social Standing 9+; p. 6 names no cost for a failure)
  1) yes
  2) no
answer 1-2>
Answered 1; the session is lost and nothing was written.
ctchargen: generating: attempting a commission: the input ended while the procedure was still asking
```

In the live view a step is a plain numbered line, where the note marks it
bold. A choice is echoed as `you chose`. When the input ends mid-session, the
player says what was answered and that nothing was written, and the run
fails, since a character finished by an answer nobody gave is not the one asked for
(ADR 0001, ADR 0003). `guided.md` was never made: the note is opened only
after the character is generated.

## 7. Reading the character back

`Character` keeps five fields, and none of them is a fact about the
character.

`internal/chargen/character.go` — `Character`

```go
type Character struct {
	Inputs Inputs

	// Policy is what answered an auto run. It is nil on a guided run, where a
	// person answered and no policy decided anything.
	Policy *Policy

	Events []traveller.Event
	Errata []traveller.Erratum
...
	Build string
}
```

Everything else is a method that folds the record. `fold` walks the outcomes
and hands each fact to the cases it is given. `sheetFold` is the set of cases
that reads the sheet.

`internal/chargen/character.go` — `Character.fold`

```go
func (c *Character) fold(cases traveller.FactCases) {
	for _, event := range c.Events {
		outcome, isOutcome := event.(traveller.OutcomeEvent)
		if !isOutcome || outcome.Fact == nil {
			continue
		}

		// An error from cases is dropped, so every FactCases handed here must
		// be one whose cases only assign and cannot fail.
		_ = outcome.Fact.Fold(cases)
	}
}
```

`internal/chargen/character.go` — `Character.Profile`

```go
func (c *Character) Profile() traveller.Profile {
	var fold sheetFold

	c.fold(&fold)

	return fold.profile
}
```

Each case only assigns, because each fact already carries the value the
engine reached. A characteristic starts at its `RolledFact` and takes the
`To` of every change after it. A skill takes the level its last fact reached.
The age and the departure take the last fact that moves them.

`internal/chargen/character.go` — `sheetFold.Rolled`

```go
func (s *sheetFold) Rolled(f traveller.RolledFact) error {
	s.profile[f.Characteristic] = f.Value

	return nil
}
```

`internal/chargen/character.go` — `sheetFold.Aged`

```go
func (s *sheetFold) Aged(f traveller.AgedFact) error {
	s.profile[f.Change.Characteristic] = f.Change.To

	return nil
}
```

`internal/chargen/character.go` — `sheetFold.DeniedReenlistment`

```go
func (s *sheetFold) DeniedReenlistment(f traveller.DeniedReenlistmentFact) error {
	s.age, s.departure = f.Age, traveller.DeniedReenlistment{}

	return nil
}
```

For our character, this works out as follows. Strength is rolled 7, raised to
8 at event 46, and aged to 7 at event 84, so the profile reads 7. Dexterity is
rolled 6 and aged to 5. The age is the `DeniedReenlistmentFact`'s 34. The
sheet's UPP, 754777, is not the 764777 that event 14 wrote. Event 14 is the
profile as rolled, and the sheet's UPP is folded to the end.

`Terms` is the one fact that is not read from an outcome. It is the number
that the last term-opening step carries, which counts the fatal term too
(E004).

`internal/chargen/character.go` — `Character.Terms`

```go
func (c *Character) Terms() int {
	terms := 0

	for _, event := range c.Events {
		if step, isStep := event.(traveller.StepEvent); isStep && step.Opens > 0 {
			terms = int(step.Opens)
		}
	}

	return terms
}
```

`History` folds the whole event list, not only the facts, through
`historyFold`. A step that opens a term starts a `TermServed`, and each fact
is filed by its source: on entering, at a rank (after the raise that
conferred it), in the term, or at mustering out. Mustering out's gains are
new in the history since #490. A characteristic that a Table 1 row raised was
in the record and the UPP, but on no line that said where it came from. The
benefits themselves, the cash and the possessions, are the sheet's, and the
history leaves them out.

`internal/chargen/history.go` — `historyFold.gained`

```go
func (h *historyFold) gained(gain traveller.Gain, source traveller.Source) {
	switch source {
	case traveller.OnEntering:
		h.out.OnEntering = append(h.out.OnEntering, gain)
	case traveller.AtRank:
		// A grant at a rank follows the raise that conferred it, in the same
		// term, so the term has one.
		raised := h.serving().Raised

		latest := &raised[len(raised)-1]

		latest.Grants = append(latest.Grants, gain)
	case traveller.InTerm:
		h.serving().Gained = append(h.serving().Gained, gain)
	case traveller.AtMusteringOut:
		h.out.MusteredOut = append(h.out.MusteredOut, gain)
	}
}
```

`Attempted` reads the service that the enlistment throw was made against from
its `EnlistedFact`. Whether the policy named that service is read from the
run's inputs: an unforced auto run is the policy's choice, and a `--service`
or a guided answer is not (#397).

`internal/chargen/attempt.go` — `Character.Attempted`

```go
		if enlisted, ok := outcome.Fact.(traveller.EnlistedFact); ok {
			_, forced := c.Inputs.Forced.Service()

			return Attempt{Service: enlisted.Service, ByPolicy: !forced && c.Policy != nil}, true
		}
```

The engine itself never calls these methods. Mid-run it reads its working
state (`r.profile`, `r.rank`, `r.terms`), and the goldens hold the working
state and the facts to each other. A fact that carried a value the working
state did not reach would change a note.

## 8. The rules the engine reads

Every number above came from `rules.Load`. It lifts every table once per
process and hands each caller the same `*Rules`. A table that will not lift
fails at first use, and not only on some path that happens to reach it.

`internal/rules/lift.go` — `load`

```go
	for _, lift := range []func() error{
		func() error { return rules.liftPriorService(transcribedServices) },
		func() error { return rules.liftGrants(transcribedServices) },
		func() error { return rules.liftRetirement(transcribedServices) },
		func() error { return rules.liftSkills(transcribedSkills) },
		func() error { return rules.liftMustering(transcribedMustering) },
		func() error { return rules.liftAging(transcribedAging) },
		func() error { return rules.liftWeapons(transcribedWeapons) },
		func() error { return rules.liftNobility(transcribedNobility) },
		func() error { return rules.liftShips(transcribedShips) },
	} {
```

The **transcription** is the page's text as a Go literal: its glyphs,
abbreviations and dashes, with the services across a row in
`traveller.ServiceNames` order. The Navy's 8+ that our character missed is the
first cell here.

`internal/rules/transcription.go` — `transcribedServices`

```go
	enlistment: printedThrows{
		targets: perService{"8+", "9+", "5+", "7+", "7+", "3+"},
		dms: []printedDMRow{
			{
				heading:    "DM of +1 if",
				conditions: perService{"Intel 8+", "Intel 8+", "Dext 6+", "Intel 6+", "Stren 7+", "—"},
			},
			{
				heading:    "DM of +2 if",
				conditions: perService{"Educ 9+", "Stren 8+", "Endur 5+", "Stren 8+", "Intel 6+", "—"},
			},
		},
	},
```

Even the reprint's broken glyphs stay as printed. P. 23's box grants the Army
`"Rifl3-1"`, and E012's normalization map turns `Rifl3` into `Rifle` in the
lift, where the judgement is named. The second transcription, in the `rules`
tests, is typed separately and in the domain's words, so a misread cell
cannot pass from one copy to the other.

Lifted, a throw is a `rules.Throw`: a target and its DMs. One test decides
which DMs a profile meets, and it serves both methods: `Modifier` sums those
DMs, and `Met` names them as the prerequisites the page prints. That is where
the record's `(Intelligence 7+)` comes from.

`internal/rules/rules.go` — `Throw.Met`

```go
func (t Throw) Met(p traveller.Profile) []traveller.Prerequisite {
	met := t.met(p)
	prerequisites := make([]traveller.Prerequisite, 0, len(met))

	for _, dm := range met {
		prerequisites = append(prerequisites, traveller.Prerequisite{
			Characteristic: dm.Characteristic, Threshold: dm.Threshold,
		})
	}

	return prerequisites
}
```

`internal/rules/rules.go` — `Throw.met`

```go
func (t Throw) met(p traveller.Profile) []DM {
	var met []DM

	for _, dm := range t.DMs {
		if dm.Threshold.Satisfied(p[dm.Characteristic]) {
			met = append(met, dm)
		}
	}

	return met
}
```

A die's six faces size the skills tables here as `rules.Faces`, and the
`dice` package rolls by its own `dice.Faces`. There are two constants because
`dice` depends on nothing and `rules` may not import it. Each constant names
the other, and a test in `chargen`, which imports both, holds them equal
(#514).

`internal/dice/dice.go` — `Faces`

```go
const Faces = 6
```

The readings live beside the tables as `rules.Reading` values, one per
erratum. `Readings` returns an array whose length is `len(traveller.Errata)`,
so a reading missing or one too many does not compile. There are sixteen
since E016. Each reading carries its pages, what the page says, the gap, the
reading, why, and the condition it is stamped on, all as prose that marks the
page's emphasis in Markdown. The tree is the only place a reading is written.
The withdrawal machinery, the `readings` subcommand and `--readings` all went
with #524.

`internal/rules/readings.go` — `Readings`

```go
func Readings() [len(traveller.Errata)]Reading { return readings }
```

## 9. Writing the note

`render.Note` writes the character in one pass from one `view`, and ends with
the generation record (#533). The closing lines that once named the ruleset,
the build and who answered are gone. The build is a property, who made each
choice is on that choice's line, and the ruleset is the same on every note.

`internal/render/note.go` — `Note`

```go
func Note(character *chargen.Character) string {
	r := viewOf(character)

	var out strings.Builder

	writeProperties(&out, r)
	writeSheet(&out, r)
	writeGenerationRecord(&out, r)

	// The record's last line ends the note, with no blank line after it.
	return strings.TrimRight(out.String(), "\n") + "\n"
}
```

Within `render`, `viewOf` is the only place where the character's methods are
called, so the sheet and the generation record cannot read the character two
ways. It turns domain values into the words the note prints. The view keeps
no errata: `render` does not import `rules`, and the package doc carries the
promise that a note names no erratum.

`internal/render/view.go` — `viewOf`

```go
	out := view{
		Name:       character.Inputs.Name,
		UPP:        profile.UPP(),
		Profile:    profile,
		Age:        ageView{Years: age.Years(), Months: age.Months()},
		Terms:      character.Terms(),
		Enlistment: foldEnlistment(character.Enlistment()),
		RankName:   character.RankName(),
		Skills:     make([]skillView, 0, len(skills)),
		Benefits:   projectBenefits(character.Benefits()),
		Pension:    int64(character.Pension()),
		ByPolicy:   character.Policy != nil,
		Events:     entries(character.Events),
		Build:      character.Build,
	}
```

### Properties

The note opens with YAML properties (#399). Their names and meanings freeze
at `v1.0.0`, and `writeProperties` states each meaning in its doc. Since #533
every note carries every key. A text property with no value is an empty key,
and a list with no items is `[]`. `skills` is the names alone, so that "which
crew have Pilot?" is an exact match (#474). `possessions` lists what
mustering out gave besides cash, one item each (#475). `ctchargen` names the
build (#476). Every text value is quoted, so nothing a later build writes can
change a property's type.

`internal/render/properties.go` — `writeProperties`

```go
	fmt.Fprintf(out, "age: %d\n", r.Age.Years)
	property(out, "service", r.Service)
	property(out, "rank", r.RankName)
	fmt.Fprintf(out, "terms: %d\n", r.Terms)
	property(out, "status", status(r))
	property(out, "departure", r.Departure.How)

	names := make([]string, 0, len(r.Skills))
	for _, skill := range r.Skills {
		names = append(names, skill.Name)
	}

	list(out, "skills", names)
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `properties`

```markdown
---
name:
aliases: []
upp: "754777"
strength: 7
dexterity: 5
endurance: 4
intelligence: 7
education: 7
social: 7
age: 34
service: "Merchants"
rank: "3rd Officer"
terms: 4
status: "alive"
departure: "denied reenlistment"
skills:
  - "Cudgel"
  - "Electronic"
  - "Jack of all Trades"
  - "Mechanical"
  - "Medical"
credits: 50000
pension: 0
nobility:
possessions:
  - "Low Passage"
  - "Low Passage"
ctchargen:
---
```

`name` is empty because `new` was given no `--name`, and a batch names nobody
(#406). A referee names a character in the vault by setting `name` and
`aliases`. `ctchargen` is empty here because a golden is generated in-process
by a test, which passes no build. A note the command writes carries the
version, such as `"v1.0.0-rc.5"`.

### Heading, summary and headline

An unnamed note is headed by its title: the service, the rank and the UPP.
A batch names the file by the same title (`render.Title`). Under the heading,
`summary` writes a paragraph a referee can read aloud (#425). It says what
the character is, for how long, how the service ended, the skills at level 2
or above, and the possessions that matter.

`internal/render/summary.go` — `summary`

```go
	switch {
	case r.Departure.Fatal:
		sentences = append(sentences, fmt.Sprintf("%s, %s in term %d, aged %d.",
			serving(r), r.Departure.How, r.Terms, r.Age.Years))
	default:
		sentences = append(sentences, fmt.Sprintf("%s, %s, %s at %d.",
			serving(r), terms(r.Terms), r.Departure.How, r.Age.Years))
	}
```

The headline that follows is the line p. 25 leads Jamison with. Where the
service on it is not the one attempted, `whoseService` says so, as it does
for the draftee here.

`internal/render/sheet.go` — `whoseService`

```go
	switch {
	case attempted != r.Service:
		return fmt.Sprintf(" (%s after the %s refused enlistment)", r.Enlistment.How, attempted)
	case byPolicy:
		return " (service chosen by the policy)"
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `heading`

```markdown
# Merchants 3rd Officer 754777

Merchants 3rd Officer, 4 terms, denied reenlistment at 34. Known for Mechanical-2. Musters out with CR 50000.

UPP 754777, age 34, Merchants 3rd Officer (drafted after the Navy refused enlistment), 4 terms, denied reenlistment
```

### The service record

`serviceLines` tells the **service history** a term at a time: how the
character entered and what entering granted, each term with something to
tell, how the service ended, and what mustering out changed.

`internal/render/sheet.go` — `serviceLines`

```go
	switch {
	case r.Departure.Characteristic != "":
		lines = append(lines, fmt.Sprintf("%s in term %d (%s reached zero)",
			r.Departure.How, r.Terms, r.Departure.Characteristic))
	case r.Departure.Fatal:
		lines = append(lines, fmt.Sprintf("%s in term %d", r.Departure.How, r.Terms))
	default:
		lines = append(lines, fmt.Sprintf("%s after term %d", r.Departure.How, r.Terms))
	}

	if len(r.History.MusteredOut) > 0 {
		lines = append(lines, "mustered out with "+strings.Join(r.History.MusteredOut, ", "))
	}
```

A change is written as the table gave it. Where a floor or ceiling held the
change short, the line says where it was held (#409): `Strength +1 (held at
15)`.

`internal/render/history.go` — `changeWords`

```go
func changeWords(change traveller.Change) string {
	words := fmt.Sprintf("%v %+d", change.Characteristic, change.Delta)
	if change.To-change.From != change.Delta {
		words += fmt.Sprintf(" (held at %d)", change.To)
	}

	return words
}
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `Service record`

```markdown
## Service record

- Merchants, drafted
- Term 1: Mechanical-1, Jack of all Trades-1
- Term 2: commissioned 4th Officer and promoted 3rd Officer; Strength +1, Mechanical-2, Electronic-1
- Term 3: Cudgel-1
- Term 4: Medical-1; aging Strength -1, Dexterity -1
- denied reenlistment after term 4
```

Our character mustered out with nothing but cash and passages, so there is no
last line. The sixth Free Trader's character took two characteristic rows off
Table 1, and its record shows both. The `(Pilot-1)` there is a grant that came
at the rank, in parentheses after the raise.

`internal/chargen/testdata/merchants-sixth-free-trader.md` — `Service record`

```markdown
- Term 6: promoted 1st Officer (Pilot-1); Mechanical-2, Sword-1; aging Dexterity -1
- Term 7: promoted Captain; Strength +1, Navigation-4; aging Endurance -1
- retired after term 7
- mustered out with Intelligence +1, Education +1
```

### The generation record

`writeGenerationRecord` writes every event under one `## Generation record`
heading (#422). Each step is a bold line, and every other event is a numbered
line from the same `entry` that the live view prints for a guided run.

`internal/render/record.go` — `writeEvent`

```go
	switch event.Kind {
	case kindStep:
		fmt.Fprintf(out, "\n**%s**\n\n", stepLine(event))
	case kindThrow, kindRoll:
		// One line for both: a roll is a throw whose target is empty, which
		// is exactly what throwLine already prints.
		fmt.Fprintln(out, throwLine(event))
	case kindChoice:
		fmt.Fprintln(out, choiceLine(event, answerer))
	case kindOutcome:
		fmt.Fprintln(out, outcomeLine(event))
	}
```

A throw's line puts the DM's prerequisites after the DM, then the total, the
target and whether it was met.

`internal/render/record.go` — `throwLine`

```go
	if event.DM != 0 {
		line += fmt.Sprintf(" %+d", event.DM)
	}

	// The characteristics the DM is earned by, with the thresholds p. 10
	// prints for them, so it can be checked against the page (#487).
	if event.DMFrom != "" {
		line += " (" + event.DMFrom + ")"
	}
```

A choice's line names who answered. That is the run's to say and not the
event's, so `writeGenerationRecord` passes the view's answerer to every
choice line.

`internal/render/record.go` — `choiceLine`

```go
func choiceLine(event entry, answerer string) string {
	return fmt.Sprintf("%d. %s: %s chose %s from %s",
		event.Seq, choiceWords(event.Point), answerer, event.Chosen,
		strings.Join(event.Alternatives, ", "))
}
```

An outcome's line is its prose (the fact's `String`, where the outcome
carries a fact), followed by the event that caused it. Nothing follows that:
the errata that the line once ended with are gone since #524.

`internal/render/record.go` — `outcomeLine`

```go
func outcomeLine(event entry) string {
	line := fmt.Sprintf("%d. %s", event.Seq, event.Description)

	if event.Because != 0 {
		line += fmt.Sprintf(" (from step %d)", event.Because)
	}

	return line
}
```

The note's last line is the record's last event: event 102 for our character,
`a Low Passage (from step 101)`.

## 10. `batch`

A batch is always an auto run. It writes only the living (#421), and only into
a folder of its own (#426). `--count` is decimal digits and nothing else
(#466). The flag package's own int reads base 0, so `010` was eight and
`0x10` was sixteen. `decimalCount` still answers `Get` with an int, so the CLI
baseline's `count int 0` holds.

`cmd/ctchargen/usage.go` — `decimalCount.Set`

```go
func (c *decimalCount) Set(value string) error {
	if strings.TrimLeft(value, "0123456789") != "" {
		return errNotAWholeNumber
	}
```

`-o` names a folder. An `-o` ending in `.md` almost certainly meant a file, so
it is refused outright, and `-` is standard output, as for `new`. Since #470 a
batch makes its own folder and nothing above it. The folder's parent must
already exist, and `os.Mkdir` replaced `MkdirAll`.

`cmd/ctchargen/batch.go` — `batchRun.intoDirectory`

```go
	folder := filepath.Dir(filepath.Clean(dir))
	if !isDirectory(folder) {
		return tally{}, fmt.Errorf("%s %w, so -o %s cannot be made", folder, errNoFolder, dir)
	}

	err := os.Mkdir(dir, noteDirMode)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return tally{}, fmt.Errorf("creating %s: %w", dir, withoutPath(dir, err))
	}

	err = ownFolder(dir)
	if err != nil {
		return tally{}, err
	}
```

`ownFolder` then refuses a folder that holds anything a person would see in a
listing. Hidden entries, such as the `.DS_Store` that Finder leaves, do not
count. The check reads the folder once, before anyone is generated.

`cmd/ctchargen/batch.go` — `ownFolder`

```go
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") {
			return fmt.Errorf("%s %w; a batch writes into a folder of its own, a new or empty one",
				dir, errFolderInUse)
		}
	}
```

`each` keeps generating until it has the number asked for, passing over the
dead. It gives up with an error after a hundred characters per member asked
for. The bound is checked by division, so a count near the top of an int
cannot overflow it (#423).

`cmd/ctchargen/batch.go` — `batchRun.each`

```go
	for i := 0; done.written < b.count; i++ {
		// Divided, not multiplied: count*attempts overflows an int for a
		// count near its top, and the batch gave up having drawn nobody (#423).
		if i/b.attempts >= b.count {
...
		}

		character, err := b.member(i)
		if err != nil {
			return done, err
		}

		if traveller.Fatal(character.Departure()) {
			done.passed++

			continue
		}

		err = take(done.written+1, character)
		if err != nil {
			return done, err
		}

		done.written++
	}
```

Into a folder, each **member** is written as `N Title.md`. N is the member's
**position**, zero-padded to the width of the count, and the title is the
same one an unnamed note is headed by (#400). Each note is written through
the same exclusive create that `new` uses without `--force`.

`cmd/ctchargen/batch.go` — `batchRun.intoDirectory`

```go
	return b.each(
		func(index int, character *chargen.Character) error {
			where, err := openDestination(nil,
				memberPath(dir, position(index, b.count), render.Title(character)), false)
			if err != nil {
				return err
			}

			return where.write(render.Note(character))
		},
	)
```

The following is a transcript of batch runs against a build of this tree.
The dice are the engine's own, so another run makes another crew. The line
that reports the count goes to standard error.

```text
$ ctchargen batch --count 4 -o crew
4 written, 4 passed over for dying
$ ls crew
1 Marines C78565.md
2 Army Major 98968C.md
3 Navy 543A78.md
4 Navy 549837.md
$ ctchargen batch --count 1 -o crew
ctchargen: crew is not empty; a batch writes into a folder of its own, a new or empty one
$ ctchargen batch --count 1 -o nowhere/crew
ctchargen: nowhere is not a folder that exists, so -o nowhere/crew cannot be made
$ ctchargen batch --count 1 -o crew.md
ctchargen: usage: -o crew.md names a file; a batch writes its notes into a folder
$ ctchargen batch --count 0x10
ctchargen: usage: invalid value "0x10" for flag --count: not a whole number; run `ctchargen batch --help`
```

Without `-o`, the notes go to standard output one after another. Each note is
written as it is generated, so a large batch holds one character at a time.

## 11. How the tests drive it

### Fixtures and goldens

A **fixture** names a path and states its **script**: every die and every
draw, in the order the procedure asks for them. Our character is this entry
of the roster.

`internal/chargen/fixtures_test.go` — `fixtures`

```go
	{
		name: "drafted-then-commissioned", forced: chargen.ForceService(traveller.Navy), policy: chargen.DefaultPolicy(),
		dice: []int{
			6, 1, 5, 1, 2, 2, 5, 2, 3, 4, 4, 3, 2, 3, 5, 3, 4, 2, 6, 6,
			1, 4, 3, 2, 5, 5, 6, 1, 2, 5, 4, 6, 5, 1, 3, 4, 5, 4, 2, 5,
			2, 4, 1, 6, 1, 2, 5, 1, 4, 1, 3, 6, 4, 4, 3, 6, 6,
		},
		among: []int{9},
	},
```

The one draw, index 9, is Cudgel from the blades list at event 65. The
scripted roller is strict both ways. A die asked for past the end fails the
test, and `Finished` reports a die left unread, so a change to the order in
which the procedure consumes dice fails the fixture that felt it.

`internal/scripted/scripted.go` — `Roller.Die`

```go
func (r *Roller) Die() int {
	r.t.Helper()

	if r.die == len(r.dice) {
		r.t.Fatalf("%s: the procedure asked for die %d, and the script holds %d",
			r.name, r.die+1, len(r.dice))
	}

	face := r.dice[r.die]
	r.die++

	return face
}
```

`TestGoldens` renders each fixture's note and compares it to its **golden**
under `internal/chargen/testdata/`. `-update` rewrites the goldens, and a
golden diff is read before it is kept. Every fact the note reads is folded
from the record, so the goldens also hold the folds to the engine.

`internal/chargen/golden_test.go` — `TestGoldens`

```go
	for _, fixture := range fixtures {
		golden(t, fixture.name+".md", []byte(render.Note(fixture.generate(t))))
	}
```

The command's own tests drive `run` with `withDice` and a scripted source.
These are **walks**, extracted from a record the tool wrote, and they let a
guided session be tested through the command, standard error questions and
all (ADR 0003). A few of the command's tests run the test binary as the
command, in a process of its own (`TestMain`), so that a file size limit can
fail a write partway through a note.

### The command line's baseline

`v1.0.0` freezes the command line's flags, types and defaults (#126).
`cli_test.go` types them out and holds them one way: a flag removed or
changed fails, and a flag added does not. Its comments record what went on
purpose before the freeze, `new`'s `readings bool false` among them, and
`TestTheReadingsAreNotOnTheCommandLine` holds that both the subcommand and
the flag are refused with exit 1.

`cmd/ctchargen/cli_test.go` — `cliBaseline`

```go
	cmdNew: {
		"auto bool false", "career string serve", baseForce,
		"muster string cash", "name string ", baseOutput,
		"service string ", "skills string advanced",
	},
```

### The stamping harness

`internal/chargen/stamping_test.go` holds every reading to its stamping
condition. It checks fixture by fixture, in both directions, because an
erratum stamped where its condition does not hold is as much a defect as one
missing. It reads each fixture's `Character`, never a note, because the note
carries no stamp. Each condition is a predicate written by hand from the
reading's stamped-on prose, never from the stamping code, and a failure quotes
the prose from `rules.Readings`.

### The markers

Every rule an implementation carries is marked beside it with its cite, and
every test that holds one says so. `internal/coverage` joins the two by the
rule's name and fails when either half is missing. The map is printed on
demand and never committed.

`internal/coverage/coverage_test.go` — `coverage_test`

```go
//	// Implements: Six 2D rolls, in order [1:4].
//
// A test that holds a rule carries one line naming it:
//
//	// Holds: Six 2D rolls, in order.
```

The fixture above carries its own marker,
`// Holds: Draftees are not eligible for commission in the first term only.`,
and the `Implements:` marker on `run.commission` names the same rule. A new
invariant is not done until `go run ./internal/mutate` has shown that a
deliberate edit to it is killed, in a scratch copy of the tree.

### The gate

`task` is the gate, and only a local run holds it, since there is no CI. It
runs `go mod tidy -diff`, golangci-lint (which also checks formatting), NilAway,
govulncheck, `go test -race`, `bun fmt --check`, and an offline link check.
Since #492, `task setup` installs every tool the gate runs from the
repository alone: the Brewfile's tools through `brew bundle`, and NilAway,
which has no formula, through `go install`.
