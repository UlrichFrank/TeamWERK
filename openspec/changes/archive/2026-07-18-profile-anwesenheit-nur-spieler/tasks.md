# Implementation Tasks

## 1. Frontend-Regel für Tab-Sichtbarkeit

- [x] 1.1 In `web/src/pages/ProfilePage.tsx` `showAttendanceTab` von
  `ownMember !== null || children.length > 0` auf
  `hasPlayerFunction(ownMember) || children.some(hasPlayerFunction)` umstellen
  (kleine lokale Helper-Funktion `hasPlayerFunction(m): m?.club_functions?.includes('spieler')`).

## 2. Frontend-Regel für Selbst-/Kind-Auswahl

- [x] 2.1 In `web/src/pages/ProfilAnwesenheitPage.tsx` `MemberRef` um
  `club_functions?: string[]` erweitern.
- [x] 2.2 In `ProfilAnwesenheitContent` die Options-Konstruktion (Zeile 31-36)
  auf denselben `hasPlayerFunction`-Filter aufsetzen. `forcedMemberId`-Pfad
  unverändert lassen (Trainer-Drilldown).
- [x] 2.3 Initial-`selectedId`-Fallback in `useEffect` (Zeile 24-26) anpassen,
  damit nur ein Spieler als Default gewählt wird.

## 3. Tests (vitest, jsdom)

- [x] 3.1 Neue Test-Datei `web/src/pages/__tests__/ProfilePage.attendance-tab.test.tsx`
  mit **10** Konstellationen (5× Tab-Sichtbarkeit über `ProfilePage`, 5×
  Options-Filter über `ProfilAnwesenheitContent` inkl. `forcedMemberId`-Bypass).
  `/profile/me` per `setupApiMock`+`reset()` gemockt, `AttendanceStatsView`
  gemockt (rendert `stats:{memberId}`-Marker), `useAuth` via `renderAsPersona`.

## 4. Manueller Rauchtest

- [ ] 4.1 Dev-Server neu starten, `http://localhost:5173/profil` als Thomas
  Eisele (Vereinsfunktion `trainer`, kein `spieler`) öffnen → Tab weg.
- [ ] 4.2 Als Spieler-Nutzer → Tab da, eigene Zahlen sichtbar.
- [ ] 4.3 Als Trainer via `/team/{id}/anwesenheit` einen Spieler drillen
  (`openMember`) → Detailseite `/profil/anwesenheit?member=X` rendert Stats,
  keine leere Auswahl.

## 5. Gate + Commit

- [x] 5.1 `pnpm -C web test` grün (72 Files, 602 Tests, kein Regressionsverlust).
- [x] 5.2 `pnpm -C web build` grün.
- [x] 5.3 `openspec validate profile-anwesenheit-nur-spieler --strict`.
- [x] 5.4 Commit `29acd4a fix(profile): Anwesenheit-Tab nur für Spieler und Eltern von Spielern`.
- [x] 5.5 Archivierung nach Grün.
