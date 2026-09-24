## Why

Ein Freigeber (Medien/Vorstand) konnte einen eingereichten Spielbericht bisher nur selbst
korrigieren, veröffentlichen oder löschen. Braucht der Bericht inhaltliche Arbeit, die nur
der Autor leisten kann (fehlende Szenen, falsche Namen, Foto-Freigaben klären), gab es keinen
Weg zurück — `spielbericht-medien-gate` hatte das in design.md D-3 bewusst ausgeschlossen
(„kein Rückweg"). In der Praxis endete das in Nachrichten außerhalb der App oder im Löschen
und Neuschreiben des Berichts.

## What Changes

- **BREAKING (Spec):** Das Requirement „State `pending_review` als Review-Gate" verliert die
  Aussage „kein Rückweg". Neu gibt es genau einen Übergang `pending_review → draft`: die
  **Rückgabe** durch einen Freigeber.
- Neue Route `POST /api/match-reports/{id}/return` `{comment}` (Freigeber-Tier
  `medien`/`vorstand`, Admin). Kommentar ist Pflicht (1–2000 Zeichen, getrimmt).
- Migration `070`: `match_reports.review_comment`, `match_reports.returned_at` (additiv).
  Kein neuer State — „zurückgegeben" ist ein `draft` mit gesetztem `review_comment`.
- Der Autor bekommt eine Benachrichtigung (Kategorie `operativ`, Push + präferenzgesteuerte
  Mail, Event-Log) mit dem Kommentar und darf den Bericht wieder bearbeiten und erneut
  einreichen.
- `GET /api/match-reports/{id}` liefert `review_comment`/`returned_at`,
  `GET /api/match-reports/my` je Bericht `returned`.
- Frontend: Knopf „Zurückgeben" mit Kommentar-Dialog für Freigeber; Hinweis mit Kommentar im
  zurückgegebenen Entwurf; Liste „Meine Berichte" markiert zurückgegebene Berichte.

## Impact

- `internal/matchreports` (`return.go`, `get.go`, `list.go`), `internal/app/router.go`,
  Migration `070`, Permission-Matrizen.
- `web/src/pages/MatchReportFormPage.tsx`, `MatchReportListPage.tsx`, Benutzerhandbuch.

## Test-Anforderungen

| Route | Test | Erwartet |
|---|---|---|
| `POST /api/match-reports/{id}/return` | `TestReturn_HappyPath_AutorDarfWiederBearbeiten` | 200, State `draft`, Autor darf `PUT` + erneut einreichen, Kommentar bleibt sichtbar |
| | `TestReturn_BenachrichtigtAutor` | Event-Log-Zeile mit Kommentar beim Autor |
| | `TestReturn_OhneKommentar_400` | 400 `comment_required`/`comment_too_long`, State unverändert |
| | `TestReturn_AutorOhneFreigeberRecht_403` | 403 |
| | `TestReturn_FalscherState_409` | 409 `not_pending_review` |
| | `TestReturn_Unbekannt_404` | 404 |

Invariante: Nur `pending_review` kann zurückgegeben werden; eine abgelehnte Rückgabe ändert
nichts am Bericht.
