# Tasks

## 1. Schema

- [x] 1.1 Migration `071_chat_album.up.sql`/`.down.sql` anlegen: `message_media`, `broadcast_media` (FK mit `ON DELETE CASCADE` auf die tragende Zeile, `position` 0–9, PK `(…_id, position)`, Unique-Index auf `media_id`) und Übernahme der Bestandsbilder als Position 0. Die Nummer vorher per `ls internal/db/migrations | sort -V | tail -1` gegenprüfen. Verifikation: neuer Test `internal/db/migration_071_test.go` (Bestandsnachricht und -mitteilung mit Bild erscheinen nach `up` mit Position 0; eine zweite Zuordnung derselben `media_id` scheitert am Unique-Index; das Löschen einer Konversation entfernt ihre `message_media`-Zeilen).
- [x] 1.2 Seeds nachziehen: `cmd/teamwerk` (e2e-seed, dev-seed) schreibt für jede geseedete Bild-Nachricht zusätzlich die `message_media`-Zeile, ebenso `internal/permissions/object_fixtures_test.go` und die Fixtures in `internal/chat/media_message_test.go`. Verifikation: `go test ./cmd/teamwerk/ ./internal/permissions/ ./internal/chat/` grün.

## 2. Backend: Senden

- [x] 2.1 `resolveAlbum` im chat-Package: Kurzform `mediaId` normalisieren, `mediaId`+`mediaIds` zugleich, mehr als 10 Einträge und Duplikate → 400 `invalid_media`; eine Abfrage prüft Existenz, `uploaded_by = Absender` und „noch nirgends zugeordnet" (beide Tabellen). Verifikation: Tabellentest `TestResolveAlbum` (leer, eins, zehn, elf, Duplikat, fremd, schon an Nachricht, schon an Mitteilung, beide Felder).
- [x] 2.2 `SendMessage` auf `resolveAlbum` umstellen; Nachricht (`media_id = ids[0]`) und `message_media`-Zeilen in einer Transaktion schreiben; Push-Vorschau über `imagePreview(n)` („Bild" / „N Bilder"). Verifikation: `TestSendMessage_AlbumMitDreiBildern` (Reihenfolge, Invariante `media_id` = Position 0, genau ein `chat:new-message`), `TestSendMessage_ElfBilderAbgelehnt`, `TestSendMessage_FremdeMediaIDAbgelehnt` (auch Kurzform), `TestSendMessage_BereitsVerwendeteMediaIDAbgelehnt`, `TestSendMessage_MediaIdUndMediaIdsZugleich`, `TestSendMessage_DoppelteIDImAlbum`, `TestImagePreview`.
- [x] 2.3 `CreateBroadcast` analog auf `resolveAlbum` + Transaktion + `broadcast_media` + `imagePreview`. Verifikation: `TestCreateBroadcast_AlbumMitBildern`, `TestCreateBroadcast_FremdeMediaIDAbgelehnt`, `TestCreateBroadcast_ElfBilderAbgelehnt`; die bestehenden Broadcast-Tests bleiben grün.

## 3. Backend: Lesen und Freigabe

- [x] 3.1 `loadMessageMedia(ctx, ids)` (eine Abfrage je Seite) und Feld `media []mediaItem` (`id`, `url`, `width?`, `height?`) in `ListMessages` und `GetMessage`; leer bei gelöschten Nachrichten; Altfelder unverändert aus `messageSelect`. Verifikation: `TestListMessages_AlbumReihenfolgeUndAltfelder`, `TestListMessages_BestandsbildAlsEinElementAlbum`, `TestListMessages_GeloeschteNachrichtOhneMedia`, `TestGetMessage_Album`.
- [x] 3.2 `ListBroadcasts` analog über `broadcast_media`. Verifikation: `TestListBroadcasts_Album`, bestehende Dimensions-Tests grün.
- [x] 3.3 `media.canSee` auf `message_media`/`broadcast_media` umstellen (die Spalten nicht mehr abfragen). Verifikation: `TestServe_DrittesAlbumbildFuerKonversationsmitglied` (200), `TestServe_AlbumbildFuerFremdeN404` (404), `TestServe_AlbumbildFuerMitteilungsempfaenger` (200), `TestServe_AusgetretenesMitgliedSiehtAlbumbild` (200); Objektrechte-Matrix (`go test ./internal/permissions/`) grün.
- [x] 3.4 Backend-Gate: `go vet ./... && go test ./... && golangci-lint run` grün (inkl. Broadcast-, Push-Fan-out- und Goroutine-Gate; keine neue Route, also kein neuer Allowlist-Eintrag).

