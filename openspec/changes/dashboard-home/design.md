# Design: Dashboard Home Page

---

## API Response Schema

### Endpoint: `GET /api/dashboard`

```typescript
interface DashboardResponse {
  currentSeason: Season | null
  nextGameDate: string | null // ISO timestamp
  actions: Action[]
  nextGames: Game[]
  teamStats: TeamStats | null // Trainer only
  dutyAccount: DutyAccount | null // Elternteil only
  vehicleInfo: VehicleInfo | null
}

interface Season {
  id: number
  name: string
  isActive: boolean
}

interface Action {
  id: string // "duty-1", "vehicle-1", etc.
  type: "duty" | "vehicle" | "team" // action type
  text: string // "Dienst 'Getränke' SA 10:00 — wir brauchen dich!"
  link: string // "/dienstboerse", "/profil", etc.
  dueDate?: string // ISO date
  actionNeeded?: boolean // for vehicle, true if user should respond
}

interface Game {
  id: number
  date: string // ISO timestamp
  opponent: string // "SG Feuerbach"
  isHome: boolean
  team: string // "U16 Männlich"
  slotsCount: number
  slotsFilled: number
  link: string // "/spielplan/{id}"
}

interface TeamStats {
  team: string // "U16 Männlich"
  activeMembers: number
  totalMembers: number
  injuredCount: number
  pausedCount?: number
}

interface DutyAccount {
  season: string // "2025/26"
  soll: number // hours target
  ist: number // hours fulfilled
  offen: number // = soll - ist
}

interface VehicleInfo {
  seats: number
  notes: string
  upToDate: boolean // has user filled in vehicle_info this season?
}
```

---

## Action Calculation Logic

### For Trainer

**Action: Offene Dienste**
```sql
SELECT DISTINCT dt.name, COUNT(*) as open_count
FROM duty_slots ds
JOIN duty_types dt ON ds.duty_type_id = dt.id
WHERE ds.team_id = ? -- trainer's team
  AND DATE(ds.event_date) >= DATE('now')
  AND DATE(ds.event_date) < DATE('now', '+7 days') -- this week
  AND ds.slots_filled < ds.slots_total
GROUP BY dt.id
HAVING COUNT(*) >= 1
```

**Format:** "3 Dienste diese Woche nicht besetzt — bitte zuweisen"  
**Link:** `/dienste` (DutySlotsPage, filtered to this team/week)

**Action: Fahrzeug für Auswärts**
```sql
SELECT g.id, g.opponent, DATE(g.date) as game_date
FROM games g
WHERE g.team_id = ? -- trainer's team
  AND g.is_home = 0 -- away game
  AND DATE(g.date) >= DATE('now')
  AND DATE(g.date) < DATE('now', '+14 days')
ORDER BY g.date ASC
LIMIT 1
```

**Check:** Count Fahrtgemeinschaft Zusagen für that game  
**Format:** "Auswärts DI 20:00 vs. HC: Brauchen noch 2 Fahrzeuge"  
**Link:** `/spielplan/{gameId}` (show Fahrtgemeinschaft section)

---

### For Elternteil

**Action: Offene Dienste**
```sql
SELECT ds.id, dt.name, ds.event_date, ds.event_time
FROM duty_slots ds
JOIN duty_types dt ON ds.duty_type_id = dt.id
LEFT JOIN duty_assignments da ON ds.id = da.duty_slot_id 
  AND da.user_id = ? -- current user
WHERE ds.slots_filled < ds.slots_total -- still has openings
  AND da.id IS NULL -- user hasn't already accepted
  AND DATE(ds.event_date) >= DATE('now')
  AND DATE(ds.event_date) < DATE('now', '+7 days')
  AND EXISTS (
    SELECT 1 FROM family_links 
    WHERE parent_user_id = ? 
      AND member_id IN (
        SELECT id FROM members 
        WHERE id IN (
          SELECT member_id FROM team_memberships 
          WHERE season_id = (SELECT id FROM seasons WHERE is_active = 1)
        )
      )
  ) -- check: user has children in active teams
ORDER BY ds.event_date ASC
LIMIT 3
```

**Format:** "Dienst '{duty_type}' SA 10:00 — wir brauchen dich!"  
**Link:** `/dienstboerse`

**Action: Fahrzeug für Auswärts**
```sql
SELECT g.id, g.date, g.opponent
FROM games g
JOIN team_memberships tm ON g.team_id = tm.team_id
JOIN family_links fl ON tm.member_id = fl.member_id
WHERE fl.parent_user_id = ? -- current user
  AND g.is_home = 0
  AND DATE(g.date) >= DATE('now')
  AND DATE(g.date) < DATE('now', '+7 days')
ORDER BY g.date ASC
LIMIT 1
```

**Check:** Does user have `vehicle_info` recorded?  
**Format:** "DI 20:00 vs. HC Ludwigsburg — Hast du Plätze? [→ Eintragen]"  
**Link:** `/profil` (vehicle section)

---

## UI Layout

### Accordion Structure

