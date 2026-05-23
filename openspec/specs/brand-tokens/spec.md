## ADDED Requirements

### Requirement: Semantische Farb-Tokens in tailwind.config.js
`tailwind.config.js` SHALL die folgenden neuen Tokens unter `theme.extend.colors` definieren:

| Token | Hex-Wert |
|---|---|
| `brand-surface-card` | `#F9FAFB` |
| `brand-text` | `#111827` |
| `brand-text-muted` | `#6B7280` |
| `brand-text-subtle` | `#9CA3AF` |
| `brand-border` | `#D1D5DB` |
| `brand-border-subtle` | `#E5E7EB` |
| `brand-danger` | `#C0253A` |
| `brand-danger-light` | `#FCEEF1` |
| `brand-info` | `#3B82F6` |

Bestehende Tokens (`brand-yellow`, `brand-black`, `brand-blue`, `brand-green`, `brand-error`, `brand-success`, `brand-warning`, `brand-table-select` und alle `*-light`-Varianten) bleiben unverändert erhalten.

#### Scenario: Token wird als Tailwind-Klasse verwendet
- **WHEN** eine TSX-Datei `bg-brand-surface-card` oder `text-brand-text-muted` enthält
- **THEN** baut Vite ohne Fehler und die Klasse wendet die korrekte Hex-Farbe an

#### Scenario: Raw-Tailwind-Farben sind nicht mehr nötig
- **WHEN** `bg-gray-50`, `text-gray-500`, `text-gray-400`, `border-gray-300` in einer Datei gesucht wird
- **THEN** sind diese Klassen durch die entsprechenden `brand-*`-Tokens ersetzt

#### Scenario: brand-danger ersetzt destructive-red
- **WHEN** ein Danger-Button oder Fehler-Alert gerendert wird
- **THEN** erscheint die Farbe `#C0253A` (Karmesin), nicht `#EF4444` (altes brand-error)
