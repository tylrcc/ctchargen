# Coding standards

The rules a reviewer holds a diff to. Each is a judgement no check makes;
what a check makes — formatting, lint, links, the coverage map's markers, the
readings' count — is the gate's, and is not repeated here. `CLAUDE.md` carries
the authority rules every change answers to; these are the ones a diff can be
read against.

## A removal is swept by its vocabulary

When a change removes, moves or renames something, every sentence that still
tells the old story is a finding: a Go comment, a doc comment, a test's failure
message, a document. Grep for the words the code and documents used to describe
the thing — what it did, what it promised, what held it ("the gate", "the
document", "row", "decision table") — not only its identifiers and paths. A
name-only sweep left stale prose in every ticket of #203 and #204.

## A comment claims no more than the code does

A doc comment, a note or a commit message that says what the code does is
read against the code, and a claim wider than the code is a finding. Check
hardest where the claim is universal — every, all, each, never, none, only —
by naming each case it covers and finding it in the code: "every table" is
every table in the file, not the one the change touched. A reading credited
with a decision is held to what that reading's prose decides. Four overclaims
reached review in #346, #354 and #359; three were true of the case in hand
and false of the rest, and one credited E005 with a convention of the code's
own.

## A quotation from the page is verbatim

Comments and documents quote Book 1 to 3 to show where a rule comes from. A
quotation moved, rewrapped or retyped must match its source word for word,
punctuation included; compare it against the same quotation elsewhere in the
tree. The page's emphasis is marked where the text is Markdown and not
elsewhere: a reading's values mark italics as `_…_`, as the `Reading` type
says, and a Go comment quotes the same words plain (#526). A reading's stamped-on prose is quoted from its value
(`rules.Readings`), never by a copy beside the predicate.

## A stamping predicate is written from the prose

`CLAUDE.md`'s gate section states the rule (#66, #77). Review a predicate in
`internal/chargen/stamping_test.go` by reading it against its reading's stamped-on
prose, not against the stamping code: one that matches the code and departs
from the prose is the defect the check exists for.

## A marker sits on what carries its rule

An `Implements:` marker belongs on the declaration that carries the rule — the
function that applies it, the table that holds it — and not on a helper that
only serves that declaration. A helper that carries a rule of its own carries
that rule's marker, and a rule carried by several declarations is marked on
each. For a promise of the tool's own that the test harness
keeps, the carrier is test code, a helper or a roster: the stamping conditions
and the fixture roster carry theirs, and the step lookup the conditions read
records with carries its own. It is never a test: a test holds a rule rather
than carrying it. A promise no single declaration carries is marked on its
package's doc, whether it is spread across the package, as the alphabets'
spellings are, or is the whole package, as the coverage check's is.

## A new invariant is shown killed

`CLAUDE.md`'s gate section states the rule. Review for its evidence: a new
invariant, or a check rewritten, comes with the mutation that killed it and
what the failure said, in the commit or the PR. `go run ./internal/mutate`
prints both, so the evidence can be pasted as it ran.
