# Tasks

> Keine Migration, keine neue Route, kein neuer Broadcast (`SaveLineup` broadcastet
> `games` bereits). Ein Commit pro Task (`feat(games): …`, `feat(termine): …` usw.).
> Voraussetzung fürs Archivieren: `termin-matrix` zuerst archivieren, damit der
> `termin-matrix`-Delta auf die Haupt-Spec trifft.

## 1. Gemeinsame Ableitung (Foundation)

- [x] 1.1 `internal/db/lineup_state.go`: `LineupStateSQL(eventTypeExpr, gameIDExpr, memberIDExpr)` + Konstanten `LineupIn/Out/Open` (design.md §1); Test `lineup_state_test.go` über eine Fixture mit drei Spielen (ohne Aufstellung, mit Mitglied, ohne Mitglied) und einem generischen Event → `open`/`in`/`out`/`NULL`; verifiziert mit `go test ./internal/db/...`

## 2. Backend — Lese-Routen

- [x] 2.1 `internal/games/handler.go` `GetParticipants`: `lineup` je Spielerzeile (Trainerzeilen ohne Feld, auch bei „Sonstiges“) und `lineup_count` in `participantsResponse`; Tests `TestGetParticipants_LineupOpen`, `TestGetParticipants_LineupOutTrotzVerborgenerZeilen` (Aufstellung nur aus Zeilen eines fremden Teams → eigene Zeile `out`, `lineup_count` korrekt), bestehender 404-Test grün; verifiziert mit `go test ./internal/games/...`
- [x] 2.2 `ListMyGames`: `my_lineup` für Stamm-/erweiterten Kader, nicht für reine Trainer und generische Events; `attachChildrenRSVPToGames`: `lineup` in beiden `UNION`-Zweigen; Tests `TestListMyGames_MyLineup_{Open,In,Out}`, `TestListMyGames_MyLineup_TrainerOhneFeld`, `TestListMyGames_ChildLineup`, 401-Fall; verifiziert mit `go test ./internal/games/...`
- [x] 2.3 `internal/attendance/matrix.go`: gebündelter `game_lineup`-Lookup für Spielspalten, `matrixCell.Lineup` (design.md §4); Tests: Spielspalte mit Aufstellung → `in`/`out`, ohne → `open`, Trainingsspalte ohne Feld, 403 unverändert, plus Deckungsgleichheits-Test Matrix ↔ `LineupStateSQL` auf derselben Fixture; verifiziert mit `go test ./internal/attendance/...`
- [x] 2.4 `internal/calendar/handler.go`: `kaderMembership` mit `kind` (0 Stamm, 1 Trainer, 2 erweitert), `lineup_exists`/`in_lineup` durch das Fragment ersetzen, `resolveLineupState` → Mapping Code → Kennwort/Satz für `kind IN (0,2)`, `kaderLabel` hängt das Kennwort auch ohne `erw. Kader` an; Tests für die Szenarien „Stammspieler bekommt den Status", „Stammspieler bei offener Aufstellung", „Doppelte Zugehörigkeit", „Trainer bekommt keinen Status"; bestehende Tests des erweiterten Kaders unverändert grün; verifiziert mit `go test ./internal/calendar/...`

## 3. Frontend — gemeinsame Bausteine

- [x] 3.1 `web/src/lib/lineup.ts` (Typ, `LINEUP_LABEL`, `LINEUP_SURFACE`, `LINEUP_TEXT`, gemäß design.md §4, alle Elemente mit 1 px Rahmen und `rounded-md`), `web/src/components/LineupBadge.tsx` und `web/src/components/LineupCheckbox.tsx` (`role="checkbox"`, `aria-checked`, Leertaste/Enter, disabled-Variante) (Text-Kennzeichen ohne Symbol, Breite `w-32`, Höhe der Zu-/Absage-Knöpfe, optional freier Text für „N aufgestellt"); Vitest prüft alle drei Bezeichnungen/Flächenklassen und dass `undefined` nichts rendert; verifiziert mit `pnpm -C web test lineup`

## 4. Frontend — Ansichten

- [x] 4.1 `TerminePage.tsx` (Liste): Typen um `my_lineup`/`children_rsvp[].lineup` erweitern, `LineupBadge` in der Zeile „Ich" und je Kind-Zeile bei Heim-/Auswärtsspielen; Vitest: Stammspieler `in` → „aufgestellt", `open` → „Aufstellung offen", Kind-Zeilen getrennt, Trainer ohne Feld → kein Kennzeichen; Sichtprüfung mobil (Umbruch unter den Knöpfen)
- [x] 4.2 `terminMatrix.ts` + `TerminMatrix.tsx`: `MatrixCell.lineup`, Box gleicher Geometrie in jeder Zelle mit `LINEUP_SURFACE` (grün / grau / gestrichelt), unsichtbar für Trainings/Sonstige, Zelltitel „<Rückmeldung> · <Aufstellung>", drei Farbfelder in der Legende; Vitest für Flächenklasse je Zustand, Titel, Trainingsspalte ohne Fläche; Sichtprüfung gegen den Entwurf (Kontrast grüner Daumen auf grüner Fläche, gelbe Eigenzeile)
- [x] 4.3 `TermineDetailPage.tsx`: `LineupCheckbox` je Spielerzeile (grün mit Haken / grau gefüllt / gestrichelt), für Spieler/Eltern `disabled` aus `row.lineup`, für Trainer bedienbar mit `open` aus `lineupMap` abgeleitet (design.md §2); Kartenkopf `LineupBadge` „Aufstellung offen" bzw. „N aufgestellt" (`lineup_count`, beim Trainer nach optimistischem Update); Vitest: Spieler sieht bei leerer Aufstellung kein „nicht aufgestellt", Trainer-Klick wechselt Kopf von „Aufstellung offen" auf „1 aufgestellt"; Vitest für Tastaturbedienung der Trainer-Checkbox
- [x] 4.4 Zusage-Grün auf `/termine`: `green-600` → `brand-green` für aktiven „Zusagen“-Knopf (Liste, Tabellen-Dialog), Zusage-Zähler der Karten, Zusage-Zähler im Kopf der Detailseite (`green-100/700` → `brand-green`/weiß) und `RsvpIcon` der Detailseite (design.md §4); `grep -n green-600 web/src/pages/TerminePage.tsx web/src/pages/TermineDetailPage.tsx` liefert nur noch Treffer außerhalb der Rückmeldung (oder keine); bestehende Vitest-Tests grün

## 5. Doku

- [x] 5.1 `docs/anleitung-trainer.md`, `web/public/benutzerhandbuch.html`, `docs/schulung/folien.txt`: Bezeichnungen „aufgestellt / nicht aufgestellt / Aufstellung offen" statt „nominiert", Hinweis auf Anzeige in Liste/Tabelle/Kalender-Abo (jetzt auch Stammkader), überholte Aussage „Änderung verschickt keine Benachrichtigung" korrigieren; `make folien` läuft durch; `grep -rn -i nominier docs web/public web/src internal` liefert nur noch Code-Kommentare ohne Nutzerwirkung

## 6. Abschluss

- [x] 6.1 `make test`, `pnpm -C web build`, `pnpm -C web test`, `pnpm -C web lint`, `openspec validate aufstellung-status-termine` grün; `/verify-change` ohne Befund
- [x] 6.2 Live-Prüfung im Browser (lokal): Trainer setzt auf der Detailseite das erste Häkchen → Liste und Tabelle eines zweiten Spieler-Logins wechseln ohne Reload von „Aufstellung offen" auf „nicht aufgestellt"/„aufgestellt"
