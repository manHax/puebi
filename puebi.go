package puebi

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeToPUEBI merapikan teks:
// - Normalisasi spasi
// - Spasi yang benar di sekitar tanda baca , . ! ? : ; ) (
// - Kapital huruf pertama setiap kalimat
// - Perbaikan umum kata depan/preposisi: di/ke/dari + lokatif, "kepada", "daripada"
func SanitizeToPUEBI(s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}

	s = normalizeSpaces(s)
	s = fixPunctuationSpacing(s)
	s = fixCommonPrepositions(s)
	s = capitalizeSentences(s)

	return strings.TrimSpace(s)
}

// --- Helpers ---

func normalizeSpaces(s string) string {
	// Jadikan semua whitespace beruntun -> 1 spasi
	ws := regexp.MustCompile(`\s+`)
	s = ws.ReplaceAllString(s, " ")
	// Hilangkan spasi di awal sebelum tanda baca penutup
	s = regexp.MustCompile(`\s+([,.;:!?])`).ReplaceAllString(s, "$1")
	return s
}

func fixPunctuationSpacing(s string) string {
	// 1) Hilangkan spasi sebelum tanda baca umum
	s = regexp.MustCompile(`\s+([,.;:!?])`).ReplaceAllString(s, "$1")

	// 2) Pastikan ada satu spasi setelah , ; : ? ! (kecuali sebelum tanda tutup atau akhir)
	s = regexp.MustCompile(`([,;:!?])(?!\s|$|\))`).ReplaceAllString(s, "$1 ")

	// 3) Titik: kalau titik akhir kalimat, beri spasi setelahnya (kecuali akhir/penutup)
	// (hindari angka desimal sederhana: digit.digit)
	s = regexp.MustCompile(`(?<!\d)\.(?!\d)(?!\s|$|\))`).ReplaceAllString(s, ". ")

	// 4) Kurung: tidak ada spasi setelah "(" dan tidak ada spasi sebelum ")"
	s = regexp.MustCompile(`\(\s+`).ReplaceAllString(s, "(")
	s = regexp.MustCompile(`\s+\)`).ReplaceAllString(s, ")")

	// 5) Strip spasi ganda yang mungkin tercipta
	s = regexp.MustCompile(`\s{2,}`).ReplaceAllString(s, " ")
	return s
}

func fixCommonPrepositions(s string) string {
	// Perbaiki gabungan kata depan lokatif umum: di/ke + luar/dalam/atas/bawah/depan/belakang/samping/antara
	locatives := []string{"luar", "dalam", "atas", "bawah", "depan", "belakang", "samping", "antara"}
	for _, w := range locatives {
		s = regexp.MustCompile(`\bdi`+w+`\b`).ReplaceAllString(s, "di "+w)
		s = regexp.MustCompile(`\bke`+w+`\b`).ReplaceAllString(s, "ke "+w)
	}

	// "dirumah" → "di rumah", "dikantor" → "di kantor" (daftar kecil kata tempat umum)
	places := []string{"rumah", "kantor", "sekolah", "pasar", "bank", "jalan", "masjid", "gereja", "kampus"}
	for _, w := range places {
		s = regexp.MustCompile(`\bdi`+w+`\b`).ReplaceAllString(s, "di "+w)
		s = regexp.MustCompile(`\bke`+w+`\b`).ReplaceAllString(s, "ke "+w)
	}

	// “kepada” (bukan “ke pada”) & “daripada” (umumnya jadi satu)
	s = regexp.MustCompile(`\b[Kk]e pada\b`).ReplaceAllString(s, "kepada")
	s = regexp.MustCompile(`\b[Dd]ari pada\b`).ReplaceAllString(s, "daripada")

	// “di tiap/di setiap/di mana/di sini/di situ/di sana” harus terpisah
	phrases := []string{"tiap", "setiap", "mana", "sini", "situ", "sana"}
	for _, w := range phrases {
		s = regexp.MustCompile(`\bdi`+w+`\b`).ReplaceAllString(s, "di "+w)
	}

	return s
}

func capitalizeSentences(s string) string {
	// Kapital huruf pertama setiap kalimat setelah (. ! ?) dan di awal teks
	r := []rune(s)
	n := len(r)

	// Kapital huruf pertama non-spasi di awal
	i := firstLetterIndex(r, 0)
	if i >= 0 {
		r[i] = unicode.ToUpper(r[i])
	}

	// Setelah tanda akhir kalimat
	for idx := 0; idx < n; idx++ {
		if r[idx] == '.' || r[idx] == '!' || r[idx] == '?' {
			// cari huruf berikutnya
			j := firstLetterIndex(r, idx+1)
			if j >= 0 {
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
		// lewati spasi/quote/kurung dsb
	}
	return -1
}

// Optional: TitleCase untuk judul (bukan EYD wajib, tapi kadang berguna)
func TitleCase(s string) string {
	return strings.Map(func(r rune) rune {
		return r
	}, strings.Title(strings.ToLower(s))) // deprecated, tapi cukup untuk contoh singkat
}

// QuickCheck: periksa apakah string sudah punya kapital awal kalimat yang benar (opsional)
func IsSentenceCapitalized(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsLetter(r) && unicode.IsUpper(r)
}