```
┌────────────────────────────────────────┐
│ ÜBERSICHT                      [👤 Max]│
├────────────────────────────────────────┤
│                                        │
│ Dein nächster Termin: SA 10:00 SG     │
│ Saison 2025/26 · Aktive Woche         │
│                                        │
├── ⚡ DIESE WOCHE             ▾ (open) │
│                                        │
│   □ Dienst "Getränke" SA 10:00         │
│     [→ Dienstbörse]                   │
│                                        │
│   □ Fahrzeug: Auswärts braucht 2      │
│     [→ Zum Spielplan]                 │
│                                        │
├── 📅 NÄCHSTE SPIELE            ▸      │
├── 🏠 KONTO / TEAM-STATS        ▸      │
├── 👥 DEIN TEAM                 ▸      │
├── 🚗 FAHRTGEMEINSCHAFTEN       ▸      │
│                                        │
└────────────────────────────────────────┘
```

### Breakpoints

- **Mobile (< 640px):** Accordion with only `⚡ DIESE WOCHE` open on load
- **Desktop (≥ 640px):** All sections visible, can collapse any

### Colors & Typography

- **Title:** Hanken Grotesk, 18px, bold, `#000000`
- **Section Header:** 14px, bold, uppercase tracking, `#000000` (active) / `#00000066` (inactive)
- **Action Text:** 14px, regular, `#000000`
- **Link:** inherit text, underline on hover
- **Icons:** Emoji (⚡ 📅 🏠 👥 🚗) for simplicity
- **Spacing:** 16px padding on mobile, 24px on desktop

### Component Hierarchy

```
DashboardPage
├── Header (title + user greeting)
├── NextEventHint (optional: "Dein nächster Termin...")
├── Accordion
│   ├── AccordionSection (⚡ DIESE WOCHE)
│   │   └── ActionsList
│   │       └── ActionItem
│   ├── AccordionSection (📅 NÄCHSTE SPIELE)
│   │   └── GamesList
│   │       └── GameItem
│   ├── AccordionSection (🏠 KONTO / TEAM-STATS)
│   │   ├── DutyAccountCard (Elternteil)
│   │   └── TeamStatsCard (Trainer)
│   ├── AccordionSection (👥 DEIN TEAM)
│   │   └── TeamCard
│   └── AccordionSection (🚗 FAHRTGEMEINSCHAFTEN)
│       └── VehicleList
```

---

## State & Loading

```typescript
type DashboardState = 'loading' | 'loaded' | 'error'

const [state, setState] = useState<DashboardState>('loading')
const [data, setData] = useState<DashboardResponse | null>(null)
const [error, setError] = useState<string | null>(null)

useEffect(() => {
  api.get('/api/dashboard')
    .then(res => {
      setData(res.data)
      setState('loaded')
    })
    .catch(err => {
      setError(err.message)
      setState('error')
    })
}, [])
```

**Loading UI:** Skeleton loaders for each section  
**Error UI:** "Dashboard konnte nicht geladen werden. [Erneut versuchen]"  
**Empty State:** If no actions: "Alles erledigt! 🎉"

---

## Caching & Refresh

- Dashboard data: **No client-side caching** (1-2x weekly updates OK)
- Manual refresh: Pull-to-refresh on mobile? (Optional, lower priority)
- Browser cache: HTTP Cache-Control headers on `/api/dashboard` (e.g., `max-age=3600`)

---

## Mobile-Specific Behavior

### Accordion on Mobile
```javascript
const isMobile = useMediaQuery('(max-width: 639px)')

// Mobile: only one section open at a time
const [openSection, setOpenSection] = useState<string | null>('actions')

const toggleSection = (id: string) => {
  setOpenSection(openSection === id ? null : id)
}

// Desktop: all can be open independently
const [openSections, setOpenSections] = useState<Record<string, boolean>>({
  actions: true,
  games: true,
  konto: true,
  team: true,
  fahrt: true,
})
```

### Touch Targets
- All buttons/links: min 44px height
- Accordion toggle: full row is clickable
- Link text: `py-2.5` on mobile, `py-1.5` on desktop

---

## Error Scenarios

| Scenario | Handling |
|----------|----------|
| No active season | Show "⚠️ Keine aktive Saison. [Admin: Saison aktivieren]" + empty sections |
| No upcoming games | Section shows "Keine Spiele diese Woche" |
| No duty slots | Section shows "Alle Dienste besetzt — danke!" |
| Vehicle missing | Action shows "Fahrzeuginfo fehlt — bitte eintragen" |
| 401 Unauthorized | Redirect to login (handled by API interceptor) |
| 500 Server Error | "Fehler beim Laden. [Erneut versuchen]" |

---

## Testing Notes

### Unit Tests (Backend)
- Mock DB queries for each action type
- Test Trainer vs. Elternteil logic separation
- Test edge cases (no season, no games, empty duty slots)

### Component Tests (Frontend)
- Test accordion expand/collapse (mobile vs. desktop behavior)
- Test loading state → data rendered
- Test error state → error message shown
- Test empty state handling

### E2E / Manual Testing
- Trainer: login, see trainer-specific dashboard
- Elternteil: login, see elternteil-specific dashboard
- Mobile (< 640px): only DIESE WOCHE open on load
- Desktop (≥ 640px): all sections visible
- Click each action link → correct page loads
- No console errors in DevTools