## 4. Frontend: Anzeige

- [x] 4.1 Typen in `ChatPage.tsx` um `media: { id: number; url: string; width?: number; height?: number }[]` erweitern (Fallback auf die Altfelder, falls `media` fehlt). Verifikation: `pnpm -C web build` (Typecheck).
- [x] 4.2 Komponente `web/src/components/ChatImageGrid.tsx`: 1 Bild = heutiges `AuthImage` mit Dimensionen; 2 / 3 / ≥4 Bilder als Raster mit fester Breite und `aspect-square`-Kacheln (`object-cover`), „+N" auf der vierten Kachel; Klick meldet den Index. Nur `brand-*`-Tokens. Verifikation: `ChatImageGrid.test.tsx` (Kachelzahl je Albumgröße, „+N"-Text, Klick-Index, Kacheln tragen ihre Größe ohne geladene Bilddaten).
- [x] 4.3 Lightbox auf `{ urls, index }` umstellen: Pfeile (`ChevronLeft`/`ChevronRight` mit `aria-label`), ←/→-Tasten, Zähler „k / N"; Sprechblase und Mitteilungs-Detail rendern `ChatImageGrid`. Verifikation: Vitest „Klick auf drittes Bild öffnet Lightbox bei 3/5, Pfeil rechts → 4/5".
- [x] 4.4 Layout-Stabilität im echten Browser: in `web/e2e/chat-scroll.spec.ts` ein Album mit 4 Bildern in die Seed-Konversation aufnehmen (Seed aus 1.2) und den bestehenden Test „Inhaltshöhe bleibt beim Decode stabil" auch für das Album prüfen. Verifikation: `make test-e2e` grün.

## 5. Frontend: Senden

- [x] 5.1 Chat-Eingabe: `pendingImages[]` statt `pendingImage`, `<input multiple>`, Einfügen aus der Zwischenablage übernimmt alle Bilder; Vorschauleiste mit Miniaturen und je einem Entfernen-Button (`X`, `aria-label`); bei mehr als 10 Toast „Höchstens 10 Bilder je Nachricht". Beim Konversationswechsel wird die Auswahl verworfen und alle Object-URLs werden freigegeben. Verifikation: Vitest (Mehrfachauswahl → N Miniaturen; Entfernen; 12 gewählt → 10 + Toast).
- [x] 5.2 Senden: Upload nacheinander mit Fortschritt „Bild k/N" am Senden-Button, dann ein `POST …/messages` mit `mediaIds`; bei Upload-Fehler Abbruch mit erhaltener Auswahl. Verifikation: Vitest (3 Uploads strikt nacheinander, Request-Body `mediaIds` in Auswahlreihenfolge; zweiter Upload scheitert → kein Message-POST, Auswahl bleibt).
- [x] 5.3 Mitteilungs-Dialog analog (Mehrfachauswahl, Vorschauleiste, sequenzieller Upload, `mediaIds`). Verifikation: Vitest für den Dialog (Request-Body).
- [x] 5.4 Frontend-Gate: `pnpm -C web build && pnpm -C web lint && pnpm -C web test` grün (Design-Token-, Typografie- und Button-Gate).

## 6. Dokumentation und Abschluss

- [x] 6.1 `docs/agent/06-gotchas.md` um einen Absatz „Chat-Alben" ergänzen: Zuordnungstabellen als Quelle der Wahrheit, `media_id` = Position 0 nur für CHECK und Altclients, `canSee` liest ausschließlich die Tabellen, Seeds müssen `message_media` mitschreiben, Raster mit fester Geometrie wegen des Platzhalter-Gotchas. Verifikation: Review des Abschnitts.
- [x] 6.2 Prod-Vorabprüfung auf doppelt referenzierte `media_id` (Abfrage aus design.md, Risks) vorbereiten und als Deploy-Hinweis in den PR aufnehmen. Verifikation: Abfrage im PR-Text.
- [x] 6.3 `/verify-change` ausführen und `openspec validate chat-mehrere-bilder --strict` grün.
