## ADDED Requirements

### Requirement: Auto-Duty-Regeneration ist verhaltenserhaltend unter der Komplexitätsschwelle zerlegt
`games.regenSingleDay` SHALL in benannte Funktionen (`loadDayGames`, `snapshotDeletedSlots`,
`snapshotCustomSlots`, `regenGameItems`, `buildNotificationIntents`) zerlegt sein, sodass die
Funktionen die Komplexitäts-Schwellen aus `metrics/thresholds.yml` einhalten. Die Zerlegung SHALL
kein beobachtbares Verhalten der Regeneration ändern (Slots, `regen_summary`, Notifications).

#### Scenario: Regen-Charakterisierungssuite bleibt grün
- **WHEN** ein Extract-Schritt durchgeführt wird
- **THEN** `go test ./internal/games/` SHALL ohne Änderung an den Regen-Charakterisierungstests grün bleiben

#### Scenario: regen_summary unverändert
- **WHEN** ein Spiel angelegt/geändert/gelöscht wird
- **THEN** die `regen_summary`-Struktur und ihre Felder (created/reduced/skipped/conflicts/notified_users) SHALL identisch zum Verhalten vor dem Refactor sein

#### Scenario: Komplexität unter der Gate-Schwelle
- **WHEN** `make metrics-gate` nach dem Refactor läuft
- **THEN** `regenSingleDay` SHALL die konfigurierten gocognit-/gocyclo-Schwellen einhalten
