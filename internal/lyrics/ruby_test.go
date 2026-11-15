package lyrics

import (
	"testing"
	"time"
)

// Readings laid over a line make one word of the runes they cover, timed from
// its first word to its last, and leave the rest as they were.
func TestApplyRuby(t *testing.T) {
	s := time.Second
	line := Line{Text: "今日の空", Words: []Word{
		{Text: "今", Start: 0, End: s},
		{Text: "日", Start: s, End: 2 * s},
		{Text: "の", Start: 2 * s, End: 3 * s},
		{Text: "空", Start: 3 * s, End: 4 * s},
	}}
	got := ApplyRuby([]Line{line}, [][]RubySpan{{
		{Start: 0, End: 2, Reading: "きょう"},
		{Start: 3, End: 4, Reading: "そら"},
	}})[0]
	want := []Word{
		{Text: "今日", Ruby: "きょう", Start: 0, End: 2 * s},
		{Text: "の", Start: 2 * s, End: 3 * s},
		{Text: "空", Ruby: "そら", Start: 3 * s, End: 4 * s},
	}
	if len(got.Words) != len(want) {
		t.Fatalf("words %+v", got.Words)
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Errorf("word %d = %+v, want %+v", i, got.Words[i], want[i])
		}
	}
	if !got.HasRuby() || line.Words[0].Ruby != "" {
		t.Errorf("HasRuby %v, original changed to %+v", got.HasRuby(), line.Words[0])
	}
}

// A reading over part of a word splits it, sharing the word's time by runes.
func TestApplyRubyPartialWord(t *testing.T) {
	s := time.Second
	line := Line{Text: "食べる", Words: []Word{{Text: "食べる", Start: 0, End: 3 * s}}}
	got := ApplyRuby([]Line{line}, [][]RubySpan{{{Start: 0, End: 1, Reading: "た"}}})[0]
	if len(got.Words) != 2 {
		t.Fatalf("words %+v", got.Words)
	}
	if got.Words[0] != (Word{Text: "食", Ruby: "た", Start: 0, End: s}) {
		t.Errorf("first %+v", got.Words[0])
	}
	if got.Words[1] != (Word{Text: "べる", Start: s, End: 3 * s}) {
		t.Errorf("rest %+v", got.Words[1])
	}
}

// Spans that don't fit, or run into one another, are left out.
func TestApplyRubyKeepsValidSpans(t *testing.T) {
	line := Line{Text: "春日影", Words: []Word{
		{Text: "春", Start: 0, End: time.Second},
		{Text: "日", Start: time.Second, End: 2 * time.Second},
		{Text: "影", Start: 2 * time.Second, End: 3 * time.Second},
	}}
	got := ApplyRuby([]Line{line}, [][]RubySpan{{
		{Start: -1, End: 1, Reading: "はる"},
		{Start: 0, End: 2, Reading: "はる"},
		{Start: 1, End: 3, Reading: "ひ"},
		{Start: 2, End: 5, Reading: "かげ"},
	}})[0]
	if len(got.Words) != 2 || got.Words[0].Text != "春日" || got.Words[0].Ruby != "はる" || got.Words[1].Text != "影" || got.Words[1].Ruby != "" {
		t.Errorf("words %+v", got.Words)
	}
}

// A line whose words don't spell its text is laid out from the text, rather
// than readings landing on the wrong characters.
func TestApplyRubyMismatchedWords(t *testing.T) {
	line := Line{Text: "今日", At: 0, End: time.Second, Words: []Word{{Text: "今", Start: 0, End: time.Second}}}
	got := ApplyRuby([]Line{line}, [][]RubySpan{{{Start: 0, End: 1, Reading: "きょう"}}})[0]
	if len(got.Words) != 2 || got.Words[0].Text != "今" || got.Words[0].Ruby != "きょう" || got.Words[1].Text != "日" || got.Words[1].Ruby != "" {
		t.Errorf("words %+v", got.Words)
	}
}

// With no readings, a line's words come back as they were.
func TestApplyRubyNone(t *testing.T) {
	line := Line{Text: "abc", Words: []Word{{Text: "abc", Start: 0, End: time.Second}}}
	got := ApplyRuby([]Line{line}, nil)[0]
	if len(got.Words) != 1 || got.Words[0] != line.Words[0] {
		t.Errorf("words %+v", got.Words)
	}
}
