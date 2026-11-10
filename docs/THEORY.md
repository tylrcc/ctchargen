# A theory of ctchargen

> Release snapshot for `v1.0.0-rc.4`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots").

## What it is for

ctchargen makes a Classic Traveller character the way Book 1 pp. 4–25 says
to, and hands the referee a note he can check against the page. That second
half is the point. Any generator can print a UPP and a skill list. This one is
built so that every number on the sheet goes back to a printed page, a die
that was thrown, and a choice someone made, and where the book is silent or
contradicts itself, to a decision recorded here under its own name.

The world it models is one procedure. Six characteristics are rolled. There
is one enlistment attempt, with the draft behind it. Then come terms of
service: survival, commission, promotion, skills, reenlistment, aging. Then
mustering out, and last a title the final Social Standing may confer. The
procedure has throws (2D against a target with modifiers), rolls (dice read
off a table, with nothing to meet), and points where the player chooses.
Each is a noun in the code: a `traveller.ServiceName`, a `traveller.Term`, a
`ThrowEvent`, a `RollEvent`, a `ChoicePoint` and a method on
`chargen.Decider`. A place the page is silent is a `rules.Reading`, and the
id a note carries for it is a `traveller.Erratum`. GLOSSARY.md keeps that
vocabulary, and its "avoid" lines matter, because some words the code uses
carefully are ones a sibling tool would use loosely: a reading is not an
erratum, a walk is not a fixture, a script is not a seed, the record is the
note's generation record and not the whole note, and the service history is
not the log.

The authority model outranks everything else. The © 1977 text governs, and
memory of later editions does not. A reading decided here governs only where
the page is silent. The errata of the implementation deleted at `41a213a` are
a last resort, and even then only a pointer to a page that has to be reread.
A maintainer who leaves this model out of his understanding will "fix"
something into the 1981 rules; the best-known trap is that survival failure
is death, with no injury option (p. 5).

rc.4 changed what a referee holds. ADR 0006 removed the JSON record:
`new` and `batch` write one markdown note per character, and **the note is
the record**. It opens with YAML properties a vault can query and a summary a
referee can read aloud, then the sheet, then the generation record, then the
ruleset, the build and who answered. Nothing reads a note back. Behind that
visible change sits the organizing idea of this release, which the next
section is about: inside the process, **the event log is the one store of
the character**.

## The log is the one store

### What the character is now

`chargen.Character` keeps five fields: `Inputs` and `Policy`, which are what
the generation was given; `Events`, the log; `Errata`, the stamp of every
reading that governed it; and `Build`, which names what wrote it. Everything
the sheet says about the character is a method that folds the log: `Profile`,
`Skills`, `Benefits`, `Pension`, `Nobility`, `Enlistment`, `Rank`,
`RankName`, `Age`, `Departure` (all through one `sheetFold`), `Terms` (the
number of the last term a step opened), `History` (through `historyFold`),
`Attempted` (the first `EnlistedFact`) and `ServedIn` (a fold of the
folded `Enlistment`). Character's doc says it outright: nothing it says of
the character is kept on it beside its log.

