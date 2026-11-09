# A theory of ctchargen

> Release snapshot for `v1.0.0-rc.1`. It describes that tag, not `main`.
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
fixture, and a script is not a seed.

The authority model ranks first and outranks everything else. The © 1977
text governs. Memory of later editions does not. A reading decided here
governs only where the page is silent. The errata of the implementation
deleted at `41a213a` are a last resort, and even then only a pointer to a
page that has to be reread. Old code and old transcriptions never come back.
A maintainer who leaves this model out of his own understanding will "fix"
something into the 1981 rules. The best-known trap is survival failure: here
it is death, with no injury option (p. 5).

## The page, and why it is read twice

The reprints' embedded font substitutes glyphs, and the substitutions look
like data. A dash cell comes out as the digit 4, a minus comes out as a 3,
and "Travellers' Aid" comes out as `Travellers9`. That one fact explains a
lot of the repository's shape. Every table is transcribed twice from the same
visual reading: once into `internal/rules/data/*.json` row by row, and once
into `internal/rules/transcription_test.go` service by service, down each
column. The rows-versus-columns split is deliberate. It is the only
independence one reading pass can offer.

`rules.Load` lifts the JSON once per process, through `sync.OnceValues`. It
validates each cell's shape and checks every name against the closed
alphabets of `traveller`. A cell that names no service, or a target written
in a notation pp. 2–3 do not use, is a build defect and fails on the first
load. The lift does not check values. A key misspelled in the data leaves
its field at zero, and only the second transcription notices that the page
prints something else. The two copies split their jobs: the lift holds
shape, and the tests hold value. A change that makes the lift check values
from a second list kept beside it repeats the copy the tests already are.

CLAUDE.md's Authority point 6 sets which numbers are data. A number that
indexes a printed table belongs in the data. The Aging Table's last printed
term is one: it picks a column. A bare procedural bound is a Go constant
with its cite, placed beside the code that applies it. The three-roll cap on
Mustering Out Table 2 and the rank that opens Table 1's +1 are examples
(`maxRollsOnTableTwo`, `minRankForTable1Modifier` in `chargen/muster.go`).
The rule moves numbers in both directions, so a number on the wrong side of
it is a defect, not a style choice.

## Readings, errata and stamping

A reading has three expressions in the tree. They are kept apart on purpose,
because the check between them only works if they stay independent.

