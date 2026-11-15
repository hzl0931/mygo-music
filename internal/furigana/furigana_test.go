package furigana

import (
	"testing"
	"unicode"

	"mygo-music/internal/lyrics"
)

// HasKanji takes a line with a kanji, whatever else it has.
func TestHasKanji(t *testing.T) {
	for text, want := range map[string]bool{
		"今日の空":         true,
		"春日影":          true,
		"悴んだ心":         true,
		"きょうのそら":       false,
		"Hello, world": false,
		"カタカナ・だけ":      false,
	} {
		if got := HasKanji(text); got != want {
			t.Errorf("HasKanji(%q) = %v, want %v", text, got, want)
		}
	}
}

// The readings over a line's kanji leave its kana alone, at the runes they
// cover, in hiragana.
func TestAnnotate(t *testing.T) {
	a := New()
	for _, c := range []struct {
		text string
		want map[string]string // the runes covered, and their reading
	}{
		{"今日の空はとても青いですね", map[string]string{"今日": "きょう", "空": "そら", "青": "あお"}},
		{"食べたこともないような夢を見た", map[string]string{"食": "た", "夢": "ゆめ", "見": "み"}},
		{"君の名前を呼んでいる", map[string]string{"君": "きみ", "名前": "なまえ", "呼": "よ"}},
		{"明日はきっと晴れるから", map[string]string{"明日": "あした", "晴": "は"}},
		{"心の中に咲いた花", map[string]string{"心": "こころ", "中": "なか", "咲": "さ", "花": "はな"}},
		{"失くしたものを探している", map[string]string{"失": "しつ", "探": "さが"}},
		{"この声が届くまで", map[string]string{"声": "こえ", "届": "とど"}},
		{"走り出した想いを止められない", map[string]string{"走": "はし", "出": "だ", "想": "おも", "止": "と"}},
		{"ずっと一緒にいたい", map[string]string{"一緒": "いっしょ"}},
		{"お茶を飲む", map[string]string{"茶": "ちゃ", "飲": "の"}},
		{"珈琲を飲む", map[string]string{"珈琲": "こーひー", "飲": "の"}},
		{"時々泣きたくなる", map[string]string{"時々": "ときどき", "泣": "な"}},
		{"大きい声で歌う", map[string]string{"大": "おお", "声": "こえ", "歌": "うた"}},
	} {
		runes := []rune(c.text)
		got := map[string]string{}
		var last int
		for _, s := range a.Annotate(c.text) {
			if s.Start < last || s.End > len(runes) || s.Start >= s.End {
				t.Errorf("%q: span %+v out of bounds or overlapping", c.text, s)
				continue
			}
			for _, r := range s.Reading {
				if !unicode.In(r, unicode.Hiragana) && r != 'ー' {
					t.Errorf("%q: reading %q isn't hiragana", c.text, s.Reading)
				}
			}
			got[string(runes[s.Start:s.End])] = s.Reading
			last = s.End
		}
		for surface, reading := range c.want {
			if got[surface] != reading {
				t.Errorf("%q: %s read %q, want %q (all %v)", c.text, surface, got[surface], reading, got)
			}
		}
	}
}

// Lines without kanji have nothing read over them.
func TestAnnotateSkips(t *testing.T) {
	a := New()
	for _, text := range []string{"きょうのそら", "", "Hello"} {
		if spans := a.Annotate(text); len(spans) != 0 {
			t.Errorf("%q: spans %+v", text, spans)
		}
	}
}

// A line of kanji alone has its readings too: the song's language is judged
// by its lyrics whole, not by each line.
func TestAnnotateKanjiOnly(t *testing.T) {
	a := New()
	for _, text := range []string{"嗚呼", "存在証明"} {
		if spans := a.Annotate(text); len(spans) == 0 {
			t.Errorf("%q: no readings", text)
		}
	}
}

// Reading a line again answers what it did before.
func TestAnnotateCache(t *testing.T) {
	a := New()
	first := a.Annotate("今日の空")
	second := a.Annotate("今日の空")
	if len(first) == 0 || len(second) != len(first) {
		t.Fatalf("first %+v, second %+v", first, second)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("span %d: %+v, then %+v", i, first[i], second[i])
		}
	}
}

// A reading over a line's words lands on the characters it names.
func TestApplyRubyFromAnnotator(t *testing.T) {
	a := New()
	line := lyrics.Line{Text: "今日の空", Words: []lyrics.Word{
		{Text: "今", Start: 0, End: 1},
		{Text: "日", Start: 0, End: 1},
		{Text: "の", Start: 0, End: 1},
		{Text: "空", Start: 0, End: 1},
	}}
	got := lyrics.ApplyRuby([]lyrics.Line{line}, [][]lyrics.RubySpan{a.Annotate(line.Text)})[0]
	if !got.HasRuby() || got.Words[0].Text != "今日" || got.Words[0].Ruby != "きょう" {
		t.Errorf("words %+v", got.Words)
	}
}

// A line's readings reach the words they were read for, their times in order.
func TestApplyRubyRealLine(t *testing.T) {
	a := New()
	line := lyrics.Build("[00:00.00]走り出した想いを止められない", "")[0]
	got := lyrics.ApplyRuby([]lyrics.Line{line}, [][]lyrics.RubySpan{a.Annotate(line.Text)})[0]
	want := map[string]string{"走": "はし", "出": "だ", "想": "おも", "止": "と"}
	seen := map[string]string{}
	for _, w := range got.Words {
		if w.Ruby != "" {
			seen[w.Text] = w.Ruby
		}
		if w.End < w.Start {
			t.Errorf("word %+v ends before it starts", w)
		}
	}
	for text, ruby := range want {
		if seen[text] != ruby {
			t.Errorf("%s read %q, want %q (all %v)", text, seen[text], ruby, got.Words)
		}
	}
}
