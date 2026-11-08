# A theory of ctchargen

> Release snapshot for `v1.0.0-beta.6`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots").

## What it is for

A Classic Traveller referee rolls characters at the table, and the 1977 rules
make that a long walk through Book 1 pp. 4–25: six characteristics, an
enlistment throw that may fail into a draft, terms of four years each with a
survival throw that kills on failure, commissions and promotions, skill rolls
on tables the character's Education may close, a reenlistment throw that can
compel another term, aging that can push a characteristic to zero and into a
crisis, and mustering out on two benefit tables. The tool walks that procedure
and writes down what happened.

Everything in the design follows from one claim the tool makes to a referee:
**any number on the sheet can be checked against the printed page from the
record alone.** The record is a JSON document that logs every die thrown, in
order, every choice put and its alternatives, every consequence, and every
place the text was silent and a reading had to be made. It is not a summary of
a character. It is the evidence that the character was made by the rules.

Two consequences shape the rest. First, the authority is a specific printed
text (the FFE reprints of the © 1977 Books 1–3), not "Traveller" as anyone
remembers it, and the code is organised so that every rule can be traced to a
page. Second, the record is the only way back to a character. The dice are not
seeded, so there is nothing to re-run (ADR 0001), and the record therefore has
to carry everything, including which build wrote it (ADR 0002).

## The page, the readings, and the font trap

The rules come in three ranks: the held page, then the readings recorded in
`internal/rules/data/ERRATA.md`, then, as a pointer only, the errata of the
implementation removed at `41a213a`. Nothing is implemented from memory,
because memory of Traveller is mostly the 1981 revision, which differs exactly
where it matters (survival failure is death, with no "injured instead").

The tables reach the code through a hazard that shaped the whole data path.
The reprints' embedded font substitutes glyphs, and the substitutes look like
data: a "—" cell extracts as the digit 4, a minus sign extracts as a 3. So
tables are read visually, one page at a time, and transcribed twice from that
one reading, once into `internal/rules/data/*.json` and once into the `rules`
tests. The second copy is the check. It catches a glyph the eye also misread
only when the misreading resolves to nothing, which is why a skill name the
three books never define is treated as a broken glyph (`Rifl3`) and not as a
new skill.

A **reading** is a decision where the page is silent or contradicts itself.
Each has an id, a page cite, a rationale, and a **stamping condition**: a
predicate over the finished record saying when that reading governed it. A
record carries the ids of the readings that governed it, in `errata`. The
readings document is not commentary. `rules` embeds it and lifts it on every
load, and a heading with no `traveller.Erratum` constant, or a constant with no
heading, refuses the load. So a malformed `ERRATA.md` stops the binary, which
is deliberate: a reading the code applies and the document does not describe
is the one failure the record cannot reveal.

