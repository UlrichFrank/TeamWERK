## Why

Drei Stellen in der App verwenden Zahlen-Eingabefelder (Sitzplätze im Profil, Freie Plätze bei Mitfahrgelegenheiten, Halbzeit/Pause in den Altersklassen-Einstellungen), die unterschiedlich implementiert sind und inkonsistent aussehen. Eine einheitliche Komponente schafft visuell konsistente UX und reduziert doppelten Code.

## What Changes

- **Neue Komponente** `web/src/components/NumberSpinner.tsx` mit einem `<input type="number">` und zwei custom Chevron-Buttons (▲/▼) absolut positioniert rechts im Feld, in Markenfarben Gelb/Schwarz
- **Refactoring** `ProfileMiscTab.tsx`: Bestehende ±-Buttons außerhalb des Feldes durch `NumberSpinner` ersetzen
- **Refactoring** `MitfahrgelegenheitenPage.tsx`: Plain `<input type="number">` durch `NumberSpinner` ersetzen
- **Refactoring** `AdminSettingsPage.tsx`: Die beiden `<input type="number">` für Halbzeit und Pause durch `NumberSpinner` ersetzen (mit `step={5}`)

## Capabilities

### New Capabilities

- `number-spinner`: Wiederverwendbarer Zahlen-Spinner mit gestylten Chevron-Buttons rechts im Eingabefeld, konfigurierbarer Schrittweite und min/max-Grenzen

### Modified Capabilities

*(keine bestehenden Specs betroffen — reine UI-Refactor-Änderung)*

## Impact

- **Neue Datei:** `web/src/components/NumberSpinner.tsx`
- **Geänderte Dateien:**
  - `web/src/components/profile/ProfileMiscTab.tsx`
  - `web/src/pages/MitfahrgelegenheitenPage.tsx`
  - `web/src/pages/AdminSettingsPage.tsx`
- **Keine API-Änderungen**, keine Datenbank-Migrationen, keine neuen Dependencies
- `lucide-react` (bereits installiert) liefert `ChevronUp` und `ChevronDown`
