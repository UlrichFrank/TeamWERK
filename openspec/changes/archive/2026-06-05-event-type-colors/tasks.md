## 1. Shared Color Mapping

- [x] 1.1 Datei `web/src/lib/eventColors.ts` erstellen mit `EVENT_COLORS`-Objekt (card, filter, pill je Typ)

## 2. TerminePage

- [x] 2.1 Training-Kacheln: `border-brand-yellow` → `border-brand-green`, Hintergrund → `bg-brand-green/10`, Dumbbell-Icon → `text-brand-green`
- [x] 2.2 Heimspiel-Kacheln: Hintergrund → `bg-brand-yellow/15`, Home-Icon → `text-brand-yellow` (Border bleibt gelb)
- [x] 2.3 Auswärtsspiel-Kacheln: `border-brand-yellow` → `border-brand-blue`, Hintergrund → `bg-brand-blue/10`, MapPin-Icon → `text-brand-blue`
- [x] 2.4 Generisch-Kacheln: `border-brand-yellow` → `border-brand-text-muted`, Hintergrund → `bg-brand-gray/40`, Calendar-Icon → `text-brand-text-muted`
- [x] 2.5 Filter-Buttons: aktive Klasse pro Typ aus `EVENT_COLORS[type].filter` statt hartkodiertem `bg-brand-yellow text-brand-black`
- [x] 2.6 Abgesagte Trainings: weiterhin `border-brand-border opacity-60`, kein Typ-Farbakzent

## 3. KalenderPage

- [x] 3.1 Training-Pills: `bg-blue-50 hover:bg-blue-100 border-blue-200 text-blue-500` → Werte aus `EVENT_COLORS.training.pill`
- [x] 3.2 Game-Pills: Hintergrund + Icon nach `event_type` aus `EVENT_COLORS[event_type].pill`

## 4. Verifikation

- [x] 4.1 TerminePage im Browser prüfen: alle vier Typen korrekt eingefärbt, abgesagte Trainings neutral
- [x] 4.2 KalenderPage im Browser prüfen: Training- und Spiel-Pills korrekt eingefärbt
- [x] 4.3 Filter-Buttons in beiden Seiten prüfen: aktive Farbe typ-spezifisch
