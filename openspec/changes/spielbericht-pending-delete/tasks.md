# Tasks

## 1. Backend

- [x] 1.1 `Delete`-Handler (`internal/matchreports/create.go`) um State `pending_review` erweitern: State-Check erlaubt `draft`, `publish_failed`, `pending_review`; für `pending_review` Berechtigung über `isReviewer(claims)` statt Autor-Check, für `draft`/`publish_failed` bestehende Autor-oder-Admin-Logik unverändert lassen. Verifikation: `go build ./...` erfolgreich.
- [x] 1.2 Doku-Kommentare in `internal/matchreports/handler.go` (Package-Doc-State-Machine + State-Konstanten-Kommentar) anpassen: „kein Rückweg" auf „kein Rückweg zum Autor" präzisieren, Lösch-Ausnahme erwähnen. Verifikation: Kommentartext gelesen, kein Code-Effekt.
- [x] 1.3 Test: Freigeber (`medien`) löscht `pending_review`-Bericht → HTTP 204, Bericht und Bilder sind aus DB/Filesystem entfernt. Verifikation: `go test ./internal/matchreports/...` grün.
- [x] 1.4 Test: Freigeber (`vorstand`) löscht `pending_review`-Bericht → HTTP 204. Verifikation: `go test ./internal/matchreports/...` grün.
- [x] 1.5 Test: Autor ohne Freigeber-Funktion versucht `DELETE` auf eigenen `pending_review`-Bericht → HTTP 403. Verifikation: `go test ./internal/matchreports/...` grün.
- [x] 1.6 Test: `DELETE` auf `published`-Bericht bleibt HTTP 409 (`already_published`) — Regressionstest für bestehendes Verhalten. Verifikation: `go test ./internal/matchreports/...` grün.

## 2. Frontend

- [x] 2.1 `MatchReportFormPage.tsx`: neue Funktion `deletePendingReview` (analog `deleteDraft`, eigener Bestätigungstext „Eingereichten Bericht endgültig löschen? Der Autor muss ggf. neu beginnen."), navigiert nach Erfolg zu `/termine`. Verifikation: `pnpm -C web build` erfolgreich.
- [x] 2.2 Löschen-Button im `canEdit`-Block ergänzt, sichtbar wenn `report.state === 'pending_review' && isReviewer` (neben Publish-Button, `BTN_DANGER`, `Trash2`-Icon). Verifikation: `pnpm -C web build` + `tsc --noEmit` grün; Bedingung `isReviewer && report.state === 'pending_review'` spiegelt exakt die Backend-Autorisierung (isReviewer = admin/medien/vorstand), Autor-only sieht den Button nicht, da `canEdit` in diesem State nur für `isReviewer` true ist.

## 3. Spec-Abgleich

- [x] 3.1 `openspec validate spielbericht-pending-delete --strict` läuft fehlerfrei durch (Delta-Format, Scenario-Header-Anzahl). Verifikation: Exit-Code 0.

## 4. Gate

- [x] 4.1 `make test` (inkl. Architektur-/Broadcast-/Objektrechte-Gates) läuft grün. Verifikation: Exit-Code 0.