The stamping conditions are checked by `stamping_test.go`, and the way they are
checked encodes a lesson. Each predicate is written from the document's prose,
never from the stamping code, because a predicate transcribed from the code it
checks is one reading written twice: the same trap the double transcription
exists for. The predicates read the checked-in goldens and key on transcript
step names, so rewording a step ("enlistment,", "aging,") can silently empty a
predicate. `TestEveryStepTheConditionsReadIsFound` exists because the split of
rolls from throws (#50) once did exactly that to E011.

## The domain, and why it is all sums

`internal/traveller` is the vocabulary a referee and the code share, and it
imports nothing else in the module. Its load-bearing idea is that a thing which
is "exactly one of" several cases is a sealed sum with a `Fold` over a cases
interface: how a character entered a service (`Enlistment`), what a skills
cell does (`TableResult`), what a Table 1 row grants (`BenefitRow`), how a
service ended (`Departure`), what a weapon benefit is (`WeaponBenefit`), and
every kind of `Event`. Adding a case adds a method, so every fold stops
compiling until it handles the new case. Non-test code never type-switches on a
sum. That is why "a renderer cannot forget a case" is a property of the build
and not of review. Plain enumerations (services, characteristics, skill
tables, choice points) are held instead by the `exhaustive` linter, which is
the weaker half of the same promise.

Every closed alphabet's `String()` is injective, and the system leans on that
in a non-obvious place. The engine refuses a decider's answer that was not
offered by comparing **rendered** names, because one choice point (the muster
weapon) answers with a sum, and comparing sums needs a fold. Two values that
spelt the same would let a decider answer with one and have the other
recorded. `traveller`'s spelling tests hold injectivity, and a second test
greps the package for every `var X = [...]T` so a new alphabet cannot escape
the first.

Zero values fail closed: the zero `Target` is "0 exactly", which no throw
meets. A characteristic has two floors because the page has two: an ordinary
alteration floors at 1, aging at 0 (E010), and zero is the doorway to the
medical crisis of pp. 7–8.

## The engine: a procedure that asks, and a record that cannot lie about who answered

`chargen` walks the procedure in the exposition's order, not the worked
example's (E002), one method per page section on a `run`. It never decides
anything on its own behalf. Every question goes to a `Decider`, which has one
method per choice point, each typed to its own alphabet. The interface is wide
on purpose. A single method over a label would let a new choice point be added
without every decider noticing, and `TestChoicePointsMatchTheDecider` holds the
method names, the `ChoicePoint` alphabet and the schema's `point` enum to one
another. Renaming a Decider method therefore changes the record.

There are two ways in, and the difference is a fact the record states once.
`chargen.Auto(in, policy)` is an **auto run**: the `--auto` policy answers
every choice point, and the record carries a `policy` object (its version and
its three strategies) whether or not the policy was asked anything.
`chargen.Guided(in, decider)` is a **guided run**: a person answers, and the
record carries no policy. The absence is the statement. Choice events do not
carry who answered. Before #276 they did, and the default credited the policy
with any decider's choices, which the worked-example test was quietly doing.
A quarter of forced-scout records are auto runs the policy answered nothing in
(killed by the first survival throw), and the golden `scouts-died-unasked`
exists to hold the rule there.

Every decider is reached through `logging`, a decorator that implements every
Decider method, so a choice point cannot reach the engine without reaching the
log. It is the single place the **offer**'s rules are kept. It renders the
offer before asking, so a decider writing into its slice cannot put its own
answer there and pass. It hands each decider its own copy of every slice,
because a shared slice once let one decider's write change what every later
character was offered (#55, #137). It refuses a list question with fewer than
two alternatives, because the Decider contract is that a method is called only
when more than one answer is legal, and a question no reader could have
decided differently has no business in the record. The sites that can reach
one answer (reenlistment at the seventh term, the third roll on Table 2)
settle it themselves as readings of the page. The gate's refusal is there for
the site that forgets.

The `--auto` policy describes itself. Each `Policy` method's doc comment says
what each strategy answers at that choice point, and there is no other policy
document. `PolicyVersion` moves only when an answer changes, which is what a
referee reads to tell which answers made a record.

## Dice, and the discipline of consumption

`internal/dice` is an unseeded stream. A decider never sees it. The one choice
point that needs variety (which weapon, #34) is handed a `Vary`, which can draw
among alternatives and do nothing else, wrapped so that a type assertion cannot
recover the roller (#136). A die thrown by a decider would reach no event, and
the record would stop carrying every die.

The order dice are consumed in is load-bearing, though not for replay, which no
longer exists. It is load-bearing for the fixtures. A **fixture** is a
**script** (the dice and draws of one path, in the order the procedure asks for
them) plus the inputs and policy that walk it, and its record is checked in as
a **golden**. A **walk** is the command tests' equivalent with no golden: a
script extracted from a record the tool wrote, proven only by the test that
drives it. Both are played through `internal/scripted`, which is strict both
ways: asking past the end of a script fails, and so does finishing with any of
it unread. A 2D throw is two dice in sequence, and a throw the procedure does
not make consumes nothing. Changing the order is an ordinary change. The
fixture that felt it fails, and its dice are re-scripted from a record that
walks the new path. A change that keeps the count and moves the order fails
only as a golden diff (#289), so the diff, not the roller, is what names the
throw that moved.

Goldens are regenerated (`go test ./internal/chargen -regenerate`), never
hand-edited. The same fixtures write the schema's two published examples, so
the documented record cannot drift from the emitted one.

## The rules lift: validation by construction

`internal/rules` embeds every table of pp. 4–25 and lifts it once per process
(`sync.OnceValues`). "The lift is the validation" means that every string in
the JSON goes through a parser keyed to `traveller`'s own alphabets, and shape
errors are refused before any character is generated:

- columns other than the six services in p. 10 order;
- a bare "4" where a Table 1 cell should be a benefit, which is the font trap;
- a commission printed without ranks;
- retirement rows that do not ascend.

Past the lift, a `Career` is a `Career`, and nothing re-checks it. Everything
the lift hands out is a copy, because `Load` gives every caller in a process
the same `Rules`.

Where a number lives is decided by one rule rather than argued case by case.
A number that **indexes** a printed table is data (the Aging Table's last
printed term, which E014 turns on, lives in `aging.json`). A bare procedural
bound is a constant beside the code that applies it, with its cite: the
three-roll cap on Table 2, the rank thresholds, the seven-term voluntary cap.
Being printed is not the test. A number on the wrong side is a finding in
either direction (#65).

## The record and its projections

`render` turns a `Character` into the record, and turns the record, never the
domain, into the sheet and the transcript. `SheetFrom` and `TranscriptFrom`
decode a record written earlier and render it through the same code, so
reading a record back is the same operation as rendering one fresh. The sheet
reads the record's events on purpose (the headline explains a draft by finding
the service choice), and an event that will not read fails both renderings
rather than being skipped. A transcript ending "the record is the only way
back" about a record it could not finish reading would be lying.

The record's shape is governed by a freeze that has not happened yet. Until
`v1.0.0` it moves freely (#62), and `record` stays 1 through every pre-release
change (#96). From the freeze it is a public contract. The schema
(`docs/character.schema.json`, hand-formatted) opens only its root object, so a
later build may add a section that an older reader skips, while every object
inside stays closed (#113). Old records are not promised to read: a beta.5
auto record, whose `policy` is a bare number, is refused rather than read as a
guided one, because misattribution is worse than refusal.

## The command

`cmd/ctchargen` keeps two channels apart. The record, sheet or transcript goes
to stdout; questions and the running commentary go to stderr. `ctchargen new |
jq` pipes a record, not a conversation. Interactive mode shows each throw
between questions through `WithObserver`. A session that stops is lost, and the
tool says what was answered and that nothing was written, rather than offering
a way back it cannot honour.

Every generation the command makes takes `source.options()`, which carries the
build name (`chargen.WithBuild`) and, under test, a scripted roller. That one
call is why a record cannot leave the command without its `build`. A test
generating in-process passes no build, so goldens do not differ by machine.

`batch` is many auto runs, written as JSONL to stdout (streamed, one record at
a time), to a file (buffered, so a failed run opens nothing), or to a directory
(one file per member, named by position so `--survivors`, which passes over the
dead and generates more, changes who fills a slot but not which files are
written). Collisions are checked before anyone is generated, from paths alone.

## The gate holds the documents to the code

There is no CI (#149). `task` is the gate, and much of what it holds is the
repo's own documents. `docs/COVERAGE.md`'s tables are generated from
`Implements:` markers beside the code and `Holds:` markers on tests. A rule
implemented and held by nothing, or held and implemented by nothing, fails the
generator, and so does a cite to an authority that does not exist. A page
cite's internals are not checked (#287). The schema's strategy enums are held
to the code by a test. The CLI's flags and defaults are typed out in
`cli_test.go` and held one way, because a new flag breaks nobody. And a new
invariant is not considered done until a deliberate mutation has been shown to
kill it: `go run ./internal/mutate` edits a scratch copy, runs a command, and
prints `KILLED` or `SURVIVED`. Note that it prints `KILLED` for any failure,
compile errors included.

The tests call code the way its user does. The interactive session is tested
through the command, because what is under test is what a person sees:
questions on stderr, the record kept off stdout, the stop and what it cost
(ADR 0003).

## What it is shaped to accommodate, and what it is not

**Cheap, by design:**

- **A new reading.** Add a heading in ERRATA.md and a constant; the lift
  refuses one without the other. Stamp it where its condition holds and write
  its predicate from its prose.
- **A new `--auto` strategy.** It is a `Policy` method branch with its doc
  comment, plus a bump of `PolicyVersion` if any answer changes. The schema's
  enum test will demand the new word.
- **A new choice point.** It breaks every Decider at compile time, the logging
  gate included, and needs a fixture that reaches it, a schema `point` value,
  and a choice-point check.
- **A corrected table cell.** Change both transcriptions from a fresh visual
  read of that page alone.

**Fundamental: rethink before touching:**

- **Seeding the dice or offering replay.** ADR 0001 withdrew it and took the
  seed off every record. Bringing it back reopens the record, the fixtures and
  the interactive session at once.
- **Anything that lets a record omit a die or a choice.** For example, a
  decider rolling its own dice, or a summary event. The audit claim rests on
  completeness.
- **Changing the record's shape after `v1.0.0`.** Before the freeze it is
  routine; after it, the `record` field counts.
- **Rules outside Books 1–3.** Books 2 and 3 are consulted only where Book 1
  points; Book 4 and later are out of authority. A referee's house rule has no
  home here, and adding one would break the "checkable against the page" claim.

**Where a maintainer who missed the theory would do damage:**

- **`task fix`.** It has already rewritten verbatim quotations of the book:
  `godot` and `dupword` are configured off now, but read the diff.
- **Trusting `pdftotext`, or reading a table in a page range.** The prerelease
  review lost a cell to a six-page read.
- **Hand-editing a golden.**
- **Rewording a transcript step name** without checking the stamping
  predicates.
- **Editing an ERRATA heading** without the same edit, verbatim, in
  `traveller/erratum.go`.
- **Rewording a rule name on one side of a marker pair.** The name is the
  join key.

## Uncertainties

Some of what I state above as intent is inferred from code, comments and
issues. The rest is held by tests, and that half I trust.

- **The stamping predicates key on step-name prefixes.** I read this as an
  accepted cost of writing them from prose against the record a referee sees,
  rather than against domain values. I did not find it argued anywhere. If it
  was never chosen, a structured event field would hold the conditions more
  firmly.
- **The lift parses all of ERRATA.md on every load,** about 40 KB of prose
  that only tests consume field by field, so the binary can refuse to start on
  a malformed document. I take that as deliberate (the readings are part of
  what the tool is), but it may be habit rather than decision.
- **E012's normalization map has entries that never fire** on the shipped
  data: `services.json` already types "Rifle" and "Submachine Gun", so `Rifl3`
  and `SMG` are exercised only by a direct test. They may be kept as the
  written-down font-trap knowledge ERRATA.md says the entry exists for, or
  they may be leftovers. The entry's own text suggests the former.
- **A withdrawn reading has no form the lift accepts.** The lift needs six
  non-empty labelled fields, and the stamping harness needs a predicate. No
  reading has been withdrawn yet, so the path is untested, and I cannot tell
  whether "withdrawn" is meant to be a state or a deletion.
- **Under `-regenerate`, goldens are written with a plain `os.WriteFile`**
  while other tests in the package read `testdata/*.json` in parallel. By
  reading I would expect those readers to see a half-written file now and
  then and report a stale golden on the regenerating run. I have not seen it
  happen.
- **Where the boundaries of "gate" fall** is unsettled in the documents
  (#293). The word names both the logging decorator ("the single gate") and
  the checks `task` runs. The coverage markers for check rules sit on test
  helpers, not on package docs as CODING_STANDARDS.md says.
- **Two lifts still range over maps** when counting rows (#286), against a
  rule (#53) their own marker promises. That looks like drift from a fix that
  reached one loop and not the others, not a design choice.
