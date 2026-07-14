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

Die Kontaktprüfung (`canContactUser`) SHALL in dieser Reihenfolge auswerten: (1) Caller ist `admin` oder `vorstand` → erlaubt; (2) Caller UND Ziel sind beide im Zugriffskreis → erlaubt; (3) Caller und Ziel teilen ein Team (`user_accessible_teams`) → erlaubt; (4) sonst HTTP 403. Die bestehenden Regeln (1) und (3) bleiben unverändert.

#### Scenario: Trainer schreibt teamfremden Trainer 1:1 an

- **WHEN** ein Kader-Trainer von T1 `POST /api/chat/conversations` mit `{ type: "direct", userId: <Trainer von T2> }` aufruft und kein gemeinsames Team besteht
- **THEN** wird die Direktkonversation erstellt (HTTP 201/200)

#### Scenario: Sportliche Leitung schreibt teamfremden Trainer an

- **WHEN** ein User mit `sportliche_leitung` einen Trainer eines Teams, in dem er nicht eingetragen ist, per Direktnachricht kontaktiert
- **THEN** wird die Konversation erstellt

#### Scenario: „Alle Trainer"-Gruppe anlegen ist erlaubt

- **WHEN** ein Zugriffskreis-Mitglied `POST /api/chat/conversations` mit `type=group` und den aus „Alle Trainer" aufgelösten Mitgliedern aufruft
- **THEN** passieren alle Mitglieder die `canContactUser`-Prüfung und die Gruppe wird erstellt

#### Scenario: Spieler kann teamfremden Trainer nicht kontaktieren

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit einen Trainer eines fremden Teams per Direktnachricht kontaktieren will
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Nutzersuche findet Zugriffskreis teamübergreifend

Das System SHALL in `GET /api/chat/users` einem Caller, der im Zugriffskreis ist, zusätzlich zu Usern mit gemeinsamem Team **alle anderen Zugriffskreis-Mitglieder** als Suchtreffer liefern (Dedup nach `user_id`, Namens-/E-Mail-Filter `q` und `LIMIT 50` bleiben bestehen). Für `admin`/`vorstand` bleibt die Suche über alle User unverändert; für Caller außerhalb des Zugriffskreises bleibt die Suche auf gemeinsame Teams beschränkt.

#### Scenario: Trainer findet teamfremden Trainer

- **WHEN** ein Kader-Trainer von T1 `GET /api/chat/users?q=<Name eines Trainers von T2>` aufruft
- **THEN** enthält das Ergebnis den Trainer von T2, obwohl kein gemeinsames Team besteht

#### Scenario: Spieler findet teamfremden Trainer nicht

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit nach einem Trainer eines fremden Teams sucht
- **THEN** ist dieser nicht im Ergebnis enthalten

