# Proposal: Season-based Kader Management

## Problem

Currently, teams are managed globally and static. This creates friction when managing multiple seasons:
- Teams must be manually recreated for each season
- Members are assigned per season, but the team structure isn't season-aware
- Copying team structures between seasons requires manual work
- Member assignment logic (by age/gender) is manual

## Vision

Implement **season-based Kader (team) management** where:
- Each season has its own set of Kader (A-Jugend m/w, B-Jugend m/w, C-Jugend m/w, D-Jugend mixed)
- Kader can be copied from previous seasons with intelligent member suggestions
- Members are auto-assignable based on birth year + gender
- Age brackets automatically adjust year-over-year (2025/26 reference model)
- Kader are always editable (add/remove members at any time)

## Scope

### In Scope
1. **Data Model**: Teams → Kader (per season)
2. **Copy Workflow**: Clone team structures between seasons with member assignment options
3. **Member Assignment**: Three options per team:
   - Empty (structure only)
   - Copy from same/previous age class (with suggestions)
   - Auto-assign by birth year + gender
4. **Edit Interface**: Always-editable Kader with content-assist member search
5. **Age Bracket Calculation**: Automatic year-over-year progression from 2025/26 reference

### Out of Scope
- Trainer assignment (handled separately)
- Game/duty slot generation based on Kader
- Member availability tracking
- Kader-specific configuration (name, capacity, etc.)

## Key Requirements

### Functional

1. **Kader per Season**
   - Each season has independent Kader definitions
   - Kader name: `{age_class} {gender}` (e.g., "A-Jugend männlich")
   - Kader includes: member list, season reference, age bracket info

2. **Copy Workflow** (`/admin/kader` → "Aus vorheriger Saison kopieren")
   - **Step 1**: Select which teams to copy (checkboxes)
   - **Step 2**: For each team, choose member source:
     - `Nur Struktur` (empty)
     - `{same age class} {previous season}` (suggested)
     - `{age class before} {previous season}` (suggested, for natural progression)
     - `Auto-Assign nach Jahrgang+Geschlecht`
   - **Step 3**: Confirm + save

3. **Member Selection (Edit View)**
   - Text input with content-assist
   - Filter suggestions by birth year bracket (auto, ignorable)
   - Display: `{name} ({birth_year}/{gender})`
   - Selection adds member to list
   - Input remains (no auto-clear)
   - Members can be removed via checkbox + save

4. **Age Bracket Auto-Calculation**
   - Reference season: **2025/26**
     - A-Jugend: 2006-2007
     - B-Jugend: 2007-2008
     - C-Jugend: 2008-2009
     - D-Jugend: 2009-2010
   - For any season: `birth_years = reference_years + (season_offset)`
   - Example 2026/27 (offset +1):
     - A-Jugend: 2007-2008
     - B-Jugend: 2008-2009
     - C-Jugend: 2009-2010
     - D-Jugend: 2010-2011

### Non-Functional

- Kader should be editable at any time (during and after season)
- Content-assist should handle large member lists efficiently
- Gender-aware filtering (männlich, weiblich, mixed)
- Terminology: Replace "Teams" → "Kader" in `/admin/teams` → `/admin/kader`

## Success Criteria

- [ ] `/admin/kader` displays all Kader for selected season
- [ ] Copy workflow creates new Kader structures with member assignment options
- [ ] Auto-assign correctly matches members to age brackets by birth year + gender
- [ ] Content-assist filters members by age bracket (with override option)
- [ ] Members can be added/removed from any Kader at any time
- [ ] Age brackets automatically adjust for each season (based on 2025/26 reference)
- [ ] No breaking changes to existing data

## Risks & Open Questions

1. **Data Migration**: How to migrate existing global teams to season-based model?
2. **Jahrgänge Validation**: Should we validate that birth years match expected ranges for a Kader?
3. **Member Conflicts**: What if a member is assigned to multiple Kader in the same season?
4. **Trainer Assignment**: Should trainers be season-specific or inherited?

## Timeline & Effort

- Estimated effort: Medium (DB schema change, new UI flows, auto-assign logic)
- Blocking: Data model design & migration strategy
