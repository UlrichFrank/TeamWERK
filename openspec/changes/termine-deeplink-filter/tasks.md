# Implementation Tasks

## 1. Frontend: URL-driven filters auf /termine

- [x] 1.1 In `web/src/pages/TerminePage.tsx` `useSearchParams` von `react-router-dom` einbinden
- [x] 1.2 Helper `parseFilters(sp: URLSearchParams)` schreiben — returns `{ team, types, past, focus }`
- [x] 1.3 `useState`-Filter (`filterTeamId`, `filterTypes`, `showPast`) durch abgeleitete Werte aus `searchParams` ersetzen
- [x] 1.4 Handler `updateFilter(patch)` implementieren, der `setSearchParams(next, { replace: true })` aufruft und Default-Werte aus der URL entfernt (saubere URL)
- [x] 1.5 Default-Behaviour absichern: ohne Query-Param verhält sich Seite identisch (Smoke-Test manuell)
- [x] 1.6 Ungültige Werte (`team=abc`, `types=foo`) müssen ignoriert werden — Unit-fähigen Parser bevorzugen oder Defensiv-`if`-Schritte

## 2. Frontend: Focus-Scroll + Highlight

- [x] 2.1 Jede Termin-Karte erhält `id={\`termin-\${kind}-\${id}\`}` (Wrapper-`div` mit `ref` oder per Attribut)
- [x] 2.2 `useEffect`, der bei (`focus`, `loading=false`, `visibleTermine`) das DOM-Element sucht, `scrollIntoView({ behavior: 'smooth', block: 'center' })` aufruft und Tailwind-Klassen `ring-2 ring-brand-yellow` für ~2s setzt
- [x] 2.3 Wenn `focus`-Termin in der Vergangenheit liegt, `past=1` automatisch in URL setzen (einmalig, in einem separaten Effect)
- [x] 2.4 Wenn `focus`-Termin durch Team-/Type-Filter ausgeblendet würde, ihn trotzdem in `visibleTermine` aufnehmen (Sonderbehandlung für die fokussierte ID)
- [x] 2.5 Wenn `focus`-Termin nicht in den geladenen Daten existiert: Inline-Info „Dieser Termin ist nicht verfügbar" oberhalb der Liste rendern, Liste rendert sonst normal
- [x] 2.6 Manueller Test: Push-Link `/termine?focus=game-<id>` scrollt korrekt; ungültiges Format `/termine?focus=foo` rendert ohne Fehler

## 3. Frontend: EventInfoModal-Button

- [x] 3.1 In `EventInfoModal` (Pfad via `grep -r "EventInfoModal" web/src` lokalisieren) Prop `kind: 'game' | 'training'` und `id: number` sicherstellen (vermutlich schon vorhanden)
- [x] 3.2 Button „In Terminen öffnen" im Footer hinzufügen mit Primary-Button-Klassen (`bg-brand-yellow text-brand-black …`)
- [x] 3.3 OnClick: `onClose()` + `navigate(\`/termine?focus=\${kind}-\${id}\`)`
- [x] 3.4 Button nur rendern, wenn `kind ∈ {game, training}` (kein Render für reine Kalendereinträge/Dienste)
- [ ] 3.5 Manueller Test: Klick im Kalender-Modal landet auf gescrollter `/termine`-Karte

## 4. Backend: Push-URL für Spiele

- [x] 4.1 In `internal/games/handler.go` Zeile ~716 (`Neues Spiel`): `"/kalender"` ersetzen durch `fmt.Sprintf("/termine?focus=game-%d", gameID)`
- [x] 4.2 In `internal/games/handler.go` Zeile ~807 (`Spielinfo geändert`): selber Ersatz mit der jeweiligen Spiel-ID
- [x] 4.3 In `internal/games/handler.go` Zeile ~899 (`Spiel abgesagt`): `"/kalender"` → `"/termine"` (kein Focus, da Spiel gelöscht)
- [x] 4.4 Bestehende Tests in `internal/games/handler_test.go` weiter grün — Test-Routen werden nicht angefasst

## 5. Backend: Push-URL für Trainings

- [x] 5.1 In `internal/trainings/handler.go` Zeile ~499 (`Training abgesagt` Einzel-Session): `"/training"` → `"/termine"` (kein Focus, Session gelöscht)
- [x] 5.2 In `internal/trainings/handler.go` Zeile ~619 (`Training geändert`): `"/training"` → `fmt.Sprintf("/termine?focus=training-%d", sessionID)`
- [x] 5.3 In `internal/trainings/handler.go` Zeile ~465 (`Trainingsserie gelöscht`): `"/training"` → `"/termine"`
- [x] 5.4 Sicherstellen, dass `sessionID` an den jeweiligen Stellen verfügbar ist (aus `r.PathValue("id")` oder Request-Body)

## 6. Backend: E-Mail-Templates Audit

- [x] 6.1 `grep -rn "kalender\|/training" internal/mailer internal/scheduler internal/games internal/trainings` ausführen
- [x] 6.2 Treffer, die in E-Mail-Bodies oder Reminder-Mails zu Spielen/Trainings führen, auf `/termine?focus=…` umstellen
- [x] 6.3 Treffer in Dienst-Reminder-Mails (`duty-reminder-emails`) bleiben unverändert auf `/dienste`
- [x] 6.4 Stichprobentest: Reminder-Mail manuell triggern (sofern Test-Helper existiert) oder mindestens Build + `go test ./...` grün

## 7. Verifikation & Commit

- [x] 7.1 `pnpm --filter ./web build` lokal grün
- [x] 7.2 `go build ./...` und `go test ./...` lokal grün
- [ ] 7.3 Manueller Walkthrough lokal: (a) Filter via URL setzen, (b) Filter ändern → URL aktualisiert, (c) `/termine?focus=game-<id>` scrollt, (d) Kalender-Modal-Button springt nach `/termine`
- [ ] 7.4 Conventional Commits nach jedem Task-Block (siehe CLAUDE.md): `feat(termine): URL-driven Filter und Focus-Param`, `feat(games): Push-Link auf konkretes Spiel`, `feat(trainings): Push-Link auf konkretes Training`, `feat(kalender): EventInfoModal-Button „In Terminen öffnen"`
- [ ] 7.5 OpenSpec Change archivieren (`/opsx:archive termine-deeplink-filter`) nach Merge
