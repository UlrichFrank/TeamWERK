## 1. Gruppierungs-Helper

- [ ] 1.1 `web/src/lib/videoGroups.ts` anlegen: Typen (`VideoGroup`), Pure-Function `groupVideos(videos: Video[]): VideoGroup[]` (Schlüssel `game:<id>` / `title:<trim(title)>` / `video:<id>`; Sortierung `created_at ASC`, Tiebreaker `id ASC`)
- [ ] 1.2 Helper `suggestNextTitle(group: VideoGroup): string` (erkennt „1. Halbzeit" → „2. Halbzeit", sonst „Video N+1")
- [ ] 1.3 Vitest-Unit-Tests für beide Helper (Spiel-Gruppe, Titel-Gruppe, Einzel-Video ohne Spiel/Titel, Sortier-Tiebreaker, Halbzeiten- und generischer Titel-Vorschlag)

## 2. Video-Liste (`VideosPage.tsx`)

- [ ] 2.1 Liste über `groupVideos(videos)` rendern: Einzel-Video-Gruppen als kompakte Karte (Klick → Detail), Mehrfach-Gruppen als Sammel-Karte (Spiel-/Titel-Header + Anzahl + Vorschau)
- [ ] 2.2 Sammel-Karten standardmäßig eingeklappt; lokaler `useState` pro Karten-ID für aufklappen/zuklappen
- [ ] 2.3 Aufgeklappte Karte listet alle Gruppen-Mitglieder in Sortierreihenfolge, jeder Eintrag verlinkt auf die Video-Detailseite
- [ ] 2.4 Sicherstellen, dass kein neuer API-Call nötig ist (Gruppierung auf dem bestehenden `GET /api/videos`-Ergebnis)
- [ ] 2.5 brand-Tokens und lucide-Icons gemäß `docs/agent/05-frontend.md` verwenden (kein `bg-gray-*`, keine Emojis)

## 3. Video-Detailseite (`VideoDetailPage.tsx`)

- [ ] 3.1 Aktuelles Video samt Geschwistern bestimmen: gesamte Video-Liste laden (oder aus Cache), `groupVideos` anwenden, eigene Gruppe finden, eigenes Video herausfiltern
- [ ] 3.2 Sektion „Weitere Videos zu …" unter dem Player rendern, nur wenn ≥ 1 weiteres Gruppen-Mitglied existiert
- [ ] 3.3 Jeder Geschwister-Eintrag zeigt Titel + Upload-Datum und verlinkt per Router auf die jeweilige Detail-URL
- [ ] 3.4 brand-Tokens / lucide-Icons konsistent zur Liste

## 4. Upload-Hinweis (`VideoUploadPage.tsx`)

- [ ] 4.1 Beim Render einmalig `GET /api/videos` laden und mit `groupVideos` indexieren (Map `groupKey → VideoGroup`)
- [ ] 4.2 Effekt auf Spiel-Auswahl: bei Treffer in Map (`game:<id>`) Info-Alert „Es gibt bereits N Video(s) zu diesem Spiel — dies wird Video Nr. N+1" einblenden, Titel-Placeholder via `suggestNextTitle` setzen
- [ ] 4.3 Effekt auf Titel-Eingabe (debounced 150 ms) ohne Spiel-Auswahl: Treffer-Logik gegen `title:<trim(title)>`-Schlüssel; gleicher Alert, kein Title-Placeholder-Overwrite (User tippt gerade selbst)
- [ ] 4.4 Hinweis ist nicht-blockierend: Submit bleibt jederzeit möglich
- [ ] 4.5 Alert-Klassen gemäß `docs/agent/05-frontend.md` (`Alert Info`-Variante)

## 5. Tests

- [ ] 5.1 Backend-Test in `internal/videos/`: `GET /api/videos` liefert beide Videos zurück, wenn zwei Zeilen mit identischer `game_id` für ein lesbares Team existieren (Happy-Path Multi-Video pro Spiel)
- [ ] 5.2 Backend-Test: `POST /api/videos` Upload-Init zum gleichen Spiel ein zweites Mal erlaubt (kein 409/Konflikt), Eigentümer-Bindung aus `preUploadCreate` greift weiterhin
- [ ] 5.3 Vitest für `VideosPage`: bei 2 Videos mit gleicher `game_id` wird **eine** Sammel-Karte gerendert; Aufklappen zeigt beide Einträge
- [ ] 5.4 Vitest für `VideoUploadPage`: Auswahl eines Spiels mit bestehendem Video erzeugt Info-Alert und setzt Titel-Placeholder „2. …"

## 6. Doku, CHANGELOG, Commit

- [ ] 6.1 `web/public/CHANGELOG.md`: Zeile `[feat] videos: mehrere Videos pro Spiel/Titel als Gruppen anzeigen + Upload-Hinweis` unter heutigem Datum ergänzen
- [ ] 6.2 Conventional Commit `feat(videos): Mehrere Videos pro Spiel/Titel gruppieren` (inkl. CHANGELOG)
- [ ] 6.3 `/verify-change` ausführen (Build/Test/Lint + Projekt-Invarianten)
- [ ] 6.4 Push (Pre-Push-Gate grün), anschließend `make deploy`
- [ ] 6.5 OpenSpec-Change `videos-grouping` via `openspec archive` ins Archiv überführen
