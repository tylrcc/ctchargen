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
what the failure said, in the commit or the PR. `go run ./internal/mutate`
prints both, so the evidence can be pasted as it ran.

## The record's shape

**The record freezes at `v1.0.0`, and is free to move before it (#62).** Until
that tag, a record written by one build is not promised to be readable by the
next; from it, the shape is a public contract and a build that changes it says
so in the record itself. `record` is the field it says it in: a required
integer set in `project()` beside every other field, so it is present on every
record, and counting the shape and nothing else — `build` names the writer and
`ruleset` names the source text, and neither says what the file looks like.

**`additionalProperties` is open at the root, and closed at every object
inside it (#113).** A later build may add a top-level section, and a reader
holding this schema will load the record and render what it understands. A key
added inside an existing object is still refused: the fourteen nested objects
are what a sheet is rendered from, and pinning them is what keeps the schema
worth validating against. **An event kind is closed the same way (#129):** the
log is one of the five kinds the procedure writes, because the schema describes
the rules we have and not an extension nobody has made. `render` printing
`(unknown event kind %q)` is how a reader survives a record it cannot
validate, and not a reason to validate it.

**`record` is 1 until `v1.0.0` and counts from there (#96).** Not from every
change: the record moves freely before the freeze, so a build that changes the
shape before it leaves the number alone, and two pre-release shapes that both
say 1 are what that freedom costs. Do not bump it to mark a pre-v1 change —
that is the reading #96 declined. The **schema** accepts any shape from 1
upward, which is a different statement: a reader must be able to validate a
record from a later build rather than refuse it before looking at its contents,
so `minimum` is what the schema says and `TestThisBuildWritesShapeOne` is what
holds this build to 1.
