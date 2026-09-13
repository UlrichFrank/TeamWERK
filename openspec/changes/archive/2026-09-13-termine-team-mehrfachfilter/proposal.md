## Why

Zwei Befunde von derselben Seite, aus demselben Meldungsweg (`/termine`):

**1. Der Fokus-Marker überlebt eine Filteränderung und zeigt dann einen fremden Termin.**
Push-Meldungen, Event-Log und die Rückkehr von der Detailseite setzen `?focus=game-<id>`
(`internal/scheduler/scheduler.go`, `internal/games/handler.go`, Karten-Klick in
`TerminePage.tsx`). Der Fokus-Durchlass in `visibleTermine` ist absolut — er ignoriert
Team-, Typ- und Textfilter, wie in `termine-unified-view` zugesagt. Nur endet dieser
Zustand nie: wer nach einer Push-Meldung zum mC2-Turnier den Team-Filter auf mA2 stellt,
sieht das mC2-Turnier weiterhin, mit Ring markiert, obwohl der Filter es ausschließt.
Der Durchlass ist für den Moment des Deep-Links gedacht, nicht für die Sitzung danach.

**2. Der Team-Filter kann nur genau eine Mannschaft.** Ein Elternteil mit zwei Kindern in
verschiedenen Mannschaften (der Normalfall dieser Seite) kann nicht „mA2 und mC2, aber
nicht den Rest" sehen — nur eine Mannschaft oder alle. Der Typ-Filter kann das längst:
`EventTypeFilter` bietet im Compact-Modus Checkboxen und mehrfache Auswahl. Auf Mobile ist
die Team-Auswahl zusätzlich gar nicht erreichbar (`hidden sm:block`, aus Platzgründen).

## What Changes

- **Der Fokus fällt, sobald der Nutzer selbst Team- oder Typ-Filter ändert.** Ein per URL
  ankommender `focus` wirkt unverändert (inkl. des dokumentierten Filter-Durchlasses); erst
  die aktive Filteränderung beendet ihn. `past` und `q` lassen ihn bewusst stehen: `past`
  wird von der Fokus-Logik selbst umgeschaltet (Termin liegt in der Vergangenheit), und für
  `q` ist das Überleben in `termin-textfilter` ausdrücklich zugesagt.
- **Der Team-Filter wird mehrfachauswählbar** — ein Dropdown mit Checkboxen, gebaut wie der
  Compact-Modus des Typ-Filters. `team` nimmt jetzt eine kommaseparierte ID-Liste
  (`team=3,7`); ein einzelnes `team=3` bleibt gültig, alte Links funktionieren weiter.
- **Deselektion aller Mannschaften verhält sich wie beim Typ-Filter:** leere Auswahl =
  kein Filter = alle Mannschaften (die Kästchen springen zurück auf „alle angehakt").
  Ebenso, wenn alle Mannschaften einzeln angehakt sind — dann steht kein `team` in der URL.
- **Der Team-Filter ist auch auf Mobile bedienbar.** Als Icon-Button mit Zähler braucht er
  den Platz nicht mehr, den das frühere `<select>` beanspruchte.

## Capabilities

### Modified Capabilities

- `termine-unified-view`: `team` ist eine ID-Liste; der Deep-Link-Fokus endet bei aktiver
  Filteränderung.

## Impact

- `web/src/pages/TerminePage.tsx` — Filter-Parsing (`team` als Menge), `updateFilter`
  (Fokus-Ende), Filterprädikat, Kopfzeile
- `web/src/components/TeamFilter.tsx` (neu) — Dropdown mit Checkboxen
- `web/src/components/EventTypeFilter.tsx` — Schließ-Logik in den gemeinsamen Hook ausgelagert
- `web/src/hooks/useDismissOnOutside.ts` (neu) — Klick/Touch/Scroll außerhalb schließt das Dropdown
- Kein Backend, keine Migration, keine neue Route. `/api/teams` liefert bereits nur die für
  den Nutzer sichtbaren Mannschaften.

## Test-Anforderungen

| Fläche | Test | Erwartung |
|---|---|---|
| `/termine?team=1,2` | `Team-Filter aus der URL nimmt mehrere IDs` | Termine beider Mannschaften sichtbar, dritte Mannschaft nicht |
| `/termine?team=1` | `einzelne Team-ID bleibt gültig` | Rückwärtskompatibilität alter Links/Push-Ziele |
| Dropdown | `Abwählen einer Mannschaft schreibt die restlichen in die URL` | aus „alle" wird die Menge ohne die abgewählte |
| Dropdown | `Abwählen aller Mannschaften = kein Filter` | `team` verschwindet aus der URL, alle Termine sichtbar |
| Dropdown | `Anhaken aller Mannschaften = kein Filter` | `team` verschwindet aus der URL |
| Fokus | `Team-Filteränderung beendet den Fokus` | `focus` verschwindet aus der URL, der fokussierte Termin fällt unter den Filter |
| Fokus | `Typ-Filteränderung beendet den Fokus` | dito |
| Fokus | `Fokus aus der URL überlebt den ersten Render mit Filter` | dokumentiertes Deep-Link-Verhalten bleibt |

**Garantierte Invariante:** Ein Termin, den der Nutzer nach einer selbst vorgenommenen
Filteränderung sieht, erfüllt den Filter — es gibt keinen Durchlass, der eine aktive
Filterentscheidung überdauert.
