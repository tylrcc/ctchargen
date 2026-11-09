# A theory of ctchargen

> Release snapshot for `v1.0.0-rc.2`. It describes that tag, not `main`.
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
`"Rifl3-1"` as the reprint shows it. Every lifted value was proven
unchanged across the move. That is why rc.2's goldens are rc.1's.

The split ADR 0005 draws is that **structure comes from the domain and cells
come from the page**. `transcription.go`'s header note says which is which.
Where the page's own text carries a layout, the transcription keeps it: the
Aging Table's spans, the DM rows' headings. Where only position carries it,
the position is keyed by `traveller`'s types: which table a grid is, which
service a column belongs to, which ship a hull is. So the skills tables are
`[len(traveller.SkillTables)][Faces][len(traveller.ServiceNames)]string`,
and a table the type does not name, a table given twice, or one cell too
many does not compile. At rc.1 the lift had to tolerate unknown JSON keys,
so a misspelled one left its field at zero (#317). Now a misspelled field
does not build. Go brings one
hazard of its own: a short array is padded with `""` without a word. So
`refuseEmpty` refuses any empty cell, on the ground that the page prints a
dash where it means nothing.

**The lift is kept on purpose.** ADR 0005's rejected alternative was to type
the tables as domain values directly and delete the lift. A broken glyph
would then fail to compile. But every normalization and every dash would be
decided in the transcriber's head, with no code to name it, and that is a
reading applied silently. The lift is where those judgements are written
down. Each one is named in code:

- `abbreviatedCharacteristics` maps the six abbreviations. It is the one
  list `parse.go` keeps beside the domain's alphabets, and its doc argues
  why that is settled by the page and is not a reading.
- E012's normalizations live in the transcription beside the table they
  govern. `Rifl3` → `Rifle` is among them. It is a font-trap glyph caught
  by the semantic cross-check CLAUDE.md describes, and recorded as a
  reading, never silently corrected.
- `boxServices` maps p. 23's spelling of the services (`Marine`, `Scout`,
  `Merchant`) under E012.
- `emDash` decides the three dashes once. The em dash means nothing. A lone
  hyphen or en dash where the page prints an em dash is refused by name,
  because a cell typed with one was not read off the page. A hyphen opening
  a number is a minus (`traveller.Sign`).

`rules.Load` lifts once per process through `sync.OnceValues` and keeps its
error, because a literal can still hold text the lift refuses. The lift
refuses shape, not value: a name outside `traveller`'s closed alphabets, a
target in a notation pp. 2–3 do not use, a rank printed below a dash, a DM
beside a dashed throw, an Aging cell that leaves a gap or runs past the last
printed term. A few of its checks do reach values, but only where the domain
itself fixes the number. The Age header must read
`traveller.StartingAge + traveller.Years × term`, which is why `StartingAge`
moved from `chargen` into `traveller`: `rules` cannot import `chargen`. A
box label's rank title must be one its service's Table of Ranks column
prints. Those are cross-checks between pages, not a second list. Anything
else about value is the second transcription's job.

The second transcription, in the `rules` tests, did not change. It is laid
out apart from the page's grid and written in the normalized vocabulary
(`"+1 Intelligence"`, `"weapon Blade Combat"`). It is independent of the
first in shape and in words: the first walks a row of the page, six services
across, and the second walks a service down its column. Two cells swapped
within a row compile and lift like any other cells, and only the second copy
notices. One table does not get the row-against-column half of that
independence. The Table of Ranks is held a service's column at a time in
both copies (#354).

CLAUDE.md's Authority point 6 sets which numbers are in the transcription. A
number that indexes a printed table is there. The Aging Table's last printed
term is one, now read off the term header row the page prints rather than a
field. A bare procedural bound is a Go constant with its cite, placed beside
the code that applies it. The three-roll cap on Mustering Out Table 2 and the
rank that opens Table 1's +1 are examples (`maxRollsOnTableTwo`,
`minRankForTable1Modifier` in `chargen/muster.go`). The rule moves numbers in
both directions, so a number on the wrong side of it is a defect, not a style
choice.

