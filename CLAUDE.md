# MyGo Music — Furigana Fork

This working tree is a fork of [MyGo Music](https://github.com/egoist/mygo-music) with
**built-in furigana (振假名) over Japanese lyrics**. It is not a plugin: the
feature lives in the app's Go core. `E:\My Files\Programming\Projects\mygo-furigana-plugin`
is intentionally unused; the implementation is here.

See `AGENTS.md` for the project's general test rules.

## Status

- `personal` is the base branch, holding the furigana work (commit 6510257),
  then merged with `upstream/main` 0.1.25 (merge commit 7ebbf49, 2026-11-15).
- `go build ./...` and `go test ./internal/...` pass, except the upstream
  `TestPrevSkipsBack` below.
- Japanese lyrics are now set in the app's Chinese font family, matching the
  Chinese and English lyrics; see “Ruby layout”.
- Test build version: `mygo.config.ts` → `0.1.25-furigana.1` (semver prerelease).
  The app's plugin version parser treats this as non-numeric and skips
  `minAppVersion` checks, which is fine for a local fork.
- Upstream 0.1.25 adds vinyl/cassette Now Playing players; its rewritten
  cover layout (`npCoverTop`, `over-cover`/`under-controls`) is in place and
  the furigana button in the tab row still sits above it.
- Pre-existing failure unrelated to this work:
  `go test ./internal/player -run TestPrevSkipsBack` fails on this Windows
  machine on `upstream/main` 0.1.25 alone too (reproduced 5/5 in a clean
  worktree), so it is not from the merge.

## Branches

- `personal` is the base branch and the fork's product line: all custom work
  lands here. Features branch from it and return with `git merge --no-ff`.
- `main` mirrors `upstream/main` and takes no commits of its own.
- Sync upstream by fetching `upstream`, merging `upstream/main` into
  `personal`, and fast-forwarding `main`. Test locally with
  `go build -o /dev/null .` and `go test ./internal/...`: pushes to
  `personal` do not trigger the GitHub test workflow.

## Build the Windows installer

```bash
scripts/build-windows.sh            # build/windows-amd64, installer included
scripts/build-windows.sh --arm64    # also windows/arm64
scripts/build-windows.sh --portable # the app alone, without the installer
```

The script runs `go build ./...`, then
`go tool mygo build -platform windows/amd64`, and prints the installer's
SHA-256. The underlying command is the second one; `mygo build` also writes
`build/SHA256SUMS` and the portable `MyGo Music.exe`.

- `mygo.config.ts` is evaluated with Bun or Node.js 22.6 or later. Node is
  installed on this machine; Bun usually is not.
- `mygo build` downloads NSIS on Windows when `makensis` is missing.
- The installer is unsigned, so SmartScreen may warn on another machine.
- Updater signing is skipped without `MYGO_UPDATER_PRIVATE_KEY`; expected.
- The version comes from `mygo.config.ts`; do not bump it for a local build.

The workflow is also the `build-windows-installer` skill
(`.claude/skills/build-windows-installer/SKILL.md`).

## What it does

- Reads kanji in Japanese lyrics with **kagome + IPADIC**, offline.
- Shows readings over the kanji in:
  - the Lyrics pane (`paneLyrics`);
  - the Stage pane (`paneStage`), in all three styles: Flow, Steps, Tilt.
- One shared toggle in both panes; state is persisted in
  `settings.FuriganaOff` (default `false`, i.e. readings show).
- Japanese lyrics are set in the app's Chinese font family, so they match the
  Chinese and English lyrics; translations keep the app font.
- Translations (`lyrics.Line.Translation`) are never annotated.
- Chinese and other non-Japanese songs are skipped.

## Architecture

### `internal/furigana/furigana.go`

- `HasKanji(text)` — a line has something to read.
- `Annotator.Annotate(text)` — returns `[]lyrics.RubySpan` at **rune offsets**.
  - Lazy `tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())`; the first
    Japanese song loads the dictionary in a background goroutine.
  - 4096-line FIFO cache keyed by line text.
  - Rejects a line if kagome's tokens don't spell it exactly.
- `readingParts(surface, reading)` — splits a token's reading over its runs of
  kanji (走り出 / はしりだ → 走(はし)り出(だ)).
- `trimReading` — okurigana fallback (食べる / たべる → 食(た)).
- Katakana readings are converted to hiragana; `ー` and `・` are allowed.
- `Token.Reading()` values of empty or `*` are skipped (unknown words/names).
- Annotate is gated by `HasKanji`, **not** by "kanji + kana". The caller
  decides whether the song is Japanese from the whole lyrics.

### `internal/lyrics/lyrics.go`

- `Word.Ruby` — hiragana reading over a word.
- `RubySpan{Start, End, Reading}` — rune offsets, end exclusive.
- `ApplyRuby(lines, spans)`:
  - maps spans onto timed words;
  - merges a span over several per-character words into one word;
  - splits a word's time proportionally when a span covers only part of it;
  - ignores out-of-range/overlapping spans;
  - if a source's `Words` don't spell `Line.Text`, re-spreads the text with
    `SpreadWords` before applying.
- `SpreadWords` **must keep spaces** with the preceding word. It cuts kana and
  kanji one rune at a time; spaces therefore attach to the previous word
  rather than being dropped. Without this, the words don't spell the line and
  `ApplyRuby` rejects the whole line.

### `internal/app/furigana.go`

- `furiganaShown(lines)` — the song-level gate:
  `lyrics.Lang(lines) == "ja"` **and** at least one line has `HasKanji`.
  This is what lets a kanji-only line such as `嗚呼` be read when the rest of
  the lyrics contain kana.
- `furiganaLines(tr, lines)` — returns cached annotated lines, or starts one
  background read and returns the plain lines meanwhile.
  - Cache key: `track.Source + "\x00" + track.ID`.
  - Cache hash: FNV-1a of all line texts; new lyrics invalidate.
  - One generation counter; stale results are dropped.
  - A panic marks the state failed; toggling the button retries.
- `furiganaButton(c)` — the “Furigana / 振假名” pill in the Lyrics and Stage
  tab rows. It is only shown when `furiganaShown` is true.

### Ruby layout

- `internal/app/lyrics_font.go`
  - `lyricsFamily(lang)` — for Japanese only, the family the app draws
    Chinese in (Windows `Microsoft YaHei UI`, macOS `PingFang SC`, Linux
    `Noto Sans CJK SC`), so Japanese lyrics set at the same weight and
    spacing as Chinese and Latin.
  - Without it, `Lang("ja")` makes DirectWrite fall back to Yu Gothic UI,
    which sets thinner and tighter at weight 800.
  - Language stays `ja` (the engine shapes the text as Japanese); only the
    family changes. Tradeoff: shared kanji take Chinese glyph forms.
  - Applied in the Lyrics pane (`ui_nowplaying.go`), all Stage styles
    (the `sung` wrapper in `ui_stage.go`, so translations override with
    `Font("")`) and the Wall pane (`ui_wall.go`); `rubyWidth` measures in
    the same family.
- `internal/app/ui_ruby.go`
  - `rubyScale = 0.42`.
  - `rubyColumn` — reading centered above the word, spacer above words with
    no reading so all bases share a baseline.
  - `rubyWidth` — the wider of base/reading, measured with `Lang: "ja"` and
    `lyricsFamily("ja")`.
  - Ruby text is `AccessHidden(true)` so screen readers don't repeat it.
- `internal/app/ui_nowplaying.go` — Lyrics pane:
  - active line: per-word lit colors, ruby included;
  - inactive lines with ruby: a wrapping `Row` of ruby columns;
  - lines without ruby keep the original `Text`/`RichText` rendering;
  - translations render unchanged.
- `internal/app/ui_stage.go` / `ui_stage_styles.go`:
  - `stageShapes`/`partitaLine` measure with `rubyWidth`;
  - `stageWord(..., ruby bool)` renders a ruby column and offsets the glow to
    the base text;
  - Tilt wraps each word's runes in a column under its reading.

### Stage layout: where an absolute box hangs

`ui` is immediate mode: `ui.Box(...)` attaches to whichever parent is current
when it is **created**, not when it is filled. An absolute box created before
`parent.Children(...)` therefore hangs off `parent`'s own parent, and its
`Top/Left/Right/Bottom` resolve against that outer box.

This bit the `sung` wrapper in `ui_stage.go`: it was created before
`box.Children` so it could take `lyricsFamily(lang)`, which made it a child of
the full-window column. The translation/next-line column inside it resolved
`Bottom(24)` from the window's foot — right on `stageBar`'s controls. Create
such nodes **inside** `box.Children(func() { ... })`; a font or language can
be set there just as well.

Regression test: `TestStageTranslationStaysAboveTheBar` in
`internal/app/stage_test.go` renders the Stage pane at 1180×780 and checks
that the translation and the line next sit above the bar's song title. On the
old ordering it fails (translation ends at y≈729, the title starts at y≈712).

### Settings and i18n

- `settings.FuriganaOff` (`json:"furiganaOff,omitempty"`), default false.
- `internal/i18n/zh_cn.go`: `"Furigana": "振假名"` and the tooltip.

## Decisions and caveats

- **Japanese is judged from the whole lyrics** (`lyrics.Lang(lines)`), then
  every line with kanji is annotated. This fixes all-kanji lines.
- Tradeoff: a Chinese song containing any kana line, or a Japanese LRC that
  embeds Chinese translation lines as extra lyric lines, may have kanji-only
  lines read as Japanese. The proper `Line.Translation` field is untouched.
- A song whose entire lyrics contain no kana cannot be reliably told from
  Chinese; it gets no furigana. A manual “force Japanese” option would be the
  next step if needed.
- Unknown names/uncommon words with no IPADIC reading are left unannotated
  rather than guessed. A user dictionary or per-kanji fallback could improve
  this.
- Japanese lyrics take the app's Chinese font family (`lyricsFamily`) instead
  of DirectWrite's Japanese fallback, Yu Gothic UI, which sets thinner and
  tighter at weight 800. Tradeoff: shared kanji show Chinese glyph forms
  (e.g. 直, 骨). To favor correct Japanese glyphs, make `lyricsFamily` return
  `""` (Yu Gothic UI again) or `Meiryo UI`, whose spacing and weight sit closer
  to YaHei.
- The dictionary loads in the background on the first Japanese song
  (~0.3 s, ~50 MB). The Windows binary is about 52 MB including IPADIC.

## Tests and checks

```bash
go build ./...

# Everything except the pre-existing player failure.
go test ./internal/...

# New UI tests, repeated for flakiness.
go test ./internal/app -run TestFurigana -count=5

# Race detection for the async annotation path.
go test -race ./internal/lyrics ./internal/furigana ./internal/app -run 'TestSpreadWords|TestApplyRuby|TestFurigana'

golangci-lint run ./internal/furigana ./internal/lyrics
```

Screenshots (skipped without `MM_FRAMES`):

```bash
MM_FRAMES=/tmp/furigana-frames go test ./internal/app -run 'TestFuriganaFrames|TestFuriganaRealFrames' -v
```

- `TestFuriganaFrames` uses a fake annotator.
- `TestFuriganaRealFrames` uses kagome and saves the Lyrics pane plus all three
  Stage styles: `furigana-real-lyrics.png`, `furigana-real-stage*.png`.

## Files added/changed

Added:

- `internal/furigana/furigana.go`, `furigana_test.go`
- `internal/app/furigana.go`, `furigana_test.go`, `ui_ruby.go`,
  `lyrics_font.go`
- `internal/lyrics/ruby_test.go`

Changed:

- `internal/lyrics/lyrics.go`, `lyrics_test.go`
- `internal/app/app.go`, `settings.go`
- `internal/app/ui_nowplaying.go`
- `internal/app/ui_stage.go`, `ui_stage_styles.go`
- `internal/app/stage_test.go` (stage layout regression)
- `internal/app/ui_wall.go`
- `internal/i18n/zh_cn.go`
- `README.md`, `go.mod`, `go.sum`, `mygo.config.ts`
- `CLAUDE.md`

## Possible next steps

1. Add a compact user/name dictionary or per-kanji fallback for IPADIC misses.
2. Add a “force Japanese” option for all-kanji lyrics.
3. Shield bilingual LRC translation lines (e.g. same timestamp, kanji-only)
   from annotation.
4. If the built-in choice is ever reversed, extract `internal/furigana` behind
   a generic lyrics-annotation plugin API; the core currently has no such hook.
