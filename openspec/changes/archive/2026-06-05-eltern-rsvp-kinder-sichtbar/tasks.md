## 1. Backend — children_rsvp in Terminliste

- [x] 1.1 `sessionListItem` in `internal/trainings/handler.go` um `ChildrenRSVP []childRSVP` erweitern (Typ: `{MemberID int, Name string, RSVP *string}`)
- [x] 1.2 In `ListSessions`: nach dem Haupt-Query, wenn `claims.IsParent`, Batch-Query ausführen die alle Kinder-RSVPs für alle Session-IDs des Zeitraums lädt
- [x] 1.3 Ergebnis der Batch-Query nach `training_id` gruppieren und jedem `sessionListItem` zuweisen
- [x] 1.4 `gameListItem` in `internal/games/handler.go` analog um `ChildrenRSVP []childRSVP` erweitern
- [x] 1.5 In `ListMyGames`: analog zu 1.2–1.3 für game_responses und game_id

## 2. Backend — GetAttendances für alle öffnen

- [x] 2.1 `attendanceItem`-Struct in `internal/trainings/handler.go` um Feld `Reason *string` ergänzen
- [x] 2.2 In `GetAttendances`: `hasTeamAccess`-Guard durch dreistufige Prüfung ersetzen: (a) admin/trainer → volles Recht, (b) Spieler der im Kader ist → Recht, (c) Elternteil mit Kind im Kader → Recht, sonst 403
- [x] 2.3 SQL-Query in `GetAttendances` um `LEFT JOIN training_responses tr ON tr.training_id = ? AND tr.member_id = m.id` erweitern, `tr.reason` selektieren
- [x] 2.4 Reason-Filterlogik implementieren: `childMemberIDs`-Map für IsParent laden, `canSeeReason`-Check analog `GetSession`
- [x] 2.5 Für Nicht-Trainer: `present` immer als `nil` setzen (nicht aus `ta.present` lesen)

## 3. Backend — ListGameResponses: vollständige Kaderliste

- [x] 3.1 In `ListGameResponses` (`internal/games/handler.go`): Query umstellen von `FROM game_responses gr JOIN members` auf Kader-Basis (alle `kader_members` für die Teams des Spiels in der aktiven Saison), per LEFT JOIN auf `game_responses` ergänzen
- [x] 3.2 Einträge ohne RSVP erscheinen mit `status: null` in der Response
- [x] 3.3 Reason-Filterlogik bereits vorhanden — sicherstellen dass sie für die neue Basis-Query korrekt greift

## 4. Frontend — TerminePage: per-Kind-RSVP für Eltern

- [x] 4.1 `Session`- und `Game`-Interface in `TerminePage.tsx` um `children_rsvp?: ChildRSVP[]` erweitern (`interface ChildRSVP { member_id: number; name: string; rsvp: string | null }`)
- [x] 4.2 `isParent`-Flag aus AuthContext ableiten (`user?.is_parent === true`)
- [x] 4.3 `respondTraining`-Funktion: optionalen Parameter `memberId?: number` ergänzen, im API-Body mitschicken wenn vorhanden
- [x] 4.4 `respondGame`-Funktion: analog zu 4.3
- [x] 4.5 `respondTraining` und `respondGame`: `catch`-Block ergänzen, der einen sichtbaren Fehlerstring pro Termin-Key setzt (neuer State `rsvpErrors: Record<string, string>`)
- [x] 4.6 Im Training-Card: wenn `isParent`, statt der eigenen RSVP-Buttons einen Abschnitt pro Kind aus `children_rsvp` rendern — jedes Kind mit eigenem Namen-Label, eigenem Reason-Input und eigenen drei RSVP-Buttons
- [x] 4.7 Im Game-Card: analog zu 4.6
- [x] 4.8 Fehlermeldung aus `rsvpErrors[key]` unterhalb des jeweiligen Termin-Cards anzeigen (Alert-Fehler-Klasse aus CLAUDE.md)

## 5. Frontend — TermineDetailPage: Kaderliste für alle

- [x] 5.1 `AttendanceItem`-Interface um `reason: string | null` erweitern
- [x] 5.2 `loadAttendances()` Aufruf in `useEffect` von `isTrainer && isTraining` auf `isTraining` ändern
- [x] 5.3 `tableRows`-Aufbau vereinheitlichen: beide Pfade (Trainer und Nicht-Trainer) nutzen `attendances` als Datenquelle; `reason` kommt direkt aus `a.reason` (bereits vom Backend gefiltert)
- [x] 5.4 `showAttendanceCol`-Bedingung bleibt unverändert (`isTrainer && isPast`) — Nicht-Trainer sehen keine Checkbox-Spalte
- [x] 5.5 Sicherstellen dass `session.responses` weiterhin für die `responseMap` geladen wird (Trainer brauchen es für die `present`-Zusammenführung nicht mehr, aber `load()` kann bestehen bleiben für andere Felder wie `my_rsvp`)
