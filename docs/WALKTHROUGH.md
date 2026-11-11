# Walkthrough

> Release snapshot for `v1.0.0-rc.6`. It describes that tag, not `main`. Between
> releases nothing reads, sweeps or updates it (`docs/CODING_STANDARDS.md`, "A
> release snapshot is left alone between releases").

## Overview

`ctchargen` is a Go command that generates Classic Traveller characters by
walking Book 1's procedure (pp. 4–25 of the 1977 text). It writes each one as
a markdown **note** (ADR 0006): properties a vault can query, a heading, the
sheet, which opens with the **profile line** and a summary a referee can read
aloud, and the **generation record** of every die thrown and every choice
made. The note ends with the record. It is the only way back to a character,
and nothing reads one back. There are three subcommands:

- `new`: one character, answered by the policy (an **auto run**, `--auto`) or
  by a person at the terminal (a **guided run**);
- `batch`: many auto runs, each written as its own note into a folder of the
  batch's own, or one after another to standard output;
- `version`: the build.

The module needs nothing beyond the standard library. The rules tables are Go
literals typed as the page prints them, and they are **lifted** into the
domain's values once per process (ADR 0005). The **readings**, the decisions
taken where the page is silent, are Go values compiled in beside them. They
are the authority's record, not the referee's (ADR 0007): the engine stamps
each erratum on the character it governed, the tests hold every stamp to its
reading's condition, and no note names one.

`THEORY.md` says why the system is shaped this way. This document follows one
character through the code. Hold one idea through all of it: the generation
record is the character. Every outcome a sheet reads carries a typed
`traveller.Fact`, and everything the note says about the character is folded
out of those facts: the UPP, the skills, the benefits, the service record,
the age, and how the service ended. While the engine runs it keeps a working
copy of the character, because every DM reads one. The finished `Character`
holds only its record and what it was given.

## Architecture

```text
cmd/ctchargen/        the subcommands; stdout carries the note, stderr the conversation
internal/traveller/   the domain: alphabets, values, the sums, events and facts (imports nothing here)
internal/rules/       every table of pp. 4–25, transcribed in Go and lifted once, and the readings as values
internal/dice/        an unseeded stream of dice
internal/chargen/     the procedure, the Decider it asks, the auto run's Policy, and the folds that read the record
internal/render/      the note: properties, sheet, generation record, and the live view's lines
internal/scripted/    test support: the strict roller that fixtures and walks play
internal/coverage/    test code only: holds the Implements: and Holds: markers to each other
internal/mutate/      a tool: runs a command against a scratch copy with one edit made
```

The imports point one way. `traveller` and `dice` import none of the others.
`rules` imports `traveller`, and `chargen` imports `traveller`, `rules` and
`dice`. `render` imports `chargen` for the character and `traveller` for its
values, and not `rules`, since a note names no reading. The command imports
`chargen`, `render` and `traveller`. Each edge ruled out that would compile is
a depguard rule.

Data flows in a line. The flags become `chargen.Inputs` and a `Policy`, or a
person answers instead. `chargen` walks the procedure against `rules` and a
roller, and hands back a `*chargen.Character` whose record is the whole of
it. `render` folds that record into the note.

This document follows the character in the golden
`internal/chargen/testdata/drafted-then-commissioned.md`: the run that
`ctchargen new --auto --service navy` makes under the default policy, played
from a scripted roller. The Navy rejects the enlistment, and the draft puts
the character in the Merchants. The character is commissioned and promoted in
the second term and draws a Cudgel in the third. After the fourth term the
reenlistment throw fails: the character is denied reenlistment, ages, and
musters out with cash and two Low Passages.

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
saves a note and not a conversation. After answering help, `run` dispatches
on the subcommand.

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

`chosen.dice` is a `source`. In a real run it is nil, and the engine rolls
its own unseeded stream. It is a scripted roller only when a test passes
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

Errors are of two kinds, and only the message tells them apart, since both
exit 1. `errUsage` is a mistake in the command line: an unknown flag, a value
the flag cannot take, a word where none belongs. Anything found at run time
is not one: a file in the way, a folder that is missing or in use, a failed
write, a build with nothing embedded (#472, #508). A batch's search that
gives up is `errSearch`, for the same reason.

`--help` on `new` or `batch` is `writeHelp`. It prints the usage line and the
flags, then three things in the command's own words around one shared block:
what `--service` comes to, what each strategy does, and what a death comes to
(#563).

`cmd/ctchargen/usage.go` — `writeHelp`

```go
	return writeUsageText(out, "\n"+about.service+"\n"+strategyHelp()+"\n"+about.death+"\n"+flagGrammar)
```

`strategyHelp` types no strategy's name. It reads each flag's values and
their lines from `chargen`'s alphabets, so a strategy the help cannot
describe is one whose `does` has no case.

`cmd/ctchargen/usage.go` — `strategyHelp`

```go
	flags := []struct {
		heading string
		values  []chargen.Described
	}{
		{"--" + careerFlag + ", when to leave the service:\n", chargen.CareerHelp()},
		{"--" + skillsFlag + ", which table each skill roll is on, of those trained on least:\n", chargen.SkillsHelp()},
		{"--" + musterFlag + ", the benefit table for each roll:\n", chargen.MusterHelp()},
	}
```

`internal/chargen/policy.go` — `Muster.does`

```go
func (m Muster) does() string {
	switch m {
	case MusterCash:
		return "Table 2 for up to three rolls, then Table 1; a repeat\n" +
			"blade or gun benefit is expertise in the first"
	case MusterGoods:
		return "Table 1; a repeat blade or gun benefit is expertise in the first"
	case MusterArsenal:
		return "Table 1; a different weapon each time"
	}

	return ""
}
```

The following is a transcript of `new --help` against a build of this tree,
from the line after the flags (nothing re-runs it). The third mustering-out
strategy is `arsenal`; it was `spartan` until #570.

```text
--service is an enlistment attempt, not a guarantee: a failed throw is the
draft. Without it the player picks the service, or --auto draws it.

The strategies are intent, and the throws decide. Every strategy submits to
the draft, attempts every commission and promotion, takes both mustering-out
DMs, assumes any title, and draws each weapon it takes. A 12 on reenlistment
forces another term whatever the strategy.

--career, when to leave the service:
  serve     reenlist whenever the throw allows, up to seven terms
  retire    reenlist to the end of term 5, then retire
  oneterm   leave at the first chance
--skills, which table each skill roll is on, of those trained on least:
  advanced  Advanced Education first, its 8+ table from Education 8
  service   Service Skills first
  personal  Personal Development first
--muster, the benefit table for each roll:
  cash      Table 2 for up to three rolls, then Table 1; a repeat
            blade or gun benefit is expertise in the first
  goods     Table 1; a repeat blade or gun benefit is expertise in the first
  arsenal   Table 1; a different weapon each time

A character can die in service; new writes the note all the same, and exits 0.
terms counts every term begun, including the term in which a character dies.
```

## 2. `new`: from flags to a generation

`new` and `batch` share the flags a character is generated from: `--service`
and the three strategies. These are declared once, in
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

A wrapped flag can be set once, and only to something (#466, #473). The flag
package alone would take the last of two values, and this command used to
read an empty `--service` or `-o` as the flag not given. `strictBool` keeps a
switch such as `--auto` a switch, which needs no value after it.

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

With every flag wrapped, the flag package would call every kind "value",
because it names a kind from the value's concrete type. `kindOf` reads the
kind from what the wrapped value holds instead. A word that a description
puts in backquotes still names the kind, which is how the help shows
`-o file` for `new`, `-o folder` for `batch` and `--career strategy` for both.

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
UTF-8, since the note writes the name as a heading and as a YAML string. It
refuses a strategy flag without `--auto`, which would decide nothing on a
guided run. Then it reads `-o`, where `-` names standard output (#467), and
refuses `--force` with no file to overwrite (#473).

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
$ ctchargen new --auto --muster spartan
ctchargen: usage: no such strategy: --muster "spartan"; want cash, goods, arsenal
```

The second command is a guided run, refused before its first question.

`generate` is where the two ways in part. An auto run hands the policy to the
engine. A guided run builds a `player`, which also watches the record as it
is written, so the throws reach the person between questions.

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

`generate` loads the rules, builds a `run` and applies the options. It wraps
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
working state, and their comments say so: each change to them is recorded as
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

Two more fields are decision aids and never reach the character. `trained`
counts the results taken off each skills table, which the decider is shown.
`fourthTableOpenedBy` holds the event that took Education to the fourth
skills table's threshold, until the next designation says so (the Skills
section below).

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

A step comes in three shapes. `step` enters a step outside every term.
`opens` enters the step that opens a term and carries the term's number in
`Opens`, which is how the finished character counts its terms, the fatal one
included. `phase` enters a step inside a term, the term's skills and training
or its aging round, and carries that term in `Within`, which is how the note
sets those steps under their term (#577).

`internal/traveller/event.go` — `StepEvent`

```go
type StepEvent struct {
	Seq   int
	Step  string
	Pages string

	// Opens is the term this step opens, and zero for every step that opens
	// none, so a term a death ended is still read as a term served (#412).
	Opens Term

	// Within is the term a step is part of without opening it - the term's
	// skills and training, and its aging - and zero for every other step, so
	// the record can set a term's phases under the term (#577).
	Within Term
}
```

An outcome is where a fact rides. `record.fact` writes an `OutcomeEvent`
whose prose is the fact's own `String`, so the line a referee reads and the
value a fold reads cannot say different things (#412). The errata it is
handed go into the record's set and nowhere else.

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
scout ship lost, a dash row, the line that says the Table 2 cap sends the
remaining rolls to Table 1. Both stamp the errata they are handed. `stamped`
returns every erratum that any outcome stamped, in id order, and that list
becomes `Character.Errata`.

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

There are twenty-six. Each carries the value the engine reached, not the
change that reached it, so a fold only assigns. A skill fact carries the
level the skill reached, a cash fact the running total, and a reenlistment
fact the age at which the term left the character.

A choice is a `ChoiceEvent`. Besides the point, the alternatives and the
answer, it carries what the answer was decided on, as data for the note to
word (#564): whether the decider drew it, and for a skills table how many
results each table offered had already given. Who answered is not on the
event; a run has one answerer, and the note says it from the run.

`internal/traveller/event.go` — `ChoiceEvent`

```go
type ChoiceEvent struct {
	Seq          int
	Point        ChoicePoint
	Alternatives []string
	Chosen       string

	// Drawn is whether the decider drew the answer among what it was offered,
	// through the Vary it was handed, rather than choosing it.
	Drawn bool

	// Taken is, for a skills table choice, how many results the character
	// had already had off each table offered, in the order of Alternatives:
	// each SkillTableOffer's Taken, the count a decider weighs. It is nil for
	// every other choice point.
	Taken []int
}
```

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

In the note each step is a heading carrying its own number, and each outcome
names the event it came from as `(from step N)`.

`internal/chargen/testdata/drafted-then-commissioned.md` — `characteristics`

```markdown
### 1. characteristics (p. 4)

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
without asking. Unforced, it builds one `EnlistmentOffer` per service, each
carrying its odds, and asks the decider, handing it the run's dice as a
`Vary`; on an auto run the decider draws the service (section 6). The throw
is made either way, and its verdict is an `EnlistedFact`, accepted or not.

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

Every throw the page prints with characteristic DMs is made through
`against`, which keeps beside the DM the prerequisites that earned it, so the
record can show where the number came from (#487).

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

Every strategy of the policy answers yes (section 6), so on an auto run the
`!submit` branch is never taken; only a person on a guided run declines, and
only that makes a civilian. E001 is stamped on the character, but no line of
the note shows it. The record reads "rejected by the Navy", with no pronoun: a
character has no sex in these rules (p. 8), and no note names one (#459).

`internal/chargen/testdata/drafted-then-commissioned.md` — `enlistment`

```markdown
### 15. enlistment (pp. 5, 10)

16. enlistment, Navy: rolled 2+3 = 5 against 8+, missed
17. rejected by the Navy (from step 16)
18. submit to the draft (p. 5): policy chose yes from yes, no
19. draft: rolled 5
20. drafted into the Merchants (from step 19)
```

`join` enters the service. It takes what p. 23's Rank and Service Skills box
grants on entering, once (E005), and applies each grant as an `AppliedFact`
whose source is `OnEntering`. The box prints nothing for the Merchants, so no
step appears here. An Army character gets Rifle-1 under a "rank and service
skills" step, as the guided transcript in section 6 shows.

### A term

`serve` runs terms until one ends the service. `term` follows the
exposition's order (E002): survival, commission, promotion, skills,
reenlistment, and then the aging round (E006). It opens with `record.opens`.

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

	err = r.trainSkills(term)
	if err != nil {
		return false, err
	}

	// The term's four years are counted before the reenlistment throw, whose
	// outcome carries the age the term left him at.
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

A failed survival throw is death (p. 5). The fatal term's four years are
counted as well (E004), and the `DiedFact` carries both the term and the age.

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
prerequisites that earn it (#482). `raise` sets the working rank and records
a `RaisedFact` with the rank's number and its name. It applies any rank-keyed
grants, each sourced `AtRank`, and tallies the skill eligibility the rank
carries under how the rank came, so the term's skills can say what gave them.

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
`trainSkills` takes the tally whole, clears it, and opens a phase step that
says how many rolls the term gave and what gave them, from the same tally the
rolls are spent from (#576).

`internal/chargen/serve.go` — `run.trainSkills`

```go
	earned := r.eligible

	r.eligible = traveller.Earned{}

	if earned.Rolls() == 0 {
		return nil
	}

	r.record.phase(term,
		fmt.Sprintf("skills and training, %d %s: %v", earned.Rolls(), rollsWord(earned.Rolls()), earned),
		"pp. 6, 11")
```

`Earned` writes itself as its sources, leaving out a source that gave none,
so term 2's step reads "3 rolls: 1 for the term, 1 for the commission, 1 for
the promotion". The guided run's skills question says it the same way.

`internal/traveller/choice.go` — `Earned.String`

```go
func (e Earned) String() string {
	return gaveRolls([]rollSource{
		{e.Term, "the term"},
		{e.Commission, "the commission"},
		{e.Promotion, "the promotion"},
	})
}
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

`offeredTables` tests Education at each designation, so the fourth table
(Education 8+) appears from the first designation after Education reaches 8.
That designation is preceded by a line that says so and points at the change
that did it (#576). The change is found where it happens.
`applyResult.Alteration` notes the event about to be written when the gate
goes from shut to open, and the next `trainOnce` writes the line and clears
the mark. Nothing shuts the gate again: no table lowers Education, and aging
leaves it alone.

`internal/chargen/enlist.go` — `applyResult.Alteration`

```go
	if !wasOpen && gate.Open(a.run.profile) {
		a.run.fourthTableOpenedBy = a.run.record.next()
	}
```

`internal/chargen/serve.go` — `run.trainOnce`

```go
	if r.fourthTableOpenedBy != 0 {
		gate := r.tables.Education()
		r.record.outcomef(r.fourthTableOpenedBy, nil, "%v %d, so this designation offers the %v",
			gate.Characteristic, r.profile[gate.Characteristic], gate.Table)

		r.fourthTableOpenedBy = 0
	}
```

No golden reaches that line. `TestTheRecordSaysWhenEducationOpensTheFourthTable`
holds it over the fixtures and four hundred seeded characters in
`explained_test.go`, and fails if none of them reaches it. Our character's
Education stays 7, so the fourth table is never offered.

`apply` folds the `TableResult` sum, p. 12's three kinds, through
`applyResult`. A skill raises the working list and records the level it
reached. A characteristic change records both ends and the table's own
delta, so a change that the floor or ceiling held is not read as a change of
nothing (#409).

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

Here are terms 2 and 3 in the note. Each DM names the prerequisite that
earned it: the Merchants' survival DM is p. 10's "Intel 7+", in the domain's
words. Each skills table line says what the policy weighed (#564). The
default skills strategy, `advanced`, designates among the tables least
trained on, and the line gives the counts. Where tables tie, it says the
strategy's ranking took the first of them. Term 3's Personal Development
roll of 5 is the Merchants' Blade Cbt, and the weapon's line reads `drew`,
because the policy drew it.

`internal/chargen/testdata/drafted-then-commissioned.md` — `terms 2 and 3`

```markdown
### 34. term 2 (pp. 5-7)

35. survival: rolled 4+3 +2 (Intelligence 7+) = 9 against 5+, made
36. survived term 2 (from step 35)
37. attempt a commission (p. 6): policy chose yes from yes, no
38. commission: rolled 2+5 +1 (Intelligence 6+) = 8 against 4+, made
39. commissioned: 4th Officer (from step 38)
40. attempt promotion (p. 6): policy chose yes from yes, no
41. promotion: rolled 5+6 = 11 against 10+, made
42. promoted: 3rd Officer (from step 41)

#### 43. skills and training, 3 rolls: 1 for the term, 1 for the commission, 1 for the promotion (pp. 6, 11)

44. acquired skills table to consult (pp. 6, 11): policy chose Personal Development Table, least trained on (0, against 1 and 1), from Personal Development Table, Service Skills Table, Advanced Education Table
45. Personal Development Table: rolled 1
46. Strength +1, 7 to 8 (from step 45)
47. acquired skills table to consult (pp. 6, 11): policy chose Advanced Education Table, tied least trained on (1) with two others, the advanced strategy's first, from Personal Development Table, Service Skills Table, Advanced Education Table
48. Advanced Education Table: rolled 2
49. Mechanical-2 (from step 48)
50. acquired skills table to consult (pp. 6, 11): policy chose Service Skills Table, tied least trained on (1) with one other, the advanced strategy's first, from Personal Development Table, Service Skills Table, Advanced Education Table
51. Service Skills Table: rolled 5
52. Electronic-1 (from step 51)
53. reenlistment: rolled 4+6 = 10 against 4+, made
54. leave the service or remain for another term (pp. 6-7, 21): policy chose continue from continue, discharge
55. reenlisted for term 3 (from step 53)

### 56. term 3 (pp. 5-7)

57. survival: rolled 5+1 +2 (Intelligence 7+) = 8 against 5+, made
58. survived term 3 (from step 57)
59. attempt promotion (p. 6): policy chose yes from yes, no
60. promotion: rolled 3+4 = 7 against 10+, missed
61. not promoted this term (from step 60)

#### 62. skills and training, 1 roll: 1 for the term (pp. 6, 11)

63. acquired skills table to consult (pp. 6, 11): policy chose Personal Development Table, least trained on (1, against 2 and 2), from Personal Development Table, Service Skills Table, Advanced Education Table
64. Personal Development Table: rolled 5
65. weapon of expertise, specified immediately (p. 11): policy drew Cudgel from Dagger, Blade, Foil, Cutlass, Sword, Broadsword, Spear, Halberd, Pike, Cudgel, Bayonet
66. Blade Combat: Cudgel-1 (from step 64)
```

### Reenlistment and leaving

The reenlistment throw is made every term, before the intent is asked,
because a 12 and a failure both settle the question, and a question with one
answer is not asked. A 12 exactly is a `RetainedFact`, stamped E003 only past
the seventh term. A failure departs. A success asks the intent, where more
than one intent is legal.

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
the fifth term, leaving is a retirement whatever brought it on, and the
`RetiredFact` keeps whether the throw denied it. Below the fifth term, leaving
is a denied reenlistment if the throw refused the character, and a discharge
if the character chose to go. Each reenlistment fact carries the age at which
the term left the character, and that is how the finished character's age is
read.

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

From the fourth term on, the aging round closes the term (E006) under a phase
step of its own, and its saving throws are made in the table's row order
(E007). A failed throw records an `AgedFact` with both ends and the table's
reduction. A characteristic can be reduced to zero only because E010 opens
p. 4's floor to 0 under aging, and one that reaches zero goes straight to a
medical crisis. Surviving the crisis records a `RecoveredFact`, which carries
the value it came back to, the months of recovery, and the age they leave
the character at.

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
the page's word. The round is a fourth-level heading inside term 4.

`internal/chargen/testdata/drafted-then-commissioned.md` — `term 4`

```markdown
80. reenlistment: rolled 1+2 = 3 against 4+, missed
81. left the service after term 4: reenlistment denied (from step 80)

#### 82. aging, end of term 4 (pp. 7-9)

83. aging, Strength: rolled 5+1 = 6 against 8+, missed
84. Strength -1, 8 to 7 (from step 83)
85. aging, Dexterity: rolled 4+1 = 5 against 7+, missed
86. Dexterity -1, 6 to 5 (from step 85)
87. aging, Endurance: rolled 3+6 = 9 against 8+, made
88. Endurance holds (from step 87)
```

### Mustering out, the pension and the title

`rules.Muster.Rolls` returns the benefit rolls in two parts, for the terms
and for the rank, so the record can say what earned them from the sum the
procedure spends.

`internal/rules/rules.go` — `Muster.Rolls`

```go
func (m Muster) Rolls(terms int, rank traveller.Rank) traveller.BenefitRolls {
	rolls := traveller.BenefitRolls{ForTerms: terms * rollsPerTerm, Rank: rank}

	switch {
	case int(rank) >= minRankForTwoExtraRolls:
		rolls.ForRank = extraRollsForRank3Plus
	case int(rank) >= minRankForOneExtraRoll:
		rolls.ForRank = extraRollsForRank1or2
	}

	return rolls
}
```

`musterOut` opens the step with that count and spends it. When the third
roll goes on Table 2 with rolls still to take, it writes a line saying the
rest must go on Table 1 (#576), since they are taken there without a
question.

`internal/chargen/muster.go` — `run.musterOut`

```go
	r.record.step(fmt.Sprintf("mustering out, %d %s: %v", rolls, rollsWord(rolls), entitled), "pp. 7, 9, 21-23")

	onTableTwo := 0

	for n := range rolls {
		table, err := r.chooseMusterTable(onTableTwo)
		if err != nil {
			return err
		}

		seq, err := r.benefit(table)
		if err != nil {
			return err
		}

		if table != traveller.TableTwo {
			continue
		}

		onTableTwo++

		if left := rolls - n - 1; onTableTwo == maxRollsOnTableTwo && left > 0 {
			r.record.outcomef(seq, nil, "%d rolls on Table 2, the most allowed: the remaining %d %s must be on Table 1",
				maxRollsOnTableTwo, left, rollsWord(left))
		}
	}
```

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
cases record a `PassageFact`, a `TravellersAidFact`, a weapon, a
characteristic change sourced `AtMusteringOut`, or a ship. `benefit` returns
the roll's number, which is what the cap line names as its cause.

`internal/chargen/muster.go` — `run.benefit`

```go
	if table == traveller.TableTwo {
		r.benefits.Cash += cash
		r.record.fact(seq, nil, traveller.CashFact{Cash: cash, Total: r.benefits.Cash})

		return seq, nil
	}

	applied := &applyBenefit{run: r, because: seq}

	err = benefit.Fold(applied)
```

A weapon row is asked of the decider as `MusterWeapon`, offered the
category's list and, for expertise, the weapons of that category already
received. Since #570 it is handed a `Vary` as `Weapon` is, so the policy can
draw a weapon rather than take the first printed. The answer is a
`WeaponBenefit` sum, folded by `takeWeapon` into a `WeaponReceivedFact` or an
`ExpertiseFact`.

`internal/chargen/muster.go` — `applyBenefit.WeaponPick`

```go
	list := a.run.tables.Weapons(category)
	received := a.run.receivedIn(category)

	taken, err := a.run.decide.MusterWeapon(category, list, received, varyOnly{a.run.roll})
	if err != nil {
		return fmt.Errorf("taking a weapon benefit: %w", err)
	}

	took := &takeWeapon{run: a.run, because: a.because}

	err = taken.Fold(took)
```

The fact says whether a result is a repeat, not a fold. `receiveShip` records
a `ShipReceivedFact` for the first ship of its kind. A second Free Trader is a
`FreeTraderAgainFact`, which carries the ship as it now stands: ten years
older, and ten years further paid down. A second scout ship is lost and
carries no fact. P. 23 counts five receipts to own a Free Trader free and
clear and says nothing of a sixth, so E016 ages the ship and leaves the
payment term at zero.

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
below is the one E016 governs, and like every line it names no erratum.

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

Our character's mustering out has five rolls, four for the terms and one for
rank 2. The `cash` strategy puts the first three on Table 2, the cap line
follows the third, and the last two go to Table 1 without a question.

`internal/chargen/testdata/drafted-then-commissioned.md` — `mustering out`

```markdown
### 89. mustering out, 5 rolls: 4 for the terms, 1 for rank 2 (pp. 7, 9, 21-23)

90. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
91. Table 2, Cash Allowances: rolled 4
92. CR 20000, bringing the total to CR 20000 (from step 91)
93. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
94. Table 2, Cash Allowances: rolled 4
95. CR 20000, bringing the total to CR 40000 (from step 94)
96. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
97. Table 2, Cash Allowances: rolled 3
98. CR 10000, bringing the total to CR 50000 (from step 97)
99. 3 rolls on Table 2, the most allowed: the remaining 2 rolls must be on Table 1 (from step 97)
100. Table 1, Material Benefits: rolled 6
101. a Low Passage (from step 100)
102. Table 1, Material Benefits: rolled 6
103. a Low Passage (from step 102)
```

The Merchants pay a pension, but the character left after four terms, so
`pension` finds no row in p. 21's table and records nothing. The final Social
Standing is 7, as rolled, so `assessTitle` neither offers a title nor stamps
E011, and the record ends there. A character that is alive at the end and
eligible is asked `AssumeTitle`, and the answer is a `TitledFact`. A dead
character is assessed and not asked (E011).

## 6. A question, through the gate

Every question reaches the decider through `logging`, which `generate` wraps
around whatever decider it was given. `logging` implements every method of
`Decider`, one per choice point, so a choice point cannot reach the engine
without reaching the record. Each method names the **offer** and refuses a
question with only one legal answer. It hands the decider a copy of what is
offered, which the decider owns, and records the answer only if that answer
was offered. The three methods that hand over a `Vary` wrap it in a
`watchedVary` first, and the record says the answer was drawn exactly when
the decider drew through it.

`internal/chargen/logging.go` — `logging.Service`

```go
func (l logging) Service(from []traveller.EnlistmentOffer, vary Vary) (traveller.ServiceName, error) {
	offer := names(from, func(o traveller.EnlistmentOffer) string { return o.Service.String() })

	err := asked(traveller.ChoiceService, offer)
	if err != nil {
		return 0, err
	}

	watched := &watchedVary{vary: vary}

	chosen, err := l.to.Service(own(from), watched)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceEvent{
		Point: traveller.ChoiceService, Alternatives: offer, Chosen: chosen.String(), Drawn: watched.drew,
	})
}
```

`internal/chargen/logging.go` — `watchedVary.Among`

```go
func (w *watchedVary) Among(n int) int {
	w.drew = true

	return w.vary.Among(n)
}
```

`SkillTable` names the counts before the decider is asked, as it names the
offer, so what is recorded is what the decider was shown.

`internal/chargen/logging.go` — `logging.SkillTable`

```go
	return chosen, l.record(traveller.ChoiceEvent{
		Point: traveller.ChoiceSkillTable, Alternatives: offer, Chosen: chosen.String(), Taken: taken,
	})
```

`internal/chargen/logging.go` — `logging.record`

```go
func (l logging) record(answered traveller.ChoiceEvent) error {
	if !slices.Contains(answered.Alternatives, answered.Chosen) {
		return fmt.Errorf("%w: %v offered %v, and was answered %q",
			errNotOffered, answered.Point, answered.Alternatives, answered.Chosen)
	}

	l.into.choice(answered)

	return nil
}
```

On an auto run the decider is a `chargen.Policy`: three strategies, each
over a group of choice points. Three answers are drawn under every strategy:
the service, the weapon a combat result names, and the weapon a benefit
names. Drawing the service (#420) is why a referee's batch is not nearly all
Army, as it was when the policy took the likeliest enlistment.

`internal/chargen/policy.go` — `Policy.Service`

```go
func (p Policy) Service(from []traveller.EnlistmentOffer, vary Vary) (traveller.ServiceName, error) {
	return from[vary.Among(len(from))].Service, nil
}
```

The fixed answers are the same under all three strategies. Since #566 that
includes the draft: `oneterm` used to decline it, and a character who
declines has served no term, so `oneterm` had not done the one term it names.

`internal/chargen/policy.go` — `Policy.SubmitToDraft`

```go
func (p Policy) SubmitToDraft() (bool, error) { return true, nil }
```

`SkillTable` designates among the tables trained on least, and the skills
strategy's ranking decides among those. That is the reasoning the note's
skills lines word.

`internal/chargen/policy.go` — `Policy.SkillTable`

```go
	return prefer(thinnest(from), ranked), nil
```

`internal/chargen/policy.go` — `thinnest`

```go
func thinnest(from []traveller.SkillTableOffer) []traveller.SkillTable {
	fewest := slices.MinFunc(from, func(a, b traveller.SkillTableOffer) int { return a.Taken - b.Taken }).Taken

	thin := make([]traveller.SkillTable, 0, len(from))

	for _, offer := range from {
		if offer.Taken == fewest {
			thin = append(thin, offer.Table)
		}
	}

	return thin
}
```

`MusterWeapon` draws a weapon from the list (#570). `cash` and `goods` take
expertise in the first weapon of the category already received, which draws
nothing and so reads `chose` in the record. `arsenal` draws among the
weapons not yet received.

`internal/chargen/policy.go` — `Policy.MusterWeapon`

```go
	if p.Muster != MusterArsenal {
		if len(received) > 0 {
			return traveller.TakeExpertise{Weapon: received[0]}, nil
		}

		return traveller.TakeWeapon{Weapon: from[vary.Among(len(from))]}, nil
	}

	fresh := slices.DeleteFunc(slices.Clone(from), func(name traveller.WeaponName) bool {
		return slices.Contains(received, name)
	})
	if len(fresh) == 0 {
		fresh = from
	}

	return traveller.TakeWeapon{Weapon: fresh[vary.Among(len(fresh))]}, nil
```

`scouts-expertise` shows both in a row: the first blade benefit is drawn, and
the second is taken as expertise in it.

`internal/chargen/testdata/scouts-expertise.md` — `weapon benefits`

```markdown
135. weapon benefit, or +1 expertise in lieu (p. 22): policy drew Blade from Dagger, Blade, Foil, Cutlass, Sword, Broadsword, Spear, Halberd, Pike, Cudgel, Bayonet
136. a Blade (from step 134)
137. mustering out table to consult (pp. 7, 9): policy chose Table 1, Material Benefits from Table 1, Material Benefits, Table 2, Cash Allowances
138. Table 1, Material Benefits: rolled 4
139. weapon benefit, or +1 expertise in lieu (p. 22): policy chose expertise in Blade from Dagger, Blade, Foil, Cutlass, Sword, Broadsword, Spear, Halberd, Pike, Cudgel, Bayonet, expertise in Blade
140. expertise instead: Blade-1 (from step 138)
```

`Vary` is a `Roller` with everything but `Among` taken off it. It is wrapped
in a value (`varyOnly`), so no type assertion can hand the decider the dice
back. A die thrown by a decider would reach no event, and the record must
carry every die (ADR 0001).

`internal/chargen/roller.go` — `varyOnly`

```go
type varyOnly struct{ roller Roller }

func (v varyOnly) Among(n int) int { return v.roller.Among(n) }
```

On a guided run the decider is the command's `player`. It asks only what the
engine asked, with exactly the alternatives offered. It asks again rather
than guess at an answer it cannot read, and the prompt carries the range it
accepts (#461). It ignores the `Vary` it is handed, so a player's answer is
never recorded as drawn.

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

The skills question says which roll of how many it is and why, and lists each
table's six results beside its name.

`cmd/ctchargen/interactive.go` — `player.SkillTable`

```go
	question := fmt.Sprintf("Which skills table? Roll %d of %d: %s. It is designated before the die (p. 11)",
		roll.N, roll.Of.Rolls(), roll.Of)
```

The following transcript is of a guided session against a build of this
tree, given two answers and then the end of its input. It shows the end of
the characteristics and everything after. The dice are the engine's own, so
another run asks other questions. Everything here goes to standard error.

```text
$ printf '3\n1\n' | ctchargen new -o guided.md
...
14. UPP 755339 at age 18

15. enlistment (pp. 5, 10)


Which service will you try to enlist in? (p. 10)
  1) Navy (enlistment 8+, DM +0)
  2) Marines (enlistment 9+, DM +0)
  3) Army (enlistment 5+, DM +2 for Endurance 5+)
  4) Scouts (enlistment 7+, DM +0)
  5) Merchants (enlistment 7+, DM +1 for Strength 7+)
  6) Other (enlistment 3+, DM +0)
answer 1-6>
16. you chose Army
17. enlistment, Army: rolled 1+1 +2 (Endurance 5+) = 4 against 5+, missed
18. rejected by the Army (from step 17)

Rejected. Submit to the draft? (p. 5) Declining ends the generation with a character who never served.
  1) yes
  2) no
answer 1-2>
19. you chose yes
20. draft: rolled 3
21. drafted into the Army (from step 20)

22. rank and service skills (p. 23)

23. Rifle-1

24. term 1 (pp. 5-7)

25. survival: rolled 4+3 = 7 against 5+, made
26. survived term 1 (from step 25)

27. skills and training, 2 rolls: 2 for the term (pp. 6, 11)


Which skills table? Roll 1 of 2: 2 for the term. It is designated before the die (p. 11)
  1) Personal Development Table: +1 Strength, +1 Dexterity, +1 Endurance, Gambling, Brawling, +1 Education
  2) Service Skills Table: ATV, Air/Raft, Forward Observer, Blade Combat, Gun Combat, Gun Combat
  3) Advanced Education Table: ATV, Mechanical, Electronic, Tactics, Blade Combat, Gun Combat
