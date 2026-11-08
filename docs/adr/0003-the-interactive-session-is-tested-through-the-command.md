# The interactive session is tested through the command

An architecture review proposed pulling the interactive session out as a
module whose questions a test could drive directly, so that re-asking an
unreadable answer, not reprinting the menu, telling a failed read from the
input ending and saying what a stopped session lost could each be tested
without a scripted walk. Decided 2026-11-07: no change. The player already is
that module — `newPlayer` builds it, and its interface is the Decider methods
and the watcher — so there is nothing shallower to pull out, and the walk
those tests pay for is one argument.

What they test is what the person at the terminal sees: the questions and
complaints on stderr and never in the record on stdout, the line that says
what was answered and that nothing was written, and the error `run` returns
for input that ran out against a read that failed. Driving the player's
methods directly would keep the wording and drop that wiring, which is the
half a person notices when it breaks. So the interactive session's tests go
through the command, and a walk that stops by design is played `partly`.
