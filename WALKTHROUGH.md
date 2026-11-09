# Walkthrough

> Release snapshot for `v1.0.0-rc.2`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots"). `THEORY.md` explains why the system is shaped this way; this
> document follows one character through the code.

## Overview

`ctchargen` is a Go command that generates Classic Traveller characters by
walking Book 1's procedure (pp. 4–25 of the 1977 text) and writes a JSON
**record** of everything that happened: every die, every choice and what was
offered, every consequence, and which recorded readings governed it. It has
five subcommands:

- `new`: one character, answered by the `--auto` policy or by a person at the
  terminal;
- `batch`: many auto characters;
- `render`: a record read back as a sheet or a transcript;
- `readings`: the readings the records name, listed or written in full;
- `version`.

There are no dependencies beyond the standard library at run time; the one
module requirement, a JSON Schema validator, is imported only by tests. The
rules tables are Go literals typed as the page prints them, lifted into the
domain's values once per process (ADR 0005). The readings, the decisions taken
where the page is silent, are Go values compiled in beside them (ADR 0004).

## Architecture

```text
cmd/ctchargen/        the subcommands; stdout carries the record, stderr the conversation
internal/traveller/   the domain: alphabets, values, sealed sums, events (imports nothing here)
internal/rules/       every table of pp. 4–25, transcribed in Go and lifted once, and the readings as values
internal/dice/        an unseeded stream of dice
internal/chargen/     the procedure, the Decider it asks, and the auto run's Policy
internal/render/      the record, the sheet and transcript projected from it, and the readings' text
internal/scripted/    test support: the strict roller fixtures and walks play
internal/coverage/    test code only: holds the Implements: and Holds: markers to each other
internal/mutate/      a tool: runs a command against a scratch copy with one edit made
```

The arrows point one way. `traveller` imports none of the others, `rules` and
`chargen` import `traveller`, and `render` imports the record, and `rules` for
the readings' prose. Data flows in a line: flags become `chargen.Inputs` and a
`Policy` (or a person), `chargen` walks the procedure against `rules` and a
roller and produces a `*chargen.Character` carrying its event log, and
`render` projects that into the record and the documents made from it.

## 1. The command

`main` hands everything to `run`, which takes the two output channels apart.
`out` is the data channel and `asking` is where the tool talks to a person,
so `ctchargen new | jq` pipes a record and not a conversation.

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

`run` dispatches on the subcommand. Its `options` exist for tests: the only
one, `withDice`, supplies a scripted roller.

`cmd/ctchargen/main.go` — `run`

```go
	switch args[0] {
	case "new":
		return newCharacter(args[1:], in, out, asking, chosen.dice)
	case "batch":
		return batch(args[1:], out, asking, chosen.dice)
	case "render":
		return renderRecord(args[1:], out)
	case "readings":
		return readings(args[1:], out)
	case "version":
		return version(args[1:], out)
```

