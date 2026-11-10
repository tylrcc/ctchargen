# ctchargen

![Status: Release candidate](https://img.shields.io/badge/Status-Release%20candidate-orange.svg)

A Go CLI that generates rules-accurate Classic Traveller characters.

Ruleset baseline: **Books 1–3 only** — the FFE reprints of the © 1977 text.
Character generation is Book 1 pp. 4–25; Books 2 and 3 are consulted only
where Book 1 points at them. Every implemented rule carries its printed-page
cite, and every place the text is silent or ambiguous has a recorded reading,
which the tool prints on request and names on every note it governs.

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
procedure a question at a time, showing each throw between the questions, and
a note can end with the full text of the readings its generation record names.

Every command lists its own flags, which is where the current set lives:

```sh
ctchargen --help          # the commands
ctchargen new --help      # one command's flags, with their values and defaults
```

## Using it

Four commands. `new` generates one character; `batch` generates many;
`readings` explains the readings a note names; and `version` writes the build.

```sh
ctchargen new --auto --service merchants
ctchargen new --auto --name "Alexander Jamison" -o jamison.md
ctchargen new --auto --readings                 # with each reading it names, in full
ctchargen new                                   # asks at every choice point

ctchargen batch --count 20 --service merchants
ctchargen batch --count 20 --service merchants -o characters/

ctchargen readings                              # every reading, and whether it is in force
ctchargen readings E002 E011                    # those two, in full
```

**A character is written as one markdown note**: the sheet — the UPP line,
skills, possessions and the service a term at a time, as Book 1 summarises a
character — and under it the generation record, every die thrown and every
choice made, step by step with its page cite. The note ends by naming the
ruleset, the build and who answered its choices, so it drops into a notes
vault as it is.

**A note opens with properties** a vault can query — `upp`, the six
characteristics, `age`, `service`, `rank`, `terms`, `status` (`alive` or
`dead`), `departure`, `skills` in the book's notation, `credits`, `pension`,
`nobility`, and the name with an alias when one was given — so Dataview or
Bases can answer "which crew have Pilot?" Their names freeze at `v1.0.0`.
Under the heading, **a one-paragraph summary** says who he is in words a
referee can read aloud: "Army Captain, 2 terms, forced out at 26. Known for
Rifle-2. Musters out with CR 30000." `age` is completed years, and `terms`
counts every term begun, the one a death ended included.

**A reading is how the tool decides where the page is silent or contradicts
itself**, and the generation record names the reading beside each outcome it
governed, as an id like `E002`. **`--readings`** ends the note with the full
text of each reading its record names: what the page says, the gap, the
reading and why. `ctchargen readings` lists them all, or writes the ones named
in full. The sheet leaves the ids out, since it is the page a player keeps.

**An id means one decision for good.** A reading may be reworded under its
id, but a changed decision gets a new id, so a note names the same readings
in every build. A reading whose decision no longer holds is withdrawn rather
than deleted: it keeps its id, still explains the notes already stamped with
it, says why, and is stamped on no new one. `ctchargen readings` marks each as
in force or withdrawn.

**The dice are not seeded, and the note is the only way back to a
character.** Every run rolls fresh dice; nothing brings a character back but
the generation record in its note, which carries every die thrown and every
choice made, and which every note ends by telling you to keep whole.

**A batch writes into a folder of its own**: `-o` names a folder that is new,
or empty but for hidden files such as `.DS_Store`. Anything else in it — a
note, a picture, a subfolder — refuses the batch before anyone is generated,
so a batch never mixes with your notes and never removes anything. To run a
batch again, remove its folder or name another. Each member's note is named
for its position and the character in it — `01 Army Captain 938788.md`,
`02 Civilian 87665B.md` — the position padded to the width of the batch.
A batch names nobody: each note is headed by its title, `# Army Captain
938788`, and you name a member in your vault by setting the note's `name`
and `aliases` properties, and its heading if you like.

**`batch` with no `-o` writes the notes to standard output**, one after
another, and it streams, so the first note arrives before the run finishes
rather than after.

**A batch is always decided by the policy**, because it has nobody to ask, so
it takes no `--auto`.

**A batch writes only the living.** The rules kill a good many characters —
the default `serve` strategy re-enlists until the service throws him out, so
the survival throw gets most Scouts in the end — and a batch passes over the
dead and generates another until it has the number asked for. It doesn't
reroll anybody: a dead character is written nowhere and another is generated
in his place, so every note in the batch is one the dice made. A batch says
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
failed enlistment throw sends the character to the draft, which may put him
somewhere else entirely — the sheet says that too.

**Three flags steer the policy** where the procedure offers a choice, on
`new --auto` and on `batch`. `--career` governs the draft, commission and
promotion attempts, reenlistment and the title; `--skills` the skills table
and the weapon a combat result names; `--muster` the mustering-out table, its
two DMs and repeated weapons. What each strategy answers at each choice is in
the doc comment of the `chargen.Policy` method that asks it. On `new` without
`--auto` the three are refused: you answer every choice they would have, and the
note names no policy for them to belong to.

| Flag       | Values                                        |
| ---------- | --------------------------------------------- |
| `--career` | `serve` (default) · `retire` · `oneterm`      |
| `--skills` | `advanced` (default) · `service` · `personal` |
| `--muster` | `cash` (default) · `goods` · `spartan`        |

**Nothing is overwritten without `--force`.** `new -o` onto an existing file
is refused unless `--force` is given. A `batch` has no `--force`: it writes
only into a folder of its own.

**A name is one line.** `new --name` is printed as the note's heading, so a name
carrying a line break is refused rather than allowed to start sections the
character does not have.

Death is an outcome and not an error: a character `new` makes who is killed by
a survival throw (Book 1 p. 5) gets a complete note like anyone else, and
nothing rerolls him; a batch passes him over.

**The exit status is 0 or 1.** 0 when the command did what it was asked —
the dead character above included — and 1 when it did not, with the reason on
standard error after `ctchargen:`.

## The documents

| File                                                     | What it holds                                                                                                      |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| [internal/rules/readings.go](internal/rules/readings.go) | Every recorded reading, with its page cite and stamping condition, as Go values. `ctchargen readings` prints them. |
| [docs/CHANGELOG.md](docs/CHANGELOG.md)                   | What each release changed.                                                                                         |
| [docs/GLOSSARY.md](docs/GLOSSARY.md)                     | The words a referee and the code share: note, record, reading, erratum, golden.                                    |
| [docs/adr/](docs/adr/)                                   | The decisions that shape the tool, and why.                                                                        |
| [CLAUDE.md](CLAUDE.md)                                   | Authority, source precedence, and the working rules for agents.                                                    |

## The gate

```sh
task
```

`go mod tidy -diff`, golangci-lint (go vet's analyzers among its own), NilAway, govulncheck,
`go test -race`, `bun fmt` over the non-Go files, and lychee over the
documents' links. The tests hold what `v1.0.0` freezes: each command's flags,
defaults and exit status, and the note's property names. There
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
