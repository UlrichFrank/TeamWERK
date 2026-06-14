## 1. AppShell — Zurück-Button-Logik

- [x] 1.1 `canGoBack`-State in AppShell einführen: `useState(() => (window.history.state?.idx ?? 0) > 0)` + `useEffect([location])` zum Aktualisieren
- [x] 1.2 Mobile-Header: `← Zurück`-Button (mit `ChevronLeft`) zwischen Hamburger-Button und „TeamWERK"-Titel einbauen, sichtbar wenn `canGoBack`
- [x] 1.3 Desktop: `← Zurück`-Button als kompakte Zeile direkt vor `<Outlet />` einbauen, sichtbar wenn `canGoBack`

## 2. Lokale Zurück-Buttons entfernen

- [x] 2.1 `TermineDetailPage.tsx`: beide lokalen Zurück-Buttons entfernen (oben ~Z. 221, unten ~Z. 315 inkl. umgebender Container-Elemente)
- [x] 2.2 `MeinTeamPage.tsx`: lokalen Zurück-Button entfernen (~Z. 205)
- [x] 2.3 `SpieltagDetailPage.tsx`: lokalen „← Zurück zum Spielplan"-Link entfernen (~Z. 195)
- [x] 2.4 `MembersPage.tsx`: lokalen Zurück-Button entfernen (~Z. 556) — war kein navigate(-1), sondern Wizard-Step-Button; kein Eingriff nötig
