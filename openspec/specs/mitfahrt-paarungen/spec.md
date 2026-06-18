## MODIFIED Requirements

### Requirement: Paarungsanfrage stellen (Sucher initiiert)
Ein Sucher, der einen Bieter-Eintrag sieht, SHALL eine Paarungsanfrage stellen können. Ein Elternteil gilt als berechtigt, wenn der Suche-Eintrag (`sucheId`) ihm selbst oder einem seiner Kinder gehört.

#### Scenario: Sucher stellt Anfrage an Bieter
- **WHEN** ein Sucher `POST /api/mitfahrt-paarungen` mit `sucheId` (eigener Eintrag) und `bieteId` sendet
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='suche'` angelegt

#### Scenario: Elternteil stellt Anfrage für Kind (Kind sucht)
- **WHEN** ein Elternteil `POST /api/mitfahrt-paarungen` sendet und `sucheId` einem seiner Kinder gehört
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='suche'` angelegt

#### Scenario: Anfrage bei unzureichender Kapazität abgewiesen
- **WHEN** der Bieter-Eintrag weniger freie Plätze hat als das Gesuch benötigt
- **THEN** antwortet die API mit 409 Conflict

#### Scenario: Sucher hat bereits eine confirmed Paarung für dieses Gesuch
- **WHEN** für die `suche_id` bereits eine Paarung mit `status='confirmed'` existiert
- **THEN** antwortet die API mit 409 Conflict

### Requirement: Paarungsanfrage stellen (Bieter initiiert)
Ein Bieter SHALL einen Sucher aktiv zur Mitfahrt einladen können. Ein Elternteil gilt als berechtigt, wenn der Biete-Eintrag (`bieteId`) ihm selbst oder einem seiner Kinder gehört.

#### Scenario: Bieter lädt Sucher ein
- **WHEN** ein Bieter `POST /api/mitfahrt-paarungen` mit `bieteId` (eigener Eintrag) und `sucheId` sendet
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='biete'` angelegt

#### Scenario: Elternteil lädt Sucher ein (Kind bietet)
- **WHEN** ein Elternteil `POST /api/mitfahrt-paarungen` sendet und `bieteId` einem seiner Kinder gehört
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='biete'` angelegt

#### Scenario: Kein Bezug zu eigenem oder Kind-Eintrag
- **WHEN** weder `bieteId` noch `sucheId` dem eingeloggten Nutzer oder einem seiner Kinder gehört
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Paarungsanfrage bestätigen
Die Gegenseite (oder ein Elternteil der Gegenseite) SHALL eine offene Anfrage bestätigen können.

#### Scenario: Bieter bestätigt Anfrage eines Suchers
- **WHEN** der Bieter `POST /api/mitfahrt-paarungen/{id}/confirm` aufruft
- **THEN** wird `status='confirmed'` gesetzt

#### Scenario: Elternteil bestätigt für Kind (Kind ist Gegenseite)
- **WHEN** ein Elternteil `confirm` aufruft und ein Kind die bestätigende Seite wäre
- **THEN** wird `status='confirmed'` gesetzt

#### Scenario: Sucher bestätigt Angebot eines Bieters
- **WHEN** der Sucher `POST /api/mitfahrt-paarungen/{id}/confirm` aufruft
- **THEN** wird `status='confirmed'` gesetzt

#### Scenario: Bestätigung bei voller Kapazität
- **WHEN** der Bieter-Eintrag bereits alle Plätze in confirmed Paarungen belegt hat
- **THEN** antwortet die API mit 409 Conflict

#### Scenario: Bestätigung durch falsche Partei
- **WHEN** der Initiator (weder er noch sein Kind ist Gegenseite) versucht zu bestätigen
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Paarungsanfrage ablehnen
Jede beteiligte Seite (oder ein Elternteil einer beteiligten Seite) SHALL eine offene oder bestätigte Paarung ablehnen bzw. stornieren können.

#### Scenario: Anfrage ablehnen (pending)
- **WHEN** Bieter, Sucher oder ein Elternteil einer der beiden Seiten `POST /api/mitfahrt-paarungen/{id}/reject` aufruft
- **THEN** wird `status='rejected'` gesetzt

#### Scenario: Bestätigte Paarung stornieren
- **WHEN** Bieter, Sucher oder ein Elternteil `reject` für eine `confirmed`-Paarung aufruft
- **THEN** wird `status='rejected'` gesetzt

#### Scenario: Unbeteiligter kann nicht ablehnen
- **WHEN** ein Nutzer ohne Bezug (weder direkt noch via Kind) `reject` aufruft
- **THEN** antwortet die API mit 403 Forbidden
