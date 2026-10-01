# chat-konversationen Specification

## Purpose

Diese Spezifikation beschreibt die Capability `chat-konversationen`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Teilnehmer einer Gruppen-Konversation einsehen

Das System SHALL jedem aktiven Mitglied einer Gruppen-Konversation erlauben, die vollständige Teilnehmerliste über ein UI-Element im Chat-Header einzusehen. Die Liste enthält pro aktiver Teilnahme: `id`, `name` und eine Kennzeichnung des Erstellers (`createdBy === user.id`). Bereits ausgetretene Mitglieder (`left_at IS NOT NULL`) erscheinen NICHT.

#### Scenario: Mitglied öffnet Teilnehmerliste

- **WHEN** ein Gruppen-Mitglied auf das Teilnehmer-Icon im Chat-Header klickt
- **THEN** öffnet sich ein Modal mit dem Titel „Teilnehmer" und listet alle aktiven Teilnehmer
- **THEN** ist der Ersteller in der Liste als solcher gekennzeichnet
- **THEN** sieht das Mitglied keinen Bearbeiten-Button, falls es nicht der Ersteller ist

#### Scenario: Ersteller öffnet Teilnehmerliste

- **WHEN** der Ersteller einer Gruppe auf das Teilnehmer-Icon klickt
- **THEN** öffnet sich das Modal im View-Modus mit zusätzlichem Bearbeiten-Button neben dem Schließen-`X`

### Requirement: Mitglied einer Gruppen-Konversation entfernen

Das System SHALL dem Ersteller einer Gruppen-Konversation erlauben, andere Mitglieder per `DELETE /api/chat/conversations/{id}/members/{userId}` zu entfernen. Direct-Konversationen und Self-Removal SIND verboten. Das Entfernen erfolgt als Soft-Delete (`left_at` wird gesetzt) und erzeugt eine Systemnachricht „X wurde entfernt".

#### Scenario: Ersteller entfernt Mitglied

- **WHEN** der Ersteller `DELETE /api/chat/conversations/{id}/members/{userId}` auf ein aktives Mitglied aufruft
- **THEN** wird `conversation_members.left_at` für das Ziel-Mitglied gesetzt
- **THEN** wird eine Systemnachricht „wurde entfernt" mit `sender_id = entferntes Mitglied` eingefügt
- **THEN** antwortet der Server mit HTTP 204
- **THEN** erhalten alle aktiven Mitglieder UND der entfernte User ein SSE-Event `chat:member-left:<conversationId>`

#### Scenario: Nicht-Ersteller versucht zu entfernen

- **WHEN** ein nicht-Ersteller-Mitglied `DELETE /api/chat/conversations/{id}/members/{userId}` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Ersteller versucht sich selbst zu entfernen

- **WHEN** der Ersteller `DELETE /api/chat/conversations/{id}/members/{userId}` mit `userId == claims.UserID` aufruft
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Versuch auf Direct-Konversation

- **WHEN** ein User versucht ein Mitglied aus einer Direct-Konversation zu entfernen
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Gruppen-Konversation umbenennen

Das System SHALL dem Ersteller erlauben, den Namen einer Gruppen-Konversation per `PUT /api/chat/conversations/{id}` mit Body `{ name }` zu ändern. Der Name MUSS zwischen 1 und 100 Zeichen lang sein (nach Trim). Direct-Konversationen können nicht umbenannt werden. Die Änderung erzeugt eine Systemnachricht „hat die Gruppe in 'Y' umbenannt" und broadcastet `chat:conv-updated:<id>`.

#### Scenario: Ersteller benennt um

- **WHEN** der Ersteller `PUT /api/chat/conversations/{id}` mit `{ name: "Taktik" }` aufruft
- **THEN** wird `conversations.name` auf den neuen Wert gesetzt
- **THEN** wird eine Systemnachricht „hat die Gruppe in 'Taktik' umbenannt" mit `sender_id = Ersteller` eingefügt
- **THEN** erhalten alle aktiven Mitglieder ein SSE-Event `chat:conv-updated:<conversationId>`

#### Scenario: Leerer Name wird abgelehnt

- **WHEN** ein Ersteller `PUT /api/chat/conversations/{id}` mit leerem `name` aufruft
- **THEN** antwortet der Server mit HTTP 400
- **THEN** wird die DB nicht verändert

#### Scenario: Nicht-Ersteller versucht umzubenennen

