package httpserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"weatherservice/internal/planner"
)

// stopTimes are the default local start times for a day's stops, one per slot, matching the
// one-pager's v2 spec. Each stop gets a stopDuration-long block.
var stopTimes = []struct{ hour, minute int }{
	{10, 0}, {12, 0}, {15, 0}, {17, 0},
}

const stopDuration = 90 * time.Minute

const icsDateTimeLayout = "20060102T150405Z"

// exportICS answers GET /trip/:slug.ics with the trip as an iCalendar (RFC 5545) file: one
// VEVENT per stop, at fixed local times converted to UTC, so no VTIMEZONE block is needed.
func (s *Server) exportICS(ctx *fiber.Ctx) error {
	plan, ok := s.resolveTripPlan(ctx)
	if !ok {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	ctx.Set(fiber.HeaderContentType, "text/calendar; charset=utf-8")
	ctx.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s.ics"`, ctx.Params("slug")))

	return ctx.SendString(buildICS(plan, ctx.Params("slug"), s.clock()))
}

// buildICS renders plan as a complete iCalendar document.
func buildICS(plan *planner.Plan, slug string, generatedAt time.Time) string {
	var lines []string
	lines = append(lines,
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//TravelTab//Trip Planner//EN",
		"CALSCALE:GREGORIAN",
	)

	stamp := generatedAt.UTC().Format(icsDateTimeLayout)
	for dayIndex, day := range plan.Days {
		for stopIndex, stop := range day.Stops {
			lines = append(lines, icsEvent(slug, dayIndex, stopIndex, stop, day.Date, stamp)...)
		}
	}

	lines = append(lines, "END:VCALENDAR")

	return strings.Join(lines, "\r\n") + "\r\n"
}

// icsEvent renders one VEVENT for stop, at its day's fixed slot time (stopTimes[stopIndex],
// or the last slot for any stop beyond that - GroupDays never gives a day more stops than
// stopTimes has entries, but the fallback keeps this correct even if that ever changes).
func icsEvent(slug string, dayIndex, stopIndex int, stop planner.Place, date time.Time, stamp string) []string {
	slot := stopTimes[len(stopTimes)-1]
	if stopIndex < len(stopTimes) {
		slot = stopTimes[stopIndex]
	}

	y, m, d := date.Date()
	start := time.Date(y, m, d, slot.hour, slot.minute, 0, 0, date.Location())
	end := start.Add(stopDuration)

	uid := fmt.Sprintf("%s-%d-%d-%s@traveltab", slug, dayIndex, stopIndex, stop.ID)

	return foldAll([]string{
		"BEGIN:VEVENT",
		"UID:" + icsEscape(uid),
		"DTSTAMP:" + stamp,
		"DTSTART:" + start.UTC().Format(icsDateTimeLayout),
		"DTEND:" + end.UTC().Format(icsDateTimeLayout),
		"SUMMARY:" + icsEscape(stop.Name),
		"LOCATION:" + icsEscape(stop.Name),
		"DESCRIPTION:" + icsEscape("Part of your TravelTab trip: /trip/"+slug),
		"END:VEVENT",
	})
}

// icsEscape applies RFC 5545's TEXT escaping: a backslash, comma or semicolon in the value
// must itself be backslash-escaped, and a literal newline becomes the two characters \n.
func icsEscape(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`;`, `\;`,
		`,`, `\,`,
		"\n", `\n`,
	)
	return replacer.Replace(s)
}

// foldAll line-folds every line in lines to RFC 5545's 75-octet limit.
func foldAll(lines []string) []string {
	folded := make([]string, 0, len(lines))
	for _, line := range lines {
		folded = append(folded, fold(line)...)
	}
	return folded
}

// fold splits one logical line into RFC 5545's "line, CRLF, single space" continuations, so no
// physical line exceeds 75 octets. The leading space of a continuation does not itself count
// toward that line's 75, since a reader is required to strip exactly one such space back out.
func fold(line string) []string {
	// The first physical line gets the full 75 octets. Every continuation line carries a
	// mandatory leading space (added below), so its own content must leave room for it: 74
	// octets of content + 1 octet of space = 75.
	const firstLimit = 75
	const continuationLimit = 74

	b := []byte(line)
	if len(b) <= firstLimit {
		return []string{line}
	}

	var chunks []string
	limit := firstLimit
	for len(b) > limit {
		n := limit
		// Never split in the middle of a UTF-8 rune: back off until n lands on a byte that
		// doesn't continue the rune starting before it. b[n] is always in bounds here,
		// because this branch only runs while len(b) > limit, i.e. n < len(b).
		for n > 0 && isUTF8Continuation(b[n]) {
			n--
		}
		chunks = append(chunks, string(b[:n]))
		b = b[n:]
		limit = continuationLimit
	}
	if len(b) > 0 {
		chunks = append(chunks, string(b))
	}

	out := make([]string, len(chunks))
	out[0] = chunks[0]
	for i := 1; i < len(chunks); i++ {
		out[i] = " " + chunks[i]
	}
	return out
}

// isUTF8Continuation reports whether b is a UTF-8 continuation byte (10xxxxxx), which must
// never start a folded line on its own.
func isUTF8Continuation(b byte) bool {
	return b&0xC0 == 0x80
}
