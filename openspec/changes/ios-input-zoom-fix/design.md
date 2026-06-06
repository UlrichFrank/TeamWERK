## Context

iOS Safari löst automatisch einen Zoom-Effekt aus, wenn ein `<input>`, `<textarea>` oder `<select>` mit `font-size < 16px` fokussiert wird. Da Tailwind `text-sm` auf 14px und `text-xs` auf 12px setzt und alle Inputs in TeamWERK `text-sm` verwenden, betrifft das die gesamte App. Der Zoom wird nach dem Blur nicht zurückgesetzt, was die UX als PWA signifikant verschlechtert.

## Goals / Non-Goals

**Goals:**
- iOS-Auto-Zoom auf Eingabefeldern dauerhaft verhindern
- Keine Änderung am visuellen Design
- Einzeilige Lösung ohne neue Dependencies

**Non-Goals:**
- Accessibility-Einschränkungen via `user-scalable=no` (kein manuelles Zoom-Sperren)
- Umstellen aller Tailwind-Klassen von `text-sm` auf `text-base`
- Android- oder Desktop-Browser-Anpassungen

## Decisions

### Globale CSS-Regel statt komponentenweiser Anpassung

**Entscheidung:** Eine einzige Regel in `web/src/index.css` setzt `font-size: 16px` als Basis für alle Eingabeelemente.

```css
input, textarea, select {
  font-size: 16px;
}
```

**Warum nicht `font-size: max(16px, 1em)` im Tailwind-Config?**  
Das würde alle `text-sm`/`text-xs`-Klassen übersteuern und das Layout verändern. Die direkte CSS-Regel gilt nur als Minimum-Basis, Tailwind-Klassen auf `<input>`-Elementen können sie weiter überschreiben — sofern nötig.

**Warum nicht `@tailwind base` anpassen?**  
Die Preflight-Base von Tailwind setzt absichtlich keine `font-size` auf Inputs, um Browser-Defaults zu erhalten. Eine explizite Regel nach den `@tailwind`-Direktiven hat Priorität und ist wartbarer.

**Alternatives considered:**
- `maximum-scale=1` im Viewport-Meta → schlechte Accessibility, ausgeschlossen
- Alle Inputs auf `text-base` umstellen → großer Diff, Risiko für Layout-Shifts

## Risks / Trade-offs

- [Visueller Eindruck] Falls ein Input explizit `text-xs` (12px) benötigt, gilt nun trotzdem 16px als Basis → Mitigation: im Einzelfall mit `!text-xs` oder direkt `style={{ fontSize: 12 }}` überschreiben (bisher kein bekannter Fall)
- [Cascading] Tailwind-Utilities mit `font-size` auf Input-Elementen gelten weiterhin normal, da sie spezifischer oder nachrangig sind — kein Breaking Change erwartet
