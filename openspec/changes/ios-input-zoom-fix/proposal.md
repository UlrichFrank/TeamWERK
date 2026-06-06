## Why

iOS Safari zoomt automatisch in Eingabefelder rein, deren `font-size` unter 16px liegt – und zoomt nach dem Verlassen nicht zuverlässig wieder zurück. Da alle Inputs in TeamWERK `text-sm` (14px) verwenden, passiert dies bei jedem Login-Formular, jeder Suche und jedem Modal. Als PWA ist das besonders störend, weil sich die App dadurch nicht nativ anfühlt.

## What Changes

- Eine globale CSS-Regel in `web/src/index.css` setzt `font-size: 16px` für alle `input`, `textarea` und `select`-Elemente
- Der visuelle Eindruck bleibt unverändert: Tailwind-Klassen wie `text-sm` steuern weiterhin die wahrgenommene Größe via `transform: scale()` (falls nötig) oder die Regel greift einfach als Minimum

## Capabilities

### New Capabilities
- `ios-input-zoom-prevention`: Globale CSS-Basis-Regel verhindert iOS-Auto-Zoom auf Eingabefeldern

### Modified Capabilities
<!-- keine bestehenden Specs betroffen -->

## Impact

- `web/src/index.css`: eine neue CSS-Regel
- Kein Backend-Code betroffen
- Keine neuen Dependencies
- Alle Seiten mit Eingabefeldern profitieren automatisch
