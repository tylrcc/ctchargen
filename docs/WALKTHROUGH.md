# Walkthrough

> Release snapshot for `v1.0.0-rc.4`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots"). `THEORY.md` explains why the system is shaped this way; this
> document follows one character through the code.

## Overview

`ctchargen` is a Go command that generates Classic Traveller characters by
walking Book 1's procedure (pp. 4–25 of the 1977 text) and writes each one as
a markdown **note** (ADR 0006): properties a vault can query, a heading and a
summary a referee can read aloud, the sheet, the **generation record** of
every die thrown and every choice made, and closing lines naming what wrote
it. The note is the only way back to a character; nothing reads one back. It
has four subcommands:

- `new`: one character, answered by the `--auto` policy (an **auto run**) or
  by a person at the terminal (a **guided run**);
- `batch`: many auto runs, each written as its own note into a folder of the
  batch's own, or one after another to standard output;
- `readings`: the readings a record's ids name, listed or written in full;
- `version`.

There are no dependencies beyond the standard library. The rules tables are
Go literals typed as the page prints them and **lifted** into the domain's
values once per process (ADR 0005). The readings, the decisions taken where
the page is silent, are Go values compiled in beside them (ADR 0004).

The one idea to hold through everything below: the generation record is the
character. Every outcome a sheet reads carries a typed `traveller.Fact`, and
everything the note says about the character — the UPP, the skills, the
possessions, the service record, the age, how the service ended — is folded
out of those facts. The engine keeps its own working copy of the character
while it runs, because every DM reads it, but the finished `Character` holds
only its log and what it was given.

## Architecture

```text
cmd/ctchargen/        the subcommands; stdout carries the note, stderr the conversation
internal/traveller/   the domain: alphabets, values, the sums, events and facts (imports nothing here)
internal/rules/       every table of pp. 4–25, transcribed in Go and lifted once, and the readings as values
internal/dice/        an unseeded stream of dice
internal/chargen/     the procedure, the Decider it asks, the auto run's Policy, and the folds that read the log
internal/render/      the note: properties, summary, sheet, generation record; and the readings' text
internal/scripted/    test support: the strict roller fixtures and walks play
internal/coverage/    test code only: holds the Implements: and Holds: markers to each other
internal/mutate/      a tool: runs a command against a scratch copy with one edit made
```

The arrows point one way. `traveller` imports none of the others; `rules` and
`chargen` import `traveller`, and `chargen` reads `rules` and rolls `dice`;
`render` imports `chargen` for the character, `traveller` for its values, and
`rules` for the readings' prose. Data flows in a line: flags become
`chargen.Inputs` and a `Policy` (or a person), `chargen` walks the procedure
against `rules` and a roller and hands back a `*chargen.Character` whose log
is the whole of it, and `render` folds that log into the note.

The character followed here is the golden
`internal/chargen/testdata/drafted-then-commissioned.md`: the run
`ctchargen new --auto --service navy` makes under the default policy, played
from a scripted roller. The Navy rejects him, the draft puts him in the
Merchants, he is commissioned and promoted in his second term, the
reenlistment throw forces him out after his fourth, and he musters out with
cash and two Low Passages.

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

`run` takes the two output channels apart. `out` carries the note and
`asking` is where the tool talks to a person, so `ctchargen new > kira.md`
saves a note and not a conversation. After answering help, it dispatches on
the subcommand.

`cmd/ctchargen/main.go` — `run`

```go
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
```

`chosen.dice` is a `source`: nil in a real run, where the engine rolls its own
unseeded stream, and a scripted roller only when a test passes `withDice`.
Every generation the command makes takes the same options, so the build name
has one place to come from (ADR 0002).

`cmd/ctchargen/main.go` — `source.options`

```go
func (s source) options() []chargen.Option {
	options := []chargen.Option{chargen.WithBuild(build())}

	if s != nil {
		options = append(options, chargen.WithRoller(s()))
	}

	return options
}
```

## 2. `new`: from flags to a generation

`new` and `batch` share the flags a character is generated from —
`--service` and the three strategies — declared once in
`declareCharacterFlags`, whose strategy defaults are read from
`chargen.DefaultPolicy` so the help cannot disagree with what an auto run
applies. `new` adds `--auto`, `--name`, `--readings`, `-o` and `--force`.

The strategies are parsed at this boundary and nowhere else. A `--service`
word becomes a `chargen.ForcedService`, one value that is either a service or
nothing, so no zero `ServiceName` (which is Navy) can stand in for "none".

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

