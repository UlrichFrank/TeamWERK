# spieler-aufgaben Specification

## Purpose

Trainer-gepflegter Aufgaben-Catalog pro Kader plus Zuweisung an Spieler. Aufgaben sind informelle Team-Verantwortlichkeiten (Mannschaftskasse, Harz, Leibchen, …) ohne weitere Semantik — reine Anzeige neben dem Spielernamen im Team-Tab. Snapshot-Labels; Catalog-Edit ändert bestehende Zuweisungen nicht.

## Requirements

### Requirement: Aufgaben-Catalog pro Kader
Das System SHALL pro Kader einen Catalog von Aufgaben-Labels (`responsibility_types`) führen, den der Trainer des Kaders anlegen, erweitern und löschen kann. Der Catalog dient ausschließlich als Vorschlagsliste für die Zuweisung und trägt keine weitere Semantik. Mutationen SHALL durch das Trainer-des-Kaders-Gate geschützt sein (`admin` passt immer); alle anderen erhalten HTTP 403.

#### Scenario: Trainer legt Aufgabe im Catalog an
- **WHEN** ein Trainer des Kaders eine Aufgabe „Harz" zum Catalog hinzufügt
- **THEN** antwortet das System mit 200/201 und die Aufgabe erscheint als Vorschlag für diesen Kader

#### Scenario: Nicht-Trainer darf Catalog nicht ändern
- **WHEN** ein Spieler (kein Trainer, kein admin) versucht, den Aufgaben-Catalog zu ändern
- **THEN** antwortet das System mit HTTP 403 und der Catalog bleibt unverändert

#### Scenario: Catalog ist kader-scoped
- **WHEN** ein Trainer den Catalog von Kader A liest
- **THEN** enthält die Antwort nur die Aufgaben von Kader A, nicht die anderer Kader

### Requirement: Zuweisung von Aufgaben an Spieler
Das System SHALL es dem Trainer des Kaders erlauben, einem Spieler eine oder mehrere Aufgaben zuzuweisen (`member_responsibilities`). Das Label wird als Snapshot gespeichert und kann aus dem Catalog stammen oder Freitext sein. Ein späteres Ändern oder Löschen eines Catalog-Eintrags SHALL bereits zugewiesene Labels NICHT verändern. Zuweisung und Entfernen SHALL durch das Trainer-des-Kaders-Gate geschützt sein.

#### Scenario: Trainer weist einem Spieler eine Aufgabe zu
- **WHEN** ein Trainer dem Spieler die Aufgabe „Mannschaftskasse" zuweist
- **THEN** antwortet das System mit 200/201 und der Spieler trägt die Aufgabe „Mannschaftskasse"

#### Scenario: Ein Spieler kann mehrere Aufgaben tragen
- **WHEN** ein Trainer demselben Spieler „Leibchen" und „Harz" zuweist
- **THEN** trägt der Spieler beide Aufgaben

#### Scenario: Catalog-Edit ändert bestehende Zuweisung nicht
- **WHEN** ein Trainer den Catalog-Eintrag „Harz" umbenennt oder löscht, nachdem er einem Spieler „Harz" zugewiesen hat
- **THEN** bleibt das zugewiesene Label des Spielers unverändert „Harz"

#### Scenario: Nicht-Trainer darf keine Aufgaben zuweisen
- **WHEN** ein Spieler oder Elternteil versucht, eine Aufgabe zuzuweisen
- **THEN** antwortet das System mit HTTP 403

### Requirement: Aufgaben-Anzeige auf der Roster-Response
Das System SHALL zugewiesene Aufgaben je Spieler als Teil der bestehenden Roster-Response (`GET /api/teams/{id}/roster`) ausliefern. Die Sichtbarkeit entspricht damit exakt der Roster-Sichtbarkeit (Spieler, Trainer, Erweiterter Kader und Eltern des Teams). Aufgaben SHALL im Team-Tab neben dem Spielernamen dargestellt werden.

#### Scenario: Spieler sieht die Aufgaben seines Teams
- **WHEN** ein Spieler die Roster-Response seines Teams lädt
- **THEN** enthält jeder Spieler-Eintrag seine zugewiesenen Aufgaben-Labels

#### Scenario: Elternteil sieht Aufgaben (Roster-Sichtbarkeit)
- **WHEN** ein Elternteil mit Team-Zugriff die Roster-Response lädt
- **THEN** enthält die Antwort die Aufgaben-Labels der Spieler

### Requirement: Live-Update bei Aufgaben-Änderung
Jede Aufgaben-Mutations-Route SHALL `h.hub.Broadcast("responsibilities")` (oder einen äquivalenten Broadcast-Helfer) aufrufen, und das Frontend SHALL via `useLiveUpdates` auf das Event reagieren und die betroffene Team-Ansicht neu laden.

#### Scenario: Broadcast nach Zuweisung
- **WHEN** ein Trainer eine Aufgabe zuweist oder entfernt
- **THEN** sendet der Handler einen `responsibilities`-Broadcast und offene Team-Ansichten aktualisieren sich ohne Reload
