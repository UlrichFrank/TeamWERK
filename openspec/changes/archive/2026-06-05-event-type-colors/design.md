## Context

TerminePage und KalenderPage zeigen aktuell alle Event-Typen mit identischem gelben Rahmen. Training-Pills im Kalender verwenden ad-hoc `bg-blue-50/border-blue-200/text-blue-500` — außerhalb des Brand-Tokensystems. Es gibt keine zentrale Stelle für typ-spezifische Farben.

## Goals / Non-Goals

**Goals:**
- Vier Event-Typen (Training, Heimspiel, Auswärtsspiel, Generisch) visuell unterscheidbar machen
- Single source of truth für Farbzuordnungen, die beide Seiten verwenden
- Bestehende Brand-Tokens nutzen, keine neuen Tailwind-Config-Einträge

**Non-Goals:**
- Keine Änderungen an TermineDetailPage
- Keine Backend-Änderungen
- Kein Dark Mode

## Decisions

**Zentrales Mapping-Objekt statt Inline-Klassen**  
`web/src/lib/eventColors.ts` exportiert `EVENT_COLORS` — ein Record keyed by event type mit je drei Nutzungsfeldern (`card`, `filter`, `pill`). Alternative wäre Inline-Conditionals direkt in den Komponenten gewesen, aber da beide Seiten die gleichen Farben brauchen, ist ein gemeinsames Modul die klarere Lösung.

**Opacity-Modifier für Hintergründe**  
Hintergrundfarben werden als `bg-brand-X/10` (10–15 % Opacity) umgesetzt statt separater Hex-Werte. Tailwind v3 JIT generiert diese aus den bestehenden Tokens korrekt — kein Tailwind-Config-Eingriff nötig.

**Farbzuordnung:**

| Typ | Farbe |
|-----|-------|
| Training | `brand-green` (#6EB42E) |
| Heimspiel | `brand-yellow` (#FDE400) |
| Auswärtsspiel | `brand-blue` (#3E4A98) |
| Generisch | `brand-text-muted` (#6B7280) für Icon/Border, `brand-gray` (#E5E7EB) für Hintergrund |

**Abgesagte Trainings:** behalten `border-brand-border opacity-60` — kein Typ-Farbakzent, da der Cancelled-Zustand die Typ-Zugehörigkeit dominiert.

## Risks / Trade-offs

- [Tailwind-JIT-Purging] Opacity-Klassen wie `bg-brand-green/10` müssen im `content`-Glob erfasst sein → kein Risiko, da `src/**/*.{ts,tsx}` bereits abgedeckt
- [Wartung] Neue Event-Typen müssen manuell in `EVENT_COLORS` ergänzt werden → akzeptabel, da Event-Typen stabil sind
