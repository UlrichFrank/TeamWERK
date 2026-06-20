## MODIFIED Requirements

### Requirement: Aktiv-Kategorie aus Stammverein-Zuordnung
Der Beitragslauf MUST die Aktiv-Kategorie eines Mitglieds **deterministisch** aus `members.home_club_id` ableiten: ist ein Stammverein zugeordnet (`home_club_id IS NOT NULL`) → Kategorie `aktiv_mit`; sonst → `aktiv_ohne`. Es MUST **kein** Fuzzy-/Freitext-Abgleich (`MatchHomeClub`) mehr stattfinden, und es MUST keine `home_club_unklar`-Warnung mehr erzeugt werden. „Kein Stammverein" (`NULL`) ist ein gültiger Zustand und führt regulär zu `aktiv_ohne`.

#### Scenario: Mitglied mit zugeordnetem Stammverein
- **WHEN** ein aktives Mitglied mit gesetztem `home_club_id` im Lauf verarbeitet wird
- **THEN** wird es der Kategorie `aktiv_mit` zugeordnet (96 €) — unabhängig von Schreibweise, da keine Textzuordnung mehr erfolgt

#### Scenario: Mitglied ohne Stammverein
- **WHEN** ein aktives Mitglied mit `home_club_id = NULL` im Lauf verarbeitet wird
- **THEN** wird es der Kategorie `aktiv_ohne` zugeordnet (226 €), ohne Warnung

#### Scenario: Keine Fuzzy-Warnung mehr
- **WHEN** der Lauf-Preview für aktive Mitglieder erzeugt wird
- **THEN** enthält kein Mitglied die Warnung `home_club_unklar`
