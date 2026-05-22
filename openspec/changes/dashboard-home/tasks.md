# Tasks: Dashboard Home Page

## Backend

### T1: `/api/dashboard` Endpoint Structure
- [ ] Define response schema in types/dashboard.go
- [ ] Create handler in `internal/dashboard/handler.go`
- [ ] Register route in main.go (`GET /api/dashboard`)
- [ ] Add to router (authenticated middleware)

### T2: Action Calculation (Trainer)
- [ ] Query: Find duty slots (this week) where `team_id = user.team_id` and `slots_filled < slots_total`
- [ ] Format actions: "X Dienste diese Woche nicht besetzt"
- [ ] Link to: `/dienste` (DutySlotsPage)

### T3: Action Calculation (Elternteil)
- [ ] Query: Find duty slots (this week) where `user_id` can participate (via family_links or direct member)
- [ ] Filter: exclude already fulfilled/assigned to user
- [ ] Format: "Dienst '{duty_type}' {date} {time} — wir brauchen dich!"
- [ ] Link to: `/dienstboerse` (DutyBoardPage)

### T4: Vehicle Action (Both Roles)
- [ ] Find next away game (is_home = false) this week or next
- [ ] Query: Fahrtgemeinschaft status for that game
- [ ] Check: Does user have vehicle_info recorded?
- [ ] Format action accordingly
- [ ] Link to: `/profil` (ProfilePage)

### T5: NextGames Data
- [ ] Query: Games for user's team(s), next 2–3, ordered by date
- [ ] Include: date, opponent, isHome, slots_count, slots_filled
- [ ] Role-aware: Trainer sees all team games; Elternteil sees their child's team games

### T6: TeamStats (Trainer) / DutyAccount (Elternteil)
- [ ] Trainer: Query team members (season, team), count active/injured/paused
- [ ] Elternteil: Query duty_accounts for current season, get soll/ist/offen
- [ ] Include in response based on role

### T7: VehicleInfo
- [ ] Query vehicle_info for user
- [ ] Include in response (seats, notes, upToDate boolean)

### T8: Season Context
- [ ] Query current active season
- [ ] Include in response (for UI: "Saison 2025/26")
- [ ] If no active season: return empty actions + warning note

### T9: Tests (Unit)
- [ ] Test action calculation for Trainer (mocked DB)
- [ ] Test action calculation for Elternteil (mocked DB)
- [ ] Test edge cases: no active season, no upcoming games, vehicle missing

---

## Frontend

### F1: DashboardPage Component
- [ ] Create `web/src/pages/DashboardPage.tsx`
- [ ] Fetch `/api/dashboard` on mount
- [ ] Handle loading state
- [ ] Handle error state (graceful fallback)

### F2: Accordion Component
- [ ] Create `web/src/components/Accordion.tsx` (reusable)
- [ ] Props: title, icon, isOpen (default), children
- [ ] Toggle state on click
- [ ] Mobile: only one section open at a time (via state)
- [ ] Desktop: all can be open (or default to specific ones)

### F3: Actions Section
- [ ] Component: `ActionsList`
- [ ] Render: `dashboard.actions` array
- [ ] Each action: checkbox icon + text + link
- [ ] Always open by default (`⚡ DIESE WOCHE ▾`)

### F4: NextGames Section
- [ ] Component: `NextGamesList`
- [ ] Render: simple list of games with date, opponent, slot status
- [ ] Link to: `/spielplan/{gameId}`

### F5: Konto / TeamStats Section
- [ ] Component: renders based on role
- [ ] **Trainer:** Team stats (members active/injured, count)
- [ ] **Elternteil:** Duty account (Soll/Ist/Offen) + progress bar
- [ ] Link to detail pages

### F6: Team Section
- [ ] Component: basic team info + member counts
- [ ] Link to: `/mitglieder` (Trainer) or `/profil` (Elternteil)

### F7: Fahrtgemeinschaften Section
- [ ] Component: upcoming away games + vehicle status
- [ ] Show: "X Plätze gemeldet" or "Brauchen noch X Plätze"
- [ ] Link to: `/profil`

### F8: Mobile Styling
- [ ] Accordion: only `⚡ DIESE WOCHE` open on load (< 640px)
- [ ] Sections: full width, touch-friendly spacing
- [ ] Icons: clear, readable on small screen
- [ ] Test: Safari mobile

### F9: Styling (Desktop)
- [ ] All sections visible + scrollable
- [ ] Cards/sections with spacing + borders
- [ ] Color/typography: match brand (Hanken Grotesk, brand colors)
- [ ] Icons: Emoji or SVG (consistent with app)

---

## Integration

### I1: Update App.tsx
- [ ] Change `Route index` from `<Navigate to="/mitglieder" />` to `<DashboardPage />`
- [ ] Verify: no breaking changes to other routes

### I2: Logo Click (already works)
- [ ] Confirm: logo in AppShell already navigates home
- [ ] Test: click logo → dashboard

### I3: Update Nav (optional)
- [ ] Decide: add "Dashboard" link to AppShell nav?
- [ ] Or: only accessible via logo?

---

## Testing

### T1: Functional Testing (Manual)
- [ ] Login as Trainer → see trainer-specific actions
- [ ] Login as Elternteil → see elternteil-specific actions
- [ ] Click each action → correct detail page loads
- [ ] Mobile (Safari): accordion works, one section open

### T2: Edge Cases
- [ ] No active season → show empty state + message
- [ ] No upcoming games → section empty but visible
- [ ] No vehicle info → action prompts to add one
- [ ] No open duty slots → "All services covered" or empty

### T3: Performance
- [ ] Dashboard loads in < 1s (measure via DevTools)
- [ ] API response time reasonable (< 500ms with typical queries)

---

## Definition of Done

- [x] All tasks checked off
- [x] Tests pass (unit + manual)
- [x] Mobile-responsive (< 640px tested in Safari)
- [x] Code reviewed
- [x] No console errors/warnings
- [x] Deployed to staging
- [x] User testing (Trainer + Eltern feedback)
