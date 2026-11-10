# A theory of ctchargen

> Release snapshot for `v1.0.0-rc.5`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots").

## What it is for

ctchargen makes a Classic Traveller character the way Book 1 pp. 4–25 says
to, and hands the referee a note that can be checked against the page. That
second half is the point. Any generator can print a UPP and a skill list. This
one is built so that every number on the sheet goes back to a printed page, a
die that was thrown, and a choice someone made. Where the book is silent or
contradicts itself, the number goes back to a decision recorded in the source
under its own name, and since this release that decision is the maintainer's
to read, not the referee's.

The world it models is one procedure. Six characteristics are rolled. There
is one enlistment attempt, with the draft behind it. Then come terms of
service: survival, commission, promotion, skills, reenlistment, aging. Then
mustering out, and last a title the final Social Standing may confer. The
procedure has throws (2D against a target with modifiers), rolls (dice read
off a table, with nothing to meet), and points where the player chooses. Each
is a noun in the code: a `traveller.ServiceName`, a `traveller.Term`, a
`ThrowEvent`, a `RollEvent`, a `ChoicePoint` and a method on
`chargen.Decider`. A place the page is silent is a `rules.Reading`, and its id
inside the tree is a `traveller.Erratum`. GLOSSARY.md keeps that vocabulary,
and its "avoid" lines matter, because some words the code uses carefully are
ones a sibling tool would use loosely: a reading is not an erratum, a walk is
not a fixture, a script is not a seed, the record is the note's generation
record and not the whole note, the service history is not the log, and a
fold only records what its case says.

The authority model outranks everything else. The © 1977 text governs, and
memory of later editions does not. A reading decided here governs only where
the page is silent. The errata of the implementation deleted at `41a213a` are
a last resort, and even then only a pointer to a page that has to be reread.
A maintainer who leaves this model out will "fix" something into the 1981
rules; the best-known trap is that survival failure is death, with no injury
option (p. 5).

