package render

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// EventLine renders one event for a reader watching a generation happen
// rather than reading it afterwards.
//
// Steps, throws, rolls and outcomes go through the same writeEvent the
// transcript does: the wire shape and the domain shape both converge on
// eventJSON before anything is written, so those lines cannot come to differ.
//
// A choice is written here instead, and deliberately does differ. It is one
// the player has just answered; leaving it out would leave a gap a watcher
// reads as a lost event, so it is written with its number. The choice is echoed
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
// Implements: The procedure is shown between the questions [Generation record]
// in The record.
func EventLine(event traveller.Event) string {
	var lined eventCodec

	_ = event.Fold(&lined)

	// Echoed after the answer, because a choice's number does not exist
	// until it is made - the prompt cannot carry it. The transcript's own
	// line names the point, the decider and the alternatives, all of which
	// the person who just typed the answer can see above.
	if lined.flat.Kind == kindChoice {
		return fmt.Sprintf("%3d. you chose %s\n", lined.flat.Seq, lined.flat.Chosen)
	}

	var out strings.Builder

	writeEvent(&out, lined.flat)

	return out.String()
}
