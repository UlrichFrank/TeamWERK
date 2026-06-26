# sse-kader-sync Specification

## Purpose

Diese Spezifikation beschreibt die Capability `sse-kader-sync`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Kader-Mutationen broadcasten `"kader"`-Event

Der `kader/handler.go` SHALL ein `hub *hub.EventHub`-Feld besitzen. Alle mutativen Handler-Methoden (UpdateKader, InitializeKader, DeleteKader, CopyFromSeason, AutoAssign, PatchGamesPerSeason) SHALL nach erfolgreicher Datenbankoperation `h.hub.Broadcast("kader")` aufrufen. Bei Batch-Operationen (CopyFromSeason, AutoAssign) wird genau ein Broadcast am Ende der gesamten Operation gesendet.

#### Scenario: Kader-Mitglied wird hinzugefügt oder entfernt

- **WHEN** ein Vorstand oder Trainer `PUT /api/kader/{id}` aufruft (members_add oder members_remove)
- **THEN** erhalten alle verbundenen SSE-Clients `data: kader`

#### Scenario: Kader wird für neue Saison angelegt

- **WHEN** ein Vorstand `POST /api/kader` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: kader`

#### Scenario: Kader wird gelöscht

- **WHEN** ein Vorstand `DELETE /api/kader/{id}` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: kader`

#### Scenario: Kader wird aus Vorsaison kopiert

- **WHEN** ein Vorstand `POST /api/kader/copy-from-season` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients genau ein `data: kader` (nicht eines pro Kader)

#### Scenario: Auto-Assign

- **WHEN** ein Vorstand `POST /api/kader/auto-assign` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients genau ein `data: kader`

### Requirement: AdminKaderPage und MeinTeamPage reagieren auf `"kader"`-Event

`AdminKaderPage` und `MeinTeamPage` SHALL `useLiveUpdates` einbinden und bei `event === "kader"` die Kader-Daten still neu laden.

#### Scenario: Andere Vorstandsperson ändert Kader-Zuweisung

- **WHEN** Nutzer A einen Kader auf AdminKaderPage bearbeitet
- **THEN** lädt Nutzer B's AdminKaderPage die Daten neu ohne sichtbaren Ladespinner
