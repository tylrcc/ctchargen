# ctchargen

![Status: Release candidate](https://img.shields.io/badge/Status-Release%20candidate-orange.svg)

A Go CLI that generates rules-accurate Classic Traveller characters.

Ruleset baseline: **Books 1–3 only** — the FFE reprints of the © 1977 text.
Character generation is Book 1 pp. 4–25; Books 2 and 3 are consulted only
where Book 1 points at them. Every implemented rule carries its printed-page
cite, and every place the text is silent or ambiguous has a recorded reading
in the source, with its page cite.

## Install

```sh
go install github.com/philoserf/ctchargen/cmd/ctchargen@v1.0.0-rc.4
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
is reproduced by the engine, and characters can be written to files or
generated in batches as markdown notes. `new` without `--auto` walks the
procedure a question at a time, showing each throw between the questions.

Every command lists its own flags, which is where the current set lives:

```sh
ctchargen --help          # the commands
ctchargen new --help      # one command's flags, with their values and defaults
```

**Flags are Go's**: one dash or two (`-auto` or `--auto`). A flag that takes a
value has it after `=` or a space; a switch such as `--auto` takes none; and
`--` where a flag could stand ends them. The command line refuses what it
cannot read one way: a flag given twice, a flag given an empty value, and a
`--count` that is not decimal digits (`010` is ten; `0x10` is refused).

## Using it

Three commands. `new` generates one character; `batch` generates many; and
`version` writes the build.

```sh
ctchargen new --auto --service merchants
ctchargen new --auto --name "Alexander Jamison" -o jamison.md
ctchargen new                                   # asks at every choice point

ctchargen batch --count 20 --service merchants
ctchargen batch --count 20 --service merchants -o characters/
```

**A character is written as one markdown note**: the sheet — the UPP line,
skills, possessions and the service a term at a time, as Book 1 summarises a
character — and under it the generation record, every die thrown and every
choice made, step by step with its page cite, each choice saying whether the
policy or the player made it. The note ends with the record, and drops into a
notes vault as it is; its `ctchargen` property names the build that wrote it.

**A note opens with properties** a vault can query. Every note carries every
one: a text property with no value is left empty, and a list with none is
`[]`. Their names and meanings freeze at `v1.0.0`.

| Property              | Type   | Holds                                                                         |
| --------------------- | ------ | ----------------------------------------------------------------------------- |
| `name`                | text   | the name given, or empty                                                      |
| `aliases`             | list   | the name given, or `[]`                                                       |
| `upp`                 | text   | the six characteristics in the book's hexadecimal                             |
| `strength` … `social` | number | each characteristic                                                           |
| `age`                 | number | completed years                                                               |
| `service`, `rank`     | text   | the one service and the rank held in it, or empty                             |
| `terms`               | number | every term begun, the one a death ended included                              |
| `status`              | text   | `alive` or `dead`                                                             |
| `departure`           | text   | how the service ended, such as `retired` or `denied reenlistment`             |
| `skills`              | list   | skill names, one each, with no level: the sheet has the levels                |
| `credits`, `pension`  | number | cash mustered out with, and annual retirement pay                             |
| `nobility`            | text   | the title assumed (Book 3 p. 22's pairs, such as `knight/dame`), or empty     |
| `possessions`         | list   | what mustering out gave besides cash, one item each, or `[]`                  |
| `ctchargen`           | text   | the version of the build that wrote the note, or empty where none is embedded |

Each skill and possession is an exact item, so "which crew have Pilot?" is an
exact match: `econtains(skills, "Pilot")` in Dataview, and
`skills.contains("Pilot")` in Bases. Under the heading, **a one-paragraph summary** says who the character is in words a
referee can read aloud: "Army Captain, 2 terms, denied reenlistment at 26. Known for
Rifle-2. Musters out with CR 30000."

**A reading is how the tool decides where the page is silent or contradicts
itself.** The readings are the tool's own record of its authority, kept in
the source with their page cites, and a note names none of them (ADR 0007).

**The dice are not seeded, and the note is the only way back to a
character.** Every run rolls fresh dice; nothing brings a character back but
the generation record in its note, which carries every die thrown and every
choice made.

**A batch writes into a folder of its own**: `-o` names a folder that is new,
or empty but for hidden files such as `.DS_Store`. A new one is made inside a
folder that exists, and nothing above it is made. Anything else in it — a
note, a picture, a subfolder — refuses the batch before anyone is generated,
so a batch never mixes with your notes and never removes anything. To run a
batch again, remove its folder or name another. Each member's note is named
for its position and the character in it — `01 Army Captain 938788.md`,
`02 Civilian 87665B.md` — the position padded to the width of the batch.
A batch names nobody: each note is headed by its title, `# Army Captain
938788`, and you name a member in your vault by setting the note's `name`
and `aliases` properties, and its heading if you like.

**`batch` with no `-o`, or `-o -`, writes the notes to standard output**, one after
another, and it streams, so the first note arrives before the run finishes
rather than after.

**A batch is always decided by the policy**, because it has nobody to ask, so
it takes no `--auto`.

**A batch writes only the living.** The rules kill a good many characters —
the default `serve` strategy re-enlists until the service refuses reenlistment, so
the survival throw gets most Scouts in the end — and a batch passes over the
dead and generates another until it has the number asked for. It doesn't
reroll anybody: a dead character is written nowhere and another is generated
to fill that slot, so every note in the batch is one the dice made. A batch says
what it did on standard error, so it never lands among the notes:

```sh
$ ctchargen batch --count 20 --service scouts -o scouts/
20 written, 68 passed over for dying
```

A dead character still comes from `new`, which writes whatever the dice make.
A batch that generates a hundred characters per one asked for without
finding enough living gives up with an error rather than drawing forever.

**A session that stops before the end is lost.** A half-built character has
no note, so nothing is written; the tool says what was answered and that it
was lost:

```sh
$ ctchargen new --service other
...
Answered 1, 2, 1; the session is lost and nothing was written.
```

**Without `--service` the tool picks one**, and on an auto run (`new --auto`,
or any batch) the policy draws one of the six at random, so a batch mixes
them rather than filling with whichever is easiest to join. The sheet says so
on its headline. Naming a service does not guarantee you get it either: a
failed enlistment throw sends the character to the draft, which may put the
character somewhere else entirely — the sheet says that too.

**Three flags steer the policy** where the procedure offers a choice, on
`new --auto` and on `batch`. `--career` governs the draft, commission and
promotion attempts, reenlistment and the title; `--skills` the skills table
and the weapon a combat result names; `--muster` the mustering-out table, its
two DMs and repeated weapons. What each strategy answers at each choice is in
the doc comment of the `chargen.Policy` method that asks it. On `new` without
`--auto` the three are refused: you answer every choice they would have, and no
choice in the note is the policy's.

| Flag       | Values                                        |
| ---------- | --------------------------------------------- |
| `--career` | `serve` (default) · `retire` · `oneterm`      |
| `--skills` | `advanced` (default) · `service` · `personal` |
| `--muster` | `cash` (default) · `goods` · `spartan`        |

**Nothing is overwritten without `--force`.** `new -o` writes into a folder
that exists, and `-o -` is standard output, as no `-o` is. `new -o` onto an
existing file is refused unless `--force` is given, and a write without it
that fails leaves no note. `--force` overwrites the file in place, as the shell's `>`
does, so the file keeps its mode and its links, and a symlink is followed to
the note it names; a write that fails partway has already lost the old note.
`--force` with no `-o` file is refused, since it has nothing to overwrite. A
`batch` has no `--force`: it writes only into a folder of its own. Notes and
folders are created with the mode your umask gives them.

**A name is one line.** `new --name` is printed as the note's heading, so a name
carrying a line break is refused rather than allowed to start sections the
character does not have.

Death is an outcome and not an error: a character `new` makes who is killed by
a survival throw (Book 1 p. 5) gets a complete note like anyone else, and
nothing rerolls the character; a batch passes the dead over.

**The exit status is 0 or 1.** 0 when the command did what it was asked —
the dead character above included — and 1 when it did not, with the reason on
standard error after `ctchargen:`.

## The documents

| File                                                     | What it holds                                                                    |
| -------------------------------------------------------- | -------------------------------------------------------------------------------- |
| [internal/rules/readings.go](internal/rules/readings.go) | Every recorded reading, with its page cite and stamping condition, as Go values. |
| [docs/CHANGELOG.md](docs/CHANGELOG.md)                   | What each release changed.                                                       |
| [docs/GLOSSARY.md](docs/GLOSSARY.md)                     | The tool's vocabulary: note, record, reading, erratum, golden.                   |
| [docs/adr/](docs/adr/)                                   | The decisions that shape the tool, and why.                                      |
| [CLAUDE.md](CLAUDE.md)                                   | Authority, source precedence, and the working rules for agents.                  |

## The gate

```sh
task
```

`go mod tidy -diff`, golangci-lint (go vet's analyzers among its own), NilAway, govulncheck,
`go test -race`, `bun fmt` over the non-Go files, and lychee over the
documents' links. The tests hold what `v1.0.0` freezes: each command's flags,
defaults and exit status, and the note's property names. There
is no CI (#149); a change is checked when this is run, and not otherwise.

On a fresh machine, `brew install go-task && task setup` installs every tool
the gate runs: the Brewfile's, and NilAway with `go install`, since Homebrew
has no formula for it. NilAway lands in Go's bin directory (`go env GOPATH`'s
`bin`), which must be on your `PATH`.

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