- **WHEN** ein Mitglied (nicht Ersteller) `PUT /api/chat/conversations/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Verwaltung einer Gruppen-Konversation übergeben

Das System SHALL dem Ersteller erlauben, die Verwaltungsrechte per `POST /api/chat/conversations/{id}/transfer-ownership` mit Body `{ newOwnerId }` an ein anderes aktives Mitglied zu übergeben. Der Empfänger MUSS aktives Mitglied (`left_at IS NULL`) sein und DARF NICHT mit dem aktuellen Ersteller identisch sein. Nach Übergabe ist `conversations.created_by` der neue User.

#### Scenario: Ersteller übergibt an aktives Mitglied

- **WHEN** der Ersteller `POST /api/chat/conversations/{id}/transfer-ownership` mit `{ newOwnerId: 42 }` aufruft und User 42 aktives Mitglied ist
- **THEN** wird `conversations.created_by` auf 42 gesetzt
- **THEN** wird eine Systemnachricht „hat die Verwaltung an {neuer Owner Name} übergeben" mit `sender_id = alter Ersteller` eingefügt
- **THEN** erhalten alle aktiven Mitglieder ein SSE-Event `chat:conv-updated:<conversationId>`

#### Scenario: Übergabe an Nicht-Mitglied

- **WHEN** der Ersteller versucht an einen User zu übergeben der kein aktives Mitglied ist
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Self-Übergabe

- **WHEN** der Ersteller `transfer-ownership` mit `newOwnerId == claims.UserID` aufruft
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Nicht-Ersteller versucht zu übergeben

- **WHEN** ein Mitglied (nicht Ersteller) `transfer-ownership` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Gruppen-Konversation für alle Mitglieder löschen

Das System SHALL dem Ersteller erlauben, eine Gruppen-Konversation samt aller Nachrichten endgültig zu entfernen, per `DELETE /api/chat/conversations/{id}/everyone`. Diese Operation ist unwiderruflich und löscht die Datensätze hart (FK-Cascade auf `messages`, `message_reactions`, `message_reads`, `conversation_members`). Direct-Konversationen können hier nicht gelöscht werden.

#### Scenario: Ersteller löscht Gruppe für alle

- **WHEN** der Ersteller `DELETE /api/chat/conversations/{id}/everyone` aufruft
- **THEN** wird die Zeile in `conversations` und per Cascade alle abhängigen Daten gelöscht
- **THEN** erhalten alle vorherigen aktiven Mitglieder ein SSE-Event `chat:conv-deleted:<conversationId>`
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Nicht-Ersteller versucht für-alle-Löschung

- **WHEN** ein Mitglied (nicht Ersteller) `DELETE /api/chat/conversations/{id}/everyone` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Versuch auf Direct-Konversation

- **WHEN** ein User `DELETE /api/chat/conversations/{id}/everyone` auf eine Direct-Konversation aufruft
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Ersteller-Exit erfordert Übergabe oder Löschung

Das Frontend SHALL beim Klick des Erstellers auf „Gruppe verlassen" ein Auswahl-Modal anzeigen, in dem zwischen „Verwaltung übergeben an…" und „Gruppe für alle löschen" gewählt werden muss. Ein direkter Self-Leave-Pfad steht dem Ersteller im UI NICHT zur Verfügung. Diese Beschränkung ist UI-seitig; das Backend lehnt einen direkten `DELETE /members/me`-Aufruf des Erstellers NICHT serverseitig ab.

#### Scenario: Ersteller-Wahl: Übergeben

- **WHEN** der Ersteller im Auswahl-Modal „Verwaltung übergeben an…" mit Mitglied B wählt und bestätigt
- **THEN** ruft das Frontend nacheinander `POST /transfer-ownership` und `DELETE /members/me` auf
- **THEN** ist die Gruppe nach beiden Calls verwaltbar von B und der alte Ersteller ist kein Mitglied mehr

#### Scenario: Ersteller-Wahl: Für alle löschen

- **WHEN** der Ersteller im Auswahl-Modal „Gruppe für alle löschen" wählt
- **THEN** zeigt das Frontend einen zweiten Confirm-Step („Diese Aktion löscht alle Nachrichten endgültig.")
- **WHEN** der Ersteller den Confirm-Step bestätigt
- **THEN** ruft das Frontend `DELETE /chat/conversations/{id}/everyone` auf

### Requirement: Systemnachrichten-Konsistenz für Gruppen-Mutationen

Das System SHALL für jede Mutations-Aktion an einer Gruppen-Konversation eine `is_system=1`-Nachricht in `messages` einfügen, damit alle Verlaufs-Aktionen für nachträgliche Mitglieder sichtbar bleiben.

| Aktion | Body | sender_id |
|---|---|---|
| `AddMember` | `wurde hinzugefügt` | hinzugefügter User |
| `RemoveMember` | `wurde entfernt` | entfernter User |
| `LeaveConversation` | `hat die Gruppe verlassen` | leaving User (bestehend) |
| `UpdateConversation` (rename) | `hat die Gruppe in "Y" umbenannt` | Ersteller |
| `TransferOwnership` | `hat die Verwaltung an {Name} übergeben` | alter Ersteller |

#### Scenario: AddMember erzeugt Systemnachricht

- **WHEN** der Ersteller `POST /chat/conversations/{id}/members` aufruft
- **THEN** wird zusätzlich zum Member-Insert/Update eine Systemnachricht „wurde hinzugefügt" mit `sender_id = hinzugefügter User` eingefügt

### Requirement: SSE-Events für Konversations-Updates

Das System SHALL für Mutations-Aktionen an einer Gruppen-Konversation jenseits der reinen Mitgliederliste die SSE-Events `chat:conv-updated:<id>` (für Rename und Transfer) bzw. `chat:conv-deleted:<id>` (für Löschen-für-alle) emittieren. Das Frontend SHALL bei `chat:conv-updated` die einzelne Konversation aus `GET /chat/conversations` neu laden und bei `chat:conv-deleted` die Konversation aus der Liste entfernen.

#### Scenario: Frontend reagiert auf Conv-Updated

- **WHEN** ein Client das SSE-Event `chat:conv-updated:<id>` empfängt während die Konversation in der Liste sichtbar ist
- **THEN** lädt der Client die Konversation neu und zeigt den aktualisierten Namen, Ersteller und Mitgliederbestand

#### Scenario: Frontend reagiert auf Conv-Deleted bei aktiver Konversation

- **WHEN** ein Client das SSE-Event `chat:conv-deleted:<id>` empfängt und genau diese Konversation gerade aktiv geöffnet hat
- **THEN** schließt der Client die Konversations-Ansicht, zeigt einen Toast „Die Gruppe wurde gelöscht" und entfernt die Konversation aus der Liste

#### Scenario: Frontend reagiert auf Conv-Deleted bei inaktiver Konversation

- **WHEN** ein Client das SSE-Event `chat:conv-deleted:<id>` empfängt und die Konversation nur in der Liste, aber nicht aktiv geöffnet hat
- **THEN** entfernt der Client die Konversation aus der Liste ohne Toast

### Requirement: Konversationsliste nach letzter Aktivität sortiert

`GET /api/chat/conversations` SHALL die Konversationen des anfragenden Nutzers absteigend nach dem Zeitpunkt der letzten Aktivität zurückgeben — die zuletzt aktive Konversation zuerst. Die letzte Aktivität MUST der `sent_at`-Zeitpunkt der jüngsten Nachricht der Konversation sein; für Konversationen ohne Nachricht MUST als Sortierschlüssel `conversations.created_at` verwendet werden.

Diese Anforderung formalisiert bestehendes Verhalten und sichert es gegen Regression; sie ändert das Verhalten nicht.

#### Scenario: Neue Nachricht hebt Konversation an die Spitze

- **WHEN** in einer weiter unten stehenden Konversation eine neue Nachricht eintrifft und die Liste erneut geladen wird
- **THEN** steht diese Konversation an erster Stelle der zurückgegebenen Liste

#### Scenario: Konversation ohne Nachrichten wird nach Erstellzeit einsortiert

- **WHEN** eine Konversation noch keine Nachricht enthält
- **THEN** wird sie anhand von `created_at` in die nach letzter Aktivität absteigend sortierte Liste einsortiert

### Requirement: Teamübergreifender Kontakt im Zugriffskreis

Das System SHALL zwei Mitgliedern des **Zugriffskreises** erlauben, sich gegenseitig zu kontaktieren — sowohl per Direktnachricht (`POST /api/chat/conversations` mit `type=direct`) als auch als Teilnehmer beim Gruppenaufbau (`type=group`) — auch ohne gemeinsames Team. Der Zugriffskreis ist definiert als: User, die (a) Trainer eines Kaders der aktiven Saison sind (`kader_trainers`), ODER Vereinsfunktion (b) `vorstand`, (c) `sportliche_leitung` ODER (d) `vorstand_beisitzer` haben; `admin` stets berechtigt.

Die Kontaktprüfung (`canContactUser`) SHALL in dieser Reihenfolge auswerten: (1) Caller hat **vereinsweite Reichweite** — Rolle `admin` ODER Vereinsfunktion `vorstand` ODER `sportliche_leitung` → erlaubt; (2) Caller UND Ziel sind beide im Zugriffskreis → erlaubt; (3) Caller und Ziel teilen ein Team (`user_accessible_teams`) → erlaubt; (4) sonst HTTP 403. Die Regeln (2) und (3) bleiben unverändert.

Die Menge aus Schritt (1) ist dieselbe, die in `chat-team-groups` die Standardgruppen **aller** Teams der aktiven Saison sieht. Beides MUSS zusammenfallen: wer eine fremde Kader-Gruppe auflösen darf, muss die aufgelösten Mitglieder auch anschreiben dürfen — sonst scheitert der Gruppenaufbau an Schritt (4) für jedes teamfremde Mitglied, obwohl die Gruppe im Modal angeboten wurde.

#### Scenario: Trainer schreibt teamfremden Trainer 1:1 an

- **WHEN** ein Kader-Trainer von T1 `POST /api/chat/conversations` mit `{ type: "direct", userId: <Trainer von T2> }` aufruft und kein gemeinsames Team besteht
- **THEN** wird die Direktkonversation erstellt (HTTP 201/200)

#### Scenario: Sportliche Leitung schreibt teamfremden Trainer an

- **WHEN** ein User mit `sportliche_leitung` einen Trainer eines Teams, in dem er nicht eingetragen ist, per Direktnachricht kontaktiert
- **THEN** wird die Konversation erstellt

#### Scenario: Sportliche Leitung legt Gruppe aus fremder Kader-Gruppe an

- **WHEN** ein User mit `sportliche_leitung` ohne eigene Teamzugehörigkeit `GET /api/chat/team-groups/{fremdesTeam}/spieler/members` auflöst und die zurückgegebenen IDs als `memberIds` an `POST /api/chat/conversations` mit `type=group` schickt
- **THEN** passieren alle Mitglieder die `canContactUser`-Prüfung und die Gruppe wird erstellt (HTTP 201)

#### Scenario: „Alle Trainer"-Gruppe anlegen ist erlaubt

- **WHEN** ein Zugriffskreis-Mitglied `POST /api/chat/conversations` mit `type=group` und den aus „Alle Trainer" aufgelösten Mitgliedern aufruft
- **THEN** passieren alle Mitglieder die `canContactUser`-Prüfung und die Gruppe wird erstellt

#### Scenario: Spieler kann teamfremden Trainer nicht kontaktieren

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit einen Trainer eines fremden Teams per Direktnachricht kontaktieren will
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Spieler kann teamfremden Spieler nicht in eine Gruppe nehmen

- **WHEN** ein Spieler von T1 `POST /api/chat/conversations` mit `type=group` und `memberIds=[<Spieler von T2>]` aufruft
- **THEN** antwortet der Server mit HTTP 403 und es entsteht keine Konversation

### Requirement: Nutzersuche findet Zugriffskreis teamübergreifend

Das System SHALL in `GET /api/chat/users` einem Caller, der im Zugriffskreis ist, zusätzlich zu Usern mit gemeinsamem Team **alle anderen Zugriffskreis-Mitglieder** als Suchtreffer liefern (Dedup nach `user_id`, Namens-/E-Mail-Filter `q` und `LIMIT 50` bleiben bestehen). Für Caller mit **vereinsweiter Reichweite** (`admin`, `vorstand`, `sportliche_leitung`) sucht der Endpoint über alle User; für Caller außerhalb des Zugriffskreises bleibt die Suche auf gemeinsame Teams beschränkt.

Die Reichweite der Suche MUSS mit Schritt (1) der Kontaktprüfung übereinstimmen — sonst erscheint ein teamfremdes Mitglied als Gruppen-Chip, ist über die Suche daneben aber nicht auffindbar.

#### Scenario: Trainer findet teamfremden Trainer

- **WHEN** ein Kader-Trainer von T1 `GET /api/chat/users?q=<Name eines Trainers von T2>` aufruft
- **THEN** enthält das Ergebnis den Trainer von T2, obwohl kein gemeinsames Team besteht

#### Scenario: Sportliche Leitung findet teamfremden Spieler

- **WHEN** ein User mit `sportliche_leitung` ohne eigene Teamzugehörigkeit `GET /api/chat/users` aufruft
- **THEN** enthält das Ergebnis auch Spieler von Teams, in denen er nicht eingetragen ist

#### Scenario: Spieler findet teamfremden Trainer nicht

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit nach einem Trainer eines fremden Teams sucht
- **THEN** ist dieser nicht im Ergebnis enthalten

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

