# THEORY.md

The understanding you need to change this system without damaging it. Not a tour
of the files — `ls -R` does that — but an account of what the code believes,
where those beliefs are enforced, and where they rest on nothing but care.

## What the system is for

This is a transcription problem wearing a code generator's clothes.

Classic Traveller's character generation, Book 1 pp. 4–25 of the 1977 text, is a
procedure a person walks with paper and two dice: roll six characteristics,
attempt enlistment in one of six services, then survive terms of four years each
— surviving, maybe being commissioned, maybe promoted, training on one of four
skill tables, throwing to reenlist, and aging — until the service ends by choice,
by refusal, or by death. Then muster out, take benefits off two printed tables,
and assess nobility.

Any competent programmer could implement that from memory in an afternoon. The
result would be wrong, and this repository's entire shape is organised around why.

The wrongness has two sources, and the code fights both.

**The rules in your head are the wrong rules.** Training-data Traveller is the
1981 revision. Under that revision a failed survival throw injures you; under the
1977 text it kills you. `CLAUDE.md`'s Authority section opens on this and the code
carries it through: `traveller.KilledBySurvivalThrow` is a case of the `Departure`
sum, not an error path, and `chargen.Character` is documented as "complete whether
it ended in mustering out or in death." Death is an outcome the tool prints, and
`batch` counts corpses rather than rerolling them.

**The pages in front of you lie under a text extractor.** The FFE reprints embed a
font whose glyph substitutions look like data. On p. 9 the em-dash cells of
Mustering Out Table 1 extract as the digit `4`; "Travellers' Aid" extracts as
`Travellers9`; a minus sign becomes a digit, so an `N−` target reads as `N3`. Each
of those is a plausible number in a table full of numbers. The defence is
threefold and every layer is visible in the tree: pages are read visually rather
than extracted, every table is transcribed **twice** — once into `internal/rules/data/*.json`
and once independently into `internal/rules/*_test.go` from the same visual pass — and the
lift refuses anything that does not resolve. `traveller.ParseTarget` is
deliberately strict about the sign for exactly this reason, and `internal/rules/parse.go`
accepts both the ASCII hyphen and U+2212 because "someone transcribing p. 11 will
type what he sees."

So: the domain the code models is not "a character" but **a printed procedure and
the readings a person had to make to follow it**. That is why the record is an
audit log first and a character sheet second.

### The vocabulary

- A **service** is one of six columns of the Prior Service Table (p. 10).
- A **term** is four years. It has no upper bound as a type, because a 12 on the
  reenlistment throw forces another one and E003 reads that as recurring.
