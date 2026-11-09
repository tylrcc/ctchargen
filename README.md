# ctchargen

![Status: Release candidate](https://img.shields.io/badge/Status-Release%20candidate-orange.svg)

A Go CLI that generates rules-accurate Classic Traveller characters.

Ruleset baseline: **Books 1–3 only** — the FFE reprints of the © 1977 text.
Character generation is Book 1 pp. 4–25; Books 2 and 3 are consulted only
where Book 1 points at them. Every implemented rule carries its printed-page
cite, and every place the text is silent or ambiguous has a recorded reading,
which the tool prints on request and names on every record it governs.

## Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-rc.2
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
is reproduced by the engine, the record has a schema, and characters can be
written to files, generated in batches and read back. `new` without
`--auto` walks the procedure a question at a time, showing each throw
between the questions, and a transcript can end with the full text of the
readings its record names.

Every command lists its own flags, which is where the current set lives:

```sh
ctchargen --help          # the commands
ctchargen new --help      # one command's flags, with their values and defaults
```

## Using it

Five commands. `new` generates one character; `batch` generates many; `render`
reads a record back as a sheet or as the transcript; `readings` explains the
readings a record names; and `version` writes the build.

```sh
ctchargen new --auto --service merchants --sheet
ctchargen new --auto --name "Alexander Jamison" -o jamison.json
ctchargen new --auto --history                  # the transcript, throw by throw
ctchargen new --sheet                           # asks at every choice point

ctchargen batch --count 20 --service merchants
ctchargen batch --count 20 --service merchants -o characters/

ctchargen render characters/01.json
ctchargen render --history characters/01.json
ctchargen render --history --readings characters/01.json

ctchargen readings                              # every reading, and whether it is in force
ctchargen readings E002 E011                    # those two, in full
```

**A reading is how the tool decides where the page is silent or contradicts
itself**, and a transcript names the reading beside each outcome it governed,
as an id like `E002`. **`--readings`** on `new --history` or
`render --history` ends the transcript with the full text of each reading the
record names: what the page says, the gap, the reading and why. `ctchargen
readings` lists them all, or writes the ones named in full. The sheet leaves
the ids out, since it is the page a player keeps.

**An id means one decision for good.** A reading may be reworded under its
id, but a changed decision gets a new id, so a record names the same readings
in every build that reads it. A reading whose decision no longer holds is
withdrawn rather than deleted: it keeps its id, still explains the records
already stamped with it, says why, and is stamped on no new one.
`ctchargen readings` marks each as in force or withdrawn.

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
ctchargen batch --count 100 | jq -r '[.upp, .service, .terms] | @tsv'
```

**A batch is always decided by the policy**, because it has nobody to ask, so
it takes no `--auto`.

**A batch says what it did**, on standard error so it cannot corrupt the JSONL:

```sh
$ ctchargen batch --count 100 --service scouts > roster.jsonl
100 written, 76 died
```

That is the rules working — the default `serve` strategy re-enlists until the
service throws him out, so the survival throw eventually gets most of them.
When what you want is twenty scouts for the starport rather than a hundred
outcomes, **`--survivors`** passes over the dead and generates another:

```sh
$ ctchargen batch --count 20 --service scouts --survivors > roster.jsonl
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

**Without `--service` the tool picks one**, and on an auto run (`new --auto`,
or any batch) that pick is the policy's rather than yours. The sheet says so on its headline, and so does the
record. Naming a service does not guarantee you get it either: a failed
enlistment throw sends the character to the draft, which may put him somewhere
else entirely — the sheet says that too.

**Three flags steer the policy** where the procedure offers a choice, on
`new --auto` and on `batch`. `--career` governs enlistment, the draft,
commission and promotion attempts, reenlistment and the title; `--skills` the skills table
and the weapon a combat result names; `--muster` the mustering-out table, its
two DMs and repeated weapons. What each strategy answers at each choice is in
the doc comment of the `chargen.Policy` method that asks it. On `new` without
`--auto` the three are refused: you answer every choice they would have, and the
record names no policy for them to belong to.

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

| File                                                     | What it holds                                                                                                      |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| [docs/character.schema.json](docs/character.schema.json) | What the tool writes, with a minimal and a complete example beside it.                                             |
| [internal/rules/readings.go](internal/rules/readings.go) | Every recorded reading, with its page cite and stamping condition, as Go values. `ctchargen readings` prints them. |
| [CHANGELOG.md](CHANGELOG.md)                             | What each release changed.                                                                                         |
| [GLOSSARY.md](GLOSSARY.md)                               | The words a referee and the code share: record, reading, erratum, golden.                                          |
| [docs/adr/](docs/adr/)                                   | The decisions that shape the tool, and why.                                                                        |
| [CLAUDE.md](CLAUDE.md)                                   | Authority, source precedence, and the working rules for agents.                                                    |

## The gate

```sh
task
```

`go mod tidy -diff`, `go vet`, golangci-lint, NilAway, govulncheck,
`go test -race`, `bun fmt` over the non-Go files, and lychee over the
documents' links. The tests hold the two things `v1.0.0` freezes: every record
against the schema, and each command's flags, defaults and exit status. There
is no CI (#149); a change is checked when this is run, and not otherwise.

Every implemented rule carries an `Implements:` marker with its cite, and the
test that holds it a `Holds:` marker; the tests refuse a rule with only one,
and a cite that names nothing. The map they make is printed on demand and
never committed:

```sh
cd internal/coverage && go test -run TestTheCoverageMapHolds -print
```

The toolchain is unpinned on purpose, and golangci-lint runs with
`default: all`, so a linter added upstream arrives switched on. Formatting is
`gofumpt`, run _inside_ golangci-lint rather than beside it, so there is one
definition of formatted rather than two that can disagree.

## Licence

[MIT](LICENSE).
