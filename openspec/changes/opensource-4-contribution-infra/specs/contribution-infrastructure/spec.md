## ADDED Requirements

### Requirement: Beitrags- und Verhaltens-Leitfäden
Der Repository MUST `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` und `SECURITY.md` enthalten. `CONTRIBUTING.md` MUST den OpenSpec-Workflow, Conventional Commits, die Hard Rules und das lokale Test-/Lint-Gate beschreiben.

#### Scenario: Erstbeitrag ist nachvollziehbar
- **WHEN** eine externe Person beitragen möchte
- **THEN** findet sie in `CONTRIBUTING.md` Setup, Workflow und Qualitäts-Gate
- **AND** `SECURITY.md` nennt einen privaten Meldeweg für Schwachstellen (kein öffentliches Issue)

### Requirement: Öffentliche CI spiegelt das pre-push-Gate
Es MUST einen CI-Workflow geben, der bei Push/PR `go vet`, `go test -race ./...` (inkl. Architektur-Test), `golangci-lint` sowie `pnpm -C web build/test/lint` und `openspec validate` ausführt.

#### Scenario: Roter PR wird nicht grün
- **WHEN** ein PR einen fehlschlagenden Test, Lint-Fehler oder ungültige OpenSpec-Änderung enthält
- **THEN** schlägt die CI fehl und markiert den PR als nicht mergebar

#### Scenario: Sauberer Checkout läuft grün
- **WHEN** die CI auf einem sauberen Checkout des Default-Branch läuft
- **THEN** durchläuft sie alle Gates erfolgreich

### Requirement: Self-Hosting-Anleitung
Es MUST eine `docs/SELF_HOSTING.md` geben, mit der ein fremder Verein eine eigene Instanz aufsetzen kann: VPS-Setup, vollständige ENV-Referenz, Migrationen, Backup (inkl. Beitragslauf-Protokoll-Verzeichnis) und Web-Push/VAPID-Einrichtung.

#### Scenario: Fremder Verein kann deployen
- **WHEN** ein technisch versierter Vereinsadmin der Anleitung folgt
- **THEN** kann er TeamWERK ohne Rückfrage an die Maintainer in Betrieb nehmen

### Requirement: AGPL-§13-Quellcode-Zugang in der App
Die laufende Anwendung MUST Nutzern Zugang zum Quellcode anbieten (Link auf das öffentliche Repository), wie von AGPL-3.0 §13 verlangt.

#### Scenario: Source-Link ist erreichbar
- **WHEN** ein eingeloggter Nutzer die App verwendet
- **THEN** ist ein Link zum öffentlichen Quellcode-Repository sichtbar (z. B. im Footer oder einer Info-Seite)
