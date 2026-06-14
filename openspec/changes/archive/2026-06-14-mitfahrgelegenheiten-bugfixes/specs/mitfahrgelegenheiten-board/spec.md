## MODIFIED Requirements

### Requirement: Mitfahrgelegenheit anlegen ist idempotent (suche)
Ein Nutzer SHALL pro Spiel und Typ höchstens eine Mitfahrgelegenheit besitzen. `POST /api/mitfahrgelegenheiten` mit `typ='suche'` MUSS einen bestehenden Eintrag aktualisieren statt einen neuen anzulegen, wenn bereits ein `suche`-Eintrag des Nutzers für dasselbe Spiel existiert.

#### Scenario: Erster suche-Eintrag wird angelegt
- **WHEN** ein Nutzer `POST /api/mitfahrgelegenheiten` mit `typ='suche'` für ein Spiel aufruft, für das er noch keinen suche-Eintrag hat
- **THEN** wird ein neuer Eintrag angelegt und HTTP 200 zurückgegeben

#### Scenario: Bestehender suche-Eintrag wird aktualisiert
- **WHEN** ein Nutzer `POST /api/mitfahrgelegenheiten` mit `typ='suche'` für ein Spiel aufruft, für das er bereits einen suche-Eintrag hat
- **THEN** wird der bestehende Eintrag aktualisiert (kein neuer Row) und HTTP 200 zurückgegeben

#### Scenario: Datenbank verhindert suche-Duplikate
- **WHEN** durch eine Race Condition zwei gleichzeitige suche-Inserts für denselben Nutzer und dasselbe Spiel eintreffen
- **THEN** schlägt einer der Inserts mit einem Constraint-Fehler fehl

---

### Requirement: Modal zeigt bestehende Einträge des Nutzers
Das Eintrag-Modal SHALL beim Öffnen die Daten eines bereits vorhandenen eigenen Eintrags des gewählten Typs vorausfüllen. Beim Typ-Wechsel innerhalb des Modals werden die Felder mit dem bestehenden Eintrag des neuen Typs befüllt — oder auf Defaults gesetzt, falls kein Eintrag existiert.

#### Scenario: Modal öffnet mit bestehendem biete-Eintrag
- **WHEN** ein Nutzer das Modal für ein Spiel öffnet, für das er bereits einen biete-Eintrag hat
- **THEN** sind Treffpunkt, Notiz und Plätze mit den gespeicherten Werten vorausgefüllt

#### Scenario: Typ-Wechsel zeigt bestehenden Eintrag des neuen Typs
- **WHEN** ein Nutzer im Modal von biete zu suche wechselt und bereits einen suche-Eintrag für dieses Spiel hat
- **THEN** werden die Felder mit den Werten des suche-Eintrags befüllt

#### Scenario: Typ-Wechsel ohne bestehenden Eintrag zeigt leere Felder
- **WHEN** ein Nutzer im Modal von biete zu suche wechselt und noch keinen suche-Eintrag hat
- **THEN** sind Treffpunkt und Notiz leer; Plätze ist auf 1 gesetzt

---

### Requirement: Generische Events mit mehreren Teams erscheinen genau einmal
Ein generisches Event, das mehreren Teams zugeordnet ist, SHALL in der Mitfahrgelegenheiten-Liste genau einmal erscheinen. Der angezeigte Team-Name zeigt alle beteiligten Teams komma-separiert.

#### Scenario: Generisches Event mit zwei Teams — eine Zeile
- **WHEN** ein generisches Event den Teams A und B zugeordnet ist
- **THEN** erscheint es genau einmal in der Liste mit Team-Name „A, B"

#### Scenario: Heimspiel mit einem Team — unverändert
- **WHEN** ein Heim- oder Auswärtsspiel genau einem Team zugeordnet ist
- **THEN** erscheint es unverändert einmal mit dem Team-Namen
