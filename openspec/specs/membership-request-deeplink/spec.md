## ADDED Requirements

### Requirement: Push-Notification verlinkt direkt auf den Beitrittsantrag
Wenn eine neue Beitrittsanfrage eingeht, SHALL die Push-Notification-URL die ID des neu erstellten Antrags enthalten.
Format: `/admin/mitgliedschaft?id={id}`

Das Backend MUSS nach dem INSERT die `LastInsertId()` auslesen und in der Notification-URL verwenden.

#### Scenario: Neue Anfrage erzeugt Notification mit korrekter ID
- **WHEN** `POST /api/auth/request-membership` erfolgreich ausgeführt wird
- **THEN** erhalten alle Admin-User eine Push-Notification mit URL `/admin/mitgliedschaft?id={newId}`

#### Scenario: ID im URL entspricht dem neu erstellten Datensatz
- **WHEN** eine Push-Notification mit `?id=42` empfangen wird
- **THEN** existiert in `membership_requests` ein Datensatz mit `id=42`

### Requirement: MembershipRequestsPage scrollt zu und hebt spezifischen Antrag hervor
Die MembershipRequestsPage SHALL beim Laden den URL-Parameter `id` auslesen.
Wenn `?id=<n>` gesetzt ist, MUSS:
1. Zur entsprechenden Antragskarte gescrollt werden
2. Die Karte visuell für ca. 2 Sekunden hervorgehoben werden (z.B. ring/outline)

#### Scenario: Seite öffnet sich direkt auf dem markierten Antrag
- **WHEN** die Seite mit `?id=42` geöffnet wird und Antrag 42 existiert
- **THEN** scrollt die Seite nach dem Laden zur Karte mit ID 42
- **AND** die Karte ist für ca. 2 Sekunden visuell hervorgehoben

#### Scenario: Kein Effekt bei fehlendem oder unbekanntem id-Param
- **WHEN** die Seite ohne `?id` oder mit einer ID geöffnet wird, die nicht in der Liste ist
- **THEN** verhält sich die Seite wie bisher ohne Highlight
