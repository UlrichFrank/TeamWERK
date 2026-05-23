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
  teamStats: TeamStats | null // Trainer + Vorstand (Vorstand: aggregate across all teams)
  dutyAccount: DutyAccount | null // all authenticated roles
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
  link: string // "/dienste", "/profil", etc.
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

// Migration required: ALTER TABLE duty_types ADD COLUMN target_role TEXT NOT NULL DEFAULT 'elternteil'
//   CHECK (target_role IN ('spieler','elternteil','trainer','admin','vorstand'))
// duty_types.target_role bestimmt, welche Nutzer-Rolle den Dienst erbringt.
// Dienste werden in duty_assignments gezählt (kein Pre-Compute mehr in duty_accounts.ist).

interface DutyAccount {
  season: string       // "2025/26"
  ist: number          // COUNT duty_assignments this season where duty_type.target_role = user.role
  soll: number | null  // null for trainer/admin/vorstand; 5*children for elternteil; 5 for spieler
  children: number     // for elternteil: number of family_links (explains soll); 0 for other roles
  recentAssignments: { date: string; dutyType: string; status: string }[]  // last 5, for expandable tile
}

interface VehicleInfo {
  seats: number
  notes: string
  upToDate: boolean // has user filled in vehicle_info this season?
}
```

---

## Action Calculation Logic

### DutyAccount Query (all roles)

```sql
-- ist: count of assignments for this user where duty type targets their role
SELECT COUNT(*) as ist
FROM duty_assignments da
JOIN duty_slots ds ON da.duty_slot_id = ds.id
JOIN duty_types dt ON ds.duty_type_id = dt.id
WHERE da.user_id = ?
  AND ds.season_id = (SELECT id FROM seasons WHERE is_active = 1)
  AND dt.target_role = ?  -- user's role
  AND da.status IN ('assigned', 'fulfilled', 'cash_substitute')

-- recentAssignments: last 5
SELECT ds.event_date, dt.name, da.status
FROM duty_assignments da
JOIN duty_slots ds ON da.duty_slot_id = ds.id
JOIN duty_types dt ON ds.duty_type_id = dt.id
WHERE da.user_id = ?
  AND ds.season_id = (SELECT id FROM seasons WHERE is_active = 1)
  AND dt.target_role = ?
ORDER BY ds.event_date DESC
LIMIT 5

-- soll calculation (in Go):
-- if role == "elternteil": SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? → 5 * count
-- if role == "spieler": 5
-- else: nil (no target)
```

---

### For Trainer

**Action: Offene Dienste**
```sql
SELECT DISTINCT dt.name, COUNT(*) as open_count
FROM duty_slots ds
JOIN duty_types dt ON ds.duty_type_id = dt.id
WHERE ds.team_id IN (
    SELECT team_id FROM kader_trainers kt
    JOIN kader k ON k.id = kt.kader_id
    WHERE kt.member_id IN (SELECT id FROM members WHERE user_id = ?)
      AND k.season_id = (SELECT id FROM seasons WHERE is_active = 1)
  )
  AND DATE(ds.event_date) >= DATE('now')
  AND DATE(ds.event_date) < DATE('now', '+7 days') -- this week
  AND ds.slots_filled < ds.slots_total
GROUP BY dt.id
HAVING COUNT(*) >= 1
```

**Format:** "3 Dienste diese Woche nicht besetzt — bitte zuweisen"  
**Link:** `/dienste` (DutyPage)

---

### For Vorstand

**Action: Offene Dienste (alle Teams)**
```sql
SELECT COUNT(*) as open_count
FROM duty_slots ds
WHERE DATE(ds.event_date) >= DATE('now')
  AND DATE(ds.event_date) < DATE('now', '+7 days')
  AND ds.slots_filled < ds.slots_total
  AND ds.season_id = (SELECT id FROM seasons WHERE is_active = 1)
```

Vorstand hat keine Team-Zuordnung — sieht offene Slots vereinsweit.

**Format:** "X Dienste diese Woche nicht besetzt (alle Mannschaften)"  
**Link:** `/dienste`

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
  AND dt.target_role = 'elternteil'  -- only duties for this role
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
**Link:** `/dienste`

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
- **Icons:** Lucide React (`lucide-react` package, neue Dependency)
- **Spacing:** 16px padding on mobile, 24px on desktop

### Icon Mapping (Lucide React)

| Bereich | Lucide-Icon | Import |
|---------|-------------|--------|
| Diese Woche | `Zap` | `import { Zap } from 'lucide-react'` |
| Nächste Spiele | `Calendar` | `import { Calendar } from 'lucide-react'` |
| Konto / Team-Stats | `BarChart2` | `import { BarChart2 } from 'lucide-react'` |
| Dein Team | `Users` | `import { Users } from 'lucide-react'` |
| Fahrtgemeinschaften | `Car` | `import { Car } from 'lucide-react'` |
| Accordion Toggle offen | `ChevronDown` | `import { ChevronDown } from 'lucide-react'` |
| Accordion Toggle zu | `ChevronRight` | `import { ChevronRight } from 'lucide-react'` |
| Action-Item | `CircleDot` | `import { CircleDot } from 'lucide-react'` |
| Link-Pfeil | `ArrowRight` | `import { ArrowRight } from 'lucide-react'` |
| Export | `Download` | `import { Download } from 'lucide-react'` |
| Nutzer (Header) | `User` | `import { User } from 'lucide-react'` |

Icon-Größe: `size={16}` inline mit Text, `size={18}` als Section-Header-Icon. Farbe erbt vom Parent (`currentColor`).

### Component Hierarchy

```
DashboardPage
├── Header (title + user greeting)
├── NextEventHint (optional: "Dein nächster Termin...")
├── Accordion
│   ├── AccordionSection (icon=<Zap> · DIESE WOCHE)
│   │   └── ActionsList
│   │       └── ActionItem (icon=<CircleDot> + text + <ArrowRight>)
│   ├── AccordionSection (icon=<Calendar> · NÄCHSTE SPIELE)
│   │   └── GamesList
│   │       └── GameItem
│   ├── AccordionSection (icon=<BarChart2> · KONTO / TEAM-STATS)
│   │   ├── DutyAccountTile (alle Rollen) ← aufklappbar mit <ChevronDown>/<ChevronRight>
│   │   │   ├── DutyAccountSummary (ist / soll-Fortschrittsbalken, oder nur Zähler wenn soll=null)
│   │   │   ├── [Aufgeklappt] RecentAssignmentsList (letzte 5: Datum, Diensttyp, Status-Badge)
│   │   │   └── [Admin/Vorstand] ExportButton (icon=<Download>) → GET /api/admin/duty-accounts/export
│   │   └── TeamStatsCard (Trainer + Vorstand)
│   ├── AccordionSection (icon=<Users> · DEIN TEAM)
│   │   └── TeamCard
│   └── AccordionSection (icon=<Car> · FAHRTGEMEINSCHAFTEN)
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
