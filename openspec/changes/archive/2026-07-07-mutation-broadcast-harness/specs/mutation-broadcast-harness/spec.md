## ADDED Requirements

### Requirement: Mutations-Routen erzwingen einen Broadcast-Aufruf

Das System SHALL einen stdlib-basierten Test bereitstellen (Teil von `make test` und des `pre-push`-Gates), der jede in `internal/app/router.go` (`BuildRouter`) registrierte mutierende Route (`POST`/`PUT`/`PATCH`/`DELETE`) daraufhin prüft, dass der zugehörige Handler-Rumpf mindestens einen Broadcast-Aufruf enthält (ein `CallExpr`, dessen Bezeichner die Teilzeichenkette `Broadcast` enthält — inklusive Helfer wie `broadcastMembers`). Fehlt ein solcher Aufruf und steht die Route nicht auf der Allowlist, SHALL der Test fehlschlagen und Package, Methode und Route benennen.

#### Scenario: Mutation ohne Broadcast schlägt fehl

- **WHEN** eine `POST`/`PUT`/`PATCH`/`DELETE`-Route auf einen Handler zeigt, dessen Rumpf keinen Broadcast-artigen Aufruf enthält
- **AND** die Route nicht auf der Allowlist steht
- **THEN** schlägt der Harness-Test fehl
- **AND** die Fehlermeldung nennt Package, Methode und Route

#### Scenario: Mutation mit direktem oder Helfer-Broadcast besteht

- **WHEN** der Handler `h.hub.Broadcast(...)`, `h.hub.BroadcastToUsers(...)` oder einen Helfer mit `Broadcast` im Namen aufruft
- **THEN** gilt die Route als konform und der Test besteht

#### Scenario: Nicht-mutierende Routen werden ignoriert

- **WHEN** eine Route mit `GET` (oder `HEAD`/`OPTIONS`) registriert ist
- **THEN** wird sie vom Harness-Test nicht auf Broadcast geprüft

### Requirement: Ausnahmen nur über eine explizite, gepflegte Allowlist

Das System SHALL Broadcast-freie Mutations-Routen ausschließlich über eine explizite Allowlist mit Begründung je Eintrag zulassen. Ein Allowlist-Eintrag, der auf keine real registrierte Route mehr zeigt, SHALL einen Testfehler erzeugen, damit die Liste nicht verrottet.

#### Scenario: Allowlist-Eintrag deckt eine bewusste Ausnahme

- **WHEN** eine mutierende Route ohne Broadcast auf der Allowlist steht (z. B. `POST /api/auth/refresh`)
- **THEN** besteht der Test für diese Route

#### Scenario: Verwaister Allowlist-Eintrag schlägt fehl

- **WHEN** ein Allowlist-Eintrag keiner in `BuildRouter` registrierten Route (mehr) entspricht
- **THEN** schlägt der Harness-Test fehl und benennt den toten Eintrag