That is the end of a line of work, and ADR 0006's dated amendments record
each step. Dropping the JSON left the service history stored twice, once as
data and once as the events' prose. #412 and #397 made the events the one
store for the service history and the service attempted. #441, in three PRs
(#450, #451, #452), did the same for the characteristics and skills, then the
benefits, pension and title, then the career. The goldens were byte-identical
across all three; the change is in where a fact lives, not in what a note
says.

### How an outcome carries a fact

An `OutcomeEvent` has prose (`Description`) and, since #412, an optional
`Fact`: a sealed sum in `traveller` with a cases interface, `FactCases`, of
twenty-six methods. The engine records a fact through `log.fact`, which sets
the outcome's `Description` to `fact.String()`. So a fact's prose is its own
`String`, and the line a referee reads and the value the sheet folds cannot
say different things. An outcome that changes nothing a sheet reads ("not
promoted this term", "a second scout ship, which is lost", "nothing: the
table prints a dash", an aging throw that holds) is written through
`log.outcomef` with prose and no fact.

The rule that makes the folds simple is **a fact carries the value the engine
reached, and the fold applies no rule**. A skill's fact carries the level it
reached, not "+1"; a cash fact carries the running total beside the roll; a
repeat Free Trader's fact carries the ship as it stands after ageing and
paying down; a characteristic change carries `From`, `To` and the table's
`Delta`; every fact that moves the age carries the age it left him at. So the
fold for skills is "the last level per name", for cash "the last total", for
age "the last age", and the fold never re-derives p. 12's accumulation, p.
22's repeats or E004's fatal four years. Where a rule would otherwise have to
live in the fold, the fact's type carries it instead: a Free Trader received
again is its own case, `FreeTraderAgainFact`, rather than a flag on any
ship, and `sheetFold.Drafted` converts `traveller.DraftedFact` to
`traveller.Drafted` by a Go conversion, so a field added to either type stops
the fold compiling.

Some facts are stated where a fold could have computed them, and that is
deliberate. `StartedFact` writes the rolled UPP and the starting age as a
line of the record; the profile is read from the six `RolledFact`s and the
fact contributes only the age. `RetiredFact` carries `Denied`, because p. 21
makes a departure at term 5 or later retirement whatever caused it, and the
record still says whether the throw refused him.

The service history is a second fold over the same facts. Each
`AppliedFact` and `WeaponPickedFact` carries a `Source` (on entering, at a
rank, in a term, at mustering out), and `historyFold.gained` files a gain by
it: on the enlistment line, under the raise that conferred it, in the term,
or nowhere, since mustering out is not a term and the sheet's possessions
already list it. A term is the `StepEvent` whose `Opens` field names it, so a
term a survival throw ended is still a term served, with nothing in it.

### Working state alongside the log

The engine cannot read its own log as it runs: every DM reads the current
characteristics, the commission and promotion throws read the rank, mustering
out reads the terms and rank, the repeat rules read the benefits so far, the
pension reads the term count, and the end reads death off the departure. So
`chargen.run` keeps working state privately: `profile`, `skills`,
`rolledSocial`, `benefits`, `enlistment`, `terms`, `rank`, `atAge` and
`departure`, beside `eligible` and `trained`, which never reach a note. Each
change to working state is logged as a fact carrying the value it left (or,
for terms, as the step that opens one), and the finished `Character` is built
from `Inputs`, `Policy`, the log and its stamped errata. The working state
dies with the run.

This is a second statement of every fact, kept for the length of one
generation, and the theory's honesty depends on naming it. The two cannot
disagree on any path a golden walks: #450-#452 each showed mutations to the
facts and folds (an expertise level off by one, the Profile fold skipping
`AgedFact`, a cash `Total` set to the roll, a recovery's age dropped) killed
by a golden diff. Nothing checks the two directly, so a path no fixture walks
is held only by reading the engine: each assignment to a working field sits
next to the `log.fact` that states its result.

### What it costs

- **The sum is wide.** `FactCases` has twenty-six methods, one per kind, and
  every fold implements all of them; `historyFold` is mostly no-op methods
  saying why each fact is the sheet's and not the service's. The width is the
  closure: a new kind of fact stops both folds compiling until each says what
  it does with it.
- **Every question folds every event.** `render.viewOf` asks the character
  for about a dozen facts, and each `sheetFold` method walks the whole log and
  keeps one field of the fold's result. A note costs a dozen passes over a few
  hundred events. It is nothing next to writing a file, and a batch holds one
  character at a time, but a maintainer adding a hot loop over `Character`
  methods should know each call is a walk.
- **A sheet fact must come through `log.fact`.** An outcome logged through
  `outcomef` reaches the record and not the sheet. A maintainer who adds a
  benefit and logs it as prose has written a note whose sheet omits it, and
  only a golden that walks the path notices.
- **The record's wording is now bound to the domain types.** A fact's
  `String` is the record's line, in `traveller`, so rewording a line is a
  change to the domain package and moves every golden that prints it.

### What it makes possible, and what it makes hard

It makes a character constructible from its record's own terms. The render
tests build a `Character` as nothing but a log, `logged(rolled(...),
enlisted(...), opened(2), traveller.RaisedFact{...})`, and assert on the note.
Before, a test set fields directly and could set a sheet the log
contradicted. It also leaves the replay ADR 0001 kept possible a little
nearer: the in-process log is now sufficient to rebuild every sheet fact.

It makes a rule that reads history expensive to put in the wrong place. A
question like "was he ever commissioned?" belongs either in the engine's
working state, if a throw needs it, or as a fold over facts that already
carry the answer; a rule computed inside a fold is the thing #441 removed.

It leaves the note itself unreadable by a tool. ADR 0006 is explicit: a
`replay` would need a record a tool can read, and the note's record is prose.
The typed facts exist only in the process that generated them.

## The page, typed as printed, and read twice

The reprints' embedded font substitutes glyphs, and the substitutions look
like data: a dash cell comes out as the digit 4, a minus as a 3, "Travellers'
Aid" as `Travellers9`, and p. 23's box shows `Rifl3-1` even in the rendered
image. That one fact explains much of the repository's shape.

Since ADR 0005 the first transcription is Go: `internal/rules/transcription.go`
holds every table as a literal of the page's text, with its glyphs,
abbreviations, notation and dashes. It holds `"Intel 8+"`, not
`"Intelligence 8+"`; the em dash wherever a service has no commission, rank
or benefit; `"DM of +1 if"` as a row heading; the Aging Table's header rows
and dotted-leader spans; and `"Rifl3-1"` as the reprint shows it. **Structure
comes from the domain and cells from the page**: which table a grid is and
which service a column belongs to are positions keyed by `traveller`'s types,
so a table the type does not name, or a cell too many, does not compile. A
short Go array is padded with `""` in silence, so `refuseEmpty` refuses an
empty cell on the ground that the page prints a dash where it means nothing.
The service tables keep the page's rows as rows, the Table of Ranks included
(#358), so the two transcriptions read every grid in opposite directions.

**The lift is kept on purpose.** ADR 0005 rejected typing the tables as domain
values directly: a broken glyph would then fail to compile, but every
normalization and every dash would be decided in the transcriber's head, which
is a reading applied silently. The lift is where those judgements are written
down. A printed name is checked against `traveller`'s closed alphabets; what
the domain does not spell lives in `rules` — `abbreviatedCharacteristics`,
`boxServices`, E012's spellings (`Rifl3` → `Rifle`, a font-trap glyph caught
by the semantic cross-check CLAUDE.md describes), and the fixed text around
values. `traveller.Sign` is the one place an en dash is recognised and refused
(#366). `rules.Load` lifts once through `sync.OnceValues` and keeps its error,
since a literal can still hold text the lift refuses. The lift refuses shape;
the few value checks it makes are sanity bounds or cross-checks of the page
against itself (the Age header against `traveller.StartingAge`, which lives in
`traveller` because `rules` cannot import `chargen`). Whether a value is the
one the page prints is the second transcription's job, in the `rules` tests,
written a service at a time in normalized words.

CLAUDE.md's Authority point 6 sorts numbers: one that indexes a printed table
is in the transcription; a bare procedural bound is a constant beside the code
that applies it, with its cite (`maxRollsOnTableTwo`,
`minRankForTable1Modifier`, `lastVoluntaryTerm`, `freeTraderPaymentYears`).
The cost ADR 0005 accepted is that linters read the page's text: `goconst` is
excluded for the transcription and the `rules` tests, `mnd` for the
transcription, and `task fix` over this file is as unsafe as over a verbatim
quotation.

## Readings, errata and stamping

A reading has three expressions, kept apart because the check between them
works only while they stay independent.

The **prose** is a `rules.Reading` value in `internal/rules/readings.go`,
verbatim with the page's emphasis as inline markdown. `rules.Readings()`
returns an array sized by its own literal, so a reading missing or one too
many does not compile, and `TestTheReadings` holds each `ID` to its position,
which is why erratum `e`'s reading is `Readings()[e-1]`.
`TestErrataListsEveryConstant` parses `traveller`'s source and requires
`Errata` to be exactly its `Erratum` constants, in order.

The **stamping site** is in the engine. An outcome a reading governed names
the erratum as a literal, and `log.stamped` takes the union in id order as
`Character.Errata`. The engine never reads a reading's prose, so a reading's
wording can change without changing a character.

The **predicate** lives in `internal/chargen/stamping_test.go`, one per
reading, written by hand from the stamped-on prose and never from the stamping
site (#66, #77), and checked both ways, record by record: an erratum stamped
where its condition fails is as much a defect as one missing. A failure quotes
the prose from `rules.Readings()`. The predicates read each fixture's
`Character` as the engine makes it now, through `recordOf`, which turns the
folded facts and the events into the words the conditions use. The note is
written from the same character, and `TestGoldens` holds the note: the two
are siblings, so a reading the engine stops stamping is named by an ordinary
run rather than discovered in a golden diff. E012 and E015 are stamped on no
record by design, which `namesNoRecord` pins and `unreached` checks; the
`parked` mechanism measures without judging a reading whose prose and code
disagree, and none is parked.

ADR 0004 gave an erratum a lifetime. From `v1.0.0` an erratum names one
decision for good; a reworded reading keeps its erratum, a changed decision
needs a new one, and a withdrawn reading stays in the array, is explained and
marked withdrawn, and is stamped on no new note. `verdict` holds a withdrawn
reading only to that last clause, and `TestAWithdrawnReadingOnAGoldenIsRefused`
stands E001 in as withdrawn, since none has been. The readings' text is this
build's text, whatever build wrote the note, and ADR 0004's rule against a
reworded decision is what makes that safe.

## The domain is sums, so the compiler counts the cases

`traveller` imports nothing in this module, and nothing in it rolls, reads a
table or writes markdown. Its rule for what becomes a type is that types carry
identity, never rule invariants. Wherever the book says "exactly one of", it
has a sealed interface with a `Fold` over a cases interface: `Enlistment`,
`TableResult`, `BenefitRow`, `Departure`, `WeaponBenefit`, `Event`, and, new
in rc.4, `Fact` and `Gain`. Go does not check a type switch for exhaustiveness, so
a fold is how adding a case breaks every consumer at compile time. The plain
enums are weaker: the exhaustive linter catches a missing switch case, and only
under `task`. `render.choiceWords` and `entryKind` are switches for that
reason, so a new choice point is a lint failure until it has the page's words.

A sum is only as right as its list of cases, and the list is meant to be the
page's: #364 removed a cash case of `BenefitRow` that no row of Table 1 can
produce. Some choices make sense only against a sibling design: the civilian
who declined the draft is a case of `Enlistment`, not a nil service with a
flag; "died" is two `Departure` cases because they carry different fields;
`Fatal` lives in `traveller` because it was once stated twice (#100); `Age`
carries months only because a survived crisis adds 1D of them; `RollEvent` is
not a `ThrowEvent` with an empty target, so a roll is never written as having
succeeded (#50); and a `Target`'s zero value is "0 exactly", which no throw
can meet, so an unset target fails closed.

## How the tree treats what cannot happen

A branch the module's own types make impossible goes. Since #199 nothing
outside the module can build a `Character`, and generation always logs an
enlistment and gives every served character a departure. So `servedIn` panics
on a nil `Enlistment` (#377), `render.foldEnlistment` does too (#393), and
`viewOf` folds a served character's departure without a guard (#445): a
generation defect crashes rather than printing a civilian or a living man. A
fold over a codec whose cases only assign drops the error with the comment
saying so, and `Character.fold`'s doc makes the same demand of every
`FactCases` handed to it.

The rule underneath is that a note is never written wrong. A defect may crash,
and an unwritable note is refused, but neither is absorbed into a note that
looks complete. Since the JSON went there is no encoder error to carry; what
remains outside the module is the file system, and the command returns every
error it gets from it.

## The engine asks, and the note says who answered

`chargen` walks the procedure and decides nothing on its own behalf. Every
point where the book lets a player choose is a method on `Decider`, twelve of
them, each typed to its own alphabet, which is what closes the set. A
reflection test holds the `ChoicePoint` enum to the interface's methods.

Three deciders exist — the auto run's `Policy`, the command's interactive
`player`, and the tests' scripted deciders — and the engine cannot tell them
apart. Each is reached through `logging`, the decorator `generate` wraps around
whatever it is handed, which keeps promises no single decider has to: a
question with fewer than two legal answers is refused before it is asked
(`asked`); each slice a decider is handed is its own copy (#55, #137, #230);
and an answer outside the offer is refused (`errNotOffered`). That last check
compares rendered names, sound only while `String()` is injective on every
alphabet, which `TestNoTwoValuesOfAnAlphabetShareASpelling` holds (#48).

Who answered is fixed when the run is entered. `chargen.Auto` hands `generate`
a pointer to its policy and `chargen.Guided` hands it nil, and that pointer,
kept as `Character.Policy`, is the whole of what marks an auto run. The note's
last lines name the policy's strategies and `PolicyVersion`, or say the player
answered every choice. An auto run is `new --auto` or any batch, which declares
no `--auto` because it has nobody to ask (#336). `PolicyVersion` moves only
when an answer changes, and is 4: since #420 the policy draws the service
uniformly among the six with the run's own dice, because a referee's batch of
120 came out 109 Army under version 3's likeliest-enlistment ranking. The
policy's documentation is its method doc comments, and `DefaultPolicy` is the
one place its defaults are written; the command's flags read them (#367).
`Attempted` reports whether the policy named the service: an unforced auto
run's attempt is the policy's, a `--service` or a guided answer is not, and
the headline says so when the draft put him elsewhere.

The narrowest seam is `Vary`. A decider may draw among the alternatives it
was offered — a weapon from pp. 12–13's lists, and now the service — because
the page gives no basis to prefer one. It may not throw dice: a die thrown by
a decider reaches no event. `varyOnly` wraps the roller so a type assertion
cannot recover the dice (#136).

The order inside a term is the exposition's, under E002: survival,
commission, promotion, skills, reenlistment, then aging (E006). The term's four
years are added before the reenlistment throw, so its fact carries the age the
term left him at. `serve` has no bound on terms, because a 12 exactly on
reenlistment recurs (E003); termination is a contract on the roller (#54).

## Dice, scripts, and why the note is the only way back

ADR 0001 withdrew replay. The `dice` stream is unseeded, and nothing brings a
character back except the generation record, which logs every die and every
choice. ADR 0006 narrowed that: the record is the note's, written as prose,
and a note is not read back by any build. There is no seed, interactive mode
cannot be resumed, a stopped session tells the player what it lost, and every
note ends by telling the referee to keep it whole.

A `chargen.Roller` is `Die` and `Among`. `twoDice` is the one place a 2D throw
is made of dice, `Roller.Die`'s doc is the one list of one-die throws, and
`TestEveryThrowIsTheNextTwoDice` holds that for every fixture the dice the
record logs, in order, are exactly the dice its script handed out. That is
what lets a walk be extracted from a note's record at all.

A **script** is a path's dice and draws in the order the procedure asks for
them, and `internal/scripted` plays one strictly both ways. A **fixture** is a
script with a checked-in golden note; a **walk** is the command tests' script,
extracted once from a record the tool wrote, with no golden. Consumption order
matters to every script: a change that reorders throws re-paths fixtures, and
a golden diff is sometimes the only symptom, so it is read before it is
regenerated. The Jamison reproduction of pp. 23–25 runs on the same scripted
roller, writing each narrated total as two dice, the first as high as the
total allows, and differs from p. 25 in exactly the benefit E015 accounts for.

## The note

`render` writes a `Character` through one `view`, built once by `viewOf`, so
the properties, summary, sheet and record cannot read the same character two
ways. The record's lines are made from `entry`, a flat projection of each
event built by one fold, `entryOf`, which the live view also uses. The live
view differs from the record in two places by design: a choice is echoed as
its number and answer, since the player can still see the question, and a step
is a plain numbered line, since a terminal prints markdown's marks (#422).

The note's shape is not frozen at `v1.0.0`; its headings and prose may keep
moving as the playtests ask (ADR 0006). What freezes is the command line (#126)
and the **property names and meanings** (#399), because a vault's queries stand
on them. `writeProperties` states the meanings in its doc: `age` is completed
years without crisis months, `terms` counts every term begun including the one
a death ended, `status` is `alive` or `dead`, a key with nothing to say is
absent but for `name`, `skills`, `credits` and `pension`, and every text value
is quoted so no later word can change a property's type.
`TestANoteOpensWithItsProperties` holds them, typed out.

The summary under the heading (#425) says only what the sheet below says, in
words a referee can read aloud. The sheet tells the service a term at a time
(#401) from the service history, and a characteristic change the floor or
ceiling held says the table's change and where it held (#409), because
`traveller.Change` keeps `Delta` beside `From` and `To`. The errata ids are on
the record and deliberately not the sheet. With `--readings` the note ends with
the full text of each reading its record names. The provenance lines name the
ruleset, the build where the command wrote it (ADR 0002; a test passes no
build, so a golden does not differ by machine), and who answered.

## The command

`cmd/ctchargen` is thin, and its seams are channels and refusals. The note goes
to stdout and the conversation to stderr, so `ctchargen new > kira.md` saves a
note. Every failure exits 1 (#121); misuse (`errUsage`) and a search that could
not finish (`errSearch`) differ in the message only. A stray positional word is
refused, because the flag package stops at it and leaves later flags unparsed.

`new -o` is refused before the first question if its folder is missing, if the
path exists without `--force`, or if it is a directory (#424): refused at the
end, a guided session lost every answer. `--force` stages the note beside the
path and renames it over only once it is whole (#191).

A batch writes only the living (#421): it passes over the dead and generates
another, rerolling nobody, so every note is one the dice made, and gives up
with `errSearch` after a hundred characters per member asked for — a bound
computed by division, since multiplication overflowed for a huge count (#423).
Into `-o` it writes only a folder that is new or empty but for hidden entries
(#426), checked once before anyone is generated, and never removes anything;
an `-o` ending in `.md` is refused (#416). Each note is named
`NN Service Rank UPP.md`, from its position padded to the batch's width and
`render.Title` (#400), and is headed by that title, since a batch names nobody
(#406). Without `-o` the notes stream to stdout as they are made (#44). The
closing tally goes to stderr, never into the last note.

The interactive `player` is a `Decider` that reads numbers and never invents an
answer: input that runs out is `errNoAnswer`. ADR 0003 decided its tests go
through the command, because the wiring is what a person notices breaking.
`cli_test.go` types out the flags, types and defaults and holds them one way: a
new flag breaks nobody, a changed or removed one does, and each removal before
`v1.0.0` (`render`, `--sheet`, `--history`, and `batch`'s `--auto`,
`--force`, `--survivors` and `--name`) is recorded beside the baseline as deliberate.

## What the gate holds

There is no CI (#149). The gate is a local `task`: tidiness, lint (every
`go vet` analyzer and the formatter), nilaway, govulncheck, the tests under
`-race`, and the non-Go files' format and links. No markdown file is parsed,
generated or held to the code (ADR 0004), and no data file is read at load (ADR
0005). `internal/coverage` is a test-only package that walks Go doc comments,
package docs and the fixture roster's comments for `Implements:` and `Holds:`
markers, and fails on a rule implemented and not held, held and not
implemented, cited to two pages, found elsewhere, or cited outside a closed
grammar: Books 1–3 pages with readings, a reading alone, an ADR by its file,
or one of eight promise labels. Since #391 those labels name a promise's topic
from a closed list in today's words — Authority, Choices, Command line,
Documents, Domain, Generation record, Sheet, Transcription — rather than the
section of the removed v1 contract it was first stated under.

Coverage is not measured (#198). The mutation rule stands in: a new invariant
is done only when `go run ./internal/mutate` shows a deliberate edit killed in
a scratch copy, reporting `NOT BUILT` rather than `KILLED` for one that does
not compile (#359). The #441 PRs list their mutations and what each failure
said in their commits.

## What it is shaped to accommodate, and what would need rethinking

**A new fact the sheet reads** is designed for and costs four places: a case of
`Fact` with a `String` that is the record's line, a `FactCases` method (which
stops `sheetFold` and `historyFold` compiling until each says what it does), a
`log.fact` at the outcome carrying the value the engine reached, and the
working-state change beside it if the engine reads it later. Put a rule in the
fact, not the fold.

**A new reading** needs a constant, an entry in `Errata`, a `Reading` value, a
stamping literal, a predicate written from its prose, a fixture that reaches
it, and a marker; the compiler or a test refuses most omissions.

**A correction to a transcribed cell** goes in both transcriptions from one
fresh visual reading of the page. If the page prints something the lift cannot
read, the lift gets a named decision, and a reading if the page does not settle
it. Editing the transcription toward what the lift accepts is the damage ADR
0005 exists to prevent.

**A new choice point** is a `Decider` method, and the compiler takes the
maintainer through the policy, `logging`, the player and the test deciders;
lint takes him to `choiceWords`. It does not bump `PolicyVersion`; a changed
answer does.

**A new property** is cheap before `v1.0.0` and a freeze after it; renaming one
after the freeze breaks every vault query that stands on it.

**Reading a note back** — a `replay`, or rendering an old note in a new layout
— would need a record a tool can read, which ADR 0006 gave up. The typed facts
are the shape such a record would have, but they never leave the process.

**Another edition, or Books 4+,** would mean founding a sibling: `Ruleset` is a
constant, the alphabets are Book 1's, the transcription's bounds are its six
services and four skill tables, and the authority model forbids opening those
books.

The damage a maintainer without the theory is most likely to do is quiet. He
might read a cell from extracted text, tidy `"Intel 8+"` into the domain's
spelling, type a hyphen for an em dash, write a predicate by looking at the
stamping site, or regenerate a golden without reading its diff. Since rc.4 he
might also log a sheet fact as prose through `outcomef`, change working state
without the fact that states it, compute a rule inside a fold, or keep a copy
of a folded fact on `Character` "for speed". Some of these the lift, the
compiler or a golden refuses; the rest leave the gate green on every path no
fixture walks.

## Uncertainties

- **Working state and the log are held equal only by the goldens.** I found no
  test that compares `run`'s fields with the folded `Character` at the end of a
  generation. By reading, every assignment to a working field is beside a
  `log.fact` stating its result, and the #441 commits show golden diffs killing
  fold mutations, but a path no fixture walks is covered by inspection alone. I
  can't tell whether a direct check was considered and declined.
- **`Errata` is kept beside the log, though it is derivable from it.** ADR 0006
  says the character's errata are "gathered from the ids its outcomes name",
  and each `OutcomeEvent` carries its `Errata`. The ADR classes the stamp as not
  a fact about the character, so I read keeping it as deliberate, but a
  hand-built `Character` in a render test can carry an `Errata` its events do
  not name, and nothing refuses that.
- **"Fact" has no glossary entry.** It is this release's organizing word, used
  in the ADR, the changelog and the code, and GLOSSARY.md does not define it or
  say how it differs from an outcome. The domain docs treat that as a signal
  for `/domain-modeling`, not a defect.
- **"Record" still has two senses in the code.** The glossary reserves it for
  the note's generation record; `Character`'s doc calls the character "the
  record a generation produces", and `Build`'s says what "wrote the record". I
  take these as older wording.
- **Whether the note could be read back is argued, not tested.** ADR 0006 says
  a replay is "not possible from the note as written", yet the command tests'
  walks are extracted by hand from a note's record, which carries every die in
  order. I read the ADR as saying no tool should parse prose a later build may
  reword, not that the information is missing.
- **Authority point 6 names two kinds of number, and the transcription holds a
  third**: amounts and targets the page states in prose that index nothing (the
  medical crisis's 8+ and its recovery, the mustering DMs, the education gate's
  8+). #43 named the gate as data without deciding where such numbers belong.
- **The withdrawn-reading form is tested only by a stand-in**, and the refusal
  to stamp a withdrawn reading holds only over the fixtures' records, not at
  generation time. ADR 0004 assigns it to the harness, so I infer that is
  deliberate.
- **Coverage of my reading.** I read every non-test file in `chargen`,
  `render`, `traveller`'s facts and events, and the command's `main`, `new` and
  `batch` in full; `rules`, the interactive player, `usage` and most tests only
  in part, and the `rules` claims here are carried from rc.3 after spot-checking
  their identifiers, since `rules` changed by eight lines since rc.3. I did not
  open the PDFs: every claim about what a page prints comes from the
  transcription, the code's quotations and the readings' prose.
