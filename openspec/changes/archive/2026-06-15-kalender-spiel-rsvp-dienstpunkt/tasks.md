## 1. Spiel-Pill in KalenderPage.tsx anpassen

- [x] 1.1 Teamname-Zeile: `flex-1` auf `<span>` mit Teamname setzen und Dienst-Punkt (`hidden @tile-sm:block w-1.5 h-1.5 rounded-full`) ans Ende der Zeile verschieben (nur wenn `slot_count > 0`)
- [x] 1.2 Dienst-Punkt aus der Uhrzeitzeile entfernen
- [x] 1.3 Uhrzeitzeile: nach `<span>{g.time}</span>` zwei Spans für RSVP einfügen — `hidden @tile-sm:inline-flex items-center gap-0.5 text-green-600` mit `<Check w-2.5 h-2.5 />{g.confirmed_count}` und `hidden @tile-sm:inline-flex items-center gap-0.5 text-brand-danger` mit `<X w-2.5 h-2.5 />{g.declined_count}`
