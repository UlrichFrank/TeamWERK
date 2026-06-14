## Context

Die `members`-Tabelle hat einen CHECK-Constraint auf `status` mit den Werten `aktiv`, `verletzt`, `pausiert`, `ausgetreten`, `passiv`, `honorar`. SQLite erlaubt keine `ALTER TABLE ... MODIFY COLUMN`, deshalb muss die Tabelle bei CHECK-Erweiterungen neu angelegt werden (PRAGMA foreign_keys OFF → CREATE TABLE new → INSERT INTO new SELECT → DROP old → RENAME).

Dieses Muster wurde bereits in Migration 018 (`honorar`) umgesetzt und kann direkt wiederverwendet werden.

## Goals / Non-Goals

**Goals:**
- `anwaerter` als gültiger `members.status`-Wert
- Vorstand kann einen Anwärter über das bestehende "Mitglied anlegen"-Formular erfassen
- Anwärter-Mitglieder können über `kader_extended_members` in Spieltag-Teilnehmerlisten erscheinen
- Kader-Ansicht zeigt einen visuellen Hinweis auf den Anwärter-Status
- Status-Upgrade zu `aktiv` erfolgt manuell über die bestehende `PUT /api/members/:id/status`-Route

**Non-Goals:**
- Kein DB-Constraint der Anwärter auf `kader_extended_members` beschränkt (Konvention, nicht Enforcement)
- Kein automatischer Übergang Anwärter → aktiv
- Kein eigener Account / Login für Anwärter
- Keine vereinfachte Schnellerfassung (das bestehende Formular mit optionalen Feldern reicht)

## Decisions

**1. Neuer Status statt neuer Tabelle**

`anwaerter` wird als weiterer `status`-Wert in die `members`-Tabelle aufgenommen — analog zu `honorar`. Alternativen wie eine separate `applicants`-Tabelle hätten mehr Code-Aufwand ohne Mehrwert erzeugt, da alle relevanten Kader- und Spieltag-Queries bereits auf `member_id` operieren.

**2. Migration per Tabellen-Rebuild**

SQLite unterstützt kein `ALTER TABLE ... MODIFY COLUMN` für CHECK-Constraints. Die Migration folgt dem gleichen Muster wie 018: `foreign_keys OFF` → neue Tabelle → INSERT INTO new SELECT → DROP old → RENAME → alle abhängigen Views/Indizes neu anlegen. Migration erhält die nächste freie Nummer (038).

**3. Kein eigenes UI-Formular**

Das bestehende "Mitglied anlegen"-Formular hat bereits alle Felder außer Name als optional. Kein separates Formular nötig — der Vorstand wählt Status `anwaerter` und füllt nur Pflichtfelder aus.

**4. Badge im Kader-Frontend**

Der Backend-API-Response enthält bereits `status` für Kader-Mitglieder. Das Frontend prüft `member.status === 'anwaerter'` und rendert ein kleines Label. Kein API-Änderung erforderlich.

## Risks / Trade-offs

- [Migration bricht bestehende DBs] → Tabellen-Rebuild ist established pattern (018). Down-Migration stellt den alten CHECK-Constraint wieder her. Falls Anwärter vor Rollback existieren, schlägt die Down-Migration fehl — akzeptabel, da Rollback nur im lokalen Dev-Kontext vorkommt.
- [Anwärter landet versehentlich im primären Kader] → Kein DB-Enforcement, aber das Badge im Kader-View macht den Status sichtbar. Konventionsverstoß ist korrigierbar.

## Migration Plan

1. `make migrate-up` lokal testen
2. `make deploy` — der neue Binary führt 038 automatisch aus
3. Rollback: `make migrate-remote-down` (entfernt `anwaerter` aus CHECK; nur möglich wenn keine Anwärter-Datensätze existieren)
