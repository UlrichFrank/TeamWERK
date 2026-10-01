## MODIFIED Requirements

### Requirement: Nachrichten einer Konversation abrufen

Das System SHALL die letzten 100 Nachrichten einer Konversation zurückgeben (absteigend nach `sent_at`, im Frontend umgekehrt angezeigt). Zu jeder Nachricht werden geliefert: `id`, `senderId`, `senderName`, `body`/`preview` (leer wenn gelöscht oder reine Bildnachricht), `media` (Liste der Bilder in Album-Reihenfolge, leer wenn kein Bild oder Nachricht gelöscht; je Eintrag `id`, `url = "/media/<id>"` ohne `/api`-Prefix, `width`/`height` nur bei bekannter Dimension, sonst weggelassen), `mediaId`/`mediaUrl`/`mediaWidth`/`mediaHeight` (Angaben zum **ersten** Bild des Albums, Kompatibilität für ältere Clients; null bzw. weggelassen ohne Bild), `sentAt`, `replyToId`, `replyToBody`, `replyToSenderName`, `editedAt`, `deletedAt`, `isSystem`, `reactions`. Dieselbe Bild-Darstellung gilt für den Einzelabruf `GET /api/chat/messages/{id}`.

#### Scenario: Mitglied ruft Nachrichten ab

- **WHEN** ein Mitglied `GET /api/chat/conversations/{id}/messages` aufruft
- **THEN** gibt der Server bis zu 100 Nachrichten zurück, jeweils inkl. `media`, `mediaId` und `mediaUrl`

#### Scenario: Nachricht mit mehreren Bildern

- **WHEN** eine Nachricht mit drei Bildern A, B, C (in dieser Reihenfolge gesendet) abgerufen wird
- **THEN** enthält `media` genau drei Einträge in der Reihenfolge A, B, C, und `mediaId`/`mediaUrl` bezeichnen A

#### Scenario: Nachricht mit Bild und bekannten Dimensionen

- **WHEN** eine Nachricht mit einem Bild abgerufen wird, dessen `media`-Zeile `width=1200`, `height=800` hat
- **THEN** enthält der `media`-Eintrag `width=1200`, `height=800`, und das Nachrichtenobjekt `mediaId`, `mediaUrl = "/media/<mediaId>"`, `mediaWidth=1200`, `mediaHeight=800`

#### Scenario: Nachricht mit Bild ohne bekannte Dimensionen (Bestand vor Backfill oder unlesbarer Header)

- **WHEN** eine Nachricht mit einem Bild abgerufen wird, dessen `media`-Zeile `width IS NULL` hat
- **THEN** fehlen `width`/`height` im `media`-Eintrag sowie `mediaWidth`/`mediaHeight` im Nachrichtenobjekt; `id`/`url` bzw. `mediaId`/`mediaUrl` sind gesetzt

#### Scenario: Bestandsnachricht mit Einzelbild

- **WHEN** eine vor Einführung der Alben gesendete Nachricht mit einem Bild abgerufen wird
- **THEN** enthält `media` genau diesen einen Eintrag

#### Scenario: Nachricht ohne Bild

- **WHEN** eine Nachricht ohne Bild abgerufen wird
- **THEN** ist `media` eine leere Liste, `mediaId` und `mediaUrl` sind null; `mediaWidth`/`mediaHeight` fehlen

#### Scenario: Gelöschte Nachricht mit Bildern

- **WHEN** eine gelöschte Nachricht abgerufen wird, die Bilder trug
- **THEN** ist `media` eine leere Liste

#### Scenario: Nicht-Mitglied wird abgewiesen

- **WHEN** ein User der nicht Mitglied der Konversation ist die Nachrichten abruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Nachricht senden

