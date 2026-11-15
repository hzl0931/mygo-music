package app

import "runtime"

// lyricsFamily is the family Japanese lyrics are drawn in: the one the app
// draws Chinese in, so that they set as its Chinese and Latin do. The
// system's Japanese font, Yu Gothic UI, sets thinner and tighter at the same
// weight. Other languages take the app's font, "".
func lyricsFamily(lang string) string {
	if lang != "ja" {
		return ""
	}
	switch runtime.GOOS {
	case "darwin":
		return "PingFang SC"
	case "windows":
		return "Microsoft YaHei UI, Microsoft YaHei"
	}
	return "Noto Sans CJK SC, Noto Sans SC, Source Han Sans SC"
}
