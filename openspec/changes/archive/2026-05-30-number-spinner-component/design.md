## Context

An drei Stellen in der App werden Zahlenfelder für kleine positive Ganzzahlen benötigt. Die Implementierungen sind inkonsistent: ProfileMiscTab hat externe ±-Buttons, die anderen nutzen native `<input type="number">`. Ziel ist eine einheitliche Komponente, die wie ein nativer Spinner aussieht aber markengerecht gestaltet ist.

## Goals / Non-Goals

**Goals:**
- Einheitliches Aussehen aller drei Einsatzorte
- Chevron-Buttons (▲/▼) rechts im Feld, Markenfarben Gelb/Schwarz
- Direktes Eintippen bleibt möglich
- Konfigurierbarer Step-Wert (z.B. 5 für Minuten-Felder)

**Non-Goals:**
- Kein Ersatz für beliebige `<input type="number">` in der gesamten App — nur diese drei Stellen
- Keine Accessibility-Extras über native `<input>` hinaus

## Decisions

### D1: Layout — Buttons absolut positioniert im Input-Wrapper

Die Chevron-Buttons sitzen in einem `position: relative`-Wrapper rechts neben dem Input-Feld. Das Input selbst bekommt genug `padding-right`, damit der Text nie unter die Buttons rutscht. Native Browser-Pfeile werden via `[appearance:none]` / `[-moz-appearance:textfield]` ausgeblendet.

**Alternative:** Separate Buttons außerhalb (wie ProfileMiscTab heute) → abgelehnt, da der User explizit Buttons *im* Feld wünscht.

### D2: Farbe — Gelb/Schwarz für den Button-Bereich

Der Button-Bereich rechts: `bg-brand-yellow`, Icons `text-brand-black`. Hover: `bg-brand-black`, Icons `text-brand-yellow`. Konsistent mit Primary-Button-Konvention der App.

### D3: Step-Prop mit Default 1

```tsx
<NumberSpinner value={20} min={1} step={5} onChange={v => ...} />
```

Beim Klick auf ▲: `Math.min(max ?? Infinity, value + step)`. Beim Klick auf ▼: `Math.max(min ?? 0, value - step)`. Beim direkten Tippen: Wert wird direkt durchgereicht (kein Snap auf Step-Raster).

### D4: Keine neue Dependency

`ChevronUp` / `ChevronDown` aus `lucide-react` (bereits installiert). Kein zusätzliches Paket.

## Risks / Trade-offs

- **`appearance: none` auf Firefox** → wird mit `-moz-appearance: textfield` abgedeckt, aber Tailwind-Klasse `appearance-none` deckt das nicht ab; muss als Inline-Style oder via `[&::-webkit-inner-spin-button]:hidden` gelöst werden. → Mitigation: beide Vendor-Prefixes im JSX via `style`-Prop setzen.
- **Schrittweite beim Tippen wird nicht erzwungen** → Beim direkten Tippen kann der Nutzer beliebige Werte eingeben (z.B. 7 statt 5 oder 10). Das ist gewünscht (keine Einschränkung des Tippens), aber bei Altersklassen kann ein ungültiger Wert entstehen. → Mitigation: Backend validiert, kein Frontend-Zwang.
