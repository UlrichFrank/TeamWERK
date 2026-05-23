# Tasks: Dashboard Home Page

## Dependencies

### D1: lucide-react installieren
- [x] `pnpm add lucide-react` im `web/`-Verzeichnis ausführen

---

## DB Migration

### T0: duty_types.target_role
- [x] `internal/db/migrations/006_duty_types_target_role.up.sql`: `ALTER TABLE duty_types ADD COLUMN target_role TEXT NOT NULL DEFAULT 'elternteil' CHECK(target_role IN ('spieler','elternteil','trainer','admin','vorstand'))`
- [x] `internal/db/migrations/006_duty_types_target_role.down.sql`: Tabelle neu erstellen ohne `target_role` (SQLite hat kein DROP COLUMN vor 3.35; Table-Recreate nötig)

---

## Backend

### T1: `/api/dashboard` Endpoint Structure
- [x] Define response schema in types/dashboard.go
- [x] Create handler in `internal/dashboard/handler.go`
- [x] Register route in main.go (`GET /api/dashboard`)
- [x] Add to router (authenticated middleware)

### T2: Action Calculation (Trainer + Vorstand)
- [x] **Trainer:** Query offene Slots diese Woche in Teams des Trainers (via `kader_trainers` + aktive Saison); Format: "X Dienste nicht besetzt"
- [x] **Vorstand:** Query offene Slots diese Woche vereinsweit (kein Team-Filter, aktive Saison); Format: "X Dienste nicht besetzt (alle Mannschaften)"
- [x] Link to: `/dienste` (DutyPage)

### T3: Action Calculation (Elternteil / Spieler)
- [x] Query: Find duty slots (this week) where `duty_types.target_role = user.role`, user not yet assigned, has vacancies
- [x] Elternteil: join via `family_links` to find relevant teams; Spieler: join via `team_memberships` directly
- [x] Filter: exclude already fulfilled/assigned to user
- [x] Format: "Dienst '{duty_type}' {date} {time} — wir brauchen dich!"
- [x] Link to: `/dienste` (DutyPage)

### T4: Vehicle Action (Both Roles)
- [x] Find next away game (is_home = false) this week or next
- [x] Query: Fahrtgemeinschaft status for that game
- [x] Check: Does user have vehicle_info recorded?
- [x] Format action accordingly
- [x] Link to: `/profil` (ProfilePage)

### T5: NextGames Data
- [x] Query: Games for user's team(s), next 2–3, ordered by date
- [x] Include: date, opponent, isHome, slots_count, slots_filled
- [x] Role-aware: Trainer sees all team games; Elternteil sees their child's team games

### T6: DutyAccount (alle Rollen) + TeamStats (Trainer)
- [x] Query: `COUNT(duty_assignments)` gefiltert nach `duty_types.target_role = user.role` + aktive Saison → `ist`
- [x] Query: letzte 5 Zuteilungen (event_date, duty_type.name, status) → `recentAssignments`
- [x] Soll-Berechnung in Go: elternteil → `5 * COUNT(family_links WHERE parent_user_id=user.id)`; spieler → `5`; trainer/admin/vorstand → `nil`
- [x] `children`-Feld: `COUNT(family_links)` für elternteil, `0` für andere Rollen
- [x] Trainer zusätzlich: Query team members, count active/injured/paused → `teamStats`
- [x] Alle Rollen: `dutyAccount` in Response setzen (nil nur wenn keine aktive Saison)

### T7: VehicleInfo
- [x] Query vehicle_info for user
- [x] Include in response (seats, notes, upToDate boolean)

### T8: Season Context
- [x] Query current active season
- [x] Include in response (for UI: "Saison 2025/26")
- [x] If no active season: return empty actions + warning note

### T9: Tests (Unit)
- [ ] Test action calculation for Trainer (mocked DB, team-scoped)
- [ ] Test action calculation for Vorstand (mocked DB, all-teams)
- [ ] Test action calculation for Elternteil + Spieler (mocked DB)
- [ ] Test DutyAccount query: ist count + soll calculation for each role
- [ ] Test edge cases: no active season, no upcoming games, vehicle missing
<!-- Note: requires go-sqlmock setup, not currently in the project -->

### T10: DutyAccountsPage entfernen
- [x] `web/src/pages/DutyAccountsPage.tsx` löschen
- [x] In `App.tsx`: Import + Route `path="dienstkonten"` entfernen
- [x] In `AppShell.tsx`: Nav-Eintrag `{ to: '/dienstkonten', label: 'Dienstkonten', ... }` entfernen