The cost ADR 0005 accepted is that linters now read the page's text.
`goconst` is excluded for `transcription.go` and the `rules` tests, because
collapsing a repeated cell into one constant lets one typo reach every cell
that shares it. `mnd` is excluded for `transcription.go`, because a number
there is the page's text. Any linter added later that reads string literals will read the cells,
and a `task fix` over this file is as unsafe as it is over a verbatim
quotation.

## Readings, errata and stamping

A reading has three expressions in the tree. They are kept apart on purpose,
because the check between them only works if they stay independent.

The **prose** is a `rules.Reading` value in `internal/rules/readings.go`. Its
fields are title, pages, what the page says, the gap, the reading, why, and
the stamping condition, all kept verbatim with the page's emphasis as inline
markdown. `rules.Readings()` returns `[len(traveller.Errata)]Reading`, so a
reading missing from the array, or one past the last erratum, does not
compile. `TestTheReadings` holds each value to its index. The gap at rc.1
between the `Errata` array and the `const` block it lists is closed:
`TestErrataListsEveryConstant` parses `traveller`'s own source for constants
of type `Erratum` and requires `Errata` to be exactly those, in order,
numbered from 1 (#333). The chain is now const block → `Errata` (a test) →
readings (the compiler). Since ADR 0004, a malformed reading no longer stops
the binary, because the compiler and the gate stop it first.

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
withdrawn. Since #332, `verdict` holds a withdrawn reading to one thing
only: no golden stamps it. Its condition is no longer judged. Its predicate
is still written from its prose, and a reading replaced by another still
describes the records its replacement governs. So the rc.1 rule that its
condition fit no golden would have turned the gate red at the first real
withdrawal. `TestAWithdrawnReadingOnAGoldenIsRefused` stands E001 in as
withdrawn under its real predicate. No reading has actually been withdrawn,
but the form no longer refuses its own main use.

## The domain is sums, so the compiler counts the cases

`traveller` imports nothing in this module, and nothing in it rolls, reads a
table or marshals. Its rule for what becomes a type is that types carry
identity, never rule invariants. A service outside the six does not exist.
A range a page prints is not a type. The few bare numbers it now carries
(`Years`, `StartingAge`) are domain facts that two packages need, and placing
them here is what the import arrows allow.

Wherever the book says "exactly one of", the code has a sealed interface
with a `Fold` method over a cases interface that has one method per case.
This covers `Enlistment` (enlisted, drafted, declined the draft),
`TableResult`, `BenefitRow`, `Departure`, `WeaponBenefit` and `Event`. Go
does not check a type switch for exhaustiveness, so a fold is how adding a
case breaks every consumer at compile time: the JSON codecs in `render`,
the death test `traveller.Fatal`, the live view, and the second
transcription's `describer`. The plain enums are a weaker strength. The
exhaustive linter catches a missing `switch` case, and only under `task`.
The package doc spells out the difference, and it is real: dropping the
linter drops the plain enums' guarantee, and nothing can drop the folds'.

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
(#48).

Who answered is fixed when the run is entered, not carried per event.
`chargen.Auto` records an `AutoPolicy`: the strategies and `PolicyVersion`.
`chargen.Guided` records none, and that absence is the record's statement
that a person made every choice. An auto run is `new --auto` or any batch:
since #336 a batch takes no `--auto`, because it has nobody else to ask, and
the policy is no longer equated with a flag anywhere. `PolicyVersion`
changes only when an answer changes, and it is currently 3. A new choice
point does not bump it, because no earlier record was answered differently.
`build` changes with every commit and cannot do this job. The policy's
documentation is its own method doc comments: each says when the engine asks
and what each strategy answers, and nothing else describes the auto run.

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
so that diff is read before it is regenerated. The same strictness makes the
transcription's numbers loud. A change to the Basic Skill Eligibility box,
for instance, changes how many skill rolls a term asks for, and the fixtures
fail on their dice before any predicate is reached. The Jamison reproduction
of pp. 23–25 is the one check written from someone else's numbers: a
narrated decider and roller walk the example's character. It follows each
stated total where the example's own arithmetic slips, and differs from
p. 25 in exactly the benefit E015 accounts for.

## The record and its projections

`render` turns a `Character` into a `record` struct with JSON tags, and
writes every output from that struct. That includes outputs for a character
generated a moment earlier: `Sheet` and `Transcript` project first, and
`SheetFrom` and `TranscriptFrom` decode a saved record into the same struct.
So `ctchargen render` cannot disagree with what `new` printed. Decoding
rebuilds no domain type. A sheet is a projection of the record, and
rebuilding the sums would add nothing but decoders that can fail. JSON is
now only the record's wire. Since ADR 0005 nothing in the tree lifts JSON
into rules, so "the wire" in `render`'s comments means the record and
nothing else.

The record's shape is frozen at `v1.0.0` (#62). `record` counts the shape,
and it stays 1 until the freeze (#96). `ruleset` names the books. `build`
names the writer, comes from `debug.ReadBuildInfo`, and only the command
sets it. A test-generated record carries none, so a golden does not differ
by machine (ADR 0002). The hand-written `docs/character.schema.json` is
open at the root and closed in every nested object (#113). Event kinds are
closed as well (#129). A later build can add a top-level section, and a
reader still validates the parts it knows.

`render` also writes the readings. It writes one in full, every one as an
index, those an id names, and, with `--readings` on a `--history`
transcript, a closing section with the full text of each erratum the record
names. This is the only reason `render` imports `rules`, and its package doc
and CLAUDE.md's arrows now say so (#334). The text printed for a record's
errata is this build's text, not the writer's. ADR 0004's rule that a
reading may be reworded but never changed into a different decision is what
makes that safe. On a transcript, an id this build does not know is named,
not refused, because the rest of the transcript still belongs to the record.
Asked for by name, an unknown id is refused (`ErrNoReading`). The sheet
still leaves ids out, since it is the page a player keeps.

## The command

`cmd/ctchargen` is thin on purpose, and its seams are about channels and
refusals. The record goes to stdout. Questions and complaints go to stderr,
so `ctchargen new | jq` pipes a record and not a conversation. Every failure
exits 1 (#121). The difference between misuse (`errUsage`) and a search
that could not finish (`errSearch`) shows up in the message only. A stray
positional word is refused, because the flag package stops at it and leaves
the flags after it unparsed. A path that exists is never overwritten without
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

There is no CI (#149). The gate is a local `task`. ADR 0004 decided that no
markdown file is parsed, generated or held to the code. ADR 0005 finished
the same move for data: no file of any kind is read at load. What used to be
a committed `docs/COVERAGE.md` is now `internal/coverage`, a test-only
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
load-bearing: it is the vocabulary a promise has to be cited in. The harness
checks that a label exists, not that it fits. Two lift promises still cite
"JSON conventions" from the time when the tables were JSON (#355).
CODING_STANDARDS.md says where a marker goes. It goes on the declaration
that carries the rule, never on a test. A promise the harness keeps goes on
the test helper or roster that carries it. A promise no single declaration
carries goes on its package doc.

Coverage is not measured (#198). The mutation rule does that job instead: a
new invariant counts as done only when `go run ./internal/mutate` shows a
deliberate edit killed and quotes what the failure said. The tool edits a
scratch copy, never the tree. Every rc.2 refusal in the lift (the dashes, the
box labels, the Aging spans, the DM headings) shipped with its mutations
listed in its commit. `task fix` is not safe to run blind in a tree that
quotes a primary source. `godot` and `dupword` have both damaged a verbatim
quotation before, and the transcription is now a large quotation in Go.

## What it is shaped to accommodate, and what would need rethinking

**A new reading** is cheap and well guarded. It needs a constant, an entry
in `Errata`, a `Reading` value, a stamping literal at the outcome it governs,
a predicate written from its prose, a fixture that reaches it, and an
`Implements:` marker citing it. Since #333, the compiler or a test refuses
each of those when it is left out, except the predicate's fidelity to the
prose and the fixture's existence, which review and `namesNoRecord` keep.

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
player and the test deciders. Two costs come with it. Each strategy has to
be given an answer. And the schema's `point` enum is closed (#123), so after
`v1.0.0` a new choice point is a change to the record's shape. A new
strategy is cheap the same way: it bumps `PolicyVersion` only if it changes
an existing answer, and the schema's strategy enums, which a test holds to
the code, are closed too.

**A new case of a sum** is a compile error in every fold, which is the
point. From `v1.0.0`, though, a case that reaches the record is a shape
change, so it also bumps `record` and loosens a schema object the freeze
closed.

**Withdrawing a reading** now has a form that admits a replacement (#332),
and still no precedent.

**Another edition, or Books 4+**, would mean rethinking things at the
foundation. `Ruleset` is a constant, the schema pins it with `const`, the
alphabets are Book 1's, the transcription's array bounds are Book 1's six
services and four skill tables, and the authority model forbids opening
those books. A maintainer adding a Mercenary prior-service table would not
be extending this tool. He would be founding a sibling.

**Replay** was withdrawn and can come back only as a re-walk of a record,
never as a seed (ADR 0001). A maintainer who adds a seed "for debugging"
brings back the parts the ADR listed as having each needed its own repair.

The damage a maintainer without the theory is most likely to do is quiet
damage. He might read a table cell from extracted text. He might "tidy" a
cell into the domain's spelling, so that `"Intel 8+"` becomes
`"Intelligence 8+"` or `"Rifl3-1"` becomes `"Rifle-1"`, which moves a
reading out of the lift and into his head. He might type a hyphen for an
em dash, write a predicate by looking at the stamping site, or regenerate a
golden without reading its diff. He might let a decider throw a die, settle
a one-answer question inside a decider instead of at its site, or print the
policy's choices as if they were the player's. Some of those the lift now
refuses (the dashes). The rest leave the gate green. The tree's defences
against them are conventions written in CLAUDE.md and CODING_STANDARDS.md,
held by review.

## Uncertainties

- **How far "structure from the domain" extends is a judgement made table by
  table.** The header note in `transcription.go` keys the Table of Ranks by
  service, which puts it against the page's rows and against the
  second transcription's direction (#354). I can't tell from the code
  whether the authors see the row-versus-column independence as a property
  every table must have, or as the usual case.
- **Some printed text is held and not checked.** The Intelligence row's
  `"no effect before age 66"` is accepted as one of two no-effect phrases.
  Nothing checks that its span ends where the age-66 column begins. The
  bands come from the spans, so the span is what governs, and the second
  transcription holds the bands. I read the phrase as being kept for
  fidelity, not as being read, but nothing says so.
- **The withdrawn-reading form is tested only by a stand-in.** #332's
  resolution makes `verdict` refuse only a stamp. A withdrawn erratum is
  still never refused at generation time. ADR 0004 assigns that guarantee to
  the harness, so it holds only over the goldens, and a random career that
  walks a path no fixture walks is not covered. I infer this is deliberate.
- **Whether `render` should own the readings' text is now documented, not
  argued.** Putting it there follows "render is what a reader sees". It also
  means a record is explained in the rendering build's words. ADR 0004 makes
  that safe by forbidding a reworded decision, and that rule holds only from
  `v1.0.0`. Records written before then, by betas, get rc.2's wording.
- **The promise labels are a closed set of section names from a document
  that no longer exists in the tree.** I treat them as a stable vocabulary.
  The source does not say whether the authors see them as permanent or as
  scaffolding left over from the PRD, and #355 shows that nothing holds a
  label to the promise it is attached to.
- **E002 is stamped on every skill outcome, but its predicate is "a survival
  throw passed".** These agree because every surviving term trains at least
  one skill. I checked by mutation that zeroing the eligibility box fails
  loudly. It does, but through the fixtures' scripts running out of dice,
  not through the predicate. Whether the predicate alone would catch the
  disagreement I have not shown.
- **The rules I call "conventions held by review"** (predicates from prose,
  golden diffs read before regenerating, marker placement, cells typed as
  printed) are ones I found no mechanical check for. One may exist in a test
  I did not read closely. The fixture and walk files, most of `policy.go`
  past its strategy types, and `malformed_test.go` past its index I read
  only in part.
- **I did not open the PDFs.** Every claim here about what a page prints
  comes from the transcription, the code's own quotations and the readings'
  prose, not from the page.
