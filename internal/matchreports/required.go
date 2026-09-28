package matchreports

import (
	"database/sql"
	"strings"
)

// missingRequiredFields liefert die Pflichtfelder, die der TYPO3-Import
// (MatchReportImportMiddleware::missingRequiredFields) ablehnen würde und die
// der Bearbeitende selbst füllen kann: Abstract und Berichtstext. Der Titel
// fehlt bewusst — für ihn gibt es den Fallback aus dem Gegnernamen
// (assemblePublishRequest).
//
// Geprüft wird VOR jedem Zustandswechsel. Früher lief ein leeres Abstract bis
// zum Publisher durch und kam als `publish_failed` mit der rohen Antwort
// „missing_fields: abstract“ zurück — für die Freigeberin unverständlich und
// für den Autor, der das Feld hätte füllen müssen, nicht mehr bearbeitbar.
func missingRequiredFields(db *sql.DB, reportID int) ([]string, error) {
	var abstract, bodyMD string
	if err := db.QueryRow(
		`SELECT COALESCE(abstract, ''), COALESCE(body_md, '') FROM match_reports WHERE id=?`, reportID,
	).Scan(&abstract, &bodyMD); err != nil {
		return nil, err
	}
	var missing []string
	if strings.TrimSpace(abstract) == "" {
		missing = append(missing, "abstract")
	}
	if strings.TrimSpace(bodyMD) == "" {
		missing = append(missing, "body_md")
	}
	return missing, nil
}
