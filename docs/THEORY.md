# A theory of ctchargen

> Release snapshot for `v1.0.0-rc.3`. It describes that tag, not `main`.
> Between releases nothing reads, sweeps or updates it (CLAUDE.md, "Release
> snapshots").

## What it is for

ctchargen makes a Classic Traveller character the way Book 1 pp. 4–25 says
to, and it hands the referee a record he can check against the page. That
second half is the point. Any generator can print a UPP and a skill list.
This one is built so that every number on the sheet goes back to a printed
page, a die that was thrown, and a choice someone made. Where the book is
silent or contradicts itself, the sheet goes back to a decision recorded
here, under its own name.

The world it models is one procedure. Six characteristics are rolled. Then
there is one enlistment attempt, with the draft behind it. Then come terms of
service: survival, commission, promotion, skills, reenlistment, aging. Then
mustering out, and last a title the final Social Standing may confer. The
procedure has throws (2D against a target with modifiers), rolls (one die,
or two, read off a table), and points where the player chooses. Every one of
those is a noun in the code. A service is a `traveller.ServiceName`. A term
is a `traveller.Term`. A throw against a target is a `ThrowEvent`, and a
roll with nothing to meet is a `RollEvent`. A choice point is a `ChoicePoint`
and a method on `chargen.Decider`. A place the page is silent is a
`rules.Reading`, and the id a record carries for that reading is a
`traveller.Erratum`. GLOSSARY.md keeps that vocabulary. Its "avoid" lines are
worth reading, because some words the code uses carefully are ones a sibling
tool would use loosely: a reading is not an erratum, a walk is not a
fixture, a script is not a seed, and a transcription is not "table data".

The authority model ranks first and outranks everything else. The © 1977
text governs. Memory of later editions does not. A reading decided here
governs only where the page is silent. The errata of the implementation
deleted at `41a213a` are a last resort, and even then only a pointer to a
page that has to be reread. Old code and old transcriptions never come back.
A maintainer who leaves this model out of his own understanding will "fix"
something into the 1981 rules. The best-known trap is survival failure: here
it is death, with no injury option (p. 5).

rc.3 changed nothing a referee runs. Every golden it holds is the one rc.1
wrote. Most of the work between rc.2 and rc.3 was a readiness review
(#362–#376) that asked whether the code was as lean as it could be, and its
answers sharpen the theory more than they change it. One idea runs through
all of them: **a fact is stated once, and everything else reads it**. The
record is the one statement of a character, so every rendering of it is made
from its bytes. The character's sums are the one statement of how he entered
and left his service, so the engine reads them rather than keeping flags
beside them. A 2D throw is made of dice in one place. The default policy is
written once and the flags read it. Most of what follows is that idea,
applied to the page, the readings, the engine and the record in turn.

## The page, typed as printed, and read twice

The reprints' embedded font substitutes glyphs, and the substitutions look
like data. A dash cell comes out as the digit 4, a minus comes out as a 3,
and "Travellers' Aid" comes out as `Travellers9`. The substitution reaches
the rendered image too: p. 23's box shows `Rifl3-1`. That one fact explains
a lot of the repository's shape.

Since ADR 0005, the first transcription is Go: one file,
`internal/rules/transcription.go`, holding every table as an unexported
literal of the page's text. There is no `data/` directory, no `//go:embed`,
and no wire layer between the page and the lift. A cell is a string holding
what the page prints, with its glyphs, abbreviations, notation and dashes.
So the transcription holds `"Intel 8+"` and `"+1 Stren"`, not
`"Intelligence 8+"`. It holds the em dash `"—"` wherever a service has no
commission, rank or benefit. It holds `"DM of +1 if"` as a row heading, not
a number in a field. It holds the Aging Table's two header rows, and each
characteristic's cells with the term columns their dotted leaders span. It
holds p. 23's box under its own labels (`"Navy Captain"`, `"Marine"`), with
`"Rifl3-1"` as the reprint shows it.

The split ADR 0005 draws is that **structure comes from the domain and cells
come from the page**. Where the page's own text carries a layout, the
transcription keeps it: the Aging Table's spans, the DM rows' headings. Where
only position carries it, the position is keyed by `traveller`'s types:
which table a grid is, which service a column belongs to, which ship a hull
is. So the skills tables are
`[len(traveller.SkillTables)][Faces][len(traveller.ServiceNames)]string`,
and a table the type does not name, a table given twice, or one cell too
many does not compile. Go brings one hazard of its own: a short array is
padded with `""` without a word. So `refuseEmpty` refuses any empty cell, on
the ground that the page prints a dash where it means nothing.

The service tables of pp. 9, 10 and 11 keep the page's rows as rows: a line
of their literals is a row the page prints, with the services across. That
is now true of all of them. At rc.2 the Table of Ranks was typed a service's
column at a time, the same direction the second transcription walks it, so
for that one table the two copies read the page the same way (#354). Since
#358 it is the page's six rows, and `rankColumn` reads a service's cells back
out for `liftRanks`.

**The lift is kept on purpose.** ADR 0005's rejected alternative was to type
the tables as domain values directly and delete the lift. A broken glyph
would then fail to compile. But every normalization and every dash would be
decided in the transcriber's head, with no code to name it, and that is a
reading applied silently. The lift is where those judgements are written
down. The note at the head of `parse.go` names their kinds and where each
lives (#375). Where the domain has a closed alphabet, a printed name is
checked against the type and nothing else. What the domain does not spell is
kept in `rules`: how the tables abbreviate a characteristic
(`abbreviatedCharacteristics`, whose doc argues it is settled by the page and
is not a reading), how p. 23's box spells the services (`boxServices`, under
E012), E012's spelling-out of the tables' names (`Rifl3` → `Rifle` among
them, a font-trap glyph caught by the semantic cross-check CLAUDE.md
describes and recorded as a reading), and the fixed text the page prints
around its values: the dashes, the box's `-1`, a weapon row's bare category,
and the Aging Table's two phrases for no effect.

The dashes show the "one statement" idea at its smallest. Three glyphs look
alike on screen. The em dash means nothing, and `emDash` says so. A lone
hyphen where the page prints an em dash is refused by name, because a cell
typed with one was not read off the page. A hyphen opening a number is a
minus. The en dash is the one that kept escaping. At rc.2, four parse paths
each checked for one before reading a sign, and three paths that read a sign
did not, so an en dash in a DM heading, a Table 1 alteration or a box grant
was refused only as "no sign". Since #366, `traveller.Sign` returns an error
for an en dash. It is the one place that recognises one, every path reads a
sign through it, and seven paths are held to refusing it by name.

`rules.Load` lifts once per process through `sync.OnceValues` and keeps its
error, because a literal can still hold text the lift refuses. The lift
refuses shape, not value: a name outside `traveller`'s closed alphabets, a
target in a notation pp. 2–3 do not use, a rank printed below a dash, a DM
beside a dashed throw, an Aging cell that leaves a gap or runs past the last
printed term. Each DM row's heading is read to its amount once, before any
service's column is read, so a bad heading is refused as the row's (#371). A
few checks do reach values. Some are sanity bounds: the draft row must be six
distinct die faces, and a cash allowance, a pension and a hull must be
positive, the pension rows ascending. Two check the page against the domain
or against itself. The Age header must read
`traveller.StartingAge + traveller.Years × term`, which is why `StartingAge`
lives in `traveller`: `rules` cannot import `chargen`. A box label's rank
title must be one its service's Table of Ranks column prints. Those are
cross-checks, not a second list. Whether a value is the one the page prints
is the second transcription's job.

The second transcription, in the `rules` tests, is laid out apart from the
page's grid and written in the normalized vocabulary (`"+1 Intelligence"`,
`"weapon Blade Combat"`). The first walks a row of the page, six services
across, and the second walks a service down its column. Two cells swapped
within a row compile and lift like any other cells, and only the second copy
notices. That independence in direction and in words is what the font trap
needs.

What the transcription holds is bounded by what the procedure consults. At
rc.2 it also held the passage prices of pp. 21–22 and the 90% resale rate.
Only tests read them, because selling a passage is play after generation and
the record carries the assets and stops. #365 removed them. Those pages state
the prices in prose, so the `rules` package's claim to hold "every table of
pp. 4-25" still stands.

CLAUDE.md's Authority point 6 sets which numbers are in the transcription. A
number that indexes a printed table is there. The Aging Table's last printed
term is one, read off the term header row the page prints. A bare procedural
bound is a Go constant with its cite, placed beside the code that applies it.
The three-roll cap on Mustering Out Table 2 and the rank that opens Table 1's
+1 are examples (`maxRollsOnTableTwo`, `minRankForTable1Modifier` in
`chargen/muster.go`), and so are the benefit roll counts and rank thresholds
beside `Muster.Rolls`. The rule moves numbers in both directions, so a number
on the wrong side of it is a defect, not a style choice.

The cost ADR 0005 accepted is that linters now read the page's text.
`goconst` is excluded for `transcription.go` and the `rules` tests, because
collapsing a repeated cell into one constant lets one typo reach every cell
that shares it. `mnd` is excluded for `transcription.go`, because a number
there is the page's text. Any linter added later that reads string literals
will read the cells, and a `task fix` over this file is as unsafe as it is
over a verbatim quotation.

## Readings, errata and stamping

A reading has three expressions in the tree. They are kept apart on purpose,
because the check between them only works if they stay independent.

The **prose** is a `rules.Reading` value in `internal/rules/readings.go`. Its
fields are title, pages, what the page says, the gap, the reading, why, and
the stamping condition, all kept verbatim with the page's emphasis as inline
markdown. `rules.Readings()` returns `[len(traveller.Errata)]Reading` from an
array sized by its own literal, so a reading missing from the array, or one
past the last erratum, does not compile. `TestTheReadings` holds each value's
`ID` to its position, which is why erratum `e`'s reading is
`Readings()[e-1]` and why a caller that wants one indexes it that way. #370
removed the lookup function only tests called, and #376 declined to key the
array by erratum: the test already fails naming the erratum, and a key would
add noise and no safety. `TestErrataListsEveryConstant` parses `traveller`'s own
source for constants of type `Erratum` and requires `Errata` to be exactly
those, in order, numbered from 1 (#333). The chain is const block → `Errata`
(a test) → readings (the compiler).

The **stamping site** is in the engine. An outcome event that a reading
governed carries the erratum as a literal, for example
`[]traveller.Erratum{traveller.E004}` on a death at the survival throw.
`log.stamped` takes the union of those, in id order, as the record's
`errata`. The engine never reads a reading's prose. `chargen` imports
`rules` for tables, and only its tests read the readings. That is why a
reading's wording can change without changing a single character.

The **predicate** lives in test code: one function per reading in
`internal/chargen/stamping_test.go`, written by hand from the stamped-on
prose and never from the stamping site (#66, #77). It is checked in both
directions, record by record. A reading stamped where its condition fails is
as much a defect as one missing where the condition holds, because a
record's errata list is the tool's claim about what the character rests on.
A failure quotes the prose from `rules.Readings()`, not from a copy typed
beside the predicate. This is the same trap as the double transcription
again: a condition copied from the code that does the stamping checks
nothing.

Since #373 the predicates judge the record each fixture writes now, not the
golden on disk. `readRecords` encodes each fixture's character and decodes
it, and `TestGoldens` holds those bytes to the checked-in goldens. Before,
an engine that stopped stamping a reading failed only as a golden diff, and
a reader had to work out which reading the diff was about. Now an ordinary
run names it.

Two readings, E012 and E015, are stamped on no record by design. E012 is a
spelling normalization. E015 matters only to the worked example's
reproduction. `namesNoRecord` pins that set, and `unreached`, now the one
check of it, fails if a reading silently stops appearing or starts. The
predicates also carry a `parked` mechanism: a reading whose prose and code
disagree, undecided, is measured and not judged, and `stillParked` pins
both the set and the count. None is parked at rc.3.

ADR 0004 gave an erratum a lifetime. From `v1.0.0`, an erratum names one
decision for good, the decision being both what the reading decides and when
it is stamped. Wording can change under an erratum. A changed decision needs
a new erratum. A decision that no longer holds is withdrawn, never deleted:
the `Withdrawal` field says why and names any replacement, the reading stays
in the array so old records still resolve, and `render.Reading` marks it
withdrawn. `verdict` holds a withdrawn reading to one thing only: no golden
stamps it (#332). Its condition is not judged, because a reading replaced by
another still describes the records its replacement governs.
`TestAWithdrawnReadingOnAGoldenIsRefused` stands E001 in as withdrawn under
its real predicate. No reading has actually been withdrawn.

## The domain is sums, so the compiler counts the cases

`traveller` imports nothing in this module, and nothing in it rolls, reads a
table or marshals. Its rule for what becomes a type is that types carry
identity, never rule invariants. A service outside the six does not exist.
A range a page prints is not a type. The few bare numbers it carries (a
term's `Years`, the `StartingAge`, the characteristic floor and cap) are
domain facts more than one package needs, and placing them here is what the
import arrows allow.

Wherever the book says "exactly one of", the code has a sealed interface
with a `Fold` method over a cases interface that has one method per case.
This covers `Enlistment` (enlisted, drafted, declined the draft),
`TableResult`, `BenefitRow`, `Departure`, `WeaponBenefit` and `Event`. Go
does not check a type switch for exhaustiveness, so a fold is how adding a
case breaks every consumer at compile time: the JSON codecs in `render`, the
death test `traveller.Fatal`, the engine's own folds, and the second
transcription's `benefitDescriber`. The plain enums are a weaker strength.
The exhaustive linter catches a missing `switch` case, and only under `task`.
The package doc spells out the difference, and it is real: dropping the
linter drops the plain enums' guarantee, and nothing can drop the folds'.

A sum is only as right as its list of cases, and the list is meant to be the
page's. #364 found `BenefitRow` carrying a cash case that no row of Table 1
can produce. Table 1 prints six kinds of row and no money. Table 2's cash is
a number, read beside the row and never a `BenefitRow`. So the sum now has
six cases, and the fold that could never run is gone.

Some modelling choices only make sense against a sibling design that did it
differently. The civilian who declined the draft is a case of `Enlistment`,
not a nil service with a flag beside it. The book's single word "died"
becomes two `Departure` cases, because a survival death and a medical-crisis
death carry different fields. `Fatal` lives in `traveller` because it was
once stated twice and the two copies could disagree (#100). `Age` carries
months only because a survived medical crisis adds 1D of them, and
`PlusMonths` floors toward negative infinity to protect its one invariant.
`RollEvent` is separate from `ThrowEvent` so that a roll with nothing to
meet is never written as `"succeeded": true` (#50). A `Target`'s modality is
unexported, and `ParseTarget` is the only way outside `traveller` to give one
a number (#370). Its zero value is "0 exactly", which no throw of two dice
can meet, so a target never set fails closed.

## How the tree treats what cannot happen

The readiness review removed a good deal of code, and the line it drew is
worth holding, because it is not "delete whatever no test reaches".

A branch the module's own types or boundary make impossible goes. Since #199
nothing outside the module can build a `Character`, and generation always
sets its `Enlistment`, so `ServedIn`'s guard for a nil one was reached only
by a test written for a caller who no longer exists (#377). It went, and a
nil `Enlistment` now panics in the fold, which shows a generation defect
instead of printing "no service". `thinnest`'s fallback for counts not
parallel to the offered tables went for the same reason (#370). One guard of
the same kind survived the sweep: `render`'s `foldEnlistment` still turns a
nil `Enlistment` into an empty record, a branch nothing reaches (#393).

An error the module's own code cannot produce is dropped where it arises,
with the reason beside it. Every fold over a codec that only assigns is
written `_ = x.Fold(&codec)` under the comment "A fold over a codec cannot
fail: every case only assigns."

An error that comes from outside the module is returned, even when nothing
today can cause it. #362 tried dropping `json`'s error from the record's
projection, then restored it. Without a returned error, an impossible marshal
failure has two places to go: a panic, or a discard that writes a record
missing an event while the record says it is the only way back to the
character. "A branch no test reaches" is not a standard here, because
coverage is not measured (#198). The live view is the one place that
drops `json`'s error, and its doc says why: a live line is a courtesy to the
person watching, and the record that follows still refuses.

Read together, the rule is that the record is never written wrong. A defect
may crash, and an unwritable record is refused, but neither is absorbed into
a record that looks complete.

## The engine asks, and the record says who answered

`chargen` walks the procedure and decides nothing on its own behalf. Every
point where the book lets a player choose is a method on `Decider`. Each
method is typed to its own alphabet, and that typing is what closes the set.
A new choice point is a new method, and every decider stops compiling. The
`ChoicePoint` enum is unavoidably a second list. A reflection test holds it
to the interface's methods, so the two cannot drift.

Three deciders exist: the `Policy` an auto run applies, the interactive
`player` in the command, and the tests' scripted deciders. The engine cannot
tell them apart. Each one is reached through `logging`, the decorator
`generate` wraps around whatever it is handed. Being the only path in, it
can make promises no single decider has to keep:

- a question with fewer than two legal answers is refused before it is asked
  (`asked`), so the record never shows a choice nobody could have made
  differently;
- each slice a decider is handed is its own copy, so writing to an offer
  changes nothing the engine offers later (#55, #137, #230);
- an answer outside the offer is refused, not applied (`errNotOffered`).

That last check compares rendered names. It is sound only while `String()`
is injective on every alphabet, and a test in `traveller` holds exactly that
(#48). The five yes-or-no points share one helper, `yesOrNo`, since only the
point differed between them (#377).

Who answered is fixed when the run is entered, not carried per event.
`chargen.Auto` hands `generate` a pointer to its policy, and
`chargen.Guided` hands it nil. That pointer, kept as `Character.Policy`, is
the whole of what marks an auto run. The record writes the policy's
strategies and `PolicyVersion` from it, and a guided record carries no policy
at all, which is its statement that a person made every choice. An auto run
is `new --auto` or any batch: a batch takes no `--auto`, because it has
nobody else to ask (#336). `PolicyVersion` changes only when an answer
changes, and it is currently 3. A new choice point does not bump it, because
no earlier record was answered differently. `build` changes with every commit
and cannot do this job. The policy's documentation is its own method doc
comments: each says when the engine asks and what each strategy answers, and
nothing else describes the auto run. The default policy is written once, in
`DefaultPolicy`, and the command's strategy flags read their defaults from it
(#367). Before, a default changed in one place alone would have kept every
golden green under a policy the tool no longer applied, since most fixtures
generate under `DefaultPolicy` and the binary under its flags.

The run's own state follows the same rule (#368). Whether he died is
`traveller.Fatal` of his departure. Whether the draft entered him is read by
folding his enlistment, as `ServedIn` does. The next event's number is the
log's length plus one. None of these is kept beside the fact it restates,
because a copy is a second statement that can come to disagree with the
first.

The narrowest seam in the engine is `Vary`. A decider may draw among the
alternatives it was offered. The weapon lists of pp. 12–13 come in a printed
order, but that order is not a preference (#34). A decider may not throw
dice: a die thrown by a decider reaches no event, and the record would stop
carrying every die. `varyOnly` wraps the roller, so a type assertion cannot
recover the dice from the value (#136).

The order inside a term is the exposition's, under E002: survival,
commission, promotion, skills, reenlistment, then aging (E006). The worked
example narrates it differently. The fixtures' scripts walk this order, so
it is a fixed choice, not an incidental one. `serve` has no bound on terms,
because a 12 exactly on reenlistment recurs without limit (E003). That makes
termination a contract on the roller, not a property of the loop (#54).

## Dice, scripts, and why the record is the only way back

ADR 0001 withdrew replay. The `dice` stream is unseeded, and nothing brings a
character back except the record, which logs every die. That decision runs
through the whole tree. There is no seed field on a record. Interactive mode
cannot be resumed, and a stopped session tells the player what it lost.
Fixtures state their dice instead of carrying a seed that happened to reach
them.

A `chargen.Roller` is two things: `Die` and `Among`. Until #377 it also had
`TwoDice`, and both rollers implemented it as two calls to `Die`. Having it
on the interface let a test roller make a 2D throw that was not the stream's
next two dice. One test roller answered 2D and 1D throws differently, and the
worked example's roller took its totals from a second queue. Now `twoDice`
in `chargen` is the one place a 2D throw is made of dice. `Roller.Die`'s doc
is the one list of the one-die throws. `TestEveryThrowIsTheNextTwoDice` holds
the result: for every fixture, the dice the record logs, in order, are
exactly the dice its script handed out. That is a stronger statement than
"the record logs every die". It says the record logs them in the order they
were drawn, and that is what lets a walk be extracted from a record at all.

A **script** is a path's dice and draws in the order the procedure asks for
them. `internal/scripted` plays one strictly in both directions: asking past
the end fails naming the roller, and a die left unread fails at `Finished`.
A **fixture** is a script with a checked-in golden. A **walk** is the
command tests' script, extracted from a record, with no golden. The order
in which dice are consumed matters to every script. A change that reorders
throws re-paths fixtures, and a golden diff is sometimes the only symptom,
so that diff is read before it is regenerated. The same strictness makes the
transcription's numbers loud. A change to the Basic Skill Eligibility box,
for instance, changes how many skill rolls a term asks for, and the fixtures
fail on their dice before any predicate is reached.

The Jamison reproduction of pp. 23–25 is the one check written from someone
else's numbers. It now runs on the same scripted roller as everything else.
The page narrates a 2D throw by its total, so the script writes each total
as two dice, the first as high as the total allows, and a decider answers
with the choices p. 24 records. Where the example's own arithmetic slips,
the script follows the stated total. The result differs from p. 25 in exactly
the benefit E015 accounts for.

## The record and its projections

`render` turns a `Character` into a `record` struct with JSON tags and
encodes it. Everything a reader sees afterwards is made from those bytes.
Since #363 there is one route. `Sheet` and `Transcript` take a record's
bytes and nothing else, whether it was generated a moment ago or written
last year. `new --sheet` encodes the character and renders the sheet from
what it encoded, as `render` does a saved record. Before, the sheet had a
route from the character and a route from the bytes, and a round-trip test
existed only to hold the two equal. Now `TestRenderReadsARecordBack` compares
`new --sheet` with `render` of the record `new` wrote, and the goldens render
their sheets and transcripts from the record they pin.

`decode` reads every event once, before anything is rendered (#362), and
refuses a record with no UPP, no ruleset or an event that will not read.
Reading is the only thing that can fail, so the sheet's and transcript's
helpers cannot. Decoding rebuilds no domain type. A sheet is a projection of
the record, and rebuilding the sums would add nothing but decoders that can
fail.
`JSON` and `JSONLine` are one encoder, indented or not, and it returns
`json`'s error for the reason the previous section gives.

The one projection not made from bytes is the live view. `EventLine` renders
each event as the engine logs it, for the person answering questions, and
there is no record yet to read. It reaches the transcript's `writeEvent`
through the event codec, which builds the flat reading shape by hand beside
the bytes it writes. `TestTheFlatShapeIsWhatWasWritten` holds that hand-built
shape to what unmarshalling the bytes gives back, one example per kind.

A `Character` never outlives the process that generated it. Nothing decodes
into one, and since #199 nothing outside the module can build one. #369
leaned on that. The character no longer carries its name beside its inputs,
or the ruleset and policy version beside the constants they always equalled.
`project` writes the name from `Inputs`, and `ruleset` and `policy.version`
from `chargen.Ruleset` and `chargen.PolicyVersion`, so the record names the
encoding build's constants. That is the generating build's only because
encoding follows generation in the same process. `build` stays on the
character, because it is not a constant of the code: the command passes it
through `WithBuild` and a test passes none (ADR 0002).

The record's shape is frozen at `v1.0.0` (#62). `record` counts the shape,
and it stays 1 until the freeze (#96). `ruleset` names the books. `build`
names the writer, comes from `debug.ReadBuildInfo`, and only the command
sets it. A test-generated record carries none, so a golden does not differ
by machine. The hand-written `docs/character.schema.json` is open at the root
and closed in every nested object (#113). Event kinds are closed as well
(#129). A later build can add a top-level section, and a reader still
validates the parts it knows. The schema states each fact once too: a throw's
and a roll's dice share one definition, and the rule that no record holds
both Travellers' Aid and a ship is argued only on the clause that refuses it
(#374). The schema is held to the code in one direction by validating every
golden, and in both directions only for the strategy enums, which no golden
chain covers (#177).

`render` also writes the readings. It writes one in full, every one as an
index, those an id names, and, with `--readings` on a `--history`
transcript, a closing section with the full text of each erratum the record
names. That is the only reason `render` imports `rules`. The text printed for
a record's errata is this build's text, not the writer's. ADR 0004's rule
that a reading may be reworded but never changed into a different decision is
what makes that safe. On a transcript, an id this build does not know is
named, not refused, because the rest of the transcript still belongs to the
record. Asked for by name, an unknown id is refused (`ErrNoReading`). The
sheet still leaves ids out, since it is the page a player keeps.

## The command

`cmd/ctchargen` is thin on purpose, and its seams are about channels and
refusals. The record goes to stdout. Questions and complaints go to stderr,
so `ctchargen new | jq` pipes a record and not a conversation. Every failure
exits 1 (#121). The difference between misuse (`errUsage`) and a search
that could not finish (`errSearch`) shows up in the message only. A stray
positional word is refused, because the flag package stops at it and leaves
the flags after it unparsed. A word after a help flag is refused the same
way, worded once (#372). A path that exists is never overwritten without
`--force`. `batch` declares no `--auto`, so `batch --auto` is the flag
package's usage error and not a flag accepted and ignored. `batch` writes
the dead like anyone else, and `--survivors` skips past them without
rerolling them, up to a bound on how many it generates.

The interactive `player` is a `Decider` that reads numbers. It never invents
an answer: input that runs out is `errNoAnswer`, not a default. ADR 0003
decided its tests go through the command and not through the player's
methods. The thing a person notices breaking is the wiring (stderr against
stdout, the line saying nothing was written), and only the command's tests
can see that. The flags, types and defaults are typed out in `cli_test.go`
and held one way only (#126): a new flag breaks nobody, and a changed or
removed one does. #336's removal of `batch --auto` is recorded beside the
baseline it changes, as a break taken deliberately before the CLI freezes.

## What the gate holds, now that checks are tests

There is no CI (#149). The gate is a local `task`: module tidiness, lint
(which runs every `go vet` analyzer and the formatter), nilaway,
govulncheck, the tests under `-race`, and the non-Go files' format and
links. #377 dropped the `vet`, `build` and `clean` tasks, each covered by
another or doing nothing. ADR 0004 decided that no markdown file is parsed,
generated or held to the code. ADR 0005 finished the same move for data: no
file of any kind is read at load. What used to be a committed
`docs/COVERAGE.md` is now `internal/coverage`, a test-only package that
writes nothing. It walks every Go doc comment, package doc, fixture-roster
comment and schema `$comment` for `Implements: <rule> [<authority>].` and
`Holds: <rule>.` markers. It fails on a rule implemented and not held, a
rule held and not implemented, two implementations citing different pages,
a marker found anywhere else, or a cite outside a closed grammar. The grammar
allows Books 1–3 pages with any readings in parentheses, a reading alone, an
ADR named by its file, or one of eight promise labels. `-print` writes the
map as one table, on demand.

The promise labels ("Choice points", "Generation record", "JSON
conventions" and the rest) are the section names of the v1 contract, which
has since been removed from the tree. The harness defines a label as the
section a promise was first stated under, and #355 ruled on that definition
to keep the lift's determinism promise under "JSON conventions" although no
JSON is lifted. But at least seven promises were first written after the
contract was removed, and their labels were chosen by topic (#391). So the
labels are used in two senses, and the harness checks only that a label
exists. That boundary is a historical accident now bearing load: it is the
vocabulary a promise has to be cited in. CODING_STANDARDS.md says where a
marker goes. It goes on the declaration that carries the rule, never on a
test. A promise the harness keeps goes on the test helper or roster that
carries it. A promise no single declaration carries goes on its package doc.

Coverage is not measured (#198). The mutation rule does that job instead: a
new invariant counts as done only when `go run ./internal/mutate` shows a
deliberate edit killed and quotes what the failure said. The tool edits a
scratch copy, never the tree. Since #359 it builds the copy first, as
`go test` would, vet included, and reports a mutation that does not build as
`NOT BUILT` rather than `KILLED`. In the work on #345–#354, several
mutations had read as killed when nothing was tested. #376 kept the build
step's `go build ./...`, at a few seconds per mutation, because without it a
package with no test file falls back to that misleading result. Each rc.3
change that added or rewrote a check listed its mutations in its commit, and
some survived before the change: no test held the record's name, the wording
of the help refusal, or the default policy the flags apply.
`task fix` is not safe to run blind in a tree that quotes a primary source.
`godot` and `dupword` have both damaged a verbatim quotation before, and the
transcription is a large quotation in Go.

## What it is shaped to accommodate, and what would need rethinking

**A new reading** is cheap and well guarded. It needs a constant, an entry
in `Errata`, a `Reading` value, a stamping literal at the outcome it governs,
a predicate written from its prose, a fixture that reaches it, and an
`Implements:` marker citing it. The compiler or a test refuses most of those
when they are left out: the compiler a missing reading, the `Errata` test a
missing entry, `oneConditionEach` a missing predicate, the predicate a missing
stamp, and `unreached` a missing fixture. Review keeps the predicate's
fidelity to the prose, and the marker: the coverage check refuses a cite to a
reading that does not exist, but not a reading nothing cites.

**A correction to a transcribed cell** goes in two places from one fresh
visual reading of the page: the literal in `transcription.go` as printed,
and the second transcription in normalized words. If the corrected cell
changes a lifted value, fixtures may re-path, and the golden diff is read
before it is regenerated. If the page prints something the lift cannot
read, such as a new abbreviation or a new dash, the lift gets a named
decision. If that decision is not settled by the page, it is a reading.
Editing the transcription toward what the lift happens to accept, rather
than toward the page, is the damage ADR 0005 exists to prevent.

**A new choice point** is also designed for. Add a `Decider` method and the
compiler takes the maintainer through the policy, the logging decorator, the
player and the test deciders. A yes-or-no point costs `logging` a method
that calls `yesOrNo`. Two costs come with it. Each strategy has to be given
an answer. And the schema's `point` enum is closed (#123), so after `v1.0.0`
a new choice point is a change to the record's shape. A new strategy is
cheap the same way: it bumps `PolicyVersion` only if it changes an existing
answer, and the schema's strategy enums, which a test holds to the code, are
closed too.

**A new case of a sum** is a compile error in every fold, which is the
point. From `v1.0.0`, though, a case that reaches the record is a shape
change, so it also bumps `record` and loosens a schema object the freeze
closed. A case the page does not print is the opposite defect, and #364 shows
it hides well: it compiles everywhere and is never built.

**A new rendering** is a function of the record's bytes. One written from a
`Character` would bring back the second route #363 removed, and with it a
test whose only job is to hold the two routes equal.

**Withdrawing a reading** has a form that admits a replacement (#332), and
still no precedent.

**Another edition, or Books 4+**, would mean rethinking things at the
foundation. `Ruleset` is a constant, the schema pins it with `const`, the
alphabets are Book 1's, the transcription's array bounds are Book 1's six
services and four skill tables, and the authority model forbids opening
those books. A maintainer adding a Mercenary prior-service table would not
be extending this tool. He would be founding a sibling.

**Replay** was withdrawn and can come back only as a re-walk of a record,
never as a seed (ADR 0001). The record logs every die in the order it was
drawn, so a re-walk is possible. A maintainer who adds a seed "for debugging"
brings back the parts the ADR listed as having each needed its own repair.

The damage a maintainer without the theory is most likely to do is quiet
damage. He might read a table cell from extracted text. He might "tidy" a
cell into the domain's spelling, so that `"Intel 8+"` becomes
`"Intelligence 8+"` or `"Rifl3-1"` becomes `"Rifle-1"`, which moves a
reading out of the lift and into his head. He might type a hyphen for an
em dash, write a predicate by looking at the stamping site, or regenerate a
golden without reading its diff. He might let a decider throw a die, settle
a one-answer question inside a decider instead of at its site, keep a copy of
a fact beside the sum that already says it, or print the policy's choices as
if they were the player's. Some of those the lift or a test now refuses (the
dashes, a throw not made of the next two dice). The rest leave the gate green.
The tree's defences against them are conventions written in CLAUDE.md and
CODING_STANDARDS.md, held by review.

## Uncertainties

- **Authority point 6 names two kinds of number, and the transcription holds
  a third.** The rule sends a number that indexes a printed table to the
  transcription and a bare procedural bound to a constant. The transcription
  also holds amounts and targets the page states in prose and that index
  nothing: the medical crisis's 8+, its recovery to 1 and its one die of
  months, the two mustering DMs, CR 2,000 per term past eight. One of them,
  the education gate's 8+, reads like the rank thresholds the rule sends to
  constants: a threshold that opens something. It may be held as the page's
  text because the fourth table's name, as the domain spells it, carries
  "(education 8+)". The code does not say, and #43 named the gate as data
  without the change that closed it deciding where it belongs.
- **Whether the live view's second route is deliberate.** #363 made every
  rendering of a record come from its bytes. `EventLine` still builds its
  reading shape field by field in the codec, held to the bytes by a test with
  one example per kind. Unmarshalling the bytes the codec already wrote would
  remove the hand mapping and its test. I can't tell whether the hand-built
  shape is kept on purpose or is outside the scope #363 set itself.
- **Some printed text is held and not checked.** The Intelligence row's
  `"no effect before age 66"` is accepted as one of two no-effect phrases.
  Nothing checks that its span ends where the age-66 column begins. The
  bands come from the spans, so the span is what governs, and the second
  transcription holds the bands. I read the phrase as being kept for
  fidelity, not as being read, but nothing says so.
- **The withdrawn-reading form is tested only by a stand-in.** #332's
  resolution makes `verdict` refuse only a stamp. A withdrawn erratum is
  still never refused at generation time. ADR 0004 assigns that guarantee to
  the harness, so it holds only over the fixtures' records, and a random
  career that walks a path no fixture walks is not covered. I infer this is
  deliberate.
- **Whether `render` should own the readings' text is documented, not
  argued.** Putting it there follows "render is what a reader sees". It also
  means a record is explained in the rendering build's words. ADR 0004 makes
  that safe by forbidding a reworded decision, and that rule holds only from
  `v1.0.0`. Records written before then are explained in whatever words the
  build rendering them carries.
- **"Record" is used in two senses.** The glossary reserves it for the JSON.
  `Character`'s doc calls the character "the record a generation produces",
  and the log's calls itself "the generation record". Since #363, the
  glossary's sense is the one the code acts on: nothing renders from a
  `Character` but the encoder, and the live view renders its events one at
  a time. I take the other uses as older wording, not as a second concept.
- **E002 is stamped on every skill outcome, but its predicate is "a survival
  throw passed".** These agree because every surviving term trains at least
  one skill. Zeroing the eligibility box, shown with `internal/mutate`, fails
  loudly, but through a fixture's script running out of dice before any
  predicate is judged. Whether the predicate alone would catch the
  disagreement I have not shown.
- **The rules I call "conventions held by review"** (predicates from prose,
  golden diffs read before regenerating, marker placement, cells typed as
  printed) are ones I found no mechanical check for. One may exist in a test
  I did not read closely. I read the fixture roster, the walks,
  `malformed_test.go`, `tables_test.go`, the interactive player and the
  command's output and usage files only in part.
- **I did not open the PDFs.** Every claim here about what a page prints
  comes from the transcription, the code's own quotations and the readings'
  prose, not from the page.
