package app

import (
	"hash/fnv"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/furigana"
	"mygo-music/internal/lyrics"
	"mygo-music/internal/music"
)

// lineAnnotator reads the kanji of a line. furigana.Annotator is one; tests
// give one of their own.
type lineAnnotator interface {
	Annotate(text string) []lyrics.RubySpan
}

// furiganaState is the readings of the song playing: the lyrics they were
// read for, and whether they are still being read.
type furiganaState struct {
	mu      sync.Mutex
	key     string // the track they are of
	hash    uint64 // of its lyrics
	lines   []lyrics.Line
	loading bool
	ready   bool
	failed  bool // reading them went wrong; don't try again
	gen     int
}

// annotator is the app's reader, made on first use. Its dictionary is large,
// and a song without kanji never needs it, so it is made for a song that has
// some, on the main thread, its reading for a goroutine.
func (a *app) annotator() lineAnnotator {
	if a.ann == nil {
		a.ann = furigana.New()
	}
	return a.ann
}

// furiganaShown reports whether lines have readings to show: those of a
// Japanese song, any line of which has kanji. The song's language is its
// lyrics whole, so a line of kanji alone, as 嗚呼, is read with the rest.
func furiganaShown(lines []lyrics.Line) bool {
	if lyrics.Lang(lines) != "ja" {
		return false
	}
	for _, l := range lines {
		if furigana.HasKanji(l.Text) {
			return true
		}
	}
	return false
}

// furiganaLines returns lines with their readings, if they are read yet,
// and starts reading them otherwise. It never holds up a frame: the reading
// is for a goroutine, and the lines show plain until it is done.
func (a *app) furiganaLines(tr *music.Track, lines []lyrics.Line) []lyrics.Line {
	if tr == nil || a.settings.FuriganaOff || !furiganaShown(lines) {
		return lines
	}
	key, hash := tr.Source+"\x00"+tr.ID, linesHash(lines)
	st := a.furi
	if st == nil {
		st = &furiganaState{}
		a.furi = st
	}
	st.mu.Lock()
	if st.key == key && st.hash == hash {
		switch {
		case st.ready:
			out := st.lines
			st.mu.Unlock()
			return out
		case st.loading, st.failed:
			st.mu.Unlock()
			return lines
		}
	}
	st.key, st.hash = key, hash
	st.lines, st.loading, st.ready, st.failed = nil, true, false, false
	st.gen++
	gen := st.gen
	st.mu.Unlock()
	go a.readFurigana(st, a.annotator(), slices.Clone(lines), key, hash, gen)
	return lines
}

// readFurigana reads the kanji of lines and keeps the result for the track it
// was asked for, unless that track has gone by then.
func (a *app) readFurigana(st *furiganaState, ann lineAnnotator, lines []lyrics.Line, key string, hash uint64, gen int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("furigana: %v", r)
			st.mu.Lock()
			if st.gen == gen && st.key == key && st.hash == hash {
				st.loading, st.failed = false, true
			}
			st.mu.Unlock()
		}
	}()
	spans := make([][]lyrics.RubySpan, len(lines))
	for i := range lines {
		spans[i] = ann.Annotate(lines[i].Text)
	}
	out := lyrics.ApplyRuby(lines, spans)
	st.mu.Lock()
	if st.gen != gen || st.key != key || st.hash != hash {
		st.mu.Unlock()
		return
	}
	st.lines, st.loading, st.ready = out, false, true
	st.mu.Unlock()
	a.post(func() {})
}

// linesHash tells one set of lyrics from another: the same song played again
// has the same readings, and a source that answers new ones is read again.
func linesHash(lines []lyrics.Line) uint64 {
	h := fnv.New64a()
	for _, l := range lines {
		h.Write([]byte(l.Text))
		h.Write([]byte{0})
	}
	return h.Sum64()
}

// furiganaButton shows and hides the readings over Japanese lyrics, in the
// lyrics pane and on the stage. It shows only while there are kanji to read.
func (a *app) furiganaButton(c *ui.Context) {
	if a.pl == nil {
		return
	}
	lines, loaded := a.pl.Lyrics()
	if !loaded || !furiganaShown(lines) {
		return
	}
	on := !a.settings.FuriganaOff
	b := ui.ButtonBase(c.Key("furigana")).Padding(6, 10).Radius(999).FocusRing(false).
		Label(T("Furigana")).Tooltip(T("Show furigana over Japanese lyrics"))
	bg, fg := ui.Transparent, white.Alpha(0.55)
	if on {
		bg, fg = white.Alpha(0.14), white
	}
	if b.Hovered() {
		fg = white.Alpha(0.85)
		if on {
			bg = white.Alpha(0.2)
		} else {
			bg = white.Alpha(0.06)
		}
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond})).TextColor(fg)
	b.Children(func() {
		ui.Text(c, T("Furigana")).FontSize(13).FontWeight(600).TextColor(fg)
	})
	if b.Clicked() {
		a.settings.FuriganaOff = !a.settings.FuriganaOff
		// Turning them back on tries again if reading them went wrong.
		if st := a.furi; st != nil {
			st.mu.Lock()
			st.failed = false
			st.mu.Unlock()
		}
		_ = a.settings.save()
	}
}
