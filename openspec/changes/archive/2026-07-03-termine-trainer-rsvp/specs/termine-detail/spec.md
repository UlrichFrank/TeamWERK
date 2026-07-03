## ADDED Requirements

### Requirement: Termin-Detail-Tabelle zeigt drei benannte Sektionen in fester Reihenfolge

Die Termin-Detail-Tabelle (`TermineDetailPage.tsx`) SHALL die Teilnehmer in drei benannten Sektionen anzeigen (nicht-leere zuerst): **Trainer**, **Spieler**, **Erweiterter Kader**. Die Trainer-Sektion MUSS oberhalb der Spieler-Sektion stehen, die Spieler-Sektion oberhalb des Erweiterten Kaders. Jede Sektion wird durch dieselbe `border-t-2 border-brand-border`-Linie visuell abgetrennt. Leere Sektionen (0 Rows) werden weggelassen.

Diese Regel gilt NICHT im Multi-Team-Modus generischer Events (`event_type='generisch'` mit ≥2 Teams) — dort bleibt die Team-Gruppierung erhalten.

#### Scenario: Alle drei Sektionen sichtbar

- **WHEN** ein User einen Termin mit ≥1 Trainer, ≥1 Spieler und ≥1 erweitertem Kader-Mitglied öffnet
- **THEN** zeigt die Tabelle drei benannte Sektionen in der Reihenfolge Trainer / Spieler / Erweiterter Kader

#### Scenario: Kein Trainer im Kader

- **WHEN** der Kader keine Trainer hat
- **THEN** wird die Trainer-Sektion samt Kopfzeile weggelassen, Spieler-Sektion erscheint zuoberst mit sichtbarem Titel „Spieler"

#### Scenario: Kein erweiterter Kader

- **WHEN** kein Mitglied im erweiterten Kader steht
- **THEN** wird die Sektion „Erweiterter Kader" samt Kopfzeile weggelassen

---

### Requirement: Anwesenheit- und Aufstellung-Zelle bleiben für Trainer-Zeilen leer

`ParticipantRow` in der Termin-Detail-Tabelle SHALL für Zeilen mit `is_trainer=true` weder eine Checkbox noch einen Platzhalter (Strich) in der Anwesend- und Aufstellung-Spalte rendern. Die `<td>`-Zellen bleiben strukturell erhalten (leerer Inhalt), damit die Spaltenausrichtung mit Spieler-Zeilen erhalten bleibt.

#### Scenario: Trainer-Zeile hat leere Anwesend-Zelle

- **WHEN** ein Trainer in der Tabelle erscheint und die Anwesend-Spalte für den User sichtbar ist (Trainer/Admin, Termin vergangen)
- **THEN** wird in der Anwesend-Spalte der Trainer-Zeile keine Checkbox und kein Text gerendert

#### Scenario: Trainer-Zeile hat leere Aufstellung-Zelle

- **WHEN** ein Spiel-Termin die Aufstellung-Spalte zeigt
- **THEN** wird in der Aufstellung-Spalte der Trainer-Zeile keine Checkbox und kein Text gerendert

#### Scenario: Spalten bleiben vertikal ausgerichtet

- **WHEN** Trainer- und Spieler-Zeilen in derselben Tabelle stehen
- **THEN** liegt die Rückmeldung-Spalte in Trainer- und Spieler-Zeilen exakt untereinander (kein `colspan`-Shift)

## MODIFIED Requirements

### Requirement: `present`-Feld nur für Trainer sichtbar

Das Feld `present` in der Attendances-Response SHALL nur für Trainer und Admins einen Wert enthalten. Für Spieler und Eltern ist `present` immer `null`. Für Trainer-Zeilen (`is_trainer=true`) ist `present` unabhängig vom Aufrufer immer `null`, da Trainer keine Anwesenheits-Erfassung haben.

#### Scenario: Nicht-Trainer erhält kein present-Flag

- **WHEN** ein Spieler oder Elternteil `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `present` für alle Einträge `null`

#### Scenario: Trainer erhält present-Flags für Spieler-Zeilen

- **WHEN** ein Trainer `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** enthält `present` in Spieler-/Erweiterter-Kader-Zeilen den tatsächlich gespeicherten Wert (true/false/null)

#### Scenario: Trainer-Zeile hat immer present=null

- **WHEN** ein Trainer/Admin die Response abruft und darin eine Zeile mit `is_trainer=true` enthält
- **THEN** ist `present=null` in dieser Zeile
