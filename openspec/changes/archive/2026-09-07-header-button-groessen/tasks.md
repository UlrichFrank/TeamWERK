## 1. Konstanten anlegen

- [x] 1.1 `web/src/lib/buttonStyles.ts` anlegen mit `HEADER_CTRL`, `HEADER_CTRL_ICON`, `HEADER_PRIMARY`, `HEADER_NEUTRAL`, `HEADER_SPLIT_MAIN`, `HEADER_SPLIT_CARET` sowie `BTN_PRIMARY`, `BTN_SMALL`, `BTN_DANGER` (Werte siehe `design.md` Entscheidung 2 und Spec-Delta). Datei-Kommentar: warum feste Höhe statt Padding (Verweis auf `index.css` / `ios-input-zoom-prevention`), und dass Header-Control und Primary zwei Rollen sind.
- [x] 1.2 `docs/agent/05-frontend.md`: vierten verbindlichen Klassen-String „Button Header" in die Liste aufnehmen, Rollentrennung Header/Formular benennen, auf `web/src/lib/buttonStyles.ts` als Fundstelle verweisen.

## 2. Kopfzeilen-Aktionen, gestapeltes Layout

- [x] 2.1 `MembersPage.tsx` (452 „+ Neu" → `HEADER_SPLIT_MAIN`, 459 Caret → `HEADER_SPLIT_CARET`) sowie Suchfeld (405) und die beiden Selects (409, 417) auf `h-11 sm:h-[30px]` umstellen.
- [x] 2.2 `AdminUsersPage.tsx` (427 / 433 Split-Button) plus Suchfeld und Selects der Kopfzeile.
- [x] 2.3 `VideosPage.tsx` (200 Upload-Button) plus Status-Select (193). Der „N neue Videos"-Pill (216) bleibt unangetastet — Pill, kein Header-Control.
- [x] 2.4 `AdminDutyTypesPage.tsx` (390) und `AdminDutyTemplatesPage.tsx` (634) — heute `px-4 py-1.5` ohne Rahmen (28 px).
- [x] 2.5 `DocumentsPage.tsx` (755 und 764, zwei Buttons nebeneinander).
- [x] 2.6 `AdminKaderPage.tsx` (386 primär → `HEADER_PRIMARY`; 392 und 398 sekundär → `HEADER_NEUTRAL`). Danach auf 375 px prüfen, ob die drei Buttons weiter nebeneinander passen.
- [x] 2.7 `AdminVenuesPage.tsx` (169 / 176 Split-Button) — heute Formular-Größe, wird deutlich kleiner. Split-Rundung und Caret-Trennlinie prüfen.
- [x] 2.8 `VideoUploadPage.tsx` (425): Abbrechen-X neben `<h1>` — heute `w-11 h-11 sm:w-9 sm:h-9` (44/36 px), rahmenlos. Auf `HEADER_CTRL_ICON` + neuen Farbsatz `HEADER_GHOST` (`border-transparent`, damit die Höhe die der Nachbarn bleibt).
- [x] 2.10 Korrektur zu diesem Task-Plan: `AdminTrainingsPage.tsx` (438, Aktion im Serien-Tab), `VideoUploadPage.tsx` (410, „Zum Video" in der Erfolgs-Karte) und `ProfilTrainingstagebuchPage.tsx` (149, Aktion in `ProfilTrainingstagebuchContent`) sind **keine** Kopfzeilen-Controls — sie stehen nicht neben einer `<h1>`. Sie behalten die Formular-Größe und laufen über Task 4.3 (Primary-Dedupe).
- [x] 2.9 `VideoDetailPage.tsx` (320 sekundär, 327 destruktiv) — Danger-Variante in Header-Größe ergänzen, falls in 1.1 noch nicht vorhanden.

## 3. Einzeilige Filterleiste

- [x] 3.1 `components/EventTypeFilter.tsx` (49 Chip, 72 Compact-Trigger) auf `HEADER_CTRL` / `HEADER_CTRL_ICON` umstellen; aktiver Zustand kommt weiter aus `getEventColors(type).filter`, inaktiver aus `HEADER_NEUTRAL`.
- [x] 3.2 `components/EventSearchInput.tsx` (46): `h-[30px]`/`h-7` durch `h-11 sm:h-[30px]` ersetzen (eine Höhe für beide Modi, Breite bleibt modusabhängig). Datei-Kommentar anpassen: Fixhöhen-Begründung behalten, Absatz zur bewussten 44-px-Unterschreitung entfernen.
- [x] 3.3 Toggle-Chips auf `DutyPage.tsx` (294, 306, 319), `TerminePage.tsx` (519), `KalenderPage.tsx` (979) und `MitfahrgelegenheitenPage.tsx` (852) auf die Konstanten umstellen; `compact ? 'px-2' : 'px-3'` bleibt als Breitenwechsel erhalten.
- [x] 3.4 Kopfzeilen-Buttons auf `KalenderPage.tsx` (1002 Haupt, 1019 Caret) auf `HEADER_SPLIT_MAIN`/`HEADER_SPLIT_CARET`.
- [x] 3.5 Team-Selects in der Leiste auf dieselbe Höhe: `DutyPage.tsx`, `TerminePage.tsx`, `TeamTrainingstagebuchPage.tsx` — beim Umbau kamen drei weitere gleichartige Fundstellen dazu (`KalenderPage.tsx`, `MitfahrgelegenheitenPage.tsx`, `TeamAnwesenheitPage.tsx`), insgesamt sechs. `hidden sm:block` bleibt, wo es steht.

## 4. Aufräumen

- [x] 4.1 Modul-lokale `BTN_PRIMARY`-Kopien entfernen und aus `lib/buttonStyles.ts` importieren: `pages/admin/BeitragslaufPage.tsx` (11), `pages/admin/TresorPage.tsx` (11), `pages/admin/WartungsmodusPage.tsx` (9).
- [x] 4.2 `DashboardPage.tsx` (667, Retry-Button im Fehlerzustand) auf `BTN_PRIMARY` ziehen — heute `rounded` statt `rounded-md`, ohne `text-brand-black` und ohne Disabled-Zustand. Kein Header-Control: der Button steht im zentrierten Fehlerblock, nicht in der Kopfzeile.
- [x] 4.3 Restliche Kopien der vier Strings auf den Import umstellen: 81 von 86 Fundstellen (68 inline im JSX, 13 als modul-lokale Konstante). Fünf bleiben mit Begründung in der Gate-Allowlist — Secondary-Buttons mit abweichendem Hover; der Secondary-Button ist in `component-standards` gar nicht definiert und existiert im Bestand in vier Varianten, das zu vereinheitlichen ist eine eigene Design-Entscheidung.

## 5. Absicherung

- [x] 5.1 `web/src/lib/__tests__/buttonStyles.gate.test.ts`: scannt `web/src/pages/` und `web/src/components/` auf die vier Metriken als Literal, meldet Datei + Zeile, Allowlist mit Begründung je Eintrag, verwaister Eintrag lässt den Test fehlschlagen (Muster: `internal/arch/broadcast_test.go`).
- [x] 5.2 Vitest-Test für die Höhen-Zusage: eine Kopfzeile rendern und prüfen, dass Button, Suchfeld und Select dieselben Höhen-Klassen tragen (jsdom misst keine Pixel — die Assertion geht auf die Klassen, nicht auf `getBoundingClientRect`).
- [x] 5.3 `pnpm -C web build`, `pnpm -C web test`, `pnpm -C web lint` grün.
- [x] 5.4 Nachgemessen statt angeschaut (Chrome gegen die E2E-Seed-Binary, `getBoundingClientRect().height` je Header-Control): Desktop **30 px einheitlich** auf allen 11 geprüften Seiten, Mobile **44 px einheitlich** auf allen 9. Kein horizontaler Overflow (`scrollWidth == clientWidth`) auf `/mitglieder`, `/termine`, `/dienste`, `/kalender`, `/veranstaltungsorte`, `/videos`, `/mitfahrgelegenheiten`. Chromes Fenster-Minimum auf macOS ist 500 px statt 375 px — für die Aussage unerheblich, der `sm:`-Breakpoint liegt bei 640 px.
- [x] 5.5 `make test-e2e` (Playwright) — 13 Tests grün.
- [x] 5.6 `openspec validate header-button-groessen --strict` und `/verify-change` — beide grün.
