# test-trainings-gaps Specification

## Purpose

Diese Spezifikation beschreibt die Capability `test-trainings-gaps`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Einzelsitzung anlegen und bearbeiten
Das System SHALL Trainern ermöglichen, eine einzelne Trainingssession anzulegen und zu bearbeiten. Nur Trainer des zugehörigen Teams oder Admins dürfen schreiben.

#### Scenario: Einzelsitzung anlegen (Admin)
- **WHEN** Admin POST /api/training-sessions mit team_id, season_id, date, start_time, end_time
- **THEN** HTTP 201, Session in DB angelegt

#### Scenario: Einzelsitzung bearbeiten
- **WHEN** Trainer PUT /api/training-sessions/{id} mit geänderten Feldern
- **THEN** HTTP 204, Änderung in DB persistiert

### Requirement: Trainingsserie löschen mit Cascade
Das System SHALL beim Löschen einer Trainingsserie alle zugehörigen Sessions und Antworten (training_responses) kaskadierend löschen.

#### Scenario: Serie mit Sessions löschen
- **WHEN** Trainer DELETE /api/training-series/{id} für Serie mit 3 Sessions, jede mit Antworten
- **THEN** HTTP 204, Serie gelöscht, alle 3 Sessions gelöscht, alle Antworten gelöscht
