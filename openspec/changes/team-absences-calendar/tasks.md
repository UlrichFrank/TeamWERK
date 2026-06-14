## 1. Bugfix — Member-Struct und getMember()

- [x] 1.1 `Member`-Struct in `internal/members/handler.go` um `AbsencesPublic int` (oder `bool`) mit JSON-Tag `absences_public` erweitern
- [x] 1.2 `getMember()` SELECT um `absences_public` erweitern und im Scan auslesen
- [x] 1.3 Prüfen: `GET /api/profile/me` → `own_member.absences_public` gibt den gespeicherten Wert zurück; Toggle in `ProfileMiscTab` zeigt korrekt an

## 2. Backend — `GET /api/absences/calendar` erweitern

- [x] 2.1 `absence`-Struct in `internal/absences/handler.go` um `IsOwn bool` mit JSON-Tag `is_own` erweitern
- [x] 2.2 In `Calendar()` bestehende Abwesenheiten (eigene + Kinder) als `is_own = true` markieren
- [x] 2.3 Query-Parameter `show_team` und `team_id` auslesen
- [x] 2.4 Rollen-Check: `show_team=true` nur für `admin`, `trainer`, `sportvorstand`-Funktion, `vorstand`-Funktion auswerten
- [x] 2.5 Bei berechtigter Rolle + `show_team=true`: zweite Abfrage für Team-Abwesenheiten (`absences_public = 1`, aktive Saison, `user_accessible_teams`, optional `team_id`-Filter) — `is_own = false`
- [x] 2.6 Beide Ergebnismengen zusammenführen und als JSON zurückgeben (Duplikate vermeiden: eigene Abwesenheit bleibt `is_own = true`)
- [x] 2.7 Manuell testen: Trainer mit `show_team=true` sieht Team-Abwesenheiten; Spieler sieht sie nicht; `team_id`-Filter schränkt korrekt ein

## 3. Frontend — Kalender-Toggle

- [x] 3.1 State `showTeamAbsences` anlegen, initialisiert aus `sessionStorage.getItem('kalender_show_team_absences') === 'true'` (Default `false`)
- [x] 3.2 Bei Toggle-Änderung: `sessionStorage.setItem('kalender_show_team_absences', ...)` und Abwesenheiten neu laden
- [x] 3.3 Toggle-Button „Mannschaftsabwesenheiten" nur rendern wenn Rolle berechtigt (`admin`, `trainer`, `sportvorstand`, `vorstand`)
- [x] 3.4 `loadAbsences()` anpassen: bei `showTeamAbsences === true` → `?show_team=true` anhängen; bei aktivem `filterTeamId` → zusätzlich `&team_id={filterTeamId}` anhängen
- [x] 3.5 `filterTeamId`-Änderung löst `loadAbsences()` neu aus (prüfen, ob bestehende `useEffect`-Dependencies das abdecken)

## 4. Frontend — Darstellung Team-Abwesenheiten

- [x] 4.1 In `absencesForDay()` / Kalender-Render: Farbklassen nach `absence.is_own` aufteilen (`brand-yellow` vs. `brand-blue`)
- [x] 4.2 Tooltip (`title`-Attribut) für alle Abwesenheiten: `${absence.member_name}: ${type} ${start}–${end}`
- [x] 4.3 Click auf Abwesenheit öffnet vorhandenes InfoModal (`setInfoItem({ type: 'absence', absence })`) — prüfen ob dies für `is_own = false` bereits funktioniert
- [x] 4.4 `Absence`-Interface in `KalenderPage.tsx` um `is_own: boolean` erweitern

## 5. Test & Verifikation

- [x] 5.1 Profil-Toggle: Aktivieren → Reload → Toggle zeigt aktiv
- [x] 5.2 Kalender: Trainer aktiviert Toggle → Team-Abwesenheiten erscheinen blau
- [x] 5.3 Team-Filter + Toggle: nur Abwesenheiten des gewählten Teams sichtbar
- [x] 5.4 Spieler-Account: kein Toggle sichtbar, keine fremden Abwesenheiten
- [x] 5.5 Tooltip zeigt Name + Typ; Click öffnet Detail
