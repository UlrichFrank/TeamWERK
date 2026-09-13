## Purpose

Mitglieder einer Gruppenkonversation können direkt im Chat eine Umfrage stellen, auf die
alle aktiven Mitglieder mit einer oder mehreren Antworten abstimmen — nach dem Vorbild der
Signal-Umfragen: nicht anonym, Stimme jederzeit änderbar, vom Ersteller beendbar.

## ADDED Requirements

### Requirement: Umfrage in einer Gruppenkonversation anlegen

Das System SHALL aktiven Mitgliedern (`left_at IS NULL`) einer Konversation vom Typ `group`
erlauben, über `POST /api/chat/conversations/{id}/polls` mit
`{question, options: string[], allowMultiple: bool}` eine Umfrage anzulegen. Die Umfrage
MUST als Nachricht im Verlauf der Konversation erscheinen, deren Text die Frage ist, und
antwortet mit HTTP 201 und `{id}` (die Nachrichten-ID).

Validierung (jeweils nach Trimmen von Leerraum):
- Frage: 1–200 Zeichen, sonst HTTP 400.
- Optionen: mindestens 2, höchstens 10; jede Option 1–100 Zeichen; keine zwei Optionen
  gleich (Groß-/Kleinschreibung ignoriert). Sonst HTTP 400.
- Direktkonversation (`type = 'direct'`): HTTP 400 — Umfragen gibt es nur in Gruppen.
- Nicht-Mitglied oder ausgetretenes Mitglied: HTTP 403.

Die Reihenfolge der Optionen MUST der Reihenfolge im Request entsprechen.

#### Scenario: Umfrage mit Einfachauswahl anlegen
- **WHEN** ein aktives Mitglied der Gruppe „D-Jugend Eltern" eine Umfrage mit Frage
  „Wer fährt am Samstag?" und den Optionen „Ich", „Ich nicht", „Nur Hinfahrt" ohne
  Mehrfachauswahl anlegt
- **THEN** antwortet der Server mit HTTP 201 und der ID der neuen Nachricht
- **THEN** zeigt der Verlauf aller Mitglieder eine Umfrage-Karte mit der Frage, den drei
  Optionen in dieser Reihenfolge und dem Hinweis „Eine Antwort wählen"

#### Scenario: Zu wenige Optionen
- **WHEN** eine Umfrage mit nur einer nicht-leeren Option angelegt werden soll
- **THEN** antwortet der Server mit HTTP 400 und legt keine Nachricht an

#### Scenario: Zu viele Optionen
- **WHEN** eine Umfrage mit 11 Optionen angelegt werden soll
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Doppelte Optionen
- **WHEN** die Optionen „Pizza" und „ pizza " enthalten sind
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Direktkonversation
- **WHEN** ein Teilnehmer einer Direktkonversation eine Umfrage anlegen will
- **THEN** antwortet der Server mit HTTP 400
- **THEN** bietet der Composer einer Direktkonversation keinen Umfrage-Button an

#### Scenario: Ausgetretenes Mitglied
- **WHEN** ein Nutzer, der die Gruppe verlassen hat, eine Umfrage anlegen will
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Neue Umfrage verhält sich wie eine neue Nachricht

Eine neu angelegte Umfrage MUST für die übrigen aktiven Mitglieder dieselben Effekte haben
wie eine neue Textnachricht: sie zählt als ungelesene Nachricht, löst das Chat-Live-Event
für neue Nachrichten aus und erzeugt eine Chat-Push-Benachrichtigung (unter Beachtung der
Chat-Push-Präferenz) mit dem Text `Umfrage: <Frage>`. In der Konversationsliste MUST eine
Umfrage als letzte Nachricht als Umfrage erkennbar sein (Kennzeichen + Frage).

#### Scenario: Push und Ungelesen-Zähler
- **WHEN** Anna in einer Gruppe mit Bob eine Umfrage „Pizza oder Nudeln?" anlegt
- **THEN** erhält Bob eine Chat-Push-Benachrichtigung mit dem Text „Umfrage: Pizza oder
  Nudeln?"
- **THEN** steigt Bobs Ungelesen-Zähler der Konversation um 1

#### Scenario: Konversationsliste
- **WHEN** die jüngste Nachricht einer Gruppe eine Umfrage ist
- **THEN** zeigt die Konversationsliste als Vorschau ein Umfrage-Kennzeichen und die Frage