The **prose** is a `rules.Reading` value in `internal/rules/readings.go`. Its
fields are title, pages, what the page says, the gap, the reading, why, and
the stamping condition, all kept verbatim with the page's emphasis as inline
markdown. The values sit in an array the compiler sizes. `rules.Readings()`
returns `[len(traveller.Errata)]Reading`, so a reading missing from the
array, or one past the last erratum, does not compile. `TestTheReadings`
holds each value to its index. The tie runs from the readings to the
`Errata` array, not to the `const` block the array lists. A constant missing
from `Errata` compiles with no reading (#333). Until rc.1 the prose was
`ERRATA.md`, embedded and parsed on every load. ADR 0004 made it code, and
gave up a single consequence: a malformed reading no longer stops the
binary, because the compiler and the gate stop it first.

The **stamping site** is in the engine. An outcome event that a reading
governed carries the erratum as a literal, for example
`[]traveller.Erratum{traveller.E004}` on a death at the survival throw.
`log.stamped` takes the union of those, in id order, as the record's
`errata`. The engine never reads a reading's prose. `chargen` imports
`rules` for tables, and only its tests read the readings. That is why a
reading's wording can change without changing a single character.

The **predicate** lives in test code: one function per reading in
`internal/chargen/stamping_test.go`, written by hand from the stamped-on
prose and never from the stamping site (#66, #77). It runs over every
checked-in golden, in both directions. A reading stamped where its condition
fails is as much a defect as one missing where the condition holds, because
a record's errata list is the tool's claim about what the character rests
on. A failure quotes the prose through `rules.ReadingOf`, not a copy typed
beside the predicate. This is the same trap as the double transcription
again: a condition copied from the code that does the stamping checks
nothing.

Two readings, E012 and E015, are stamped on no record by design. E012 is a
spelling normalization. E015 matters only to the worked example's
reproduction. `namesNoRecord` pins that set, so a reading that silently stops
appearing is a failure, and so is one that starts.

ADR 0004 also gave an erratum a lifetime. From `v1.0.0`, an erratum names one
decision for good, the decision being both what the reading decides and when
it is stamped. Wording can change under an erratum. A changed decision needs
a new erratum. A decision that no longer holds is withdrawn, never deleted:
the `Withdrawal` field says why and names any replacement, the reading stays
in the array so old records still resolve, and `render.Reading` marks it
withdrawn. Nothing at generation time refuses a withdrawn erratum. The
harness's `verdict` refuses any golden that stamps one, or that its
predicate fits. No reading is withdrawn yet, and the form has never been
used for real. As built, it collides with the predicate rule for any
withdrawal whose replacement governs the same records (#332).

## The domain is sums, so the compiler counts the cases

`traveller` imports nothing in this module, and nothing in it rolls, reads a
table or marshals. Its rule for what becomes a type is that types carry
identity, never rule invariants. A service outside the six does not exist.
A range a page prints is not a type.

Wherever the book says "exactly one of", the code has a sealed interface
with a `Fold` method over a cases interface that has one method per case.
This covers `Enlistment` (enlisted, drafted, declined the draft),
`TableResult`, `BenefitRow`, `Departure`, `WeaponBenefit` and `Event`. Go
does not check a type switch for exhaustiveness, so a fold is how adding a
case breaks every consumer at compile time: the JSON codecs in `render`,
the death test `traveller.Fatal`, the live view. The plain enums are a
weaker strength. The exhaustive linter catches a missing `switch` case, and
only under `task`. The package doc spells out the difference, and it is real:
dropping the linter drops the plain enums' guarantee, and nothing can drop
the folds'.

Some modelling choices only make sense against a sibling design that did it
differently. The civilian who declined the draft is a case of `Enlistment`,
not a nil service with a flag beside it. The book's single word "died"
becomes two `Departure` cases, because a survival death and a medical-crisis
death carry different fields. `Fatal` lives in `traveller` because it was
once stated twice and the two copies could disagree (#100). `Age` carries
months only because a survived medical crisis adds 1D of them, and
`PlusMonths` floors toward negative infinity to protect its one invariant.
`RollEvent` is separate from `ThrowEvent` so that a roll with nothing to
meet is never written as `"succeeded": true` (#50).

## The engine asks, and the record says who answered

`chargen` walks the procedure and decides nothing on its own behalf. Every
point where the book lets a player choose is a method on `Decider`. Each
method is typed to its own alphabet, and that typing is what closes the set.
A new choice point is a new method, and every decider stops compiling. The
`ChoicePoint` enum is unavoidably a second list. A reflection test holds it
to the interface's methods, so the two cannot drift.

Three deciders exist: the `--auto` `Policy`, the interactive `player` in
the command, and the tests' scripted deciders. The engine cannot tell them
apart. Each one is reached through `logging`, the decorator `generate`
wraps around whatever it is handed. Being the only path in, it can make
promises no single decider has to keep:

- a question with fewer than two legal answers is refused before it is asked
  (`asked`), so the record never shows a choice nobody could have made
  differently;
- each slice a decider is handed is its own copy, so writing to an offer
  changes nothing the engine offers later (#55, #137, #230);
- an answer outside the offer is refused, not applied (`errNotOffered`).

That last check compares rendered names. It is sound only while `String()`
is injective on every alphabet, and a test in `traveller` holds exactly that
(#48).

Who answered is fixed when the run is entered, not carried per event.
`chargen.Auto` records an `AutoPolicy`: the strategies and `PolicyVersion`.
`chargen.Guided` records none, and that absence is the record's statement
that a person made every choice. `PolicyVersion` changes only when an answer
changes, and it is currently 3. A new choice point does not bump it, because
no earlier record was answered differently. `build` changes with every
commit and cannot do this job. The policy's documentation is its own method
doc comments: each says when the engine asks and what each strategy answers,
and nothing else describes `--auto`.

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

A **script** is a path's dice and draws in the order the procedure asks for
them. `internal/scripted` plays one strictly in both directions: asking past
the end fails naming the roller, and a die left unread fails at `Finished`.
A **fixture** is a script with a checked-in golden. A **walk** is the
command tests' script, extracted from a record, with no golden. The order
in which dice are consumed matters to every script. A change that reorders
throws re-paths fixtures, and a golden diff is sometimes the only symptom,
so that diff is read before it is regenerated. The Jamison reproduction of
pp. 23–25 is the one check written from someone else's numbers: a narrated
decider and roller walk the example's character. It follows each stated
total where the example's own arithmetic slips, and differs from p. 25 in
exactly the benefit E015 accounts for.

## The record and its projections

`render` turns a `Character` into a `record` struct with JSON tags, and
writes every output from that struct. That includes outputs for a character
generated a moment earlier: `Sheet` and `Transcript` project first, and
`SheetFrom` and `TranscriptFrom` decode a saved record into the same struct.
So `ctchargen render` cannot disagree with what `new` printed. Decoding
rebuilds no domain type. A sheet is a projection of the record, and
rebuilding the sums would add nothing but decoders that can fail.

The record's shape is frozen at `v1.0.0` (#62). `record` counts the shape,
and it stays 1 until the freeze (#96). `ruleset` names the books. `build`
names the writer, comes from `debug.ReadBuildInfo`, and only the command
sets it. A test-generated record carries none, so a golden does not differ
by machine (ADR 0002). The hand-written `docs/character.schema.json` is
open at the root and closed in every nested object (#113). Event kinds are
closed as well (#129). A later build can add a top-level section, and a
reader still validates the parts it knows.

Since #324, `render` also writes readings. `--readings` on a `--history`
transcript ends the transcript with the full text of each erratum the record
names. `ctchargen readings` lists every reading with its status, or prints
the ones named. This adds a `render` → `rules` dependency. The text printed
for a record's errata is this build's text, not the writer's. ADR 0004's rule
that a reading may be reworded but never changed into a different decision
is what makes that safe. An
id this build does not know is reported, not refused, because the rest of
the transcript still belongs to the record. The sheet still leaves ids out,
since it is the page a player keeps. The package doc and CLAUDE.md's tree do
not yet describe this responsibility (#334).

## The command

`cmd/ctchargen` is thin on purpose, and its seams are about channels and
refusals. The record goes to stdout. Questions and complaints go to stderr,
so `ctchargen new | jq` pipes a record and not a conversation. Every failure
exits 1 (#121). The difference between misuse (`errUsage`) and a search
that could not finish (`errSearch`) shows up in the message only. A stray
positional word is refused, because the flag package stops at it and leaves
the flags after it unparsed. A path that exists is never overwritten without
`--force`. `batch` writes the dead like anyone else, and `--survivors` skips
past them without rerolling them, up to a bound on how many it generates.

The interactive `player` is a `Decider` that reads numbers. It never invents
an answer: input that runs out is `errNoAnswer`, not a default. ADR 0003
decided its tests go through the command and not through the player's
methods. The thing a person notices breaking is the wiring (stderr against
stdout, the line saying nothing was written), and only the command's tests
can see that. The flags, types and defaults are typed out in `cli_test.go`
and held one way only (#126): a new flag breaks nobody, and a changed one
does.

## What the gate holds, now that checks are tests

There is no CI (#149). The gate is a local `task`. ADR 0004's last move was
that no markdown file is parsed, generated or held to the code. What used to
be a committed `docs/COVERAGE.md` is now `internal/coverage`, a test-only
package that writes nothing. It walks every Go doc comment, package doc,
fixture-roster comment and schema `$comment` for `Implements: <rule>
[<authority>].` and `Holds: <rule>.` markers. It fails on a rule implemented
and not held, a rule held and not implemented, two implementations citing
different pages, a marker found anywhere else, or a cite outside a closed
grammar. The grammar allows Books 1–3 pages with any readings in
parentheses, a reading alone, an ADR named by its file, or one of eight
promise labels. `-print` writes the map as one table, on demand.

The promise labels ("Choice points", "Generation record", "JSON
conventions" and the rest) are the section names of the v1 contract, which
has since been removed from the tree. They were kept as labels for the
tool's own promises. That boundary is a historical accident, and it is now
load-bearing: it is the vocabulary a promise has to be cited in.
CODING_STANDARDS.md says where a marker goes. It goes on the declaration that
carries the rule, never on a test. A promise the harness keeps goes on the
test helper or roster that carries it. A promise no single declaration
carries goes on its package doc.

Coverage is not measured (#198). The mutation rule does that job instead: a
new invariant counts as done only when `go run ./internal/mutate` shows a
deliberate edit killed and quotes what the failure said. The tool edits a
scratch copy, never the tree. `task fix` is not safe to run blind in a tree
that quotes a primary source. `godot` and `dupword` have both damaged a
verbatim quotation before.

## What it is shaped to accommodate, and what would need rethinking

**A new reading** is cheap and well guarded. It needs a constant, an entry
in `Errata`, a `Reading` value, a stamping literal at the outcome it governs,
a predicate written from its prose, a fixture that reaches it, and an
`Implements:` marker citing it. The compiler or the gate refuses most of
those when they are left out. The constant-to-`Errata` link is one
exception (#333).

**A new choice point** is also designed for. Add a `Decider` method and the
compiler takes the maintainer through the policy, the logging decorator, the
player and the test deciders. Two costs come with it. Each strategy has to
be given an answer. And the schema's `point` enum is closed (#123), so after
`v1.0.0` a new choice point is a change to the record's shape. A new `--auto` strategy is cheap the same way: it bumps
`PolicyVersion` only if it changes an existing answer, and the schema's
strategy enums, which a test holds to the code, are closed too.

**A new case of a sum** is a compile error in every fold, which is the
point. From `v1.0.0`, though, a case that reaches the record is a shape
change, so it also bumps `record` and loosens a schema object the freeze
closed.

**Withdrawing a reading** has a form and no precedent, and the first use
will have to settle #332 first.

**Another edition, or Books 4+**, would mean rethinking things at the
foundation. `Ruleset` is a constant, the schema pins it with `const`, the
alphabets are Book 1's, and the authority model forbids opening those books.
A maintainer adding a Mercenary prior-service table would not be extending
this tool. He would be founding a sibling.

**Replay** was withdrawn and can come back only as a re-walk of a record,
never as a seed (ADR 0001). A maintainer who adds a seed "for debugging"
brings back the parts the ADR listed as having each needed its own repair.

The damage a maintainer without the theory is most likely to do is quiet
damage. He might read a table cell from extracted text, write a predicate by
looking at the stamping site, regenerate a golden without reading its diff,
let a decider throw a die, settle a one-answer question inside a decider
instead of at its site, or print the policy's choices as if they were the
player's. Each of those leaves the gate green. The tree's defences against
them are conventions written in CLAUDE.md and CODING_STANDARDS.md, held by
review.

## Uncertainties

- **The withdrawn-reading form is untested against a real case.** I read it
  as requiring a withdrawn reading's prose-derived predicate to fit no
  golden. A replaced reading usually governs the same records as its
  replacement, so I think the form refuses its own main use (#332). The
  authors may instead intend a withdrawn reading's predicate to be rewritten
  as never-true. Nothing written down says so.
- **The guarantee that a withdrawn erratum is never stamped holds only over
  the goldens.** At run time nothing in `chargen` checks `Withdrawn()`. I
  infer that this is deliberate, since ADR 0004 assigns the guarantee to the
  harness, but a random career that walks a path no fixture walks is not
  covered.
- **Whether `render` should own the readings' text is not decided anywhere I
  found.** Putting it there follows "render is what a reader sees". It also
  means a record is explained in the rendering build's words. ADR 0004 makes
  that safe by forbidding a reworded decision, and that rule holds only from
  `v1.0.0`. Records written before then, by betas, get rc.1's wording.
- **The promise labels are a closed set of section names from a document
  that no longer exists in the tree.** I treat them as a stable vocabulary.
  The source does not say whether the authors see them as permanent or as
  scaffolding left over from the PRD.
- **E002 is stamped on every skill outcome, but its predicate is "a survival
  throw passed".** These agree because every surviving term trains at least
  one skill. That depends on the Basic Skill Eligibility box always granting
  one or more per term, which is data in `rules`. I have not checked whether
  a data change there would break the agreement silently or loudly.
- **The rules I call "conventions held by review"** (predicates from prose,
  golden diffs read before regenerating, marker placement) are ones I found
  no mechanical check for. One may exist in a test I did not read closely.
  The fixture and walk files, and most of `policy.go` past its strategy
  types, I read only in part.
- **I did not open the PDFs.** Every claim here about what a page prints
  comes from the code's own quotations and the readings' prose, not from
  the page.