Das System SHALL das Senden einer Nachricht erlauben. Der Request kann optional `replyToId` sowie Bilder enthalten. Bilder werden als `mediaIds` (Liste mit 1 bis 10 media-IDs, Reihenfolge = Album-Reihenfolge) oder als `mediaId` (Kurzform für genau ein Bild) übergeben; beide Felder zugleich MUST der Server mit HTTP 400 ablehnen. Mindestens `body` (nicht leer) **oder** ein Bild MUSS vorhanden sein. Jede übergebene media-ID MUSS auf eine existierende `media`-Zeile verweisen, die der Absender selbst hochgeladen hat und die noch keiner anderen Nachricht und keiner Mitteilung zugeordnet ist; dieselbe ID darf in einem Request nicht doppelt vorkommen. Verletzt der Request eine dieser Bedingungen oder enthält er mehr als 10 Bilder, MUST der Server mit HTTP 400 antworten und weder Nachricht noch Bildzuordnung speichern. Die referenzierte Nachricht bei `replyToId` MUSS zur selben Konversation gehören. Nach erfolgreichem Speichern SHALL der Server via SSE alle aktiven Mitglieder benachrichtigen und **eine** Push je Offline-Mitglied senden, auch bei mehreren Bildern. Die Push-Vorschau einer Nachricht ohne Text lautet „Bild" bei einem und „N Bilder" bei N > 1 Bildern.

#### Scenario: Textnachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ "body": "Hallo!" }` aufruft
- **THEN** wird die Nachricht gespeichert, HTTP 201 zurückgegeben und ein SSE-Event `chat:new-message:<id>` verteilt

#### Scenario: Reine Bildnachricht erfolgreich gesendet

- **WHEN** ein Mitglied `POST /api/chat/conversations/{id}/messages` mit `{ "body": "", "mediaId": <id> }` aufruft und das Bild selbst hochgeladen hat
- **THEN** wird die Nachricht mit diesem einen Bild und leerem Body gespeichert und HTTP 201 zurückgegeben

#### Scenario: Album mit mehreren Bildern gesendet

- **WHEN** ein Mitglied `{ "body": "Spieltag", "mediaIds": [a, b, c] }` sendet und alle drei Bilder selbst hochgeladen hat
- **THEN** wird **eine** Nachricht mit drei Bildern in der Reihenfolge a, b, c gespeichert, HTTP 201 zurückgegeben, genau ein SSE-Event `chat:new-message:<id>` verteilt und je Offline-Mitglied genau eine Push gesendet

#### Scenario: Push-Vorschau eines reinen Albums

- **WHEN** ein Mitglied `{ "body": "", "mediaIds": [a, b, c] }` sendet
- **THEN** lautet die Push-Vorschau „3 Bilder"

#### Scenario: Bild mit Text kombiniert

- **WHEN** ein Mitglied `{ "body": "Schaut mal", "mediaId": <id> }` sendet
- **THEN** wird eine Nachricht mit `body` und diesem Bild gespeichert

#### Scenario: Mehr als zehn Bilder werden abgelehnt

- **WHEN** ein Mitglied eine Nachricht mit 11 media-IDs sendet
- **THEN** antwortet der Server mit HTTP 400 und speichert keine Nachricht

#### Scenario: Fremdes Bild wird abgelehnt

- **WHEN** ein Mitglied eine media-ID sendet, die ein anderer Nutzer hochgeladen hat (über `mediaIds` oder `mediaId`)
- **THEN** antwortet der Server mit HTTP 400 und speichert keine Nachricht

#### Scenario: Bereits verwendetes Bild wird abgelehnt

- **WHEN** ein Mitglied eine eigene media-ID sendet, die schon einer Nachricht oder Mitteilung zugeordnet ist
- **THEN** antwortet der Server mit HTTP 400 und speichert keine Nachricht

#### Scenario: Doppelte media-ID im Album wird abgelehnt

- **WHEN** ein Mitglied `{ "mediaIds": [a, a] }` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: mediaId und mediaIds zugleich werden abgelehnt

- **WHEN** ein Mitglied `{ "mediaId": a, "mediaIds": [b] }` sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Leere Nachricht ohne Bild wird abgelehnt

- **WHEN** ein User eine Nachricht mit leerem `body` und ohne Bild sendet (auch mit `mediaIds: []`)
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Ausgetretenes Mitglied kann nicht senden

- **WHEN** ein User der die Gruppe verlassen hat eine Nachricht sendet
- **THEN** antwortet der Server mit HTTP 403
