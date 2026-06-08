## Why

Der aktuelle Abwesenheits-Balken im Monatskalender liegt im normalen Dokumentfluss und ist zu dünn, zu gesättigt und falsch positioniert — die Tag-Zahl sitzt nicht mittig darin. Gleichzeitig fehlt eine Backend-Validierung die verhindert, dass derselbe Abwesenheitstyp für ein Mitglied mehrfach im gleichen Zeitraum eingetragen wird.

## What Changes

- Abwesenheits-Balken wird absolut hinter dem Zell-Inhalt positioniert (nicht mehr im Fluss unterhalb der Tag-Zahl)
- Gleichmäßiger Abstand zu allen Trennlinien: das bestehende Cell-Padding (`p-1.5`) dient als natürlicher Abstand
- Balkenhöhe 20 px sodass die Tag-Zahl vertikal mittig im Balken sitzt
- Radius nur am ersten Tag (linke Ecken) und letzten Tag (rechte Ecken); Mitteltage bleiben eckig
- Geringere Farbsättigung: gedämpftes Gelb mit dezent dunklerem Rahmen (Border doppelt so opak wie Füllung)
- Verschiedene Abwesenheitstypen erhalten unterschiedliche Farben (Urlaub = gedämpftes Gelb, Verletzung = gedämpftes Rot) und dürfen sich überlagern
- **Backend-Validierung**: `POST /api/absences` gibt HTTP 409 zurück, wenn für dasselbe Mitglied bereits eine Abwesenheit desselben Typs den angefragten Zeitraum überdeckt

## Visualisierung

### Einzelne Zelle mit Abwesenheit

```
Kalenderzelle (min-h-[90px], p-1.5)
┌──────────────────────────────────────┐
│  ╭────────────────────────────────╮  │  ← Balken: top-[4px] left-[4px] right-[4px] h-4
│  │  [12]                      [+] │  │  ← Tag-Zahl mittig im Balken (relative z-10)
│  ╰────────────────────────────────╯  │
│  [Spiel-Pill]                        │
│  [Training-Pill]                     │
└──────────────────────────────────────┘
  ↑                                  ↑
  4 px Abstand zur Border auf allen Seiten (oben, links, rechts)
```

### Mehrtägige Abwesenheit — Radius-Logik

```
   Mo (erster Tag)    Di (Mitteltag)    Mi (letzter Tag)
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│╭──────────╮  │   │ ┌──────────┐ │   │  ╭──────────╮│
││   [28]   │  │   │ │   [29]   │ │   │  │   [30]   ││
│╰──────────╯  │   │ └──────────┘ │   │  ╰──────────╯│
│  [event]     │   │              │   │  [event]      │
└──────────────┘   └──────────────┘   └──────────────┘
  rounded-l          keine Rundung      rounded-r
  (linke Ecken)      (eckig)            (rechte Ecken)
```

Eintägige Abwesenheit: alle vier Ecken abgerundet.  
Jede Zelle hat denselben gleichmäßigen Abstand zur Border — keine Balken überbrücken Zellgrenzen.

### Zwei Abwesenheitstypen am gleichen Tag

```
┌──────────────────────────────────────┐
│▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒│  Urlaub:    gedämpftes Gelb
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  Verletzung: gedämpftes Rot
│  [15]                            [+] │  (überlagert: Farben mischen sich)
└──────────────────────────────────────┘
```

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `member-absences`: Neue Anforderung für Überlappungsschutz gleicher Typen; visuelles Verhalten des Kalender-Banners wird präzisiert (Höhe, Positionierung, Radius-Logik, Farbschema)

## Impact

- `web/src/pages/KalenderPage.tsx` — Zell-Rendering und Balken-Klassen
- `internal/absences/handler.go` — Overlap-Check in `Create`
- `openspec/specs/member-absences/spec.md` — Delta für neue Anforderungen
