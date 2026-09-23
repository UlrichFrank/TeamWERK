# Spec Delta

## Purpose

Informiert die Trainer eines Kaders aktiv per Push, wenn ein Spieler in der letzten Woche vor einem Spiel oder Training seine bereits abgegebene Zu-/Absage ändert, damit kurzfristige Umentscheidungen in die Planung einfließen.

## ADDED Requirements

### Requirement: Trainer-Push bei kurzfristiger RSVP-Änderung
Das System SHALL nach einer erfolgreichen RSVP über `POST /api/games/{id}/respond` oder `POST /api/training-sessions/{id}/respond` eine Benachrichtigung der Kategorie `operativ` an die Trainer des betroffenen Kaders senden, wenn alle folgenden Bedingungen erfüllt sind:
1. Für das Ziel-Mitglied existierte vor dem Request bereits eine explizite Antwort auf diesen Termin (`game_responses` bzw. `training_responses`).
2. Der neue Status (`confirmed`/`declined`/`maybe`) unterscheidet sich vom vorherigen.
3. Der Terminbeginn (Datum + Uhrzeit, Europe/Berlin) liegt in der Zukunft und höchstens 7 × 24 Stunden nach dem Zeitpunkt der Änderung.
4. Das Ziel-Mitglied ist nicht selbst Trainer des betroffenen Kaders.

Betroffene Kader sind bei einem Spiel die Kader der Mannschaften des Spiels (`game_teams`) in der Saison des Spiels, bei einem Training der Kader der Einheit (`training_sessions.kader_id`, inkl. Übungsgruppen). Empfänger sind die Nutzerkonten der Mitglieder in `kader_trainers` dieser Kader, dedupliziert.

#### Scenario: Absage drei Tage vor dem Spiel
- **WHEN** ein Spieler mit bestehender Antwort `confirmed` drei Tage vor Spielbeginn auf `declined` umstellt
- **THEN** erhalten alle Trainer der Kader des Spiels genau eine Meldung der Kategorie `operativ`, die Spielername, Termin, alten Status „Zusage" und neuen Status „Absage" nennt

#### Scenario: Änderung beim Training innerhalb des Fensters
- **WHEN** ein Spieler mit bestehender Antwort `declined` zwei Tage vor einer Trainingseinheit auf `confirmed` umstellt
- **THEN** erhalten die Trainer des Kaders der Einheit eine Meldung

#### Scenario: Erste Antwort löst keine Meldung aus
- **WHEN** ein Spieler ohne bisherige Antwort zwei Tage vor dem Termin erstmals `declined` sendet
- **THEN** wird keine Trainer-Meldung erzeugt

#### Scenario: Unveränderter Status löst keine Meldung aus
- **WHEN** ein Spieler mit bestehender Antwort `declined` erneut `declined` mit geändertem Grund sendet
- **THEN** wird keine Trainer-Meldung erzeugt

#### Scenario: Termin außerhalb des 7-Tage-Fensters
- **WHEN** ein Spieler seine Antwort acht Tage vor Terminbeginn von `confirmed` auf `declined` ändert
- **THEN** wird keine Trainer-Meldung erzeugt

#### Scenario: Grenze exakt 7 Tage
- **WHEN** die Änderung genau 7 × 24 Stunden vor Terminbeginn erfolgt
- **THEN** wird die Meldung erzeugt

#### Scenario: Änderung durch Elternteil
- **WHEN** ein Elternteil für sein Kind innerhalb des Fensters eine bestehende Antwort ändert
- **THEN** erhalten die Trainer die Meldung mit dem Namen des Kindes

#### Scenario: Trainer ändert eigene Antwort
- **WHEN** ein Mitglied, das Trainer des betroffenen Kaders ist, seine eigene Antwort innerhalb des Fensters ändert
- **THEN** wird keine Trainer-Meldung erzeugt

#### Scenario: Abgelehnte RSVP erzeugt keine Meldung
- **WHEN** die RSVP mit 403 (`rsvp_locked_absence`, `series_unavailable`, fremdes Mitglied) oder wegen Cutoff abgelehnt wird
- **THEN** wird keine Trainer-Meldung erzeugt

### Requirement: Auslöser erhält keine eigene Meldung
Das System MUST das Nutzerkonto, das die RSVP-Änderung abgesendet hat, aus der Empfängermenge entfernen.

#### Scenario: Trainer sagt für Spieler um
- **WHEN** ein Trainer des Kaders innerhalb des Fensters die bestehende Antwort eines Spielers ändert
- **THEN** erhalten die übrigen Trainer die Meldung, der handelnde Trainer nicht

### Requirement: Meldungsinhalt und Zustellung
Die Meldung SHALL den Grund enthalten, sofern ein nicht-leerer Grund übermittelt wurde, und auf den Termin verlinken (`/termine?focus=game-<id>` bzw. `/termine?focus=training-<id>`). Die Zustellung MUST die Push-Präferenz der Kategorie `operativ` respektieren und die Meldung im Event-Log des Empfängers ablegen. Ein Fehler bei der Empfängerauflösung oder Zustellung MUST die RSVP-Antwort (HTTP 204) nicht beeinflussen.

#### Scenario: Grund wird mitgeschickt
- **WHEN** die Änderung auf `declined` mit Grund „krank" erfolgt
- **THEN** enthält der Meldungstext „krank"

#### Scenario: Trainer hat operativ-Push abgeschaltet
- **WHEN** ein Trainer `operativ` `push_enabled=0` gesetzt hat
- **THEN** bekommt er keine Push, die Meldung steht aber in seinem Event-Log

### Requirement: Schalter im Profil benennt die Meldung
Das Frontend SHALL in der Beschreibung des Präferenz-Schalters „Vereinsaufgaben“ (Kategorie `operativ`, Profil-Tab „Sonstiges“) die kurzfristigen Umentscheidungen der Spieler als Beispiel nennen, damit die Meldung dem zuständigen Schalter zugeordnet werden kann.

#### Scenario: Beschreibung nennt Umentscheidungen
- **WHEN** ein Nutzer im Profil den Tab „Sonstiges“ öffnet
- **THEN** nennt die Beschreibung unter „Vereinsaufgaben“ neben Anwesenheiten und Spielberichten auch kurzfristige Umentscheidungen der Spieler
