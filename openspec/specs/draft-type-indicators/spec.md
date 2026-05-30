## ADDED Requirements

### Requirement: Typ-spezifische Pending-Indikatoren in Mitgliederliste

Das System SHALL in der Mitgliederliste (Admin-Ansicht) für jedes Mitglied separat anzeigen, ob ein ausstehender Persönliche-Daten-Draft (`profil`) und/oder ein ausstehender Bankdaten-Draft (`bankdaten`) vorliegt.

#### Scenario: Mitglied hat ausstehenden profil-Draft

- **WHEN** ein Admin die Mitgliederliste aufruft und ein Mitglied hat einen Draft mit `field_name='profil'`
- **THEN** wird neben dem Namen ein `User`-Icon (Lucide) angezeigt

#### Scenario: Mitglied hat ausstehenden bankdaten-Draft

- **WHEN** ein Admin die Mitgliederliste aufruft und ein Mitglied hat einen Draft mit `field_name='bankdaten'`
- **THEN** wird neben dem Namen ein `CreditCard`-Icon (Lucide) angezeigt

#### Scenario: Mitglied hat beide Draft-Typen

- **WHEN** ein Admin die Mitgliederliste aufruft und ein Mitglied hat sowohl einen `profil`- als auch einen `bankdaten`-Draft
- **THEN** werden beide Icons nebeneinander angezeigt

#### Scenario: Mitglied hat keine ausstehenden Drafts

- **WHEN** ein Admin die Mitgliederliste aufruft und ein Mitglied hat keine offenen Drafts
- **THEN** werden keine Icons angezeigt

#### Scenario: Nicht-Admin sieht keine Icons

- **WHEN** ein Nutzer ohne Admin-Rolle die Mitgliederliste aufruft
- **THEN** werden keine Draft-Typ-Indikatoren angezeigt

### Requirement: Backend liefert getrennte Draft-Typ-Flags

Das Backend SHALL im `GET /api/members`-Response pro Mitglied die Felder `has_pending_profil_draft: bool` und `has_pending_bank_draft: bool` zurückgeben (nur wenn Admin).

#### Scenario: Korrekte Flag-Werte bei gemischten Drafts

- **WHEN** ein Mitglied einen `profil`-Draft hat aber keinen `bankdaten`-Draft
- **THEN** ist `has_pending_profil_draft=true` und `has_pending_bank_draft=false`

#### Scenario: Kein Flag ohne Drafts

- **WHEN** ein Mitglied keine offenen Drafts hat
- **THEN** sind beide Flags `false` (oder werden im JSON weggelassen per `omitempty`)
