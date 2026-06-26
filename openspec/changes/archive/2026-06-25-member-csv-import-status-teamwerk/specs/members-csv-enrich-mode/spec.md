## ADDED Requirements

### Requirement: Status-Spalte „Status TeamWERK"

Das System SHALL beim CSV-Import den Lebenszyklus-Status eines Mitglieds ausschließlich aus der CSV-Spalte `Status TeamWERK` ableiten. Erlaubte Eingabewerte sind die zulässigen Werte des CHECK-Constraint auf `members.status` (`aktiv`, `verletzt`, `pausiert`, `ausgetreten`, `passiv`, `honorar`, `anwaerter`) sowie der Alias `gekündigt` → `ausgetreten`. Unbekannte Werte werden beim Anlegen auf `aktiv` gemappt; beim Update wird `members.status` in diesem Fall nicht verändert.

#### Scenario: Status TeamWERK = passiv legt passives Mitglied an

- **GIVEN** eine CSV-Zeile mit `Status TeamWERK=passiv` und kein bestehendes Mitglied
- **WHEN** der Import im Modus `update` (Create-Pfad bei `not_found`) oder `append` läuft
- **THEN** wird das Mitglied mit `members.status='passiv'` angelegt

#### Scenario: Status TeamWERK = gekündigt aliasiert ausgetreten

- **GIVEN** eine CSV-Zeile mit `Status TeamWERK=gekündigt`
- **WHEN** der Import einen Bestandsmatch findet
- **THEN** wird `members.status='ausgetreten'` gesetzt

### Requirement: Direkte Beitragsfrei-Spalte

Das System SHALL den `beitragsfrei`-Flag ausschließlich aus der CSV-Spalte `beitragsfrei` ableiten: Wert `ja` (case-insensitive, getrimmt) → `1`, sonst `0`. Im Enrich-Modus darf das Flag NUR von `0` auf `1` ergänzt werden; ein bestehendes `1` wird nicht auf `0` zurückgesetzt.

#### Scenario: Beitragsfrei = ja setzt Flag bei Anlage

- **WHEN** eine CSV-Zeile mit `beitragsfrei=ja` ein neues Mitglied erzeugt
- **THEN** wird `members.beitragsfrei=1` gespeichert

#### Scenario: Enrich respektiert bestehendes Flag

- **GIVEN** ein Bestandsmitglied mit `members.beitragsfrei=1`
- **WHEN** der Enrich-Import die Zeile mit `beitragsfrei=` (leer) verarbeitet
- **THEN** bleibt `members.beitragsfrei=1` unverändert

### Requirement: Spalte „Grund für Beitragsfreiheit"

Das System SHALL die CSV-Spalte `Grund für Beitragsfreiheit` auf das Feld `members.beitragsfrei_grund` mappen. Im Enrich-Modus wird ein bereits gefülltes DB-Feld nicht überschrieben; ein leeres DB-Feld wird mit dem CSV-Wert befüllt.

#### Scenario: Grund wird beim Anlegen übernommen

- **WHEN** eine CSV-Zeile mit `beitragsfrei=ja` und `Grund für Beitragsfreiheit=kein aktiver Sportler mehr` ein neues Mitglied erzeugt
- **THEN** wird `members.beitragsfrei_grund='kein aktiver Sportler mehr'` gespeichert

#### Scenario: Enrich überschreibt belegten Grund nicht

- **GIVEN** ein Bestandsmitglied mit `members.beitragsfrei_grund='Zweitspielrecht'`
- **WHEN** der Enrich-Import die Zeile mit `Grund für Beitragsfreiheit=kein aktiver Sportler mehr` verarbeitet
- **THEN** bleibt `members.beitragsfrei_grund='Zweitspielrecht'`

## REMOVED Requirements

### Requirement: Ableitung von `beitragsfrei` aus der Status-Spalte

**Reason:** Die CSV führt seit dem aktuellen Export eine eigene Spalte `beitragsfrei`. Die bisherige Ableitung (`Status == "beitragsfrei"` → `members.beitragsfrei = 1`) ist mehrdeutig und liefert mit dem neuen Schema falsche Ergebnisse, weil die Spalte „Status" inzwischen Freitext-Begründungen statt eines kontrollierten Statuswertes enthält.

**Migration:** Die alte Spalte „Status" wird ersatzlos ignoriert. Bestandsmitglieder behalten ihren `beitragsfrei`-Flag aus früheren Importen oder manueller Pflege; neue Imports lesen ausschließlich die Spalte `beitragsfrei`.
