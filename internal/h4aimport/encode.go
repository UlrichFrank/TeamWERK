package h4aimport

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// xajax 0.5 verlangt pro Wert ein Typ-Präfix (hier immer "S" = String). Werte mit
// XML-Sonderzeichen müssen zusätzlich in CDATA gekapselt werden, sonst bricht der
// Server-seitige Parser (daran scheitern naive Nachbauten). Siehe design.md §1.1.
const xajaxSpecialChars = ";<>&["

// EncodeValue kodiert einen einzelnen xajax-Argumentwert:
//   - leer            → "S"
//   - Sonderzeichen   → "S<![CDATA[<v>]]>"
//   - sonst           → "S<v>"
func EncodeValue(v string) string {
	if v == "" {
		return "S"
	}
	if strings.ContainsAny(v, xajaxSpecialChars) {
		return "S<![CDATA[" + v + "]]>"
	}
	return "S" + v
}

// kvPair ist ein <e><k>…</k><v>…</v></e>-Paar innerhalb eines <xjxobj>.
type kvPair struct {
	key string
	val string
}

// encodeObj baut ein <xjxobj>…</xjxobj> aus den (bereits fachlichen, unkodierten) Werten.
func encodeObj(pairs []kvPair) string {
	var b strings.Builder
	b.WriteString("<xjxobj>")
	for _, p := range pairs {
		b.WriteString("<e><k>")
		b.WriteString(p.key)
		b.WriteString("</k><v>")
		b.WriteString(EncodeValue(p.val))
		b.WriteString("</v></e>")
	}
	b.WriteString("</xjxobj>")
	return b.String()
}

// BuildXajaxArgs baut die zwei xjxargs[]-Objekte für einen Periodenabruf mit dem
// Filter „nur eigene Beteiligung" (opOwnGames=on). periodID ist z.B. "142".
// Rückgabe sind die zwei rohen <xjxobj>-Strings (ohne "xjxargs[]="-Präfix,
// ohne URL-Kodierung) — verifizierter Ziel-Payload aus design.md §1.1.
func BuildXajaxArgs(periodID string) (arg1, arg2 string) {
	arg1 = encodeObj([]kvPair{
		{"ge_statsel", "0"},
		{"ge_dasel", "all;all"},
		{"ge_gameno", ""},
		{"sbdasel", "Los"},
		{"ge_periods", periodID},
	})
	arg2 = encodeObj([]kvPair{
		{"dummy", "1"},
		{"opOwnGames", "on"},
	})
	return arg1, arg2
}

// BuildGamesFormBody baut den vollständigen, URL-kodierten POST-Body für den
// Spielabruf gegen /games/edit.php. unixMs ist der xajax-Cache-Buster (unix-ms).
func BuildGamesFormBody(periodID string, unixMs int64) string {
	a1, a2 := BuildXajaxArgs(periodID)
	var b strings.Builder
	b.WriteString("xjxfun=xajax_update")
	b.WriteString("&xjxr=")
	b.WriteString(strconv.FormatInt(unixMs, 10))
	b.WriteString("&xjxargs[]=")
	b.WriteString(url.QueryEscape(a1))
	b.WriteString("&xjxargs[]=")
	b.WriteString(url.QueryEscape(a2))
	return b.String()
}

// nowUnixMilli ist ausgelagert für Testbarkeit.
func nowUnixMilli() int64 { return time.Now().UnixMilli() }
