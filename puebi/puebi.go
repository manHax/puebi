package puebi

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeToPUEBI merapikan teks:
// - Normalisasi spasi & tanda baca
// - Perbaikan preposisi umum (di/ke terpisah saat preposisi; kepada/daripada disatukan)
// - Normalisasi frasa khusus (real time) tanpa memaksa tanda hubung
// - Kapital awal kalimat
// - Turunkan kapital nyasar di tengah kalimat (heuristik, dengan konteks & pengecualian)
// - Normalisasi format "Rp12.000"
func SanitizeToPUEBI(s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}

	s = normalizeSpaces(s)
	s = fixPunctuationSpacing(s)
	s = fixCommonPrepositions(s)

	// frasa khusus; diletakkan sebelum kapitalisasi agar awal kalimat tetap benar
	s = normalizeRealTime(s)

	// Kapitalisasi awal kalimat
	s = capitalizeSentences(s)

	// >>> Baru: pastikan nama setelah "Hai" huruf besar
	s = fixGreetingNameCase(s)

	// Turunkan kapital yang nyasar di tengah kalimat
	s = decapitalizeMidSentence(s, defaultExceptions(), protectedHeads())

	// Format Rupiah
	s = fixIDRCurrency(s)

	return strings.TrimSpace(s)
}

/* ---------- Precompiled regex ---------- */

var (
	reMultiWS                = regexp.MustCompile(`\s+`)
	reSpaceBeforePunct       = regexp.MustCompile(`\s+([,.;:!?])`)
	reCommaSemiColonEtcSpace = regexp.MustCompile(`([,;:!?])([^\s\)])`)
	reDotSentenceSpace       = regexp.MustCompile(`(^|[^\d])\.([^\d\s\).])`) // hindari pecah angka desimal sederhana
	reOpenParenSpaces        = regexp.MustCompile(`\(\s+`)
	reCloseParenSpaces       = regexp.MustCompile(`\s+\)`)

	// Elipsis: rapikan ke "..." lalu atur spasi kiri/kanan secara aman
	reMultiDots         = regexp.MustCompile(`[.]{3,}|…`)
	reEllipsisNoLeftSp  = regexp.MustCompile(`([^ \t\n\r(\["'])\.{3}`)
	reEllipsisNoRightSp = regexp.MustCompile(`\.{3}([^ \t\n\r)\]"'».,;:!?])`)

	// Kutip: rapikan spasi sebelum kutip penutup & setelah kutip pembuka, tanpa memaksa spasi setelah kutip
	reSpaceBeforeCloseQuote = regexp.MustCompile(`\s+(['"])`)
	reSpaceAfterOpenQuote   = regexp.MustCompile(`(['"])\s+`)

	// Em-dash: rapat kiri/kanan
	reDashTighten = regexp.MustCompile(`\s*—\s*`)

	// real time
	reRealTimeGeneric  = regexp.MustCompile(`(?i)\bReal[ -]?Time\b`)
	reTransferRealTime = regexp.MustCompile(`(?i)\bTransfer\s+real time\b`)

	// Rupiah
	reRpNumber  = regexp.MustCompile(`(?i)\brp\.?\s+([0-9])`)
	reRpNoSpace = regexp.MustCompile(`(?i)\brp([0-9])`)
)

/* ---------- Helpers ---------- */

// TitleCase satu kata (huruf pertama besar, sisanya kecil)
func titleWord(w string) string {
	rs := []rune(w)
	if len(rs) == 0 {
		return w
	}
	rs[0] = unicode.ToUpper(rs[0])
	for i := 1; i < len(rs); i++ {
		rs[i] = unicode.ToLower(rs[i])
	}
	return string(rs)
}

// Memastikan kata/deret nama setelah "Hai" ditulis kapital.
// Contoh: "Hai luqmanul hakim," -> "Hai Luqmanul Hakim,"
func fixGreetingNameCase(s string) string {
	// Ambil segmen setelah "Hai " sampai tanda baca penutup umum.
	re := regexp.MustCompile(`\bHai\b\s+([^\n\r,.!?;:()]+)`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// pecah jadi: "Hai " + sisa
		parts := strings.SplitN(match, " ", 2)
		if len(parts) < 2 {
			return match
		}
		head := parts[0] // "Hai"
		rest := parts[1]

		// tokenisasi nama sampai sebelum tanda baca (sudah dipastikan oleh regex)
		tokens := strings.Fields(rest)
		if len(tokens) == 0 {
			return match
		}

		// batasi 1–4 token pertama sebagai nama (umum: 1–3),
		// selebihnya biarkan (mis. "Hai Agus segera ...")
		limit := len(tokens)
		if limit > 4 {
			limit = 4
		}
		for i := 0; i < limit; i++ {
			// hanya kapitalisasi token alfabet (abaikan angka/simbol)
			allLetter := true
			for _, r := range []rune(tokens[i]) {
				if !unicode.IsLetter(r) && r != '\'' && r != '-' {
					allLetter = false
					break
				}
			}
			if allLetter {
				tokens[i] = titleWord(tokens[i])
			}
		}

		return head + " " + strings.Join(tokens, " ")
	})
}

