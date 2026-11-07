# ctchargen

![Status: Beta](https://img.shields.io/badge/Status-Beta-yellow.svg)

A Go CLI that generates rules-accurate Classic Traveller characters.

Ruleset baseline: **Books 1–3 only** — the FFE reprints of the © 1977 text.
Character generation is Book 1 pp. 4–25; Books 2 and 3 are consulted only
where Book 1 points at them. Every implemented rule carries its printed-page
cite, and every place the text is silent or ambiguous has a recorded reading.

## Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-beta.4
```

Go excludes prereleases from `@latest`, so a prerelease installs by name. That
is the intended friction.

**Do not install `v1.0.0-alpha.1` or `v1.0.0-alpha.2`.** Both predate the
rebuild at `41a213a` and are builds of a different implementation — they have
a `replay` subcommand and flags this tool does not. Their release notes are
accurate about the tag they head and describe nothing below.

From a clone:

```sh
go build ./cmd/ctchargen
```

## Status

All six services generate, the book's own worked character (pp. 23–25)
replays against the engine, the record has a schema, and characters can be
written to files, generated in batches and read back. `new` without
`--auto` walks the procedure a question at a time, showing each throw
between the questions.

Every command lists its own flags, which is where the current set lives:

```sh
ctchargen --help          # the commands
ctchargen new --help      # one command's flags, with their values and defaults
```

## Using it

Four commands. `new` generates one character; `batch` generates many; `render`
reads a record back as a sheet or as the transcript; and `version` writes the
build.

```sh
ctchargen new --auto --service merchants --sheet
ctchargen new --auto --name "Alexander Jamison" -o jamison.json
ctchargen new --auto --history                  # the transcript, throw by throw
ctchargen new --sheet                           # asks at every choice point

ctchargen batch --count 20 --auto --service merchants
ctchargen batch --count 20 --auto --service merchants -o characters/

ctchargen render characters/01.json
ctchargen render --history characters/01.json
```

**The dice are not seeded, and the record is the only way back to a
character.** Every run rolls fresh dice; nothing brings a character back but
the JSON record, which carries every die thrown and every choice made, and
which every sheet ends by telling you to keep. `render` reads one back.

Batch members are named by their position, `01.json` through `20.json`, padded
to the width of the batch.

**`batch` with no `-o` writes NDJSON to standard output**, one record to the
line, which is the shape that pipes — and it streams, so the first record
reaches the pipe before the run finishes rather than after:

```sh
ctchargen batch --count 100 --auto | jq -r '[.upp, .service, .terms] | @tsv'
```

`batch` requires `--auto`, because it has nobody to ask.

**A batch says what it did**, on standard error so it cannot corrupt the JSONL:

```sh
$ ctchargen batch --count 100 --auto --service scouts > roster.jsonl
100 written, 76 died
```

That is the rules working — the default `serve` strategy re-enlists until the
service throws him out, so the survival throw eventually gets most of them.
When what you want is twenty scouts for the starport rather than a hundred
outcomes, **`--survivors`** passes over the dead and generates another:

```sh
$ ctchargen batch --count 20 --auto --service scouts --survivors > roster.jsonl
20 written, 68 passed over for dying
```

It does not reroll anybody: a dead character is written nowhere and another is
generated in his place, so every record in the roster is one the dice made.

**A session that stops before the end is lost.** A half-built character is
not a record, so nothing is written; the tool says what was answered and that
it was lost:

```sh
$ ctchargen new --service other
...
Answered 1, 2, 1; the session is lost and nothing was written.
```

**Without `--service` the tool picks one**, and with `--auto` that pick is the
policy's rather than yours. The sheet says so on its headline, and so does the
record. Naming a service does not guarantee you get it either: a failed
enlistment throw sends the character to the draft, which may put him somewhere
else entirely — the sheet says that too.

**Three flags steer `--auto`** where the procedure offers a choice, on both
`new` and `batch`. [`docs/POLICY.md`](docs/POLICY.md) carries a row per choice
point saying what each one does:

| Flag       | Values                                        |
| ---------- | --------------------------------------------- |
| `--career` | `serve` (default) · `retire` · `oneterm`      |
| `--skills` | `advanced` (default) · `service` · `personal` |
| `--muster` | `cash` (default) · `goods` · `spartan`        |

**Nothing is overwritten without `--force`.** `-o` onto an existing file is
refused, and so is a `batch` any of whose members would replace one — the
whole batch, before a byte is written, so a refusal leaves the directory as it
was.

**A name is one line.** `--name` is printed as the sheet's heading, so a name
carrying a line break is refused rather than allowed to start sections the
character does not have.

Death is an outcome and not an error: a character killed by a survival throw
(Book 1 p. 5) gets a complete record like anyone else, and no flag rerolls him.

**The exit status is 0 or 1.** 0 when the command did what it was asked —
the dead character above included — and 1 when it did not, with the reason on
standard error after `ctchargen:`.

## The documents

| File                                                     | What it holds                                                                    |
| -------------------------------------------------------- | -------------------------------------------------------------------------------- |
| [docs/PRD.md](docs/PRD.md)                               | The delivered v1 contract — historical, kept for why the tree has this shape.    |
| [docs/ERRATA.md](docs/ERRATA.md)                         | Every recorded reading, with its page cite and its stamping condition.           |
| [docs/POLICY.md](docs/POLICY.md)                         | The `--auto` decision table, one row per choice point.                           |
| [docs/COVERAGE.md](docs/COVERAGE.md)                     | Every implemented rule mapped to its page cite, its implementation and its test. |
| [docs/character.schema.json](docs/character.schema.json) | What the tool writes, with a minimal and a complete example beside it.           |
| [docs/PRERELEASE.md](docs/PRERELEASE.md)                 | What each tag ships with open, and the review that preceded it.                  |
| [THEORY.md](THEORY.md)                                   | What the code believes and where each belief is enforced. Binds nothing.         |
| [WALKTHROUGH.md](WALKTHROUGH.md)                         | A linear read of the code in execution order. Binds nothing.                     |
| [CLAUDE.md](CLAUDE.md)                                   | Authority, source precedence, and the working rules for agents.                  |

## The gate

```sh
task
```

`go mod tidy -diff`, `go vet`, golangci-lint, NilAway, govulncheck, `go test -race`, and
`bun fmt` over the non-Go files. The tests hold the three things `v1.0.0`
freezes: every record against the schema, the exported Go API against a
baseline, and each command's flags, defaults and exit status. There is no CI
(#149); a change is checked when this is run, and not otherwise.

The toolchain is unpinned on purpose, and golangci-lint runs with
`default: all`, so a linter added upstream arrives switched on. Formatting is
`gofumpt`, run _inside_ golangci-lint rather than beside it, so there is one
definition of formatted rather than two that can disagree.

## Licence

[MIT](LICENSE).