`newCharacter` then refuses what it can refuse before anyone is asked
anything: a stray positional word, a `--name` that is not one line of UTF-8
(the note writes it as a heading and as a YAML string), and a strategy flag
without `--auto`, which would decide nothing on a guided run. What is left is
four steps.

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

	text := render.Note(character, how.readings)

	where, err := openDestination(out, how.output, how.force)
	if err != nil {
		return err
	}

	return where.write(text)
}
```

`refusedBefore` is new since rc.3 (#424). A guided session used to run to its
end and only then find that `-o` named a file already there, losing every
answer. Now the path is checked first: its folder must exist, and the path
itself must be absent unless `--force` is given, and never a directory.

`cmd/ctchargen/new.go` — `refusedBefore`

```go
func refusedBefore(path string, force bool) error {
	if path == "" {
		return nil
	}

	folder := filepath.Dir(path)
	if !isDirectory(folder) {
		return fmt.Errorf("%w: %s is not a folder that exists, so -o %s cannot be written",
			errUsage, folder, path)
	}

	_, err := os.Lstat(path)
	if err == nil && (!force || isDirectory(path)) {
		// inTheWay names a directory as one, with no --force advice (#254).
		return inTheWay(path)
	}

	return nil
}
```

A transcript of commands run against a build of this tree, from a scratch
folder (nothing re-runs it):

```text
$ ctchargen new --auto -o crew/x/y.md
ctchargen: usage: crew/x is not a folder that exists, so -o crew/x/y.md cannot be written
$ touch kira.md; ctchargen new -o kira.md
ctchargen: usage: kira.md already exists; pass --force to replace it
```

The second is a guided run refused before its first question. The check is
not the last word: without `--force`, the note is still written through an
exclusive open, which refuses a file that appeared during the session, and
with `--force` it is staged beside the path and renamed over it only once the
whole note is written (`replacing`).

`generate` is the fork between the two ways in. An auto run hands the policy
to the engine; a guided run builds a `player` and also watches the record as
it is written, so the throws reach the person between questions.

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
run: `Guided` passes nil, so the note can never name a policy that did not
answer.

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

`generate` loads the rules, builds a `run`, applies the options, wraps the
decider in the `logging` gate (section 6), and walks the procedure. When the
walk ends, the character is handed its events and its stamped errata, and
nothing else: there is no step that copies a profile, a skill list or a
career onto it.

`internal/chargen/generate.go` — `generate`

```go
	record := newLog()

	run := &run{
		tables: tables,
		roll:   dice.New(),
		log:    record,
		char:   &Character{Inputs: in, Policy: policy},
		atAge:  traveller.NewAge(traveller.StartingAge),
	}

	for _, option := range options {
		option(run)
	}

	run.decide = logging{to: decider, log: record}

	err = run.generate()
	if err != nil {
		return nil, err
	}

	run.char.Events = record.events
	run.char.Errata = record.stamped()

	return run.char, nil
```

The `run` is where the character lives while he is being made. Its fields
are working state, and their comments say so: each change to them is logged
as a fact carrying the value it left, and the finished character is read from
those facts (#441).

`internal/chargen/generate.go` — `run`

```go
	// profile and skills are the character as the procedure has him so far,
	// which every DM and the gambling modifier read mid-run. They are the
	// engine's working state and not the record: each change to them is
	// logged as a fact carrying the value it left, and the finished
	// Character's Profile and Skills are read from those facts (#441).
	profile traveller.Profile
	skills  []traveller.Skill
...
	enlistment traveller.Enlistment
	terms      int
	rank       traveller.Rank
	atAge      traveller.Age
	departure  traveller.Departure
```

`run.generate` is the procedure's outline: roll the six, enlist, and either
stop (a civilian, assessed for a title) or serve, muster out if alive, take
the pension, and assess the title.

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

The `log` is an append-only list of `traveller.Event`, a sealed sum of five
kinds: a step entered, a throw against a target, a roll with nothing to meet,
a choice, and an outcome. Each is numbered by its place in the list, and each
is shown to a watcher, if there is one, as it is appended.

An outcome is where a fact rides. `log.fact` writes an `OutcomeEvent` whose
prose is the fact's own `String`, so the line a referee reads and the value a
fold reads cannot say different things (#412).

`internal/chargen/log.go` — `log.fact`

```go
func (l *log) fact(because int, errata []traveller.Erratum, fact traveller.Fact) {
	for _, e := range errata {
		l.errata[e] = true
	}

	l.add(traveller.OutcomeEvent{
		Seq: l.next(), Because: because, Description: fact.String(), Errata: errata, Fact: fact,
	})
}
```

Its sibling `log.outcomef` writes an outcome with prose and no fact. It is
kept for outcomes nothing is read from: "survived term 1", "not promoted this
term", an aging throw that holds, a second Travellers' Aid wasted, a second
scout ship lost, a dash row. Both stamp the errata they are handed, and
`stamped` later returns every erratum any outcome named, in id order.

`traveller.Fact` is a sealed sum like the others: a `Fold` over a
`FactCases` interface with one method per kind, so a new kind of fact stops
every fold compiling until it is handled.

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
	ForcedOut(f ForcedOutFact) error
	Discharged(f DischargedFact) error
}
```

There are twenty-six. The pattern for each is the same: the fact carries the
value the engine reached, not the change that reached it, so a fold only
assigns. A skill fact carries the level it reached, a cash fact the running
total, a reenlistment fact the age the term left him at.

## 5. The procedure, one character through it

### The six rolls

`rollProfile` throws two dice per characteristic in p. 4's order. Each roll
is a `RollEvent` and a `RolledFact`; after the six, a `StartedFact` writes the
rolled UPP and the age every character starts at.

`internal/chargen/generate.go` — `run.rollProfile`

