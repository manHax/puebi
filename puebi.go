package puebi

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeToPUEBI merapikan teks:
// - Normalisasi spasi & tanda baca
// - Perbaikan preposisi umum
// - Normalisasi frasa khusus: "real time" (kecil di tengah kalimat)
// - Kapital awal kalimat
// - Turunkan kapital nyasar di tengah kalimat (heuristik)
// - Normalisasi format "Rp12.000"
func SanitizeToPUEBI(s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}

	s = normalizeSpaces(s)
	s = fixPunctuationSpacing(s)
	s = fixCommonPrepositions(s)

	// Frasa khusus yang harus kecil di tengah kalimat.
	// Diletakkan sebelum capitalizeSentences agar di awal kalimat tetap kapital kata pertamanya.
	s = normalizeRealTime(s)

	// Kapitalisasi awal kalimat
	s = capitalizeSentences(s)

	// Turunkan kapital yang nyasar di tengah kalimat
	s = decapitalizeMidSentence(s, defaultExceptions())

	// Format Rupiah
	s = fixIDRCurrency(s)

	return strings.TrimSpace(s)
}

// --- Helpers ---

func normalizeSpaces(s string) string {
	ws := regexp.MustCompile(`\s+`)
	s = ws.ReplaceAllString(s, " ")
	// hilangkan spasi sebelum tanda baca penutup
	s = regexp.MustCompile(`\s+([,.;:!?])`).ReplaceAllString(s, "$1")
	return s
}

func fixPunctuationSpacing(s string) string {
	// 1) Hilangkan spasi sebelum tanda baca umum
	s = regexp.MustCompile(`\s+([,.;:!?])`).ReplaceAllString(s, "$1")
	// 2) Tambah satu spasi setelah , ; : ? ! jika setelahnya bukan spasi/")" dan bukan akhir
	s = regexp.MustCompile(`([,;:!?])([^\s\)])`).ReplaceAllString(s, "$1 $2")
	// 3) Titik akhir kalimat: beri spasi setelahnya (hindari angka desimal sederhana)
	s = regexp.MustCompile(`(^|[^\d])\.([^\d\s\).])`).ReplaceAllString(s, "$1. $2")
	// 4) Kurung: tidak ada spasi setelah "(" dan tidak ada spasi sebelum ")"
	s = regexp.MustCompile(`\(\s+`).ReplaceAllString(s, "(")
	s = regexp.MustCompile(`\s+\)`).ReplaceAllString(s, ")")
	// 5) Rapatkan spasi ganda
	s = regexp.MustCompile(`\s{2,}`).ReplaceAllString(s, " ")
	return s
}

func fixCommonPrepositions(s string) string {
	// di/ke + (luar, dalam, atas, bawah, depan, belakang, samping, antara)
	locatives := []string{"luar", "dalam", "atas", "bawah", "depan", "belakang", "samping", "antara"}
	for _, w := range locatives {
		s = regexp.MustCompile(`\bdi`+w+`\b`).ReplaceAllString(s, "di "+w)
		s = regexp.MustCompile(`\bke`+w+`\b`).ReplaceAllString(s, "ke "+w)
	}

	// tempat umum
	places := []string{"rumah", "kantor", "sekolah", "pasar", "bank", "jalan", "masjid", "gereja", "kampus"}
	for _, w := range places {
		s = regexp.MustCompile(`\bdi`+w+`\b`).ReplaceAllString(s, "di "+w)
		s = regexp.MustCompile(`\bke`+w+`\b`).ReplaceAllString(s, "ke "+w)
	}

	// kepada / daripada
	s = regexp.MustCompile(`\b[Kk]e pada\b`).ReplaceAllString(s, "kepada")
	s = regexp.MustCompile(`\b[Dd]ari pada\b`).ReplaceAllString(s, "daripada")

	// di tiap/di setiap/di mana/di sini/di situ/di sana
	phrases := []string{"tiap", "setiap", "mana", "sini", "situ", "sana"}
	for _, w := range phrases {
		s = regexp.MustCompile(`\bdi`+w+`\b`).ReplaceAllString(s, "di "+w)
	}

	return s
}

// Frasa "real time" selalu kecil; bila ada "Transfer Real Time" di tengah,
// akan menjadi "transfer real time". Di awal kalimat nanti "Transfer real time".
func normalizeRealTime(s string) string {
	// variasi: "Real Time" / "Real-Time" / "Realtime"
	s = regexp.MustCompile(`(?i)\bReal[ -]?Time\b`).ReplaceAllString(s, "real time")
	// jika ada "Transfer Real Time" (title case), turunkan "Transfer" juga
	s = regexp.MustCompile(`(?i)\bTransfer\s+real time\b`).ReplaceAllString(s, "transfer real time")
	return s
}

