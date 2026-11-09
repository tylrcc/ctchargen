# Changelog

## Unreleased

Bullets as the work lands. The release pass rewrites this into whatever the
release turns out to be.

- Table 1's benefit sum has the six kinds of row p. 9 prints: passage,
  characteristic, weapon, Travellers' Aid, ship and the dash. It also had a
  cash kind no Table 1 row produces, so the engine's case for it could never
  run; Table 2's cash is a number read beside the row, unchanged (#364).
- Rendering a sheet or a transcript decodes the record's events once, and
  only that reading can fail: the helpers that read the decoded events
  return no error, and a malformed record is refused with the same error.
  `render.JSON` and `render.JSONLine` are one encoder, indented or not, and
  still return json's error: none can arise today, but a record that failed
  to marshal is refused rather than written wrong (#362).
- The sheet and the transcript are rendered only from the record's bytes.
  `render.Sheet` and `render.Transcript` take the record, where there were
  also forms taking the generated character; `new` renders what it encodes,
  as `render` does with a record written earlier, and the test that held the
  two routes equal is gone with the second route. Every sheet and transcript
  is unchanged (#363).
- A 2D throw is two dice in sequence in one place, `chargen`'s `twoDice`,
  where every roller implemented it. `Roller` drops `TwoDice`, so a test
  roller can no longer make a 2D throw that is not the stream's next two
  dice; the worked example's reproduction now runs on the scripted roller
  the fixtures use, its throws still written as the totals the page narrates
  (#376, #377).
- `Character.ServedIn` no longer guards against a character that was never
  enlisted: generation enlists every one, and since #199 nothing outside the
  module can build one. A nil enlistment would now panic, so a generation
  defect shows rather than reading as no service (#376, #377).
- The engine's five yes-or-no choice points record their answer through one
  helper, and the strategy lists in the help are built from the same `names`
  the record's choices are (#376, #377).
- The Taskfile drops `vet`, `build` and `clean`. golangci-lint's govet runs
  every analyzer `go vet` does, `test` compiles every package, and no task
  makes the binary `clean` removed (#376, #377).
- The `--career`, `--skills` and `--muster` defaults are read from
  `chargen.DefaultPolicy`, where they were typed out beside it, and the fixtures generate under `DefaultPolicy`, so a
  default changed in one place alone would have kept every golden passing
  under a policy the tool no longer applied. No default changes (#367).
- An en dash where a sign belongs is refused by name, with `ErrEnDash`,
  everywhere a sign is read. `traveller.Sign` is the one place that recognises
  it, where four parse paths each checked for it before; the Prior Service
  Table's DM headings, Table 1's alterations and the Rank and Service Skills
  box's grants missed it, and refused one only as no sign, no benefit or no
  level (#366).
- The changelog, the coding standards, the glossary and the release snapshots
  `THEORY.md` and `WALKTHROUGH.md` are in `docs/`. The root keeps only what a
  tool outside the repo looks for there: the README, the licence and
  `CLAUDE.md`.
- `internal/mutate` builds the mutated copy and its test binaries before it
  runs the command, the way `go test` builds them, vet checks included, and
  reports a mutation that does not build as `NOT BUILT`, exiting 1. It said
  `KILLED` for one before, since the command failed on the build before any
  test ran, so a mutation could read as shown killed when nothing had been
  tested.
- The Table of Ranks is transcribed as p. 10 prints it: Rank 1 to Rank 6 as
  rows, the services across. It was held a service's column at a time, the
  same way the second transcription walks it, so a slip made reading down a
  column could have passed both; the two copies now read it in opposite
  directions, as they do the other service tables. Every lifted rank is unchanged.
  `muster.go` no longer names the deleted `mustering.json` (#354, #355).

## 1.0.0-rc.2

The second release candidate for `v1.0.0`. One command line changes: `batch`
no longer takes `--auto`. Everything else is inside the tool. The rules tables
are now the page's own text in Go, the last embedded file is gone, and the
withdrawn-reading and erratum checks are corrected. The record, its shape, the
sheet and the transcript are what rc.1 wrote, and every golden is unchanged.

### Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-rc.2
```

**No binaries are attached.** A release is a tag, installed with `go install`.

### What a referee will notice

- **`batch` takes no `--auto`.** A batch is always decided by the policy,
  having nobody to ask, so `ctchargen batch --count 20` is a batch now. A
  command line that passes `batch --auto` is refused as an unknown flag: drop
  `--auto` from it. This is a deliberate break, taken before `v1.0.0` freezes
  the CLI. `new`'s `--auto` is unchanged (#336).
- A record written by rc.1 reads back, renders and validates as it did, and
  `record` stays 1.

### The transcription

ADR 0005 records the decision: the rules tables are Go literals of what the
page prints, with the lift kept, and the last `//go:embed` is gone (#340).

- **All seven tables moved from embedded JSON to `internal/rules/transcription.go`**
  (#337, #338). Each table's lifted values were proven equal to what its JSON
  lifted before the JSON was deleted. A table, a service or a column the type
  doesn't have no longer compiles, and an empty cell is refused.
- **Every cell is the page's text** (#339, #344, #345, #346):
  - the characteristic abbreviations the tables print (`Intel 8+`, `+1 Stren`);
  - p. 23's `Rifl3-1` and `SMG-1`;
  - an em dash wherever the page prints one;
  - p. 10's DM amounts as the row headings it prints ("DM of +1 if");
  - p. 9's Aging Table with both header rows, all six characteristic rows and
    each cell's span;
  - p. 23's box under its own labels (`Navy Captain`, `Marine`).

  Each table was read again from the page, one page at a time, before typing.
  Every lifted value is unchanged.

### The readings

- **A withdrawn reading is held only to being stamped on no golden** (#332). A
  reading replaced by another still describes the records the replacement
  governs. So requiring its condition to fit no golden, as rc.1 did, would have
  turned the gate red at the first withdrawal that names a replacement.
- **A test holds `traveller.Errata` to the `Erratum` constants** (#333). A
  constant left out of the list would have had no reading and still compiled.

### Notes

- `render`'s package doc and CLAUDE.md name the readings `render` writes and
  its import of `rules` (#334).
- The glossary defines a transcription and the lift. Its **Auto run** now
  covers both `new --auto` and every batch.

## 1.0.0-rc.1

The first release candidate for `v1.0.0`. The record, its shape and the sheet
are what beta.7 wrote, and every golden is unchanged. What changes is where
the readings live and that the tool can now explain them: ADR 0004,
"documents are for people, data is code, checks are tests", carried out in
five steps (#321–#325). No markdown is parsed, generated or held to the code
by the gate any more.

### Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-rc.1
```

**No binaries are attached.** A release is a tag, installed with `go install`.

### What a referee will notice

- **The tool explains its readings.** A transcript names the reading beside
  each outcome it governed, as an id like `E002`. `--readings` on
  `new --history` and `render --history` ends the transcript with the full
  text of each reading the record names: what the page says, the gap, the
  reading and why. It is refused without `--history`, and without it every
  transcript is what it was (#324).
- **`ctchargen readings`** lists every reading with its status, or writes the
  ones named in full; an id that names no reading is a usage error (#324).
- A record written by beta.7 reads back, renders and validates as it did, and
  `record` stays 1 (#96).

### The readings

- The readings are Go values in `rules`, one per erratum, so a reading
  missing or one too many does not compile (#321). `ERRATA.md`, its embed and
  its parser are gone (#323). The readings carry the document's provenance in
  their doc comment, and the eight questions it recorded as settled by the
  page are doc comments, verbatim, on the declarations that implement them.
- An erratum names one decision for good: a reading may be reworded, but a
  changed decision is a new erratum. A reading whose decision no longer holds
  is withdrawn, not deleted. It keeps its erratum, says why, and names any
  erratum that replaced it; the stamping harness refuses any golden that stamps a
  withdrawn reading or that its condition fits. No reading is withdrawn
  (#322).
- Two rewordings, no decision changed: E007's rationale gains the comma
  "printed in, in the UPP" needed (#321), and the twelve links between
  readings, anchors into the deleted document, are plain ids (#324).

### The coverage map

- The coverage map is a check that writes nothing. `docs/COVERAGE.md` is gone,
  and every `Implements:` marker drops its `in <Section>.` suffix. Every check
  stays: each rule implemented and held, its implementations agreeing on its
  page, and every cite valid. The map prints on demand as one table, in page
  order, with `go test -run TestTheCoverageMapHolds -print` from
  `internal/coverage` (#325).

### Notes

- ADR 0004 records the course, and the glossary defines a withdrawn reading
  and says an erratum's decision never changes (#326).
- The `traveller.Erratum` constants no longer repeat their readings' titles;
  a title lives once, in its reading (#323).

## 1.0.0-beta.7

Nothing a referee sees at the table changes: the record, its shape, the sheet
and the transcript are what beta.6 wrote, and every golden is unchanged. This
release is the rules data, the checks that hold it, and the readings document,
brought closer to the page and to what the tree does.

### Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.7
```

**No binaries are attached.** A release is a tag, installed with `go install`.

### What a referee will notice

**Nothing at the table.** A record written by beta.6 reads back, renders and
validates as it did, and `record` stays 1 (#96).

### The rules data and its checks

- **An en dash is never a minus.** A target typed with one (`8–`, `–3`) is
  refused with an error saying so, rather than a bare parse error (#296), and
  an Acquired Skills cell opening with one (`–1 Social Standing`), which used
  to lift silently as a skill of that name, is refused the same way (#288).
  Which glyphs are a sign is defined once, in `traveller`, and read by both
  (#297).
- **A data file broken two ways reports the same fault every run** in the
  Prior Service Table's row counts and the Acquired Skills tables, the first
  fault in the order pp. 10 and 11 print them (#286).
- **The coverage generator checks a page cite whole**: its shape, every book
  it names (1, 2 or 3) and every reading it names (#287). A range takes an
  en dash, and a hyphenated one is refused (#311).
- `services.json` names the retirement increment `perAdditionalTerm`, after
  p. 21's own words, where it was `perTermBeyondEight`; the 8 lives only in
  the table's last row (#312).

### The readings and the page

- **ERRATA.md's quotations carry only the emphasis the page prints.** Ten
  bold passages across seven readings were added here and are removed, each
  read against its page; E011's quotation of p. 5 regains a dropped "the"
  (#310).
- ERRATA.md no longer argues from the withdrawn PRD Determinism section: E002
  and E013 say what the order and the throw change in the record (#290).
- Comments no longer say the page is set in U+2212, which no page image can
  show (#313).

### Notes

- "Values 2–12 initially" is marked on the six 2D rolls that carry it, and
  `Profile.Alter` on "Never above 15" alone (#294).
- CODING_STANDARDS.md's marker rule says what the tree does: a promise the
  test harness keeps is marked on the test code that carries it, never on a
  test; a rule several declarations carry is marked on each; only a promise
  no single declaration carries goes on a package doc (#293, #313).
- CLAUDE.md says what a dice-consumption change does to a fixture: nothing
  names the throw that moved, so a golden diff is read before it is
  regenerated (#289).
- The worked example is "reproduced", not "replayed" (#290); comments stop
  counting things that grow (#291); `Retirement.lastTabled`'s comment gives
  the indexing test (#292), and `traveller`'s package doc points to Authority
  point 6 (#313).
- `rules`' package doc and CLAUDE.md say the lift checks cells and names and
  leaves values, a misspelled key's zero included, to the second
  transcription; `read` tolerates an unknown key for the files' own `_cite`
  and `_note` notes (#317). `asked`'s comment names what the one-answer sites
  settle (#318).

## 1.0.0-beta.6

A record says who answered it once, at the top. The record's shape changed to
say it, so a beta.5 `--auto` record no longer reads back; the rest is the code
behind it, simplified in four places.

### Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.6
```

**No binaries are attached.** A release is a tag, installed with `go install`.

### What a referee will notice

**`policy` is an object, present exactly when `--auto` answered** (#276). It
carries the policy's version and its three strategies, on every `--auto`
record whether or not the policy was asked anything, and a record a person
answered carries none. `inputs` no longer carries the strategies, and a choice
event no longer carries `by`. The transcript still says `policy chose` or
`player chose`, and the sheet still says "(service chosen by the policy)".

**A beta.5 `--auto` record no longer reads back.** Its `policy` is a bare
number, and `render` refuses it rather than reading it as a record the player
made. A beta.5 interactive record still renders. Until `v1.0.0` a record is
not promised to be readable by the next build (#62); `record` stays 1 (#96).

**`new` without `--auto` refuses `--career`, `--skills` and `--muster`**,
which decided nothing there (#276).

### Notes

- `chargen.Generate` becomes `chargen.Auto` and `chargen.Guided` (#276), so a
  run cannot be credited to an answerer it did not have; the worked-example
  test had been crediting the policy with its scripted choices.
- The gate every decider passes through hands each one its own copy of what
  it is offered, and refuses a question with one legal answer (#278).
- One scripted roller, `internal/scripted`, drives both the fixtures and the
  command's walks (#280).
- The build reaches a record through the options every generation takes,
  `chargen.WithBuild`, and a batch member's `build` is tested for the first
  time (#282).
- A golden for an `--auto` run the policy answered nothing in, which is a
  quarter of forced-scout records (#284).
- ADR 0003 records that the interactive session is tested through the
  command (#285).
- `THEORY.md` and `WALKTHROUGH.md` are release snapshots: regenerated in each
  release's prep, stamped with its tag, and ignored between releases (#274,
  #275).

## 1.0.0-beta.5

The dice are no longer seeded. A character cannot be brought back by running
the tool again: the JSON record, which logs every die, is the only way back to
one (ADR 0001). Most of what a referee will notice follows from that; the rest
are corrections.

### Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.5
```

**No binaries are attached.** A release is a tag, installed with `go install`.
Earlier releases' binaries stay on their own release pages.

### What a referee will notice

**`--seed` and `--answers` are gone** (#209). A record carries no
`inputs.seed`, and `batch` generates each member from dice of its own. The
sheet and the transcript end with one line, "Keep the JSON record: it is the
only way back to this character" (#206), in place of the "Regenerate with …"
footer, and a stopped interactive session says what was answered and that
nothing was written (#207). A record written by beta.4 still renders; its
`inputs.seed` is ignored.

**`batch -o <dir>` names members by position**, `01.json` through `20.json`
for a batch of twenty, padded to the batch's width, so a directory sorts in
generation order (#208).

**A `batch` to a `.jsonl` file that stops claims nothing it never wrote**
(#246). A file already there is refused before anyone is generated, and a run
that fails after generating prints no "written before the run stopped" tally.

**A directory where a record is to go is named as a directory**, for `new -o`
and for a batch member, without the `--force` advice that `--force` could not
honour (#254).

**The kept transcript numbers its step headings**, `## 20. term 1 (pp. 5-7)`,
as the live view always did, so no `--history` transcript skips a number
(#249).

**A "yes" can no longer be refused as not offered** (#230). The yes-or-no
questions shared one pair of alternatives, and a write through one record's
could make every later character's answer fail.

**Two readings are corrected.** E003 is stamped from the end of term 8, as
`ERRATA.md` says, and no longer on the 12 at the end of term 7. E004 now says
what the engine does with a survival death's age, months from an earlier
medical crisis included (#233); no record changes.

### Notes

There is no Go library (#199): every package is under `internal/`, so
`v1.0.0` will freeze two things, the record and the command line. Until that
tag a record is not promised to be readable by the next build (#62).

The gate holds more of the tree to itself. It re-runs `WALKTHROUGH.md`'s
transcripts, checks the documents' links, and generates `docs/COVERAGE.md`
from markers beside the code; `rules` lifts `ERRATA.md`, and the `--auto`
policy is its own doc comments. `docs/POLICY.md`, `docs/PRD.md` and
`docs/PRERELEASE.md` are removed, and the history keeps them. The coverage
ratchet is gone (#198). `go run ./internal/mutate` shows whether a test kills a
mutation, in a scratch copy of the tree. There is no CI; `task` is the gate.

**Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
rebuild at `41a213a` and are builds of a different implementation.

## 1.0.0-beta.4

Every character this build generates is the character beta.3 generated from
the same seed. One change reaches a referee: `--force` no longer loses the
record it was asked to replace when the write fails.

### Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.4
```

**No binaries are attached.** A release is a tag, installed with `go install`.
Earlier releases' binaries stay on their own release pages.

### What a referee will notice

**`--force` no longer empties the file it replaces before writing the new
one** (#191). The record is written beside it and renamed into place, so a
write that fails partway leaves last week's record whole. A replaced record is
written with the same `0600` mode as every other record, rather than keeping
the old file's mode. Without `--force` nothing changes: an existing file is
still refused.

### Notes

The repository's non-Go files are formatted with `bun fmt` rather than
prettier, against the same `.prettierrc.json` and `.prettierignore`; no file
was reformatted by the switch (#189). There is no CI; `task` is the gate.

**Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
rebuild at `41a213a` and are builds of a different implementation.

## 1.0.0-beta.3

Every character this build generates is the character beta.2 generated from
the same seed. What changed is what surrounds a character: two record fields
are renamed to the book's words, the schema says what it actually accepts, and
the exported Go API and the command line are held to a baseline ahead of the
`v1.0.0` freeze.

### Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.3
```

**No binaries are attached.** A release is a tag, installed with `go install`.
Earlier releases' binaries stay on their own release pages.

### What a referee will notice

**Two record fields are renamed, to the book's words.** Book 1 keeps "title"
for nobility (p. 5; Book 3 p. 22) and calls Captain a rank (pp. 10, 25), so:

```json
"rank": 4, "rankName": "Captain", "nobility": { "eligible": true, "title": "duke/duchess", "assumed": true }
```

This replaces `rankTitle` and `title: { eligible, rank, assumed }`. The sheet
and the transcript print exactly what they printed before. The record is
still `record: 1`: the shape moves freely until `v1.0.0`.

**`--name` is one line.** A name carrying a line break is refused, because the
sheet prints it as a heading and a newline would start sections the character
does not have.

**A batch that stops partway says what it wrote**, as
`2 written before the run stopped, none died`, before its error.

**The exit status is 0 or 1**, and the README says so. A character who died is
a delivered record and exits 0.

### What changed in the schema

**An event of a kind this build does not write is refused.** beta.2's notes
said a reader holding the schema could load a record carrying a new event
kind; the schema never allowed that, and now says so. The root still accepts a
new top-level section, and `ctchargen render` still prints `(unknown event
kind …)` for a record it cannot validate.

The schema's `point` and strategy enums stay closed: they describe the rules
this build has.

### For Go importers

`v1.0.0` freezes the exported API of `traveller`, `dice`, `rules`, `chargen`
and `render` under ordinary Go rules, and this release settles it:

- Six names nothing outside their package used are unexported:
  `rules.ErrMalformed`, `rules.ErrNoSuchRow`, `render.ErrNotARecord`,
  `traveller.ErrTarget`, `traveller.MaxCharacteristic`,
  `traveller.MinUnderAging`.
- `rules.Rules` hands out nothing a caller can write to. `Aging`, `Muster`,
  `Eligibility`, `Education` and `Retirement` are methods rather than fields,
  and `Service`, `Weapons` and `Aging.At` return copies.
- `rules.Service.Title` is `RankName`, `chargen.Character.RankTitle` is
  `RankName`, and `chargen.Title` is `chargen.Nobility` with a `Title` field.
- `chargen.ParsePolicy` replaces `ParseCareer`, `ParseSkills` and
  `ParseMuster`.
- A `Decider` cannot reach the run's dice through the `Vary` it is handed, and
  cannot answer with something it wrote into its own offer.

### Notes

The repository's gate gained three baselines, each verified by breaking it:
the exported API, held both ways; each command's flags, types, defaults and
exit status; and the schema's strategy enums against the strategies the code
takes. There is no CI; `task` is the gate.

**Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
rebuild at `41a213a` and are builds of a different implementation.

## v1.0.0-beta.2 — 2026-09-08

The tracker is empty, and this is the shape `v1.0.0` will freeze. Nothing here
changes a character.

### Install

A binary, needing no Go toolchain. This is the line for an Apple-silicon Mac:

```sh
curl -Lo ctchargen https://github.com/philoserf/ctchargen/releases/download/v1.0.0-beta.2/ctchargen-v1.0.0-beta.2-darwin-arm64
chmod +x ctchargen
```

Substitute `darwin-amd64`, `linux-amd64` or `linux-arm64` in the filename for
another machine; `checksums.txt` is attached beside them. A **browser**
download is quarantined and needs `xattr -d com.apple.quarantine ctchargen`
first — `curl` is not. The binaries are **unsigned**, so a recent macOS may
refuse one regardless; if it does, the tool is not broken and
`go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.2` is the
way in.

### What a referee will notice

Nothing at the table. Every character this build generates is the character
beta.1 generated from the same seed. The two changes are to the record's
published schema and to the repository's own coverage gate.

That is what a last beta before a freeze looks like: the work is in what the
record promises rather than in what it says.

### What changed in the record

**`docs/character.schema.json` now describes the record `v1.0.0` will freeze.**

A record from a later build validates against it. The root object accepts a
top-level section this schema has never seen, and `record` — the field naming
the record's shape — accepts a number from the future rather than being pinned
to `1`. Both halves were needed: a reader that refused the shape number would
never have reached the new field it was meant to allow.

That is the point of a frozen record. A third party holding the `v1.0.0` schema
should be able to load a record written by a later build and render what it
understands, which is what `ctchargen render` already does for an event kind it
does not know — it prints `(unknown event kind …)` rather than failing.

**The record's own structures are still pinned.** A key added inside an event,
inside `enlistment`, inside a benefit, is refused exactly as before. Those are
what a sheet is rendered from, and pinning them is what keeps the schema worth
validating against. A later build may add a section; it may not quietly change
an event.

This build still writes `record: 1`, and there is now a test that says so.

### What this tag does not promise

**The freeze is not made.** `v1.0.0` means two things and only the first holds
today: a referee can trust what the tool prints at the table, and the record is
frozen and supported. A record written by this build is still not promised to
be readable by the next until that tag.

The shape is ready. The promise is what `v1.0.0` is for.

### Notes

**Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
rebuild at `41a213a` and are builds of a different implementation.

`docs/PRERELEASE.md` carries the full account, including the review that
preceded this tag and the four new gates broken on purpose.

## v1.0.0-beta.1 — 2026-09-05

The backlog is empty. Every milestone is closed, and this is the first tag to
ship with no finding open at any severity.

### Install

A binary, needing no Go toolchain. This is the line for an Apple-silicon Mac:

```sh
curl -Lo ctchargen https://github.com/philoserf/ctchargen/releases/download/v1.0.0-beta.1/ctchargen-v1.0.0-beta.1-darwin-arm64
chmod +x ctchargen
```

Substitute `darwin-amd64`, `linux-amd64` or `linux-arm64` in the filename for
another machine; `checksums.txt` is attached beside them. A **browser**
download is quarantined and needs `xattr -d com.apple.quarantine ctchargen`
first — `curl` is not. The binaries are **unsigned**, so a recent macOS may
refuse one regardless; if it does, the tool is not broken and
`go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.1` is the
way in.

### What a referee will notice

- **Twenty NPCs stop reading as twenty copies.** The auto policy took the first
  weapon on the printed list every time — Body Pistol and Dagger, and nothing
  else, across thirty characters — and designated Advanced Education every time
  it was offered. Over the same thirty it now names twenty-one distinct weapons
  and uses all four skills tables. A character generated under the default no
  longer goes his whole career without raising a characteristic.
- **`batch` says what it did**, on standard error so it cannot corrupt the
  JSONL: `100 written, 76 died`. That is the rules working — the default
  strategy re-enlists until the survival throw gets him — but a directory of
  corpses with no word about it was a trap.
- **`--survivors`** passes over the dead and goes on to the next seed, so
  "twenty scouts for the starport" is one command. It rerolls nobody: each
  character written is still exactly what his own seed makes, and
  `new --seed <that>` brings him back. The seeds are simply no longer
  consecutive.
- **`batch` streams.** `ctchargen batch --count 50000 --auto | jq` produces its
  first record immediately rather than after the run.
- **Two flags that were quietly ignored are refused**: `--sheet --history`
  together, and a word typed after a help flag.

### What changed in the record

The record now names **its own shape** and **the policy that answered it**, so
two records from one seed can be told apart by why they differ rather than only
that they do. A roll with nothing to meet — the draft die, a skills table, a
mustering out roll — is its own event kind rather than a throw recorded as
having "succeeded" against nothing.

`docs/character.schema.json` describes all of it, and every golden is validated
against it.

### What this tag does not promise

**The record is not frozen.** `v1.0.0` means two things and only the first
holds today: a referee can trust what the tool prints at the table, and the
record is frozen and supported. A record written by this build is not promised
to be readable by the next until that tag.

### Notes

**Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
rebuild at `41a213a` and are builds of a different implementation.

`docs/PRERELEASE.md` carries the full account, including the review that
preceded this tag and what it kept finding.

## v1.0.0-alpha.5 — 2026-09-04

The tag that closes the referee's alpha report. Every finding of
[#26](https://github.com/philoserf/ctchargen/issues/26) is answered, and this
is the first release whose binaries are attached by CI rather than by hand.

### Install

A binary, needing no Go toolchain:

```sh
curl -Lo ctchargen https://github.com/philoserf/ctchargen/releases/download/v1.0.0-alpha.5/ctchargen-v1.0.0-alpha.5-darwin-arm64
chmod +x ctchargen
```

Substitute `darwin-amd64`, `linux-amd64` or `linux-arm64` in the filename for
another machine; `checksums.txt` is attached beside them. A **browser**
download is quarantined and needs `xattr -d com.apple.quarantine ctchargen`
first — `curl` is not, which is why the line above is one. The binaries are
**unsigned**, so a recent macOS may refuse one regardless; if it does, the tool
is not broken and `go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-alpha.5`
is the way in.

### What a referee will notice

- **The sheet prints the seed**, so a character you liked can be generated
  again — and the footer stops offering a command for a character the seed
  cannot bring back.
- **The headline says whose service it is.** Where you named one, it says the
  policy did not choose it; where the draft placed him after a refused
  enlistment, it names the service that refused him.
- **A session that stops offers the way back in.** Ctrl-D or a closed pipe
  halfway through generation now prints the seed and the answers you gave, and
  `--answers 1,2,1` replays them to the question you stopped on. A half-built
  character is still not a record — but a long session is no longer retyped.
- **Interactive generation reads straight.** Event numbers no longer skip, the
  two Advanced Education menu entries are told apart, and a non-numeric answer
  is complained about by name instead of silently re-printing the menu.
- **A one-term character no longer prints "1 terms".**
- **Errata are off the sheet.** They stay in the record and in the transcript,
  where an id can be looked up; on the sheet they were ids a reader could not
  expand.
- **The regenerate line quotes what it pastes.** A record is a file people
  share, and its fields were reaching your shell unquoted.

### What ships open, and why

**This tag departs from the shipping bar** stated in
[`docs/PRERELEASE.md`](https://github.com/philoserf/ctchargen/blob/v1.0.0-alpha.5/docs/PRERELEASE.md),
as alpha.4 did and for the same reason: #41, #42 and #43 are still open. All
three are structural findings from the prerelease code audit (#27) about the
shape of the code, and none of them changes a character the tool generates.
They are the next milestone's first work.

Also open: `batch` and the record's shape (#33, #34, #44, #46, #50, #76), and
housekeeping (#51–#60, #83, #84). #76 matters more than its number suggests —
the record does not yet name its own shape, so the freeze promised at `v1.0.0`
has nothing to attach to.

That document's alpha.5 section carries the full account, including the review
that preceded this tag: no whole-tool pass, but a review per PR, three of which
in a row caught a regression introduced while fixing something else.

### Notes

**Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
rebuild at `41a213a` and are builds of a different implementation.

alpha.4's binaries were built locally and uploaded after the fact; its notes
say so. These come from `release.yml`, which checks out this tag itself.

## v1.0.0-alpha.4 — 2026-09-04

The tool describes itself now.

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-alpha.4
```

Go excludes prereleases from `@latest`, so an alpha installs by name. That is
the intended friction.

> **Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
> rebuild at `41a213a` and are builds of a **different implementation** — they
> have a `replay` subcommand and flags this tool does not. Their notes are
> accurate about the tag they head and describe nothing here. They are not an
> upgrade path; this tag is a fresh install.

### Every command lists its own flags

This is the first finding of a Classic Traveller referee's alpha report (#26),
and the one his verdict named first. Before this tag, `new --help` and
`batch -h` printed `usage: flag: help requested` and exited 1, `--help` at top
level read as a mistyped subcommand, and there was no flag list anywhere.

```
$ ctchargen --help
usage: ctchargen <command> [flags]

  new      generate one character, asking at every choice unless --auto
  batch    generate many characters from one base seed, under --auto
  render   write a record saved earlier as a sheet or as a transcript
  version  write the build

Run `ctchargen <command> --help` for that command's flags.
```

`ctchargen new --help` lists all eleven of its flags with their descriptions,
**their values and their defaults** — which matters here, because the notes for
`v1.0.0-alpha.2` documented three strategies that were later renamed, and a
reader following that page had no way to find out from the tool.

### A typo no longer becomes a session

`ctchargen new extra-arg` used to ignore the word and start an interactive
generation on a seed nobody chose. Every command refuses a word it was not
expecting now.

**This narrows accepted input in one place:** `ctchargen version junk` was
accepted at exit 0 and is an error here. **That is not settled** — #70 is open
precisely to decide whether it keeps the refusal, softens it to unknown flags
only, or reverts. Shipping this tag is not a verdict on it.

### The README says how to install it

It never did. It opened with four `ctchargen …` lines and never said how
`ctchargen` gets onto a path, and it documented about a third of the flags. It
now carries an install section, all four commands, and recipes that use the
flags it never mentioned — including `batch` with no `-o`, which writes NDJSON
to standard output.

An unknown strategy is also refused in words rather than by pointing at a
repository file:

```
before: no such strategy: --muster "benefits"; POLICY.md carries [cash goods spartan]
after:  no such strategy: --muster "benefits"; want cash, goods, spartan
```

### Still alpha, and what that means here

[`docs/PRERELEASE.md`](https://github.com/philoserf/ctchargen/blob/v1.0.0-alpha.4/docs/PRERELEASE.md)
carries the full list. In short: three high findings from the code audit (#41,
#42, #43) are open, and this tag departs from that document's own shipping bar
to say so. All three are structural — about the shape of the code — and none of
them changes a character the tool generates. The rest of the referee's report
is open too, the seed absent from the character sheet (#32) being the next one
answered.

The engine is the part that has earned trust: determinism holds, the book's own
worked character (pp. 23–25) replays against it, and the 1977 death rule is
implemented as printed. Everything around the engine is still alpha.

### Binaries

Attached after the fact, and **built locally rather than by CI** — the release
workflow that produces them lands after this tag, so from the next release the
assets come from a runner checking out the tag itself. Both routes stamp the
same version; the difference is only in what a downloader is trusting, and it
is recorded here rather than left to look identical.

`checksums.txt` covers every binary beside it.

## v1.0.0-alpha.3 — 2026-09-03

Tagged without published release notes.

## v1.0.0-alpha.2 — 2026-08-30

> [!WARNING]
> **Superseded — do not install this.** This release is a build of an
> implementation **removed from this repository** at `41a213a` ("Fresh
> start"). It has a `replay` subcommand and flags the current tool does not,
> and it is not an upgrade path to anything below it.
>
> Install [**v1.0.0-alpha.4**](https://github.com/philoserf/ctchargen/releases/tag/v1.0.0-alpha.4)
> instead. The notes below are unedited, and are kept as the record of what
> this tag installs.

---

`--auto` can now be steered.

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-alpha.2
```

### Named strategies

Three rows of the policy are selectable, and the defaults are unchanged — a record generated without these flags differs from an `alpha.1` record only in its version stamps.

```sh
ctchargen new --auto --skills rounded          # a term improving himself, a term learning
                                               # the trade, a term specialising
ctchargen batch --count 20 --auto --skills advanced --career 4
```

| Flag       | Values                                                    |
| ---------- | --------------------------------------------------------- |
| `--skills` | `service` (default) · `personal` · `advanced` · `rounded` |
| `--muster` | `cash` (default) · `benefits`                             |
| `--career` | `max` (default) · a term 1–7 to leave after               |

**Why `--skills` matters.** The default always chose Service Skills, which meant a character never rolled on Personal Development — so never gained a characteristic in service — and never rolled on Advanced Education, putting Medical, Navigation, Computer, Leader and Administration out of reach outside the p. 23 rank grants. It compounded: the fourth table needs Education 8+, and Education rises only on Personal Development.

Across 30 characters from one base seed:

|                     | in-service characteristic gains | distinct skills                             |
| ------------------- | ------------------------------- | ------------------------------------------- |
| default             | 1                               | 13                                          |
| `--skills rounded`  | **53**                          | 15                                          |
| `--skills advanced` | 0                               | **18** (reaches the Education 8+ table 55×) |

`--career` sets **intent, not outcome**: the reenlistment throw is still required each term (p. 6) and a 12 exactly still forces another term (pp. 6–7). `--muster benefits` never rolls for cash, so the character musters out with none.

Every strategy is a pure function of the choice it is handed, so the policy stays **total** and **deterministic** — same inputs, same character. [`docs/POLICY.md`](https://github.com/philoserf/ctchargen/blob/v1.0.0-alpha.2/docs/POLICY.md) describes each one, and a gate holds the strategies the CLI accepts to the ones that document describes, in both directions.

### Breaking: record versions moved

`schema_version` **2 → 3** and `policy_version` **3 → 4**.

Which strategy generated a character is caller input, so it is recorded in the record's `inputs`; `policy_version` continues to name the document rather than the selection within it. The new fields are omitted when the defaults were used.

**Records written by `v1.0.0-alpha.1` need `replay --ignore-provenance` against this build.** That is the whole reason this tag follows so quickly — leaving the only tag behind the change would have hidden the break.

`engine_version` stays **0.6.0**: a strategy changes the character the way `--service` does, through input, not through the engine. No golden fixture moved.

### Still alpha

For the same reason as before: every defect found in review has come from reading a held page, not from running a test. `v1.0.0` waits on characters generated in anger against the book.

## v1.0.0-alpha.1 — 2026-08-30

> [!WARNING]
> **Superseded — do not install this.** This release is a build of an
> implementation **removed from this repository** at `41a213a` ("Fresh
> start"). It has a `replay` subcommand and flags the current tool does not,
> and it is not an upgrade path to anything below it.
>
> Install [**v1.0.0-alpha.4**](https://github.com/philoserf/ctchargen/releases/tag/v1.0.0-alpha.4)
> instead. The notes below are unedited, and are kept as the record of what
> this tag installs.

---

The full Classic Traveller prior-service procedure of **Book 1 pp. 4–25**, generated against the held © 1977 text: characteristics, enlistment and the draft, terms with survival, commissions, promotions and skills, aging and the medical crisis, mustering out with ships and titles. Death is an outcome, not an error — the dead get complete records too.

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-alpha.1
```

Go excludes prereleases from `@latest`, so the alpha installs by name. That is the intended friction.

### What it does

```sh
ctchargen new --auto                        # one character, policy decides, JSON to stdout
ctchargen new --seed 42 --service navy      # interactive: you answer each choice
ctchargen batch --count 20 --auto -o npcs.jsonl
ctchargen render character.json             # Markdown character sheet
ctchargen render --history character.json   # the full generation transcript
ctchargen replay character.json             # verify a record reproduces exactly
```

The JSON record is canonical; the Markdown is a render of it. Every record carries its seed, versions, inputs, and the complete event log, and `replay` re-runs the engine from the seed and recorded choices, exiting non-zero at the first divergence.

### What is verified

Every rule carries a printed-page cite, and every interpretation is recorded in [`docs/ERRATA.md`](https://github.com/philoserf/ctchargen/blob/v1.0.0-alpha.1/docs/ERRATA.md) with its cite and **stamped on the records it governed** — E001 through E009, so a record says which readings shaped it.

The rule data was checked cell by cell against the held pages: the Prior Service Table, the Table of Ranks, all 24 acquired-skills tables, both mustering-out tables, the Aging Table, the Rank and Service Skills box, the retirement pay table, both weapons lists, and Book 3's Nobility table.

The book's own worked example reproduces. Jamison (p. 25): UPP 779C99, four merchant-ship receipts becoming a 30-year-old ship owing 10 years of payments, CR 4,000 retirement, two extra muster rolls at rank 5, aging saves at 34 and 38.

[`docs/COVERAGE.md`](https://github.com/philoserf/ctchargen/blob/v1.0.0-alpha.1/docs/COVERAGE.md) maps every rule to its cite, implementation, and test. [`docs/POLICY.md`](https://github.com/philoserf/ctchargen/blob/v1.0.0-alpha.1/docs/POLICY.md) is the auto mode's decision table. [`docs/PRERELEASE_REVIEW.md`](https://github.com/philoserf/ctchargen/blob/v1.0.0-alpha.1/docs/PRERELEASE_REVIEW.md) is the v1.0.0 bar and the evidence for each line of it.

### Why alpha

Because the remaining instrument is use. Every defect found in the last three review passes was found by **reading a held page**, not by running a test — a page cite that was wrong in ten places including every titled record's event log, and a rules interpretation that had been applied silently since the beginning. What the suite can prove, it now proves; the rest needs characters generated in anger.

`v1.0.0` follows when use against the book turns up nothing the suite missed.

### Versions

`engine_version` 0.6.0, `schema_version` 2, `policy_version` 3. These track record behaviour rather than releases and are independent of this tag — `engine_version` changes when generation behaviour or any event's text changes, because `replay` compares whole events.