```go
func (r *run) rollProfile() {
	r.log.step("characteristics", "p. 4")

	for _, c := range traveller.Characteristics {
		first, second := twoDice(r.roll)

		r.profile[c] = first + second

		if c == traveller.SocialStanding {
			r.rolledSocial = r.profile[c]
		}

		seq := r.log.dice(c.String(), first, second)
		r.log.fact(seq, nil, traveller.RolledFact{Characteristic: c, Value: r.profile[c]})
	}

	r.log.fact(0, nil, traveller.StartedFact{Profile: r.profile, Age: r.atAge})
}
```

In the note, each outcome names the event that caused it, `(from N)`:

`internal/chargen/testdata/drafted-then-commissioned.md` — `characteristics`

```markdown
**1. characteristics (p. 4)**

  2. Strength: rolled 6+1
  3. Strength 7 (from 2)
  4. Dexterity: rolled 5+1
  5. Dexterity 6 (from 4)
  6. Endurance: rolled 2+2
  7. Endurance 4 (from 6)
  8. Intelligence: rolled 5+2
  9. Intelligence 7 (from 8)
 10. Education: rolled 3+4
 11. Education 7 (from 10)
 12. Social Standing: rolled 4+3
 13. Social Standing 7 (from 12)
 14. UPP 764777 at age 18
```

`rolledSocial` is kept beside the profile because E011, the title reading,
turns on whether the Social Standing at the end differs from the one rolled.

### Enlistment and the draft

