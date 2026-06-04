# Design: Farbkodierung der Termin-Kacheln

**Datum:** 2026-06-04

## Ziel

Termin-Kacheln und Kalender-Zellen in `TerminePage` und `KalenderPage` sollen sich nach Event-Typ farblich unterscheiden. Vier Typen, vier Farben aus der Brand-Palette.

## Farbzuordnung

| Typ | Rahmen & Icon | Hintergrund (abgeschwächt) | Filter-Button (aktiv) |
|-----|--------------|----------------------------|-----------------------|
| Training | `brand-green` `#6EB42E` | `bg-brand-green/10` | `bg-brand-green text-white` |
| Heimspiel | `brand-yellow` `#FDE400` | `bg-brand-yellow/15` | `bg-brand-yellow text-brand-black` |
| Auswärtsspiel | `brand-blue` `#3E4A98` | `bg-brand-blue/10` | `bg-brand-blue text-white` |
| Generisch | `brand-text-muted` `#6B7280` | `bg-brand-gray/40` | `bg-brand-gray text-brand-black` |

Referenz-Helligkeit: aktuell verwendetes `bg-blue-50` für Training-Events im KalenderPage.

## Implementierung

### Neue Datei: `web/src/lib/eventColors.ts`

Zentrales Mapping-Objekt `EVENT_COLORS` mit drei Anwendungsfeldern je Typ:
- `card` — für die Kacheln in TerminePage (`border`, `bg`, `icon`)
- `filter` — Tailwind-Klassenstring für aktive Filter-Buttons
- `pill` — für die kleinen Event-Pills im Kalender-Monatsraster (`bg`, `hover`, `border`, `icon`)

### TerminePage.tsx

- **Kacheln:** `border-brand-yellow` durch `EVENT_COLORS[type].card.border` ersetzen; `bg-brand-surface-card` durch `EVENT_COLORS[type].card.bg`; Icon-Element erhält `EVENT_COLORS[type].card.icon`
- **Filter-Buttons:** Aktiv-Klasse `bg-brand-yellow text-brand-black border-brand-yellow` wird pro Typ durch `EVENT_COLORS[type].filter` ersetzt
- **Abgesagte Trainings:** behalten weiterhin `border-brand-border opacity-60` (kein Typ-Farbakzent)

### KalenderPage.tsx

- **Game-Pills im Monatsraster:** Aktuell neutral-grau → erhalten `EVENT_COLORS[event_type].pill.*`
- **Training-Pills:** `bg-blue-50 hover:bg-blue-100 border-blue-200 text-blue-500` → `EVENT_COLORS.training.pill.*`
- **Filter-Buttons** (falls vorhanden): wie TerminePage

## Nicht im Scope

- Keine Änderungen an TermineDetailPage
- Keine neuen Tailwind-Config-Tokens
- Keine Änderungen an Backend-Routen
