## 1. Komponente erstellen

- [ ] 1.1 `web/src/components/NumberSpinner.tsx` anlegen mit Props `value`, `min?`, `max?`, `step?` (default 1), `onChange`, `className?`
- [ ] 1.2 Layout implementieren: `position: relative`-Wrapper, `<input type="number">` mit `padding-right` für Button-Bereich, native Pfeile per `style` ausblenden (`WebkitAppearance: 'none'`, `MozAppearance: 'textfield'`)
- [ ] 1.3 Chevron-Buttons (▲/▼) absolut rechts positionieren mit `ChevronUp`/`ChevronDown` aus `lucide-react`, Farben `bg-brand-yellow text-brand-black`, Hover `bg-brand-black text-brand-yellow`
- [ ] 1.4 Button-Logik: ▲ ruft `onChange(Math.min(max ?? Infinity, value + step))` auf, ▼ ruft `onChange(Math.max(min ?? 0, value - step))` auf
- [ ] 1.5 Buttons bei min/max-Grenzen disablen (▼ disabled wenn `value <= min`, ▲ disabled wenn `value >= max`)
- [ ] 1.6 Input `onChange` weiterleiten: `onChange(parseInt(e.target.value) || 0)`

## 2. Einsatzorte refactorn

- [ ] 2.1 `ProfileMiscTab.tsx`: Bestehende externe ±-Buttons und `<Minus>`/`<Plus>`-Imports entfernen, durch `<NumberSpinner value={vehicle.seats ?? 0} min={0} max={10} onChange={...} />` ersetzen
- [ ] 2.2 `MitfahrgelegenheitenPage.tsx`: Plain `<input type="number">` für Freie Plätze durch `<NumberSpinner value={parseInt(plaetze) || 1} min={1} max={8} onChange={v => setPlaetze(String(v))} />` ersetzen
- [ ] 2.3 `AdminSettingsPage.tsx`: Beide `<input type="number">` für Halbzeit und Pause durch `<NumberSpinner value={parseInt(s.half) || 1} min={1} step={5} onChange={v => updateRow(rule.age_class, 'half', String(v))} />` bzw. `brk` ersetzen, `INPUT_NUM`-Konstante entfernen falls nicht mehr verwendet