### Requirement: Abstimmen, Stimme ändern und zurückziehen

Aktive Mitglieder der Konversation SHALL über `PUT /api/chat/messages/{id}/poll/vote` mit
`{optionIds: number[]}` ihre **vollständige** Auswahl für eine offene Umfrage setzen. Der
Aufruf ersetzt die bisherige Auswahl des Nutzers atomar; eine leere Liste zieht die Stimme
zurück. Antwort HTTP 204.

- Einfachauswahl (`allowMultiple = false`) mit mehr als einer Option: HTTP 400.
- Option-ID, die nicht zu dieser Umfrage gehört, oder doppelte ID: HTTP 400.
- Beendete Umfrage: HTTP 409, die Auswahl bleibt unverändert.
- Nachricht existiert nicht, ist gelöscht oder ist keine Umfrage: HTTP 404.
- Nicht-Mitglied oder ausgetretenes Mitglied: HTTP 403.

Stimmen MUST keine Push-Benachrichtigung auslösen und den Ungelesen-Zähler nicht verändern.

#### Scenario: Erste Stimme bei Einfachauswahl
- **WHEN** Bob in einer offenen Einfachauswahl-Umfrage „Ich" wählt
- **THEN** antwortet der Server mit HTTP 204
- **THEN** zeigt die Karte bei „Ich" eine Stimme mehr und markiert „Ich" als Bobs Wahl

#### Scenario: Stimme bei Einfachauswahl ändern
- **WHEN** Bob, der „Ich" gewählt hat, in derselben Umfrage „Nur Hinfahrt" wählt
- **THEN** zählt „Ich" eine Stimme weniger und „Nur Hinfahrt" eine Stimme mehr
- **THEN** hat Bob genau eine Stimme in dieser Umfrage

#### Scenario: Mehrfachauswahl
- **WHEN** Bob in einer Umfrage mit Mehrfachauswahl „Samstag" und „Sonntag" wählt
- **THEN** zählen beide Optionen Bobs Stimme
- **THEN** zählt die Umfrage Bob als einen Abstimmenden, nicht zwei

#### Scenario: Stimme zurückziehen
- **WHEN** Bob seine einzige gewählte Option erneut antippt
- **THEN** sendet der Client `{optionIds: []}` und Bob hat keine Stimme mehr in der Umfrage

#### Scenario: Mehrere Optionen bei Einfachauswahl
- **WHEN** ein Client für eine Einfachauswahl-Umfrage zwei Option-IDs sendet
- **THEN** antwortet der Server mit HTTP 400 und die bisherige Auswahl bleibt unverändert

#### Scenario: Fremde Option
- **WHEN** ein Client eine Option-ID einer anderen Umfrage sendet
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Beendete Umfrage
- **WHEN** Bob in einer beendeten Umfrage abstimmen will
- **THEN** antwortet der Server mit HTTP 409
- **THEN** sind die Optionen der Karte nicht mehr antippbar

#### Scenario: Stimme löst keine Push aus
- **WHEN** Bob in Annas Umfrage abstimmt
- **THEN** erhält niemand eine Push-Benachrichtigung und kein Ungelesen-Zähler ändert sich

### Requirement: Nicht-anonyme Ergebnisanzeige

Das System SHALL allen Mitgliedern der Konversation (auch ausgetretenen, die den Verlauf
lesen dürfen) je Umfrage anzeigen: je Option die Stimmenzahl und einen Anteilsbalken, die
eigene Auswahl, die Zahl der Abstimmenden und die Namen der Abstimmenden je Option. Die
Umfrage-Daten MUST in der Nachrichtenliste pro Umfrage-Nachricht mitgeliefert werden und
über `GET /api/chat/messages/{id}/poll` einzeln abrufbar sein (HTTP 403 für Nicht-Mitglieder,
HTTP 404 für gelöschte Nachrichten und Nachrichten ohne Umfrage).

Stimmen ausgetretener oder entfernter Mitglieder MUST erhalten bleiben und weiter zählen.

#### Scenario: Stimmen anzeigen
- **WHEN** Anna auf einer Umfrage-Karte „Stimmen anzeigen" antippt
- **THEN** öffnet sich eine Ansicht, die je Option die Namen der Abstimmenden auflistet