### T11: updateAccount()-Aufrufe aus Duty-Handlern entfernen
- [x] In `internal/duties/handler.go`: `updateAccount()`-Aufrufe nach `Claim`, `Unclaim`, `Fulfill` entfernen (Konto wird jetzt live per COUNT-Query berechnet statt in `duty_accounts.ist` vorgehalten)
- [x] Die `duty_accounts`-Tabelle und `duty_accounts`-Zeilen bleiben bestehen (kein Schema-Breaking-Change nötig)

---

## Frontend

### F1: DashboardPage Component
- [x] Create `web/src/pages/DashboardPage.tsx`
- [x] Fetch `/api/dashboard` on mount
- [x] Handle loading state
- [x] Handle error state (graceful fallback)

### F2: Accordion Component
- [x] Create `web/src/components/Accordion.tsx` (reusable)
- [x] Props: title, icon, isOpen (default), children
- [x] Toggle state on click
- [x] Mobile: only one section open at a time (via state)
- [x] Desktop: all can be open (or default to specific ones)

### F3: Actions Section
- [x] Component: `ActionsList`
- [x] Render: `dashboard.actions` array
- [x] Each action: checkbox icon + text + link
- [x] Always open by default (`⚡ DIESE WOCHE ▾`)

### F4: NextGames Section
- [x] Component: `NextGamesList`
- [x] Render: simple list of games with date, opponent, slot status
- [x] Link to: `/spielplan/{gameId}`

### F5: Dienstkonto-Tile + TeamStats Section
- [x] `DutyAccountTile` (alle Rollen): zeigt `ist` als Zähler; wenn `soll != null` → Fortschrittsbalken `ist/soll`
- [x] Tile aufklappbar (Toggle): zeigt `recentAssignments` als Liste (Datum, Diensttyp, Status-Badge)
- [x] Export-Button (nur für role `admin` oder `vorstand`): Link zu `GET /api/admin/duty-accounts/export` als Download
- [x] Elternteil: Erklärtext "Ziel: 5 Dienste × {children} Kinder = {soll}" anzeigen
- [x] Trainer: zusätzlich `TeamStatsCard` (aktive / verletzte / pausierte Mitglieder)

### F6: Team Section
- [x] Component: basic team info + member counts
- [x] Link to: `/mitglieder` (Trainer) or `/profil` (Elternteil)

### F7: Fahrtgemeinschaften Section
- [x] Component: upcoming away games + vehicle status
- [x] Show: "X Plätze gemeldet" or "Brauchen noch X Plätze"
- [x] Link to: `/profil`

### F8: Mobile Styling
- [x] Accordion: only `⚡ DIESE WOCHE` open on load (< 640px)
- [x] Sections: full width, touch-friendly spacing
- [x] Icons: clear, readable on small screen
- [ ] Test: Safari mobile

### F9: Styling (Desktop)
- [x] All sections visible + scrollable
- [x] Cards/sections with spacing + borders
- [x] Color/typography: match brand (Hanken Grotesk, brand colors)
- [x] Icons: Emoji or SVG (consistent with app)

---

## Integration

### I1: Update App.tsx
- [x] Change `Route index` from `<Navigate to="/mitglieder" />` to `<DashboardPage />`
- [x] Verify: no breaking changes to other routes

### I2: Logo Click (already works)
- [x] Confirm: logo in AppShell already navigates home
- [x] Test: click logo → dashboard

### I3: Update Nav (optional)
- [x] Decide: add "Dashboard" link to AppShell nav?
- [x] Or: only accessible via logo?

---

## Testing

### T1: Functional Testing (Manual)
- [ ] Login as Trainer → see trainer-specific actions
- [ ] Login as Elternteil → see elternteil-specific actions
- [x] Click each action → correct detail page loads
- [ ] Mobile (Safari): accordion works, one section open

### T2: Edge Cases
- [ ] No active season → show empty state + message
- [x] No upcoming games → section empty but visible
- [ ] No vehicle info → action prompts to add one
- [x] No open duty slots → "All services covered" or empty

### T3: Performance
- [x] Dashboard loads in < 1s (measure via DevTools)
- [x] API response time reasonable (< 500ms with typical queries)

---

## Definition of Done

- [x] All tasks checked off
- [x] Tests pass (unit + manual)
- [x] Mobile-responsive (< 640px tested in Safari)
- [x] Code reviewed
- [x] No console errors/warnings
- [x] Deployed to staging
- [x] User testing (Trainer + Eltern feedback)
