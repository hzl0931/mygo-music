package app

import (
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/lyrics"
	"mygo-music/internal/music"
)

type fakeAnnotator struct {
	mu    sync.Mutex
	calls int
	spans map[string][]lyrics.RubySpan
}

func (f *fakeAnnotator) Annotate(text string) []lyrics.RubySpan {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	return f.spans[text]
}

func (f *fakeAnnotator) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// Japanese lyrics are read in the background, once, and the readings are
// kept for the song until its lyrics change.
func TestFuriganaLines(t *testing.T) {
	a := newTestApp()
	ch := posts(a)
	ann := &fakeAnnotator{spans: map[string][]lyrics.RubySpan{
		"今日の空": {{Start: 0, End: 2, Reading: "きょう"}, {Start: 3, End: 4, Reading: "そら"}},
	}}
	a.ann = ann
	tr := song("7", "宝船")
	lines := lyrics.Build("[00:01.00]今日の空", "")

	if got := a.furiganaLines(&tr, lines); got[0].HasRuby() {
		t.Fatal("read before it was done")
	}
	runPost(t, ch)
	got := a.furiganaLines(&tr, lines)
	if !got[0].HasRuby() || got[0].Words[0].Ruby != "きょう" || got[0].Words[2].Ruby != "そら" {
		t.Fatalf("words %+v", got[0].Words)
	}
	if n := ann.count(); n != 1 {
		t.Errorf("read %d times, want once", n)
	}
	// Asking again answers from what was read, not the annotator again.
	a.furiganaLines(&tr, lines)
	if n := ann.count(); n != 1 {
		t.Errorf("read %d times after asking again, want once", n)
	}
}

// A song without a Japanese line to read, or with its readings hidden, is
// not read at all.
func TestFuriganaLinesSkips(t *testing.T) {
	for _, c := range []struct {
		name  string
		lines []lyrics.Line
		off   bool
	}{
		{"Chinese", lyrics.Build("[00:01.00]春日影", ""), false},
		{"kana only", lyrics.Build("[00:01.00]きょうのそら", ""), false},
		{"hidden", lyrics.Build("[00:01.00]今日の空", ""), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := newTestApp()
			ann := &fakeAnnotator{}
			a.ann = ann
			a.settings.FuriganaOff = c.off
			tr := song("7", "宝船")
			if got := a.furiganaLines(&tr, c.lines); got[0].HasRuby() {
				t.Errorf("words %+v", got[0].Words)
			}
			if n := ann.count(); n != 0 {
				t.Errorf("read %d times, want none", n)
			}
		})
	}
}

// The reading over a line leaves its translation alone.
func TestFuriganaLeavesTranslation(t *testing.T) {
	a := newTestApp()
	ch := posts(a)
	ann := &fakeAnnotator{spans: map[string][]lyrics.RubySpan{
		"今日の空": {{Start: 0, End: 2, Reading: "きょう"}},
	}}
	a.ann = ann
	tr := song("7", "宝船")
	lines := lyrics.Build("[00:01.00]今日の空", "[00:01.00]The sky")
	a.furiganaLines(&tr, lines)
	runPost(t, ch)
	got := a.furiganaLines(&tr, lines)
	if got[0].Translation != "The sky" {
		t.Errorf("translation %q", got[0].Translation)
	}
}

// japanesePlayingApp is an app playing a Japanese song, its lyrics read
// already, and a tester showing it.
func japanesePlayingApp(t *testing.T) (*app, *ui.Tester, *fakeAnnotator) {
	t.Helper()
	a := newTestApp()
	ch := posts(a)
	ann := &fakeAnnotator{spans: map[string][]lyrics.RubySpan{
		"今日の空はとても青い": {{Start: 0, End: 2, Reading: "きょう"}, {Start: 3, End: 4, Reading: "そら"}, {Start: 8, End: 9, Reading: "あお"}},
		"何度でも立ち上がってこの声が届くまで走り出した想いを止められない": {
			{Start: 0, End: 2, Reading: "なんど"}, {Start: 4, End: 5, Reading: "た"}, {Start: 6, End: 7, Reading: "あ"},
			{Start: 12, End: 13, Reading: "こえ"}, {Start: 14, End: 15, Reading: "とど"},
			{Start: 18, End: 19, Reading: "はし"}, {Start: 20, End: 21, Reading: "だ"},
			{Start: 23, End: 24, Reading: "おも"}, {Start: 26, End: 27, Reading: "と"},
		},
	}}
	a.ann = ann
	a.pl.Cue([]music.Track{song("7", "宝船", "DOUDOU")})
	lines := lyrics.Build("[00:01.00]今日の空はとても青い\n[00:08.00]何度でも立ち上がってこの声が届くまで走り出した想いを止められない", "[00:01.00]The sky is blue")
	a.pl.SetLyrics(lines)
	a.pl.Hold(2 * time.Second)
	tr := a.pl.Current()
	if tr == nil {
		t.Fatal("nothing plays")
	}
	a.furiganaLines(tr, lines)
	runPost(t, ch)
	tt := ui.NewTester(a.view, 1200, 800)
	return a, tt, ann
}

