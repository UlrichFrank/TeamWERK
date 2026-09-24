## 1. Backend

- [x] 1.1 Migration `070_match_report_return` (`review_comment`, `returned_at`)
- [x] 1.2 `internal/matchreports/return.go`: Handler `Return` (Rolle → Objekt/State → Kommentar), Broadcast, Autor-Meldung; Route im Freigeber-Tier; Tier- und Objektrechte-Matrix
- [x] 1.3 `get.go`/`list.go`: `review_comment`, `returned_at`, `returned`
- [x] 1.4 Tests `internal/matchreports/return_test.go` gemäß Test-Anforderungen

## 2. Frontend

- [x] 2.1 `MatchReportFormPage`: Knopf „Zurückgeben" + Kommentar-Dialog (speichert vorher), Hinweis mit Kommentar im Entwurf bzw. für den Freigeber (+ Vitest)
- [x] 2.2 `MatchReportListPage`: zurückgegebene Berichte markieren
- [x] 2.3 Benutzerhandbuch: Rückgabe beschreiben
