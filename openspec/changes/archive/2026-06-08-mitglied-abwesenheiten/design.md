## Context

Spieler und Elternteile möchten Abwesenheitszeiträume (Urlaub, Verletzung) eintragen können, ohne jede einzelne Training- oder Spiel-Zusage manuell zurückziehen zu müssen. Aktuell gibt es in `training_responses` und `game_responses` manuelle Declined-Einträge, aber kein Konzept für "gesperrte" Responses, die aus einem Systemereignis entstehen.

Bestehende Tabellen:
- `training_responses`: `(training_id, member_id, responded_by, status, reason, responded_at)`
- `game_responses`: `(game_id, member_id, responded_by, status, reason, responded_at)`
- `members`: hat `user_id FK` (verbindet Member mit User-Account) und `status`
- `family_links`: `(parent_user_id, member_id)` — Elternteil → Kind

## Goals / Non-Goals

**Goals:**
- Abwesenheitszeiträume anlegen und löschen (Spieler selbst + Elternteile)
- Auto-decline aller Training/Spiel-Responses im Zeitraum beim Anlegen
- Auto-decline bei neuen Events (Training/Spiel) die in bestehenden Abwesenheitszeitraum fallen
- Auto-declined Responses sind für alle Rollen nicht manuell änderbar
- Optionale Trainer-Sichtbarkeit über `absences_public`-Toggle im Profil
- Kalender-Banner (KalenderPage) für Abwesenheitszeiträume
- Confirmation-Modal vor dem Anlegen wenn bestehende Zusagen betroffen sind

**Non-Goals:**
- Duty Slots / duty_assignments werden nicht berücksichtigt (v2)
- Automatische Benachrichtigungen an Trainer (kein Push bei Abwesenheit)
- Abwesenheiten für Members ohne User-Account (kein Login möglich)
- Admin/Trainer können Abwesenheiten nicht anlegen oder überschreiben

## Decisions

### 1 — `absence_id` FK statt 4. Status-Wert

**Entscheidung:** Beide Response-Tabellen erhalten eine nullable `absence_id`-Spalte (FK auf `member_absences`). `status` bleibt `CHECK('confirmed','declined','maybe')`.

**Warum:** Nur drei visuelle Zustände gewünscht. `absence_id IS NOT NULL` signalisiert "auto-declined, gesperrt" ohne einen neuen Status-String einzuführen. `ON DELETE CASCADE` löst das Aufräumen beim Löschen der Abwesenheit automatisch.

**Alternative:** 4. Status `'absent'` — abgelehnt, da der Nutzer explizit drei Zustände wollte und ein 4. Status die gesamte RSVP-Anzeigelogik im Frontend berührt hätte.

### 2 — Neues Package `internal/absences/`

**Entscheidung:** Eigenes Handler-Struct, eigene Routen. Die Auto-decline-Logik wird als interne Hilfsfunktion im `absences`-Package gehalten und von `games`/`trainings` direkt per SQL-Query umgesetzt (kein gemeinsamer Aufruf).

**Warum:** `games` und `trainings` müssen bei `Create` keine `absences`-Dependency importieren — sie führen dieselbe SQL-Logik lokal durch. Das verhindert zirkuläre Imports und hält die Packages unabhängig.

**Alternative:** Shared `absences.AutoDecline(db, memberID, date)`-Funktion — abgelehnt wegen unnötiger Kopplung für eine einfache SQL-Insert-Operation.

### 3 — Preview-Endpoint vor dem Anlegen

**Entscheidung:** `GET /api/absences/preview` gibt betroffene Events zurück (Trainings + Spiele mit bestehender `confirmed`-Response im Zeitraum). Das Frontend zeigt ein Confirmation-Modal — erst danach `POST /api/absences`.

**Warum:** Das Löschen bestehender Zusagen ist destruktiv. Nutzer soll wissen was passiert, bevor die Aktion ausgeführt wird.

**Alternative:** POST immer ausführen + Ergebnis-Modal — abgelehnt, da die Aktion dann nicht mehr abbrechbar ist.

### 4 — `absences_public` auf `members`-Tabelle

**Entscheidung:** `ALTER TABLE members ADD COLUMN absences_public INTEGER NOT NULL DEFAULT 0`. Kein separates `user_preferences`-Table.

**Warum:** Die Sichtbarkeits-Präferenz ist eine Member-Eigenschaft (nicht User-Eigenschaft). Simpler ALTER statt neue Tabelle für ein einzelnes Boolean.

### 5 — Migration 030 bündelt alle Schema-Änderungen

Eine Migration für: `member_absences`-Tabelle + `members.absences_public` + `training_responses.absence_id` + `game_responses.absence_id`. SQLite erlaubt `ALTER TABLE ADD COLUMN` mit FK (NULL-Default), kein Table-Rebuild nötig.

## Risks / Trade-offs

**Race Condition bei gleichzeitigem Anlegen** → SQLite serialisiert Writes (WAL), kein echtes Problem bei Single-VPS-Deployment.

**Auto-decline überschreibt bestehende `confirmed`-Responses** → Nutzer wird per Confirmation-Modal informiert und muss explizit bestätigen. Accepted trade-off für einfaches Datenmodell.

**`ON DELETE CASCADE` entfernt Response beim Löschen der Abwesenheit** → Member steht danach auf "keine Antwort" und muss ggf. neu zusagen. Ist das gewünschte Verhalten (B-Entscheidung).

**Neue Events in bestehender Abwesenheit: kein Batch-Job nötig** → Auto-decline passiert synchron im Create-Handler. Kein Scheduler-Eintrag.

## Migration Plan

1. Migration 030 wird beim nächsten `make deploy` automatisch via `migrate up` angewendet
2. Bestehende `training_responses`/`game_responses`-Zeilen haben `absence_id = NULL` → bleiben unverändert editierbar
3. Kein Rollback-Risiko für Bestandsdaten
4. `make migrate-remote-up` zum manuellen Testen auf VPS vor Deploy
