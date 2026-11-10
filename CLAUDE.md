# CLAUDE.md

Guidance for Claude Code in this repository. The v1 contract was
`docs/PRD.md`, delivered and then removed from the tree after `ec7289f`, where
the history keeps it; this file carries the authority model live.

## What this repo is

A Go CLI that generates rules-accurate Classic Traveller characters from
Books 1–3 (© 1977 text, FFE reprints). The repo was emptied at `41a213a` and
rebuilt from that contract.

**What governs the work now is what Classic Traveller referees report about
using the tool** (#26 was the first such report); the page and the clean room
govern how a report gets answered. The current release is the README's
`go install` line, which the release gate reads; this file does not repeat it.

**What `v1.0.0` means, decided in #74 and narrowed in #398.** A referee can
trust what the tool prints at the table, and the command line he types is
held: its flags, types, defaults and exit status (#126), and so are the note's
property names and their meanings, which vault queries stand on (#399). The
rest of the note's shape is not frozen; it may keep moving after `v1.0.0`, as
the playtests ask (ADR 0006). The `v1` label carries which issues that covers
today — this paragraph states the bar and never the list, because a list here
goes stale every time one closes.

## Authority — read this before implementing any rule

1. **Never implement a rule from memory.** Training-data Traveller is mostly
   the 1981 revision and later editions. The held © 1977 text governs even
   where it differs — most notoriously, survival failure is death (Book 1
   p. 5), with no "injured instead" option.

2. **Never read a table out of `pdftotext`.** The reprints' embedded font
   substitutes glyphs and the substitutions look like data: on Book 1 p. 9 the
   "—" cells of Mustering Out Table 1 extract as the digit **4**, "Travellers'
   Aid" extracts as `Travellers9`, and a minus sign goes the same way, so an
   `N−` target reads as `N3`. A run that trusts the extraction gives a Scout a
   seventh benefit he does not have.

   Read pages **visually** instead — the `Read` tool with a `pages` range on
   the PDF.

   **One page at a time when the page carries a table.** A range renders each
   page smaller, and the prerelease review lost a cell to it: in a six-page
   read, Merchants row 3 of the Service Skills Table came out `Electronic`
   where p. 11 prints `Blade Cbt`. Re-reading that page alone was right at
   once. Ranges are fine for prose; a table gets its own call.

   **And the visual read is not the whole defence.** The substitution reaches
   the rendered image too, not only the extracted text: p. 23's Rank and
   Service Skills box shows `Rifl3-1` in the page image, and Rifle-1 is what
   it means. What catches that is a semantic cross-check — no skill or weapon
   named `Rifl3` is defined anywhere in the three books, and Rifle is on the
   p. 13 gun list. Read every extracted name back against the description
   headings and the weapon lists, and treat one that resolves to nothing as a
   broken glyph rather than a new name.

3. **Transcribe every table twice.** Once as the page prints it, the
   transcription in `rules`, once into the `rules` tests, both from the same visual reading pass. The second
   transcription is the check the font trap needs. Retyping the second copy
   from the first checks nothing.

4. **Every implemented rule carries its printed-page cite** in the code, as
   an `Implements:` marker beside the implementation, and the test that holds
   it carries a `Holds:` marker. `internal/coverage` holds the two to each
   other.

5. **Where the text is silent or ambiguous, a new reading is a new erratum
   and a `Reading` value in `rules`** with its page cite and stamping
   condition, named on every record it governs. Never applied silently.

6. **A number that indexes a printed table belongs in the transcription; a
   bare procedural bound is a constant beside the code that applies it, with
   its cite** (#65). The aging table's last printed term indexes a column, so
   it belongs in the transcription and not beside the procedure; the three-roll
   cash cap and the rank thresholds bound a procedure and index nothing, so they
   belong beside the code that applies them. The rule is stated once here rather
   than argued per number, and it moves numbers both ways — a number on the
   wrong side of it is a finding, not a preference.

### Page offsets

Printed page N is PDF page **N+6** in Book 1, **N+5** in Books 2 and 3.

- `~/Documents/Traveller/Classic/Book 1 Characters and Combat.pdf` —
  authoritative for the whole procedure; chargen is printed pp. 4–25 (PDF
  10–31).
- `~/Documents/Traveller/Classic/Book 2 Starships.pdf` — Type S p. 18, Type A
  p. 19, consulted only because Book 1 p. 22 points there.
- `~/Documents/Traveller/Classic/Book 3 Worlds and Adventures.pdf` — Nobility
  p. 22, consulted only because Book 1 p. 5 points there.

Everything else in that directory is **out of authority**: Books 4+,
supplements, the Starter Edition, The Traveller Book, JTAS, and the
_Consolidated Errata_ PDF. Do not open them for rules.

## Precedence

Three sources can answer a rules question. They rank, and a lower rank never
overrides a higher one:

1. **The held page.** Book 1, and Books 2–3 where Book 1 points at them. If
   the page settles it, nothing else is consulted — not memory, not an
   erratum, not the past.
2. **`rules`' readings, decided here.** Each was decided against the page
   and carries its cite, so it governs wherever the page is silent
   or self-contradictory.
3. **The errata of the implementation removed at `41a213a`.** Last resort,
   and only where 1 and 2 are both silent — that is, a gap this pass has not
   yet found.

A third-rank answer is never applied as it stands. Take it as a **pointer to
a page**, go read that page, and decide the reading here: it earns an
erratum and a `Reading` value in `rules`, its own rationale, and its own
stamping condition, at which point it is a rank-2 reading like any other
and its origin is history. A reading inherited unread has an authority it never earned — that is the whole
reason the ordering puts it third rather than first.

## Clean room, this repo's own past included

Sibling repos are not imported from or copied; consult them only when
explicitly asked. **The same rule governs this repository's own history at
and before `41a213a`**, with the single exception the ordering above carves
out. That tree held a complete implementation: Go code, tests, table
transcriptions, a policy table, goldens and a verified worked-example
reproduction. **None of that comes forward at any rank.** Do not read the old
code, tests, transcriptions, fixture roster or policy table — a transcription
re-used is the font trap uncaught, and a policy row re-used is a decision
never made.

What may be consulted, and only at rank 3 above, is that tree's **errata** —
its list of places the book is silent, and the readings it reached there.
The v1 contract carries the list — `docs/PRD.md` as of `ec7289f`; the
readings are the fallback.

## Documents and what each governs

| File                       | Governs                                                              |
| -------------------------- | -------------------------------------------------------------------- |
| `docs/CODING_STANDARDS.md` | The rules a reviewer holds a diff to: the judgements no check makes. |

Each fact lives in one place, so there is no second copy to hold to the
first. The readings are values in `rules`, one per entry of
`traveller.Errata`, so a reading missing or one too many does not compile,
and a test holds `Errata` to the `Erratum` constants. The auto run's policy
is its own doc comments. The coverage map is the markers beside the code, and a
rule an implementation names and no test holds, or the reverse, fails
`internal/coverage`; the map is printed on demand and never committed. Each
gate is verified by breaking it.

## The shape of the tree

The arrows point one way: `traveller` imports none of the others; `rules` and
`chargen` import `traveller`; `render` imports `chargen` for the character and
`traveller` for its values, and `rules` for the readings it writes. A domain type that
needs to know about dice or markdown is in the wrong package. Each package's doc
comment is its contract; read it before changing the package.

Every package but the command is under `internal/`: there is no Go library,
so nothing outside the module can import one and no exported name is a
promise (#199).

- `traveller`: the domain. Alphabets, values, and the sums that say "exactly
  one of".
- `rules`: every table of pp. 4–25, transcribed and lifted once, and the
  readings, declared as values. The lift validates the tables' cells and
  names; their values are the second transcription's.
- `dice`: an unseeded stream. The note is the only way back (ADR 0001, ADR 0006).
- `chargen`: the procedure, the `Decider` it asks, and the auto run's `Policy`,
  which describes itself. Its tests hold the readings and the choice points to
  the goldens.
- `render`: the note — its properties, summary, sheet and generation record,
  from one view of the character — and the readings written out, in full or as an index.
- `internal/coverage`: test code only. Holds the `Implements:` and `Holds:`
  markers to each other and their cites to what exists, writing nothing;
  `-print` shows the map.
- `internal/mutate`: a tool, not part of the CLI. Builds a scratch copy of
  the tree with one exact edit made, runs a command against it, and says
  whether the edit was killed.
- `internal/scripted`: test support, imported only by test files. The strict
  roller that fixtures and walks both drive their runs through.
- `cmd/ctchargen`: the subcommands. The note goes to stdout and the
  conversation to stderr, so `ctchargen new > kira.md` saves a note.

## Commands

`task` is the gate and the default. `task --list` carries the current set;
narrower than a task, in a loop:

```sh
go test ./internal/chargen                   # one package
go test ./internal/chargen -run TestGoldens  # one test
go test ./internal/chargen -regenerate       # rewrite the goldens
(cd internal/coverage && go test -run TestTheCoverageMapHolds -print)  # print the coverage map
go run ./internal/mutate -file F -old TEXT -new TEXT go test ./pkg -run T  # show a mutation killed
go run ./cmd/ctchargen new --auto
```

- **A package that passes on its own can still move the gate.** `-race`, the
  linters and the formatting check run only under `task`. Run `task` before
  calling a change done.
- **`task fix` is not a safe blind operation here**, on a repository that
  quotes a primary source. Two of its fixers have already rewritten this tree
  incorrectly: `godot` appended a period after a closing quotation mark in
  doc comments, and `dupword` deleted a word from a verbatim quotation
  of Book 1 p. 3. Both are configured off that behaviour now. Read the diff.

## The gate and what it holds

- **The gate is `task`, and only a local run holds it.** There is no CI
  (#149), so a change is checked when someone runs `task` and not otherwise.
  gofumpt runs inside
  golangci-lint, so there is one definition of formatted: format with
  `golangci-lint fmt`, never a standalone `gofumpt`, whose version and rule
  set need not match `.golangci.yml`.
- **Coverage is not measured (#198).** The gate writes no coverage profile
  and holds no count. What keeps a test honest is the mutation rule below,
  and a test that holds nothing is not kept.
- **Dice-stream consumption order is load-bearing for the fixtures.** A 2D
  throw is two dice in sequence, and a throw the procedure does not make
  consumes nothing. Changing the order changes which path every fixture
  walks; that is an ordinary change, not a breaking one, and the fixture's
  dice are re-scripted from a record that walks the new path. No failure
  names the throw that moved: a script says only where its dice ran out or
  were left unread, and some changes fail nothing but a golden. So a golden
  diff after a change to consumption is read before it is regenerated, or the
  fixture is silently re-pathed.
- **Goldens are regenerated, never hand-edited.**
- **The CLI is held to a baseline (#126).** Its flags, types and defaults are
  typed out in
  `cmd/ctchargen/cli_test.go` and held one way, because a new flag breaks
  nobody; the exit status beside them is 0 or 1.
- **A new invariant is not done until a deliberate mutation has been shown to
  kill it**, and the failure names what was broken. Show it with
  `go run ./internal/mutate`, which makes the edit in a scratch copy and never
  in the tree, prints the command's output, and says `KILLED` or `SURVIVED`,
  or `NOT BUILT` for a mutation that does not build, so no test ran.
- **An erratum's stamping condition is machine-checked, not just reachable**
  (#66). Each reading states it as a predicate over the finished record, and
  `internal/chargen/stamping_test.go` holds what a record stamps to what the
  reading says it should, quoting its stamped-on prose when it fails. Write the
  predicate from the reading's prose, never from the stamping code: a condition
  transcribed from the thing it checks is one reading written twice, which is
  the same trap rule 3 above exists for.

## Working conventions

- **The review rules are `docs/CODING_STANDARDS.md`**: sweeping a removal by its
  vocabulary, verbatim quotation, predicates written from the prose, marker
  placement, mutations shown. Read it before
  calling a change done; the reviewer holds the diff to it.
- **What a change does to the narrative documents.** The global release-docs
  rule brings `README.md` and `CLAUDE.md`
  current at release; here a change also sweeps them as `docs/CODING_STANDARDS.md`'s
  "A removal is swept by its vocabulary" says — every sentence that still
  tells the old story, not only the lines that name it — and adds its bullet
  to the changelog's Unreleased section. Everything else in them waits for
  the release pass.
- The history is squash-merged PRs with sentence-case subjects that say what
  changed ("Correct two typed ranges, name the font trap, and add milestone 0").
- **Release snapshots.** `docs/THEORY.md` and `docs/WALKTHROUGH.md` describe the
  release they were written for, not `main`, and each opens with a stamp
  naming that tag. The release prep PR regenerates them with `/code-theory`
  and `/code-walkthrough`, overwriting the last release's copies, on the
  release plan's `prep_branch` (`release/<version>`, from
  `release-plan.sh <version>`); the tag step finds the prep merge by that
  branch. Between
  releases they are ignored: no change reads, sweeps or updates them, drift
  in them is never a finding, and nothing in the gate holds them.

## Agent skills

### Issue tracker

Issues live in GitHub Issues for `philoserf/ctchargen`, worked through `gh`.
See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical labels, unchanged: `needs-triage`, `needs-info`,
`ready-for-agent`, `ready-for-human`, `wontfix`. See
`docs/agents/triage-labels.md`.

### Domain docs

Single-context: `docs/GLOSSARY.md` and `docs/adr/`, created lazily. See
`docs/agents/domain.md`. The glossary is in `docs/` whatever a skill assumes
about the root: never create a `GLOSSARY.md` at the repo root.
