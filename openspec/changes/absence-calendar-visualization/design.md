## Context

Die `KalenderPage` rendert Abwesenheitsbalken (`dayAbsences`) aktuell als `h-1.5`-Div im normalen Dokumentfluss unterhalb der Tag-Zahl. Jede Kalenderzelle ist `@container group min-h-[90px] p-1.5 border-r border-b`. Die Balken verwenden `bg-brand-yellow/40 border border-brand-yellow` mit runden Enden für erste/letzte Tage und `-mx-1.5` um bei Fortsetzungen die Zellgrenzen zu überbrücken.

Im Backend (`internal/absences/handler.go`) prüft `Create` bisher nicht, ob der Member bereits eine Abwesenheit desselben Typs im angefragten Zeitraum hat.

## Goals / Non-Goals

**Goals:**
- Balken absolut hinter dem Zell-Content positionieren, sodass die Tag-Zahl visuell mittig sitzt
- Gleichmäßige Abstände zu allen Trennlinien durch das bestehende Cell-Padding
- Typ-differenzierte Farben bei deutlich reduzierter Sättigung
- Radius ausschließlich am Start- und Endtag der Abwesenheit
- Overlap-Schutz im Backend für gleichen Typ desselben Members

**Non-Goals:**
- Dritter Abwesenheitstyp „Sonstiges" (kein Schema, separate Change)
- Klickbare Balken oder Detail-Popups
- Änderung der Sichtbarkeitslogik (`absences_public`)

## Decisions

### Entscheidung: Absolutes Positioning mit Cell-Padding als Abstandsgeber

**Gewählt:** Zell-Div behält `p-1.5` (für den In-Flow-Content). Balken-Div bekommt `absolute top-[4px] left-[4px] right-[4px] h-4`. Der Content wird in `<div className="relative z-10">` eingewickelt.

**Warum:** Absolut positionierte Kinder ignorieren das Padding ihres Eltern-Divs — `inset-x-0 top-0` würde den Balken direkt an die innere Borderkante legen (kein sichtbarer Abstand). Explizite 4 px-Offsets erzeugen einen gleichmäßigen Abstand auf allen drei Seiten. Die Geometrie ist konsistent: Balken-Oberkante bei 4 px, Höhe 16 px → Balken-Mitte bei 12 px; Tag-Zahl mit `p-1.5` (6 px) plus halbe Zeilenhöhe (~6 px) → Zahlen-Zentrum ebenfalls bei 12 px. Das `p-1.5` des Zell-Divs bleibt für Event-Pills und andere In-Flow-Kinder erhalten.

**Alternative verworfen:** `inset-x-0 top-0` (Balken berührt die Borders direkt, kein sichtbarer Abstand).

### Entscheidung: Radius-Logik — nur Start und Ende

**Gewählt:** `rounded-l` (isFirst), `rounded-r` (isLast), `rounded` (isFirst && isLast), kein Radius (Mitteltag). Kein negativer Margin / kein Bleeding über Zellgrenzen.

**Warum:** Durchgehende Balken über Zellgrenzen verdecken die `border-r`-Trennlinien. Separate Rechtecke mit identischer Höhe und Farbe vermitteln die Mehrtägigkeit implizit. Der Radius signalisiert Anfang und Ende der Abwesenheit.

### Entscheidung: Transparente Farbüberlagerung für verschiedene Typen

**Gewählt:** `bg-brand-yellow/20 border border-brand-yellow/60` (Urlaub), `bg-red-400/20 border border-red-400/60` (Verletzung). Border mit dreifacher Opacity der Füllung ergibt einen kräftigeren, aber noch dezenten Rahmen. Abrundung `rounded-l-full` / `rounded-r-full` / `rounded-full` für halbkreisförmige Enden. Beide werden als separate DOM-Elemente übereinander gerendert; Transparenz mischt die Farben.

**Warum:** Keine komplexe Split-Logik nötig. Max. 2 Typen gleichzeitig → Farbmischung ist eindeutig erkennbar (gelb + rot = orange-ish). Ohne Border wirkt die Fläche deutlich dezenter als zuvor.

### Entscheidung: Overlap-Schutz per SQL-Check vor INSERT

**Gewählt:** Im `Create`-Handler vor dem INSERT eine COUNT-Abfrage:
```sql
SELECT COUNT(*) FROM member_absences
WHERE member_id = ? AND type = ?
  AND start_date <= ? AND end_date >= ?
```
Bei Count > 0 → HTTP 409 mit `{"error":"overlap"}`.

**Warum:** Einfachste Lösung, kein zusätzlicher Lock nötig (SQLite serialisiert Writes ohnehin). Frontend zeigt einen spezifischen Fehlermeldungstext bei 409.

## Risks / Trade-offs

- [Race condition bei gleichzeitigem POST] → vernachlässigbar, SQLite schreibt serialisiert; im Worst-case werden beide akzeptiert, was harmlos ist
- [Farben nicht im Brand-Token-System] → `bg-red-400/20` ist raw Tailwind; akzeptabel für einen seltenen Sonderfall (Verletzung); kein eigener Token nötig
