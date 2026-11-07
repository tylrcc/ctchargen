package render

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// EventLine renders one event for a reader watching a generation happen
// rather than reading it afterwards.
//
// Throws, rolls and outcomes go through the same writeEvent the transcript
// does: the wire shape and the domain shape both converge on eventJSON
// before anything is written, so those lines cannot come to differ.
//
// Steps and choices are written here instead, and deliberately do differ.
// A step's heading is unnumbered in the transcript, and a choice is one the
// player has just answered; leaving either out would leave a gap a watcher
// reads as a lost event, so both are written with their number. The choice is echoed
// as that number and the answer alone rather than the transcript's full
// form, because the player has just been asked the question and can still
// see the point, the decider and the alternatives above him; repeating them
// pushes the useful lines off the screen.
//
// It returns no error, as foldDeparture does not: the codec's cases fill
// struct fields and marshal shapes that cannot fail to marshal, and
// traveller.Event is sealed, so no further case can arrive from outside to
// fail either. An error return here would be a branch no test could reach.
//
// Implements: The procedure is shown between the questions [FR11] in The
// record.
func EventLine(event traveller.Event) string {
	var lined eventCodec

	_ = event.Fold(&lined)

	// The gap these two close is what made the numbers appear to skip - 17,
	// 18, then 20. Nothing was ever missing: the headings printed without
	// their number, and the questions printed nothing at all.
	switch lined.flat.Kind {
	case kindStep:
		return fmt.Sprintf("\n## %d. %s (%s)\n\n",
			lined.flat.Seq, lined.flat.Step, lined.flat.Pages)
	case kindChoice:
		// Echoed after the answer, because a choice's number does not exist
		// until it is made - the prompt cannot carry it. The transcript's
		// own line names the point, the decider and the alternatives, all
		// of which the person who just typed the answer can see above.
		return fmt.Sprintf("%3d. you chose %s\n", lined.flat.Seq, lined.flat.Chosen)
	}

	var out strings.Builder

	writeEvent(&out, lined.flat)

	return out.String()
}
