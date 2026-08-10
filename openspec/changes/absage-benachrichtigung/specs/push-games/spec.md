## MODIFIED Requirements

### Requirement: Push bei Spiel-Ereignissen

Das System SHALL allen berechtigten Team-Mitgliedern und deren Elternteilen eine Push
Notification senden, wenn ein Spiel erstellt, geändert oder gelöscht wird — sofern der
Nutzer Push für die Kategorie `games` nicht deaktiviert hat und die Benachrichtigung nicht
per `silent`-Flag unterdrückt wurde.

Für **bestehende** Spiele MUSS die Notification-`url` auf den konkreten Spieltermin in der
Termine-Seite zeigen (`/termine?focus=game-<id>`), damit der Empfänger direkt zu- oder
absagen kann.

Für **gelöschte** Spiele SHALL die `url` der **leere String** sein. Die frühere Regelung
(`/termine`) SHALL NICHT mehr gelten: sie führte den Empfänger in einen Kalender, in dem
der Termin gerade verschwunden war. Der Klick auf eine solche Benachrichtigung öffnet bzw.
fokussiert die App, ohne zu navigieren.

Die Benachrichtigung über ein gelöschtes Spiel SHALL Gegner, Datum und den Namen des
auslösenden Nutzers enthalten sowie — falls angegeben — den Löschgrund.

#### Scenario: Neues Spiel erstellt
- **WHEN** ein Admin oder Trainer ein neues Spiel über `POST /api/games` anlegt
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification mit Titel „Neues Spiel" und der Gegnerinfo
- **THEN** zeigt der Klick-Link auf `/termine?focus=game-<id>` des neu erstellten Spiels

#### Scenario: Spiel verschoben oder geändert
- **WHEN** ein Admin oder Trainer ein Spiel über `PUT /api/games/{id}` aktualisiert (Datum, Zeit oder Ort geändert)
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification „Spielinfo geändert"
- **THEN** zeigt der Klick-Link auf `/termine?focus=game-<id>`

#### Scenario: Spiel abgesagt (gelöscht)
- **WHEN** ein Admin oder Trainer ein Spiel über `DELETE /api/games/{id}` löscht
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification „Spiel abgesagt"
- **THEN** enthält der Body Gegner, Datum im Format `TT.MM.JJJJ` und den Namen des auslösenden Nutzers
- **THEN** ist die `url` der leere String

#### Scenario: Spiel mit Grund abgesagt
- **WHEN** ein Admin oder Trainer ein Spiel mit `{"reason":"Halle gesperrt"}` löscht
- **THEN** enthält der Body zusätzlich den Text „Halle gesperrt"

#### Scenario: Vorstand löscht ohne Benachrichtigung
- **WHEN** ein Nutzer mit Capability `suppress_event_notification` ein Spiel mit `{"silent":true}` löscht
- **THEN** erhält kein Nutzer eine `games`-Notification
- **THEN** wird das SSE-Live-Update trotzdem gesendet

#### Scenario: Nutzer mit deaktiviertem Push erhält keine Notification
- **WHEN** ein Spiel-Ereignis eintritt und ein Nutzer hat `push_enabled=0` für Kategorie `games` in `notification_preferences`
- **THEN** erhält dieser Nutzer keine Push Notification für dieses Ereignis