- A **throw** has something to meet; a **roll** does not. That distinction is a
  type distinction (`ThrowEvent` vs `RollEvent`) and it was won the hard way — the
  two used to be one type and every targetless roll was written to the record as
  `"succeeded": true`, a claim about nothing (#50).
- A **reading** is a decision recorded where the page is silent, ambiguous, or
  contradicts itself, and an **erratum** is its id, `E001`–`E015`, which a
  record carries to say the reading governed it. `GLOSSARY.md` keeps the two
  words apart on purpose: the decision is prose in `ERRATA.md`, and the id is
  what gets stamped. Every reading is named on every record it governed.
- A **fixture** is a script of dice and draws that walks the procedure down one
  chosen path. A **golden** is the record a fixture generates, checked in and
  never hand-edited. Neither is a seed: there are none (ADR 0001).
- A **choice point** is a place the procedure asks a question with more than one
  legal answer. There are exactly twelve.
- A **rank** is a position in a service's column of the Table of Ranks (p. 10),
  and what that column prints for it is the rank's _name_ —
  `rules.Service.RankName`, `chargen.Character.RankName`, `rankName` in the
  record. A **title** is nobility and nothing else (p. 5, Book 3 p. 22):
  `traveller.Title`, held by `chargen.Nobility` and written as
  `nobility{eligible,title,assumed}`. The code's words are the book's words,
  because the book is the authority on the domain's vocabulary as well as its
  rules; `RankName`'s own comment carries the cites for why it is not "title".

## The organizing ideas

### 1. Exhaustiveness has two strengths, and the code knows which it is using

`internal/traveller/doc.go` states this outright, and it is the single most load-bearing
idea in the tree:

- **Sums** — `Enlistment`, `TableResult`, `BenefitRow`, `Departure`, `Event`,
  `WeaponBenefit` — are sealed interfaces with a `Fold(cases)` method and an
  unexported seal. Adding a case adds a method to the cases interface, and _every
  implementation stops compiling_. That is the compiler, and nothing turns it off.
- **Plain enums** — `ServiceName`, `Characteristic`, `SkillTable` — are checked by
  the `exhaustive` linter. That is not the compiler. Dropping the linter drops the
  guarantee.

Go's type switch over an interface promises nothing, and the code says so. The
practical consequence: **when you add a case to a sum, follow the compiler
errors** — they are the complete list of places that must learn about it, and they
span packages (`internal/chargen/enlist.go`'s `applyResult`, `internal/chargen/muster.go`'s
`applyBenefit` and `takeWeapon`, `internal/render/fold.go`'s codecs). When you add a value
to an enum, the compiler will not help you; grep and trust the linter.

The `Decider` interface is the same idea applied to questions. Twelve methods, one
per choice point, each typed to its own alphabet. `//nolint:interfacebloat` sits on
it with the reason: "one method per choice point closes the set." Spelling it as
one method over a label would lose exactly that.

### 2. A fold that cannot fail still returns an error, and the code says why each time

You will see this pattern constantly:

```go
// A fold over a codec cannot fail: every case only assigns.
_ = c.Enlistment.Fold(&codec)
```

The `error` return exists because `Fold` is one signature serving cases that _can_
fail (applying a result rolls dice and consults a decider) and cases that cannot
(projecting to JSON). Discarding it is correct at the codec sites and wrong at the
application sites. The comment is load-bearing; do not delete it and do not add
error handling that no test can reach.

### 3. Indexing a printed table is what makes a number data

`CLAUDE.md` Authority point 6, and it moves numbers **both ways**. The Aging
Table's last printed term indexes a column of the table, so it lives in
`aging.json` and `rules.Aging.LastPrintedTerm()` reads it from there — because it
is also E014's stamping condition, and holding it as a Go constant would let the
bands move without the term at which a record declares the reading moving with
them. The three-roll cap on Mustering Out Table 2 and the rank thresholds bound a
procedure and index nothing, so they are constants in `internal/chargen/muster.go` and
`internal/rules/rules.go` beside the code that applies them. `rules.go` states the test
in one line: "Being printed is not the test; indexing a printed table is."

If you find yourself moving a number, this is the rule you are arguing with. A
number on the wrong side of it is a finding, not a preference.

### 4. The record is the only way back to a character

`dice` depends on nothing and judges nothing, and it is not seeded (ADR 0001). The
die is one of six faces; a 2D throw is two of those in sequence, first then second;
`Among(n)` is one index in n, used only where the book hands a choice to a player
and names no basis for it. No run can be repeated: what a referee keeps is the
record, which logs every die of every roll and throw, and every draw as the choice
it produced. Every golden is generated from a fixture that scripts its dice:
**a throw the procedure does not make consumes nothing**, and a throw it makes
beyond the script fails the test naming it.

The strictness runs only one way. Nothing checks that a script is consumed to
its end, so a fixture can carry dice and draws its path never throws and every
test stays green. When a change makes the procedure throw fewer dice, the golden
comparison catches it, but not "naming the die asked for" as the roster's
comment promises. `-regenerate` then accepts the shorter path and leaves the
unread tail in the script. A fixture is therefore an upper bound on its path's
dice, not an exact statement of them, until something holds the other side.

The fixtures are also where the record proves it is a way back. Every
script was extracted from its golden's own event log, and the goldens
regenerated byte-identical. So a scripted roller plus a decider that answers
from the choice events is all a re-walk of a record needs. ADR 0001 keeps that
`replay` unbuilt until a referee asks for it. The machinery to build it already
exists, in `fixtures_test.go`.

Three corollaries you must hold:

- Changing the consumption order changes which path every fixture walks. That is
  an ordinary change — the fixture that felt it fails naming the throw that moved,
  and its dice are re-scripted from a record that walks the new path — but never
  an accidental one.
- A `Decider` gets `Vary`, not `Roller`. A decider handed the whole roller could
  throw dice the procedure never makes, and a die thrown there reaches no event,
  so the record would stop carrying every die. `Vary` is
  `interface{ Among(n int) int }` and that is the only variation
  a decider may take. The narrowing is the value's and not only the parameter's:
  the one site that hands a decider a `Vary` (`internal/chargen/enlist.go`, `applyResult`'s
  `WeaponPick`) passes `varyOnly{a.run.roll}`, a wrapper with nothing on it but
  `Among`, so a type assertion back to `Roller` finds nothing. The run's own
  `Roller` passed bare would have kept its dynamic type, and the dice with it.
  `TestADeciderCannotReachTheRollerThroughVary` holds it, and refuses to pass
  vacuously if no fixture reaches the Weapon choice.
- The record is the only way back to a character (ADR 0001). Every rendering ends
  by saying so, and nothing prints a command to re-run.

### 5. Nothing decides on its own behalf, and the log is wired in from the first step

`chargen.Generate` takes a `Decider`. Three implementations exist — the auto
`Policy`, the interactive `player`, and the scripted deciders the tests use — and
the engine cannot tell them apart. A method is called **only when more than one
answer is legal**. The `Decider` contract gives the reason: "a question with one
answer is not a choice, and asking it would put an entry in the generation
record that no reader could have decided differently." `reenlist.go`'s `intent`
is where it bites most often. After term 7 the only offer is Retire, and it is
settled without asking.

The engine also never offers an _empty_ set, and `Policy` is written on that
premise: its doc comment says every offered set must be non-empty, that nothing
in it checks, and that a direct caller handing it one gets the panic of indexing
it. That is a stated precondition rather than an enforced one, deliberately — a
guard for a set the engine cannot produce is the runtime check #41 removed, and
answering an empty offer would return the zero value of an alphabet, which is a
real-looking answer.

The log is not added afterwards, "because a log that is added afterwards records
what the author remembered." Every event carries a sequence number, and an
`OutcomeEvent` carries `Because` — the sequence number of the throw or choice that
caused it, or zero where it follows from the procedure itself. That causal edge is
what makes the record auditable rather than merely verbose.

### 6. `logging` is the single gate, and it rests on a stated assumption

`internal/chargen/logging.go` wraps every `Decider` — `generate.go` does this
unconditionally, so it is the only way a decider reaches the engine. It must
implement all twelve methods to compile, "so a choice point cannot reach the engine
without reaching the log."

Its `record` method does two jobs: it writes the choice event, and it **refuses an
answer that was not offered**. Checking there rather than at each application site
leaves one definition of "answered outside the offer." But it compares _rendered
names_, not values, because `MusterWeapon` answers with a sum and comparing a sum
against a list needs a fold — so one path would have to be strings whatever the
other eleven did.

That check is sound exactly as long as `String()` is injective on every alphabet
the engine offers. This was true and stated nowhere (#48); it is now stated in the
method's own comment and held by
`traveller.TestNoTwoValuesOfAnAlphabetShareASpelling` — which is itself held by
`TestEveryAlphabetDeclaredHereIsChecked`, a test that reads the package's own
source for `var X = [...]` declarations and fails on one nobody checks. The
data-driven weapon names are the exception, and they are covered from the other
end: `rules.liftWeapons` refuses a weapon listed twice.

This is the tightest reasoning in the repository. Understand it before you touch
`logging.go`.

The order inside each method matters. It names `from` _before_ it asks, because
the slice is the decider's to write to: every offered set is the decider's own —
built fresh for the question, or copied out of the ruleset — so a write cannot
reach a later character, and an offer named afterwards would let a decider
rewrite this question's offer and answer with what it wrote. Named first, the
forgery is not in what `record` checks, and is refused.
`TestADeciderCannotForgeItsOwnOffer` holds it.

That defence covers what a decider can write. The six yes-or-no points need
a second one, because what they offer is never handed to a decider at all:
`log.choice` stores it uncopied as the event's `Alternatives`, where the
watcher `WithObserver` installs and every reader of `Character.Events` can
write to it. So `logging.go`'s `offered()` builds the pair fresh for each
question, and a write through one event changes that event and nothing else.
It was a package-level slice until #230, shared by every yes-or-no event of
every character in the process and checked against by `record`, so one write
made every later "yes" refused — the exposure `MusterTables[:]` (#55) and
`rules.Weapons` (#137) were before they were closed.
`TestAYesOrNoOfferIsEachEventsOwn` holds it, for a finished record and for a
watcher. A watcher still shares an event's slices with the record it is being
written into: `WithObserver` promises that a write reaches no other generation,
not that it reaches nothing.

### 7. Each fact lives in one place, and each gate is verified by breaking it

A fact stated twice is held by a test whose catches are a record of the copies
drifting, so the tree states each fact once and the documents that remain
either are that statement or are generated from it:

- **The readings are `internal/rules/data/ERRATA.md`**, hand-written and lifted once by
  `rules` with the tables: every `Erratum` constant ↔ every `### E0NN` entry
  both ways, so a constant with no entry, an entry with no constant, or an id
  twice fails the lift naming the id (`rules.TestMalformedReadings`). Each
  lifted reading carries its prose as text.
- **The `--auto` policy is `chargen.Policy`'s doc comments**: each method says
  when it is asked and what each strategy answers, and `PolicyVersion` says
  what bumps it. There is no policy document to disagree with them.
- **`docs/COVERAGE.md`'s tables are generated** by `internal/coverage` from an
  `Implements:` marker on each implementation and a `Holds:` marker on each
  test or golden; a rule with only one side fails the generator naming it.
  The bracket in a marker is the rule's authority. For the book's rules it is
  a page, `[1:5]`. For the tool's own promises it is nearly always a PRD
  section name: `[FR9]`, `[CLI sketch]`, `[JSON conventions]`, `[FR11]`,
  `[Determinism]`. The generator checks none of them. So the PRD, which
  governs nothing, is still the cited authority for about fifty rows of the
  map.

Two second copies survive that sweep. Each `Erratum` constant in
`internal/traveller/erratum.go` carries a one-line paraphrase of its reading,
and the lift holds the ids to `ERRATA.md` but not that prose. Each choice point
says when it is asked twice, once on the `Decider` method and again on the
`Policy` method, and the engine's own conditions are a third statement. Read
the paraphrases as pointers to the readings, not as the readings.

`chargen`'s tests, beside the fixture roster, hold:

- every `ChoicePoint` ↔ every `Decider` method, and every choice point reached
  by some golden;
- in `stamping_test.go`, **each reading's stamping condition as a predicate over
  a finished record**, checked over every golden both ways — written from the
  reading's prose, never from the stamping code, because "a condition
  transcribed from the thing it checks is one reading written twice." A failure
  quotes the lifted stamped-on prose, and the readings no golden stamps by
  design are pinned in one list. When a reading's document and its code
  disagree and neither has been ruled wrong, the reading can be _parked_. It is
  then measured but not judged, and its count of disagreeing records is
  asserted, so settling the question fails the test until it is unparked. No
  reading is parked today. The mechanism is the documented route for the next
  disagreement, not dead code;
- in `strategy_test.go`, the schema's strategy enums ↔ the code's strategies.

**There is no Go library (#199).** `traveller`, `dice`, `rules`, `chargen` and
`render` are under `internal/`, so the compiler refuses an import of any of them
from outside this module, and there is no exported API for a tag to freeze or a
baseline to hold.

The command line is what a script depends on, and it has a baseline beside the
code it holds. `cmd/ctchargen/cli_test.go` types out each subcommand's flags with their types
and defaults and holds them **one way**
(`TestEveryBaselineFlagIsStillDeclared`): a flag gone, or retyped, or with a
moved default breaks a script that passed it, while a new flag breaks nobody and
is held instead by `usage_test.go`'s `TestEveryDeclaredFlagIsListed`, which walks
the declared set against the help's list. `TestTheExitStatusIsZeroOrOne` holds
`main`'s `exitStatus` to real runs. These are the two things `v1.0.0` freezes —
the record and the CLI — and each has a gate.

`CLAUDE.md` adds the discipline that makes these worth having: "A new invariant is
not done until a deliberate mutation has been shown to kill it, and the failure
names what was broken."

The converse holds too: a test exists because it holds something. An assertion
whose only possible failure is an edit to a `fmt.Sprintf` fallback's format
string holds nothing about the rules or the record, and the tree does not keep
such tests. Coverage is not measured here (#198): the ratchet that held each
package's uncovered-statement count was removed after it induced tests of
that kind (#166) and nothing else the record shows.

**The gate is local.** There is no CI. `task` is the gate, and a change is checked
when someone runs it and not otherwise — so every "held by" in this document
means "held by `task`, when it is run."

## The seams

### The package arrows

`traveller` imports nothing in the module, and `dice` imports nothing but
`math/rand/v2`. `rules` and `chargen` import `traveller`, and `chargen` also
imports `rules` and `dice`. `render` imports `chargen` and `traveller`.
`cmd/ctchargen` imports `chargen`, `render` and `traveller`. Verified against
`go list`; the rule in `CLAUDE.md` holds.

`traveller` is the domain — alphabets, values, sums. It does not roll dice, read a
table, or marshal JSON. `rules` holds the printed tables as embedded JSON and lifts
them into domain values, and **the lift is the validation**: a cell naming no
service, no characteristic, no benefit any row prints, or a target in a notation
pp. 2–3 do not use fails at `Load()`, which happens once. A table that will not lift
is a build defect that surfaces immediately, not a runtime condition some path
might reach. That reaches the Prior Service Table's draft row too: `internal/rules/lift.go`'s
`checkDraft` holds it to one die face per service, each face printed once, so a
draft die always names a service and `Rules.Draft` cannot miss for any die the
engine throws.

**The shared ruleset is read-only in fact.** `Load` is `sync.OnceValues`, so every
caller in a process holds the same `*Rules`, and nothing it hands out can write
to it. Its tables are unexported fields reached through methods (`Aging()`,
`Muster()`, `Eligibility()`, `Education()`, `Retirement()`, each returning a
value), `Retirement`'s pay map is unexported, and every method that returns a
slice returns a copy: `Weapons`, `Service` (its ranks and the DMs of all four
throws, through `Service.clone` and `Throw.clone`), and `Aging.At`. The comment
beside `Load` states it once, and `internal/rules/tables_test.go`'s
`TestAWeaponListIsTheCallersOwn` and `TestNothingTheRulesHandOutIsShared` hold
it by writing through each and reading a fresh `Load()`. A new method that hands
out a slice of the ruleset without copying it is the defect those tests exist to
catch, and they catch it only for the slices they name.

**`rules` transcribes pages, not the engine's needs.** Its scope is "every
table of pp. 4–25", not "every table the procedure reads". That is why the
passage prices and the 90% resale rate of pp. 21–22 are lifted, validated and
transcribed twice even though no step of the procedure, the record or a
rendering reads them. #57 and PRERELEASE P2-1 settled them as reference data
v1 applies to nothing. The same holds for the readings: `Rules.Reading` lifts
every field of every `ERRATA.md` entry, and only the stamping test reads one.
A field with no production caller in `rules` is not automatically dead. Ask
whether the page prints it before you remove it.

### The record boundary — the thinnest seam in the system

`render` projects a `*chargen.Character` into a flat `record` struct, and this is
where the domain's vocabulary and the wire's diverge. `internal/render/json.go` is honest
about it: "Where the two disagree — a sum that must marshal flat, an array-backed
profile that must marshal as six named keys — the codec here absorbs the difference
and the domain type keeps its shape."

The write side is held by folds: `enlistmentCodec`, `departureCodec`, the event
codec. Add a case to a sum and this package stops compiling.

**The read side is held by nothing.** `internal/render/decode.go` unmarshals into `record`
and deliberately does not rebuild domain types — "rebuilding an `Enlistment` or a
`Departure` back into an interface would give the renderer nothing it does not
already have, and would need a decoder for every sum." Consequently `SheetFrom` and
`TranscriptFrom` work on strings: `sheet.go` compares `ship.Kind ==
traveller.ScoutShip.String()` and `choice.By == traveller.ByPolicy.String()`.
The compiler cannot see those. What holds them is the round-trip test in
`internal/render/roundtrip_test.go` and the schema tests.

If you rename a `String()` on a domain value, the write side moves and the read
side silently does not. This is the seam to be careful at.

`docs/character.schema.json` is the third participant, and it describes the rules
this build has, not some perhaps future. The root is open — a later build may add
a top-level section, and a reader holding this schema loads the record and
renders what it understands. Everything inside is closed: the fourteen nested
objects refuse an unknown key, and the event kinds, the choice-point names and
the three strategy vocabularies are closed enums. `internal/render/transcript.go` prints
`(unknown event kind %q)` for a kind it does not know, which is how a reader
survives a record it could not validate — not a reason to validate it.
`internal/render/schema_test.go`'s `TestAnUnknownEventKindDoesNotValidate` holds the
schema to that, and `TestALaterShapeWithAnUnknownFieldStillValidatesAndRenders`
holds the open root.

Two record fields look like one claim and are two. `inputs` is what the run was
**given**. It names the three strategies on every record, a hand-walked one
included, because the flags are parsed in both modes and the schema has to be
able to refuse a word that is not a strategy. `policy` is what **answered**.
`policyThatAnswered` writes `PolicyVersion` only when some choice event says
`by: policy`, because naming a policy that decided nothing "is a claim the
record cannot support". A record a person walked carries
`inputs.career: "serve"` and no `policy`, and that is correct. Read the
strategies from `inputs` only where `policy` is present.

`render` prints what the record carries and escapes nothing. The one string an
operator types into a record is the name, and it is held at the command line
rather than here — see below.

### The command line

`cmd/ctchargen` is where strings become domain values and where they stop being
strings. `new` and `batch` declare their six generating flags once, as
`characterFlags` (`declareCharacterFlags`), and `characterFlags.inputs()` is the
one parse both share. It calls `chargen.ParsePolicy` once for all three `--auto`
strategies, at the boundary: "past here a Career is a Career, and no part of the
engine asks again whether it is one of three. That is why chargen has no Validate
any more (#41)." The `Inputs` and the `Policy` are both written from that one
parse, so what a record says it was given and what decided it cannot name
different strategies.

The same parse refuses a `--name` containing a line break, `\n` or `\r`. The
sheet and the transcript print the name as markdown, so a line break in it would
start sections the character does not have. That is the whole of the defence:
`render` does not escape, and a hand-edited record still prints whatever it
carries.

The two output channels are separated deliberately: `out` carries the record, the
sheet, the transcript; `asking` carries everything the tool says to the person
driving it, so that `ctchargen new | jq` pipes a record and not a
conversation. `batch`'s closing tally goes to `asking` for the same reason — "a
summary on standard output would be a line of JSONL that is not JSON." A batch
that stops partway still says what it wrote: the tally goes to `asking`, worded
as "written before the run stopped", ahead of the error — except where nothing
was written, and for `errSearch`, whose message already carries the counts.

The exit status is 0 or 1 and nothing else. `main` hands `run`'s error to
`exitStatus`, the one place that decides it: 0 for a run that did what it was
asked, 1 for one that did not. A character who died is a delivered record and
exits 0; help asked for returns `nil` and exits 0; a bare `ctchargen` is a usage
error. `errUsage` and `errSearch` differ in the message a person reads, not in
the status a script sees.

## What the system is shaped to accommodate

**A new reading.** Write the `### E0NN` entry in
`internal/rules/data/ERRATA.md` first, with its page cite and its stamping
condition as a predicate over the finished record. Then add its erratum
constant to `internal/traveller/erratum.go` and to `Errata`, stamp it at the
outcome it governs, and write the predicate into
`internal/chargen/stamping_test.go` **from the document's prose**. The gates
will tell you what you forgot in both directions. A comment that settles how
two readings combine is a reading too. If you find yourself writing "rather
than" beside a stamp, the decision belongs in `ERRATA.md`.

**Pricing passages, or a `replay`.** Both are anticipated rather than built. The
prices are already lifted (above). A re-walk needs only what the fixtures
already have: a scripted roller fed from a record's dice, and a decider that
answers from its choice events. Neither needs a new seam.

**A new choice point.** Add the `Decider` method; the compiler names every
implementation that must learn it, `Policy` among them, whose new method's doc
comment says when it is asked and what each strategy answers. Add the
`ChoicePoint` constant; `chargen`'s tests hold it to the interface. The schema's `point` enum
is the part the compiler cannot reach, and it is held by a chain of tests instead:
`chargen`'s `TestEveryChoicePointIsReachedByARecord` demands a golden that
reaches the new point, and `internal/render/schema_test.go`'s
`TestEveryGoldenMatchesTheSchema` refuses that golden until the enum learns the
name.

**A new `--auto` strategy for an existing question.** Add the constant, the
`String()` case, the entry in the unexported strategy slice (`careerStrategies`,
`skillsStrategies`, `musterStrategies`), its answer in each affected
`Policy` method's doc comment, the schema's
`inputs` enum, and bump `PolicyVersion` if an existing answer moved.
`chargen.TestTheSchemaNamesEveryStrategy` holds the schema's enum to the
strategy slice in both directions, so the enum cannot be left behind.

**A new mustering-out benefit or departure kind.** Add the case to the sum and
follow the compiler across `chargen` and `render`.

**A new rank or title name.** Use the book's word for the book's thing. A
service's position is a rank and what the Table of Ranks prints is its name;
"title" belongs to nobility. One word with two meanings is how two fields of one
record come to name each other's thing.

**A new rules table.** Add the JSON under `internal/rules/data/`, a wire type in `lift.go`,
a lift function that validates every cell against a closed alphabet, and — this is
not optional — an independent second transcription in `internal/rules/*_test.go` from the
same visual reading pass. Retyping the second copy from the first checks nothing.

## What would require rethinking something fundamental

**Concurrency.** `rules.Load()` is `sync.OnceValues` over one `*Rules` shared
process-wide, and that value now hands out nothing that can write to it, so the
ruleset is no longer the obstacle. Generation is single-threaded everywhere today;
`batch` is a sequential loop, and each member rolls from a stream of its own.
The rest of the package-level state is read-only in practice: the `traveller`
alphabets, the unexported strategy slices, and the lift's label tables. None
of them leaves its package by reference since the yes-or-no pair stopped being
one (#230), so parallel generation would not race on them.

**Cross-build record compatibility, before `v1.0.0`.** Deliberately unpromised.
`recordShape` is 1 and stays 1 until the tag; #96 explicitly declined the reading
that a pre-v1 shape change should bump it. Do not bump it to mark a change now.

**A `Roller` that never fails a reenlistment throw.** `run.serve()` is an unbounded
loop, and the comment is exact: termination is "a contract on the `Roller`, not a
property of this function." The fixture that walks past the Aging Table's last
column scripts a _finite_ list of dice for precisely this reason (#54). Nothing
enforces the contract.

**Injecting a decider that mutates what it is offered.** The engine's whole
premise is that a decider is a pure function of what it is handed. Every offered
set is the decider's own: `chooseService`, `offeredTables` and
`offeredIntents` build theirs per question, `chooseMusterTable` copies
`traveller.MusterTables`, and `rules.Weapons` copies its list. So a write cannot
reach a later character. Nor can it pass for an offer in this one: each
`logging` method names its offer before it asks, so an answer the decider wrote
into its own slice is not in what `record` checks against, and is refused.
`takeWeapon` does not re-check the answer because that gate already has;
`chargen.TestADeciderCannotForgeItsOwnOffer` holds it at both weapon choices.
The premise covers deciders only. The yes-or-no pair in idea 6 is never handed
to a decider; it is built per question, so a write through a finished
record's event reaches that event and no other question's offer.

## Where a maintainer who did not understand the theory would cause damage

- **Editing the wording of a log line.** It used to be load-bearing:
  `run.depart` chose the departure type by comparing the reason string against the
  literal `"reenlistment denied"`, so editing the sentence changed the domain
  outcome (#45). Fixed — the departure is now passed in — but the instinct that log
  text is inert is wrong here in general, because `render`'s read side compares
  `String()` output.
- **"Simplifying" a fold into a type switch.** It compiles, and it silently
  discards the only exhaustiveness guarantee the tree has.
- **Adding a `Validate()` to `chargen`.** #41 deleted exactly that apparatus. A
  runtime check for a value a closed type cannot hold is a sign the type is wrong.
- **Restating a precondition in a caller.** #52's lesson: `Retirement.Pay` used to
  return a real-looking zero for four terms and the only thing between that and a
  record was a guard in `run.pension`. A precondition belongs to the function that
  has it.
- **Reading a table out of `pdftotext`, or retyping the second transcription from
  the first.** Both defeat the defence the whole repository is built around.
- **Returning a slice of the ruleset from a new `rules` method.** `Load` hands
  every caller the same value, so a slice handed out uncopied is a way for one
  character's decider to change the next character's tables. Copy it, and add it
  to `TestNothingTheRulesHandOutIsShared`.
- **Handing a decider the `Roller`, or anything that can be asserted back to
  one.** `varyOnly` is what makes `Vary` the only variation a decider takes.
- **Hoisting the yes-or-no pair back into a package-level slice.** It reads
  like a harmless allocation saved, and it is the slice every later character
  would be offered and checked against, reachable through every record (#230).
- **Reading a fixture's script as the exact dice of its path.** It is held
  against running short and not against running long. After a change that
  throws fewer dice, look for an unread tail by hand.
- **Escaping in `render`, or relaxing the `--name` check.** The name is held to
  one line at the command line and `render` prints what it is given. Doing the
  first moves the defence to a place that would change every sheet; doing the
  second removes it.
- **Writing a test for coverage.** A test that can fail only when a format
  string is edited holds nothing. Coverage is neither measured nor chased.
- **Moving a package out of `internal/`.** That exports a Go API, which is a
  third promise beside the record and the CLI, and #199 decided there is none.
- **Consulting the repository's own history at or before `41a213a`.** The clean
  room covers it, at every rank but one: only that tree's _errata_ may be
  consulted, only where the page and `ERRATA.md` are both silent, and only as a
  pointer to a page to go read.

## Uncertainties

Marked because you should know which claims to trust.

**Inferred from code alone.** I did not open the Traveller PDFs — they are outside
this repository and outside the review's scope. Every page citation above is
repeated from the code's own comments and `internal/rules/data/ERRATA.md`. I verified that the
citations are internally consistent and that the gates hold them to the documents;
I did **not** verify any of them against the printed page. If a transcription is
wrong, nothing in this document would catch it, and the twice-transcribed
discipline is the only thing that would.

**`docs/PRD.md` "governs nothing," but code still appeals to it.**
`CLAUDE.md`'s document table says the PRD governs nothing now and is kept as
history. Yet `cmd/ctchargen/render.go` justifies a parse decision with "the PRD's
CLI sketch says so plainly: flags precede the filename," `new.go` argues from
"Neither flag is in the PRD's CLI sketch," and `internal/traveller/choice.go` quotes the
PRD's "a rendering of that interface, never a second list kept parallel to it"
as the standard `ChoicePoint` is held to. All three are historical rationales
rather than live authority, and I read them that way — but a maintainer could
reasonably read them as the PRD still governing. For those three comments,
nothing is wrong and the fix would be a wording preference.

The coverage map's cites are a different matter, and they make the PRD's status
less settled than `CLAUDE.md` says. About fifty `Implements:` markers name a PRD
section as their rule's authority (idea 7). Three of them cite
`[Determinism]`: `chargen.Generate`, `chargen.Ruleset` and
`cmd/ctchargen.stamp`. That section is the seed contract ADR 0001 withdrew: one
recorded seed, `base + i` batch members, a character that "can be regenerated".
`ERRATA.md` does the same in places. Its preamble calls it a "Companion to
`PRD.md`", and E002, E004 and E014 argue partly from PRD sections. I cannot tell
from the tree whether the PRD's section names are meant to stay as the citation
vocabulary for the tool's own promises, or are leftovers no sweep has reached.
The withdrawn `[Determinism]` cite is wrong either way.

**Settled: the strategy vocabulary is part of the record's shape.** The schema
describes the rules this build has, so `inputs.career`/`skills`/`muster`, the
choice-point names and the event kinds are closed enums, and growing any of them
after `v1.0.0` is a change to a frozen shape, said in the record's `record`
number. That is a decision, not an oversight.

**Settled: growing `Decider` or a sum's cases interface is not a version
question.** Every implementation is inside the module, under `internal/`
(#199), so a new method breaks only code the same change must fix, and the
compiler names each one. Idea 1's growth path costs nothing after the tag.

**E004's age and the engine's age agree only until a crisis is survived.** E004
records a character who dies in term N at "age 18 + 4N", and defends that
formula as the relation the Aging Table's note fixes. `run.survive` instead
records "the character's own [age], advanced by the fatal term's four years ...
rather than recomputed from the term count", so the months a survived medical
crisis added stay on the dead character's age. I believe the engine follows the
page here, since pp. 7–8 add the months and nothing takes them away. But the
choice is made in a Go comment and not in `ERRATA.md`, which is the one place
`CLAUDE.md` says a reading may live. Two survived crises whose months reach
twelve would also put the recorded years past 18 + 4N. No golden walks that
path, and `TestACareerPastTheTablesLastColumn`'s 18 + 4 × terms assertion would
not hold for one that did. Treat E004's formula as describing the usual case,
not every record.

**Settled: `inputs` and `policy` answer different questions.** A hand-walked
record names strategies in `inputs` and carries no `policy`. Both `new.go` and
`policyThatAnswered` argue for this, and I read it as deliberate.

**The `Roller` termination contract is documented and unenforced**, and I take that
as deliberate rather than as drift — the comment names the failure mode and the one
test that could hit it works around it explicitly. A reader who wanted a hard bound
could reasonably disagree. Not filed.

**Test coverage of `--survivors` under adversarial policies.** I read
`survivors_test.go`'s structure but did not trace every case. The `survivorAttempts
= 100` bound is argued from a referee's measurement of 74 deaths in 100 under the
default strategy; I did not re-measure.

**Consistencies I checked, and they held.** `CLAUDE.md` claims "fourteen nested
objects" carry `additionalProperties: false`. The schema has fifteen
`additionalProperties` sites: one `true` at the root and fourteen `false` inside.
The count is right. The package arrows match `go list`. Every test, function
and quotation this document names exists in the tree as written.

**Two claims I tested by breaking them, in a scratch copy and not in this
tree.** Writing `"maybe"` into one finished record's yes-or-no alternatives made
the next character fail with "answered outside what was offered"; #230 closed
it, and `TestAYesOrNoOfferIsEachEventsOwn` now makes that write. Appending four
unused dice and two unused draws to `other-serve`'s script left the whole
`chargen` suite green. That one is stated above as a fact about the code, not
as an intention. I do not know whether it was a known and accepted cost.
Nothing in the issue history says so.
