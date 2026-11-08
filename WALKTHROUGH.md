# Walkthrough

> Release snapshot for `v1.0.0-beta.6`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots"). `THEORY.md` explains why the system is shaped this way; this
> document follows one character through the code.

## Overview

`ctchargen` is a Go command that generates Classic Traveller characters by
walking Book 1's procedure (pp. 4–25 of the 1977 text) and writes a JSON
**record** of everything that happened: every die, every choice and what was
offered, every consequence, and which recorded readings governed it. It has
four subcommands:

- `new`: one character, answered by the `--auto` policy or by a person at the
  terminal;
- `batch`: many auto characters;
- `render`: a record read back as a sheet or a transcript;
- `version`.

There are no dependencies beyond the standard library at run time. The rules
tables and the readings document are embedded in the binary.

## Architecture

```text
cmd/ctchargen/        the subcommands; stdout carries the record, stderr the conversation
internal/traveller/   the domain: alphabets, values, sealed sums, events (imports nothing here)
internal/rules/       every table of pp. 4–25 and ERRATA.md, embedded and lifted once
internal/dice/        an unseeded stream of dice
internal/chargen/     the procedure, the Decider it asks, and the --auto Policy
internal/render/      the record, and the sheet and transcript projected from it
internal/scripted/    test support: the strict roller fixtures and walks play
internal/coverage/    test code only: generates docs/COVERAGE.md from markers
internal/mutate/      a tool: runs a command against a scratch copy with one edit made
```

The arrows point one way. `traveller` imports none of the others, `rules` and
`chargen` import `traveller`, and `render` imports the record. Data flows in a
line: flags become `chargen.Inputs` and a `Policy` (or a person), `chargen`
walks the procedure against `rules` and a roller and produces a
`*chargen.Character` carrying its event log, and `render` projects that into
the record and the documents made from it.

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
	case "version":
		return version(args[1:], out)
```

The exit status is 0 or 1, whatever the failure (#121), and the reason goes to
stderr after `ctchargen:`.

## 2. `new`: from flags to a generation

`newCharacter` parses the flags, refuses a positional word (the flag package
stops at the first one, so `new foo --auto` would leave `--auto` unparsed),
and turns the strategy flags into a `chargen.Policy`. That is the one place
strategies are parsed: past it, a `Career` is a `Career`. Without `--auto`, a
strategy flag that was explicitly set is refused, because a person answers
every choice it would have decided (`strategiesWithoutAuto`).

`writeCharacter` is the whole of `new` after parsing: generate, render, write.

`cmd/ctchargen/new.go` — `writeCharacter`

```go
	character, err := generate(inputs, policy, how.auto, in, asking, dice)
	if err != nil {
		return err
	}

	text, err := rendered(character, how.sheet, how.history)
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
			Name: in.Name, Age: traveller.NewAge(startingAge), Inputs: in, Policy: policy, Ruleset: Ruleset,
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
set `Character.Build` on it. The decider is only ever reached through
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
Each reading's stamping condition is stated as prose in `ERRATA.md`, and
`stamping_test.go` holds every golden's `errata` to a predicate written from
that prose rather than from this code.

## 7. The rules the engine reads

`rules.Load` embeds every JSON table and `ERRATA.md` and lifts them once per
process. Every caller gets the same `*Rules`, so every method that returns a
slice returns a copy (#137).

`internal/rules/rules.go` — `Load`

```go
func Load() (*Rules, error) { return loaded() }

var loaded = sync.OnceValues(load)
```

`load` reads each file into a wire type that mirrors the JSON and nothing
more, then runs the lifts in order. Lifting is the validation: a column out of
p. 10's order, a Table 1 cell that is no benefit (the font trap's bare "4"),
or a reading with no `traveller.Erratum` constant refuses the load, and with
it the binary.

`internal/rules/lift.go` — `load`

```go
	for _, lift := range []func() error{
		func() error { return rules.liftPriorService(services) },
		func() error { return rules.liftGrants(services) },
		func() error { return rules.liftRetirement(services) },
		func() error { return rules.liftSkills(skills) },
		func() error { return rules.liftMustering(mustering) },
		func() error { return rules.liftAging(aging) },
		func() error { return rules.liftWeapons(weapons) },
		func() error { return rules.liftNobility(nobility) },
		func() error { return rules.liftShips(ships) },
		rules.liftReadings,
	} {
```

Every name in the data is parsed against `traveller`'s own alphabet, through
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

So the spelling a record prints and the spelling the data is checked against
are the same function.

## 8. The record and what is rendered from it

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

On an auto record that reads `22. SkillTable: policy chose Advanced Education
Table from Personal Development Table, Service Skills Table, Advanced
Education Table`.

The sheet's headline explains a service the character did not ask for. It
finds the service attempted (forced in `inputs`, or the Service choice) and
says the policy picked it only on an auto record:

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

## 9. `batch`

`batch` requires `--auto`, because it has nobody to ask, and builds one
`batchRun` value carrying the inputs, the policy, the count, `--survivors`,
the search bound and the dice. `write` picks the destination:

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

## 10. How the tests drive it

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
When the count changes, the script fails. When only the order changes, the
golden fails (#289). Either way the fixture is re-scripted from a record that
walks the new path, and the goldens are regenerated with
`go test ./internal/chargen -regenerate`, which also writes the schema's two
published examples.
