# The transcription is the page's text, in Go

ADR 0004 made the readings code and left the tables where they were: seven
JSON files under `internal/rules/data`, embedded and lifted once on first use.
The JSON bought one thing Go does not, a corner of the tree no linter reads,
and cost the rest: a key misspelled in a file left its field at zero and the
lift accepted it (#317), the files' notes forced the lift to tolerate unknown
keys, and a wire layer and a set of shape-only tests existed because the
compiler could not see the data. Decided 2026-11-08: the tables become Go
literals in one file, `internal/rules/transcription.go`, each table's cite and
note becoming its doc comment verbatim, and `data/` and the last `//go:embed`
go.

What moves into Go is the page's text, not the page's meaning. The first
transcription records what is printed: its glyphs, its abbreviations, its
notation and its dashes. The lift still turns that text into the domain's
values, applying the normalizations the readings record, E012's among them,
and reading a dash as nothing, and it still refuses what the page could not
have printed. The alternative was to
type the tables as domain values directly and delete the lift. A broken glyph
would then fail to compile, which is stronger than a refused name; but every
normalization and every dash would be decided in the transcriber's head with
no code to name it, which is the reading applied silently that the authority
rules exist to prevent. The lift stays because it is where those judgements
are written down.

The transcription's structure comes from the domain and its cells from the
page. Which table a grid is and which service a column belongs to are
positions or keys built from `traveller`'s types, so a misspelled table or a
column out of order does not compile; only the cells are printed text, and the
page's rows stay rows, in the page's order. A short Go array is padded with
empty strings in silence, so the lift refuses an empty cell, since the page
prints a dash where it means nothing. `rules.Load` keeps
its error: a literal can still hold text the lift refuses.

The second transcription, in the `rules` tests, is unchanged. It is laid out
apart from the page's grid, the service tables a service at a time, and written
in the normalized vocabulary, so it is independent of the first in shape and in
words; that independence is what the font trap needs. CLAUDE.md's rule that a
number indexing a printed table is data follows the move: such a number
belongs in the transcription.

Each table moves with its lifted values proven equal to what the JSON lifts,
and the files go only when the last has moved. Restoring the page's own text
where today's files depart from it (abbreviated characteristics in the DM
conditions, and the dashes the page prints) is a separate change made after
the move. The move carries today's text unchanged, so it changes nothing a
record or a referee sees.

What is given up: the transcription is no longer a file a linter ignores. A
prototype of the skills table drew findings from goconst alone, so its existing
exclusion is extended to the file, and any linter that later reads the cells
reads the page's text. The tables are no longer data a non-Go reader could
load, which nothing in the tree does.
