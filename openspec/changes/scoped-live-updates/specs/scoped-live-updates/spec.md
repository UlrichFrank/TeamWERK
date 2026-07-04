## ADDED Requirements

### Requirement: Adressierter Domänen-Event-Versand

Das System SHALL Domänen-Live-Events (bisher global via `EventHub.Broadcast`) nur an die Clients der Nutzer zustellen, die von der Änderung betroffen sind oder die zugehörige Ressource unter den bestehenden Auth-/Sichtbarkeitsregeln lesen dürfen. Der `/api/events`-Stream SHALL dazu pro Nutzer abonniert werden (`SubscribeUser`), sodass Events adressierbar sind.

#### Scenario: Event erreicht nur Zielnutzer

- **WHEN** ein Domänen-Event an eine Menge von Nutzer-IDs gesendet wird
- **THEN** empfangen ausschließlich die `/api/events`-Streams dieser Nutzer das Event
- **AND** die Streams anderer Nutzer empfangen es nicht

#### Scenario: Mitglieder-Event nur an Finance-Gruppe

- **WHEN** eine Mitglieder- oder Nutzer-Mutation ein `members`- bzw. `users`-Event auslöst
- **THEN** empfangen es die Streams von Nutzern mit Vereinsfunktion `vorstand`, `vorstand_beisitzer` oder `kassierer` sowie Admins
- **AND** der Stream eines reinen Spielers ohne diese Funktion empfängt es nicht

#### Scenario: Spiel-Event team-gescopet

- **WHEN** eine Spiel-Mutation ein `games`-Event auslöst
- **THEN** empfangen es die Streams der Mitglieder der beteiligten Teams sowie der zuständigen Trainer/sportlichen Leitung und des Vorstands
- **AND** der Stream eines teamfremden Spielers empfängt es nicht

### Requirement: Vereinsweite Topics bleiben global

Das System SHALL explizit als vereinsweit klassifizierte Topics (`venues`, `settings`, `beitragssatz-changed`, `stammvereine`) weiterhin an alle verbundenen Clients senden.

#### Scenario: Einstellungs-Event erreicht alle

- **WHEN** eine Vereins-Einstellung geändert wird und ein `settings`-Event auslöst
- **THEN** empfangen alle verbundenen `/api/events`-Streams das Event

### Requirement: Scoping ändert die Datensichtbarkeit nicht

Das System SHALL durch das Event-Scoping die Autorisierungs- und Sichtbarkeitsregeln der Lese-Routen NICHT verändern. Das Scoping bestimmt ausschließlich, welcher Client zum Nachladen aufgefordert wird; welche Daten der Client daraufhin tatsächlich erhält, entscheiden unverändert die Lese-Routen.

#### Scenario: Lese-Route bleibt autoritativ

- **WHEN** ein Client ein Live-Event empfängt und daraufhin die zugehörige Lese-Route abruft
- **THEN** erhält er genau die Daten, die ihm die Lese-Route unter den bestehenden Auth-Regeln liefert — Scoping gewährt keinen zusätzlichen Zugriff

#### Scenario: Konservatives Scoping bei Mehrdeutigkeit

- **WHEN** die betroffene Nutzergruppe für ein Event nicht eindeutig eingrenzbar ist
- **THEN** SHALL das System breiter senden (bis hin zu global), statt einen legitimen Empfänger auszulassen
