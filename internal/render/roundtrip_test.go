package render_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/render"
)

// A record written earlier renders to the same sheet and the same transcript
// the generating run produced.
//
// This is what the render subcommand promises, and it is worth asserting
// rather than assuming: the two go through one renderer precisely so they
// cannot disagree, and this is the test that would notice if a second one
// appeared.
//
// Holds: A record renders to what generating it rendered.
func TestARecordRendersToWhatGeneratedItRendered(t *testing.T) {
	t.Parallel()

	records, err := filepath.Glob(filepath.Join(goldenDir, "*.json"))
	if err != nil {
		t.Fatalf("looking for goldens: %v", err)
	}

	if len(records) == 0 {
		t.Fatal("no goldens found")
	}

	for _, path := range records {
		name := strings.TrimSuffix(filepath.Base(path), ".json")

		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		sheet, err := render.SheetFrom(text)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		compare(t, name+".sheet.md", sheet)

		transcript, err := render.TranscriptFrom(text, false)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		compare(t, name+".transcript.md", transcript)
	}
}

func compare(t *testing.T, name, got string) {
	t.Helper()

	want, err := os.ReadFile(filepath.Join(goldenDir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	if got != string(want) {
		t.Errorf("%s rendered from the record differs from what generating it rendered", name)
	}
}

// A file that is not a record is refused, rather than rendered as a blank
// character.
func TestSomethingElseIsNotARecord(t *testing.T) {
	t.Parallel()

	for name, text := range map[string]string{
		"not JSON":      "# a character sheet\n",
		"another shape": `{"hello":"world"}`,
		"empty object":  `{}`,
	} {
		_, err := render.SheetFrom([]byte(text))
		if err == nil {
			t.Errorf("%s was accepted as a record", name)
		}
	}
}

// minimalRecord is the least decode accepts, so a test can pin one detail
// without carrying a whole character.
func minimalRecord(body string) []byte {
	return []byte(`{"upp":"777777","ruleset":"test",` + body + `}`)
}

// A ship's terms are read off its kind, not inferred from its numbers. A
// Free Trader received and never paid down carries years 0 and paymentYears
// 0, and is owned - not held in constructive possession, which is the scout
// ship's arrangement (p. 23).
//
// Holds: A ship's terms follow its kind, not its numbers.
func TestAShipsTermsFollowItsKind(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct{ kind, want, avoid string }{
		"a new Free Trader": {
			kind:  "Free Trader, Type A",
			want:  "owned free and clear",
			avoid: "constructive possession",
		},
		"a scout ship": {
			kind:  "Scout ship, Type S",
			want:  "held in constructive possession",
			avoid: "owned free and clear",
		},
	} {
		sheet, err := render.SheetFrom(minimalRecord(
			`"benefits":{"cash":0,"ships":[{"kind":"` + tc.kind +
				`","tons":200,"years":0,"paymentYears":0}]}`,
		))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if !strings.Contains(sheet, tc.want) {
			t.Errorf("%s does not read %q:\n%s", name, tc.want, sheet)
		}

		if strings.Contains(sheet, tc.avoid) {
			t.Errorf("%s reads %q, which is the other ship's terms:\n%s", name, tc.avoid, sheet)
		}
	}
}

// An event kind this build does not know is said to be unknown, rather than
// rendered as an outcome - which would print a blank numbered line and claim
// the transcript was complete. Reading another build's record is the whole
// reason the render subcommand exists.
//
// Holds: An event kind this build does not know is said to be unknown.
func TestAnUnknownEventKindIsSaidToBeUnknown(t *testing.T) {
	t.Parallel()

	transcript, err := render.TranscriptFrom(minimalRecord(
		`"events":[{"seq":9,"kind":"portent"}]`,
	), false)
	if err != nil {
		t.Fatalf("reading a record with an unknown event kind: %v", err)
	}

	if !strings.Contains(transcript, `unknown event kind "portent"`) {
		t.Errorf("an unknown event kind rendered as %q", transcript)
	}
}

// Each of decode's two refusals names what is missing.
func TestARecordMustCarryAUPPAndARuleset(t *testing.T) {
	t.Parallel()

	_, err := render.SheetFrom([]byte(`{"ruleset":"test"}`))
	if err == nil || !strings.Contains(err.Error(), "no UPP") {
		t.Errorf("a record with no UPP was refused as %v", err)
	}

	_, err = render.SheetFrom([]byte(`{"upp":"777777"}`))
	if err == nil || !strings.Contains(err.Error(), "no ruleset") {
		t.Errorf("a record with no ruleset was refused as %v", err)
	}
}

// A record that names its policy as a bare version, as builds before #276
// wrote it, is refused rather than read. Read, it would carry no policy, and
// a record with none says a person answered it: an auto character read back
// as one the player made is the misattribution #276 removed.
func TestAPolicyNamedByVersionAloneIsRefused(t *testing.T) {
	t.Parallel()

	old := minimalRecord(`"policy":3`)

	_, err := render.SheetFrom(old)
	if err == nil || !strings.Contains(err.Error(), "not a character record") {
		t.Errorf("the sheet read a policy named by its version alone: %v", err)
	}

	_, err = render.TranscriptFrom(old, false)
	if err == nil || !strings.Contains(err.Error(), "not a character record") {
		t.Errorf("the transcript read a policy named by its version alone: %v", err)
	}
}

// A record carrying an event that will not read is refused by both
// renderings, rather than rendered without it.
//
// The sheet shows one choice event at most, so it has to read the events on
// purpose. A sheet that swallowed the failure would end by telling the referee
// that the record is the only way back to this character - about a record it
// could not finish reading.
func TestAnEventThatWillNotReadIsRefused(t *testing.T) {
	t.Parallel()

	// Both, because the sheet reaches the events by two routes: once up front
	// for every record, and again for the headline of a serving character,
	// which asks whose service this is. A civilian returns from the headline
	// before that question, so only a record carrying a service exercises
	// the second route.
	for name, unreadable := range map[string][]byte{
		"a civilian": minimalRecord(`"events":[3]`),
		"a serving character": minimalRecord(
			`"service":"Navy","terms":2,"enlistment":{"how":"enlisted"},"events":[3]`,
		),
	} {
		_, err := render.SheetFrom(unreadable)
		if err == nil || !strings.Contains(err.Error(), "an event will not read") {
			t.Errorf("%s: the sheet rendered a record it could not read: %v", name, err)
		}

		_, err = render.TranscriptFrom(unreadable, false)
		if err == nil || !strings.Contains(err.Error(), "an event will not read") {
			t.Errorf("%s: the transcript rendered a record it could not read: %v", name, err)
		}
	}
}

// headlineCase is one record and what its headline must and must not read.
type headlineCase struct{ body, wants, does string }

// served is a record of a character in a service, with whatever further
// JSON fields the case under test wants to say about how he got there.
func served(service, how, fields string) string {
	return fmt.Sprintf(`"service":%q,"terms":2,"enlistment":{"how":%q},%s`,
		service, how, fields)
}

// chose is a Service choice event answered by whoever the case names: the
// policy on an auto record, which names it, and the player on a guided one,
// which names none.
func chose(by string) string {
	policy := ""
	if by == "policy" {
		policy = `"policy":{"version":1,"career":"serve","skills":"advanced","muster":"cash"},`
	}

	return policy + `"inputs":{},"events":[{"seq":1,"kind":"choice",` +
		`"point":"Service","alternatives":["Navy","Other"],"chosen":"Navy"}]`
}

// unchanged is the headline of a character with nothing to explain.
const unchanged = "Navy, 2 terms"

// The headline says whose service it is, where that is not obvious.
//
// The service record four sections down has always been honest. The headline
// is the line a referee reads, and a character who was typed as a Marine and
// came back a Navy man had nothing there to explain it.
//
// Everything turns on the service ATTEMPTED, not on who named it. The draft
// can override either decider, and crediting one with an outcome it did not
// pick is worse than saying nothing.
//
// Holds: The headline names the service attempted, not the one the draft gave.
func TestTheHeadlineSaysWhoseServiceItIs(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]headlineCase{
		"asked for one service and got another": {
			body:  served("Navy", "drafted", `"inputs":{"service":"Marines"}`),
			wants: "Navy (drafted after the Marines refused him)",
		},
		"asked for a service and got it": {
			body:  served("Navy", "enlisted", `"inputs":{"service":"Navy"}`),
			wants: unchanged, does: "refused",
		},
		"asked for nothing and the policy picked": {
			body:  served("Navy", "enlisted", chose("policy")),
			wants: "Navy (service chosen by the policy)",
		},
		// A player who named it himself is told nothing he does not know.
		"asked for nothing and the player picked": {
			body:  served("Navy", "enlisted", chose("player")),
			wants: unchanged, does: "chosen by",
		},
		// The draft overrides the attempt. Saying "chosen by the policy"
		// over a draft's outcome credits the policy with a choice it never
		// made, in the case this exists to expose.
		"the policy picked one and the draft gave another": {
			body:  served("Marines", "drafted", chose("policy")),
			wants: "Marines (drafted after the Navy refused him)", does: "chosen by the policy",
		},
		// And after a draft the player does not know either: he chose Navy.
		"the player picked one and the draft gave another": {
			body:  served("Marines", "drafted", chose("player")),
			wants: "Marines (drafted after the Navy refused him)",
		},
		// A record saying nothing about how he got there. The tool writes
		// none, but the renderer is handed records rather than generating
		// them, and inventing a decider is worse than saying nothing.
		"nothing recorded about the service at all": {
			body:  served("Navy", "enlisted", `"inputs":{}`),
			wants: unchanged, does: "(",
		},
	} {
		checkHeadline(t, name, tc)
	}
}

// checkHeadline reads the headline out of the sheet and asserts on that
// alone. Searching the whole sheet is too loose - "(" would find the
// "(unnamed)" title - and the headline is what every case here is about.
func checkHeadline(t *testing.T, name string, tc headlineCase) {
	t.Helper()

	sheet, err := render.SheetFrom(minimalRecord(tc.body))
	if err != nil {
		t.Errorf("%s: %v", name, err)

		return
	}

	lead := ""

	for line := range strings.Lines(sheet) {
		if strings.HasPrefix(line, "UPP ") {
			lead = line

			break
		}
	}

	if lead == "" {
		t.Errorf("%s: no headline in\n%s", name, sheet)

		return
	}

	if !strings.Contains(lead, tc.wants) {
		t.Errorf("%s: the headline does not read %q: %q", name, tc.wants, lead)
	}

	if tc.does != "" && strings.Contains(lead, tc.does) {
		t.Errorf("%s: the headline reads %q and should not: %q", name, tc.does, lead)
	}
}

// One term is one term.
//
// "1 term" is a substring of "1 terms", so the singular case is pinned by what
// must NOT appear. Asserting only the positive would pass on the bug.
//
// Holds: A one-term character reads as one term.
func TestOneTermIsSingular(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		terms       int
		wants, does string
	}{
		"one":  {1, "1 term", "1 terms"},
		"two":  {2, "2 terms", ""},
		"none": {0, "0 terms", ""},
	} {
		sheet, err := render.SheetFrom(minimalRecord(fmt.Sprintf(
			`"service":"Navy","terms":%d,"enlistment":{"how":"enlisted"},"inputs":{"service":"Navy"}`,
			tc.terms,
		)))
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if !strings.Contains(sheet, tc.wants) {
			t.Errorf("%s: the headline does not read %q:\n%s", name, tc.wants, sheet)
		}

		if tc.does != "" && strings.Contains(sheet, tc.does) {
			t.Errorf("%s: the headline reads %q:\n%s", name, tc.does, sheet)
		}
	}
}

