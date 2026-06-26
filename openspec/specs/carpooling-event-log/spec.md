# carpooling-event-log Specification

## Purpose

Diese Spezifikation beschreibt die Capability `carpooling-event-log`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Vollständige Ereignisse persistieren

Das System SHALL für alle relevanten Carpooling-Ereignisse einen Eintrag in `carpooling_events` schreiben. Jeder Eintrag gehört zu genau einem betroffenen User (`user_id`), einem Spiel (`game_id`), hat einen `type` und einen `actor_name` (Name des auslösenden Users).

Erlaubte `type`-Werte: `biete_created`, `suche_created`, `pairing_requested`, `pairing_confirmed`, `pairing_rejected`, `pairing_cancelled`, `biete_deleted`, `suche_deleted`.

#### Scenario: Neuer Biete-Eintrag

- **WHEN** ein User einen `biete`-Eintrag für ein Spiel anlegt und andere User bereits `suche`-Einträge für dasselbe Spiel haben
- **THEN** wird für jeden dieser User ein Event `type='biete_created'` mit `actor_name` des Bieters geschrieben

#### Scenario: Neuer Suche-Eintrag

- **WHEN** ein User einen `suche`-Eintrag für ein Spiel anlegt und andere User bereits `biete`-Einträge für dasselbe Spiel haben
- **THEN** wird für jeden dieser User ein Event `type='suche_created'` geschrieben

#### Scenario: Paarungsanfrage gestellt

- **WHEN** ein User eine Paarungsanfrage stellt (POST /api/mitfahrt-paarungen)
- **THEN** wird für die Gegenseite ein Event `type='pairing_requested'` geschrieben

#### Scenario: Paarung bestätigt

- **WHEN** die Gegenseite eine Paarungsanfrage bestätigt (POST /api/mitfahrt-paarungen/{id}/confirm)
- **THEN** wird für den Initiator der Anfrage ein Event `type='pairing_confirmed'` geschrieben

#### Scenario: Paarungsanfrage abgelehnt

- **WHEN** eine `pending`-Paarung abgelehnt wird (POST /api/mitfahrt-paarungen/{id}/reject)
- **THEN** wird für den Initiator der Anfrage ein Event `type='pairing_rejected'` geschrieben

#### Scenario: Bestätigte Paarung storniert

- **WHEN** eine `confirmed`-Paarung abgelehnt/storniert wird
- **THEN** wird für die Gegenseite ein Event `type='pairing_cancelled'` geschrieben

#### Scenario: Biete-Eintrag gelöscht mit aktiver Paarung

- **WHEN** ein `biete`-Eintrag gelöscht wird und `pending` oder `confirmed` Paarungen dagegen existieren
- **THEN** wird für jeden betroffenen Suche-User ein Event `type='biete_deleted'` geschrieben, *bevor* das DELETE ausgeführt wird (Transaktion)

#### Scenario: Suche-Eintrag gelöscht mit aktiver Paarung

- **WHEN** ein `suche`-Eintrag gelöscht wird und eine `pending` oder `confirmed` Paarung dagegen existiert
- **THEN** wird für den Biete-User ein Event `type='suche_deleted'` geschrieben

#### Scenario: Eintrag gelöscht ohne aktive Paarung

- **WHEN** ein Eintrag gelöscht wird und keine `pending`/`confirmed` Paarung existiert
- **THEN** wird kein Event angelegt

### Requirement: Atomarität bei Löschungen

Das System SHALL Lösch-Events und das zugehörige DELETE in einer einzigen Transaktion ausführen.

#### Scenario: Fehler beim Event-Write

- **WHEN** das Schreiben eines Lösch-Events fehlschlägt
- **THEN** wird das DELETE nicht ausgeführt und die Transaktion zurückgerollt

### Requirement: Events nur für zukünftige Spiele anzeigen

Das System SHALL beim Laden des Dashboards nur Events zurückgeben, deren verknüpftes Spiel ein Datum >= heute hat.

#### Scenario: Event zu vergangenem Spiel

- **WHEN** ein `carpooling_events`-Eintrag existiert und `DATE(g.date) < DATE('now')`
- **THEN** wird er im Dashboard-Response nicht zurückgegeben