#### Scenario: Anteilsbalken
- **WHEN** von vier Abstimmenden drei „Pizza" und einer „Nudeln" gewählt haben
- **THEN** zeigt die Karte bei „Pizza" 3 Stimmen mit einem Balken von 75 % und bei „Nudeln"
  1 Stimme mit einem Balken von 25 %
- **THEN** zeigt die Karte „4 Stimmen" als Zahl der Abstimmenden

#### Scenario: Einzelabruf durch Nicht-Mitglied
- **WHEN** ein Nutzer, der nie Mitglied der Konversation war,
  `GET /api/chat/messages/{id}/poll` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Stimme eines ausgetretenen Mitglieds
- **WHEN** Bob abgestimmt hat und danach die Gruppe verlässt
- **THEN** zählt seine Stimme weiterhin und sein Name bleibt in der Stimmen-Ansicht

### Requirement: Umfrage beenden

Nur der Ersteller einer Umfrage SHALL sie über `POST /api/chat/messages/{id}/poll/close`
beenden können (HTTP 204). Andere Nutzer — auch Admins — erhalten HTTP 403. Das Beenden ist
endgültig und idempotent: ein erneuter Aufruf auf eine beendete Umfrage antwortet mit
HTTP 204 ohne Änderung. Eine beendete Umfrage MUST als „Beendet" gekennzeichnet sein und das
Ergebnis weiter anzeigen. Gelöschte Nachricht oder Nachricht ohne Umfrage: HTTP 404.

#### Scenario: Ersteller beendet die Umfrage
- **WHEN** Anna im Kontextmenü ihrer offenen Umfrage „Umfrage beenden" wählt
- **THEN** antwortet der Server mit HTTP 204
- **THEN** zeigt die Karte bei allen Mitgliedern „Beendet" und keine Option ist mehr wählbar

#### Scenario: Fremder Nutzer will beenden
- **WHEN** Bob `POST /api/chat/messages/{id}/poll/close` für Annas Umfrage aufruft
- **THEN** antwortet der Server mit HTTP 403
- **THEN** enthält Bobs Kontextmenü der Umfrage keinen Eintrag „Umfrage beenden"

#### Scenario: Erneutes Beenden
- **WHEN** Anna eine bereits beendete Umfrage erneut beendet
- **THEN** antwortet der Server mit HTTP 204 und der Zeitpunkt des Beendens bleibt unverändert

### Requirement: Live-Aktualisierung von Umfragen

Jede Stimmabgabe und jedes Beenden MUST allen aktiven Mitgliedern der Konversation über den
Chat-Live-Kanal das Event `chat:poll-updated:<convId>:<messageId>` senden. Ein Client, der die
Konversation geöffnet hat, MUST daraufhin nur diese eine Umfrage aktualisieren, ohne die
Nachrichtenliste neu zu laden oder die Scroll-Position zu verändern.

#### Scenario: Stimme erscheint live
- **WHEN** Anna die Gruppe geöffnet hat und Bob abstimmt
- **THEN** aktualisiert sich die Stimmenzahl auf Annas Umfrage-Karte ohne Neuladen der Seite

#### Scenario: Beenden erscheint live
- **WHEN** Bob die Gruppe geöffnet hat und Anna ihre Umfrage beendet
- **THEN** zeigt Bobs Karte „Beendet" und die Optionen sind nicht mehr wählbar

### Requirement: Löschen, Antworten und Reaktionen

Eine Umfrage-Nachricht SHALL wie jede Nachricht gelöscht werden können (gleiche Rechte, Soft-
Delete, Placeholder „Nachricht gelöscht"). Nach dem Löschen MUST die Umfrage nicht mehr
ausgeliefert werden und alle Umfrage-Routen für diese Nachricht mit HTTP 404 antworten.
Auf eine Umfrage kann geantwortet und reagiert werden wie auf jede Nachricht; das
Antwort-Zitat zeigt die Frage.

#### Scenario: Gelöschte Umfrage
- **WHEN** Anna ihre Umfrage löscht und Bob danach abstimmen will
- **THEN** zeigt der Verlauf den Placeholder „Nachricht gelöscht" statt der Karte
- **THEN** antwortet `PUT /api/chat/messages/{id}/poll/vote` mit HTTP 404

#### Scenario: Antwort auf eine Umfrage
- **WHEN** Bob auf Annas Umfrage antwortet
- **THEN** zeigt das Zitat seiner Antwort die Frage der Umfrage
