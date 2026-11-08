# ctchargen Walkthrough

## Overview

`ctchargen` is a Go CLI that generates Classic Traveller characters by walking
Book 1's character generation procedure, pp. 4-25 of the 1977 text. It has no
dependencies outside the standard library except a JSON Schema validator used
only in tests.

What it produces is not just a character. It produces a **record**: a JSON
document carrying the finished character _and_ an ordered log of every step
entered, every die thrown, every question answered and every consequence — with
each consequence naming the throw that caused it. A referee can walk that log
against the printed page and check the tool's work.

The entry point is `cmd/ctchargen/main.go`. Four subcommands:

| Command   | What it does                                                      |
| --------- | ----------------------------------------------------------------- |
| `new`     | generate one character, asking at every choice unless `--auto`    |
| `batch`   | generate many characters, `--auto` only                           |
| `render`  | read a record saved earlier and write it as a sheet or transcript |
| `version` | write the build                                                   |

Everything below follows the call chain from `main` outward. Each quoted
snippet is labelled with its file and named by its symbol, and is verbatim from
the source; a transcript is a command run while writing, and says so.

## Architecture

Six packages, and a seventh, `internal/coverage`, that is tests alone and
imports nothing outside them. Every package but the command is under
`internal/`, so nothing outside this module can import one: the tool is the
command, and there is no Go library (#199). The import arrows point one way. The compiler holds most of
that, because a reverse edge is an import cycle; the edges that would still
compile — `traveller` or `rules` importing `dice`, the command importing `rules`
or `dice` — are denied by `depguard` in `.golangci.yml`. It is also easy to
check.

A transcript of `go list -deps ./... | grep philoserf | sed 's|github.com/philoserf/ctchargen|.|'`, run while writing:

```text
./internal/dice
./internal/traveller
./internal/rules
./internal/chargen
./internal/render
./cmd/ctchargen
./internal/coverage
./internal/mutate
./internal/walkthrough
```

That listing is in dependency order — `dice` and `traveller` first because
neither imports anything else in the module. The edges themselves:

Each package's in-module imports, summarised from `go list -f '{{.Imports}}'` while writing:

```text
internal/traveller ->
internal/dice      ->
internal/rules     -> internal/traveller
internal/chargen   -> internal/dice internal/rules internal/traveller
internal/render    -> internal/chargen internal/traveller
cmd/ctchargen      -> internal/chargen internal/render internal/traveller
internal/coverage  ->
```

Read that as four layers:

- **`traveller`** — the domain. The alphabets Book 1 prints (six services, six
  characteristics, four skill tables), the values it works in (`Target`, `Age`,
  `Credits`, `Profile`), and the sums that say "exactly one of". It rolls no
  dice, reads no table and marshals no JSON.
- **`dice`** — an unseeded stream. It rolls and never judges; whether a throw met
  its target is a `Target`'s business, which is why `dice` can depend on nothing.
- **`rules`** — every printed table, embedded as JSON, lifted into domain values.
- **`chargen`** — the procedure itself, plus the `Decider` interface it asks
  through and the `Policy` that answers under `--auto`.
- **`render`** — projections: JSON, a character sheet, a transcript.
- **`cmd/ctchargen`** — flag parsing, the interactive player, file output.

`internal/coverage` has no non-test code at all. It generates
`docs/COVERAGE.md` from markers beside the code.

## Layer one: the domain

Start with `traveller`, because everything else is expressed in its vocabulary.

### Targets, and why the zero value fails closed

Book 1 pp. 2-3 fix the die-roll notation: `8+` means eight or more, `3-` means
three or less, and a bare number must be thrown exactly. A `Target` is that
number plus which of the three it is. `internal/traveller/target.go`, the modality
constants:

`internal/traveller/target.go` — `Modality`

```go
// The three modalities (pp. 2-3).
//
// Exactly is first, and deliberately: it makes the zero value of a Target
// "0 exactly", which no throw of dice can meet. A Target that was never set
// therefore fails closed. Were AtLeast the zero value, an unset target would
// read as "0+" and every throw in the procedure would succeed against it.
const (
	Exactly Modality = iota // N, with no sign
	AtLeast                 // N+
	AtMost                  // N-
)
```

`ParseTarget` reads the book's own notation, once, when a table lifts — never
again at throw time. It is deliberately strict about the sign, and the reason is
the reprint's font: under text extraction a printed minus becomes a digit, so an
`N-` target silently reads as `N3`. It also accepts two different minus glyphs,
because someone transcribing p. 11 types what he sees on the page.
`internal/traveller/target.go`, `minusSigns` and `ParseTarget`:

`internal/traveller/target.go` — `ParseTarget`

```go
// minusSigns are the two characters a minus may arrive as: the ASCII hyphen
// a data file is typed with, and the U+2212 the page is set in.
const minusSigns = "-−"

...

func ParseTarget(s string) (Target, error) {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) == 0 {
		return Target{}, fmt.Errorf("%w: it is empty", errTarget)
	}

	modality := Exactly

	if last := runes[len(runes)-1]; last == '+' || strings.ContainsRune(minusSigns, last) {
		if len(runes) == 1 {
			return Target{}, fmt.Errorf("%w %q: a sign with no number", errTarget, s)
		}

		modality = AtLeast
		if last != '+' {
			modality = AtMost
		}

		runes = runes[:len(runes)-1]
	}

	// P. 3 puts the sign of a throw after its number and the sign of a
	// modifier before it: "throws are always followed by a sign unless the
	// number must be thrown exactly, and DMs are always preceded by a
	// sign." A leading sign here means a DM was passed off as a target.
	digits := string(runes)
	if digits == "" || digits[0] == '+' || digits[0] == '-' || strings.HasPrefix(digits, "−") {
		return Target{}, fmt.Errorf("%w %q: a leading sign marks a modifier, not a throw", errTarget, s)
	}

	number, err := strconv.Atoi(digits)
	if err != nil {
		return Target{}, fmt.Errorf("%w %q: %w", errTarget, s, err)
	}

	return Target{number: number, modality: modality}, nil
}
```

Every refusal wraps the one sentinel, `errTarget`. It is unexported: nothing
outside `traveller` matches on it, and `rules` carries its message into the
error naming the table that would not lift.

### Sums: "exactly one of", held by the compiler

The domain's most important idea is how it closes a set of alternatives. Go's
type switch over an interface promises nothing — the compiler will not tell you a
case is missing, and neither will the `exhaustive` linter. So the six places
where "exactly one of" is the rule are written as **sealed interfaces with a fold**.
`internal/traveller/sums.go`, the file's opening comment:

`internal/traveller/sums.go` — `the sums`

```go
// The sums: the places where "exactly one of" is the rule.
//
// Each is an interface with a Fold method taking a cases interface that has
// one method per case, and an unexported seal so the set cannot be extended
// from outside this package. Adding a case adds a method to the cases
// interface, and every implementation stops compiling until it handles it.
//
// This is the same mechanism the Decider uses for choice points, and it is
// here for the same reason: Go's compiler does not check that a type switch
// covers every implementation of an interface, and neither does the
// exhaustive linter. A type switch over a sealed interface is an
// exhaustiveness promise that nothing keeps.
```

`Enlistment` is the smallest one, and it shows the pattern whole. There are
exactly three ways p. 5's enlistment step can end, and "declined the draft" is a
case of the type rather than a nil service with a flag beside it.
`internal/traveller/sums.go`, `Enlistment` through its last seal:

`internal/traveller/sums.go` — `Enlistment`

```go
// Enlistment is how — or whether — a character entered a service (p. 5).
//
// The civilian who declined the draft is a case of the type, not a nil
// service with a flag beside it. The draft is declinable (E001).
type Enlistment interface {
	Fold(cases EnlistmentCases) error
	sealedEnlistment()
}

// EnlistmentCases handles each way p. 5's enlistment step can end.
type EnlistmentCases interface {
	Enlisted(service ServiceName) error
	Drafted(service ServiceName) error
	DeclinedTheDraft() error
}

// Enlisted is a successful enlistment throw against the Prior Service Table
// (pp. 5, 10).
type Enlisted struct{ Service ServiceName }

// Drafted is entry by the draft: one die, and the service with that draft
// number — "possibly the very service which had just previously rejected his
// enlistment" (p. 5).
type Drafted struct{ Service ServiceName }

// DeclinedTheDraft ends generation with an eighteen-year-old civilian: no
// service, no terms, no skills, no benefits. A complete record (E001).
type DeclinedTheDraft struct{}

func (e Enlisted) Fold(c EnlistmentCases) error       { return c.Enlisted(e.Service) }
func (d Drafted) Fold(c EnlistmentCases) error        { return c.Drafted(d.Service) }
func (DeclinedTheDraft) Fold(c EnlistmentCases) error { return c.DeclinedTheDraft() }
func (Enlisted) sealedEnlistment()                    {}
func (Drafted) sealedEnlistment()                     {}
func (DeclinedTheDraft) sealedEnlistment()            {}
```

The six sums are `Enlistment`, `TableResult`, `BenefitRow`, `Departure`,
`WeaponBenefit` and `Event`. Their cases interfaces are where you will feel the
compiler if you add a case:

A transcript of `grep -n 'Cases interface' internal/traveller/*.go`, run while writing:

```text
internal/traveller/event.go:19:type EventCases interface {
internal/traveller/sums.go:26:type EnlistmentCases interface {
internal/traveller/sums.go:108:type TableResultCases interface {
internal/traveller/sums.go:148:type BenefitRowCases interface {
internal/traveller/sums.go:220:type DepartureCases interface {
internal/traveller/sums.go:271:type WeaponBenefitCases interface {
```

## Layer two: the tables

`rules` holds every table of pp. 4-25 as embedded JSON and lifts it into domain
values. The package's own doc comment says why the lift is where validation
lives. `internal/rules/rules.go`, the package doc:

`internal/rules/rules.go` — `package rules`

```go
// Package rules holds every table of Book 1 pp. 4-25 as embedded data, and
// lifts it into the domain values of the traveller package.
//
// The lift is the validation. A cell that names no service, no
// characteristic, no benefit any row prints, or a target in a notation
// pp. 2-3 do not use, fails here — and because loading happens once on first
// use, a table that will not lift is a build defect that surfaces
// immediately rather than a runtime condition some path might reach.
//
// The readings document, data/ERRATA.md, is embedded and lifted the same
// way. It stays hand-written prose; only its entries are parsed, and an
// erratum with no entry or an entry with no erratum fails the lift.
//
// Every table is transcribed twice: once into data/, and once into this
// package's tests, from the same visual reading of the page. The two must
// agree. That is what the reprint's font requires — its glyph substitutions
// look like data, so on Book 1 p. 9 the dash cells of Mustering Out Table 1
// extract as the digit 4 and "Travellers' Aid" extracts as Travellers9. A
// run that trusted the extraction would give a Scout a seventh benefit he
// does not have.
package rules
```

A transcript of `ls internal/rules/data/`, run while writing:

```text
aging.json
ERRATA.md
mustering.json
nobility.json
services.json
ships.json
skills.json
weapons.json
```

Each file carries its own page citation. Here is the head of the Prior Service
Table's data, `internal/rules/data/services.json` through its `draft` row — the six
services of p. 10, their enlistment throws, the die modifiers that apply, and the
draft number of each:

`internal/rules/data/services.json` — `services`

```json
{
  "_cite": "Book 1 p. 10 (Prior Service Table, Table of Ranks); p. 21 (Annual Retirement Pay); p. 23 (Rank and Service Skills)",
  "_note": "Transcribed from a visual read of the page. Never from extracted text: the reprint's font turns a printed minus into a digit and an em dash into a 4.",

  "services": ["Navy", "Marines", "Army", "Scouts", "Merchants", "Other"],

  "enlistment": [
    { "target": "8+", "dms": [{ "dm": 1, "if": "Intelligence 8+" }, { "dm": 2, "if": "Education 9+" }] },
    { "target": "9+", "dms": [{ "dm": 1, "if": "Intelligence 8+" }, { "dm": 2, "if": "Strength 8+" }] },
    { "target": "5+", "dms": [{ "dm": 1, "if": "Dexterity 6+" }, { "dm": 2, "if": "Endurance 5+" }] },
    { "target": "7+", "dms": [{ "dm": 1, "if": "Intelligence 6+" }, { "dm": 2, "if": "Strength 8+" }] },
    { "target": "7+", "dms": [{ "dm": 1, "if": "Strength 7+" }, { "dm": 2, "if": "Intelligence 6+" }] },
    { "target": "3+", "dms": [] }
  ],

  "draft": [1, 2, 3, 4, 5, 6],
```

Nothing in that file is trusted. `internal/rules/parse.go` checks every printed string
against the domain's own closed alphabet, indexing each type by the name its
values give themselves — so a data file's spelling is checked against the type
and against nothing else. `internal/rules/parse.go`, `named` and the indexes built with
it:

`internal/rules/parse.go` — `named`

```go
// named indexes an alphabet by the name its values give themselves, so that
// a data file's spelling is checked against the type and nothing else.
func named[T fmt.Stringer](values []T) map[string]T {
	index := make(map[string]T, len(values))
	for _, v := range values {
		index[v.String()] = v
	}

	return index
}

var (
	characteristics  = named(traveller.Characteristics[:])
	serviceNames     = named(traveller.ServiceNames[:])
	skillTables      = named(traveller.SkillTables[:])
	weaponCategories = named(traveller.WeaponCategories[:])
	passageClasses   = named(traveller.PassageClasses[:])
	shipKinds        = named(traveller.ShipKinds[:])
	titles           = named(traveller.Titles[:])
)
```

Some checks are about a whole row rather than a cell. The draft row is the one
the draft die indexes, so it is held at load to what `Rules.Draft` needs of it.
`internal/rules/lift.go`, `checkDraft`:

`internal/rules/lift.go` — `checkDraft`

```go
// checkDraft holds the draft row to what Draft needs of it (p. 5): one die
// face per service, each face printed once. Six distinct faces over six
// services is every face, so a draft die always names a service.
func checkDraft(draft []int) error {
	seen := make(map[int]bool, len(draft))

	for i, face := range draft {
		if face < 1 || face > Faces {
			return fmt.Errorf("%w: prior service table, draft, %v: %d is not a die face",
				errMalformed, traveller.ServiceNames[i], face)
		}

		if seen[face] {
			return fmt.Errorf("%w: prior service table, draft: %d is printed twice",
				errMalformed, face)
		}

		seen[face] = true
	}

	return nil
}
```

The tables are loaded once per process and validated on the way in.
`internal/rules/rules.go`, `Load`:

`internal/rules/rules.go` — `Load`

```go
// Load returns the lifted rules, loading and validating them once.
//
// Every caller in a process gets the same Rules, so nothing on it hands out a
// way to write to it: the tables are reached through methods, and a method
// that returns a slice returns a copy (#137).
func Load() (*Rules, error) { return loaded() }

var loaded = sync.OnceValues(load)
```

A cell that will not lift is therefore a build defect that surfaces on the first
call, not a runtime condition some rare path might reach. The `rules` tests
carry the _second_ transcription of every table, typed independently from the
same visual reading pass — that second copy is the check the reprint's font
requires, and retyping it from the first copy would check nothing.

Because `Load` hands every caller in a process the same `*Rules`, nothing on it
is a field a caller can write. The tables are reached through methods, and a
method that returns a slice returns a copy. `internal/rules/rules.go`, the accessors:

`internal/rules/rules.go` — `Rules.Aging`

```go
// Aging is the Aging Table (p. 9) and the medical crisis of pp. 7-8.
func (r *Rules) Aging() Aging { return r.aging }

// Muster is what p. 9's notes fix about mustering out.
func (r *Rules) Muster() Muster { return r.muster }

// Eligibility is the Basic Skill Eligibility box (p. 6).
func (r *Rules) Eligibility() Eligibility { return r.eligibility }

// Education is the gate on the fourth Acquired Skills table (p. 11).
func (r *Rules) Education() Gate { return r.education }

// Retirement is the Annual Retirement Pay table (p. 21).
func (r *Rules) Retirement() Retirement { return r.retirement }
```

`Service` returns its column through `clone`, which copies the ranks and every
throw's modifiers; `Aging.At` returns a clone of the effects it collects; and
`Weapons` copies its list, because that list goes straight to a decider.
`internal/rules/rules.go`, `Weapons`:

`internal/rules/rules.go` — `Rules.Weapons`

```go
// Weapons is the printed list for a category (pp. 12-13), column-major.
//
// A copy: Load hands every caller in a process the same Rules, and the list
// goes straight to a decider, so returning the ruleset's own slice would let
// one decider's write change what every later character is offered (#137).
func (r *Rules) Weapons(c traveller.WeaponCategory) []traveller.WeaponName {
	list := make([]traveller.WeaponName, len(r.weapons[c]))
	copy(list, r.weapons[c])

	return list
}
```

## Layer three: the dice

`dice` is thirty lines of substance. Its stream is not seeded, so no character
can be rolled again; the record is the only way back to one, and the package doc
states the properties that still have to hold. `internal/dice/dice.go`, the
package doc:

`internal/dice/dice.go` — `package dice`

```go
// Package dice draws the die rolls Classic Traveller's character generation
// procedure calls for.
//
// It rolls and it never judges. Whether a throw meets its target is the
// business of a Target (Book 1 pp. 2-3), which lives in the traveller
// package; keeping that out of here is what lets dice depend on nothing.
//
// The stream is not seeded. No two are alike and none can be brought back:
// the record is the only way back to a character (ADR 0001), and it logs
// every die this hands out. Three properties are load-bearing all the same,
// because every fixture's script and every reader of a record rest on them:
//
//   - The die is one of six faces, each as likely as the others.
//   - A 2D throw is two of those in sequence, first die then second, and
//     both are handed back because the record logs the dice and not only
//     their total.
//   - Among is one index in n, each as likely as the others, for the
//     choices the book leaves to a player without naming a basis.
package dice
```

`Among` deserves a note, because it is not a die. Where the book hands a choice
to a player and names no basis for making it — which weapon to take off the p. 12
blade list — the tool draws rather than always taking the first name. The index
it draws is not logged; the weapon it picks is, as the answer to that choice
point. `internal/dice/dice.go`, `Stream.Among`:

`internal/dice/dice.go` — `Stream.Among`

```go
// Among returns an index into a set of n alternatives: 0 through n-1.
//
// It is not a die. The procedure has no throw here - it is drawn where the
// book hands the choice to a player and names no basis for it, so that
// --auto stops answering every such question the same way (#34). A die and a
// modulo would not do: the weapon lists of pp. 12-13 are longer than six, and
// even where they are not, 6 mod n makes the first names likelier.
//
// n must be positive; IntN says so itself.
func (s *Stream) Among(n int) int {
	return s.r.IntN(n)
}
```

## Layer four: the engine

`chargen.Generate` is the whole public entry point. It loads the tables, builds a
`run`, applies options, wraps the caller's `Decider` in the logging decorator,
and walks the procedure. `internal/chargen/generate.go`, `Generate`:

`internal/chargen/generate.go` — `Generate`

```go
// Generate walks Book 1's character generation procedure, pp. 4-25, rolling
// dice of its own unless a roller is given.
//
// The order within a term is the exposition's, not the worked example's
// (E002): survival, commission, promotion, skills, reenlistment, and then
// the aging round at the end of the term (E006). That order is what every
// fixture's script walks, so it is a fixed choice rather than an incidental
// one.
//
// Implements: A fixture generates one golden, and the same one twice
// [ADR 0001] in The record.
// Implements: The worked example reproduces [1:23–25] in The book's own
// character.
func Generate(in Inputs, decider Decider, options ...Option) (*Character, error) {
	tables, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("loading the rules: %w", err)
	}

	record := newLog()

	run := &run{
		tables: tables,
		roll:   dice.New(),
		log:    record,
		by:     traveller.ByPolicy,
		char: &Character{
			Name: in.Name, Age: traveller.NewAge(startingAge), Inputs: in, Ruleset: Ruleset,
		},
	}

	for _, option := range options {
		option(run)
	}

	run.decide = logging{to: decider, by: run.by, log: record}

	err = run.generate()
	if err != nil {
		return nil, err
	}

	run.char.sortSkills()

	run.char.Events = record.events
	run.char.Errata = record.stamped()

	return run.char, nil
}
```

`run.generate` is the procedure at its coarsest. Note that a civilian — someone
who declined the draft — returns before `serve` is ever entered, and that the
dead skip mustering out but are still assessed for a title.
`internal/chargen/generate.go`, `run.generate`:

`internal/chargen/generate.go` — `run.generate`

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
	if err != nil {
		return err
	}

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

### One term

`run.serve` loops terms until one of them ends the service. The loop has no
bound and needs none under dice — the reenlistment throw fails eventually — but
that is a **contract on the `Roller`**, not a property of the function, and the
comment says so. A test roller that answers every 2D throw with 12 would never
return. `internal/chargen/serve.go`, `run.serve`:

`internal/chargen/serve.go` — `run.serve`

```go
// serve runs terms until one of them ends the service.
//
// The loop has no bound, and needs none under dice: the reenlistment throw of
// p. 6 fails eventually and the survival throw of p. 5 ends things sooner
// than that, so a career terminates with probability 1. That is a contract on
// the Roller, not a property of this function - a roller that answers every
// 2D throw with 12 never fails a reenlistment and this never returns, which is
// why the fixture that walks past the Aging Table's last column scripts a
// finite list of dice rather than an endless supply of 12s (#54).
func (r *run) serve() error {
	for term := 1; ; term++ {
		done, err := r.term(traveller.Term(term))
		if err != nil {
			return err
		}

		if done {
			return nil
		}
	}
}
```

`run.term` is the six steps of a term, in the order the exposition gives rather
than the order the book's worked example rolls them (E002), with the aging round
last (E006). `internal/chargen/serve.go`, `run.term`:

`internal/chargen/serve.go` — `run.term`

```go
func (r *run) term(term traveller.Term) (bool, error) {
	r.log.step(fmt.Sprintf("term %d", term), "pp. 5-7")

	r.char.Terms = int(term)

	if !r.survive(term) {
		// The fatal term counts, and its four years with it (E004).
		r.char.Age = r.char.Age.PlusYears(traveller.Years)
		r.char.Departure = traveller.KilledBySurvivalThrow{}
		r.dead = true

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

	intent, forced, err := r.reenlist(term)
	if err != nil {
		return false, err
	}

	r.char.Age = r.char.Age.PlusYears(traveller.Years)

	err = r.agingRound(term)
	if err != nil {
		return false, err
	}

	if r.dead {
		return true, nil
	}

	if forced || intent == traveller.Continue {
		return false, nil
	}

	return true, nil
}
```

### The Decider, and the one gate every answer passes

The engine never decides anything on its own behalf. Every question goes to a
`Decider` — twelve methods, one per choice point, each typed to its own alphabet.
That typing is what closes the set: adding a thirteenth question breaks every
implementation at compile time.

A transcript of `grep -nE '^	[A-Z][A-Za-z0-9]*\(' internal/chargen/decider.go | sed 's/(.*//' | cat -n`, run while writing:

```text
     1	30:	Service
     2	33:	SubmitToDraft
     3	37:	AttemptCommission
     4	41:	AttemptPromotion
     5	49:	SkillTable
     6	58:	Weapon
     7	66:	ReenlistIntent
     8	70:	MusterTable
     9	74:	MusterTable1DM
    10	78:	MusterTable2DM
    11	85:	MusterWeapon
    12	93:	AssumeTitle
```

Twelve, and the enum `traveller.ChoicePoint` has twelve constants whose
`String()` is each method's name. That is unavoidably a second list kept
parallel to the interface, so a reflection check in `chargen`'s tests holds the
two together. When the engine asks each, and what the auto policy answers,
is in the doc comment of `Policy`'s method of the same name.

Every decider — the auto policy, the interactive player, the scripted test
deciders — reaches the engine through one decorator, `internal/chargen/logging.go`. It
must implement all twelve to compile, so a choice point cannot reach the engine
without reaching the log. Each of its methods names the offer before it asks,
so what `record` checks an answer against is what the decider was shown, not
whatever the decider left in its slice afterwards. `record` itself does two
jobs.
`internal/chargen/logging.go`, `logging.record`:

`internal/chargen/logging.go` — `logging.record`

```go
// record writes the answer into the generation record, and refuses one that
// was not offered.
//
// It is the single gate: the decorator is the only way a decider reaches the
// engine (generate.go wraps every one of them), and it already holds both
// the offered set and the answer, because it records both. Each method names
// the offer before it asks, so what is checked is what the decider was
// shown: a decider can write to the slice it is handed, and an offer named
// afterwards would let it put its own answer there and pass. Checking here
// rather than at each application site leaves one definition of answering
// outside the offer, which two cannot disagree with.
//
// It compares rendered names rather than values, and every method above has
// the typed offered set and the typed answer in hand before rendering. That
// is deliberate: MusterWeapon answers with a sum, and comparing a sum against
// a list of them needs a fold, so one path here would have to be strings
// whatever the other eleven did.
//
// What it costs is an assumption. The check is sound exactly as long as
// String() is injective on every alphabet the engine offers - two values that
// spelled the same would let a decider answer with one and have the other
// recorded, and nothing would notice. It holds, and
// traveller.TestNoTwoValuesOfAnAlphabetShareASpelling holds it (#48).
//
// Implements: An answer outside the offered set is refused, not applied
// [Choice points] in The record.
func (l logging) record(point traveller.ChoicePoint, from []string, chosen string) error {
	if !slices.Contains(from, chosen) {
		return fmt.Errorf("%w: %v offered %v, and was answered %q",
			errNotOffered, point, from, chosen)
	}

	l.log.choice(point, l.by, from, chosen)

	return nil
}
```

That injectivity assumption is itself held by a test — and that test's claim to
cover _every_ alphabet is held by a second test that reads the package's own
source for `var X = [...]` declarations and fails on one nobody checks:

A transcript of `grep -n 'var alphabet' -A 1 internal/traveller/spelling_test.go; grep -c 'spelledOnce(t,' internal/traveller/spelling_test.go`, run while writing:

```text
87:var alphabet = regexp.MustCompile(`(?m)^var ([A-Z]\w*) = \[\.\.\.\]`)
88-
12
```

One method is handed more than the offered set. `Weapon` also gets a `Vary`, the
only variation a decider may take — and what it is handed is not the run's
`Roller` under a narrower name. A `Roller` passed as a `Vary` would keep its
dynamic type, and one type assertion would give the decider the dice back, so
`WeaponPick` in `internal/chargen/enlist.go` hands it a wrapper with nothing on it but
`Among`. `internal/chargen/roller.go`, `varyOnly`:

`internal/chargen/roller.go` — `varyOnly`

```go
// varyOnly is the Vary a decider is handed: a Roller with everything but
// Among taken off it, so the narrowing is the value's and not only the
// parameter's. Passing the Roller itself as a Vary would leave its dynamic
// type behind it, and one type assertion would hand a decider the dice back
// (#136), which is what Vary exists to prevent.
type varyOnly struct{ roller Roller }

func (v varyOnly) Among(n int) int { return v.roller.Among(n) }
```

### The auto policy

`internal/chargen/policy.go` is the `--auto` decider. It is total, deterministic, and a
pure function of what it is handed: where it needs to know an option is legal,
the engine supplies that by what it puts in the offered set, never by the policy
reaching into the character. It also assumes that set is non-empty, and checks
nothing: the engine never offers an empty one.

Three orthogonal strategies, each a closed type rather than a string — which is
why `chargen` has no `Validate` method. `internal/chargen/policy.go`, the skills and
mustering out constants and the three strategy sets:

`internal/chargen/policy.go` — `Skills`

```go
// The skills strategies, in their listed order; the first is the default.
const (
	SkillsAdvanced Skills = iota
	SkillsService
	SkillsPersonal
)

// The mustering out strategies, in their listed order; the first is the
// default.
const (
	MusterCash Muster = iota
	MusterGoods
	MusterSpartan
)

// The three alphabets, for parsing a flag and for listing the choices. They
// are unexported: nothing outside needs to iterate a strategy set, and the
// two things that do need one - the parser and the help - are below.
//
//nolint:gochecknoglobals // three immutable tables, and Go has no const slice.
var (
	careerStrategies = []Career{CareerServe, CareerRetire, CareerOneTerm}
	skillsStrategies = []Skills{SkillsAdvanced, SkillsService, SkillsPersonal}
	musterStrategies = []Muster{MusterCash, MusterGoods, MusterSpartan}
```

A flag word becomes a domain value once, at the boundary, and past there a
`Career` is a `Career`. The boundary is one call that parses all three words
together and hands back the policy they name. `internal/chargen/policy.go`, `ParsePolicy`,
its doc and signature:

`internal/chargen/policy.go` — `ParsePolicy`

```go
// ParsePolicy turns the three strategy flags' words into the policy they name.
//
// This is where a strategy stops being a string: at the boundary, once,
// rather than being carried inward and re-checked wherever it is used. The
// command line is the one thing that parses strategies, and it parses all
// three together every time (#135).
func ParsePolicy(career, skills, muster string) (Policy, error) {
```

Most methods are one line — `AttemptCommission` is always yes, because nothing
in the rules costs a character a failed attempt. Three are more interesting.

`Service` computes the odds of meeting each offer's printed target on two dice
after its printed modifier, and takes the best, ties going to book order. It is
not a table and not a rule; it is arithmetic over what the offer carries.
`internal/chargen/policy.go`, `odds`:

`internal/chargen/policy.go` — `odds`

```go
// odds is the chance of meeting an offer's target on two dice after its
// modifier. It is computed from the printed target and the printed DM; it is
// not a table of its own and it is not a rule.
func odds(offer traveller.EnlistmentOffer) float64 {
	ways := 0

	const faces = 6

	for first := 1; first <= faces; first++ {
		for second := 1; second <= faces; second++ {
			if offer.Target.Satisfied(first + second + offer.DM) {
				ways++
			}
		}
	}

	return float64(ways) / float64(faces*faces)
}
```

`SkillTable` does not rank alone. Ranking alone would designate the Advanced
Education table every time it was offered and Personal Development never, so a
character generated under the default would never raise a characteristic —
though p. 11's first table is exactly how that is done. The engine hands the
decider a `taken` slice parallel to `from`, and the policy prefers the tables he
has trained on least before applying its ranking. `internal/chargen/policy.go`,
`thinnest`:

`internal/chargen/policy.go` — `thinnest`

```go
// thinnest is the offered tables the character has had fewest results off.
//
// A caller with no counts - a test constructing a policy directly - gets the
// whole offered set back, so the ranking decides alone.
func thinnest(from []traveller.SkillTable, taken []int) []traveller.SkillTable {
	if len(taken) != len(from) {
		return from
	}

	fewest := slices.Min(taken)

	thin := make([]traveller.SkillTable, 0, len(from))

	for i, table := range from {
		if taken[i] == fewest {
			thin = append(thin, table)
		}
	}

	return thin
}
```

`Weapon` is the only method that draws. The page prints a list in an order and
printed order is not a preference — taking the first name is as invented as
drawing, and drawing admits it. So the policy's whole answer is
`from[vary.Among(len(from))]`.

## Layer five: rendering

`render` projects a `*chargen.Character` into a flat `record` struct and then
into JSON, a character sheet, or a transcript. The domain types are the
interface; JSON is a projection, and where the two disagree the codec absorbs it.
`internal/render/json.go`, the package doc:

`internal/render/json.go` — `package render`

```go
// Package render projects a generated character into the shapes a reader
// wants: JSON, a character sheet in the book's own style, and a transcript
// of the generation record.
//
// The domain types are the interface and JSON is a projection. Where the two
// disagree - a sum that must marshal flat, an array-backed profile that must
// marshal as six named keys - the codec here absorbs the difference and the
// domain type keeps its shape.
package render
```

The wire shape is one struct. Its JSON tags are the record's public field names,
and `docs/character.schema.json` describes it in draft 2020-12. The names follow
the book: a service rank's name is `rankName` ("Captain"), and the hereditary
title of p. 5 is `nobility`, whose own `title` field is the title it confers.
`internal/render/json.go`, `record`:

`internal/render/json.go` — `record`

```go
type record struct {
	Record          int               `json:"record"`
	Name            string            `json:"name"`
	UPP             string            `json:"upp"`
	Characteristics map[string]int    `json:"characteristics"`
	Age             ageRecord         `json:"age"`
	Terms           int               `json:"terms"`
	Enlistment      enlistmentRecord  `json:"enlistment"`
	Service         string            `json:"service,omitempty"`
	Rank            int               `json:"rank,omitempty"`
	RankName        string            `json:"rankName,omitempty"`
	Skills          []skillRecord     `json:"skills"`
	Benefits        benefitsRecord    `json:"benefits"`
	Pension         int64             `json:"annualRetirementPay,omitempty"`
	Departure       *departureRecord  `json:"departure,omitempty"`
	Nobility        *nobilityRecord   `json:"nobility,omitempty"`
	Inputs          inputsRecord      `json:"inputs"`
	Errata          []string          `json:"errata"`
	Ruleset         string            `json:"ruleset"`
	Policy          int               `json:"policy,omitempty"`
	Events          []json.RawMessage `json:"events"`
	Build           string            `json:"build,omitempty"`
}
```

The schema leaves the record's root open to a section a later build may add, and
closes everything inside it — the event kinds included, so an event of a kind
the procedure does not log does not validate. `internal/render/schema_test.go` holds both
halves.

The write side is held by folds — one codec per sum, so adding a case stops this
package compiling until the projection learns to write it:

A transcript of `grep -n 'Codec struct\|^func fold' internal/render/fold.go`, run while writing:

```text
13:type enlistmentCodec struct{ out enlistmentRecord }
33:func foldEnlistment(from traveller.Enlistment) enlistmentRecord {
46:type departureCodec struct{ out departureRecord }
80:func foldDeparture(from traveller.Departure) departureRecord {
160:type eventCodec struct {
```

The **read** side is not. `internal/render/decode.go` unmarshals into the same `record`
struct and deliberately does not rebuild domain types, so `SheetFrom` and
`TranscriptFrom` work on strings — comparing, for example, a ship's kind against
`traveller.ScoutShip.String()`. The compiler cannot see those comparisons; what
holds them is a round-trip test. `internal/render/decode.go`, `errNotARecord` and the head
of `decode`:

`internal/render/decode.go` — `errNotARecord`

```go
var errNotARecord = errors.New("not a character record")

// decode reads a record back from what JSON wrote.
//
// Nothing is rebuilt into a domain type. The record is a projection of the
// domain values, and a sheet is a projection of the record: rebuilding an
// Enlistment or a Departure back into an interface would give the renderer
// nothing it does not already have, and would need a decoder for every sum
// that could go wrong in a way the wire shape cannot.
func decode(text []byte) (record, error) {
```

### The line at the bottom of every rendering

A sheet is what a referee reads and keeps, and the record is the only way back
to the character on it (ADR 0001). So every sheet and every transcript ends with
the same one line, `internal/render/sheet.go`, `keepTheRecord`:

```text
Keep the JSON record: it is the only way back to this character.
```

Nothing prints a command to re-run. The sheet still needs to know who named the
service, which is the one choice event it reads back, through `choiceAt` in the
same file.

## Layer six: the command line

`cmd/ctchargen/main.go` is a switch over the first argument and nothing else.
`main` runs it, prints any error once, and leaves by `exitStatus`, the one place
that decides the exit status: 0 for a run that did what it was asked and 1 for
one that did not, whichever kind of failure it was. `cmd/ctchargen/main.go`,
`main` and `exitStatus`:

`cmd/ctchargen/main.go` — `main`

```go
func main() {
	err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ctchargen:", err)
	}

	os.Exit(exitStatus(err))
}

...

// exitStatus is 0 for a run that did what it was asked and 1 for one that did
// not. Those are the two the README promises (#121), and this is the one
// place that decides them.
func exitStatus(err error) int {
	if err == nil {
		return 0
	}

	return 1
}
```

`run` is where the two output channels are taken apart: `out` is the data
channel — the record, the sheet, the transcript — and `asking` is where the tool
talks to the person driving it. Note too that help asked for leaves by `out` and
returns nil, so `--help` exits 0; a bare `ctchargen` with no arguments stays an
error, because nothing was asked for. Its one option, `withDice`, is how a test
hands in a scripted roller; `main` passes none, so every character the command
makes comes from the engine's own unseeded stream. `cmd/ctchargen/main.go`,
`run`:

`cmd/ctchargen/main.go` — `run`

```go
// run takes the two output channels apart. `out` is the data channel - the
// record, the sheet, the transcript - and `asking` is where the tool talks to
// the person driving it. Interactive mode writes its questions to the second,
// so that `ctchargen new | jq` pipes a record and not a conversation.
//
// Implements: What the command writes matches the schema, build stamp and all
// [JSON conventions] in The record.
// Implements: The questions go to stderr, not into the record [CLI sketch] in
// The record.
func run(args []string, in io.Reader, out, asking io.Writer, options ...option) error {
	var chosen settings

	for _, apply := range options {
		apply(&chosen)
	}

	if len(args) == 0 {
		return fmt.Errorf("%w: ctchargen %s; run `ctchargen --help`", errUsage, commandList)
	}

	switch args[0] {
	case "new":
		return newCharacter(args[1:], in, out, asking, chosen.dice)
	case "batch":
		return batch(args[1:], out, asking, chosen.dice)
	case "render":
		return renderRecord(args[1:], out)
	case "version":
		return version(args[1:], out)
	// Help asked for is not a misuse: it leaves by out and exits 0. A bare
	// `ctchargen` stays an error, because nothing was asked for at all.
	// There is no `help <command>`; --help on the command is the one way in,
	// so there is one place a command's flags are described - and a word
	// after it is refused rather than dropped, as `new` refuses one.
	case "help", "-h", "-help", "--help":
		if len(args) > 1 {
			return fmt.Errorf(
				"%w: help takes no arguments, and was given %q; run `ctchargen <command> --help` for one command's flags",
				errUsage, args[1],
			)
		}

		return writeTopLevelHelp(out)
	default:
		return fmt.Errorf("%w: unknown command %q; want %s; run `ctchargen --help`",
			errUsage, args[0], commandList)
	}
}
```

Here is the tool actually running.

A transcript of `ctchargen --help`, run while writing:

```text
usage: ctchargen <command> [flags]

  new      generate one character, asking at every choice unless --auto
  batch    generate many characters, under --auto
  render   write a record saved earlier as a sheet or as a transcript
  version  write the build

Run `ctchargen <command> --help` for that command's flags.
```

The two generating subcommands share the flags that say what a character is
generated from — name, service and the three strategies. Their names,
types and defaults are declared once, in `declareCharacterFlags`, and each
command supplies only its own descriptions; from there they travel as one value,
so five adjacent strings cannot be transposed on the way in.
`cmd/ctchargen/new.go`, `characterFlags`:

`cmd/ctchargen/new.go` — `characterFlags`

```go
// characterFlags are the flags both generating subcommands take: what a
// character is generated from, as opposed to what is done with it afterwards.
//
// Each command gives them its own descriptions, which legitimately differ;
// the names, types and defaults are declared once, in declareCharacterFlags,
// so the two cannot come to disagree about them. They are carried as one
// value from there on, because five adjacent strings in a signature are five
// strings that can be transposed, and a transposed strategy is one the record
// then reports as the one the operator asked for (#132).
type characterFlags struct {
	set                                   *flag.FlagSet
	name, service, career, skills, muster *string
}
```

`characterFlags.inputs` is where those words become `Inputs` and a `Policy`, from
the one `ParsePolicy` call, so what a record says it was given and what decided
it cannot name different strategies.

The help lists each command's flags from the flag set itself rather than from a
hand-typed list — so the list cannot go stale, and the defaults are printed
because a strategy flag left alone still chooses:

A transcript of `ctchargen new --help`, run while writing:

```text
usage: ctchargen new [flags]

  --auto             decide by the policy at every choice, rather than asking
  --career strategy  the --auto career strategy: serve, retire, oneterm (default serve)
  --force            replace the output file if it already exists
  --history          write the generation record rather than JSON
  --muster strategy  the --auto mustering out strategy: cash, goods, spartan (default cash)
  --name string      the character's name
  -o string          write to this file rather than to standard output
  --service string   attempt enlistment in this service: navy, marines, army, scouts, merchants, other
  --sheet            write the character sheet rather than JSON
  --skills strategy  the --auto skills strategy: advanced, service, personal (default advanced)
```

`inputs` also refuses a name that is more than one line. The sheet and the
transcript print the name as markdown, where a line break would start sections
the character does not have. A built binary shows the exit status too, which
`go run` would replace with its own:

A transcript of `ctchargen new --auto --name "$(printf 'Alexander\nJamison')"; echo "exit $?"`, run while writing:

```text
ctchargen: usage: --name "Alexander\nJamison" is more than one line
exit 1
```

### A character, end to end

A record is the only way back to a character (ADR 0001), so this section reads
one that is checked in: the `army-serve` golden, rendered as the sheet a
referee would keep. A fresh `ctchargen new --auto --sheet` prints a different
character every time.

A transcript of `ctchargen render internal/chargen/testdata/army-serve.json`, run while writing:

```text
# (unnamed)

UPP 965BCD, age 46, Army General, 7 terms, marquis/marchioness, retired

## Skills

- ATV-2
- Administration-1
- Electronic-2
- Gambling-1
- Halberd-1
- Mechanical-1
- Rifle-1
- Submachine Gun-2
- Tactics-3

## Possessions

- CR 30000
- Middle Passage
- High Passage
- High Passage
- High Passage
- CR 8000 a year in retirement pay

## Service record

- Army, enlisted
- retired after term 7

---

Keep the JSON record: it is the only way back to this character.
```

The fixture forced the Army, so the headline names it plainly. Had the policy
chosen the service, the headline would say `(service chosen by the policy)`,
and had a failed throw sent him to the draft, it would say where he was sent
and who refused him.

The same record as a transcript is the audit trail. Its opening:

A transcript of `ctchargen render --history internal/chargen/testdata/army-serve.json | sed -n '1,22p'`, run while writing:

```text
# Generation record: (unnamed)

## 1. characteristics (p. 4)

  2. Strength: rolled 5+5
  3. Strength 10 (from 2)
  4. Dexterity: rolled 3+3
  5. Dexterity 6 (from 4)
  6. Endurance: rolled 2+6
  7. Endurance 8 (from 6)
  8. Intelligence: rolled 6+4
  9. Intelligence 10 (from 8)
 10. Education: rolled 5+5
 11. Education 10 (from 10)
 12. Social Standing: rolled 6+6
 13. Social Standing 12 (from 12)
 14. UPP A68AAC at age 18

## 15. enlistment (pp. 5, 10)

 16. enlistment, Army: rolled 5+1 +3 = 9 against 5+, made
 17. enlisted in the Army (from 16)
```

Three things to read out of that.

The `(from N)` suffixes are the causal edges: an `OutcomeEvent` carries the
sequence number of the throw or choice that produced it, so "enlisted in the
Army" points at the throw one line above it. That is what makes the log
auditable rather than merely verbose.

The step headings are events too, so they carry their number: `## 15.` is
the enlistment step, and the numbers run without a gap from 1 to the last.

And the UPP is `A68AAC` at 18 but `965BCD` on the sheet above: seven terms of
service, four of them followed by aging rounds, moved four of the six
characteristics. The transcript is where you find out which throw moved each.

Here is the first term:

A transcript of `ctchargen render --history internal/chargen/testdata/army-serve.json | sed -n '/## [0-9]*\. term 1 /,/## [0-9]*\. term 2 /p'`, run while writing:

```text
## 20. term 1 (pp. 5-7)

 21. survival: rolled 2+5 +2 = 9 against 5+, made
 22. survived term 1 (from 21)
 23. AttemptCommission: policy chose yes from yes, no
 24. commission: rolled 2+3 +1 = 6 against 5+, made
 25. commissioned: Lieutenant (from 24)
 26. Submachine Gun-1 (from 24)
 27. AttemptPromotion: policy chose yes from yes, no
 28. promotion: rolled 2+1 +1 = 4 against 6+, missed
 29. not promoted this term (from 28)

## 30. skills and training (pp. 6, 11)

 31. SkillTable: policy chose Advanced Education Table (education 8+) from Personal Development Table, Service Skills Table, Advanced Education Table, Advanced Education Table (education 8+)
 32. Advanced Education Table (education 8+): rolled 3
 33. Tactics-1 (from 32) [E002]
 34. SkillTable: policy chose Advanced Education Table from Personal Development Table, Service Skills Table, Advanced Education Table, Advanced Education Table (education 8+)
 35. Advanced Education Table: rolled 5
 36. Weapon: policy chose Halberd from Dagger, Blade, Foil, Cutlass, Sword, Broadsword, Spear, Halberd, Pike, Cudgel, Bayonet
 37. Blade Combat: Halberd-1 (from 35) [E002]
 38. SkillTable: policy chose Service Skills Table from Personal Development Table, Service Skills Table, Advanced Education Table, Advanced Education Table (education 8+)
 39. Service Skills Table: rolled 6
 40. Weapon: policy chose Submachine Gun from Body Pistol, Automatic Pistol, Revolver, Carbine, Rifle, Laser Carbine, Laser Rifle, Automatic Rifle, Submachine Gun, Shotgun
 41. Gun Combat: Submachine Gun-2 (from 39) [E002]
 42. reenlistment: rolled 6+3 = 9 against 7+, made
 43. ReenlistIntent: policy chose continue from continue, discharge
 44. reenlisted for term 2 (from 42)

## 45. term 2 (pp. 5-7)
```

And there is `thinnest` working. Three eligibilities were spent — two for the
initial term and one for the commission; the promotion throw missed — and the
policy designated a _different_ table each time: the Education 8+ table first
(its ranking), then Advanced Education, then Service Skills. Under pure ranking
all three would have been the 8+ table.

`[E002]` is an erratum stamp. Every skill taken this term names the reading that
put the training step where it is — the exposition's per-term order rather than
the worked example's. Fifteen such readings are recorded in `internal/rules/data/ERRATA.md`, and
a record names every one that governed it.

The two `Weapon` lines are the one choice the policy draws for: the book names
no basis for picking a blade or a gun, so the policy takes `Among` the offer,
and the record logs the draw as the choice it produced. The term ends on a
reenlistment throw that was made. Six terms later this character retires, which
is `traveller.Retired` — one of the five cases of the `Departure` sum. The other
four are `Discharged`, `ForcedOut`, and two deaths.

### Death is an outcome, not an error

Under the 1977 text a failed survival throw kills the character. There is no
"injured instead" option — that is the 1981 revision, and it is the single
likeliest thing for a programmer to get wrong from memory. The tool prints the
corpse:

A transcript of `ctchargen render internal/chargen/testdata/drafted-and-died.json | head -5`, run while writing:

```text
# (unnamed)

UPP 8777A8, age 22, Other (drafted after the Navy refused him), 1 term, killed by the survival throw

## Service record
```

That one line carries three of the ideas above at once. The fixture forced the
Navy, which forced the enlistment _attempt_ only — the throw was still made, it
failed, and the draft put him in the Other service, "possibly the very service
which had just previously rejected his enlistment" (p. 5). The headline says
`drafted after the Navy refused him` rather than silently showing a service
nobody asked for. And he died in term 1, which still counts: the fatal term and
its four years are included, so he is 22 and not 18 (E004).

### Batches

`batch` generates many characters, each with dice of its own. It always says
what it did, on the `asking` channel, because a run that wrote seventy-four
corpses and printed nothing would be a trap. The dice are random, so these are
samples: another run prints other counts.

A sample of `ctchargen batch --auto --count 8 2>&1 >/dev/null`, run while writing:

```text
8 written, 1 died
```

`--survivors` does not reroll anyone. It passes over a dead character and
generates another in his place, so every character written is still exactly
the character his own dice made — what changes is which characters appear:

A sample of `ctchargen batch --auto --count 8 --survivors 2>&1 >/dev/null`, run while writing:

```text
8 written, 1 passed over for dying
```

The records themselves go to standard output as JSONL, one per line, streamed as
they are generated so a pipe sees the first one immediately:

A sample of `ctchargen batch --auto --count 3 2>/dev/null | cut -c1-96`, run while writing:

```text
{"record":1,"name":"","upp":"639999","characteristics":{"Dexterity":3,"Education":9,"Endurance":
{"record":1,"name":"","upp":"B66985","characteristics":{"Dexterity":6,"Education":8,"Endurance":
{"record":1,"name":"","upp":"976594","characteristics":{"Dexterity":7,"Education":9,"Endurance":
```

A batch that fails partway has already written records, so `write` reports the
tally before the error — worded so that it does not read like a finished batch.
It skips the line where no record was written, and for a failed `--survivors`
search, whose error already carries the counts. `cmd/ctchargen/batch.go`,
`tally.reportStopped`:

`cmd/ctchargen/batch.go` — `tally.reportStopped`

```go
// reportStopped writes the closing line of a run that failed partway. It says
// the run stopped, because the error printed after it does, and a line that
// read like a finished batch would contradict the line beneath it.
func (t tally) reportStopped(asking io.Writer) error {
	return t.reportAs(asking, fmt.Sprintf("%d written before the run stopped", t.written))
}
```

### Reading a record back

`render` takes a record written earlier and writes it as a sheet or a transcript.
It **reads** the record rather than regenerating anything: the dice are not
seeded, so there is nothing to regenerate from, and the record is the only way
back to the character (ADR 0001).

The repository's goldens are records, so they are what `render` can be pointed
at directly:

A transcript of `ctchargen render internal/chargen/testdata/navy-captain.json`, run while writing:

```text
# (unnamed)

UPP 766BDF, age 46, Navy Captain, 7 terms, duke/duchess, retired

## Skills

- Engineer-1
- Forward Observer-1
- Jack of all Trades-1
- Navigation-2
- Pilot-1
- Ship's Boat-2
- Vacc Suit-2

## Possessions

- CR 35000
- High Passage
- Travellers' Aid Society membership
- CR 8000 a year in retirement pay

## Service record

- Navy, enlisted
- retired after term 7

---

Keep the JSON record: it is the only way back to this character.
```

The same line ends every sheet and every transcript, whoever generated the
record and however: it names nothing about the run, because there is nothing to
re-run.

Each golden is a triple: the record, the sheet it renders to, and the transcript.
All three are regenerated, never hand-edited.

A transcript of `ls internal/chargen/testdata/ | grep '^navy-captain'`, run while writing:

```text
navy-captain.json
navy-captain.sheet.md
navy-captain.transcript.md
```

## The gates

Each fact lives in one place, so no test exists to hold a second copy to the
first. The readings document's entries and the `Erratum` constants are held
to each other by the `rules` lift; the `--auto` policy is its own doc
comments; `COVERAGE.md` is generated from markers beside the code.

`chargen`'s tests hold the readings and the choice points to the goldens,
beside the roster that generates them: the stamping conditions of
`internal/rules/data/ERRATA.md` both ways, every choice point to a golden that reaches
it, and the schema's three strategy enums to the strategies the code takes.
`internal/coverage`, a package with no non-test code, generates the coverage
map:

A transcript of `grep -h '^func Test' internal/coverage/*.go | sed 's/func //; s/(t \*testing.T) {//'`, run while writing:

```text
TestTheCoverageMapIsGenerated
TestAMarkerRemovedFailsNamingTheRule
TestACiteNamesAnAuthority
TestOnlyTheTablesAreGenerated
TestMarkersThatCannotMakeADocument
TestAMarkerOutOfPlaceOrOutOfShape
TestAMarkerWraps
```

There is no Go API to hold: the packages are under `internal/`. The command
line is what a script depends on, and it has a baseline in
`cmd/ctchargen/cli_test.go`, held in one direction only — a flag a script passes
must keep its name, type and default, and a new flag breaks nothing — beside a
test that the exit status is only ever 0 or 1:

A transcript of `grep -h '^func Test' cmd/ctchargen/cli_test.go | sed 's/func //; s/(t \*testing.T) {//'`, run while writing:

```text
TestEveryBaselineFlagIsStillDeclared
TestTheExitStatusIsZeroOrOne
```

The sharpest of these checks is chargen's
`TestEveryReadingIsStampedWhereItsConditionHolds`. `ERRATA.md` states each
reading's stamping condition as a predicate over the finished record; the check
re-states that predicate in Go and checks every golden both ways — a record whose
condition holds must stamp the erratum, and one whose condition does not must
not. The predicate is written from the document's prose and never from the
stamping code, because a condition transcribed from the thing it checks is one
reading written twice.

The whole gate is `task`, and there is no CI: a change is checked when someone
runs `task` and not otherwise. It runs `go mod tidy -diff`, `go vet`,
golangci-lint (which also reports formatting), NilAway, govulncheck, `go test -race`,
and `bun fmt --check`. Here is the test suite alone, with the
timings stripped so the output is stable:

A transcript of `go test ./... -count=1 2>&1 | sed -E 's/[[:space:]]+[0-9.]+s$//'`, run while writing:

```text
ok  	github.com/philoserf/ctchargen/cmd/ctchargen
ok  	github.com/philoserf/ctchargen/internal/chargen
ok  	github.com/philoserf/ctchargen/internal/coverage
ok  	github.com/philoserf/ctchargen/internal/dice
ok  	github.com/philoserf/ctchargen/internal/mutate
ok  	github.com/philoserf/ctchargen/internal/render
ok  	github.com/philoserf/ctchargen/internal/rules
ok  	github.com/philoserf/ctchargen/internal/traveller
ok  	github.com/philoserf/ctchargen/internal/walkthrough
```

## Where the linear order breaks down

One place, recorded because a reader following the call chain will hit it.

**The decider you pass is not the decider the engine calls.** Reading `chargen`
top to bottom, you meet `r.decide.AttemptCommission()` in `rank.go` and naturally
look for the implementation you handed to `Generate`. It is one hop further out:
`generate.go` wraps every decider in `logging`, which records the answer and
refuses one that was not offered before delegating. That indirection is the whole
point — it is what makes the offered-set check exist in one place rather than
twelve — but the call chain does not show it at the call site.

A reader tracing `assessTitle` should hold the book's two words apart. A
**rank** is a position on a service's Table of Ranks: `traveller.Rank`,
`chargen.Character.Rank`, and its printed name in `Character.RankName` and
`rules.Service.RankName`. A **title** is nobility (p. 5, Book 3 p. 22):
`traveller.Title`, carried in `chargen.Nobility.Title` and in the record's
`nobility` object.

Everything else reads in order: `main` dispatches, `newCharacter` parses flags
into `Inputs` and a `Policy`, `Generate` walks the procedure asking a `Decider`,
`render` projects the result. The engine is genuinely linear.