func normalizeSpaces(s string) string {
	s = reMultiWS.ReplaceAllString(s, " ")
	// hilangkan spasi sebelum tanda baca penutup
	s = reSpaceBeforePunct.ReplaceAllString(s, "$1")
	return s
}

func fixPunctuationSpacing(s string) string {
	// (1) Hilangkan spasi sebelum tanda baca umum
	s = reSpaceBeforePunct.ReplaceAllString(s, "$1")

	// (2) Elipsis → "..."
	s = reMultiDots.ReplaceAllString(s, "...")
	// pastikan ada spasi kiri jika sebelumnya bukan spasi atau pembuka
	s = reEllipsisNoLeftSp.ReplaceAllString(s, "$1 ...")
	// pastikan ada spasi kanan jika sesudahnya bukan spasi/penutup/tanda baca akhir
	s = reEllipsisNoRightSp.ReplaceAllString(s, "... $1")

	// (3) Tambah satu spasi setelah , ; : ? ! jika setelahnya bukan spasi/")" dan bukan akhir
	s = reCommaSemiColonEtcSpace.ReplaceAllString(s, "$1 $2")

	// (4) Titik akhir kalimat: beri spasi setelahnya (hindari angka desimal sederhana)
	s = reDotSentenceSpace.ReplaceAllString(s, "$1. $2")

	// (5) Kurung: tidak ada spasi setelah "(" dan sebelum ")"
	s = reOpenParenSpaces.ReplaceAllString(s, "(")
	s = reCloseParenSpaces.ReplaceAllString(s, ")")

	// (6) Kutip: hilangkan spasi berlebih di dalam kutip
	s = reSpaceBeforeCloseQuote.ReplaceAllString(s, "$1") // sebelum kutip penutup tidak ada spasi
	s = reSpaceAfterOpenQuote.ReplaceAllString(s, "$1")   // setelah kutip pembuka tidak ada spasi

	// (7) Em-dash dirapatkan
	s = reDashTighten.ReplaceAllString(s, "—")

	// (8) Rapatkan spasi ganda
	s = reMultiWS.ReplaceAllString(s, " ")
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

// Frasa "real time": jadikan huruf kecil bila bukan awal kalimat; tidak memaksa tanda hubung.
func normalizeRealTime(s string) string {
	s = reRealTimeGeneric.ReplaceAllString(s, "real time")
	// jika "Transfer real time", turunkan "Transfer" juga (bukan awal kalimat)
	s = reTransferRealTime.ReplaceAllString(s, "transfer real time")
	return s
}

// Kepala nama diri yang melindungi kata sesudahnya dari decap (contoh: "Jalan Sudirman", "Bank Indonesia").
func protectedHeads() map[string]bool {
	return map[string]bool{
		"Jalan":       true,
		"Gunung":      true,
		"Sungai":      true,
		"Danau":       true,
		"Kota":        true,
		"Provinsi":    true,
		"Universitas": true,
		"Institut":    true,
		"Sekolah":     true,
		"Rumah":       true, // Rumah Sakit
		"Bank":        true,
		"PT":          true,
		"CV":          true,
		"RS":          true, // Rumah Sakit
		"Hai":         true,
	}
}

// Turunkan kapital di tengah kalimat (bukan akronim / bukan rangkaian Title Case / bukan setelah head proper-noun).
func decapitalizeMidSentence(s string, exceptions map[string]bool, heads map[string]bool) string {
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

			// lewati ALL CAPS (akronim) dan pengecualian eksak
			if isAllCaps(wordRunes) || exceptions[word] {
				continue
			}

			// lewati kalau bagian dari rangkaian Title Case bertetangga (contoh: "Call Center", "Bank Indonesia")
			//prevTitle := wi-1 >= 0 && isTitleCase(kal[words[wi-1].start:words[wi-1].end])
			//nextTitle := wi+1 < len(words) && isTitleCase(kal[words[wi+1].start:words[wi+1].end])
			//if prevTitle || nextTitle {
			//	continue
			//}

			// lindungi kata setelah head proper-noun (Jalan/Gunung/Bank/Universitas/RS/PT/...)
			if wi-1 >= 0 {
				prev := string(kal[words[wi-1].start:words[wi-1].end])
				if heads[prev] {
					continue
				}
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
// (hindari memasukkan kata generik seperti "Bank" di sini; dilindungi via heads)
func defaultExceptions() map[string]bool {
	return map[string]bool{
		"Indonesia": true,
		"Jakarta":   true,
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

// Normalisasi format Rupiah:
// - "rp 12.000", "Rp. 12.000", "RP    12.000" -> "Rp12.000"
// - "rp12.000" -> "Rp12.000"
func fixIDRCurrency(s string) string {
	s = reRpNumber.ReplaceAllString(s, "Rp$1")
	s = reRpNoSpace.ReplaceAllString(s, "Rp$1")
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
	runes := []rune(s)
	if i := firstLetterIndex(runes, 0); i >= 0 {
		return unicode.IsUpper(runes[i])
	}
	_, _ = utf8.DecodeRuneInString(s)
	return true
}
