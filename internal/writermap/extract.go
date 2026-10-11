package writermap

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

var dayHeading = regexp.MustCompile(`(?i)^(?:day|dia)\s*([0-9]{1,2})(?:\D|$)`)

// ExtractPost indexes rendered WordPress title and body HTML without returning post text.
// Offsets are end-exclusive Unicode rune indices in a plain-text stream: HTML entities
// are decoded, whitespace is collapsed, block boundaries become one newline, and the
// title and body are separated by one newline. Empty parts add no separator. Inline
// tags do not add spaces. Script, style, template, and noscript content is omitted.
// Candidates are capitalized name sequences, plus lowercase words followed by a
// connector and a capitalized word; arbitrary lowercase prose is not indexed.
func ExtractPost(titleHTML, bodyHTML string) ExtractedPost {
	var stream textStream
	stream.appendHTML(titleHTML, false)
	stream.breakLine()
	stream.appendHTML(bodyHTML, true)
	days := extractDays(stream.text, stream.headings)
	return ExtractedPost{
		Phrases:     extractPhrases(stream.text),
		Days:        days,
		IsItinerary: itineraryDays(days),
	}
}

// ExtractedPost contains only indexable names and offsets, never title or body text.
type ExtractedPost struct {
	Phrases     []NamePhrase
	Days        []DaySection
	IsItinerary bool
}

// NamePhrase is a folded candidate place name at its original plain-text rune span.
type NamePhrase struct {
	Folded       string
	OriginalCase bool
	Start        int
	End          int
}

// DaySection spans a day heading and all content until the next day heading.
type DaySection struct {
	Number int
	Start  int
	End    int
}

func extractPhrases(text []rune) []NamePhrase {
	words := scanWords(text)
	var phrases []NamePhrase
	seen := make(map[NamePhrase]bool)
	for i := 0; i < len(words); {
		if !words[i].capitalized {
			if i+2 < len(words) && !isConnector(words[i].text) &&
				isConnector(words[i+1].text) && words[i+2].capitalized &&
				gapIsSpace(text, words[i].end, words[i+1].start) &&
				gapIsSpace(text, words[i+1].end, words[i+2].start) {
				for last := i + 2; last < len(words) && last-i < 5; last++ {
					if last > i+2 && (!gapIsSpace(text, words[last-1].end, words[last].start) ||
						(!words[last].capitalized && !isConnector(words[last].text))) {
						break
					}
					if words[last].capitalized {
						addPhrase(&phrases, seen, text, words[i].start, words[last].end, false)
					}
				}
			}
			i++
			continue
		}
		end := i
		for end+1 < len(words) && gapIsSpace(text, words[end].end, words[end+1].start) &&
			(words[end+1].capitalized || isConnector(words[end+1].text)) {
			end++
		}
		for end > i && !words[end].capitalized {
			end--
		}
		for start := i; start <= end; start++ {
			if !words[start].capitalized {
				continue
			}
			for last := start; last <= end && last-start < 5; last++ {
				if words[last].capitalized {
					addPhrase(&phrases, seen, text, words[start].start, words[last].end, true)
				}
			}
		}
		if end-i >= 5 {
			addPhrase(&phrases, seen, text, words[i].start, words[end].end, true)
		}
		i = end + 1
	}
	return phrases
}

func extractDays(text []rune, headings []headingSpan) []DaySection {
	var days []DaySection
	for _, heading := range headings {
		match := dayHeading.FindStringSubmatch(strings.TrimSpace(string(text[heading.start:heading.end])))
		if match == nil {
			continue
		}
		number, _ := strconv.Atoi(match[1])
		if len(days) > 0 {
			days[len(days)-1].End = heading.start
		}
		days = append(days, DaySection{Number: number, Start: heading.start, End: len(text)})
	}
	return days
}