`--service navy` forced the attempt, so `chooseService` returns the Navy
without asking. Unforced, it would build one `EnlistmentOffer` per service
and ask the decider, which on an auto run now draws the service (#420,
section 6). The throw is made either way, and its verdict is an
`EnlistedFact`, accepted or not: that fact is where the service attempted is
read from afterwards (#397).

`internal/chargen/enlist.go` — `run.enlist`

```go
	throw := roll(r.roll, service.Enlistment.Target, service.Enlistment.Modifier(r.profile))
	seq := r.log.throw("enlistment, "+name.String(), throw)

	if throw.succeeded {
		r.log.fact(seq, nil, traveller.EnlistedFact{Service: name, Accepted: true})

		return r.join(traveller.Enlisted{Service: name})
	}

	r.log.fact(seq, nil, traveller.EnlistedFact{Service: name})

	return r.draft()
```

A 5 against the Navy's 8+ misses. P. 5 prints the draft both ways, "may" and
"must"; E001 reads "may", so the draft is a question.

`internal/chargen/enlist.go` — `run.draft`

```go
	submit, err := r.decide.SubmitToDraft()
	if err != nil {
		return fmt.Errorf("submitting to the draft: %w", err)
	}

	if !submit {
		r.enlistment = traveller.DeclinedTheDraft{}
		r.log.fact(0, []traveller.Erratum{traveller.E001}, traveller.DeclinedFact{})

		return nil
	}

	face := r.roll.Die()

	name, err := r.tables.Draft(face)
	if err != nil {
		return fmt.Errorf("the draft: %w", err)
	}

	seq := r.log.dice("draft", face)
	r.log.fact(seq, []traveller.Erratum{traveller.E001}, traveller.DraftedFact{Service: name})

	return r.join(traveller.Drafted{Service: name})
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `enlistment`

```markdown
**15. enlistment (pp. 5, 10)**

 16. enlistment, Navy: rolled 2+3 = 5 against 8+, missed
 17. the Navy rejected him (from 16)
 18. submit to the draft (p. 5): policy chose yes from yes, no
 19. draft: rolled 5
 20. drafted into the Merchants (from 19) [E001]
```

`join` enters the service and takes what p. 23's Rank and Service Skills box
grants on entering it, once (E005), each grant applied as an `AppliedFact`
whose source is `OnEntering`. The box prints nothing for the Merchants, so no
step appears here; a Marine would get Cutlass-1 under a "rank and service
skills" step.

### A term

`serve` runs terms until one ends the service. `term` is the exposition's
order (E002): survival, commission, promotion, skills, reenlistment, then the
aging round (E006). It opens with `log.opens`, a step that carries the term's
number in `StepEvent.Opens`; that field is how the finished character counts
his terms, the fatal one included.

`internal/chargen/serve.go` — `run.term`

```go
func (r *run) term(term traveller.Term) (bool, error) {
	r.log.opens(term, "pp. 5-7")

	r.terms = int(term)

	if !r.survive(term) {
		return true, nil
	}

	err := r.commission(term)
	if err != nil {
		return false, err
	}

	err = r.promote(term)
	if err != nil {
		return false, err
	}

	r.grantEligibility(term)

	err = r.trainSkills()
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
counted (E004), and the `DiedFact` carries both the term and the age.

`internal/chargen/serve.go` — `run.survive`

```go
	r.atAge = r.atAge.PlusYears(traveller.Years)
	r.departure = traveller.KilledBySurvivalThrow{}
	r.log.fact(seq, []traveller.Erratum{traveller.E004}, traveller.DiedFact{Term: term, Age: r.atAge})
```

Our draftee survives. In term 1 `commission` asks nothing: p. 5 bars a
draftee from a commission in his first term only, and `r.drafted()` reads
that off the enlistment.

`internal/chargen/rank.go` — `run.commission`

```go
	throw, ok := r.service.Commission()
	if !ok || r.rank.Commissioned() || (r.drafted() && term == 1) {
		return nil
	}
```

In term 2 he is asked, makes the throw, and is commissioned; the same term he
is promoted (p. 6 allows both). `raise` sets the working rank, adds the
eligibility the rank carries, logs a `RaisedFact` with the rank's number and
its name, and applies any rank-keyed grants, each sourced `AtRank`.

`internal/chargen/rank.go` — `run.raise`

```go
func (r *run) raise(seq int, conferred traveller.Rank, eligibility int, how traveller.Advancement) error {
	r.rank = conferred

	r.eligible += eligibility

	r.log.fact(seq, nil, traveller.RaisedFact{How: how, Rank: conferred, Name: r.rankName()})
```

At the top of a service's rank column no promotion throw is made at all
(E013); the outcome says so and the decider is not asked.

### Skills

`grantEligibility` adds two eligibilities for the first term and one after;
each commission and promotion has already added one. `trainSkills` spends
them one at a time, and `trainOnce` asks for the table before the die, as
p. 11 requires, handing the decider how many results the character has
already had off each table.

`internal/chargen/serve.go` — `run.trainOnce`

```go
	table, err := r.decide.SkillTable(offered, taken)
	if err != nil {
		return fmt.Errorf("designating a skills table: %w", err)
	}

	if r.trained == nil {
		r.trained = make(map[traveller.SkillTable]int)
	}

	r.trained[table]++

	face := r.roll.Die()

	result, err := r.service.Result(table, face)
	if err != nil {
		return fmt.Errorf("consulting %v: %w", table, err)
	}

	seq := r.log.dice(table.String(), face)

	return r.apply(result, seq, []traveller.Erratum{traveller.E002}, traveller.InTerm)
```

`apply` folds the `TableResult` sum — p. 12's three kinds — through
`applyResult`. A skill raises the working list and logs the level it reached;
a characteristic change logs both ends and the table's own delta, so a change
the floor or ceiling held is not read as one of nothing (#409).

`internal/chargen/enlist.go` — `applyResult.Skill`

```go
func (a *applyResult) Skill(name traveller.SkillName) error {
	level := a.run.addSkill(name)
	a.run.log.fact(a.because, a.errata, traveller.AppliedFact{
		Gain: traveller.SkillGain{Skill: traveller.Skill{Name: name, Level: level}}, Source: a.source,
	})

	return nil
}
```

A Blade Cbt or Gun Cbt result names a weapon at once. The list is the page's,
and the decider is handed a `Vary` to draw from — the run's own dice and
nothing more — because the page gives no basis to choose.

`internal/chargen/enlist.go` — `applyResult.WeaponPick`

```go
	chosen, err := a.run.decide.Weapon(category, list, varyOnly{a.run.roll})
	if err != nil {
		return fmt.Errorf("choosing a weapon: %w", err)
	}

	level := a.run.addSkill(traveller.SkillName(chosen))
	a.run.log.fact(a.because, a.errata, traveller.WeaponPickedFact{
		Category: category, Skill: traveller.Skill{Name: traveller.SkillName(chosen), Level: level}, Source: a.source,
	})
```

Term 2 of our character, in the note:

`internal/chargen/testdata/drafted-then-commissioned.md` — `term 2`

```markdown
**34. term 2 (pp. 5-7)**

 35. survival: rolled 4+3 +2 = 9 against 5+, made
 36. survived term 2 (from 35)
 37. attempt a commission (p. 6): policy chose yes from yes, no
 38. commission: rolled 2+5 +1 = 8 against 4+, made
 39. commissioned: 4th Officer (from 38)
 40. attempt promotion (p. 6): policy chose yes from yes, no
 41. promotion: rolled 5+6 = 11 against 10+, made
 42. promoted: 3rd Officer (from 41)

**43. skills and training (pp. 6, 11)**

 44. acquired skills table to consult (pp. 6, 11): policy chose Personal Development Table from Personal Development Table, Service Skills Table, Advanced Education Table
 45. Personal Development Table: rolled 1
 46. Strength +1, 7 to 8 (from 45) [E002]
 47. acquired skills table to consult (pp. 6, 11): policy chose Advanced Education Table from Personal Development Table, Service Skills Table, Advanced Education Table
 48. Advanced Education Table: rolled 2
 49. Mechanical-2 (from 48) [E002]
```

His Education is 7, so the fourth table (Education 8+) is not offered:
`offeredTables` tests the current value at each designation.

### Reenlistment and leaving

The reenlistment throw is made every term, before the intent is asked,
because a 12 and a failure both settle the question and a question with one
answer is not asked. A 12 exactly is a `RetainedFact`; a failure departs; a
success asks the intent, where more than one is legal.

`internal/chargen/reenlist.go` — `run.reenlist`

```go
	attempt := roll(r.roll, r.service.Reenlist, 0)
	seq := r.log.throw("reenlistment", attempt)

	if attempt.dice[0]+attempt.dice[1] == forcedByTwelve {
...
		r.log.fact(seq, errata, traveller.RetainedFact{Age: r.atAge})

		return traveller.Continue, nil
	}

	if !attempt.succeeded {
		r.depart(seq, term, true)

		return traveller.Discharge, nil
	}

	return r.intent(seq, term)
```

`depart` decides the departure by the term count, as p. 21 does: at or past
the fifth term it is a retirement whatever brought it on; below it, forced
out if the throw refused him and discharged if he chose to go. Every one of
the five reenlistment facts carries the age the term left him at, which is
how the finished character's age is read.

`internal/chargen/reenlist.go` — `run.depart`

```go
	switch {
	case term >= firstRetiringTerm:
		r.departure = traveller.Retired{}
		r.log.fact(seq, nil, traveller.RetiredFact{After: term, Age: r.atAge, Denied: denied})
	case denied:
		r.departure = traveller.ForcedOut{}
		r.log.fact(seq, nil, traveller.ForcedOutFact{After: term, Age: r.atAge})
	default:
		r.departure = traveller.Discharged{}
		r.log.fact(seq, nil, traveller.DischargedFact{After: term, Age: r.atAge})
	}
```

### Aging

From the fourth term, the aging round closes the term (E006), its saving
throws made in the table's row order (E007). A failed throw logs an
`AgedFact` with both ends and the table's reduction; a characteristic reduced
to zero — possible only because E010 opens p. 4's floor to 0 under aging —
goes straight to a medical crisis, whose survival logs a `RecoveredFact`
carrying the months of recovery and the age they left him at.

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

	r.log.fact(seq, errata, traveller.AgedFact{
		Change: traveller.Change{Characteristic: characteristic, From: before, To: after, Delta: -reduction},
	})
```

Our character's fourth term ends both ways at once: the reenlistment throw
fails, and he is old enough for a round.

`internal/chargen/testdata/drafted-then-commissioned.md` — `term 4`

```markdown
 80. reenlistment: rolled 1+2 = 3 against 4+, missed
 81. left the service after term 4: reenlistment denied (from 80)

**82. aging, end of term 4 (pp. 7-9)**

 83. aging, Strength: rolled 5+1 = 6 against 8+, missed
 84. Strength -1, 8 to 7 (from 83) [E006 E007]
 85. aging, Dexterity: rolled 4+1 = 5 against 7+, missed
 86. Dexterity -1, 6 to 5 (from 85) [E006 E007]
 87. aging, Endurance: rolled 3+6 = 9 against 8+, made
 88. Endurance holds (from 87) [E006 E007]
```

### Mustering out, the pension and the title

`musterOut` reads the number of rolls off the terms and rank, and for each
asks for a table — unless three have already gone on Table 2, when p. 9
leaves only Table 1 and nothing is asked.

`internal/chargen/muster.go` — `run.chooseMusterTable`

```go
func (r *run) chooseMusterTable(onTableTwo int) (traveller.MusterTable, error) {
	if onTableTwo >= maxRollsOnTableTwo {
		return traveller.TableOne, nil
	}
```

`benefit` offers the +1 a table allows where it is earned, rolls one die, and
reads the row. Table 2 is cash, and the `CashFact` carries the running total;
Table 1 is folded through `applyBenefit`, whose cases log a `PassageFact`, a
`TravellersAidFact`, a `WeaponReceivedFact` or `ExpertiseFact`, or a ship.

`internal/chargen/muster.go` — `run.benefit`

```go
	if table == traveller.TableTwo {
		r.benefits.Cash += cash
		r.log.fact(seq, nil, traveller.CashFact{Cash: cash, Total: r.benefits.Cash})

		return nil
	}

	applied := &applyBenefit{run: r, because: seq}

	err = benefit.Fold(applied)
```

A repeat is the fact's to say, not a fold's. `receiveShip` logs a
`ShipReceivedFact` for a first ship of its kind; a second Free Trader is a
`FreeTraderAgainFact` carrying the ship as it now stands, aged ten years and
paid down; a second scout ship is lost and carries no fact.

`internal/chargen/testdata/drafted-then-commissioned.md` — `mustering out`

```markdown
**89. mustering out, 5 rolls (pp. 7, 9, 21-23)**

 90. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
 91. Table 2, Cash Allowances: rolled 4
 92. CR 20000, bringing the total to CR 20000 (from 91)
 93. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
 94. Table 2, Cash Allowances: rolled 4
 95. CR 20000, bringing the total to CR 40000 (from 94)
 96. mustering out table to consult (pp. 7, 9): policy chose Table 2, Cash Allowances from Table 1, Material Benefits, Table 2, Cash Allowances
 97. Table 2, Cash Allowances: rolled 3
 98. CR 10000, bringing the total to CR 50000 (from 97)
 99. Table 1, Material Benefits: rolled 6
100. a Low Passage (from 99)
101. Table 1, Material Benefits: rolled 6
102. a Low Passage (from 101)
```

The fourth and fifth rolls ask nothing: the cash strategy put three on
Table 2, so Table 1 is all that is left. He left after four terms, so
`pension` finds no row in p. 21's table and logs nothing. His final Social
Standing is 7, as rolled, so `assessTitle` neither offers a title nor stamps
E011, and the record ends there. A character eligible at the end and alive is
asked `AssumeTitle`, and the answer is a `TitledFact`; a dead one is assessed
and not asked (E011).

## 6. A question, through the gate

Every question reaches the decider through `logging`, which `generate` wraps
around whatever decider it was given. It implements every method of
`Decider`, one per choice point, so a choice point cannot reach the engine
without reaching the log. Each method names the offer, refuses a question
with one legal answer, hands the decider its own copy of what it is offered,
and records the answer only if it was offered.

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

	l.log.choice(point, from, chosen)

	return nil
}
```

On an auto run the decider is a `chargen.Policy`: three strategies, each over
a group of choice points, and `PolicyVersion`, which moves only when an
answer does. Version 4 (#420) draws the service where version 3 took the
likeliest enlistment, which had made a referee's batch nearly all Army.

`internal/chargen/policy.go` — `Policy.Service`

```go
func (p Policy) Service(from []traveller.EnlistmentOffer, vary Vary) (traveller.ServiceName, error) {
	return from[vary.Among(len(from))].Service, nil
}
```

`Vary` is a `Roller` with everything but `Among` taken off it, wrapped in a
value (`varyOnly`) so no type assertion can hand the decider the dice back:
a die thrown by a decider would reach no event, and the record must carry
every die. The other draw is `Weapon`; every other answer is a ranking over
what is offered, or a fixed yes or no. Our character's `--service navy`
suppressed `Service`, and his one draw was the weapon at event 65.

On a guided run the decider is the command's `player`. It asks only what
the engine asked, with exactly the alternatives offered, and asks again
rather than guessing at an answer it cannot read.

`cmd/ctchargen/interactive.go` — `player.choose`

```go
		typed := strings.TrimSpace(p.in.Text())

		chosen, err := strconv.Atoi(typed)
		if err == nil && chosen >= 1 && chosen <= len(options) {
			p.sayf("\n")

			p.given = append(p.given, chosen)

			return chosen - 1, nil
		}

		p.sayf("  %q is not one of them; answer with a number from 1 to %d\n",
			typed, len(options))
```

If the input ends mid-session, the player says what was answered and that
nothing was written, and the run fails: a character finished by an answer
nobody gave is not the one asked for (ADR 0001, ADR 0003).

## 7. Reading the character back

This is what changed most since rc.3 (#412, #397, #441). `Character` keeps
five fields, and none of them is a fact about the character.

`internal/chargen/character.go` — `Character`

```go
type Character struct {
	Inputs Inputs

	// Policy is what answered an auto run. It is nil on a guided run, where a
	// person answered and no policy decided anything. The version it
	// answered as is this build's PolicyVersion, which the record writes
	// beside it (#369).
	Policy *Policy

	Events []traveller.Event
	Errata []traveller.Erratum
...
	Build string
}
```

Everything else is a method that folds the log. `fold` walks the outcomes and
hands each fact to the cases given; `sheetFold` is the cases that read the
sheet.

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
`To` of every change after it; a skill takes the level its last fact
reached; the age and the departure take the last fact that moves them.

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

`internal/chargen/character.go` — `sheetFold.ForcedOut`

```go
func (s *sheetFold) ForcedOut(f traveller.ForcedOutFact) error {
	s.age, s.departure = f.Age, traveller.ForcedOut{}

	return nil
}
```

So for our character: Strength is rolled 7, raised to 8 at event 46, and aged
to 7 at event 84; the profile reads 7. Dexterity is rolled 6 and aged to 5.
The age is the `ForcedOutFact`'s 34. The UPP the sheet heads with, 754777,
is not the 764777 event 14 wrote: that line is the profile as rolled, and the
sheet's is folded to the end.

`Terms` is the one fact not read from an outcome. It is the number the last
term-opening step carries, which counts the fatal term too (E004).

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
`historyFold`: a step that opens a term starts a `TermServed`, and each fact
is filed by its source — on entering, at a rank (after the raise that
conferred it), or in the term. Mustering out's results and the possessions
are the sheet's, and the history leaves them out.

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
	}
}
```

`Attempted` reads the service the enlistment throw was made against from its
`EnlistedFact`, and whether the policy named it from the run's inputs: an
unforced auto run is the policy's choice, a `--service` or a guided answer is
not (#397).

`internal/chargen/attempt.go` — `Character.Attempted`

```go
		if enlisted, ok := outcome.Fact.(traveller.EnlistedFact); ok {
			_, forced := c.Inputs.Forced.Service()

			return Attempt{Service: enlisted.Service, ByPolicy: !forced && c.Policy != nil}, true
		}
```

The engine itself never calls these methods. Mid-run it reads its working
state (`r.profile`, `r.rank`, `r.terms`), and the golden notes hold the two
to each other: a fact that carried a value the working state did not reach
would change a note.

## 8. The rules the engine reads

Every number above came from `rules.Load`, which lifts every table once per
process and hands each caller the same `*Rules`. A table that will not lift
fails at first use, not on some path that reaches it.

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

Even the reprint's broken glyphs stay as printed: p. 23's box grants the Army
`"Rifl3-1"`, and E012's normalization map turns `Rifl3` into `Rifle` in the
lift, where the judgement is named. The second transcription, in the `rules`
tests, is typed separately and in the domain's words, so a misread cell
cannot pass from one copy to the other.

Lifted, a throw is a `rules.Throw`: a target and its DMs, summed where the
profile earns them.

`internal/rules/rules.go` — `Throw.Modifier`

```go
func (t Throw) Modifier(p traveller.Profile) int {
	total := 0

	for _, dm := range t.DMs {
		if dm.Threshold.Satisfied(p[dm.Characteristic]) {
			total += dm.Amount
		}
	}

	return total
}
```

The readings live beside the tables as `rules.Reading` values, one per
erratum, in an array whose length is `len(traveller.Errata)`, so a reading
missing or one too many does not compile. Each carries its pages, what the
page says, the gap, the reading, why, and the condition it is stamped on.

## 9. Writing the note

`render.Note` writes the character in one pass from one `view`.

`internal/render/note.go` — `Note`

```go
func Note(character *chargen.Character, readings bool) string {
	r := viewOf(character)

	var out strings.Builder

	writeProperties(&out, r)
	writeSheet(&out, r)
	writeGenerationRecord(&out, r)

	if readings {
		writeReadings(&out, r.Errata)
	}

	fmt.Fprintf(&out, "---\n\n%s\n\n%s\n", provenance(r), keepTheNote)

	return out.String()
}
```

Within `render`, `viewOf` is the only place the character's methods are
called, so the sheet and the generation record cannot read him two ways. It turns domain values
into the words the note prints.

`internal/render/view.go` — `viewOf`

```go
	profile, skills, age := character.Profile(), character.Skills(), character.Age()

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
		Errata:     character.Errata,
		Ruleset:    chargen.Ruleset,
		Policy:     projectPolicy(character.Policy),
		Events:     entries(character.Events),
		Build:      character.Build,
	}
```

### Properties

The note opens with YAML properties (#399). Their names and meanings freeze
at `v1.0.0`; the meanings are stated on `writeProperties`. Every text value
is quoted, so nothing a later build writes can change a property's type.

`internal/render/properties.go` — `writeProperties`

```go
	fmt.Fprintf(out, "upp: %s\n", strconv.Quote(r.UPP))

	for _, c := range traveller.Characteristics {
		fmt.Fprintf(out, "%s: %d\n", characteristicKeys[c], r.Profile[c])
	}

	fmt.Fprintf(out, "age: %d\n", r.Age.Years)
	property(out, "service", r.Service)
	property(out, "rank", r.RankName)
	fmt.Fprintf(out, "terms: %d\nstatus: %s\n", r.Terms, strconv.Quote(status(r)))
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `properties`

```markdown
---
name:
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
departure: "forced out"
```

`name` is empty: `new` was given no `--name`, and a batch names nobody
(#406). A referee names a character in the vault by setting `name` and
`aliases`.

### Heading, summary and headline

An unnamed note is headed by its title, the service and rank and the UPP,
which is also what a batch names the file by (`render.Title`). Under it,
`summary` writes a paragraph a referee can read aloud (#425): who he is, how
long and how he left, his skills at level 2 or above, and the possessions
that matter.

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
service on it is not the one attempted, `whoseService` says so — the case
our draftee is.

`internal/render/sheet.go` — `whoseService`

```go
	switch {
	case attempted != r.Service:
		return fmt.Sprintf(" (%s after the %s refused him)", r.Enlistment.How, attempted)
	case byPolicy:
		return " (service chosen by the policy)"
	default:
		// He asked for this service and is in it. He knows.
		return ""
	}
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `heading`

```markdown
# Merchants 3rd Officer 754777

Merchants 3rd Officer, 4 terms, forced out at 34. Known for Mechanical-2. Musters out with CR 50000.

UPP 754777, age 34, Merchants 3rd Officer (drafted after the Navy refused him), 4 terms, forced out
```

### The service record

`serviceLines` tells the service history a term at a time: how he entered
and what entering granted, each term with something to tell, and how the
service ended. `termText` joins a term's raises (with their grants in
parentheses), its gains, its aging and any recovery.

`internal/render/sheet.go` — `serviceLines`

```go
	entered := fmt.Sprintf("%s, %s", r.Enlistment.Service, r.Enlistment.How)

	entered = joinNonEmpty("; ", entered, strings.Join(r.History.OnEntering, ", "))

	terms := r.History.Terms

	lines := []string{entered}

	for _, term := range terms {
		told := termText(term)
		if told == "" {
			continue
		}

		lines = append(lines, fmt.Sprintf("Term %d: %s", term.Term, told))
	}
```

A change is written as the table gave it, and where a floor or ceiling held
it, where (#409): `Strength +1 (held at 15)`.

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
- forced out after term 4
```

### The generation record and the closing lines

`writeGenerationRecord` writes every event under one `## Generation record`
heading (#422), each step a bold line, and every other event a numbered line
from the same `entry` the live view prints for a guided run.

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

An outcome line is its prose — the fact's `String` where it carries one —
then the event that caused it and the errata it was stamped with.

`internal/render/record.go` — `outcomeLine`

```go
func outcomeLine(event entry) string {
	line := fmt.Sprintf("%3d. %s", event.Seq, event.Description)

	if event.Because != 0 {
		line += fmt.Sprintf(" (from %d)", event.Because)
	}

	if len(event.Errata) > 0 {
		line += " [" + strings.Join(event.Errata, " ") + "]"
	}

	return line
}
```

With `--readings` the full text of each reading the record names follows.
Then `provenance` names the ruleset, the build (only the command passes one,
so a golden names none), and who answered.

`internal/render/note.go` — `provenance`

```go
	if r.Policy == nil {
		return generated + "; every choice answered by the player."
	}

	return fmt.Sprintf("%s; every choice answered by the policy, version %d: career %s, skills %s, muster %s.",
		generated, r.Policy.Version, r.Policy.Career, r.Policy.Skills, r.Policy.Muster)
```

`internal/chargen/testdata/drafted-then-commissioned.md` — `closing lines`

```markdown
Generated under Classic Traveller, Books 1-3 (FFE reprint of the 1977 text); every choice answered by the policy, version 4: career serve, skills advanced, muster cash.

Keep this note whole: its generation record is the only way back to this character.
```

## 10. `readings`

A record names readings by id. `ctchargen readings` lists every one as its
erratum, title and status; with ids, it writes those in full, in the order
asked, and an id that names none is a usage error with nothing written.

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

`render.Reading` writes one reading's labelled parts verbatim, its page
emphasis included, and a withdrawn reading says so first.

## 11. `batch`

A batch is always an auto run, and since rc.3 it writes only the living
(#421) and only into a folder of its own (#426). `-o` must name a folder that
is new, or empty but for hidden entries; `ownFolder` checks that before
anyone is generated, and an `-o` ending in `.md` is refused outright.

`cmd/ctchargen/batch.go` — `ownFolder`

```go
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") {
			return fmt.Errorf("%w: %s is not empty; a batch writes into a folder of its own, a new or empty one",
				errUsage, dir)
		}
	}
```

`each` generates until it has the number asked for, passing over the dead,
and gives up with an error after a hundred characters per member asked for.
The bound is checked by division, so a count near the top of an int cannot
overflow it (#423).

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

Into a folder, each member is written as `NN Title.md`: its position,
zero-padded to the width of the count, and the same title an unnamed note is
headed by (#400).

`cmd/ctchargen/batch.go` — `batchRun.intoDirectory`

```go
	return b.each(
		func(index int, character *chargen.Character) error {
			where, err := openDestination(nil,
				memberPath(dir, position(index, b.count), render.Title(character)), false)
			if err != nil {
				return err
			}

			return where.write(render.Note(character, false))
		},
	)
```

A transcript of a batch run against a build of this tree; the dice are the
engine's own, so another run makes another crew. The closing line goes to
standard error.

```text
$ ctchargen batch --count 4 -o crew
4 written, 1 passed over for dying
$ ls crew
1 Other 4B4BCA.md
2 Scouts 997787.md
3 Army Major 789885.md
4 Merchants 3rd Officer 4869A9.md
$ ctchargen batch --count 1 -o crew
ctchargen: usage: crew is not empty; a batch writes into a folder of its own, a new or empty one
```

Four services in four members is the version 4 policy at work. Without `-o`
the notes go to standard output one after another, each written as it is
generated, so a large batch holds one character at a time.

## 12. How the tests drive it

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
scripted roller is strict both ways: a die asked for past the end fails the
test, and `Finished` reports one left unread, so a change to the order the
procedure consumes dice fails the fixture that felt it.

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

`TestGoldens` renders every fixture's note and compares it to its golden
under `internal/chargen/testdata/`; `-regenerate` rewrites them, and the
diff is read before it is kept. Because every fact the note reads is folded
from the log, the goldens are also what hold the folds to the engine: #441
moved every sheet fact into the log and no golden changed.

`internal/chargen/golden_test.go` — `TestGoldens`

```go
	for _, fixture := range fixtures {
		character := fixture.generate(t)

		golden(t, fixture.name+".md", []byte(render.Note(character, false)))
```

The command's own tests drive `run` with `withDice` and a scripted source —
**walks**, extracted from a record the tool wrote — so a guided session is
tested through the command, stderr questions and all (ADR 0003).

### The stamping harness

`internal/chargen/stamping_test.go` holds every reading to its stamping
condition, record by record and in both directions: an erratum stamped where
its condition does not hold is as much a defect as one missing. Each
condition is a predicate written by hand from the reading's stamped-on
prose, never from the stamping code, and a failure quotes the prose from
`rules.Readings`.

### The markers

Every rule an implementation carries is marked beside it, with its cite, and
every test that holds one says so; `internal/coverage` joins the two by the
rule's name and fails on either half missing. The map is printed on demand
and never committed.

`internal/coverage/coverage_test.go` — `coverage_test`

```go
//	// Implements: Six 2D rolls, in order [1:4].
//
// A test that holds a rule carries one line naming it:
//
//	// Holds: Six 2D rolls, in order.
```

The fixture above carries its own: `// Holds: Draftees are not eligible for
commission in the first term only.`, which the `Implements:` marker on
`run.commission` names. And a new invariant is not done until
`go run ./internal/mutate` has shown a deliberate edit to it killed, in a
scratch copy of the tree.
