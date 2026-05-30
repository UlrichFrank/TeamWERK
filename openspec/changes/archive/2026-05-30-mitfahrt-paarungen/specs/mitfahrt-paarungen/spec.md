## ADDED Requirements

### Requirement: Paarungsanfrage stellen (Sucher initiiert)
Ein Sucher, der einen Bieter-Eintrag sieht, SHALL eine Paarungsanfrage stellen können. Dabei gibt er an, wie viele Plätze sein Gesuch umfasst. Die Anfrage landet mit Status `pending` beim Bieter.

#### Scenario: Sucher stellt Anfrage an Bieter
- **WHEN** ein Sucher `POST /api/mitfahrt-paarungen` mit `sucheId` und `bieteId` sendet
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='suche'` angelegt und der Bieter erhält eine Push-Benachrichtigung

#### Scenario: Anfrage bei unzureichender Kapazität abgewiesen
- **WHEN** der Bieter-Eintrag weniger freie Plätze hat als das Gesuch benötigt — wobei bereits bestehende `pending`- und `confirmed`-Paarungen auf die Kapazität angerechnet werden
- **THEN** antwortet die API mit 409 Conflict und die Paarung wird nicht angelegt

#### Scenario: Sucher hat bereits eine confirmed Paarung für dieses Gesuch
- **WHEN** für die `suche_id` bereits eine Paarung mit `status='confirmed'` existiert
- **THEN** antwortet die API mit 409 Conflict

### Requirement: Paarungsanfrage stellen (Bieter initiiert)
Ein Bieter SHALL einen Sucher aktiv zur Mitfahrt einladen können. Auch hier entsteht eine `pending`-Paarung — der Sucher muss bestätigen.

#### Scenario: Bieter lädt Sucher ein
- **WHEN** ein Bieter `POST /api/mitfahrt-paarungen` mit `bieteId` (eigener Eintrag) und `sucheId` sendet
- **THEN** wird eine Paarung mit `status='pending'` und `initiiert_von='biete'` angelegt und der Sucher erhält eine Push-Benachrichtigung

#### Scenario: Bieter versucht fremden Bieter-Eintrag zu nutzen
- **WHEN** `bieteId` gehört nicht dem authentifizierten Nutzer und der Nutzer ist auch nicht Sucher dieser Paarung
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Paarungsanfrage bestätigen
Die Gegenseite (Bieter oder Sucher, je nach Initiator) SHALL eine offene Anfrage bestätigen können.

#### Scenario: Bieter bestätigt Anfrage eines Suchers
- **WHEN** der Bieter `POST /api/mitfahrt-paarungen/{id}/confirm` aufruft
- **THEN** wird `status='confirmed'` gesetzt und der Sucher erhält eine Push-Benachrichtigung

#### Scenario: Sucher bestätigt Angebot eines Bieters
- **WHEN** der Sucher `POST /api/mitfahrt-paarungen/{id}/confirm` aufruft
- **THEN** wird `status='confirmed'` gesetzt und der Bieter erhält eine Push-Benachrichtigung

#### Scenario: Bestätigung bei voller Kapazität
- **WHEN** der Bieter-Eintrag bereits `plaetze` Plätze in confirmed Paarungen belegt hat
- **THEN** antwortet die API mit 409 Conflict

#### Scenario: Bestätigung durch falsche Partei
- **WHEN** der Initiator versucht, seine eigene Anfrage zu bestätigen (statt die Gegenseite)
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Paarungsanfrage ablehnen
Jede Seite SHALL eine offene oder bestätigte Paarung ablehnen bzw. stornieren können.

#### Scenario: Anfrage ablehnen (pending)
- **WHEN** Bieter oder Sucher `POST /api/mitfahrt-paarungen/{id}/reject` aufruft bei einer `pending`-Paarung
- **THEN** wird `status='rejected'` gesetzt und die Gegenseite erhält eine Push-Benachrichtigung

#### Scenario: Bestätigte Paarung stornieren
- **WHEN** Bieter oder Sucher `POST /api/mitfahrt-paarungen/{id}/reject` aufruft bei einer `confirmed`-Paarung
- **THEN** wird `status='rejected'` gesetzt und die Gegenseite erhält eine Push-Benachrichtigung über die Stornierung


### Requirement: Paarungen im Board anzeigen
Bestätigte und offene Paarungen MUST für alle authentifizierten Nutzer im Board sichtbar sein.

#### Scenario: Paarungen in der List-Antwort
- **WHEN** `GET /api/mitfahrgelegenheiten` aufgerufen wird
- **THEN** enthält die Antwort pro Spiel ein `paarungen`-Array mit Bieter-Name, Sucher-Name, Anzahl (suche.plaetze) und Status (`pending` / `confirmed`)

#### Scenario: Rejected Paarungen ausgeblendet
- **WHEN** eine Paarung den Status `rejected` hat
- **THEN** erscheint sie nicht im `paarungen`-Array

### Requirement: Eigene Paarungsanfragen einsehen
Jeder Nutzer MUST sehen können, welche Paarungsanfragen ihn betreffen (als Bieter oder Sucher), inklusive Status.

#### Scenario: Eigene pending-Anfragen erkennbar
- **WHEN** ein Nutzer das Board lädt
- **THEN** sind Paarungen, die ihn betreffen, mit einem `isOwn`-Flag oder Nutzer-Kontext markiert, sodass das Frontend Aktions-Buttons (Bestätigen/Ablehnen) anzeigen kann
