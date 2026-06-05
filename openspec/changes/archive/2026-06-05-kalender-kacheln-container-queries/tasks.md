## 1. Abhängigkeit einrichten

- [x] 1.1 `pnpm add -D @tailwindcss/container-queries` im `web/`-Verzeichnis ausführen
- [x] 1.2 Plugin in `web/tailwind.config.js` unter `plugins` eintragen
- [x] 1.3 Custom Container-Breakpoints `tile-sm` (80px) und `tile-md` (120px) in `tailwind.config.js` unter `theme.extend` definieren

## 2. Tageszelle als Container deklarieren

- [x] 2.1 In `KalenderPage.tsx` die Tageszelle (`<div key={day}>`) um die Klasse `@container` ergänzen

## 3. Kachel-Markup für drei Stufen anpassen

- [x] 3.1 **Spielkachel Zeile 1** (Icon + Teamname): Icon immer sichtbar; Teamname nur ab `@tile-sm:inline` sichtbar
- [x] 3.2 **Spielkachel Zeile 2** (Gegner): nur ab `@tile-md:block` sichtbar, darunter `hidden`
- [x] 3.3 **Spielkachel Zeile 3** (Zeit + Dienst-Dot): Zeit immer sichtbar; Dienst-Dot ab `@tile-sm:inline-flex`
- [x] 3.4 **Trainingskachel Zeile 1** (Icon + Teamname): analog zu 3.1
- [x] 3.5 **Trainingskachel Leerzeile**: nur ab `@tile-md:block` sichtbar
- [x] 3.6 **Trainingskachel Zeile 3** (Zeit + RSVP-Counts): Zeit immer; Counts ab `@tile-sm:inline-flex`
- [x] 3.7 `title`-Attribut auf jede Kachel setzen (Vollinfo für Stufe 1 / Screen-Reader)

## 4. Visuelle Prüfung

- [x] 4.1 Dev-Server starten, Kalender auf 360px, 420px, 768px und 1280px Viewport-Breite testen
- [x] 4.2 Sicherstellen dass Swipe-Geste (Monatsnavigation) auf Mobile weiterhin funktioniert
- [x] 4.3 Heute-Kreis und Plus-Button in Stufe 1 prüfen (kein Layout-Overflow)
