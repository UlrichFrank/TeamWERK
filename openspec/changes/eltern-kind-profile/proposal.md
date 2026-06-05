## Why

Eltern (Rolle `elternteil`) verwalten ihre Kinder im Verein, haben aber keinen direkten Zugriff auf die vollständige Profilbearbeitung der Kinder. Derzeit sehen Eltern ihre Kinder nur im „Kontakt"-Tab ihres eigenen Profils — ohne die Möglichkeit, Mitgliedsdaten, Bankdaten oder Kontaktinfos der Kinder direkt zu bearbeiten. Das führt zu Umwegen über den Admin.

## What Changes

- In der Sidebar erscheinen für `elternteil`-Nutzer unterhalb von „Mein Profil" dynamische Einträge für jedes verknüpfte Kind (z.B. „Jannes Profil")
- Neue Route `/profil/kind/:memberId` zeigt das vollständige Profil eines Kindes mit denselben Tabs wie „Mein Profil"
- Eltern können alle Felder des Kind-Profils bearbeiten (Kontakt, Mitgliedsdaten, Bankdaten, Sonstiges)
- Neue Backend-Endpunkte für das Lesen und Schreiben von Kindprofilen mit Autorisierungsprüfung via `family_links`

## Capabilities

### New Capabilities

- `kind-profil`: Kindprofil-Ansicht und -Bearbeitung durch Elternteile — neue Route, dynamische Nav-Einträge, Backend-Endpunkte mit `family_links`-Autorisierung

### Modified Capabilities

- `members`: Kein Änderungsbedarf auf Spec-Ebene (family_links existieren bereits, keine neuen Verhaltensregeln für Members selbst)

## Impact

- **Frontend:** `AppShell.tsx` (dynamische Nav-Einträge), neues `ChildProfilePage.tsx`, neue Route in `App.tsx`
- **Backend:** `internal/members/handler.go` — neue Endpunkte `GET /api/profile/kind/:memberId` und `PUT /api/profile/kind/:memberId/*` mit family_links-Autorisierung
- **Keine neuen DB-Migrationen** — `family_links`-Tabelle ist bereits vorhanden
- **Keine neuen Abhängigkeiten**