answer 1-3>
Answered 3, 1; the session is lost and nothing was written.
ctchargen: generating: designating a skills table: the input ended while the procedure was still asking
```

`EventLine` writes the live view. A step is a plain numbered line there,
where the note makes it a heading, and a choice is echoed as `you chose`.
Every other line comes from the same `writeEvent` the note uses. The grant
on entering, `23. Rifle-1`, names no step it came from, because it follows
from entering the service and not from an event. When the input ends
mid-session, the player says what was answered and that nothing was written,
and the run fails, since a character finished by an answer nobody gave is
not the one asked for (ADR 0001, ADR 0003). `guided.md` was never made: the
note is opened only after the character is generated.

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

For our character, this works out as follows. Strength is rolled 7, raised
to 8 at event 46, and aged to 7 at event 84, so the profile reads 7.
Dexterity is rolled 6 and aged to 5. The age is the
`DeniedReenlistmentFact`'s 34. The sheet's UPP, 754777, is not the 764777
that event 14 wrote. Event 14 is the profile as rolled, and the sheet's UPP
is folded to the end.

`Terms` is the one count not read from an outcome. It is the number that the
last term-opening step carries, which counts the fatal term too (E004).

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
`historyFold`. A step that opens a term starts a `TermServed`, and each gain
is filed by its source: on entering, at a rank (after the raise that
conferred it), in the term, or at mustering out. Each is filed as a `Moved`,
the gain beside the value it moved from, so the sheet can tell a skill or a
characteristic before and after (#562).

`internal/chargen/history.go` — `historyFold.gained`

```go
func (h *historyFold) gained(gain traveller.Gain, source traveller.Source) {
	from := fromFold{levels: h.levels}

	// This fold cannot fail: every case only assigns.
	_ = gain.Fold(&from)

	moved := Moved{Gain: gain, From: from.from}

	switch source {
	case traveller.OnEntering:
		h.out.OnEntering = append(h.out.OnEntering, moved)
	case traveller.AtRank:
		// A grant at a rank follows the raise that conferred it, in the same
		// term, so the term has one.
		raised := h.serving().Raised

		latest := &raised[len(raised)-1]

		latest.Grants = append(latest.Grants, moved)
	case traveller.InTerm:
		h.serving().Gained = append(h.serving().Gained, moved)
	case traveller.AtMusteringOut:
		h.out.MusteredOut = append(h.out.MusteredOut, moved)
	}
}
```

A characteristic change carries its own `From`. A skill's level before the
gain is not on the fact, so `fromFold` keeps each skill's last level as the
fold walks the record.

`internal/chargen/history.go` — `fromFold.Skill`

```go
func (f *fromFold) Skill(skill traveller.Skill) error {
	f.from = f.levels[skill.Name]
	f.levels[skill.Name] = skill.Level

	return nil
}
```

The history also keeps what the record knew and the service record used to
leave out (#561): a term a 12 on the reenlistment throw required
(`Retained`), each medical crisis recovery with its value and months, a
retirement the reenlistment throw refused (`RetiredDenied`), and expertise
taken at mustering out instead of a repeat weapon. The benefits themselves,
the cash, the passages, the weapons and the ships, are the sheet's, and the
history leaves them out.

`Attempted` reads the service that the enlistment throw was made against from
its `EnlistedFact` (#397). It says nothing of who named that service: the
choice's line in the record says that.

`internal/chargen/attempt.go` — `Character.Attempted`

```go
		if enlisted, ok := outcome.Fact.(traveller.EnlistedFact); ok {
			return enlisted.Service, true
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
`traveller.ServiceNames` order. The Navy's 8+ that our character missed is
the first cell here.

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
lift, where the judgement is named; that is the `Rifle-1` in the guided
transcript above. The second transcription, in the `rules` tests, is typed
separately and in the domain's words, so a misread cell cannot pass from one
copy to the other.

`internal/rules/transcription.go` — `transcribedSkills`

```go
	normalizations: map[string]string{
		"Blade Cbt":   "Blade Combat",
		"Gun Cbt":     "Gun Combat",
		"Fwd Obsv":    "Forward Observer",
		"Engnrng":     "Engineer",
		"Jack-o-T":    "Jack of all Trades",
		"Admin":       "Administration",
		"Electronics": "Electronic",
		"Rifl3":       "Rifle",
		"SMG":         "Submachine Gun",
	},
```

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

The fourth skills table's threshold is a `rules.Gate`, which `offeredTables`
and `applyResult.Alteration` both read through `Education()`, so the offer
and the line that announces it test the same thing.

`internal/rules/rules.go` — `Gate.Open`

```go
func (g Gate) Open(p traveller.Profile) bool { return g.Threshold.Satisfied(p[g.Characteristic]) }
```

A die's six faces size the skills tables here as `rules.Faces`, and the
`dice` package rolls by its own `dice.Faces`. There are two constants because
`dice` depends on nothing and `rules` may not import it. Each names the
other, and a test in `chargen`, which imports both, holds them equal (#514).

`internal/dice/dice.go` — `Faces`

```go
const Faces = 6
```

The readings live beside the tables as `rules.Reading` values, one per
erratum. `Readings` returns an array whose length is `len(traveller.Errata)`,
so a reading missing or one too many does not compile. There are sixteen,
E001 to E016. Each carries its pages, what the page says, the gap, the
reading, why, and the condition it is stamped on. The tree is the only place
a reading is written, and the command line does not print one.

`internal/rules/readings.go` — `Readings`

```go
func Readings() [len(traveller.Errata)]Reading { return readings }
```

## 9. Writing the note

`render.Note` writes the character in one pass from one `view`: the
properties, the sheet, and the generation record, which ends the note.

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
		Answerer:   answererOf(character.Policy),
		Events:     entries(character.Events),
		Build:      character.Build,
	}
```

The `answerer` is who answered every choice, read from whether the run had a
policy. On an auto run it also carries the skills strategy's name, which the
record uses where that strategy's ranking broke a tie.

`internal/render/view.go` — `answererOf`

```go
func answererOf(policy *chargen.Policy) answerer {
	if policy == nil {
		return answerer{Who: "player"}
	}

	return answerer{Who: "policy", ByPolicy: true, Skills: policy.Skills.String()}
}
```

### Properties

The note opens with YAML properties (#399). Their names and meanings freeze
at `v1.0.0`, and `writeProperties` states each meaning in its doc. Every note
carries every key: a text property with no value is an empty key, and a list
with no items is `[]`. `skills` is the names alone, so "which crew have
Pilot?" is an exact match (#474). `possessions` lists what mustering out gave
besides cash, one item each (#475). `ctchargen` names the build (#476).
Every text value is quoted, so nothing a later build writes can change a
property's type.

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
version, such as `"v1.0.0-rc.6"`.

### Heading, profile line and summary

`writeSheet` writes the heading, the profile line, the summary where it has
something to say, and three sections: skills, benefits and the service
record.

`internal/render/sheet.go` — `writeSheet`

```go
	fmt.Fprintf(out, "# %s\n\n%s\n\n", heading(r), profileLine(r))

	if told := summary(r); told != "" {
		fmt.Fprintf(out, "%s\n\n", told)
	}

	writeSection(out, "Skills", skillLines(r))
	writeSection(out, "Benefits", benefitLines(r))
	writeSection(out, "Service record", serviceLines(r))
```

An unnamed note is headed by its title: the service, the rank and the UPP. A
batch names the file by the same title (`render.Title`). The profile line
comes first under the heading (#557), as p. 25 leads with Jamison's: the UPP,
the age, the service and rank with how the service was entered, the terms,
any title assumed, and how the service ended. `entered` says how the
character came in: `(enlisted)`, or drafted after the service attempted
refused. Who named that service is not this line's to say (#583); the
record's choice line says it.

`internal/render/sheet.go` — `entered`

```go
func entered(r view) string {
	if r.Enlistment.Drafted {
		return fmt.Sprintf(" (drafted after the %s refused enlistment)", r.Attempted)
	}

	return " (" + r.Enlistment.How + ")"
}
```

`summary` follows, a paragraph a referee can read aloud (#425). It repeats
the profile line only for the dead, who are told first how and when they
died. Then it names the three highest-level skills at any level, ties in the
sheet's order (#558), and what was mustered out with that matters: cash,
retirement pay, Travellers' Aid and a ship.

`internal/render/summary.go` — `summary`

```go
	if r.Departure.Fatal {
		sentences = append(sentences, fmt.Sprintf("%s in term %d, aged %d.",
			capitalised(r.Departure.How), r.Terms, r.Age.Years))
	}

	if known := best(r.Skills); len(known) > 0 {
		sentences = append(sentences, "Skills include "+listed(known)+".")
	}

	if musters := notable(r.Benefits, r.Pension); len(musters) > 0 {
		sentences = append(sentences, "Musters out with "+listed(musters)+".")
	}
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `heading`

```markdown
# Merchants 3rd Officer 754777

UPP 754777, age 34, Merchants 3rd Officer (drafted after the Navy refused enlistment), 4 terms, denied reenlistment

Skills include Mechanical-2, Cudgel-1 and Electronic-1. Musters out with CR 50000.
```

### Skills and benefits

The skills are listed in the book's notation, `Skill-1`, in name order. The
benefits section lists the cash, then the possessions, then any retirement
pay. Passages of one class are one line with a count (#560), while the
`possessions` property keeps one item for each.

`internal/render/sheet.go` — `passageLines`

```go
	for _, passage := range order {
		if count[passage] == 1 {
			lines = append(lines, passage)

			continue
		}

		lines = append(lines, fmt.Sprintf("%s ×%d", passage, count[passage]))
	}
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `Skills and Benefits`

```markdown
## Skills

- Cudgel-1
- Electronic-1
- Jack of all Trades-1
- Mechanical-2
- Medical-1

## Benefits

- CR 50000
- Low Passage ×2
```

### The service record

`serviceLines` tells the **service history** a term at a time: how the
character entered and what entering granted, each term with something to
tell, how the service ended, and what mustering out changed. The departure
is the last that ended the service, so a crisis that kills a character
already retired is told, and the retirement it overtook is not.

`internal/render/sheet.go` — `serviceLines`

```go
	switch {
	case r.Departure.Characteristic != "":
		lines = append(lines, fmt.Sprintf("%s in term %d (%s reached zero)",
			r.Departure.How, r.Terms, r.Departure.Characteristic))
	case r.Departure.Fatal:
		lines = append(lines, fmt.Sprintf("%s in term %d", r.Departure.How, r.Terms))
	case r.History.RetiredDenied:
		lines = append(lines, fmt.Sprintf("%s after term %d, reenlistment denied", r.Departure.How, r.Terms))
	default:
		lines = append(lines, fmt.Sprintf("%s after term %d", r.Departure.How, r.Terms))
	}

	if len(r.History.MusteredOut) > 0 {
		lines = append(lines, "mustered out with "+strings.Join(r.History.MusteredOut, ", "))
	}
```

`termText` writes one term in the order it ran: the rank, the gains, a 12 that
required another term, the aging, and any recovery, leaving out each part the
term has none of.

`internal/render/sheet.go` — `termText`

```go
	retained := ""
	if term.Retained {
		retained = "the service required another term"
	}

	return joinNonEmpty("; ",
		strings.Join(raised, " and "),
		strings.Join(term.Gained, ", "),
		retained,
		labelled("aging ", strings.Join(term.Aged, ", "), ""),
		strings.Join(term.Recovered, ", "))
```

Each list of gains is told by `told`, which merges the moves of one skill or
characteristic into one entry, from the value before its first move to the
value after its last (#562). Where a floor or a ceiling held a
characteristic short of the table's change, it says where (#409). The
generation record still says each move on its own line.

`internal/render/history.go` — `told`

```go
	for _, name := range order {
		m := merged[name]

		words := fmt.Sprintf("%s %d→%d", m.name, m.from, m.to)
		if m.held {
			words += fmt.Sprintf(" (held at %d)", m.heldAt)
		}

		out = append(out, words)
	}
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `Service record`

```markdown
## Service record

- Merchants, drafted
- Term 1: Mechanical 0→1, Jack of all Trades 0→1
- Term 2: commissioned 4th Officer and promoted 3rd Officer; Strength 7→8, Mechanical 1→2, Electronic 0→1
- Term 3: Cudgel 0→1
- Term 4: Medical 0→1; aging Strength 8→7, Dexterity 6→5
- denied reenlistment after term 4
```

Our character mustered out with nothing but cash and passages, so there is no
last line. `other-crisis-survived` shows the parts ours lacks: a medical
crisis recovery in term 7, and two characteristics Table 1 raised at
mustering out. Intelligence went from 4 to 7 there over more than one roll,
and `told` writes it once.

`internal/chargen/testdata/other-crisis-survived.md` — `Service record`

```markdown
- Term 7: Social Standing 10→9; aging Strength 1→0, Endurance 7→6; Strength recovered from a medical crisis to 1 (1 month older)
- retired after term 7
- mustered out with Intelligence 4→7, Education 8→9
```

### The generation record

`writeGenerationRecord` writes every event under one `## Generation record`
heading, after a sentence that says what `(from step N)` names (#577).

`internal/render/record.go` — `writeGenerationRecord`

```go
	out.WriteString("## Generation record\n\n")
	out.WriteString("A line's `(from step N)` names the step it came from, most often the roll " +
		"it was read off, and not every cause behind it.\n")
```

Each step is a heading carrying its number (#577): a step outside every term,
and the step that opens one, at the third level, and a phase within a term
at the fourth. A heading also ends the Markdown list before it, so a viewer
numbers each list from its first line and shows each line's own number.
Every other event is a numbered line from the same `entry` that the live
view prints.

`internal/render/record.go` — `writeEvent`

```go
	switch event.Kind {
	case kindStep:
		level := "###"
		if event.Within != 0 {
			level = "####"
		}

		fmt.Fprintf(out, "\n%s %s\n\n", level, stepLine(event))
	case kindThrow, kindRoll:
		// One line for both: a roll is a throw whose target is empty, which
		// is exactly what throwLine already prints.
		fmt.Fprintln(out, throwLine(event))
	case kindChoice:
		fmt.Fprintln(out, choiceLine(event, answeredBy))
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

A choice's line names who answered, from the view's answerer, and reads
`drew` where the event says the answer was drawn (#564). For a policy's
skills table it adds the reason, from the counts the event carries.

`internal/render/record.go` — `choiceLine`

```go
func choiceLine(event entry, answeredBy answerer) string {
	verb := "chose"
	if event.Drawn {
		verb = "drew"
	}

	return fmt.Sprintf("%d. %s: %s %s %s%s from %s",
		event.Seq, choiceWords(event.Point), answeredBy.Who, verb, event.Chosen,
		leastTrained(event, answeredBy), strings.Join(event.Alternatives, ", "))
}
```

`leastTrained` writes nothing for a player, or for a choice without counts.
For the policy it has two forms: one table least trained on, with the
others' counts, or a tie, with how many others tied and the strategy whose
ranking took the first. A policy answer that was not least trained on is a
generation defect, and it panics rather than word a reason that was not the
policy's.

`internal/render/record.go` — `leastTrained`

```go
	if tied == 0 {
		return fmt.Sprintf(", least trained on (%d, against %s),", fewest, listed(others))
	}

	return fmt.Sprintf(", tied least trained on (%d) with %s, the %s strategy's first,",
		fewest, othersInWords[tied], answeredBy.Skills)
```

An outcome's line is its prose (the fact's `String`, where the outcome
carries a fact), followed by the event it came from. Nothing follows that:
no line names an erratum.

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

The note's last line is the record's last event: event 103 for our
character, `a Low Passage (from step 102)`.

## 10. `batch`

A batch is always an auto run. It writes only the living (#421), and only
into a folder of its own (#426). `--count` is decimal digits and nothing else
(#466). The flag package's own int reads base 0, so `010` would be eight and
`0x10` sixteen. `decimalCount` still answers `Get` with an int, so the CLI
baseline's `count int 0` holds.

`cmd/ctchargen/usage.go` — `decimalCount.Set`

```go
func (c *decimalCount) Set(value string) error {
	if strings.TrimLeft(value, "0123456789") != "" {
		return errNotAWholeNumber
	}
```

`-o` names a folder. An `-o` ending in `.md` almost certainly meant a file,
so it is refused outright, and `-` is standard output, as for `new`. A batch
makes its own folder and nothing above it (#470): the folder's parent must
already exist.

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
dead. It gives up with an `errSearch` after a hundred characters per member
asked for. The bound is checked by division, so a count near the top of an
int cannot overflow it (#423).

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
4 written, 2 passed over for dying
$ ls crew
1 Merchants 4th Officer 3689A8.md
2 Army Captain 578368.md
3 Army Captain B98AA7.md
4 Merchants 3rd Officer 5796D6.md
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
of the roster of thirty.

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

`v1.0.0` freezes the command line's flags, types, defaults and the values
each takes (#126, #591). `cli_test.go` types them out and holds them one way:
a flag or value removed or changed fails, and one added does not. Its comments
record what went on purpose before the freeze, `new`'s `readings bool false`
among them, and `TestTheReadingsAreNotOnTheCommandLine` holds that both the
subcommand and the flag are refused with exit 1.

`cmd/ctchargen/cli_test.go` — `cliBaseline`

```go
	cmdNew: {
		"auto bool false", "career string serve", baseForce,
		"muster string cash", "name string ", baseOutput,
		"service string ", "skills string advanced",
	},
```

The values sit beside the flags, typed out rather than read from `chargen`'s
alphabets, so the test cannot hold the code to itself.
`TestEveryBaselineValueIsStillTaken` runs each value through both commands
and reads it back off the flag's help line. #591 added them after `spartan`
became `arsenal` with nothing in the gate noticing.

`cmd/ctchargen/cli_test.go` — `valueBaseline`

```go
var valueBaseline = map[string][]string{
	flagCareer:  {serve, retire, oneTerm},
	flagSkills:  {advanced, serviceSkills, "personal"},
	flagMuster:  {cash, goods, "arsenal"},
	flagService: {navy, "marines", "army", "scouts", merchants, other},
}
```

### The stamping harness

`internal/chargen/stamping_test.go` holds every reading to its stamping
condition. It checks record by record, over every fixture and the package's
guided walks, in both directions, because an erratum stamped where its
condition does not hold is as much a defect as one missing. It reads each
`Character`, never a note, because the note carries no stamp. Each condition
is a predicate written by hand from the reading's stamped-on prose, never
from the stamping code, and a failure quotes the prose from
`rules.Readings`.

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
runs `go mod tidy -diff`, golangci-lint (which also checks formatting),
NilAway, govulncheck, `go test -race`, `bun fmt --check`, and an offline link
check. The test step also fails when the tests leave a new untracked file in
the tree. `task setup` installs every tool the gate runs: the Brewfile's
through `brew bundle`, and NilAway, which has no formula, through
`go install`.
