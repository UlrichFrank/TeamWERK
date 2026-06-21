## Purpose

Mitfahrt-Paarungen verknüpfen Fahrangebote (Biete) mit Mitfahrgesuchen (Suche) zu einem Spiel: anfragen, bestätigen, ablehnen — inklusive Eltern-für-Kind-Vertretung und einseitiger One-Click-Paarung mit implizit angelegtem Spiegel-Eintrag.
## Requirements
### Requirement: Paarungsanfrage stellen (Sucher initiiert)
Ein Sucher, der einen Bieter-Eintrag sieht, SHALL eine Paarungsanfrage stellen können — auch ohne vorab einen eigenen Suche-Eintrag zu besitzen. Bei einem einseitigen Request (`bieteId` ohne `sucheId`) legt das Backend den Suche-Spiegel-Eintrag in derselben Transaktion an (get-or-create). Ein Elternteil gilt als berechtigt, wenn der Suche-Eintrag (`sucheId`) bzw. der angegebene `forUserId` ihm selbst oder einem seiner Kinder gehört.

#### Scenario: Sucher stellt Anfrage an Bieter
- **WHEN** ein Sucher `POST /api/mitfahrt-paarungen` mit `sucheId` (eigener Eintrag) und `bieteId` sendet
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='suche'` angelegt

#### Scenario: Elternteil stellt Anfrage für Kind (Kind sucht)
- **WHEN** ein Elternteil `POST /api/mitfahrt-paarungen` sendet und `sucheId` einem seiner Kinder gehört
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='suche'` angelegt

#### Scenario: Sucher fragt ohne eigenen Eintrag an (einseitig)
- **WHEN** ein Nutzer `POST /api/mitfahrt-paarungen` mit nur `bieteId` (und optional `plaetze`) sendet und für dieses Spiel noch keinen Suche-Eintrag besitzt
- **THEN** wird ein Suche-Eintrag für den Nutzer (`plaetze` oder Default 1) angelegt und eine Paarung mit `status='pending'` und `initiiert_von='suche'` erstellt

#### Scenario: Elternteil fragt für Kind ohne Eintrag an (einseitig)
- **WHEN** ein Elternteil `POST /api/mitfahrt-paarungen` mit `bieteId` und `forUserId` eines seiner Kinder sendet und das Kind noch keinen Suche-Eintrag für dieses Spiel hat
- **THEN** wird ein Suche-Eintrag für das Kind angelegt und eine Paarung mit `initiiert_von='suche'` erstellt

#### Scenario: Vorhandener Suche-Eintrag wird wiederverwendet
- **WHEN** ein Nutzer einen einseitigen Request mit `bieteId` sendet und bereits einen Suche-Eintrag ohne aktive Paarung für dieses Spiel besitzt
- **THEN** wird dieser bestehende Eintrag wiederverwendet statt ein zweiter angelegt

#### Scenario: forUserId ohne Bezug zum Nutzer
- **WHEN** ein einseitiger Request `forUserId` enthält, der weder der eingeloggte Nutzer noch eines seiner Kinder ist
- **THEN** antwortet die API mit 403 Forbidden und es wird kein Eintrag angelegt

#### Scenario: Anfrage bei unzureichender Kapazität abgewiesen
- **WHEN** der Bieter-Eintrag weniger freie Plätze hat als das Gesuch benötigt
- **THEN** antwortet die API mit 409 Conflict und es wird kein Spiegel-Eintrag persistiert

#### Scenario: Sucher hat bereits eine confirmed Paarung für dieses Gesuch
- **WHEN** für die `suche_id` bereits eine Paarung mit `status='confirmed'` existiert
- **THEN** antwortet die API mit 409 Conflict

### Requirement: Paarungsanfrage stellen (Bieter initiiert)
Ein Bieter SHALL einen Sucher aktiv zur Mitfahrt einladen können — auch ohne vorab einen eigenen Biete-Eintrag zu besitzen. Bei einem einseitigen Request (`sucheId` ohne `bieteId`) legt das Backend den Biete-Spiegel-Eintrag in derselben Transaktion an (get-or-create), stets für den eingeloggten Nutzer. Ein Elternteil gilt als berechtigt, wenn der Biete-Eintrag (`bieteId`) ihm selbst oder einem seiner Kinder gehört; der einseitige Biete-Pfad legt den Eintrag jedoch ausschließlich für den eingeloggten Nutzer selbst an (kein `forUserId`).

#### Scenario: Bieter lädt Sucher ein
- **WHEN** ein Bieter `POST /api/mitfahrt-paarungen` mit `bieteId` (eigener Eintrag) und `sucheId` sendet
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='biete'` angelegt

#### Scenario: Elternteil lädt Sucher ein (Kind bietet)
- **WHEN** ein Elternteil `POST /api/mitfahrt-paarungen` sendet und `bieteId` einem seiner Kinder gehört
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='biete'` angelegt

#### Scenario: Bieter bietet ohne eigenen Eintrag an (einseitig)
- **WHEN** ein Nutzer `POST /api/mitfahrt-paarungen` mit nur `sucheId` (und optional `plaetze`) sendet und für dieses Spiel noch keinen Biete-Eintrag besitzt
- **THEN** wird ein Biete-Eintrag für den Nutzer angelegt und eine Paarung mit `status='pending'` und `initiiert_von='biete'` erstellt

#### Scenario: Vorhandener Biete-Eintrag wird wiederverwendet
- **WHEN** ein Nutzer einen einseitigen Request mit `sucheId` sendet und bereits einen Biete-Eintrag für dieses Spiel besitzt
- **THEN** wird dieser bestehende Biete-Eintrag wiederverwendet (Unique-Index `(game_id,user_id)`)

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