// Turunkan kapital di tengah kalimat (bukan akronim/bukan rangkaian Title Case).
func decapitalizeMidSentence(s string, exceptions map[string]bool) string {
	type wordPos struct{ start, end int } // [start,end)
	rs := []rune(s)
	n := len(rs)

	isSentenceEnd := func(r rune) bool { return r == '.' || r == '!' || r == '?' }
	isLetter := func(r rune) bool { return unicode.IsLetter(r) }

	isAllCaps := func(runes []rune) bool {
		has := false
		for _, r := range runes {
			if unicode.IsLetter(r) {
				has = true
				if !unicode.IsUpper(r) {
					return false
				}
			}
		}
		return has
	}
	isTitleCase := func(runes []rune) bool {
		if len(runes) == 0 || !unicode.IsUpper(runes[0]) {
			return false
		}
		for i := 1; i < len(runes); i++ {
			if unicode.IsLetter(runes[i]) && !unicode.IsLower(runes[i]) {
				return false
			}
		}
		return true
	}

	var out []rune
	for i := 0; i < n; {
		// ambil satu kalimat
		j := i
		for j < n && !isSentenceEnd(rs[j]) {
			j++
		}
		kal := rs[i:j]

		// token kata
		var words []wordPos
		k := 0
		for k < len(kal) {
			for k < len(kal) && !isLetter(kal[k]) {
				k++
			}
			start := k
			for k < len(kal) && isLetter(kal[k]) {
				k++
			}
			if k > start {
				words = append(words, wordPos{start, k})
			}
		}

		// turunkan kapital nyasar
		for wi := 0; wi < len(words); wi++ {
			if wi == 0 {
				continue // kata pertama kalimat dibiarkan
			}
			wp := words[wi]
			wordRunes := kal[wp.start:wp.end]
			word := string(wordRunes)

			// lewati ALL CAPS (akronim) dan pengecualian
			if isAllCaps(wordRunes) || exceptions[word] {
				continue
			}
			// lewati kalau bagian dari rangkaian Title Case bertetangga (contoh: "Call Center")
			prevTitle := wi-1 >= 0 && isTitleCase(kal[words[wi-1].start:words[wi-1].end])
			nextTitle := wi+1 < len(words) && isTitleCase(kal[words[wi+1].start:words[wi+1].end])
			if prevTitle || nextTitle {
				continue
			}
			// turunkan TitleCase tunggal
			if isTitleCase(wordRunes) {
				for t := wp.start; t < wp.end; t++ {
					kal[t] = unicode.ToLower(kal[t])
				}
			}
		}

		out = append(out, kal...)

		// tambahkan delimiter & spasi setelahnya (apa adanya)
		if j < n && isSentenceEnd(rs[j]) {
			out = append(out, rs[j])
			j++
		}
		for j < n && unicode.IsSpace(rs[j]) {
			out = append(out, rs[j])
			j++
		}
		i = j
	}
	return string(out)
}

// Pengecualian TitleCase yang tetap kapital walau di tengah kalimat.
func defaultExceptions() map[string]bool {
	return map[string]bool{
		"Indonesia": true,
		"Jakarta":   true,
		"Bank":      true,
		"Sahabat":   true,
		"Sampoerna": true,
		"Call":      true,
		"Center":    true,
		"ATM":       true,
		"KTP":       true,
		"BI":        true,
		"BNI":       true,
		"BCA":       true,
	}
}

func capitalizeSentences(s string) string {
	r := []rune(s)
	n := len(r)
	if i := firstLetterIndex(r, 0); i >= 0 {
		r[i] = unicode.ToUpper(r[i])
	}
	for idx := 0; idx < n; idx++ {
		if r[idx] == '.' || r[idx] == '!' || r[idx] == '?' {
			if j := firstLetterIndex(r, idx+1); j >= 0 {
				r[j] = unicode.ToUpper(r[j])
			}
		}
	}
	return string(r)
}

func firstLetterIndex(r []rune, start int) int {
	for i := start; i < len(r); i++ {
		if unicode.IsLetter(r[i]) {
			return i
		}
	}
	return -1
}

// Normalisasi format Rupiah: "rp 12.000" / "Rp 12.000" / "Rp    12.000" -> "Rp12.000"
func fixIDRCurrency(s string) string {
	// Hilangkan spasi setelah Rp / rp (tanpa mengubah penulisan angka)
	re := regexp.MustCompile(`\b[Rr]p\s+([0-9])`)
	s = re.ReplaceAllString(s, "Rp$1")
	// Pastikan "Rp" kapital
	re2 := regexp.MustCompile(`\brp([0-9])`)
	s = re2.ReplaceAllString(s, "Rp$1")
	return s
}

// TitleCase sederhana
func TitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		rs := []rune(w)
		if len(rs) == 0 {
			continue
		}
		rs[0] = unicode.ToUpper(rs[0])
		for j := 1; j < len(rs); j++ {
			rs[j] = unicode.ToLower(rs[j])
		}
		words[i] = string(rs)
	}
	return strings.Join(words, " ")
}

func IsSentenceCapitalized(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsLetter(r) && unicode.IsUpper(r)
}
