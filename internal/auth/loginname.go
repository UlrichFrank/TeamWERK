package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// umlautReplacer transliteriert deutsche Umlaute/ß für den login_name, damit
// der Schlüssel aus reinem ASCII besteht (keine URL-/Tipp-Probleme beim Login).
var umlautReplacer = strings.NewReplacer(
	"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
	"Ä", "Ae", "Ö", "Oe", "Ü", "Ue",
)

// looksLikeEmail prüft minimal, ob s wie eine E-Mail aussieht: genau ein "@"
// mit nicht-leerem lokalen Teil und einer Domain mit Punkt. Bewusst einfach
// (keine RFC-Validierung) — konsistent mit dem sonstigen E-Mail-Handling.
func looksLikeEmail(s string) bool {
	s = strings.TrimSpace(s)
	at := strings.IndexByte(s, '@')
	if at <= 0 || at != strings.LastIndexByte(s, '@') {
		return false
	}
	domain := s[at+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// normalizeNamePart bereitet einen einzelnen Namensteil (Vor- oder Nachname)
// für den login_name auf: Umlaut-Transliteration, Leerzeichen → Bindestrich,
// Reduktion auf [A-Za-z0-9-]. Die Schreibweise (Groß/Klein) bleibt erhalten;
// der Vergleich erfolgt später case-insensitiv.
func normalizeNamePart(s string) string {
	s = umlautReplacer.Replace(strings.TrimSpace(s))
	// Mehrfach-Whitespace kollabieren und in Bindestriche umwandeln.
	s = strings.Join(strings.Fields(s), "-")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		}
	}
	// Führende/abschließende Bindestriche und Doppel-Bindestriche säubern.
	return strings.Trim(collapseHyphens(b.String()), "-")
}

func collapseHyphens(s string) string {
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

// normalizeLoginName bildet aus Vor- und Nachname den Basis-login_name im
// Format "Vorname.Nachname". Liefert einen leeren String, wenn nach der
// Normalisierung kein verwertbarer Vor- oder Nachname übrig bleibt.
func normalizeLoginName(first, last string) string {
	f := normalizeNamePart(first)
	l := normalizeNamePart(last)
	if f == "" || l == "" {
		return ""
	}
	return f + "." + l
}

// loginNameTaken prüft case-insensitiv, ob ein login_name bereits vergeben ist —
// über ALLE Konten hinweg (unabhängig von can_login), damit auch zwei noch
// inaktive Kinder-Accounts (can_login=0) nicht denselben Namen erhalten.
func loginNameTaken(ctx context.Context, q queryRower, name string) (bool, error) {
	var taken bool
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) > 0 FROM users WHERE LOWER(login_name) = LOWER(?)`, name,
	).Scan(&taken)
	return taken, err
}

// queryRower wird von *sql.DB und *sql.Tx erfüllt, damit die Generierung
// sowohl innerhalb als auch außerhalb einer Transaktion nutzbar ist.
type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// maxLoginNameAttempts begrenzt die Kollisions-Suffix-Schleife als Sicherheitsnetz.
const maxLoginNameAttempts = 1000

// generateUniqueLoginName erzeugt aus Vor-/Nachname einen eindeutigen login_name.
// Bei Kollision wird ein numerisches Suffix an den Nachnamen-Teil gehängt
// ("Lena.Schmidt" → "Lena.Schmidt2" → "Lena.Schmidt3" …), bis ein freier Name
// gefunden ist. Gibt einen Fehler zurück, wenn der Name leer ist oder nach
// maxLoginNameAttempts kein freier Name gefunden wurde.
func generateUniqueLoginName(ctx context.Context, q queryRower, first, last string) (string, error) {
	base := normalizeLoginName(first, last)
	if base == "" {
		return "", fmt.Errorf("login name leer nach Normalisierung (first=%q last=%q)", first, last)
	}
	for i := range maxLoginNameAttempts {
		candidate := base
		if i > 0 {
			// Suffix an den Nachnamen-Teil; der Punkt-Trenner bleibt erhalten.
			candidate = fmt.Sprintf("%s%d", base, i+1)
		}
		taken, err := loginNameTaken(ctx, q, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("kein freier login_name für %q nach %d Versuchen", base, maxLoginNameAttempts)
}
