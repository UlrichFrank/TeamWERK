## ADDED Requirements

### Requirement: AutoAssign nach DHB-Jahrgangsbracket
Das System SHALL beim AutoAssign alle aktiven Mitglieder einem Kader zuweisen, deren Geburtsjahr ins DHB-Bracket der Altersklasse fällt. Ausgetretene Mitglieder werden ausgeschlossen. Bei Kader mit `dedicated_birth_year` wird exakt nach diesem Jahrgang gefiltert.

#### Scenario: Bracket-Filter weist passende Mitglieder zu
- **WHEN** POST /api/admin/kader/auto-assign für Kader A-Jugend Saison 2025/26 mit Mitgliedern Jg. 2007 und 2005
- **THEN** Nur Jg. 2007 in kader_members (Bracket 2007–2008 für A-Jugend 2025/26)

#### Scenario: Ausgetretene werden ausgeschlossen
- **WHEN** POST /api/admin/kader/auto-assign mit 1 aktivem und 1 ausgetretenem Mitglied im Bracket
- **THEN** Nur das aktive Mitglied wird zugewiesen

#### Scenario: dedicated_birth_year filtert exakt
- **WHEN** POST /api/admin/kader/auto-assign für Kader mit dedicated_birth_year=2008, Mitglieder Jg. 2007/2008/2009
- **THEN** Nur Jg. 2008 wird zugewiesen

### Requirement: MemberSuggestions mit optionalem Bracket-Filter
Das System SHALL bei MemberSuggestions standardmäßig nach Jahrgangsbracket filtern. Mit `?filter_age_bracket=false` MÜSSEN alle aktiven Mitglieder unabhängig vom Bracket vorgeschlagen werden.

#### Scenario: Bracket-Filter aktiv (default)
- **WHEN** GET /api/admin/kader/{id}/member-suggestions für A-Jugend mit 1 Mitglied im Bracket und 1 außerhalb
- **THEN** Nur das Mitglied im Bracket in suggestions

#### Scenario: Bracket-Filter deaktiviert
- **WHEN** GET /api/admin/kader/{id}/member-suggestions?filter_age_bracket=false
- **THEN** Beide Mitglieder in suggestions
