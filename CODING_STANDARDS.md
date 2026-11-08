# Coding standards

The rules a reviewer holds a diff to. Each is a judgement no check makes;
what a check makes — formatting, lint, links, the coverage map's markers, the
readings' lift — is the gate's, and is not repeated here. `CLAUDE.md` carries
the authority rules every change answers to; these are the ones a diff can be
read against.

## A removal is swept by its vocabulary

When a change removes, moves or renames something, every sentence that still
tells the old story is a finding: a Go comment, a doc comment, a test's failure
message, a document. Grep for the words the code and documents used to describe
the thing — what it did, what it promised, what held it ("the gate", "the
document", "row", "decision table") — not only its identifiers and paths. A
name-only sweep left stale prose in every ticket of #203 and #204.

## A quotation from the page is verbatim

Comments and documents quote Book 1 to 3 to show where a rule comes from. A
quotation moved, rewrapped or retyped must match its source word for word,
punctuation and emphasis included; compare it against the same quotation
elsewhere in the tree. A reading's stamped-on prose is quoted by lifting it
(`rules.Reading`), never by a copy beside the predicate.

## A stamping predicate is written from the prose

`CLAUDE.md`'s gate section states the rule (#66, #77). Review a predicate in
`internal/chargen/stamping_test.go` by reading it against its reading's stamped-on
prose, not against the stamping code: one that matches the code and departs
from the prose is the defect the check exists for.

## A marker sits on what carries its rule

An `Implements:` marker belongs on the declaration that carries the rule — the
function that applies it, the table that holds it — and not on a helper beside
it or on the test that holds it. A gate's rule is its package's, marked on the
package doc.

## A new invariant is shown killed

`CLAUDE.md`'s gate section states the rule. Review for its evidence: a new
invariant, or a check rewritten, comes with the mutation that killed it and
what the failure said, in the commit or the PR.

## A walkthrough excerpt matches the tree

`WALKTHROUGH.md` labels each code excerpt `` `path` — `symbol` `` and each
transcript with the command that produced it. A change that edits a quoted
symbol, or changes what a transcribed command prints, leaves the excerpt or
transcript wrong; it is brought into line in the same change, from the source
or by re-running the command, never by hand from memory.

The transcripts are held by the gate: `internal/walkthrough` re-runs each one
and fails on a difference, and `go test ./internal/walkthrough -regenerate`
rewrites them. A run whose output depends on the dice is labelled "A sample
of", is held only to still run, and keeps the output its prose describes. The
excerpts are not held by the gate, so they are still the reviewer's to check.
