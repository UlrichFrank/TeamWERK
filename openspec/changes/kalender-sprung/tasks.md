# Tasks

## 1. Ziel-URL

- [x] 1.1 Helfer `calendarLink(kind: 'game' | 'training', id, date)` in `web/src/lib/calendarLink.ts` anlegen, der `/kalender?date=<YYYY-MM-DD>&focus=<kind>-<id>` baut (Datum per `.slice(0, 10)`). Verifikation: Vitest `calendarLink.test.ts` mit ISO-Timestamp-Datum (`2026-10-11T00:00:00Z` → `date=2026-10-11`) und beiden Kinds.

## 2. Aktion auf /termine und /dienste

- [x] 2.1 `TerminePage.tsx`: Icon-Button `CalendarDays` (`aria-label`/`title` „Im Kalender öffnen“, `p-2`, nur `brand-*`-Farben) rechts in der Kopfzeile jeder Spiel-/Event- und Trainings-Karte; `stopPropagation`, dann `navigate(calendarLink(...))`. Verifikation: Vitest — Klick navigiert zu `/kalender?date=…&focus=game-17` bzw. `training-42` und NICHT zur Detailseite.
- [x] 2.2 `DutyPage.tsx`: derselbe Icon-Button im Block-Kopf rechts neben den Team-Namen, nur bei gesetztem `game_id`. Verifikation: Vitest — Klick navigiert zu `/kalender?date=…&focus=game-<id>`; ein Block ohne `game_id` hat keinen Button.

## 3. Fokus im Kalender

- [x] 3.1 `KalenderPage.tsx`: `focus` (`^(game|training)-(\d+)$`) aus den Search-Params lesen; ungültige Werte ignorieren. Verifikation: Vitest — `?date=2026-10-11&focus=foobar` zeigt Oktober 2026 ohne Dialog.
- [x] 3.2 Nach dem Laden von Spielen und Trainings den Termin in den geladenen Listen suchen, `infoItem` setzen (gleiche Form wie beim Kachel-Klick) und `focus` per `setSearchParams(…, { replace: true })` entfernen; ein Ref verhindert erneutes Öffnen nach SSE-Reloads. Verifikation: Vitest — `?date=2026-10-11&focus=game-17` öffnet den Dialog für Spiel 17, danach steht `date`, aber kein `focus` in der URL; Schließen öffnet ihn nicht erneut.
- [x] 3.3 Fallback-Einzelabruf über `GET /api/games/{id}` bzw. `GET /api/training-sessions/{id}`, wenn der Termin nicht in den geladenen Listen steht; Antwort auf die Form von `EventInfoModal` abbilden (Feldform gegen die echten Go-Antworten prüfen). 403/404 → Hinweis „Dieser Termin ist nicht verfügbar“ (Alert-Info-Stil). Verifikation: Vitest mit MockAdapter — Spiel fehlt in `/games`, Einzelabruf liefert es → Dialog offen; Einzelabruf 404 → Hinweis sichtbar, kein Dialog.
- [x] 3.4 Termin-Kacheln im Gitter mit IDs `kalender-game-<id>` / `kalender-training-<id>` versehen und die fokussierte Kachel ca. 2 s mit `ring-2 ring-brand-yellow` hervorheben. Verifikation: Vitest — nach dem Fokus trägt die Kachel die Ring-Klasse.
- [ ] 3.5 Spec-Hinweis im Kalender-Kapitel der Agent-Doku nur ergänzen, falls beim Implementieren eine nicht-ableitbare Falle auftaucht (z. B. Feldform der Einzelrouten); sonst entfällt. Verifikation: `git diff docs/agent` leer oder begründet.

## 4. Abschluss

- [ ] 4.1 Gesamtprüfung: `pnpm -C web test`, `pnpm -C web lint`, `pnpm -C web build`, `openspec validate kalender-sprung --strict` grün; manuell im Browser von `/termine` und `/dienste` in den Kalender springen (Desktop und Mobile-Breite).
- [ ] 4.2 Folgebefund festhalten: Kalender lädt Spiele über `GET /api/games?limit=500`, der Server deckelt auf 200 — späte Saisontermine fehlen im Gitter. Verifikation: Notiz/Issue angelegt und im Abschlussbericht genannt.