// The lyrics pane shows the readings over its kanji, and the button hides
// and shows them, without reading the song again.
func TestFuriganaButtonOnLyrics(t *testing.T) {
	a, tt, ann := japanesePlayingApp(t)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Move(600, 400)
	settled(tt)
	for _, want := range []string{"きょう", "そら", "あお", "The sky is blue"} {
		if !tt.HasText(want) {
			t.Errorf("no %q in the lyrics: %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Furigana"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if !a.settings.FuriganaOff {
		t.Error("the readings still show")
	}
	if tt.HasText("きょう") || tt.HasText("そら") {
		t.Errorf("readings show after hiding them: %q", tt.Texts())
	}
	if err := tt.Click("Furigana"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if a.settings.FuriganaOff || !tt.HasText("きょう") {
		t.Errorf("readings came back: off %v, %q", a.settings.FuriganaOff, tt.Texts())
	}
	if n := ann.count(); n != 2 {
		t.Errorf("read %d lines, want both once", n)
	}
}

// Every style of the stage shows the readings too, and its button hides
// them.
func TestFuriganaOnStageStyles(t *testing.T) {
	a, tt, _ := japanesePlayingApp(t)
	a.nowPlaying, a.npPane = true, paneStage
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Move(600, 400)
	for _, st := range stageStyles {
		a.settings.StageStyle = st.id
		settled(tt)
		for _, want := range []string{"きょう", "そら", "あお"} {
			if !tt.HasText(want) {
				t.Errorf("in %s: no %q on the stage: %q", st.label, want, tt.Texts())
			}
		}
	}
	if err := tt.Click("Furigana"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if !a.settings.FuriganaOff || tt.HasText("きょう") {
		t.Errorf("readings still show on the stage: off %v, %q", a.settings.FuriganaOff, tt.Texts())
	}
}

// With MM_FRAMES set, saves the lyrics pane and each stage style with the
// readings over their kanji.
func TestFuriganaFrames(t *testing.T) {
	dir := framesDir(t)
	a, tt, _ := japanesePlayingApp(t)
	tt.SetDark(true)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt.Move(600, 400)
	settled(tt)
	saveFrame(t, dir, "furigana-lyrics.png", tt.Image())
	for _, st := range stageStyles {
		a.settings.StageStyle = st.id
		a.npPane = paneStage
		settled(tt)
		name := "furigana-stage.png"
		if st.id != "" {
			name = "furigana-stage-" + st.id + ".png"
		}
		saveFrame(t, dir, name, tt.Image())
	}
}

// runPostWithin runs the next thing the app posted, waiting d for it: the
// reader's dictionary takes a while to load on a slow or race-detecting run.
func runPostWithin(t *testing.T, ch chan func(), d time.Duration) {
	t.Helper()
	select {
	case fn := <-ch:
		fn()
	case <-time.After(d):
		t.Fatal("nothing was posted")
	}
}

// The real reader turns a line's kanji into readings, and the app lays them
// over its words.
func TestFuriganaWithRealReader(t *testing.T) {
	a := newTestApp()
	ch := posts(a)
	a.pl.Cue([]music.Track{song("7", "宝船")})
	lines := lyrics.Build("[00:01.00]今日の空はとても青い", "")
	a.pl.SetLyrics(lines)
	a.pl.Hold(2 * time.Second)
	tr := a.pl.Current()
	a.furiganaLines(tr, lines)
	runPostWithin(t, ch, 10*time.Second)
	got := a.furiganaLines(tr, lines)
	seen := map[string]string{}
	for _, w := range got[0].Words {
		if w.Ruby != "" {
			seen[w.Text] = w.Ruby
		}
	}
	for text, ruby := range map[string]string{"今日": "きょう", "空": "そら", "青": "あお"} {
		if seen[text] != ruby {
			t.Errorf("%s read %q, want %q (all %v)", text, seen[text], ruby, got[0].Words)
		}
	}
}

// With MM_FRAMES set, saves the lyrics pane and the stage styles with the
// readings the real reader gives them.
func TestFuriganaRealFrames(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	ch := posts(a)
	a.pl.Cue([]music.Track{song("7", "宝船", "DOUDOU")})
	lines := lyrics.Build("[00:01.00]嗚呼\n[00:04.00]存在証明 見つけたい\n[00:08.00]今日の空はとても青い", "")
	a.pl.SetLyrics(lines)
	a.pl.Hold(2 * time.Second)
	tr := a.pl.Current()
	a.furiganaLines(tr, lines)
	runPostWithin(t, ch, 10*time.Second)
	tt := ui.NewTester(a.view, 1200, 800)
	tt.SetDark(true)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt.Move(600, 400)
	settled(tt)
	saveFrame(t, dir, "furigana-real-lyrics.png", tt.Image())
	for _, st := range stageStyles {
		a.settings.StageStyle = st.id
		a.npPane = paneStage
		settled(tt)
		name := "furigana-real-stage.png"
		if st.id != "" {
			name = "furigana-real-stage-" + st.id + ".png"
		}
		saveFrame(t, dir, name, tt.Image())
	}
}

// A Japanese song is judged by its lyrics whole: a line of kanji alone is
// read too, and a line's spaces stay between its words.
func TestFuriganaLinesWholeSong(t *testing.T) {
	for _, c := range []struct {
		name  string
		lrc   string
		spans map[string][]lyrics.RubySpan
	}{
		{
			"kanji only",
			"[00:01.00]嗚呼\n[00:08.00]きょうのそら",
			map[string][]lyrics.RubySpan{"嗚呼": {{Start: 0, End: 2, Reading: "ああ"}}},
		},
		{
			"spaces",
			"[00:01.00]存在証明 見つけたい",
			map[string][]lyrics.RubySpan{"存在証明 見つけたい": {
				{Start: 0, End: 2, Reading: "そんざい"}, {Start: 2, End: 4, Reading: "しょうめい"}, {Start: 5, End: 6, Reading: "み"},
			}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := newTestApp()
			ch := posts(a)
			a.ann = &fakeAnnotator{spans: c.spans}
			tr := song("7", "宝船")
			lines := lyrics.Build(c.lrc, "")
			a.furiganaLines(&tr, lines)
			runPost(t, ch)
			got := a.furiganaLines(&tr, lines)
			if !got[0].HasRuby() {
				t.Errorf("no readings: words %+v", got[0].Words)
			}
		})
	}
}
