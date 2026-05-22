## Context

`MemberDetailPage.tsx` ist in vier Sektionen aufgebaut: Stammdaten, Mannschaft zuweisen, Nutzer verknüpfen, Elternteile. Die `family_links`-Tabelle hat `(parent_user_id, member_id)` als zusammengesetzten Primary Key — das macht einen DELETE ohne zusätzliche Spalten möglich. Der vorhandene `CreateFamilyLink`-Handler schreibt blind (`INSERT OR IGNORE`) ohne Limit-Prüfung.

## Goals / Non-Goals

**Goals:**
- Mannschafts-Sektion aus der UI entfernen (kein API-Eingriff)
- Erziehungsberechtigte: alle Nutzer wählbar, entfernbar, max. 2
- Umbenennung durchgängig in der UI

**Non-Goals:**
- Datenbankschema ändern (`family_links` bleibt wie es ist)
- Den Team-Assignment-Endpoint löschen (kann intern noch nützlich sein)
- Rollen-Check beim Verknüpfen (wer darf verknüpfen bleibt Admin-only, wer verknüpft werden darf ist offen)

## Decisions

### DELETE /api/admin/family-links mit Body

**Entschieden**: `DELETE /api/admin/family-links` mit JSON-Body `{"parent_user_id": N, "member_id": M}`.

**Alternative**: Pfad-Parameter `DELETE /api/admin/family-links/{parent_user_id}/{member_id}`. Semantisch sauberer, aber verschachtelte numerische IDs in der URL sind unüblich und erfordern mehr Router-Konfiguration.

**Rationale**: Body-basiertes DELETE ist für diesen Stack (Chi, kein REST-Framework) einfacher und konsistent mit dem vorhandenen POST.

### Limit-Durchsetzung auf beiden Ebenen

Frontend deaktiviert den Hinzufügen-Button wenn `linkedParents.length >= 2`. Backend prüft via `SELECT COUNT(*) FROM family_links WHERE member_id = ?` vor dem INSERT und gibt 409 zurück wenn bereits 2 Einträge existieren. Beide Ebenen nötig: das Frontend verhindert versehentliche Klicks, das Backend verhindert Race Conditions oder direkte API-Aufrufe.

### Kein separater „Erziehungsberechtigten"-Rollentyp

Alle Nutzer können verknüpft werden, unabhängig von ihrer Systemrolle. Ein Elternteil, das selbst Spieler oder Trainer ist, soll trotzdem als Erziehungsberechtigter eines Kindes eingetragen werden können.

## Risks / Trade-offs

- **Mannschafts-Zuweisung entfernen**: Wer bisher über die Mitgliederseite Teams zugewiesen hat, findet diese Funktion nicht mehr. Das ist gewollt — die Kaderplanung ist der neue Weg. Kein Datenverlust, da der Endpoint erhalten bleibt.
- **Bestehende Links > 2**: Wenn in der DB bereits mehr als 2 Erziehungsberechtigte verknüpft sind (technisch möglich bisher), zeigt das Frontend alle an, erlaubt aber keine neuen mehr. Kein automatisches Bereinigen.

## Migration Plan

Keine DB-Migration nötig. Deployment via `make deploy` reicht.
