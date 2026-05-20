## ADDED Requirements

### Requirement: Kader-Modus konfigurierbar (gemischt vs. dediziert)

Jeder Kader MUSS einem von zwei Modi zugeordnet sein:

- **Jahrgangsmischung** (`dedicated_birth_year = NULL`): Der Kader umfasst alle Jahrgänge der Altersklasse. Mitglieder-Vorschläge und Auto-Assign filtern nach dem vollen Bracket (z.B. 2011 und 2012 für C-Jugend).
- **Dedizierter Jahrgang** (`dedicated_birth_year = <year>`): Der Kader ist einem einzelnen Geburtsjahr gewidmet. Mitglieder-Vorschläge und Auto-Assign filtern ausschließlich nach diesem Jahrgang.

Das Feld `team_number` (1 oder 2) unterscheidet mehrere Teams derselben Altersklasse und desselben Geschlechts.

In beiden Modi DARF der Benutzer manuell Spieler außerhalb des Jahrgangfilters zuweisen (Override).

#### Scenario: Vorschläge bei Jahrgangsmischung

- **WHEN** `GET /api/admin/kader/{id}/member-suggestions?filter_age_bracket=true` für einen Kader mit `dedicated_birth_year=NULL` (C-Jugend, Saison 2025/26) aufgerufen wird
- **THEN** werden nur Spieler mit Geburtsjahr 2011 oder 2012 vorgeschlagen

#### Scenario: Vorschläge bei dediziertem Jahrgang

- **WHEN** `GET /api/admin/kader/{id}/member-suggestions?filter_age_bracket=true` für einen Kader mit `dedicated_birth_year=2011` aufgerufen wird
- **THEN** werden nur Spieler mit Geburtsjahr 2011 vorgeschlagen, nicht Jahrgang 2012

#### Scenario: Manueller Override trotz Filter

- **WHEN** ein Benutzer `filter_age_bracket=false` setzt oder einen Spieler direkt via PUT zuweist
- **THEN** MUSS die Zuweisung auch für Spieler außerhalb des Jahrgangfilters funktionieren

#### Scenario: Modus ändern via PUT

- **WHEN** `PUT /api/admin/kader/{id}` mit `{"dedicated_birth_year": 2011}` gesendet wird
- **THEN** wird der Kader auf dediziert (Jahrgang 2011) umgestellt, und `birth_years` im Response zeigt [2011]

#### Scenario: Modus auf gemischt zurücksetzen

- **WHEN** `PUT /api/admin/kader/{id}` mit `{"dedicated_birth_year": null}` gesendet wird
- **THEN** wird der Kader auf Jahrgangsmischung zurückgestellt

### Requirement: Kachel zeigt Jahrgänge

Die Kachel eines Kaders auf `AdminKaderPage` MUSS die zugeordneten Jahrgänge anzeigen
(z.B. „Jg. 2011" oder „Jg. 2011/2012"), sodass der Nutzer den Filter-Scope auf einen Blick erkennt.

Der Modus-Umschalter (gemischt / dediziert) MUSS direkt auf der Kachel zugänglich sein.
Bei Wechsel auf „dediziert" MUSS der Nutzer den konkreten Jahrgang auswählen können
(Dropdown mit den beiden Jahrgängen der Altersklasse).

#### Scenario: Jahrgänge auf Kachel sichtbar

- **WHEN** der Benutzer die Kaderplanung aufruft
- **THEN** zeigt jede Kachel neben der Altersklasse die zugeordneten Jahrgänge an

#### Scenario: Modus-Umschalter in der Kachel

- **WHEN** der Benutzer den Modus-Umschalter von „gemischt" auf „dediziert" stellt
- **THEN** erscheint ein Jahrgang-Dropdown mit den beiden Jahrgängen der Altersklasse
- **THEN** nach Auswahl wird `PUT /api/admin/kader/{id}` mit `dedicated_birth_year` aufgerufen

### Requirement: Auto-Assign berücksichtigt dedizierten Jahrgang

Beim Copy-Workflow mit `member_source = "auto-assign"` MUSS `dedicated_birth_year` des
Ziel-Kaders berücksichtigt werden: ist er gesetzt, werden nur Spieler des entsprechenden
Geburtsjahrs zugewiesen; ist er NULL, wird der volle Bracket verwendet.

#### Scenario: Auto-Assign für dedizierten Kader

- **WHEN** `POST /api/admin/kader/copy-from-season` mit `member_source="auto-assign"` für einen Zielkader mit `dedicated_birth_year=2011` aufgerufen wird
- **THEN** werden nur Spieler des Jahrgangs 2011 zugewiesen, nicht 2012