The exit status is 0 or 1, whatever the failure (#121), and the reason goes to
stderr after `ctchargen:`.

## 2. `new`: from flags to a generation

`newCharacter` parses the flags, refuses a positional word (the flag package
stops at the first one, so `new foo --auto` would leave `--auto` unparsed),
and turns the strategy flags into a `chargen.Policy`. That is the one place
strategies are parsed: past it, a `Career` is a `Career`. Then it refuses the
flag combinations that cannot all be meant. Without `--auto`, a strategy flag
that was explicitly set is refused, because a person answers every choice it
would have decided (`strategiesWithoutAuto`). `--sheet` with `--history` is
refused (`flagsThatContradict`). And `--readings`, which ends a transcript
with the readings it names, is refused without `--history`, because on a
sheet or a record it would add nothing and say nothing:

`cmd/ctchargen/new.go` — `readingsWithoutHistory`

```go
func readingsWithoutHistory(readings, history bool) error {
	if readings && !history {
		return fmt.Errorf("%w: --readings ends a transcript, and needs --history", errUsage)
	}

	return nil
}
```

`render` makes the same check through the same function. What the flags say
to do with the character is gathered into one `newRendering`, and
`writeCharacter` is the whole of `new` after parsing: generate, render, write.

`cmd/ctchargen/new.go` — `writeCharacter`

```go
	character, err := generate(inputs, policy, how.auto, in, asking, dice)
	if err != nil {
		return err
	}

	text, err := rendered(character, how)
	if err != nil {
		return err
	}

	where, err := openDestination(out, how.output, how.force)
	if err != nil {
		return err
	}

	return where.write(text)
```

`generate` picks the way in. `--auto` is an **auto run** under the policy. A
run without it is a **guided run**, answered by a `player` that reads stdin and
watches the record as it is written, so each throw reaches the person between
questions.

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

Both branches take `dice.options()`. It is the one call every generation the
command makes, which is why the build name travels in it:

`cmd/ctchargen/main.go` — `options`

```go
func (s source) options() []chargen.Option {
	options := []chargen.Option{chargen.WithBuild(build())}

	if s != nil {
		options = append(options, chargen.WithRoller(s()))
	}

	return options
}
```

In production `s` is nil, so a generation rolls dice of its own. A test
passes a scripted source.

## 3. The engine's two ways in

`Auto` and `Guided` differ in one argument to the shared `generate`. An auto
run records the policy that answered it and the version of what it answers. A
guided run records none, and that absence is what says a person answered.

`internal/chargen/generate.go` — `Auto`

```go
func Auto(in Inputs, policy Policy, options ...Option) (*Character, error) {
	return generate(in, policy, &AutoPolicy{Policy: policy, Version: PolicyVersion}, options)
}
```

`internal/chargen/generate.go` — `Guided`

```go
func Guided(in Inputs, decider Decider, options ...Option) (*Character, error) {
	return generate(in, decider, nil, options)
}
```

`generate` loads the rules, builds a `run`, applies the options (a roller, an
observer, the build), wraps the decider in the `logging` gate, and walks.

`internal/chargen/generate.go` — `generate`

```go
	run := &run{
		tables: tables,
		roll:   dice.New(),
		log:    record,
		char: &Character{
			Name: in.Name, Age: traveller.NewAge(traveller.StartingAge), Inputs: in, Policy: policy, Ruleset: Ruleset,
		},
	}

	for _, option := range options {
		option(run)
	}

	run.decide = logging{to: decider, log: record}

	err = run.generate()
	...
	run.char.Events = record.events
	run.char.Errata = record.stamped()
```

The character is created before the options are applied, so `WithBuild` can
set `Character.Build` on it. It begins at `traveller.StartingAge`, p. 4's 18,
which is the domain's constant and not the engine's because the Aging Table's
lift reads it too (section 8). The decider is only ever reached through
`logging`, which is the subject of section 5.

## 4. The procedure

`run.generate` is the page order of Book 1, top to bottom.

`internal/chargen/generate.go` — `generate`

```go
func (r *run) generate() error {
	r.rollProfile()

	err := r.enlist()
	if err != nil {
		return err
	}

	_, served := r.char.ServedIn()
	if !served {
		return r.assessTitle()
	}

	err = r.serve()
	...
	if !r.dead {
		err := r.musterOut()
		if err != nil {
			return err
		}

		r.pension()
	}

	return r.assessTitle()
}
```

A dead character does not muster out. That is not a reading: the doc comment
above this method says the question was checked as a possible silence and
found settled by p. 5. The same form of comment, "Settled by the page and not
a reading", sits on each declaration whose question was checked that way, so
it is answered beside the code that implements the answer and is not
re-litigated. The 2D throw is one:

`internal/chargen/roller.go` — `roll`

```go
// Settled by the page and not a reading: checked as a possible silence at
// milestone 0, and kept so it is not re-litigated as one. Which throws are two
// dice. P. 10: "All rolls except draft are two-die throws." The draft is one
// die (p. 5); so are the skills-table roll (p. 11), each mustering-out roll
// (p. 9) and the medical-crisis recovery months (p. 7).
```

`rollProfile` throws six 2D rolls in p. 4's order. `enlist` chooses a service
(unless `--service` forced one), throws against its enlistment target, and on
failure goes to the draft, which can land anywhere. A forced service forces
the attempt only: the throw is still made.

`internal/chargen/enlist.go` — `enlist`

```go
	throw := roll(r.roll, service.Enlistment.Target, service.Enlistment.Modifier(r.char.Profile))
	seq := r.log.throw("enlistment, "+name.String(), throw)

	if throw.succeeded {
		r.log.outcomef(seq, nil, "enlisted in the %v", name)

		return r.join(traveller.Enlisted{Service: name})
	}

	r.log.outcomef(seq, nil, "the %v rejected him", name)

	return r.draft()
```

Every throw is logged as it is made, and every consequence names the event
that caused it (`seq`). That is what lets a transcript be walked against the
page.

### A term

`serve` runs terms until one ends the service. A term follows the order of
the exposition, not the worked example's (E002): survival, commission,
promotion, skills, reenlistment, then the aging round at the end of the term
(E006).

`internal/chargen/serve.go` — `term`

```go
	if !r.survive(term) {
		// The fatal term counts, and its four years with it (E004).
		r.char.Age = r.char.Age.PlusYears(traveller.Years)
		r.char.Departure = traveller.KilledBySurvivalThrow{}
		r.dead = true

		return true, nil
	}

	err := r.commission(term)
	...
	r.grantEligibility(term)

	err = r.trainSkills()
	...
	intent, forced, err := r.reenlist(term)
	...
	r.char.Age = r.char.Age.PlusYears(traveller.Years)

	err = r.agingRound(term)
```

Survival failure is death (p. 5), with no "injured instead". The outcome is
stamped with E004, the reading of how a dead character's age is recorded:

`internal/chargen/serve.go` — `survive`

```go
	r.log.outcomef(seq, []traveller.Erratum{traveller.E004},
		"died in term %d, aged %v", term, r.char.Age.PlusYears(traveller.Years))
```

Here is that path in full. It is the checked-in golden transcript of the
fixture `scouts-died-unasked`, a scout killed by his first survival throw
before anything was asked:

`internal/chargen/testdata/scouts-died-unasked.transcript.md` — `term 1`

```markdown
## 20. term 1 (pp. 5-7)

 21. survival: rolled 1+3 = 4 against 7+, missed
 22. died in term 1, aged 22 (from 21) [E004]
```

### A skill roll

Each eligibility earned this term is one roll. The decider designates the
table, the roller throws one die, and the service's table answers.

`internal/chargen/serve.go` — `trainOnce`

```go
	offered := r.offeredTables()

	taken := make([]int, len(offered))
	for i, table := range offered {
		taken[i] = r.trained[table]
	}

	table, err := r.decide.SkillTable(offered, taken)
	...
	face := r.roll.Die()

	result, err := r.service.Result(table, face)
	...
	seq := r.log.dice(table.String(), face)

	return r.apply(result, seq, []traveller.Erratum{traveller.E002})
```

`offeredTables` closes the Advanced Education Table below Education 8 (p. 11),
so the page decides what may be chosen. `taken` is a decision aid, how often
each table has been trained on, and it is not part of the record.

The auto policy's answer depends on `--skills`. Each strategy ranks the
tables, and `thinnest` keeps the tables trained on least before the ranking
breaks the tie (policy version 3, #34):

`internal/chargen/policy.go` — `SkillTable`

```go
	switch p.Skills {
	case SkillsAdvanced:
		ranked = []traveller.SkillTable{
			traveller.AdvancedEducationEight, traveller.AdvancedEducation,
			traveller.ServiceSkills, traveller.PersonalDevelopment,
		}
	...
	return prefer(thinnest(from, taken), ranked), nil
```

The rest of the procedure has the same shape:

- **reenlistment** throws, and a 12 exactly compels another term (p. 6);
- **the aging round** throws a saving throw per characteristic the table names
  for this term, and resolves a medical crisis inline when one reaches zero;
- **mustering out** spends the earned rolls, designating each one's table
  before its die, with at most three on Table 2.

`internal/chargen/aging.go` — `agingRound`

```go
	for _, effect := range effects {
		throw := roll(r.roll, effect.Saving, 0)
		seq := r.log.throw("aging, "+effect.Characteristic.String(), throw)

		if throw.succeeded {
			r.log.outcomef(seq, r.agingErrata(term), "%v holds", effect.Characteristic)

			continue
		}

		err := r.age(seq, term, effect.Characteristic, effect.Reduction)
```

## 5. A question, through the gate

Every `r.decide` call above lands on `logging`, which implements every
Decider method. Here is the skill-table question:

`internal/chargen/logging.go` — `SkillTable`

```go
	offer := names(from, traveller.SkillTable.String)

	err := asked(traveller.ChoiceSkillTable, offer)
	if err != nil {
		return 0, err
	}

	chosen, err := l.to.SkillTable(own(from), own(taken))
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceSkillTable, offer, chosen.String())
```

The order matters:

1. **The offer is rendered to names first**, so a decider that writes into its
   slice cannot slip its own answer in.
2. **`asked` refuses a question with only one answer.** The Decider contract
   is that a method is called only when more than one answer is legal, so a
   question with one is the engine's bug, and it fails rather than reaching the
   record as a choice.
3. **The decider gets its own copies (`own`)**, so a write to them changes
   nothing the engine offers later.
4. **`record` refuses an answer outside the offer**, then logs the choice.

`internal/chargen/logging.go` — `asked`

```go
func asked(point traveller.ChoicePoint, alternatives []string) error {
	if len(alternatives) < aChoice {
		return fmt.Errorf("%w: %v offered only %v; the engine asks only when more than one answer is legal",
			errOneAnswer, point, alternatives)
	}

	return nil
}
```

`internal/chargen/logging.go` — `record`

```go
	if !slices.Contains(from, chosen) {
		return fmt.Errorf("%w: %v offered %v, and was answered %q",
			errNotOffered, point, from, chosen)
	}

	l.log.choice(point, from, chosen)
```

The comparison is by rendered name, which is sound only because every
alphabet's `String()` is injective. `traveller`'s spelling tests hold that.
The choice event records the point, the alternatives and the answer, but not
who answered: one run has one answerer, and the record says it once, in
`policy`.

The one choice point that may vary its answer, which weapon a combat result
names, is handed a `Vary` rather than the roller: a value that can draw among
alternatives and do nothing else.

`internal/chargen/roller.go` — `varyOnly`

```go
type varyOnly struct{ roller Roller }

func (v varyOnly) Among(n int) int { return v.roller.Among(n) }
```

It is a struct and not the roller passed as an interface, because a type
assertion would hand the dice back (#136). A die a decider threw would reach
no event, and the record would stop carrying every die.

## 6. The log and the readings it stamps

Every event goes through the log, numbered in order. A consequence can name
readings, and the log keeps the union of them.

`internal/chargen/log.go` — `outcomef`

```go
func (l *log) outcomef(because int, errata []traveller.Erratum, format string, args ...any) {
	for _, e := range errata {
		l.errata[e] = true
	}
```

`internal/chargen/log.go` — `stamped`

```go
	for _, e := range traveller.Errata {
		if l.errata[e] {
			stamped = append(stamped, e)
		}
	}
```

`stamped` returns them in id order, and that list is the record's `errata`.
An id is a `traveller.Erratum`, a closed set of constants `E001` to `E015`,
and `traveller.Errata` lists them; a const block cannot count itself, so a
test that reads `erratum.go` holds the list to the constants (#333). The
engine names a reading only by its constant; what the reading says lives
in `rules`.

## 7. The readings, as values

Each reading is a `rules.Reading`: the erratum it is named by, a title, and
its argument in prose. That prose is kept verbatim, the page's own emphasis
included as inline markdown, so whatever prints it prints the decided words.

`internal/rules/readings.go` — `Reading`

```go
type Reading struct {
	ID    traveller.Erratum
	Title string
	...
	// StampedOn is the stamping condition: which records name the erratum.
	StampedOn string

	// Withdrawal is zero for a reading that stands.
	Withdrawal Withdrawal
}
```

Between the title and the stamping condition are the pages, what the page
says, the gap, the reading and why. The readings are one array, the one at
index `i` named by erratum `i+1`, and its size is the count check: the array
is sized by its literal, and `Readings` returns it as an array sized by the
errata, so a reading missing or one too many does not compile.

`internal/rules/readings.go` — `Readings`

```go
func Readings() [len(traveller.Errata)]Reading { return readings }
```

`internal/rules/readings.go` — `ReadingOf`

```go
func ReadingOf(e traveller.Erratum) Reading { return readings[e-1] }
```

That each value sits in its own erratum's place is a test's, `TestTheReadings`,
which also quotes E001's fields typed out by hand, so a field swapped with its
neighbour fails. `ReadingNamed` finds a reading by the id a record spells
(`"E004"`), and is what reads a record back to its readings.

An erratum names one decision for good. A reading whose decision no longer
holds is not deleted but withdrawn: its `Withdrawal` says why, and names the
erratum that replaced it, if one did. It stays in the array, so a record
already stamped with it still names a reading that exists. No reading is
withdrawn at this tag, so every path that handles one is exercised by a stand-in.
In rendering and the listing, the stand-ins are test-only readings built for
the purpose (`TestAWithdrawalNamesARealReplacement`, and
`internal/render/readings_test.go`); in the stamping harness, E001 under its
real predicate, treated as withdrawn (section 12).

## 8. The rules the engine reads

The tables are a **transcription**: Go literals in one file,
`internal/rules/transcription.go`, each cell typed as the page prints it, with
its glyphs, its abbreviations, its notation and its dashes (ADR 0005). The
structure comes from the domain and the cells from the page. Which service a
column belongs to is its position in `traveller.ServiceNames`, and a DM row
keeps the heading p. 10 prints over it, so the amount is read off the row the
way the page states it:

`internal/rules/transcription.go` — `transcribedServices`

```go
	enlistment: printedThrows{
		targets: perService{"8+", "9+", "5+", "7+", "7+", "3+"},
		dms: []printedDMRow{
			{
				heading:    "DM of +1 if",
				conditions: perService{"Intel 8+", "Intel 8+", "Dext 6+", "Intel 6+", "Stren 7+", "—"},
			},
```

So a table the types do not name, a table given twice or a cell too many does
not compile. Two cells swapped within a row still compile, as any two printed
cells would, and catching that is the second transcription's work (the end of
this section).

`rules.Load` lifts the transcription once per process. Every caller gets the
same `*Rules`, so every method that returns a slice returns a copy (#137).

`internal/rules/rules.go` — `Load`

```go
func Load() (*Rules, error) { return loaded() }

var loaded = sync.OnceValues(load)
```

`load` runs the lifts in order, each reading one transcribed table. Lifting
is the validation, and where the page's text becomes the domain's values: each
judgement about the text, a normalization or what a dash means, is written
down there rather than made in the transcriber's head. A Table 1 cell that is
no benefit (the font trap's bare "4") refuses the load, and with it the binary.
The grants are lifted after the Prior Service Table because p. 23's box labels
name a rank by its title, which is read off the Table of Ranks. The readings
are not among the lifts. They are not text to be parsed but values already
typed.

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

Every name a cell prints is parsed against `traveller`'s own alphabet, through
an index built from each value's `String()`:

`internal/rules/parse.go` — `named`

```go
func named[T fmt.Stringer](values []T) map[string]T {
	index := make(map[string]T, len(values))
	for _, v := range values {
		index[v.String()] = v
	}

	return index
}
```

So the spelling a record prints and the spelling the transcription is checked
against are the same function. A characteristic may also arrive abbreviated,
as the narrower tables print it ("Intel 8+", "+1 Stren"). Each abbreviation
opens exactly one of p. 4's six names, so `parseCharacteristic` reads it as a
question settled by the page and not a reading.

P. 12 makes skills "of three basic types: characteristic alterations …,
weapon expertise …, and basic skill". Only a basic skill's name is open, an
alphabet the code does not close, so the guards that matter sit in front of
it. A cell that opens with a sign and did not parse as an alteration would
otherwise lift as a skill whose name is the whole cell, and so would one that
opens with an en dash, which looks like a minus and is not one:

`internal/rules/parse.go` — `parseTableResult`

```go
	if isAlteration {
		return traveller.AlterationResult{Characteristic: characteristic, Delta: size}, nil
	}

	if glyph, _ := utf8.DecodeRuneInString(cell); glyph == traveller.EnDash {
		return nil, fmt.Errorf("%w: %q: %w", errMalformed, cell, traveller.ErrEnDash)
	}

	if startsWithSign(cell) {
		return nil, fmt.Errorf("%w: %q begins with a sign but is not a characteristic alteration", errMalformed, cell)
	}
```

Which glyphs are a sign is answered in one place, in `traveller`, and a
target reads the same function, so the two cannot disagree (#297):

`internal/traveller/target.go` — `Sign`

```go
func Sign(glyph rune) (int, bool) {
	switch glyph {
	case '+':
		return 1, true
	case '-', '\u2212':
		return -1, true
	default:
		return 0, false
	}
}
```

Three dashes look alike on screen, and the page prints only one of them where
a cell means nothing: the em dash, which the lift reads as no benefit, no
rank, no DM or no throw. A lone hyphen or en dash in that place is refused by
name. A Go array short of a cell is padded with `""` in silence, so an empty
cell is refused too, as a cell nobody read off the page:

`internal/rules/parse.go` — `refuseEmpty`

```go
func refuseEmpty(cell string) error {
	if strings.TrimSpace(cell) == "" {
		return fmt.Errorf("%w: an empty cell; the page prints a dash where it means nothing", errMalformed)
	}

	return nil
}
```

The Aging Table is typed with both of p. 9's header rows and each row's cells
as the spans their dotted leaders run across. The lift takes the bands from
where the cells begin, and holds the headers to the procedure: each age must
be the age its term ends at, which is why the starting age is `traveller`'s:

`internal/rules/lift.go` — `liftAgingHeaders`

```go
		if age != strconv.Itoa(traveller.StartingAge+traveller.Years*term) {
			return 0, 0, fmt.Errorf("%w: aging table: age %q is not the age term %d ends at", errMalformed, ages[i], term)
		}
```

The same header row carries the plus on 74+ that E014 rests on; the lift
refuses a plus anywhere else, or none on the last age, and holds only that
fact, not the reading.

A transcription broken two ways reports the same fault every run, so a lift
that finds faults in a loop walks the page's order rather than a map, whose
order Go randomises (#53). The skill tables are lifted in
`traveller.SkillTables` order, which is p. 11's, and each row's cells in the
order of services:

`internal/rules/lift.go` — `liftSkills`

```go
	for _, table := range traveller.SkillTables {
		for die, row := range t.tables[table] {
			for i, cell := range row {
				service := traveller.ServiceNames[i]

				result, err := parseTableResult(cell, r.normalize)
				if err != nil {
					return fmt.Errorf("acquired skills, %v, row %d, %v: %w", table, die+1, service, err)
				}
```

The names a table abbreviates are expanded by E012's normalizations, which
the transcription carries beside the tables they govern. They include the one
glyph the font broke in the page image itself: p. 23's box is typed as
printed, `Rifl3-1` among its grants, and no skill or weapon named `Rifl3`
exists in the three books, so the normalization reads it as Rifle.

`internal/rules/transcription.go` — `transcribedServices`

```go
	grants: []printedGrant{
		{label: "Navy Captain", grant: "+1 Social"},
		...
		{label: "Army", grant: "Rifl3-1"},
```

What the lift checks is the shape of a cell and the alphabet of a name, not
that a value is the one the page prints. That is the second transcription's
work, in this package's tests, laid out apart from the page's grid and written
in the normalized vocabulary, so it is independent of the first in shape and
in words. A field misnamed in a literal does not compile, so the slip #317
found, a misspelled field lifting as zero, cannot reach the lift.

## 9. The record and what is rendered from it

`render.JSON` projects the character into the record and marshals it. The
sheet and the transcript are projected from that same record, never from the
domain types, and `SheetFrom` and `TranscriptFrom` decode a record written
earlier into it. Reading a record back is the same code path as rendering a
fresh one.

`internal/render/json.go` — `JSON`

```go
func JSON(character *chargen.Character) ([]byte, error) {
	projected, err := project(character)
	if err != nil {
		return nil, err
	}
```

The `policy` object is present exactly when the character came from `Auto`:

`internal/render/json.go` — `projectPolicy`

```go
func projectPolicy(answered *chargen.AutoPolicy) *policyRecord {
	if answered == nil {
		return nil
	}

	return &policyRecord{
		Version: answered.Version,
		Career:  answered.Policy.Career.String(),
		Skills:  answered.Policy.Skills.String(),
		Muster:  answered.Policy.Muster.String(),
	}
}
```

The transcript walks the events in order. A choice's line names who answered,
which comes from the record, not the event:

`internal/render/transcript.go` — `transcriptOf`

```go
		if event.Kind == kindChoice {
			fmt.Fprintln(&out, choiceLine(event, r.answerer()))

			continue
		}

		writeEvent(&out, event)
```

`internal/render/transcript.go` — `choiceLine`

```go
func choiceLine(event eventJSON, answerer string) string {
	return fmt.Sprintf("%3d. %s: %s chose %s from %s",
		event.Seq, event.Point, answerer, event.Chosen,
		strings.Join(event.Alternatives, ", "))
}
```

On an auto record that reads `28. SkillTable: policy chose Advanced Education
Table from Personal Development Table, Service Skills Table, Advanced
Education Table`.

`Transcript` and `TranscriptFrom` take a `readings` flag, which is
`--readings`. With it, the transcript keeps its inline ids and, after the
last event, writes a section of its own:

`internal/render/transcript.go` — `transcriptOf`

```go
	out.WriteString("\n")

	if readings {
		writeReadings(&out, r.Errata)
	}

	endWithKeep(&out)
```

`writeReadings` reads the ids from the record's `errata`, so a record written
earlier is explained by the build reading it. It sorts them, because a record
written elsewhere need not list them in order, and it says so rather than
failing when a record names an id this build has no reading for: the rest of
the transcript is still the record's.

`internal/render/readings.go` — `writeReadings`

```go
// Implements: A transcript with its readings ends with each reading its record
// names, in full [CLI sketch].
func writeReadings(out *strings.Builder, errata []string) {
	...
	for _, id := range slices.Sorted(slices.Values(errata)) {
		r, ok := rules.ReadingNamed(id)
		if !ok {
			fmt.Fprintf(out, "### %s\n\nThe record names %s, and this build has no reading of that id.\n\n", id, id)

			continue
		}

		out.WriteString(Reading(r) + "\n")
	}
```

The `Implements:` line above the function is a marker; section 12 says what
reads it. `render.Reading` writes one reading as markdown: a heading of id and
title, then each labelled part in the order a referee argues it. A withdrawn
reading says so before anything else:

`internal/render/readings.go` — `Reading`

```go
	if r.Withdrawn() {
		fmt.Fprintf(&out, "**Withdrawn.** %s", r.Withdrawal.Why)

		if r.Withdrawal.ReplacedBy != 0 {
			fmt.Fprintf(&out, " Replaced by %v.", r.Withdrawal.ReplacedBy)
		}

		out.WriteString("\n\n")
	}
```

One fixture, `army-serve`, keeps its transcript a second time with its
readings, as `army-serve.readings.transcript.md`; it names E002, whose prose
carries the page's italic, so the golden shows that emphasis arriving
unchanged.

The sheet never carries the readings. Its headline explains a service the
character did not ask for. It finds the service attempted (forced in
`inputs`, or the Service choice) and says the policy picked it only on an
auto record:

`internal/render/sheet.go` — `serviceAttempted`

```go
	if r.Inputs.Service != "" {
		return r.Inputs.Service, false, nil
	}

	choice, asked, err := choiceAt(r, traveller.ChoiceService.String())
	if err != nil || !asked {
		return "", false, err
	}

	return choice.Chosen, r.Policy != nil, nil
```

An event that will not read fails both renderings rather than being skipped.
A record whose `policy` is a bare number (beta.5 and earlier) fails to decode,
so an old auto record is never read as a guided one.

## 10. `readings`

`ctchargen readings` is how an id is read out without a transcript to hand.
It has no flag set, as `version` has none: its words are ids. With none, it
lists every reading; with some, it writes those in full, in the order asked.

`cmd/ctchargen/readings.go` — `readings`

```go
	text := render.ReadingsIndex()

	if len(args) > 0 {
		var err error

		text, err = render.ReadingsNamed(args)

		switch {
		case errors.Is(err, render.ErrNoReading):
			return fmt.Errorf("%w: %s; %w; run `ctchargen readings` for the list", errUsage, readingsUsage, err)
		case err != nil:
			return fmt.Errorf("writing the readings: %w", err)
		}
	}
```

`ReadingsNamed` composes the whole text before returning any of it, so the
first id that names no reading fails the call and nothing is written:

`internal/render/readings.go` — `ReadingsNamed`

```go
	for i, id := range ids {
		r, ok := rules.ReadingNamed(id)
		if !ok {
			return "", fmt.Errorf("%q %w", id, ErrNoReading)
		}
```

A transcript of commands run while writing this, trimmed:

```text
$ ctchargen readings
# Readings

| Erratum | Title | Status |
| --- | --- | --- |
| E001 | The draft is _may_, not _must_ | in force |
| E002 | The per-term order is the exposition's, not the example's | in force |
...
| E015 | A printed table governs over the worked example's stated result | in force |

$ ctchargen readings E099
ctchargen: usage: readings [E001 ...]; "E099" names no reading; run `ctchargen readings` for the list

$ ctchargen new --auto --sheet --readings
ctchargen: usage: --readings ends a transcript, and needs --history
```

## 11. `batch`

A batch is always an auto run. It has nobody to ask, so it declares no
`--auto`, and a command line passing one is refused by the flag package as a
flag `batch` does not have, not accepted and ignored (#336). Run while writing
this:

```text
$ ctchargen batch --count 1 --auto
ctchargen: usage: flag provided but not defined: -auto; run `ctchargen batch --help`
```

`batch` builds one `batchRun` value carrying the inputs, the policy, the
count, `--survivors`, the search bound and the dice. `write` picks the
destination:

- a directory takes one file per member, named by position;
- standard output streams JSONL a record at a time;
- a file is buffered, so a failed run opens nothing.

Collisions are checked from the planned paths before anyone is generated.

The member loop is shared by all three:

`cmd/ctchargen/batch.go` — `each`

```go
	for i := 0; done.written < b.count; i++ {
		if b.survivors && i >= b.count*b.attempts {
			...
			return done, fmt.Errorf(
				"%w: --survivors generated %d characters for %d living and found %d",
				errSearch, i, b.count, done.written,
			)
		}

		character, err := b.member(i)
		if err != nil {
			return done, err
		}

		if traveller.Fatal(character.Departure) {
			if b.survivors {
				done.passed++

				continue
			}

			done.died++
		}
```

`--survivors` passes over a dead character and generates another, so every
member written is still exactly what its dice made. It gives up at a hundred
attempts per character asked for, and when it does, that is a run-time error
and not a usage error.

## 12. How the tests drive it

A **fixture** in `internal/chargen` is a script (the dice and draws of one
path, in the order the procedure asks for them) plus inputs and a policy, and
its record is checked in as a golden. A **walk** in `cmd/ctchargen` is the
same idea for the command's tests, with no golden. Both play their script
through `internal/scripted`, whose roller fails the test when asked past the
end of the script, and whose `Finished` fails it when any of the script was
left unread.

`internal/scripted/scripted.go` — `Die`

```go
func (r *Roller) Die() int {
	r.t.Helper()

	if r.die == len(r.dice) {
		r.t.Fatalf("%s: the procedure asked for die %d, and the script holds %d",
			r.name, r.die+1, len(r.dice))
	}
```

A change to the order dice are consumed in changes which path a fixture walks.
No failure names the throw that moved. The roller names only an index, when a
script runs out or is left unread; a reorder usually re-paths a fixture and
changes its count, so it fails there and wherever the new path diverges; and a
change that moves no path at all fails nothing but a golden, whose message
says to regenerate. So the golden diff is read before it is regenerated (#289).
The fixture is then re-scripted from a record that walks the new path, and the
goldens are regenerated with `go test ./internal/chargen -regenerate`, which
also writes the schema's two published examples.

### The stamping harness

`stamping_test.go` reads every checked-in golden and holds its `errata` to a
predicate per reading. Each predicate is written by hand from the reading's
`StampedOn` prose, never from the stamping code, and a failure quotes that
prose as `rules` declares it. Whether a reading is withdrawn comes from
`rules` too:

`internal/chargen/stamping_test.go` — `conditions`

```go
	for i := range all {
		all[i].withdrawn = rules.ReadingOf(all[i].id).Withdrawn()
	}
```

`verdict` is what is wrong with how one record stamps one reading. A
withdrawn reading is held only to being stamped on no record: the goldens are
new records, and no new record may carry one. Its condition is not judged,
since a reading replaced by another still describes records the replacement
now governs (#332). A standing reading is stamped exactly where its condition
holds, in both directions:

`internal/chargen/stamping_test.go` — `verdict`

```go
	got := slices.Contains(rec.Errata, r.id.String())

	if r.withdrawn {
		if got {
			return []string{fmt.Sprintf("%s: stamps %v, which is withdrawn", rec.name, r.id)}
		}

		return nil
	}

	want := r.holds(rec)

	switch {
	case want && !got:
```

Every golden agrees with every reading, so the main stamping test never feeds
`verdict` a disagreement. Two tests do, from copies of the goldens with
E001's stamp flipped. `TestAStandingReadingRefusesADisagreement` strikes it
from records its condition fits and adds it to the rest, and expects each
refused. `TestAWithdrawnReadingOnAGoldenIsRefused` treats E001 as withdrawn,
under its real predicate: every golden stamping it is refused, and each with
the stamp struck passes, though E001's prose still describes it.

A reading can be **parked**, naming the issue where its prose and its code
disagree undecided: the harness counts its disagreements and asserts the
count, rather than judging it, unless it is withdrawn.

### The markers

Every implemented rule carries an `Implements:` marker with its authority in
brackets, and the test that holds it a `Holds:` marker naming the same rule.
`internal/coverage` is test code only. It parses the tree's Go, the fixture
roster and the schema's `$comment`s for markers, and fails when a rule is
named by an implementation and held by no test, held by a test and named by no
implementation, or cited to an authority that does not exist. A marker is one
rule's name and one cite:

`internal/coverage/coverage_test.go` — `implementsMarker`

```go
	implementsMarker = regexp.MustCompile(`^Implements: (.+) \[([^\]]+)\]\.$`)
	holdsMarker      = regexp.MustCompile(`^Holds: (.+)\.$`)
```

A cite is a page in a closed shape (`1:4`, `1:9 (E006, E007)`), an ADR that
exists in `docs/adr`, one of the v1 contract's promise labels, or a reading
alone; and the readings it may name are read from the values, not typed into
the check:

`internal/coverage/coverage_test.go` — `readAuthorities`

```go
	for _, r := range rules.Readings() {
		known.cites["("+r.ID.String()+")"] = true
	}
```

The check writes nothing. The map from each rule to its page, implementation
and test is what the markers already say, and `-print` writes it as a table
on demand:

`internal/coverage/coverage_test.go` — `TestTheCoverageMapHolds`

```go
	if *printTable {
		_, err := os.Stdout.WriteString(strings.Join(table(rows), "\n") + "\n")
```

Run from `internal/coverage` while writing this, `go test -run
TestTheCoverageMapHolds -print` began, trimmed to its first rows:

```text
| Rule | Authority | Implementation | Test |
| --- | --- | --- | --- |
| Die roll conventions: `N+`, `N−`, exact | 1:2–3 | `internal/traveller.Target` | `internal/traveller.TestParseTarget`, `internal/traveller.TestTargetSatisfied` |
| All characters begin at 18 | 1:4 | `internal/traveller.StartingAge` | `internal/chargen.TestGoldens` |
| Floor of 0 for an aging reduction | 1:4, 7–9 (E010) | `internal/traveller.Profile.AgeReduce` | `internal/traveller.TestAgeReduceFloorsAtZero` |
```

Each of these checks runs under `task`, the gate, with the rest of the suite;
there is no CI (#149), so it holds when someone runs it.
