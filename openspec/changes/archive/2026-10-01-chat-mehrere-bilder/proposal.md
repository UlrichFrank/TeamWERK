# Proposal

## Why

Eine Chat-Nachricht und eine Mitteilung tragen heute genau ein Bild (`messages.media_id`, `broadcasts.media_id`). Wer nach einem Spieltag fünf Fotos teilen will, muss fünf Nachrichten schicken. Jede davon löst eine eigene Push aus, zählt einzeln als ungelesen und steht als eigener Eintrag im Verlauf. Gewünscht ist ein Album: bis zu 10 Bilder in **einer** Nachricht, mit einer Push und einer Zeile im Verlauf.

## What Changes

- Eine Chat-Nachricht (`POST /api/chat/conversations/{id}/messages`) und eine Mitteilung (`POST /api/chat/broadcasts`) nehmen neu `mediaIds: number[]` mit 1–10 Einträgen. Das bisherige `mediaId` bleibt als Ein-Element-Kurzform gültig; beide zugleich → HTTP 400.
- Jede übergebene media-ID MUSS vom Absender selbst hochgeladen sein (`media.uploaded_by = Absender`) und darf noch an keiner anderen Nachricht oder Mitteilung hängen. Heute prüft der Server nur, ob die ID existiert. Ein Nutzer konnte so eine fremde media-ID an eine eigene Nachricht hängen und sich über `media.canSee` Lesezugriff auf das Bild verschaffen. Das gilt künftig auch für die Kurzform `mediaId`.
- Lese-Antworten (`GET …/messages`, `GET /api/chat/messages/{id}`, `GET /api/chat/broadcasts`) liefern zusätzlich `media: [{id, url, width?, height?}]` in Album-Reihenfolge. `mediaId`/`mediaUrl`/`mediaWidth`/`mediaHeight` bleiben als **erstes** Bild erhalten, damit eine noch gecachte PWA-Version weiter mindestens ein Bild zeigt.
- `GET /api/media/{id}` gibt ein Bild frei, wenn es an einer sichtbaren Nachricht oder Mitteilung hängt, egal an welcher Album-Position. Die bisherigen Regeln (Hochladender, Konversationsmitglied auch nach Austritt, Mitteilungsempfänger, sonst 404) bleiben unverändert.
- Die Push-Vorschau einer reinen Bild-Nachricht lautet „Bild" bzw. „N Bilder".
- Frontend: Mehrfachauswahl im Bild-Picker von Chat-Eingabe und Mitteilungs-Dialog sowie Einfügen per Zwischenablage. Die Vorschauleiste zeigt Miniaturen, jede einzeln entfernbar; die Grenze von 10 Bildern wird sichtbar gemeldet. In der Sprechblase erscheinen die Bilder als Raster, ab dem 5. Bild mit „+N" auf der vierten Kachel. Die Vollbildansicht blättert innerhalb des Albums (Pfeile, Tastatur).
- Bearbeiten bleibt reine Textänderung: ein Album lässt sich nachträglich nicht umsortieren oder ergänzen, wie heute beim Einzelbild.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `chat-konversationen`: Nachrichten tragen bis zu 10 Bilder (Senden mit `mediaIds`, Abruf mit `media[]`, Eigentümerprüfung der media-IDs, Push-Vorschau).
- `chat-broadcasts`: Mitteilungen tragen bis zu 10 Bilder (Senden mit `mediaIds`, Abruf mit `media[]`, Eigentümerprüfung).
- `media-storage`: „Bild abrufen" gibt Bilder über alle Album-Positionen frei.

## Impact

- **DB:** Migration `071`: neue Tabellen `message_media` und `broadcast_media`, jeweils mit Fremdschlüssel, Position und Unique-Index auf `media_id`. Die Bestandsbilder aus `messages.media_id`/`broadcasts.media_id` werden als Position 0 übernommen. Die alten Spalten bleiben bestehen und tragen weiter das erste Bild, unter anderem wegen des `CHECK (length(body) > 0 OR media_id IS NOT NULL)`. Die Migration ist additiv, ein Rollback per `make deploy-rollback` bleibt ohne `migrate down` möglich.
- **Backend:** `internal/chat/handler.go` (`messageSelect`, `ListMessages`, `GetMessage`, `SendMessage`, `ListBroadcasts`, `CreateBroadcast`, Push-Vorschau), `internal/media/handler.go` (`canSee`).
- **Frontend:** `web/src/pages/ChatPage.tsx` (Eingabe, Mitteilungs-Dialog, Sprechblase, Mitteilungs-Detail, Lightbox), voraussichtlich eine neue Komponente `ChatImageGrid`.
- **Ressourcen:** Hochgeladen wird nacheinander, Bild für Bild; jedes wird wie bisher clientseitig auf höchstens 1 MB verkleinert. Der Server sieht pro Request weiter genau eine Datei, der Speicherbedarf auf dem VPS bleibt gleich.
- **Kein neuer externer Dienst, keine neue Route.** Die Berechtigungen ändern sich nicht: wer senden darf, bestimmt weiter Konversationsmitgliedschaft bzw. Mitteilungs-Ziel. Neu ist nur die Bindung der media-ID an den Absender.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `POST /api/chat/conversations/{id}/messages` | `TestSendMessage_AlbumMitDreiBildern` | 201; `message_media` hat 3 Zeilen in Request-Reihenfolge, `messages.media_id` = erstes Bild |
| | `TestSendMessage_ElfBilderAbgelehnt` | 400, keine Nachricht angelegt |
| | `TestSendMessage_FremdeMediaIDAbgelehnt` | 400 bei media-ID eines anderen Nutzers (auch über die Kurzform `mediaId`) |
| | `TestSendMessage_BereitsVerwendeteMediaIDAbgelehnt` | 400, wenn die ID schon an einer Nachricht hängt |
| | `TestSendMessage_MediaIdUndMediaIdsZugleich` | 400 |
| | `TestSendMessage_DoppelteIDImAlbum` | 400 |
| `GET /api/chat/conversations/{id}/messages` | `TestListMessages_AlbumReihenfolgeUndAltfelder` | `media[]` in Positionsreihenfolge, `mediaId` = erstes Bild |
| | `TestListMessages_BestandsbildAlsEinElementAlbum` | per Migration übernommenes Bild erscheint als `media` mit Länge 1 |
| `POST /api/chat/broadcasts` | `TestCreateBroadcast_AlbumMitBildern` / `…_FremdeMediaIDAbgelehnt` | 201 bzw. 400 |
| `GET /api/chat/broadcasts` | `TestListBroadcasts_Album` | `media[]` in Reihenfolge |
| `GET /api/media/{id}` | `TestServe_DrittesAlbumbildFuerKonversationsmitglied` | 200 |
| | `TestServe_AlbumbildFuerFremdeN404` | 404 |
| | `TestServe_AlbumbildFuerMitteilungsempfaenger` | 200 |

**Invarianten:** (1) Ein Bild ist nie für jemanden sichtbar, der die tragende Nachricht oder Mitteilung nicht sehen darf, an welcher Position es auch steht. (2) Eine media-ID hängt an höchstens einem Objekt, und nur an einem des eigenen Hochladenden. (3) `messages.media_id` ist immer gleich dem Bild auf Position 0 bzw. NULL ohne Bild.
