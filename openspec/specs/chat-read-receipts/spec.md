# chat-read-receipts Specification

## Purpose

Absender von Chat-Nachrichten sehen einen WhatsApp-artigen Lese-Zustand pro
eigener Nachricht (gesendet/gelesen; in Gruppen ein `N/M gelesen`-Aggregat mit
on-demand-Detail). Diese Capability deckt die **Absender-Sicht** ab (die
Empfänger-/Unread-Sicht liegt in `chat-konversationen` bzw.
`chat-unread-app-badge`) und schließt Broadcasts bewusst aus.

## Requirements

### Requirement: Absender sieht Lese-Zustand seiner eigenen Nachrichten

Das System SHALL Absendern eines Chat-Beitrags pro Nachricht anzeigen, ob
mindestens ein Empfänger die Nachricht gelesen hat (Direct-Konversation:
zwei-Zustands-Anzeige gesendet/gelesen; Gruppen: Aggregat `N/M gelesen`
plus on-demand-Detail). Fremde Nachrichten tragen KEIN Read-Rendering — der
Zustand ist ausschließlich für den eigenen Absender-View sichtbar.

Die Route `GET /api/chat/messages/{id}/reads` liefert die Reader-Liste
`[{userId, name, readAt}]` sortiert nach `readAt` aufsteigend. Nur der
Absender der Nachricht (`m.sender_id = claims.UserID`) darf die Route
aufrufen; andere Nutzer bekommen HTTP 403. Für nicht existierende oder
gelöschte Nachrichten liefert die Route 404.

Der `MarkRead`-Handler (`POST /api/chat/conversations/{id}/read`) MUST nach
dem `INSERT OR IGNORE` in `message_reads` pro Absender der neu markierten
Nachrichten ein SSE-Event `chat:read-receipt` mit Payload
`{convId, readerUserId, upToMessageId, readAt}` an `sender.user_id` senden.
Bulk-Reads werden dabei pro Absender zu einem Event mit dem höchsten neu
markierten `message_id` als `upToMessageId` zusammengefasst, damit ein
Bulk-Read mit N Nachrichten NICHT N Events auslöst.

Die Ausgabe von `GET /api/chat/conversations/{id}/messages` (bzw. der
entsprechenden Listing-Route) MUST pro Nachricht `readCount` (Anzahl Reader
außer Sender) und `readTotal` (Anzahl aktive Konversations-Mitglieder außer
Sender) enthalten. Für Direct-Konversationen kollabiert das auf `read: bool`
(`readCount >= 1`).

#### Scenario: 1:1 — Empfänger liest, Absender sieht Zustandswechsel live

- **WHEN** Bob eine Nachricht an Anna sendet und Anna die Konversation öffnet
- **THEN** empfängt Bobs Client per SSE `chat:read-receipt` mit
  `readerUserId=Anna` und `upToMessageId=<Bobs Nachricht>`
- **THEN** wechselt die Nachricht in Bobs UI von `✓ gesendet` auf
  `✓✓ gelesen`

#### Scenario: Gruppe — Aggregat-Anzeige und Detail-Ansicht

- **WHEN** Bob in eine 8-Personen-Gruppe schreibt und drei Personen die
  Konversation öffnen
- **THEN** zeigt Bobs Nachricht `3/7 gelesen`
- **WHEN** Bob auf die eigene Nachrichten-Bubble tippt
- **THEN** lädt der Client `GET /api/chat/messages/{id}/reads` und rendert
  eine Liste mit den drei Readern samt `readAt`

#### Scenario: Detail-Route — Absender-only

- **WHEN** ein anderer Konversations-Teilnehmer als der Sender
  `GET /api/chat/messages/{id}/reads` aufruft
- **THEN** antwortet der Server mit HTTP 403 und leerem Body

#### Scenario: Detail-Route — gelöschte Nachricht

- **WHEN** eine Nachricht per Message-Delete entfernt wurde und der frühere
  Sender `GET /api/chat/messages/{id}/reads` aufruft
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Bulk-Read löst genau ein SSE-Event pro Absender aus

- **WHEN** Anna eine Konversation mit 30 ungelesenen Nachrichten von Bob
  öffnet (POST `/chat/conversations/{id}/read`)
- **THEN** empfängt Bob genau ein SSE-Event `chat:read-receipt` mit
  `upToMessageId=<höchste neu markierte message_id>`, nicht 30 einzelne

#### Scenario: Fremde Nachrichten tragen kein Read-Rendering

- **WHEN** Anna in der Konversationsansicht auf Bobs Nachricht tippt
- **THEN** öffnet sich KEIN Read-Detail-Modal, und Bobs Nachricht zeigt
  keine Tick-Icons für Anna

#### Scenario: Broadcast-Nachrichten sind ausgeschlossen

- **WHEN** der Vorstand einen Broadcast an alle Eltern schickt
- **THEN** trägt die Broadcast-Nachricht KEINEN Read-Tick und der Sender
  bekommt KEINEN `chat:read-receipt`-Event bei Empfänger-Reads. Die
  Zustellungs-Statistik von Broadcasts ist Sache eines separaten Changes
  ([[broadcast-delivery-report]], nicht in diesem Scope).

### Requirement: `message_reads.read_at` als Pflichtzeitstempel

Die Tabelle `message_reads` MUST die Spalte `read_at TIMESTAMP` tragen. Neue
Einträge SHALL `CURRENT_TIMESTAMP` gesetzt bekommen. Migration 031 fügt die
Spalte hinzu und füllt Bestandseinträge idempotent mit dem Migrations-
Zeitpunkt (der historische echte Read-Zeitpunkt ist nicht mehr rekonstruier-
bar; die Detail-Ansicht ist für Bestandsnachrichten damit ungenau, was
akzeptiert wird).

#### Scenario: Migration idempotent für Bestandsdaten

- **WHEN** `make migrate-up` mit einer bestehenden `message_reads`-Tabelle
  läuft
- **THEN** existiert die Spalte `read_at` und alle Bestandseinträge tragen
  einen `read_at`-Wert (nicht NULL), auch wenn dieser dem Migrations-
  Zeitpunkt entspricht

#### Scenario: Neuer Read setzt read_at explizit

- **WHEN** ein Nutzer eine Konversation öffnet und der `MarkRead`-Handler
  neue Einträge einfügt
- **THEN** trägt jeder neue Eintrag `read_at = CURRENT_TIMESTAMP`
