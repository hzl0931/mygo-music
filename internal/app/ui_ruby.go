package app

import (
	"github.com/egoist/mygo/ui"

	"mygo-music/internal/lyrics"
)

// rubyScale is how large a reading is beside the word it reads: about a
// third of it, as ruby is set.
const rubyScale = 0.42

// rubyStyle is how a word and its reading are set.
type rubyStyle struct {
	size          float32
	weight        int
	color         ui.Color
	reading       ui.Color
	lineHeight    float32
	letterSpacing float32
	italic        bool
	blur          float32
	lang          string
}

// rubyColumn shows a word with its reading centered above it. A word without
// a reading keeps the room, so every word of a line sits on one baseline.
func rubyColumn(c *ui.Context, w lyrics.Word, st rubyStyle) ui.Element {
	col := ui.Column(c).AlignItems(ui.Center)
	fam := lyricsFamily(st.lang)
	col.Children(func() {
		h := rubyHeight(st.size)
		if w.Ruby != "" {
			r := ui.Text(c, w.Ruby).FontSize(st.size * rubyScale).FontWeight(600).LineHeight(1).
				TextColor(st.reading).NoWrap().AccessHidden(true).Height(h)
			if st.lang != "" {
				r.Lang(st.lang)
			}
			if fam != "" {
				r.Font(fam)
			}
			if st.blur > 0.3 {
				r.Blur(st.blur)
			}
		} else {
			ui.Box(c).Height(h)
		}
		t := ui.Text(c, w.Text).FontSize(st.size).FontWeight(st.weight).TextColor(st.color).NoWrap()
		if st.lang != "" {
			t.Lang(st.lang)
		}
		if fam != "" {
			t.Font(fam)
		}
		if st.lineHeight > 0 {
			t.LineHeight(st.lineHeight)
		}
		if st.letterSpacing != 0 {
			t.LetterSpacing(st.letterSpacing)
		}
		if st.italic {
			t.Italic()
		}
		if st.blur > 0.3 {
			t.Blur(st.blur)
		}
	})
	return col
}

// rubyWidth is how wide a word shows with its reading: the wider of the two,
// both in the Japanese font that sets them.
func rubyWidth(c *ui.Context, w lyrics.Word, size float32, weight int) float32 {
	base := ui.Span{Text: w.Text, Size: size, Weight: weight}
	if w.Ruby != "" {
		base.Lang = "ja"
		base.Font = lyricsFamily("ja")
	}
	width, _ := c.MeasureText(0, base)
	if w.Ruby != "" {
		if rw, _ := c.MeasureText(0, ui.Span{Text: w.Ruby, Size: size * rubyScale, Weight: 600, Lang: "ja", Font: lyricsFamily("ja")}); rw > width {
			width = rw
		}
	}
	return width
}

// rubyHeight is the room a reading takes above its word.
func rubyHeight(size float32) float32 { return size * rubyScale * 1.2 }

// rubyWord shows a word of the lyrics pane as the line sets it, its reading
// a little dimmer than it is.
func (a *app) rubyWord(c *ui.Context, w lyrics.Word, st linesStyle, color ui.Color, lang string) {
	rubyColumn(c, w, rubyStyle{
		size:          st.size,
		weight:        st.weight,
		color:         color,
		reading:       color.Alpha(0.75),
		lineHeight:    st.lineHeight,
		letterSpacing: -0.4,
		lang:          lang,
	})
}
