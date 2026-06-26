# code-metrics Specification

## Purpose

Diese Spezifikation beschreibt die Capability `code-metrics`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Metrik-Reporting via `make metrics`

Das System SHALL ein Make-Target `make metrics` bereitstellen, das Code-Kennzahlen für Go-Backend und TS/React-Frontend erhebt und ausgibt. Der Lauf SHALL ausschließlich berichten und MUST sich unter allen Umständen mit Exit-Code 0 beenden — auch wenn Komplexitäts-, Duplikations- oder Lint-Schwellwerte überschritten sind.

#### Scenario: Erfolgreicher Metrik-Lauf
- **WHEN** ein Entwickler `make metrics` in einem Arbeitsverzeichnis mit installierten Tools ausführt
- **THEN** gibt das System eine kompakte Kennzahlen-Tabelle auf stdout aus
- **AND** schreibt einen Markdown-Report nach `metrics/REPORT.md`
- **AND** beendet sich mit Exit-Code 0

#### Scenario: Überschrittene Schwellwerte blockieren den Default-Lauf nicht
- **WHEN** `make metrics` läuft und Funktionen oberhalb der Komplexitäts-Schwellwerte existieren
- **THEN** weist der Report diese Funktionen als Hotspots aus
- **AND** beendet sich der Lauf dennoch mit Exit-Code 0

### Requirement: Erhobene Kennzahlen

Der Metrik-Report SHALL mindestens folgende Kennzahlen enthalten: LOC pro Sprache, Prod-zu-Test-LOC-Verhältnis, Kommentar-Anteil, zyklomatische und kognitive Komplexität für Go, Funktionslängen-Verstöße, Test-Coverage für Go und Frontend, Lint-Issue-Dichte pro kLOC sowie Code-Duplikation für Go und Frontend. Der Report SHALL die komplexesten Go-Funktionen als Hotspot-Liste ausweisen.

#### Scenario: Report enthält alle Kennzahlen-Kategorien
- **WHEN** `metrics/REPORT.md` nach einem Lauf gelesen wird
- **THEN** enthält er Abschnitte für Größe, Komplexität, Coverage, Lint-Dichte und Duplikation
- **AND** enthält er eine nach Komplexität sortierte Liste der Top-Go-Funktionen

#### Scenario: Frontend- und Backend-Coverage getrennt ausgewiesen
- **WHEN** der Report gelesen wird
- **THEN** sind Go-Coverage und Frontend-Coverage als getrennte Werte ausgewiesen

### Requirement: Trennung vom blockierenden Lint-Gate

Die Erhebung der Go-Komplexitäts- und Duplikations-Kennzahlen SHALL über eine separate Konfiguration (`.golangci.metrics.yml`) mit `--issues-exit-code 0` erfolgen. Die bestehende `.golangci.yml` (blockierendes Gate in `make lint`/pre-push) MUST unverändert bleiben.

#### Scenario: Bestehendes Lint-Gate bleibt unverändert
- **WHEN** der Change implementiert ist
- **THEN** ist `.golangci.yml` inhaltlich unverändert
- **AND** verwendet `make metrics` eine separate `.golangci.metrics.yml`

### Requirement: Optionales Schwellwert-Gate via `make metrics-gate`

Das System SHALL ein separates Make-Target `make metrics-gate` bereitstellen, das die erhobenen Kennzahlen gegen in `metrics/thresholds.yml` konfigurierte Schwellwerte prüft. Bei Überschreitung eines Schwellwerts MUST sich dieses Target mit Exit-Code 1 beenden; andernfalls mit Exit-Code 0.

#### Scenario: Gate failt bei Regression
- **WHEN** `make metrics-gate` läuft und eine Kennzahl ihren konfigurierten Schwellwert verletzt
- **THEN** gibt das System die verletzten Schwellwerte aus
- **AND** beendet sich mit Exit-Code 1

#### Scenario: Gate ist grün innerhalb der Schwellwerte
- **WHEN** `make metrics-gate` läuft und alle Kennzahlen innerhalb der Schwellwerte liegen
- **THEN** beendet sich das System mit Exit-Code 0

### Requirement: Reproduzierbare Tool-Verankerung

Externe Werkzeuge SHALL über das jeweilige Ökosystem-Manifest gepinnt werden: `jscpd` als pnpm-devDependency, `scc` als `go.mod`-`tool`-Direktive. Das System MUST NOT auf `npm` oder ein globales `go install` angewiesen sein. Fehlt ein benötigtes Werkzeug, SHALL der Lauf mit einer klaren Hinweismeldung samt Installationsbefehl abbrechen.

#### Scenario: Fehlendes Werkzeug liefert klaren Hinweis
- **WHEN** `make metrics` ausgeführt wird und ein benötigtes Werkzeug nicht verfügbar ist
- **THEN** gibt das System eine Fehlermeldung mit dem konkreten Installationsbefehl aus

#### Scenario: Generierter Report ist nicht versioniert
- **WHEN** nach einem Lauf der Git-Status geprüft wird
- **THEN** taucht `metrics/REPORT.md` nicht als unversionierte Änderung auf (via `.gitignore`)