rc.5 exists because `v1.0.0` freezes two things and rc.4's playtests found
both still moving (#516). What freezes is the command line a referee types
(its flags, types, defaults and exit status, #126) and the note's property
names and meanings, which vault queries stand on (#399). Everything else in
the note stays free (ADR 0006). Two ideas organize this release's changes, and
the rest of this document keeps returning to them:

- **The note carries only what a referee acts on.** The readings' ids and
  text, the ruleset line, the policy's version and strategies, and the line
  telling the referee to keep the note whole are all gone from it. Each was
  either said elsewhere or served the tool rather than the character (ADR
  0007, ADR 0002 and ADR 0006 as amended, #525).
- **Where an input can be read two ways, refuse it now.** After the freeze, a
  refusal can be given a meaning as an addition, but a meaning once accepted
  cannot be taken away without a break (#466). The command line's new
  strictness follows from that asymmetry rather than from tidiness.

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

Since ADR 0007, `Errata` is the one field that is neither an input nor
derivable from the record. Through rc.4 each `OutcomeEvent` carried the ids
that governed it, and the stamp was a union of them. Now an outcome carries
none: the engine passes the erratum to `record.fact` or `record.outcomef` at
the stamping site, the record keeps a set beside its events, and
`record.stamped` writes it to the character in id order at the end. No note
names an erratum, `render` may not import `rules` (a depguard rule), and the
only readers of `Character.Errata` are the stamping harness and three tests
that look for a particular reading on a particular fixture. The stamp is the
authority's record of itself, kept on the character only so the gate can hold
it.

### How an outcome carries a fact

An `OutcomeEvent` has prose (`Description`) and an optional `Fact`: a sealed
sum in `traveller` with a cases interface, `FactCases`, of twenty-six methods.
The engine records a fact through `record.fact`, which sets the outcome's
`Description` to `fact.String()`. So a fact's prose is its own `String`, and
the line a referee reads and the value the sheet folds cannot say different
things. An outcome that changes nothing a sheet reads ("survived term 2", "not
promoted this term", "a second scout ship, which is lost", a dash row of Table
1, an aging throw that holds) goes through `record.outcomef` with prose and no
fact. (The type was `log` until #507 renamed it to the glossary's word, so
older commits and issues say `log.fact`.)

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
any departure at term 5 or later retirement, and the record still says
whether the throw refused the character.

The service history is a second fold over the same facts. Each `AppliedFact`
and `WeaponPickedFact` carries a `Source` (on entering, at a rank, in a term,
at mustering out), and `historyFold.gained` files a gain by it. New in rc.5
(#490), a gain at mustering out is filed too, into `MusteredOut`, and the
sheet's service record tells it after the departure line ("mustered out with
Education +1"), since mustering out comes after the service ends. A term is
the `StepEvent` whose `Opens` field names it, so a term a survival throw ended
is still a term served, with nothing in it.

### Working state alongside the record

The engine cannot read its own record as it runs: every DM reads the current
characteristics, the commission and promotion throws read the rank,
mustering out reads the terms and rank, the repeat rules read the benefits so
far, the pension reads the term count, and the end reads death off the
departure. So `chargen.run` keeps working state privately: `profile`,
`skills`, `rolledSocial`, `benefits`, `enlistment`, `terms`, `rank`, `atAge`
and `departure`, beside `eligible` and `trained`, which never reach a note.
Each change to working state is recorded as a fact carrying the value it left
(or, for terms, as the step that opens one), and the finished `Character` is
assembled from `Inputs`, `Policy`, the events and the stamp. The working state
dies with the run.

That is a second statement of every fact, kept for the length of one
generation, and the theory's honesty depends on naming it. The two cannot
disagree on any path a golden walks: the #441 PRs each showed fact and fold
mutations killed by a golden diff. Nothing compares the two directly, so a
path no fixture walks is held only by reading the engine: each assignment to a
working field sits next to the `record.fact` that states its result.

### What it costs, and why that cost stays

- **The sum is wide.** `FactCases` has twenty-six methods, one per kind, and
  every fold implements all of them; `historyFold` is mostly one-line
  `return nil` methods. #499 asked whether the visitor was accidental and was
  closed as not planned: the width is the closure, each `return nil` is a fold
  saying "not mine" in so many words, and a new kind of fact stops both folds
  compiling until each says what it does with it.
- **Every question folds every event.** `render.viewOf` asks the character
  for about a dozen facts, and each `sheetFold` method walks the whole record
  to keep one field. A note costs a dozen passes over a few hundred events,
  which is nothing beside writing a file, but a hot loop over `Character`
  methods would pay a walk per call.
- **A sheet fact must come through `record.fact`.** An outcome written through
  `outcomef` reaches the record and not the sheet. A benefit recorded as prose
  is a note whose sheet omits it, and only a golden that walks the path
  notices.
- **The record's wording is bound to the domain types.** A fact's `String` is
  the record's line, in `traveller`, so rewording a line is a change to the
  domain package. rc.5 paid this twice: "the Navy rejected him" became
  "rejected by the Navy" inside `EnlistedFact.String` (#459), and "forced
  out" became "denied reenlistment" by renaming `ForcedOut` and
  `ForcedOutFact` to `DeniedReenlistment` and `DeniedReenlistmentFact` (#485),
  each moving every golden that printed it.

### What it makes possible, and what it makes hard

It makes a character constructible from its record's own terms: the render
tests build a `Character` as nothing but events and assert on the note, so a
test cannot set a sheet the record contradicts. It also leaves the replay ADR
0001 kept possible a little nearer, since the in-process record is sufficient
to rebuild every sheet fact.

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

Since ADR 0005 the first transcription is Go: `internal/rules/transcription.go`
holds every table as a literal of the page's text, with its glyphs,
abbreviations, notation and dashes. It holds `"Intel 8+"`, not
`"Intelligence 8+"`; the em dash wherever a service has no commission, rank or
benefit; `"DM of +1 if"` as a row heading; the Aging Table's header rows and
dotted-leader spans; and `"Rifl3-1"` as the reprint shows it. **Structure
comes from the domain and cells from the page**: which table a grid is and
which service a column belongs to are positions keyed by `traveller`'s types,
so a table the type does not name, or a cell too many, does not compile. A
short Go array is padded with `""` in silence, so `refuseEmpty` refuses an
empty cell on the ground that the page prints a dash where it means nothing.
The service tables keep the page's rows as rows, the Table of Ranks included
(#354), so the two transcriptions read every grid in opposite directions.

**The lift is kept on purpose.** ADR 0005 rejected typing the tables as domain
values directly: a broken glyph would then fail to compile, but every
normalization and every dash would be decided in the transcriber's head, which
is a reading applied silently. The lift is where those judgements are written
down, and its malformed-input suite is what shows it refusing text the page
could not have printed. #501 argued the suite defends only against text typed
into Go by hand, and was closed as not planned for exactly this reason. A printed
name is checked against `traveller`'s closed alphabets; what the domain does
not spell lives in `rules` (`abbreviatedCharacteristics`, `boxServices`,
E012's spellings such as `Rifl3` to `Rifle`, and the fixed text around
values). `traveller.Sign` is the one place an en dash is recognised and
refused. `rules.Load` lifts once through `sync.OnceValues` and keeps its
error, since a literal can still hold text the lift refuses. The lift refuses
shape; the few value checks it makes are sanity bounds or cross-checks of the
page against itself (the Age header against `traveller.StartingAge` plus four
years a term). Whether a value is the one the page prints is the second
transcription's job, in the `rules` tests, written a service at a time in
normalized words.

rc.5 gives the transcription a new reader. Since #487 a throw's record line
names the characteristic DMs it earned: `rules.Throw.Met` returns each
prerequisite met, read from the lifted transcription, `Modifier` sums the same
ones, and `ThrowEvent.DMFrom` and `Odds.From` carry them. So a transcribed
threshold now reaches the note's text, and the #533 commit showed a mutation to
Other's survival cell moving the record. What reaches the note is the lifted
value, not the printed text: the record says "Intelligence 6+" where p. 10
prints "Intel 6+", and the comments say so: the characteristic's name and the
threshold the page prints (#539).

CLAUDE.md's Authority rule 6 sorts numbers: one that indexes a printed table is
in the transcription; a bare procedural bound is a constant beside the code
that applies it, with its cite (`maxRollsOnTableTwo`,
`minRankForTable1Modifier`, `lastVoluntaryTerm`, `freeTraderPaymentYears`). A
die's six faces are the one number said twice on purpose (#534):
`dice.Faces` for the dice and `rules.Faces` for the tables a die indexes,
because `dice` depends on nothing and `rules` may not import it, so `chargen`,
which imports both, holds them equal in a test. The cost ADR 0005 accepted is
that linters read the page's text: `goconst` is excluded for the transcription
and the `rules` tests, `mnd` for the transcription, and `task fix` over this
file is as unsafe as over a verbatim quotation.

## Readings: the authority's record, not the referee's

ADR 0007 is the largest change in this release, and it narrows what a reading
is for without changing how one is kept. A reading still does its first job:
it is how the authority model records a decision wherever the page is silent
or contradicts itself (Authority rule 5), held by the gate. Its second job,
being shown to the referee, is gone. The rc.4 playtest found E002 stamped on
nearly every skill line and E006 and E007 on every aging throw with nothing
telling the referee what to do with them, and E002 changes no outcome at all
(#460, #488). So no note names an erratum, `new --readings` and the `readings`
subcommand are removed, and `render`'s package doc carries the promise that a
note names none.

A reading still has three expressions, kept apart because the check between
them works only while they stay independent:

- The **prose** is a `rules.Reading` value in `internal/rules/readings.go`,
  with the page's emphasis as inline markdown. `rules.Readings()` returns an
  array sized by its own literal, so a reading missing or one too many does
  not compile, and `TestErrataListsEveryConstant` parses `traveller`'s source
  and requires `Errata` to be exactly its `Erratum` constants, in order. No
  production code calls `Readings()`; its readers are the tests and a person
  reading the source, which the README now points to.
- The **stamping site** is in the engine: thirteen sites across
  `enlist.go`, `serve.go`, `aging.go`, `rank.go`, `reenlist.go` and
  `muster.go`, each passed at the outcome it governs. The engine never reads a
  reading's prose, so a reading's wording can change without changing a
  character.
- The **predicate** lives in `internal/chargen/stamping_test.go`, one per
  reading, written by hand from the stamped-on prose and never from the
  stamping site (#66, #77), and checked both ways, record by record: an
  erratum stamped where its condition fails is as much a defect as one
  missing. A failure quotes the prose from `rules.Readings()`. The predicates
  read each fixture's `Character` through `recordOf`, beside `TestGoldens`
  rather than through it, so a reading the engine stops stamping is named by
  an ordinary run. E012 and E015 are stamped on no record by design, which
  `namesNoRecord` pins; the `parked` mechanism measures without judging a
  reading whose prose and code disagree, and none is parked.

What ADR 0007 takes away is ADR 0004's promise that an erratum names one
decision for good. No note carries an id, so nothing outside the tree is
stamped with a decision that might change, and the withdrawal machinery (a
withdrawn reading kept, explained and never stamped) is removed unused. A
reading whose decision no longer holds is now changed or removed like any
other code, and the predicates are held to what it says then. E016, a Free
Trader received past the fifth time (#530), was deliberately added after #460
so its id never reached a note.

The ADR names its price: a referee cannot tell from a note which reading
governed an outcome. Where a reading changes something the referee acts on,
the tool says so in plain words without naming it; the guided session's draft
question says that declining ends generation with a character who never
served, which is E001's consequence stated as a consequence (#483).

## The domain is sums, so the compiler counts the cases

`traveller` imports nothing in this module, and nothing in it rolls, reads a
table or writes markdown. Its rule for what becomes a type is that types carry
identity, never rule invariants. Wherever the book says "exactly one of", it
has a sealed interface with a `Fold` over a cases interface: `Enlistment`,
`TableResult`, `BenefitRow`, `Departure`, `WeaponBenefit`, `Event`, `Fact` and
`Gain`. Go does not check a type switch for exhaustiveness; the sums' comment
now says that `gochecksumtype` could, but only for a switch over an annotated
interface and only when the linter runs, while a fold's exhaustiveness is a
method set the compiler checks on every build (#498). The plain enums are
weaker: the exhaustive linter catches a missing switch case, and only under
`task`. `render.choiceWords` and `entryKind` are switches for that reason, so
a new choice point is a lint failure until it has the page's words. Every type
that reads a sum into a value is now named with the suffix `Fold`, and one that
reads it into prose with `WordsFold` (#512); a type that handles the cases to
act on them, and can fail, is named for what it does.

A sum is only as right as its list of cases, and the list is meant to be the
page's. Some choices make sense only against a sibling design: the civilian
who declined the draft is a case of `Enlistment`, not a nil service with a
flag; "died" is two `Departure` cases because they carry different fields;
`Fatal` lives in `traveller` because it was once stated twice (#100); `Age`
carries months only because a survived crisis adds 1D of them; `RollEvent` is
not a `ThrowEvent` with an empty target, so a roll is never written as having
succeeded (#50); and a `Target`'s zero value is "0 exactly", which no throw
can meet, so an unset target fails closed.

## How the tree treats what cannot happen

A branch the module's own types make impossible goes. Nothing outside the
module can build a `Character` (#199), and generation always records an
enlistment and gives every served character a departure and an enlistment
throw. So `servedIn` panics on a nil `Enlistment` (#377), `render.foldEnlistment`
does too (#393), `viewOf` folds a served character's departure without a
guard (#445), and `projectAttempt` panics on a served character with no
enlistment throw (#454): a generation defect crashes rather than printing a
civilian or a character who never left the service. A fold whose cases only assign drops the error with
the comment saying so, and `Character.fold`'s doc makes the same demand of
every `FactCases` handed to it.

The rule underneath is that the tool never writes a wrong note by its own
logic: a defect may crash, and an unwritable note is refused, but neither is
absorbed into a note that looks complete. What lies outside the module is the
file system, and rc.5 settles how far the command reaches into it: no further
than a plain write. #520 took out the staged write and rename that `--force`
had used since #191 and writes in place, as the shell's `>` does, and #532's
last commit removed two refusals that second-guessed the open, on the review's
ground that "we rely on the OS for the error".

## The engine asks, and the note says who answered

`chargen` walks the procedure and decides nothing on its own behalf. Every
point where the book lets a player choose is a method on `Decider`, twelve of
them, each typed to its own alphabet, which is what closes the set.
`TestChoicePointsMatchTheDecider` holds the `ChoicePoint` enum to the
interface's methods by reflection.

Three deciders exist (the auto run's `Policy`, the command's interactive
`player`, and the tests' scripted deciders), and the engine cannot tell them
apart. Each is reached through `logging`, the decorator `generate` wraps around
whatever it is handed, which keeps promises no single decider has to: a
question with fewer than two legal answers is refused before it is asked
(`asked`); each slice a decider is handed is its own copy (#55, #137, #230);
and an answer outside the offer is refused (`errNotOffered`). That last check
compares rendered names, which is sound only while `String()` is injective on
every alphabet, as `TestNoTwoValuesOfAnAlphabetShareASpelling` holds (#48).

The engine now shows a decider the stakes as well as the offer. The commission
and promotion questions are handed the throw a "yes" leads to, the
reenlistment question the survival throw the next term opens with, and the
service offer each service's enlistment throw, each as a `traveller.Odds`
with its target, DM and the prerequisites that earned it. The policy ignores
them all, since no strategy prices a failed throw, and the player prints them
(#518). The reenlistment question's survival odds are computed before the
term's aging round, which E006 places after the reenlistment throw, so they
can overstate the next term's DM by one aging round. `Decider.ReenlistIntent`'s
doc states which DM it is.

Who answered is fixed when the run is entered. `chargen.Auto` hands `generate`
a pointer to its policy and `chargen.Guided` hands it nil, and that pointer,
kept as `Character.Policy`, is the whole of what marks an auto run. In rc.4
the note's footer named the policy's strategies and `PolicyVersion`. #533
dropped the footer, and with it `PolicyVersion` and `chargen.Ruleset`: each
choice's line now says "policy chose" or "player chose", from that one pointer,
through `render`'s `answerer`. No note names a strategy any more, so nothing in
a note says which strategies an auto run used; `TestAnAutoRunAppliesTheStrategiesGiven`
was added to hold that `new --auto`'s strategy flags still reach the
generation once the footer no longer showed them. An auto run is `new --auto`
or any batch, which declares no `--auto` because it has nobody to ask (#336).
Since #420 the policy draws the service uniformly among the six with the run's
own dice, because a referee's 120-member batch came out 109 Army under the
older likeliest-enlistment ranking. The policy's documentation is its method
doc comments, and `DefaultPolicy` is the one place its defaults are written;
the command's flags read them (#367). `Attempted` reports whether the policy
named the service: an unforced auto run's attempt is the policy's, a
`--service` or a guided answer is not, and the headline says so when the draft
put the character elsewhere.

The narrowest seam is `Vary`. A decider may draw among the alternatives it was
offered (a weapon from pp. 12–13's lists, and the service) because the page
gives no basis to prefer one. It may not throw dice: a die thrown by a decider
reaches no event. `varyOnly` wraps the roller so a type assertion cannot
recover the dice (#136).

The order inside a term is the exposition's, under E002: survival,
commission, promotion, skills, reenlistment, then aging (E006). The term's
four years are added before the reenlistment throw, so its fact carries the
age the term left. `serve` has no bound on terms, because a 12 exactly on
reenlistment recurs (E003); termination is a contract on the roller (#54).

## Dice, scripts, and why the note is the only way back

ADR 0001 withdrew replay. The `dice` stream is unseeded, and nothing brings a
character back except the generation record, which logs every die and every
choice. ADR 0006 narrowed that: the record is the note's, written as prose,
and no build reads a note back. There is no seed, interactive mode cannot be
resumed, and a stopped session says what was answered and that nothing was
written. The note no longer ends by telling the referee to keep it whole
(#525): that was advice about the tool, and the note is the character's
record.

A `chargen.Roller` is `Die` and `Among`. `twoDice` is the one place a 2D throw
is made of dice, `Roller.Die`'s doc is the one list of one-die throws, and
`TestEveryThrowIsTheNextTwoDice` holds, for every fixture, that the dice the
record logs, in order, are exactly the dice its script handed out. That is
what lets a walk be extracted from a note's record at all.

A **script** is a path's dice and draws in the order the procedure asks for
them, and `internal/scripted` plays one strictly both ways. A **fixture** is a
script with a checked-in golden note, thirty-one of them; a **walk** is the
command tests' script, extracted once from a record the tool wrote, with no
golden. Consumption order matters to every script: a change that reorders
throws re-paths fixtures, and a golden diff is sometimes the only symptom, so
it is read before it is regenerated (`-update`, #506), and a failing golden
now prints the hunk that changed (#505). The goldens of `render.Note` live in
`chargen`'s tests because they run end to end over the fixture roster, which
is why #511 was closed as not planned. The Jamison reproduction of pp. 23–25
runs on the same scripted roller, writing each narrated total as two dice, the
first as high as the total allows, and differs from p. 25 in exactly the
benefit E015 accounts for.

## The note

`render` writes a `Character` through one `view`, built once by `viewOf`, so
the properties, summary, sheet and record cannot read the same character two
ways. The record's lines are made from `entry`, a flat projection of each
event built by one fold, `entryOf`, which the live view also uses. The live
view differs from the record in two places by design: a choice is echoed as
its number and answer, since the player can still see the question, and a step
is a plain numbered line, since a terminal prints markdown's marks (#422).

The note has two layers with two lifetimes. The **properties** freeze at
`v1.0.0`, in name and meaning, and rc.5 gave them the shape that freezes
(#474). Every property is on every note: text with no value is an empty key
and a list with none is `[]`, `aliases` on an unnamed note included, so a
query never meets a missing key. `skills` is names alone, one exact item each,
so "which crew have Pilot?" is an exact match, and the sheet has the levels.
`possessions` is new: what mustering out gave besides cash, one item each, a
Free Trader received again still one item. `ctchargen` is new: the build's
version alone, empty where nothing is embedded, while `ctchargen version` still
prints the module path too. Every text value is quoted, so no later word can
change a property's type. The meanings are stated once in `writeProperties`'
doc and once, as a typed table, in the README, and
`TestANoteOpensWithItsProperties` holds the shape typed out.

The **body** stays free and moved a good deal. The summary under the heading
(#425) says only what the sheet below says. The sheet tells the service a
term at a time (#401), and a characteristic change the floor or ceiling held
says the table's change and where it held (#409). A throw's line now names
the characteristics its DM was earned by, and an outcome names its cause as
"(from step N)", where "(from 2)" read as a characteristic that had been 2
(#487). A failed reenlistment throw below term 5 reads "denied reenlistment",
p. 6's own word (#485). And no line names the character by a pronoun, because
a character has no sex in these rules (p. 8, #459);
`TestNoNoteUsesAPronoun` reads every golden for one, and the promise sits on
`render`'s package doc because it holds across everything the package writes.
The note ends with its generation record; nothing follows it (#533).

## The command

`cmd/ctchargen` is thin, and its seams are channels and refusals. The note goes
to stdout and the conversation to stderr, so `ctchargen new > kira.md` saves a
note. Every failure exits 1 (#121). `errUsage` now marks only a mistake in the
command line: a file in the way, a missing or non-empty folder and a build with
nothing embedded are found at run time and carry no "usage" label (#472,
#508). `errSearch` marks a batch that could not find enough living. A flag is
written in errors as the help writes it (`--count`, `-o`), where the flag
package writes one dash, and a path appears once in a message.

The refusals follow #466's rule. `strictly` wraps every flag a set declares, so
a flag given twice or given an empty value is refused by name rather than read
as the last one or as absent. `--count` is a `decimalCount`: `010` is ten, and
`0x10`, `1_000` and `+3` are refused, because the flag package's base-0 int
read `010` as eight. `-o -` is standard output for both commands, as no `-o`
is. `--force` with no `-o` file is refused, having nothing to overwrite. A
strategy flag without `--auto` is refused, because the player would answer
every choice it names. A stray positional word is refused, because the flag
package stops at it and leaves later flags unparsed. A batch `-o` ending in
`.md` is refused (#416). Each is a refusal now so that a meaning can be added
after `v1.0.0` without breaking anybody. The help says how flags are written,
and names each flag's kind, read from the value itself because the strict
wrapper hides the kind the flag package would print (`-o file` for `new`,
`-o folder` for `batch`, #517).

`new -o` is checked before the first question by `refusedBefore`: its folder
must exist, and the path must not exist without `--force` or be a directory.
Refused only at the end, a guided session lost every answer (#424). Without
`--force` the note is opened create-exclusive, and a write that fails removes
the note this run made (#468). With `--force` the file is truncated and written
in place, as `>` writes (#469), so a symlink is followed and the file keeps its
mode and hard links; the trade is that a `--force` write that fails partway has
lost the old note. Notes and folders are created `0666` and `0777`, less the
umask. Whatever else is in the way goes to the open: `new --force -o /dev/null`
succeeds as `> /dev/null` does.

A batch writes only the living (#421): it passes over the dead and generates
another, rerolling nobody, so every note is one the dice made, and gives up
with `errSearch` after a hundred characters per member asked for, a bound
computed by division because multiplication overflowed for a huge count
(#423). Into `-o` it writes only a folder that is new or empty but for hidden
entries (#426), checked once before anyone is generated, and makes that folder
with `Mkdir`, so only inside one that exists (#470); it never removes anything
but a member's note whose write failed. Each note is named
`NN Service Rank UPP.md`, from its position padded to the batch's width and
`render.Title` (#400), and is headed by that title, since a batch names nobody
(#406). Without `-o` the notes stream to stdout as they are made (#44). The
closing tally goes to stderr, never into the last note.

The interactive `player` is a `Decider` that reads numbers and never invents an
answer: input that runs out is `errNoAnswer`. ADR 0003 decided its tests go
through the command, because the wiring is what a person notices breaking.
`cli_test.go` types out the flags, types and defaults and holds them one way: a
new flag breaks nobody, a changed or removed one does, and each removal before
`v1.0.0` is recorded beside the baseline as deliberate, now including
`--readings` and the `readings` subcommand (#460).

## What the gate holds

There is no CI (#149). The gate is a local `task`: tidiness, lint (config
verify first, then every linter golangci-lint ships but five, with every
`go vet` analyzer and the formatter), nilaway, govulncheck, the tests under
`-race`, and the non-Go files' format and offline links. A note is Markdown
the formatter keeps as written: its record lines carry bare numbers, and
`bunfig.toml` ignores by name only the goldens whose step crosses from 99 to
100, where the formatter aligns the list by where it starts (#542). `task setup` installs
it from the repository alone: the Brewfile lists every Homebrew tool the gate
runs and the task adds NilAway with `go install` (#492). The toolchain is
unpinned on purpose, so a red gate on untouched code is a tool moving, to be
answered rather than pinned away (#493). No markdown file is parsed, generated
or held to the code (ADR 0004), and no data file is read at load (ADR 0005).

`internal/coverage` is a test-only package that walks Go doc comments,
package docs and the fixture roster's comments for `Implements:` and `Holds:`
markers, and fails on a rule implemented and not held, held and not
implemented, cited to two pages, found elsewhere, or cited outside a closed
grammar: Books 1–3 pages with readings, a reading alone, an ADR by its file,
or one of eight promise labels (Authority, Choices, Command line, Documents,
Domain, Generation record, Sheet, Transcription). Coverage is not measured
(#198). The mutation rule stands in: a new invariant is done only when
`go run ./internal/mutate` shows a deliberate edit killed in a scratch copy,
reporting `NOT BUILT` for one that does not compile. #500 asked whether this
machinery earns its lines and was closed as not planned: the markers, the
check that joins them and the mutations shown are the authority model's gate.
The rc.5 PRs list each mutation and what killed it in their commits.

## What it is shaped to accommodate, and what would need rethinking

**A new fact the sheet reads** costs four places: a case of `Fact` with a
`String` that is the record's line, a `FactCases` method (which stops
`sheetFold` and `historyFold` compiling until each says what it does), a
`record.fact` at the outcome carrying the value the engine reached, and the
working-state change beside it if the engine reads it later. Put a rule in the
fact, not the fold.

**A new reading** needs a constant, an entry in `Errata`, a `Reading` value, a
stamping literal, a predicate written from its prose, a fixture that reaches
it (or an entry in `namesNoRecord`), and a marker; the compiler or a test
refuses most omissions. Since ADR 0007 it needs nothing in `render`, and its
number is not a promise. A change to a reading's decision is now an edit, but
the predicate must be rewritten from the new prose, never from the code.

**A correction to a transcribed cell** goes in both transcriptions from one
fresh visual reading of the page. If the page prints something the lift cannot
read, the lift gets a named decision, and a reading if the page does not settle
it. Editing the transcription toward what the lift accepts is the damage ADR
0005 exists to prevent. A cell that is a DM threshold now also moves the
record's text.

**A new choice point** is a `Decider` method, and the compiler takes the
maintainer through the policy, `logging`, the player and the test deciders;
lint takes them to `choiceWords`. There is no policy version to bump.

**A new flag** is an addition and breaks nobody. Giving a meaning to something
the command line now refuses (a repeated flag, an empty value, a non-decimal
count) is an addition too. Changing what an accepted flag means after
`v1.0.0` is a break the baseline test catches only for types and defaults.

**A new property** is a freeze the moment `v1.0.0` ships, and renaming one
after it breaks every vault query that stands on it. Wording in the body is
free.

**Reading a note back**, as a `replay` or a re-render in a new layout, would
need a record a tool can read, which ADR 0006 gave up. The typed facts are the
shape such a record would have, but they never leave the process.

**Another edition, or Books 4+,** would mean founding a sibling: the alphabets
are Book 1's, the transcription's bounds are its six services and four skill
tables, and the authority model forbids opening those books. With
`chargen.Ruleset` gone no note names its books; the README says it once.

The damage a maintainer without the theory is most likely to do is quiet:
reading a cell from extracted text, tidying `"Intel 8+"` into the domain's
spelling in the transcription, typing a hyphen for an em dash, writing a
predicate by looking at the stamping site, regenerating a golden without
reading its diff, recording a sheet fact as prose through `outcomef`,
changing working state without the fact that states it, computing a rule
inside a fold, or keeping a copy of a folded fact on `Character` "for speed".
rc.5 adds four: giving a referee-facing surface an erratum id; accepting an
ambiguous input "for convenience", which after the freeze cannot be taken
back; staging or second-guessing a write the operating system already
reports on; and writing a pronoun for a character in any line a note prints.
Some of these the lift, the compiler, a test or a golden refuses; the rest
leave the gate green on every path no fixture walks.

## Uncertainties

- **Working state and the record are held equal only by the goldens.** I found
  no test that compares `run`'s fields with the folded `Character` at the end
  of a generation. By reading, every assignment to a working field is beside a
  `record.fact` stating its result, but a path no fixture walks is covered by
  inspection alone. I can't tell whether a direct check was considered and
  declined.
- **The stamp is now unobservable outside the tests.** Since ADR 0007,
  `Character.Errata` is not derivable from the events and no production code
  reads it. A hand-built `Character` can carry any stamp and nothing would
  notice. I read this as ADR 0007's intent (the stamp exists for the
  predicates), not a gap.
- **`Character.Policy` keeps three strategies that nothing outside the tests
  reads.** Since the footer went, `render` and `Attempted` read only whether
  it is nil. I don't know whether keeping the whole policy is meant as
  provenance for a later surface or is what the footer left behind; nothing I
  found decides it.
- **"Record" still has two senses, and #507 sharpened the clash.** The
  glossary reserves the word for the note's generation record, and `chargen`'s
  generation-record type is now `record`. Yet `Character`'s doc calls the
  character "the record a generation produces", every reading's stamped-on
  prose says "Records in which ...", and the stamping harness's own `record`
  type is the finished character. I take this as long-standing usage the
  readings' prose has frozen, not drift to correct in passing.
- **"Fact" has no glossary entry.** It is the organizing word of the record
  store, used in ADR 0006, the changelog and the code, and GLOSSARY.md does
  not define it or say how it differs from an outcome.
- **Whether the note could be read back is argued, not tested.** ADR 0006 says
  a replay is "not possible from the note as written", yet the command tests'
  walks are extracted by hand from a note's record, which carries every die in
  order. I read the ADR as saying no tool should parse prose a later build may
  reword, not that the information is missing.
- **Authority rule 6 names two kinds of number, and the transcription holds a
  third**: amounts and targets the page states in prose that index no table
  (the medical crisis's 8+, its recovery to 1 and its 1D months; the education
  gate's 8+). #43 named the gate as data without deciding where such numbers
  belong in general.
- **The no-pronoun promise is held on the note only.** `TestNoNoteUsesAPronoun`
  reads the goldens. The #533 commit says the guided session has no pronoun
  either, and I found none in its questions, but no test reads them.
- **Coverage of my reading.** I read in full `chargen`'s character, record,
  generate, logging, decider, policy, history, serve, reenlist, attempt and
  roller files, `traveller`'s facts, events, errata and sums header, every
  non-test file in `render`, the command's `main`, `new`, `batch` and `output`,
  and the readings' type and E013 to E016. I read `rules`' lift and parse,
  `usage`, `interactive`, `aging`, `enlist`, `muster` and most tests only in
  part, checking each identifier this document names. I ran `go test ./...`
  and the command's refusals from a scratch directory. I did not open the PDFs:
  every claim about what a page prints comes from the transcription, the
  code's quotations and the readings' prose.
