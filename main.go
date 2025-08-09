package main

import (
	"fmt"
	"github.com/manhax/puebi/puebi"
	"time"

	//"crypto/tls"
	//"gopkg.in/mail.v2"
	//"log"
	"strings"
)

const (
	email, pass = "bss.online@banksampoerna.com", "BSS4dm1n!"
	from, to    = "bss.online@banksampoerna.com", "mochamadfahmiandreanto@gmail.com"
	server      = "smtp-relay.gmail.com:25"
	hostname    = "smtp-relay.gmail.com"
	customEHLO  = "ib.banksampoerna.com"
)

func main() {
	data := "20250701"
	fmt.Println(data[:6])

	endDate := time.Now()
	startDate := endDate.AddDate(0, -12, 0)

	// Format string YYYY-MM-DD
	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	fmt.Println(startDateStr)
	fmt.Println(endDateStr)

	dataS := puebi.SanitizeToPUEBI("Hai Luqman, Anda telah melakukan Transfer Real Time dari rekening 1023613267 sejumlah Rp 12.000. Pastikan transaksi ini benar dilakukan atau Hubungi Call Center 1500 035.")
	fmt.Println(">>>>> " + dataS)

}

func SplitFixedLengthLines(input string, width int) []string {
	var lines []string
	input = strings.TrimSpace(input)

	for i := 0; i < len(input); i += width {
		end := i + width
		if end > len(input) {
			end = len(input)
		}
		segment := strings.TrimSpace(input[i:end])
		if segment != "" {
			segment = strings.Replace(segment, ":", "\n", -1)
			lines = append(lines, segment)
		}
	}
	return lines
}
