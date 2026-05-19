# Design: Season-based Kader Management

## Data Model

### New/Modified Tables

#### `kader` (new)
Season-specific team structures. Replaces global teams for per-season management.

```sql
CREATE TABLE kader (
  id INTEGER PRIMARY KEY,
  season_id INTEGER NOT NULL,
  age_class TEXT NOT NULL,  -- A-Jugend, B-Jugend, C-Jugend, D-Jugend
  gender TEXT NOT NULL,     -- m, f, mixed
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (season_id) REFERENCES seasons(id),
  UNIQUE(season_id, age_class, gender)
);
```

#### `kader_members` (new)
Maps members to Kader for a specific season.

```sql
CREATE TABLE kader_members (
  id INTEGER PRIMARY KEY,
  kader_id INTEGER NOT NULL,
  member_id INTEGER NOT NULL,
  added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (kader_id) REFERENCES kader(id),
  FOREIGN KEY (member_id) REFERENCES members(id),
  UNIQUE(kader_id, member_id)
);
```

#### `teams` (existing, deprecated)
Keep for backwards compatibility. New code uses `kader` and `kader_members`.

### Age Bracket Reference

Reference model for 2025/26 season stored as constants in code/config:

```go
type AgeBracketRef struct {
  AgeClass string    // A-Jugend, B-Jugend, etc.
  StartYear int      // Birth year (inclusive)
  EndYear int        // Birth year (inclusive)
}

// Reference: 2025/26 season (start_date = 2025-07-01)
var AGE_BRACKET_REFERENCE_2025_26 = []AgeBracketRef{
  {AgeClass: "A-Jugend", StartYear: 2006, EndYear: 2007},
  {AgeClass: "B-Jugend", StartYear: 2007, EndYear: 2008},
  {AgeClass: "C-Jugend", StartYear: 2008, EndYear: 2009},
  {AgeClass: "D-Jugend", StartYear: 2009, EndYear: 2010},
}
```

**Calculation Logic:**

For any season `S` with `start_date` year `Y`:
1. Reference offset = `Y - 2025` (e.g., 2026/27 → offset +1)
2. For each age bracket: `birth_years = reference_years + offset`

Example (2026/27, offset +1):
- A-Jugend: 2007–2008
- B-Jugend: 2008–2009
- C-Jugend: 2009–2010
- D-Jugend: 2010–2011

## API Endpoints

### Kader Management

#### `GET /api/admin/kader`
List all Kader for the active season.

**Response:**
```json
{
  "season_id": 1,
  "kader": [
    {
      "id": 10,
      "age_class": "A-Jugend",
      "gender": "m",
      "members": [
        {"id": 1, "name": "Max Müller", "birth_year": 2007, "gender": "m"},
        {"id": 2, "name": "Anna Schmidt", "birth_year": 2007, "gender": "f"}
      ],
      "member_count": 2
    }
  ]
}
```

#### `POST /api/admin/kader`
Create Kader structure(s) for a season.

**Request:**
```json
{
  "season_id": 2,
  "kader_specs": [
    {
      "age_class": "A-Jugend",
      "gender": "m",
      "members": []  // or copy from previous season
    }
  ]
}
```

#### `POST /api/admin/kader/copy-from-season`
Copy Kader structures from previous season with member assignment options.

**Request:**
```json
{
  "from_season_id": 1,
  "to_season_id": 2,
  "assignments": [
    {
      "age_class": "A-Jugend",
      "gender": "m",
      "member_source": "empty"  // or "same-age-previous", "age-before-previous", "auto-assign"
    }
  ]
}
```

**Response:** Returns created Kader with assigned members.

#### `GET /api/admin/kader/{id}`
Get Kader details with members.

#### `PUT /api/admin/kader/{id}`
Update Kader (member add/remove).

**Request:**
```json
{
  "members_add": [5, 6],
  "members_remove": [3]
}
```

#### `GET /api/admin/kader/{id}/member-suggestions`
Content-assist: suggest members for Kader based on age bracket.

**Query params:**
- `search`: partial name match (optional)
- `filter_age_bracket`: boolean (default true) — filter by computed age bracket for this Kader

**Response:**
```json
{
  "suggestions": [
    {
      "id": 7,
      "name": "Jan Hoffmann",
      "birth_year": 2007,
      "gender": "m",
      "reason": "Matches age bracket 2007–2008",
      "already_in_kader": false
    }
  ]
}
```

### Season Targets (existing endpoint, adapted)

#### `PUT /api/admin/seasons/{id}/duty-targets`
No changes, but note: duty targets are per season, Kader are also per season.

## Copy Workflow — UI Flow

### Step 1: Season Selection
- Show "Aus Saison XXXX/YY kopieren" button
- Dialog: source season selector (dropdown of previous seasons)