// The errata are on the transcript and not on the sheet.
//
// On a sheet handed to a player they are codes he cannot expand; beside the
// outcome they governed they are how a reading is checked against the page,
// which is what the transcript is for.
//
// Holds: The errata are on the transcript and not the sheet.
func TestTheErrataAreOnTheTranscriptAndNotTheSheet(t *testing.T) {
	t.Parallel()

	record := minimalRecord(`"errata":["E001"],` +
		`"events":[{"seq":1,"kind":"outcome","description":"drafted","errata":["E001"]}]`)

	sheet, err := render.SheetFrom(record)
	if err != nil {
		t.Fatalf("the sheet: %v", err)
	}

	if strings.Contains(sheet, "E001") || strings.Contains(sheet, "Readings applied") {
		t.Errorf("the sheet carries an erratum id:\n%s", sheet)
	}

	transcript, err := render.TranscriptFrom(record, false)
	if err != nil {
		t.Fatalf("the transcript: %v", err)
	}

	if !strings.Contains(transcript, "E001") {
		t.Errorf("the transcript lost the erratum that governed an outcome:\n%s", transcript)
	}
}

// Both renderings end with the one line that tells a referee what to keep.
//
// The record is the only way back to a character (ADR 0001), and a sheet is
// what a referee reads and keeps, so the sheet says so once, last, and the
// transcript says the same. Neither offers a command, a seed or a build: there
// is nothing to re-run.
func TestBothRenderingsEndBySayingToKeepTheRecord(t *testing.T) {
	t.Parallel()

	const keep = "---\n\nKeep the JSON record: it is the only way back to this character.\n"

	body := minimalRecord(`"inputs":{"career":"serve","skills":"advanced","muster":"cash"}`)

	sheet, err := render.SheetFrom(body)
	if err != nil {
		t.Fatalf("sheet: %v", err)
	}

	transcript, err := render.TranscriptFrom(body, false)
	if err != nil {
		t.Fatalf("transcript: %v", err)
	}

	for name, rendering := range map[string]string{"sheet": sheet, "transcript": transcript} {
		if !strings.HasSuffix(rendering, keep) {
			t.Errorf("the %s does not end by saying to keep the record:\n%s", name, rendering)
		}

		for _, gone := range []string{"Regenerate with", "Seed ", "--seed", "different build"} {
			if strings.Contains(rendering, gone) {
				t.Errorf("the %s still carries %q:\n%s", name, gone, rendering)
			}
		}
	}
}
