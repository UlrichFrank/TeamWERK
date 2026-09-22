# Proposal

## Why

Ein Spielbericht kann heute nur als `draft` gelöscht werden. Sobald er zur Prüfung
eingereicht wurde (`pending_review`), gibt es keinen Weg mehr, ihn loszuwerden — die
dokumentierte Design-Entscheidung „kein Rückweg, kein Reject, kein Withdraw"
(`spielbericht-medien-gate`) hat das bewusst ausgeschlossen. In der Praxis heißt das:
ein Freigeber, der einen eingereichten Bericht als unbrauchbar einstuft (falscher
Spielbezug, ungeeigneter Inhalt, doppelt angelegt, Autor storniert informell), hat
keine andere Option als ihn dauerhaft in der Prüf-Warteschlange liegen zu lassen —
er zählt weiter im Reminder-Job mit und bleibt in `GET /pending` sichtbar, obwohl nie
veröffentlicht werden soll.

## What Changes

- `DELETE /api/match-reports/{id}` akzeptiert zusätzlich zu `draft`/`publish_failed`
  auch den State `pending_review`.
- Für `pending_review` gilt eine eigene Berechtigung: **nur Freigeber**
  (Vereinsfunktion `medien` ODER `vorstand` ODER Rolle `admin`) dürfen löschen — der
  Autor selbst nicht (er hat mit dem Submit die Verfügung über den Bericht abgegeben,
  siehe `spielbericht-medien-gate`). Für `draft`/`publish_failed` bleibt das
  bestehende Autor-oder-Admin-Recht unverändert.
- Bilder + Bild-Dateien werden beim Löschen eines `pending_review`-Berichts genauso
  aufgeräumt wie bisher bei `draft` (`removeAllImageFiles`).
- Frontend (`MatchReportFormPage.tsx`): Freigeber sehen im State `pending_review`
  einen „Bericht löschen"-Button (Bestätigungsdialog) neben dem bestehenden
  Publish-Button. Kein Löschen-Button in der Listenübersicht
  (`MatchReportListPage.tsx`) — nur auf der Detailseite.
- **BREAKING (Verhaltensänderung, keine API-Signatur)**: Die bisherige Zusage „kein
  Rückweg aus `pending_review`" wird präzisiert — es gibt weiterhin keinen Übergang
  zurück zu `draft` und kein Reject-an-den-Autor, aber neu einen endgültigen Abbruch
  per Löschen durch Freigeber. Dokumentation und State-Machine-Kommentare in
  `internal/matchreports/handler.go` werden entsprechend angepasst.

## Capabilities

### New Capabilities
(keine)

### Modified Capabilities
- `match-reports`: neues Requirement „Löschen im State `pending_review` durch
  Freigeber" (Ergänzung zum bestehenden Requirement „State `pending_review` als
  Review-Gate", das „kein Rückweg" bislang ohne Lösch-Ausnahme formuliert).

## Impact

- **Backend:** `internal/matchreports/create.go` (`Delete`-Handler), Doku-Kommentare
  in `internal/matchreports/handler.go` (Package-Doc + State-Konstanten-Kommentar).
- **Tests:** `internal/matchreports/handler_test.go` bzw. `medien_gate_test.go`
  (Happy-Path Freigeber löscht `pending_review`; Fehlerfall Autor ohne
  Freigeber-Funktion → 403; `published` bleibt 409).
- **Frontend:** `web/src/pages/MatchReportFormPage.tsx` (neuer Löschen-Button für
  Freigeber im State `pending_review`).
- **Spec:** `openspec/specs/match-reports/spec.md`, Requirement „State
  `pending_review` als Review-Gate" (Präzisierung „kein Rückweg" + neues
  Delete-Requirement).
- Keine Migration, keine neue Route, kein neuer externer Dienst.
