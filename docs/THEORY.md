# A theory of ctchargen

> Release snapshot for `v1.0.0-rc.6`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it
> (`docs/CODING_STANDARDS.md`, "A release snapshot is left alone between
> releases").

## What it is for

ctchargen makes a Classic Traveller character the way Book 1 pp. 4–25 says to,
and hands the referee a note that can be checked against the page. That second
half is the point. Any generator can print a UPP and a skill list. This one is
built so that every number on the sheet goes back to a printed page, a die
that was thrown, and a choice someone made, and so that the note says what
each of those was decided on: what earned a count of rolls, why the policy
gave the answer it gave, what a gain moved a value from. Where the book is
silent or contradicts itself, the number goes back to a decision recorded in
the source under its own name, and that decision is the maintainer's to read,
not the referee's (ADR 0007).

The world it models is one procedure. Six characteristics are rolled. There is
one enlistment attempt, with the draft behind it. Then come terms of service:
survival, commission, promotion, skills, reenlistment, aging. Then mustering
out, and last a title the final Social Standing may confer. The procedure has
throws (2D against a target, with modifiers), rolls (dice read off a table,
with nothing to meet), and points where the player chooses. Each is a noun in
the code: a `traveller.ServiceName`, a `traveller.Term`, a `ThrowEvent`, a
`RollEvent`, a `ChoicePoint` and a method on `chargen.Decider`. A place the
page is silent is a `rules.Reading`, and its id inside the tree is a
`traveller.Erratum`. `docs/GLOSSARY.md` keeps that vocabulary, and its "avoid"
lines matter, because some words the code uses carefully are ones a sibling
tool would use loosely: a reading is not an erratum, a walk is not a fixture,
a script is not a seed, the record is the note's generation record and not the
whole note, the service history is not the log, the profile line is not a
headline, and a fold only records what its case says.

The authority model outranks everything else. The © 1977 text governs, and
memory of later editions does not. A reading decided here governs only where
the page is silent. The errata of the implementation deleted at `41a213a` are
a last resort, and even then only a pointer to a page that has to be reread. A
maintainer who leaves this model out will "fix" something into the 1981 rules;
the best-known trap is that survival failure is death, with no injury option
(p. 5).

`v1.0.0` freezes two things: the command line a referee types (its flags,
types, defaults and exit status, #126) and the note's property names and
meanings, which vault queries stand on (#399). rc.5 gave both the shape they
freeze in. rc.6 answers a referee evaluation of rc.5 (#549, #552, #553, and
the issues filed from them) and moves neither frozen layer: no property is
added, renamed or given a new meaning, and the one change a referee types is a
strategy's value, `--muster spartan` renamed `arsenal` (#570). What it changes
is the note's body, the help, and what the policy answers. Three ideas
organize those changes, and the rest of this document keeps returning to them:

- **The note says what it knows, and what each answer was decided on.** A
  count of rolls names what earned it; a policy's answer says it was drawn, or
  what a skills table choice weighed; a gain on the service record is a value
  before and after; a term a 12 required, a crisis's months, a retirement the
  throw refused and expertise taken at mustering out reach the service record.
  Each was already in the record or in the engine's hands at the moment it
  decided, and is now said where a referee reads it (#561, #562, #564, #565).
  The explanations are derived from what the decision was made on, never from
  how things turned out.
- **What the page gives no basis for, the policy draws; what the page prices
  nothing against, every strategy takes.** Before rc.6 a weapon benefit was
  the first name printed, a Dagger or a Body Pistol every time, and `oneterm`
  declined the draft, which left a character who had not served the one term
  it names. Now the three answers the page leaves open (the service, a combat
  result's weapon, a benefit's weapon) are drawn under every strategy, and
  every strategy says yes to the draft, the commission, the promotion, both
  mustering-out DMs and the title (#566, #570). A "no" at any of those is a
  person's answer, reached only by a guided run.
- **The sheet leads with what a referee reads first.** The profile line opens
  the sheet, the summary follows it and repeats only a death, and each step of
  the record is a heading a viewer's outline can show (#557, #577).

## The generation record is the one store

### What the character is

`chargen.Character` keeps five fields: `Inputs` and `Policy`, which are what
the generation was given; `Events`, the record; `Errata`, the stamp of every
reading that governed it; and `Build`, which names what wrote it. Everything
the sheet says about the character is a method that folds the record:
`Profile`, `Skills`, `Benefits`, `Pension`, `Nobility`, `Enlistment`, `Rank`,
`RankName`, `Age` and `Departure` (all through one `sheetFold`), `Terms` (the
number of the last term a step opened), `History` (through `historyFold`),
`Attempted` (the first `EnlistedFact`) and `ServedIn` (a fold of the folded
`Enlistment`). Character's doc says it outright: nothing it says of the
character is kept on it beside its record (#441).

`Errata` is the one field that is neither an input nor derivable from the
record. The engine passes an erratum to `record.fact` or `record.outcomef` at
the stamping site, the record keeps a set beside its events, and
`record.stamped` writes it to the character in id order at the end. No note
names an erratum, `render` may not import `rules` (a depguard rule), and the
readers of `Character.Errata` are the stamping harness and a handful of tests
that look for one reading on one path. The stamp is the authority's record of
itself, kept on the character so the gate can hold it.

### How an outcome carries a fact

An `OutcomeEvent` has prose (`Description`) and an optional `Fact`: a sealed
sum in `traveller` with a cases interface, `FactCases`, of twenty-six methods.
The engine records a fact through `record.fact`, which sets the outcome's
`Description` to `fact.String()`. So a fact's prose is its own `String`, and
the line a referee reads and the value the sheet folds cannot say different
things. An outcome that changes nothing a sheet reads ("survived term 2", "not
promoted this term", "a second scout ship, which is lost", a dash row of
Table 1, an aging throw that holds, and rc.6's two lines about the offer, the
Table 2 cap and the fourth skills table) goes through `record.outcomef` with
prose and no fact.

The rule that keeps the folds simple is **a fact carries the value the engine
reached, and the fold applies no rule**. A skill's fact carries the level
reached, not "+1"; a cash fact carries the running total beside the roll; a
repeat Free Trader's fact carries the ship as it stands after ageing and
paying down; a characteristic change carries `From`, `To` and the table's
`Delta`; every fact that moves the age carries the age it left. So the fold
for skills is "the last level per name", for cash "the last total", for age
"the last age", and no fold re-derives p. 12's accumulation, p. 22's repeats
or E004's fatal four years. Where a rule would otherwise have to live in the
fold, the fact's type carries it: a Free Trader received again is its own
case, `FreeTraderAgainFact`, and `sheetFold.Drafted` converts
`traveller.DraftedFact` to `traveller.Drafted` by a Go conversion, so a field
added to either type stops the fold compiling.

Some facts are stated where a fold could have computed them, deliberately.
`StartedFact` writes the rolled UPP and the starting age as a line of the
record; the profile is read from the six `RolledFact`s and the fact
contributes only the age. `RetiredFact` carries `Denied`, because p. 21 makes
any departure at term 5 or later retirement, and the record still says whether
the throw refused the character.

### The service history: a second fold, which now remembers

The service history is a second fold over the same facts. Each `AppliedFact`
and `WeaponPickedFact` carries a `Source` (on entering, at a rank, in a term,
at mustering out), and `historyFold.gained` files a gain by it. A term is the
`StepEvent` whose `Opens` field names it, so a term a survival throw ended is
still a term served, with nothing in it. rc.6 widens what the fold files
(#561): a `RetainedFact` marks the term a 12 required, a `RecoveredFact`
becomes a `Recovery` with the value it came back to and its months, a
`RetiredFact`'s `Denied` becomes `ServiceHistory.RetiredDenied`, and an
`ExpertiseFact` is filed as a mustering-out gain. That last is filed by the
fold's own knowledge rather than a `Source`, since `ExpertiseFact` carries
none and only mustering out takes expertise.

rc.6 also gives the fold state of its own (#562). The sheet tells each gain as
a value before and after, `Administration 0→2`, so a gain is filed as a
`Moved`: the gain and the value it moved from. A characteristic's `Change`
carries its own `From`, but a skill's fact carries only the level it reached,
so `historyFold` keeps `levels`, each skill at the level the last fact naming
it reached, and `fromFold` reads the prior level off it before storing the new
one. That is reading the record, not applying p. 12, and it keeps the rule
above. It does lay a promise on every case that carries a skill: each must go
through `gained`, or the next gain of that skill reads a stale "from".
`gained`'s doc states the promise, the three cases that carry a skill
(`Applied`, `WeaponPicked` and `Expertise`) keep it, and only review would
catch a fourth that did not.

### Working state alongside the record

The engine cannot read its own record as it runs: every DM reads the current
characteristics, the commission and promotion throws read the rank, mustering
out reads the terms and rank, the repeat rules read the benefits so far, the
pension reads the term count, and the end reads death off the departure. So
`chargen.run` keeps working state privately: `profile`, `skills`,
`rolledSocial`, `benefits`, `enlistment`, `terms`, `rank`, `atAge` and
`departure`, beside three fields that never reach a sheet: `eligible`, the
term's skills rolls by what earned them; `trained`, the results taken off each
table, which a decider is shown and the record now keeps on the choice; and,
new in rc.6, `fourthTableOpenedBy`, the event that took Education to the
fourth skills table's threshold, held until the next designation says so. Each
change to the sheet's working state is recorded as a fact carrying the value
it left (or, for terms, as the step that opens one), and the finished
`Character` is assembled from `Inputs`, `Policy`, the events and the stamp.
The working state dies with the run.

That is a second statement of every fact, kept for the length of one
generation, and the theory's honesty depends on naming it. The two cannot
disagree on any path a golden walks. Nothing compares the two directly, so a
path no fixture walks is held only by reading the engine: each assignment to a
working field sits next to the `record.fact` that states its result.

### What it costs, and why that cost stays

- **The sum is wide.** `FactCases` has twenty-six methods, one per kind, and
  every fold implements all of them; `historyFold` is mostly one-line
  `return nil` methods. #499 asked whether the visitor was accidental and was
  closed as not planned: the width is the closure, each `return nil` is a fold
  saying "not mine" in so many words, and a new kind of fact stops both folds
  compiling until each says what it does with it.
- **Every question folds every event.** `render.viewOf` asks the character for
  about a dozen facts, and each `sheetFold` method walks the whole record to
  keep one field. A note costs a dozen passes over a few hundred events, which
  is nothing beside writing a file, but a hot loop over `Character` methods
  would pay a walk per call.
- **A sheet fact must come through `record.fact`.** An outcome written through
  `outcomef` reaches the record and not the sheet. A benefit recorded as prose
  is a note whose sheet omits it, and only a golden that walks the path
  notices.
- **The record's wording is bound to the domain types.** A fact's `String` is
  the record's line, in `traveller`, so rewording a line is a change to the
  domain package that moves every golden printing it. The engine's step names
  are bound the same way to the stamping predicates, which match on them.

### What it makes possible, and what it makes hard

It makes a character constructible from its record's own terms: the render
tests build a `Character` as nothing but events and assert on the note, so a
test cannot set a sheet the record contradicts. rc.6's service record changes
were made inside that shape: every new clause on the sheet is a fact the
record already carried, filed by the history fold and worded by `render`.

It makes a rule that reads history expensive to put in the wrong place. "Was
the character ever commissioned?" belongs either in working state, if a throw
needs it, or in a fold over facts that already carry the answer; a rule
computed inside a fold is what #441 removed.

It leaves the note unreadable by a tool. ADR 0006 is explicit: a `replay`
would need a record a tool can read, and the note's record is prose. The typed
facts exist only in the process that generated them.

## The page, typed as printed, and read twice

The reprints' embedded font substitutes glyphs, and the substitutions look
like data: a dash cell comes out as the digit 4, a minus as a 3, "Travellers'
Aid" as `Travellers9`, and p. 23's box shows `Rifl3-1` even in the rendered
image. That one fact explains much of the repository's shape.

Since ADR 0005 the first transcription is Go:
`internal/rules/transcription.go` holds every table as a literal of the page's
text, with its glyphs, abbreviations, notation and dashes. It holds
`"Intel 8+"`, not `"Intelligence 8+"`; the em dash wherever a service has no
commission, rank or benefit; `"DM of +1 if"` as a row heading; the Aging
Table's header rows and dotted-leader spans; and `"Rifl3-1"` as the reprint
shows it. **Structure comes from the domain and cells from the page**: which
table a grid is and which service a column belongs to are positions keyed by
`traveller`'s types, so a table the type does not name, or a cell too many,
does not compile. A short Go array is padded with `""` in silence, so
`refuseEmpty` refuses an empty cell on the ground that the page prints a dash
where it means nothing. The service tables keep the page's rows as rows, the
Table of Ranks included (#354), so the two transcriptions read every grid in
opposite directions.

**The lift is kept on purpose.** ADR 0005 rejected typing the tables as domain
values directly: a broken glyph would then fail to compile, but every
normalization and every dash would be decided in the transcriber's head, which
is a reading applied silently. The lift is where those judgements are written
down, and its malformed-input suite is what shows it refusing text the page
could not have printed. #501 argued the suite defends only against text typed
into Go by hand, and was closed as not planned for exactly this reason. A
printed name is checked against `traveller`'s closed alphabets; what the
domain does not spell lives in `rules` (`abbreviatedCharacteristics`,
`boxServices`, E012's spellings such as `Rifl3` to `Rifle`, and the fixed text
around values). `traveller.Sign` is the one place an en dash is recognised and
refused. `rules.Load` lifts once through `sync.OnceValues` and keeps its
error, since a literal can still hold text the lift refuses. The lift refuses
shape; the few value checks it makes are sanity bounds or cross-checks of the
page against itself. Whether a value is the one the page prints is the second
transcription's job, in the `rules` tests, written a service at a time in
normalized words.

The transcription reaches the note's text in two places. A throw's record line
names the characteristic DMs it earned (#487): `rules.Throw.Met` returns each
prerequisite met, `Modifier` sums the same ones, and `ThrowEvent.DMFrom` and
`Odds.From` carry them, so a mutated threshold moves the record. What reaches
the note is the lifted value in the domain's words, "Intelligence 6+" where
p. 10 prints "Intel 6+" (#539). And since rc.6 the mustering-out step's count
of rolls is `rules.Muster.Rolls` returned as a `traveller.BenefitRolls`, the
terms' share and the rank's kept apart, so the step says "7 for the terms, 2
for rank 5" from the same value the procedure spends rather than by working it
out again.

CLAUDE.md's Authority rule 6 sorts numbers: one that indexes a printed table
is in the transcription; a bare procedural bound is a constant beside the code
that applies it, with its cite (`maxRollsOnTableTwo`,
`minRankForTable1Modifier`, `lastVoluntaryTerm`, `freeTraderPaymentYears`). A
die's six faces are the one number said twice on purpose (#534): `dice.Faces`
for the dice and `rules.Faces` for the tables a die indexes, because `dice`
depends on nothing and `rules` may not import it, so `chargen`, which imports
both, holds them equal in a test. The cost ADR 0005 accepted is that linters
read the page's text: `goconst` is excluded for the transcription and the
`rules` tests, `mnd` for the transcription, and `task fix` over this file is
as unsafe as over a verbatim quotation.

## Readings: the authority's record, not the referee's

A reading is how the authority model records a decision wherever the page is
silent or contradicts itself (Authority rule 5), and the gate holds each one.
Since ADR 0007 (rc.5) it is not shown to the referee: no note names an
erratum, and `render`'s package doc carries that promise. The rc.4 playtest
had found E002 stamped on nearly every skill line, and E006 and E007 on every
aging throw, with nothing telling the referee what to do with them (#460,
#488).

A reading has three expressions, kept apart because the check between them
works only while they stay independent:

- The **prose** is a `rules.Reading` value in `internal/rules/readings.go`,
  with the page's emphasis as inline markdown. `rules.Readings()` returns an
  array sized by its own literal, so a reading missing or one too many does
  not compile, and `TestErrataListsEveryConstant` parses `traveller`'s source
  and requires `Errata` to be exactly its `Erratum` constants, in order. No
  production code calls `Readings()`; its readers are the tests and a person
  reading the source.
- The **stamping site** is in the engine: thirteen sites across `enlist.go`,
  `serve.go`, `aging.go`, `rank.go`, `reenlist.go` and `muster.go`, each
  passed at the outcome it governs. The engine never reads a reading's prose,
  so a reading's wording can change without changing a character.
- The **predicate** lives in `internal/chargen/stamping_test.go`, one per
  reading, written by hand from the stamped-on prose and never from the
  stamping site (#66, #77), and checked both ways, record by record: an
  erratum stamped where its condition fails is as much a defect as one
  missing. A failure quotes the prose from `rules.Readings()`. The predicates
  read the roster's characters and the guided walks' beside them, so E001's
  declined draft, which no auto run reaches since #566, is still judged. E012
  and E015 are stamped on no record by design, which `namesNoRecord` pins; the
  `parked` mechanism measures without judging a reading whose prose and code
  disagree, and none is parked.

What ADR 0007 took away is ADR 0004's promise that an erratum names one
decision for good. No note carries an id, so nothing outside the tree is
stamped with a decision that might change. A reading whose decision no longer
holds is changed or removed like any other code, and its predicate is held to
what it says then. The ADR names its price: a referee cannot tell from a note
which reading governed an outcome. Where a reading changes something the
referee acts on, the tool says so in plain words without naming it; the guided
session's draft question says that declining ends generation with a character
who never served, which is E001's consequence stated as a consequence (#483).

The discipline of the predicates has a second user in rc.6.
`TestTheRecordSaysWhichAnswersWereDrawn` holds each choice's `Drawn` against
`policyDraws`, which is typed from `Policy`'s doc ("the service, the weapon a
combat result names, and the weapon a benefit names, where expertise is not
taken") and not from the code that records a draw. A condition written from
the prose a second time, rather than read off the mechanism, is how this tree
checks a claim it makes about itself.

## The domain is sums, so the compiler counts the cases

`traveller` imports nothing in this module, and nothing in it rolls, reads a
table or writes markdown. Its rule for what becomes a type is that types carry
identity, never rule invariants. Wherever the book says "exactly one of", it
has a sealed interface with a `Fold` over a cases interface: `Enlistment`,
`TableResult`, `BenefitRow`, `Departure`, `WeaponBenefit`, `Event`, `Fact` and
`Gain`. Go does not check a type switch for exhaustiveness; `gochecksumtype`
could, but only for a switch over an annotated interface and only when the
linter runs, while a fold's exhaustiveness is a method set the compiler checks
on every build (#498). The plain enums are weaker: the exhaustive linter
catches a missing switch case, and only under `task`. `render.choiceWords` and
`entryKind` are switches for that reason, so a new choice point is a lint
failure until it has the page's words, and so are the strategies' new `does`
methods, so a new strategy value is a lint failure until the help can say what
it does. Every type that reads a sum into a value is named with the suffix
`Fold`, and one that reads it into prose with `WordsFold` (#512); a type that
handles the cases to act on them, and can fail, is named for what it does.

A sum is only as right as its list of cases, and the list is meant to be the
page's. Some choices make sense only against a sibling design: the civilian
who declined the draft is a case of `Enlistment`, not a nil service with a
flag; "died" is two `Departure` cases because they carry different fields;
`Fatal` lives in `traveller` because it was once stated twice (#100); `Age`
carries months only because a survived crisis adds 1D of them; `RollEvent` is
not a `ThrowEvent` with an empty target, so a roll is never written as having
succeeded (#50); and a `Target`'s zero value is "0 exactly", which no throw
can meet, so an unset target fails closed. `render`'s flat `entry` does read a
roll as a throw with an empty target, and `throwLine` prints "made" or
"missed" only where there is a target: the distinction #50 drew is kept in the
domain and spent, deliberately, at the last step before text.

rc.6's new data are not sums. `ChoiceEvent.Drawn` is a bool and
`ChoiceEvent.Taken` a slice that is nil except on a skills table choice;
`StepEvent.Within` is a `Term` that is zero except on a term's phases. Each is
an optional field on an existing case rather than a new case, so adding it
stopped nothing compiling, and each is held by a test of what it records
rather than by the compiler. `Earned` and `BenefitRolls` are plain counts that
reach the record only as the words of a step's name, which the explanation
tests read back with a regular expression.

## How the tree treats what cannot happen

A branch the module's own types make impossible goes. Nothing outside the
module can build a `Character` (#199), and generation always records an
enlistment and gives every served character a departure and an enlistment
throw. So `servedIn` panics on a nil `Enlistment` (#377),
`render.foldEnlistment` does too (#393), `viewOf` folds a served character's
departure without a guard (#445), and `projectAttempt` panics on a served
character with no enlistment throw (#454): a generation defect crashes rather
than printing a civilian or a character who never left the service. A fold
whose cases only assign drops the error with a comment saying so, and
`Character.fold`'s doc makes the same demand of every `FactCases` handed to
it.

rc.6 adds one more crash of the same kind, and it reaches further.
`render.leastTrained` words the policy's reason for a skills table from the
counts the choice carries, and panics if the policy's table was not among the
least trained, since `Policy.SkillTable` designates among those only. That is
`Policy`'s rule stated a second time, in the renderer, held equal to the first
by a panic on any note a golden or a render test writes. It is consistent with
the rule above (a wrong reason is never printed) but it is the one place
`render` knows how a strategy decides.

The rule underneath is that the tool never writes a wrong note by its own
logic: a defect may crash, and an unwritable note is refused, but neither is
absorbed into a note that looks complete. What lies outside the module is the
file system, and the command reaches into it no further than a plain write:
#520 replaced the staged write and rename with a write in place, as the
shell's `>` does, and two refusals that second-guessed the open were removed
on the ground that the operating system reports its own errors.

## The engine asks, and the record says who answered and on what

`chargen` walks the procedure and decides nothing on its own behalf. Every
point where the book lets a player choose is a method on `Decider`, twelve of
them, each typed to its own alphabet, which is what closes the set.
`TestChoicePointsMatchTheDecider` holds the `ChoicePoint` enum to the
interface's methods by reflection.

Three kinds of decider exist (the auto run's `Policy`, the command's
interactive `player`, and the tests' own deciders, such as `refusing`), and
the engine cannot tell them apart. Each is reached through `logging`, the
decorator `generate` wraps around whatever it is handed, which keeps promises
no single decider has to: a question with fewer than two legal answers is
refused before it is asked (`asked`); each slice a decider is handed is its
own copy (#55, #137, #230); and an answer outside the offer is refused
(`errNotOffered`). That last check compares rendered names, which is sound
only while `String()` is injective on every alphabet, as
`TestNoTwoValuesOfAnAlphabetShareASpelling` holds (#48).

The engine shows a decider the stakes as well as the offer. The commission and
promotion questions are handed the throw a "yes" leads to, the reenlistment
question the survival throw the next term opens with, the service offer each
service's enlistment throw, each as a `traveller.Odds` with its target, DM and
the prerequisites that earned it, and the skills question each table's six
results, how many results the character has already had off it, and which roll
of how many this is and what earned them. The policy ignores the throws, since
no strategy prices a failed one, and the player prints them (#518). The
reenlistment question's survival odds are computed before the term's aging
round, which E006 places after the reenlistment throw, so they can overstate
the next term's DM by one aging round; `Decider.ReenlistIntent`'s doc states
which DM it is.

### What the record keeps about an answer

Who answered is fixed when the run is entered. `chargen.Auto` hands `generate`
a pointer to its policy and `chargen.Guided` hands it nil, and that pointer,
kept as `Character.Policy`, is the whole of what marks an auto run; each
choice's line says "policy" or "player" from it, through `render`'s
`answerer`. An auto run is `new --auto` or any batch, which declares no
`--auto` because it has nobody to ask (#336).

rc.6 makes the record say what an answer was decided on (#564), and the design
keeps the explanation to the decision's own inputs. Two facts are recorded on
the `ChoiceEvent`, both by `logging` and neither by the decider:

- **`Drawn`** says the decider drew the answer. `logging` hands a decider a
  `watchedVary` around the `Vary` the engine passed, which forwards every draw
  and remembers that one was made. So "policy drew" is written because a draw
  happened, not because of which point asked or who answered: a policy taking
  expertise in a weapon it has received draws nothing, and a player never
  draws. `watchedVary` holds the engine's `Vary`, never the `Roller`, so it
  narrows as `varyOnly` does (#136).
- **`Taken`** is, for a skills table choice, how many results the character
  had had off each table offered, named before the decider is asked, as the
  offer is, so what is recorded is what it was shown.

`render` words them. A drawn answer reads "drew". A policy's skills table
reads "least trained on (1, against 2 and 2)", or, where several tables tie,
"tied least trained on (0) with three others, the advanced strategy's first".
That clause is the one place a note names a strategy. rc.5 had removed the
footer that named the policy's strategies, on the ground that nothing else in
the note said them and no referee needs them to read a character (ADR 0006,
amended); rc.6 names one strategy again, at the one point where the strategy,
and not the counts, decided the answer, and `DefaultPolicy`'s doc records the
exception. The name comes from `Character.Policy.Skills` at render time, not
from the event. The career and muster strategies are still named nowhere in a
note, and once a run is over nothing outside the tests reads them off
`Character.Policy`.

Two changes to the offer are explained by lines of their own rather than on a
choice, because nothing was asked. When the third roll on Table 2 leaves rolls
to take, `musterOut` records that the rest must be on Table 1; when an
`Alteration` takes Education to the fourth table's threshold, the next
designation is preceded by a line saying so, pointing back at the change.
Neither is a `ChoiceEvent`, since a question with one answer is not put
(`asked`), and neither carries a fact.

### The policy

The policy's documentation is its method doc comments, and `DefaultPolicy` is
the one place its defaults are written; the command's flags read them (#367).
Its shape since rc.6:

- **Three answers draw, under every strategy**, uniformly, with the run's own
  dice: the service (#420, after a referee's 120-member batch came out 109
  Army under a likeliest-enlistment ranking), a combat result's weapon (#34),
  and a benefit's weapon (#570). `MusterWeapon` gained a `Vary` for that.
  `arsenal` draws among the weapons of the category not yet received; `cash`
  and `goods` draw a first weapon and take expertise in the first received on
  a repeat, which draws nothing.
- **Six answers are yes under every strategy**: the draft, a commission, a
  promotion, both mustering-out DMs and the title. Each method's doc gives the
  reason, which is the same each time: the page prices nothing against the
  yes. `oneterm` had declined the draft and so never served the term it names
  (#566); `spartan` had declined the DMs. A civilian now comes only from a
  guided run, where a person answers no, and the roster's civilian fixture
  went with the change: E001's declined draft and the declined Table 1
  modifier are held by two guided walks (`declinedTheDraft`,
  `declinedTheTableOneModifier`).
- **The strategies differ only where the page gives a basis**:
  `ReenlistIntent` (career), `SkillTable` (skills, among the tables least
  trained on, #34) and `MusterTable` (muster). `prefer` takes the first of a
  ranking that is on offer. Every ranking lists every alternative of its
  alphabet, so the ranking alone decides, and a ranking that holds nothing on
  offer is a defect `prefer` panics on rather than answering in book order
  (#586).

The help says what each strategy does (#581), and the words for each value
live in `chargen` beside the method they summarize: `CareerHelp`, `SkillsHelp`
and `MusterHelp` return a `Described` per value, from a `does` method per
alphabet, and `CareerHelp`'s doc says a line that claims more than its method
does is a finding. The paragraph about what every strategy shares,
`strategiesShare`, lives in the command's `usage.go` instead, so it is the one
statement of the policy's behaviour not beside the code it describes. The
README's strategy table is gone; the help is where the values, defaults and
behaviour are said.

The narrowest seam is still `Vary`. A decider may draw among the alternatives
it was offered because the page gives no basis to prefer one. It may not throw
dice: a die thrown by a decider reaches no event. `varyOnly` wraps the roller
so a type assertion cannot recover the dice (#136).

### The term

The order inside a term is the exposition's, under E002: survival, commission,
promotion, skills, reenlistment, then aging (E006). The term's four years are
added before the reenlistment throw, so its fact carries the age the term
left. `serve` has no bound on terms, because a 12 exactly on reenlistment
recurs (E003); termination is a contract on the roller (#54). The term's
skills step and its aging step are recorded through `record.phase`, which sets
`StepEvent.Within` to the term, so the note can set them under it (#577);
every other step is outside any term or opens one.

## Dice, scripts, and why the note is the only way back

ADR 0001 withdrew replay. The `dice` stream is unseeded, and nothing brings a
character back except the generation record, which logs every die and every
choice. ADR 0006 narrowed that: the record is the note's, written as prose,
and no build reads a note back. There is no seed, interactive mode cannot be
resumed, and a stopped session says what was answered and that nothing was
written.

A `chargen.Roller` is `Die` and `Among`. `twoDice` is the one place a 2D throw
is made of dice, `Roller.Die`'s doc is the one list of one-die throws, and
`TestEveryThrowIsTheNextTwoDice` holds, for every fixture, that the dice the
record logs, in order, are exactly the dice its script handed out. That is
what lets a walk be extracted from a note's record at all.

The tests walk the procedure three ways. A **script** is a path's dice and
draws in the order the procedure asks for them, and `internal/scripted` plays
one strictly both ways. A **fixture** is an auto run's script with a
checked-in golden note, thirty of them in the roster. A **walk** is a script
with no golden, extracted once from a record the tool wrote: the command tests
take walks, and since rc.6 the engine's tests take guided walks, the paths
only a person's "no" reaches. The third way is new in rc.6 and has no glossary
word: `explained_test.go` generates four hundred characters from seeded PCG
rollers, because the path it needs, Education reaching 8 between two
designations, is one no fixture walks, and it fails if none of the four
hundred reaches it.

Consumption order matters to every script: a change that reorders throws
re-paths fixtures, and a golden diff is sometimes the only symptom, so it is
read before it is regenerated (`-update`, #506), and a failing golden prints
the hunk that changed (#505). The goldens of `render.Note` live in `chargen`'s
tests because they run end to end over the fixture roster, which is why #511
was closed as not planned. The Jamison reproduction of pp. 23–25 runs on the
same scripted roller, writing each narrated total as two dice, the first as
high as the total allows, and differs from p. 25 in exactly the benefit E015
accounts for; it draws nothing, which `TestTheRecordSaysWhichAnswersWereDrawn`
also checks.

## The note

`render` writes a `Character` through one `view`, built once by `viewOf`, so
the properties, the sheet and the record cannot read the same character two
ways. The record's lines are made from `entry`, a flat projection of each
event built by one fold, `entryOf`, which the live view also uses. The live
view differs from the record in two places by design: a choice is echoed as
its number and answer, since the player can still see the question, and a step
is a plain numbered line, since a terminal prints markdown's marks (#422).

The note has two layers with two lifetimes. The **properties** freeze at
`v1.0.0`, in name and meaning, and rc.6 leaves them as rc.5 shaped them
(#474). Every property is on every note: text with no value is an empty key
and a list with none is `[]`, so a query never meets a missing key. `skills`
is names alone, one exact item each; `possessions` is what mustering out gave
besides cash, one item each, so two Low Passages are two items even where the
sheet now writes `Low Passage ×2` (#560); `ctchargen` is the build's version.
Every text value is quoted, so no later word can change a property's type. The
meanings are stated once in `writeProperties`' doc and once, as a typed table,
in the README, and `TestANoteOpensWithItsProperties` holds the shape typed
out.

The **body** stays free, and rc.6 rebuilt most of it:

- **The sheet opens with the profile line** (#557): the UPP, the age, the
  service and rank with how it was entered, `(enlisted)` or
  `(drafted after the Marines refused enlistment)`, the terms, any title
  assumed, and how the service ended. Who named the service attempted is no
  longer said there; it is the record's choice line (#583). `entered` reads
  the attempted service off `Attempted`, the first `EnlistedFact`.
- **The summary follows it** and repeats only a death, which it opens with
  ("Killed by the survival throw in term 3, aged 30."). Its skills are "Skills
  include" the three highest at any level, ties in the sheet's order (#558,
  #559), and its assets add retirement pay after the cash, in the one wording
  `retirementPay` gives the summary and the benefits both.
- **The "Benefits" heading** replaces "Possessions", with one line per class
  of passage and a count (#560).
- **The service record tells values before and after** (#562): `render.told`
  merges the moves in one list of gains into one entry per skill or
  characteristic, from the value before the first to the value after the last,
  with "(held at N)" only where a floor or a ceiling stopped a change short.
  That replaces the table's change that #409 put there; the record still says
  each change as the table gave it. It also tells a term a 12 required, a
  crisis recovery's value and months, a retirement the throw refused, and
  expertise taken at mustering out (#561). Where a crisis kills a character
  the throw has already retired, the death ends the service record and the
  retirement it overtook is not told.
- **Each step of the record is a heading** (#577): a step outside every term,
  or the step that opens one, at the third level, and a term's skills and
  aging at the fourth. The heading carries the step's number, so "from step N"
  still finds it, and the record opens by saying that a line's "(from step N)"
  names the step it came from, most often the roll it was read off, and not
  every cause behind it. Between two headings the lines are one ordered list,
  numbered from the line after the heading, so a Markdown viewer's numbering
  is the line's own; `TestNoViewerRenumbersTheGenerationRecord` holds that
  over every golden.
- **Counts say what earned them** (#565): the skills step reads
  `skills and training, 2 rolls: 1 for the term, 1 for the commission`, the
  mustering-out step `mustering out, 9 rolls: 7 for the terms, 2 for rank 5`.
  `traveller.Earned.String` and `BenefitRolls.String` are the one wording, and
  the guided session's skills question uses the same `Earned.String`.

Two promises cover everything the package writes and so sit on its package
doc: a note names no erratum, and no line names the character by a pronoun,
because a character has no sex in these rules (p. 8, #459).
`TestNoNoteUsesAPronoun` reads every golden for one. The note ends with its
generation record; nothing follows it.

A note is Markdown the formatter keeps as written. Since #577 each step's
lines are a list of their own, and `bunfig.toml` ignores by name only the ten
goldens where a step's numbered lines cross from 99 to 100, where the
formatter aligns the list by where it starts (#542).

## The command

`cmd/ctchargen` is thin, and its seams are channels and refusals. The note
goes to stdout and the conversation to stderr, so `ctchargen new > kira.md`
saves a note. Every failure exits 1 (#121). `errUsage` marks only a mistake in
the command line: a file in the way, a missing or non-empty folder and a build
with nothing embedded are found at run time and carry no "usage" label (#472,
#508). `errSearch` marks a batch that could not find enough living. A flag is
written in errors as the help writes it (`--count`, `-o`), where the flag
package writes one dash, and a path appears once in a message.

The refusals follow #466's rule: where an input can be read two ways, refuse
it now, because a refusal can be given a meaning after `v1.0.0` as an addition
while a meaning once accepted cannot be taken away without a break. `strictly`
wraps every flag a set declares, so a flag given twice or given an empty value
is refused by name. `--count` is a `decimalCount`: `010` is ten, and `0x10`,
`1_000` and `+3` are refused. `-o -` is standard output for both commands, as
no `-o` is. `--force` with no `-o` file is refused. A strategy flag without
`--auto` is refused, because the player would answer every choice it names. A
stray positional word is refused, because the flag package stops at it and
leaves later flags unparsed. A batch `-o` ending in `.md` is refused (#416).

The help is where a referee learns the tool. Each generating command's
`--help` prints its flags (with kinds read from the value, since the strict
wrapper hides the kind the flag package would print, #517), then, since rc.6,
what `--service` comes to on that command, what every strategy shares, each
strategy flag's values with a line on what each does, and what a death comes
to (#563). The per-command parts are a `commandHelp` value each; `new`'s says
the player picks the service or `--auto` draws it, and that `terms` counts the
term a death ends, `batch`'s that each member's service is drawn and the dead
are replaced.

`new -o` is checked before the first question by `refusedBefore`: its folder
must exist, and the path must not exist without `--force` or be a directory.
Refused only at the end, a guided session lost every answer (#424). Without
`--force` the note is opened create-exclusive, and a write that fails removes
the note this run made (#468). With `--force` the file is truncated and
written in place, as `>` writes (#469), so a symlink is followed and the file
keeps its mode and hard links; the trade is that a `--force` write that fails
partway has lost the old note.

A batch writes only the living (#421): it passes over the dead and generates
another, rerolling nobody, and gives up with `errSearch` after a hundred
characters per member asked for, a bound computed by division because
multiplication overflowed for a huge count (#423). Into `-o` it writes only a
folder that is new or empty but for hidden entries (#426), checked once before
anyone is generated, and makes that folder with `Mkdir`, so only inside one
that exists (#470). Each note is named `NN Service Rank UPP.md`, from its
position padded to the batch's width and `render.Title` (#400), and is headed
by that title, since a batch names nobody (#406). Since #566 a batch member is
never a civilian (the README still shows one, #588). Without `-o` the notes
stream to stdout as they are made (#44). The closing tally goes to stderr,
never into the last note.

The interactive `player` is a `Decider` that reads numbers and never invents
an answer: input that runs out is `errNoAnswer`. It is handed a `Vary` at
three points and ignores it, since a person chooses. ADR 0003 decided its
tests go through the command, because the wiring is what a person notices
breaking. `cli_test.go` types out the flags, types, defaults and the values
each strategy and service flag takes, and holds them one way: a new flag or
value breaks nobody, a changed or removed one does, and each removal before
`v1.0.0` is recorded beside the baseline as deliberate. The values joined it
in #591, after `spartan` became `arsenal` with nothing in the gate noticing.

## What the gate holds

There is no CI (#149). The gate is a local `task`: tidiness, lint (config
verify first, then golangci-lint with every `go vet` analyzer and the
formatter), nilaway, govulncheck, the tests under `-race`, and the non-Go
files' format and offline links. Since #545 `task test` also fails when the
tests leave a new untracked file in the tree, comparing
`git ls-files --others` before and after; `docs/CODING_STANDARDS.md` now says
a test writes only into its own temporary folder and matches an expected line
whole. `task setup` installs the toolchain from the repository alone: the
Brewfile lists every Homebrew tool the gate runs and the task adds NilAway
with `go install` (#492). The toolchain is unpinned on purpose, so a red gate
on untouched code is a tool moving, to be answered rather than pinned away
(#493). No markdown file is parsed, generated or held to the code (ADR 0004),
and no data file is read at load (ADR 0005).

`internal/coverage` is a test-only package that walks Go doc comments, package
docs and the fixture roster's comments for `Implements:` and `Holds:` markers,
and fails on a rule implemented and not held, held and not implemented, cited
to two pages, found elsewhere, or cited outside a closed grammar: Books 1–3
pages with readings, a reading alone, an ADR by its file, or one of eight
promise labels (Authority, Choices, Command line, Documents, Domain,
Generation record, Sheet, Transcription). Most of rc.6's new promises are
Sheet and Generation record markers: the service record's clauses, the
record's headings and counts, a drawn answer. Coverage is not measured (#198).
The mutation rule stands in: a new invariant is done only when
`go run ./internal/mutate` shows a deliberate edit killed in a scratch copy,
reporting `NOT BUILT` for one that does not compile. #500 asked whether this
machinery earns its lines and was closed as not planned.

## What it is shaped to accommodate, and what would need rethinking

**A new fact the sheet reads** costs four places: a case of `Fact` with a
`String` that is the record's line, a `FactCases` method (which stops
`sheetFold` and `historyFold` compiling until each says what it does), a
`record.fact` at the outcome carrying the value the engine reached, and the
working-state change beside it if the engine reads it later. Put a rule in the
fact, not the fold. If the fact carries a skill, its `historyFold` case must
go through `gained`.

**A new clause on the service record** is a field on `TermServed` or
`ServiceHistory` filled by `historyFold`, its view in `render/history.go`, and
its words in `termText` or `serviceLines`. rc.6 added four this way without a
new fact.

**A new reason on a choice** is data on `ChoiceEvent` written by `logging`,
from what the decider was handed, and words in `render`. It needs a test that
reads the reason back against something other than the code that records it,
as `policyDraws` is read from the policy's doc. Recording the decider's own
account of itself would put the explanation in the hands of whatever stands
for the player.

**A new reading** needs a constant, an entry in `Errata`, a `Reading` value, a
stamping literal, a predicate written from its prose, a fixture or walk that
reaches it (or an entry in `namesNoRecord`), and a marker; the compiler or a
test refuses most omissions. It needs nothing in `render`, and its number is
not a promise. A change to a reading's decision is an edit, but the predicate
must be rewritten from the new prose, never from the code.

**A correction to a transcribed cell** goes in both transcriptions from one
fresh visual reading of the page. If the page prints something the lift cannot
read, the lift gets a named decision, and a reading if the page does not
settle it. Editing the transcription toward what the lift accepts is the
damage ADR 0005 exists to prevent. A cell that is a DM threshold also moves
the record's text.

**A new choice point** is a `Decider` method, and the compiler takes the
maintainer through the policy, `logging`, the player and the test deciders;
lint takes them to `choiceWords`. If the page gives no basis for the answer,
the method takes a `Vary` and the policy draws.

**A new strategy value** is a case in the alphabet, a ranking in the method
that strategy answers, and a `does` case, which lint requires; the help reads
it from there. A strategy that would answer no where the page prices nothing
against yes is the shape #566 removed.

**A new flag** is an addition and breaks nobody. Giving a meaning to something
the command line now refuses is an addition too. Changing what an accepted
flag means after `v1.0.0` is a break the baseline test catches for types,
defaults and the values each takes, but not for a value whose behaviour
changes under the same word.

**A new property** is a freeze the moment `v1.0.0` ships, and renaming one
after it breaks every vault query that stands on it. Wording in the body is
free, and rc.6 is the evidence of how freely it moves.

**Reading a note back**, as a `replay` or a re-render in a new layout, would
need a record a tool can read, which ADR 0006 gave up. The typed facts are the
shape such a record would have, but they never leave the process.

**Another edition, or Books 4+,** would mean founding a sibling: the alphabets
are Book 1's, the transcription's bounds are its six services and four skill
tables, and the authority model forbids opening those books.

The damage a maintainer without the theory is most likely to do is quiet:
reading a cell from extracted text, tidying `"Intel 8+"` into the domain's
spelling in the transcription, typing a hyphen for an em dash, writing a
predicate by looking at the stamping site, regenerating a golden without
reading its diff, recording a sheet fact as prose through `outcomef`, changing
working state without the fact that states it, computing a rule inside a fold,
keeping a copy of a folded fact on `Character` "for speed", giving a
referee-facing surface an erratum id, accepting an ambiguous input "for
convenience", staging or second-guessing a write the operating system already
reports on, or writing a pronoun for a character. rc.6 adds three: explaining
an answer from how things turned out rather than from what the decider was
handed; taking the first name a printed list offers, which reads as a
preference the page never stated; and giving a strategy a "no" the page does
not price. Some of these the lift, the compiler, a test or a golden refuses;
the rest leave the gate green on every path no fixture walks.

## Uncertainties

- **Working state and the record are held equal only by the goldens.** I found
  no test that compares `run`'s fields with the folded `Character` at the end
  of a generation. By reading, every assignment to a working field is beside a
  `record.fact` stating its result, but a path no fixture walks is covered by
  inspection alone. The seeded runs in `explained_test.go` walk four hundred
  more paths, but hold only the lines that explain counts and the offer, not
  the sheet.
- **The stamp is unobservable outside the tests.** `Character.Errata` is not
  derivable from the events and no production code reads it, so a hand-built
  `Character` can carry any stamp and nothing would notice. I read this as
  ADR 0007's intent, not a gap.
- **`render` holds a copy of the policy's skills rule.** `leastTrained` panics
  where a policy's table was not least trained on, which is `Policy.thinnest`
  restated. The panic makes a disagreement loud on every path a golden walks;
  whether a renderer should know how a strategy decides, or the event should
  carry the policy's own reason, I can't tell from the code, and #564 asked
  only that the explanation come from the decision's inputs.
- **`Character.Policy` is half read.** Since #564 `render` reads its `Skills`
  to name the strategy that broke a tie; its `Career` and `Muster` are read by
  nothing outside the tests. Keeping the whole policy may be provenance for a
  later surface or what the removed footer left behind.
- **The shared strategy paragraph is not beside what it describes.**
  `strategiesShare` in `cmd/ctchargen/usage.go` states what every strategy
  answers, while each strategy's own line lives in `chargen` beside its method
  so that it can be read against it (#581). I can't tell whether the split is
  deliberate. Its claim once ran wider than `MusterWeapon` under `cash` and
  `goods` (#589), which is the drift the split invites.
- **"Record" still has two senses.** The glossary reserves the word for the
  note's generation record, and `chargen`'s generation-record type is
  `record`. Yet `Character`'s doc calls the character "the record a generation
  produces", every reading's stamped-on prose says "Records in which ...", and
  the stamping harness's own `record` type is the finished character. I take
  this as long-standing usage the readings' prose has frozen.
- **"Fact" has no glossary entry**, though it is the organizing word of the
  record store, used in ADR 0006, the changelog and the code; nor does the
  seeded run, the third way the tests walk the procedure.
- **Whether the note could be read back is argued, not tested.** ADR 0006 says
  a replay is "not possible from the note as written", yet the walks are
  extracted by hand from a note's record, which carries every die in order. I
  read the ADR as saying no tool should parse prose a later build may reword,
  not that the information is missing.
- **Authority rule 6 names two kinds of number, and the transcription holds a
  third**: amounts and targets the page states in prose that index no table
  (the medical crisis's 8+, its recovery to 1 and its 1D months; the education
  gate's 8+, now a `printedGate`). #43 named the gate as data without deciding
  where such numbers belong in general.
- **A title that is not on the profile line is on no part of the sheet.** The
  profile line names a title assumed; a dead noble's title, or one a person
  declined, appears only in the record's titles step, and the summary's doc
  says the title is the profile line's alone. That has been so since before
  rc.6, and I can't tell whether it is a decision or an omission.
- **The no-pronoun promise is held on the note only.**
  `TestNoNoteUsesAPronoun` reads the goldens. I found no pronoun in the guided
  session's questions or the help, but no test reads them.
- **Coverage of my reading.** I read in full `chargen`'s character, record,
  generate, logging, decider, policy, history, attempt, serve, enlist, rank,
  reenlist, muster and roller files; `traveller`'s facts, events, choices,
  gains and the sums header; every non-test file in `render`; and the
  command's `main`, `new` and `usage`. I read `rules` only where rc.6 changed
  it and to confirm each identifier this document names, `aging`, `batch`,
  `output` and `interactive` in part, and the tests named here. I diffed the
  tree against `v1.0.0-rc.5`, ran `go test ./...`, ran `new --help` and
  `batch --help`, and checked every golden against `bun fmt --check`. I did
  not open the PDFs: every claim about what a page prints comes from the
  transcription, the code's quotations and the readings' prose.