### Step 2: Team/Kader Selection
- List all Kader from source season as checkboxes
- Default: all checked
- User can uncheck to skip any age class/gender combo

### Step 3: Member Assignment (per selected Kader)
- For each selected Kader, show radio options:
  1. **Nur Struktur** (empty)
  2. **{same age class} {source season}** (suggested, pre-selected)
  3. **{age class before} {source season}** (suggested if exists)
  4. **Auto-Assign nach Jahrgang+Geschlecht** (auto-fill from members born in bracket)

- Display: Source season and count: `"B-Jugend männlich aus 2024/25 (8 Mitglieder)"`

### Step 4: Confirm & Save
- Summary: "Werden {N} Kader mit {M} Mitgliedern angelegt"
- Save button triggers `POST /api/admin/kader/copy-from-season`

## Edit Interface

### Location
`/admin/kader` — lists all Kader for active season

### Layout
```
┌─────────────────────────────────────────────────────┐
│ Kader — 2025/26                                     │
├─────────────────────────────────────────────────────┤
│ [+ Aus vorheriger Saison kopieren]                  │
├─────────────────────────────────────────────────────┤
│
│ ┌─ A-Jugend männlich (12 Mitglieder) ────────────┐
│ │ [+] Mitglied hinzufügen [Textfeld mit Assist]  │
│ │                                                 │
│ │ ☐ Max Müller (2007/m)                    [×]   │
│ │ ☐ Jan Hoffmann (2007/m)                  [×]   │
│ │ ...                                             │
│ │ [Speichern]                                    │
│ └─────────────────────────────────────────────────┘
│
│ ┌─ A-Jugend weiblich (8 Mitglieder) ─────────────┐
│ │ [+] Mitglied hinzufügen [Textfeld mit Assist]  │
│ │                                                 │
│ │ ☐ Anna Schmidt (2007/f)                  [×]   │
│ │ ...                                             │
│ │ [Speichern]                                    │
│ └─────────────────────────────────────────────────┘
```

### Member Addition
- Text input: `"Name (z.B. Max Müller)"`
- As user types, content-assist shows matching members from `member_suggestions` endpoint
- Display format: `{name} ({birth_year}/{gender})`
- Filter by computed age bracket (can be toggled)
- Selection adds member to list (input doesn't auto-clear)
- Member can be removed with `[×]` button

### Save Behavior
- Only save when [Speichern] clicked
- Batch update via `PUT /api/admin/kader/{id}` with members_add/members_remove
- On success: reload and show success toast

## Terminology Change

Replace "Teams" with "Kader" in:
- Page title: `/admin/teams` → `/admin/kader`
- Navigation label: "Teams" → "Kader"
- UI headings: "Teams" → "Kader"
- Section headers: "Neues Team" → "Neuer Kader"
- Dialog/button labels: "Team anlegen" → "Kader anlegen"

Existing `teams` table and routes remain for backwards compatibility.

## Age Bracket Calculation — Reference Implementation

```go
func ComputeAgeBracketForSeason(season *Season) map[string][2]int {
  // season.StartDate = "2025-07-01" for 2025/26
  startYear := parseYearFromDate(season.StartDate) // 2025
  offset := startYear - 2025
  
  result := make(map[string][2]int)
  for _, ref := range AGE_BRACKET_REFERENCE_2025_26 {
    result[ref.AgeClass] = [2]int{
      ref.StartYear + offset,
      ref.EndYear + offset,
    }
  }
  return result
}

func SuggestMembersForKader(kader *Kader, season *Season, searchTerm string) []*Member {
  brackets := ComputeAgeBracketForSeason(season)
  ageRange := brackets[kader.AgeClass]
  
  var members []*Member
  // Query: members where:
  // - gender matches kader.gender (or kader is "mixed")
  // - birth_year in [ageRange[0], ageRange[1]]
  // - name LIKE searchTerm (if provided)
  // - not already in kader
  
  return members
}

func AutoAssignMembersForKader(kader *Kader, season *Season) error {
  suggestions := SuggestMembersForKader(kader, season, "")
  for _, m := range suggestions {
    // INSERT into kader_members(kader_id, member_id)
  }
  return nil
}
```

## No Breaking Changes

- Existing `/api/admin/teams` endpoint continues to work
- `teams` table unchanged; `team_memberships` unchanged
- New code uses `kader` and `kader_members`
- Migration strategy: dual-write period if needed; eventually deprecate global teams

## Open Questions for Implementation

1. **Trainer Assignment**: Should trainers be assigned per Kader (like members) or remain team-wide?
2. **Duty Slot Generation**: Should duty slots reference Kader or Teams? (Assumed: Kader once teams are deprecated)
3. **Member Conflicts**: Prevent member from being in multiple Kader of same season?
4. **Inactive Seasons**: Can Kader be created for inactive seasons?
