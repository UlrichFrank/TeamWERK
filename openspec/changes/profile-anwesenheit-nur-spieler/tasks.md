# Implementation Tasks

## 1. Frontend-Regel für Tab-Sichtbarkeit

- [ ] 1.1 In `web/src/pages/ProfilePage.tsx` `showAttendanceTab` von
  `ownMember !== null || children.length > 0` auf
  `hasPlayerFunction(ownMember) || children.some(hasPlayerFunction)` umstellen
  (kleine lokale Helper-Funktion `hasPlayerFunction(m): m?.club_functions?.includes('spieler')`).

## 2. Frontend-Regel für Selbst-/Kind-Auswahl

- [ ] 2.1 In `web/src/pages/ProfilAnwesenheitPage.tsx` `MemberRef` um
  `club_functions?: string[]` erweitern.
- [ ] 2.2 In `ProfilAnwesenheitContent` die Options-Konstruktion (Zeile 31-36)
  auf denselben `hasPlayerFunction`-Filter aufsetzen. `forcedMemberId`-Pfad
  unverändert lassen (Trainer-Drilldown).
- [ ] 2.3 Initial-`selectedId`-Fallback in `useEffect` (Zeile 24-26) anpassen,
  damit nur ein Spieler als Default gewählt wird.

## 3. Tests (vitest, jsdom)

- [ ] 3.1 Neue Test-Datei `web/src/pages/__tests__/ProfilePage.attendance-tab.test.tsx`
  mit den sechs Konstellationen aus `proposal.md#Test-Anforderungen`. `api.get`
  mocken (`vi.mock('../lib/api')`), `AttendanceStatsView` mocken (rendert
  Marker mit memberId), `useAuth` mocken (User).

## 4. Manueller Rauchtest

- [ ] 4.1 Dev-Server neu starten, `http://localhost:5173/profil` als Thomas
  Eisele (Vereinsfunktion `trainer`, kein `spieler`) öffnen → Tab weg.
- [ ] 4.2 Als Spieler-Nutzer → Tab da, eigene Zahlen sichtbar.
- [ ] 4.3 Als Trainer via `/team/{id}/anwesenheit` einen Spieler drillen
  (`openMember`) → Detailseite `/profil/anwesenheit?member=X` rendert Stats,
  keine leere Auswahl.

## 5. Gate + Commit

- [ ] 5.1 `pnpm -C web test` grün (neuer Test + keine Regression).
- [ ] 5.2 `pnpm -C web build` grün.
- [ ] 5.3 `openspec validate profile-anwesenheit-nur-spieler --strict`.
- [ ] 5.4 Commit `fix(profile): Anwesenheit-Tab nur für Spieler und Eltern von Spielern`.
- [ ] 5.5 Archivierung nach Grün (`openspec archive profile-anwesenheit-nur-spieler`).
