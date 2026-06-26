# test-standard-rule Specification

## Purpose

Diese Spezifikation beschreibt die Capability `test-standard-rule`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: make coverage Target
Das Projekt SHALL ein `make coverage`-Target bereitstellen, das den Coverage-Report lokal erzeugt.

#### Scenario: Coverage-Report erzeugen
- **WHEN** Entwickler führt `make coverage` aus
- **THEN** Alle Tests laufen durch, Package-Coverage wird auf stdout ausgegeben, HTML-Report wird nach `/tmp/teamwerk-coverage.html` geschrieben

### Requirement: Test-Standard in Projektkonvention
Jede neue HTTP-Route in einem OpenSpec-Change SHALL mindestens einen Happy-Path-Test und einen Fehlerfall-Test erhalten. Tests MÜSSEN fachliche Invarianten prüfen, keine Coverage-Dummy-Assertions.

#### Scenario: Neue Route ohne Tests
- **WHEN** OpenSpec-Proposal enthält eine neue HTTP-Route ohne Test-Anforderungen-Abschnitt
- **THEN** Der Proposal gilt als unvollständig — Test-Anforderungen MÜSSEN vor apply ergänzt werden

#### Scenario: Test-Anforderungen in Proposal
- **WHEN** Entwickler schreibt einen OpenSpec-Proposal für eine neue Route
- **THEN** Proposal enthält Abschnitt „## Test-Anforderungen" mit mindestens einem Szenario pro neuer Route
