## Context

`SpieltagDetailPage.tsx` prüft aktuell `user?.role === 'admin'` für alle Bearbeitungs-Aktionen. Der `DELETE /api/admin/games/{id}`-Endpunkt ist im Backend vorhanden und gibt HTTP 204 zurück. Beim Löschen eines Games werden alle zugehörigen `duty_slots` via `ON DELETE CASCADE` automatisch mitgelöscht.

## Goals / Non-Goals

**Goals:**
- Löschen-Button auf der Detailseite für admin, vorstand, trainer
- Bestätigungs-Dialog mit Hinweis auf Konsequenzen
- Redirect nach erfolgreichem Löschen

**Non-Goals:**
- Löschen direkt aus der Kalenderansicht
- Soft-Delete / Archivierung

## Decisions

### 1. Berechtigungsprüfung im Frontend

`isAdmin` wird zu `canEdit` umbenannt: `user?.role === 'admin' || user?.role === 'vorstand' || user?.role === 'trainer'`. Das ist konsistent mit dem geplanten `spielplan-event-wizard`-Change.

### 2. Löschen-Button Platzierung

Roter „Event löschen"-Button in der Header-Zeile der Detailseite, rechts neben dem bestehenden „Neu generieren"-Button. Nur für `canEdit` sichtbar.

### 3. Bestätigungs-Dialog

Einfacher Inline-Confirm-Dialog (kein Browser-`confirm()`): zeigt Eventname und warnt dass alle Dienste mitgelöscht werden. Zwei Buttons: „Abbrechen" und „Endgültig löschen".

### 4. Redirect nach Löschen

`navigate('/spielplan')` nach erfolgreichem DELETE.

## Risks / Trade-offs

- Trainer könnte theoretisch fremde Events löschen (Backend-Check fehlt) → Mitigation: Backend-Scope-Prüfung ist Teil von `spielplan-event-wizard`; für jetzt reicht Frontend-Scoping da Trainer den Spielplan sehen aber fremde Detailseiten nicht aktiv nutzen