func itineraryDays(days []DaySection) bool {
	if len(days) < 2 || days[0].Number != 1 {
		return false
	}
	for i := 1; i < len(days); i++ {
		if days[i].Number <= days[i-1].Number {
			return false
		}
	}
	return true
}

type textStream struct {
	text     []rune
	pending  rune
	headings []headingSpan
}

type headingSpan struct{ start, end int }

func (s *textStream) appendHTML(markup string, headings bool) {
	context := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, _ := html.ParseFragment(strings.NewReader(markup), context)
	for _, node := range nodes {
		s.walk(node, headings)
	}
}

func (s *textStream) walk(node *html.Node, headings bool) {
	if node.Type == html.TextNode {
		s.appendText(node.Data)
		return
	}
	if node.Type != html.ElementNode && node.Type != html.DocumentNode {
		return
	}
	if node.DataAtom == atom.Script || node.DataAtom == atom.Style ||
		node.DataAtom == atom.Template || node.DataAtom == atom.Noscript {
		return
	}
	if node.DataAtom == atom.Br {
		s.breakLine()
		return
	}
	heading := headings && isHeading(node.DataAtom)
	block := heading || isBlock(node.DataAtom)
	if block {
		s.breakLine()
	}
	start := s.nextIndex()
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		s.walk(child, headings)
	}
	if heading && len(s.text) > start {
		s.headings = append(s.headings, headingSpan{start, len(s.text)})
	}
	if block {
		s.breakLine()
	}
}

func (s *textStream) appendText(value string) {
	for _, r := range value {
		if unicode.IsSpace(r) {
			if s.pending == 0 {
				s.pending = ' '
			}
			continue
		}
		if s.pending != 0 && len(s.text) > 0 {
			s.text = append(s.text, s.pending)
		}
		s.pending = 0
		s.text = append(s.text, r)
	}
}

func (s *textStream) breakLine() {
	s.pending = '\n'
}

func (s *textStream) nextIndex() int {
	if s.pending != 0 && len(s.text) > 0 {
		return len(s.text) + 1
	}
	return len(s.text)
}

func isBlock(tag atom.Atom) bool {
	switch tag {
	case atom.P, atom.Div, atom.Article, atom.Section, atom.Blockquote,
		atom.Ul, atom.Ol, atom.Li, atom.Table, atom.Tr, atom.Td:
		return true
	}
	return false
}

func isHeading(tag atom.Atom) bool {
	switch tag {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		return true
	}
	return false
}

type phraseWord struct {
	text        string
	start, end  int
	capitalized bool
}

func scanWords(text []rune) []phraseWord {
	var words []phraseWord
	for i := 0; i < len(text); {
		if !isWordRune(text[i]) {
			i++
			continue
		}
		start := i
		for i < len(text) && isWordRune(text[i]) {
			i++
		}
		words = append(words, phraseWord{string(text[start:i]), start, i, unicode.IsUpper(text[start])})
	}
	return words
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r)
}

func gapIsSpace(text []rune, start, end int) bool {
	if end == start {
		return false
	}
	for _, r := range text[start:end] {
		if r != ' ' {
			return false
		}
	}
	return true
}

func isConnector(word string) bool {
	switch strings.ToLower(word) {
	case "de", "do", "da", "dos", "das", "e", "of", "the", "and":
		return true
	}
	return false
}

func addPhrase(phrases *[]NamePhrase, seen map[NamePhrase]bool, text []rune, start, end int, originalCase bool) {
	phrase := NamePhrase{Folded: foldName(string(text[start:end])), OriginalCase: originalCase, Start: start, End: end}
	if !seen[phrase] {
		*phrases = append(*phrases, phrase)
		seen[phrase] = true
	}
}

func foldName(value string) string {
	var folded strings.Builder
	for _, r := range norm.NFKD.String(value) {
		if !unicode.Is(unicode.Mn, r) {
			folded.WriteRune(r)
		}
	}
	return cases.Fold().String(folded.String())
}
