// Package furigana reads the kanji of Japanese lyrics: their words, and the
// kana over them.
package furigana

import (
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"

	"mygo-music/internal/lyrics"
)

// Annotator reads Japanese lines, keeping what it has read: the same line
// comes up again and again as a song plays, and across songs.
type Annotator struct {
	once sync.Once
	tok  *tokenizer.Tokenizer

	tokMu sync.Mutex // guards the tokenizer, which isn't for two callers

	mu    sync.Mutex
	cache map[string][]lyrics.RubySpan
	order []string
	max   int
}

// New returns an annotator. Its tokenizer is read on the first line it reads.
func New() *Annotator { return &Annotator{max: 4096} }

// HasKanji reports whether text has a kanji, a line with something to read
// over it. Whether a song is Japanese is the caller's to say from its lyrics
// whole: a line of kanji alone, as 嗚呼, may still be Japanese.
func HasKanji(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// Annotate returns readings over text's kanji, at offsets of runes. It
// answers nil for text without kanji. What it answers must not be changed:
// another line may have the same.
func (a *Annotator) Annotate(text string) []lyrics.RubySpan {
	if !HasKanji(text) {
		return nil
	}
	a.mu.Lock()
	if spans, ok := a.cache[text]; ok {
		a.mu.Unlock()
		return spans
	}
	a.mu.Unlock()

	spans := a.read(text)

	a.mu.Lock()
	if a.cache == nil {
		a.cache = make(map[string][]lyrics.RubySpan)
	}
	if _, ok := a.cache[text]; !ok {
		max := a.max
		if max <= 0 {
			max = 4096
		}
		if len(a.order) >= max && len(a.order) > 0 {
			delete(a.cache, a.order[0])
			a.order = a.order[1:]
		}
		a.cache[text] = spans
		a.order = append(a.order, text)
	}
	a.mu.Unlock()
	return spans
}

// read tokenizes text and takes the readings of its kanji.
func (a *Annotator) read(text string) []lyrics.RubySpan {
	tok := a.tokenizer()
	if tok == nil {
		return nil
	}
	a.tokMu.Lock()
	tokens := tok.Analyze(text, tokenizer.Normal)
	a.tokMu.Unlock()

	// Tokens must spell the line as it is, or the offsets don't mean it.
	var spelled strings.Builder
	for _, t := range tokens {
		spelled.WriteString(t.Surface)
	}
	if spelled.String() != text {
		return nil
	}

	var spans []lyrics.RubySpan
	at := 0
	for _, t := range tokens {
		n := utf8.RuneCountInString(t.Surface)
		reading, ok := t.Reading()
		if ok && reading != "" && reading != "*" {
			for _, p := range readingParts(t.Surface, reading) {
				spans = append(spans, lyrics.RubySpan{Start: at + p.start, End: at + p.end, Reading: p.reading})
			}
		}
		at += n
	}
	return spans
}

func (a *Annotator) tokenizer() *tokenizer.Tokenizer {
	a.once.Do(func() {
		a.tok, _ = tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	})
	return a.tok
}

// A part is a run of a word's kanji and the kana over it, at offsets of the
// word's runes.
type part struct {
	start, end int
	reading    string
}

// readingParts splits a word's reading among the runs of its kanji, as 走り出
// / はしりだ reads 走 はし and 出 だ, its kana in place. Where the kana don't
// line up, the whole kanji of the word takes its reading.
func readingParts(surface, reading string) []part {
	s, r := []rune(surface), []rune(toHiragana(reading))
	var pattern strings.Builder
	pattern.WriteString("^")
	type run struct{ start, end int }
	var runs []run
	for i := 0; i < len(s); {
		if unicode.Is(unicode.Han, s[i]) {
			j := i
			for j < len(s) && unicode.Is(unicode.Han, s[j]) {
				j++
			}
			runs = append(runs, run{i, j})
			pattern.WriteString("(.+)")
			i = j
			continue
		}
		pattern.WriteString(regexp.QuoteMeta(string(s[i])))
		i++
	}
	pattern.WriteString("$")
	if re, err := regexp.Compile(pattern.String()); err == nil {
		if m := re.FindStringSubmatchIndex(string(r)); m != nil {
			parts := make([]part, 0, len(runs))
			for k, run := range runs {
				a, b := m[2+2*k], m[3+2*k]
				if a < 0 || b <= a {
					continue
				}
				p := part{run.start, run.end, string(r)[a:b]}
				if allKana(p.reading) {
					parts = append(parts, p)
				}
			}
			if len(parts) > 0 {
				return parts
			}
		}
	}
	if start, end, ruby, ok := trimReading(surface, reading); ok {
		return []part{{start, end, ruby}}
	}
	return nil
}

// trimReading takes the kana that a word and its reading share at their ends
// off, as the okurigana of 食べる / たべる, and answers where the rest of the
// word sits in it and the kana over it.
func trimReading(surface, reading string) (start, end int, ruby string, ok bool) {
	s, r := []rune(surface), []rune(toHiragana(reading))
	suffix := 0
	for suffix < len(s) && suffix < len(r) {
		a, b := s[len(s)-1-suffix], r[len(r)-1-suffix]
		if a != b || !isKana(a) {
			break
		}
		suffix++
	}
	prefix := 0
	for prefix < len(s)-suffix && prefix < len(r)-suffix {
		a, b := s[prefix], r[prefix]
		if a != b || !isKana(a) {
			break
		}
		prefix++
	}
	start, end = prefix, len(s)-suffix
	if start >= end || !hasKanji(s[start:end]) {
		return 0, 0, "", false
	}
	ruby = string(r[prefix : len(r)-suffix])
	if !allKana(ruby) {
		return 0, 0, "", false
	}
	return start, end, ruby, true
}

// toHiragana writes a reading in hiragana, as the words of a line are.
func toHiragana(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'ァ' && r <= 'ヶ' {
			return r - 0x60
		}
		return r
	}, s)
}

func isKana(r rune) bool {
	return unicode.In(r, unicode.Hiragana, unicode.Katakana) || r == 'ー' || r == '・'
}

func allKana(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isKana(r) {
			return false
		}
	}
	return true
}

func hasKanji(rs []rune) bool {
	for _, r := range rs {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
