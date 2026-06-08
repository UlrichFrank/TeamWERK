# Tasks: Season-based Kader Management

## Phase 1: Database & Core Logic

### Task 1.1: Create Database Migration
- [ ] Create `internal/db/migrations/006_add_kader_tables.up.sql`
  - Create `kader` table with (id, season_id, age_class, gender, created_at, updated_at)
  - Create `kader_members` table with (id, kader_id, member_id, added_at)
  - Add UNIQUE constraint on (season_id, age_class, gender) for kader
  - Add UNIQUE constraint on (kader_id, member_id) for kader_members
  - Add indexes on foreign keys
- [ ] Create corresponding `.down.sql` file
- [ ] Test migration locally: `make migrate-up` and `make migrate-down`

### Task 1.2: Implement Kader Handler (Backend)
- [ ] Create `internal/kader/handler.go` with `Handler` struct
- [ ] Implement `GetKaderForSeason(w, r)` — GET /api/admin/kader
- [ ] Implement `GetKaderDetail(w, r)` — GET /api/admin/kader/{id}
- [ ] Implement `UpdateKaderMembers(w, r)` — PUT /api/admin/kader/{id}
- [ ] Add kader routes to chi router in `cmd/teamwerk/main.go`
- [ ] Require admin role for all endpoints

### Task 1.3: Age Bracket Calculation
- [ ] Create `internal/kader/age_brackets.go`
- [ ] Define `AGE_BRACKET_REFERENCE_2025_26` constant
- [ ] Implement `ComputeAgeBracketForSeason(season *Season) map[string][2]int`
- [ ] Implement `MemberBirthYearInBracket(birthYear int, ageClass string, season *Season) bool`
- [ ] Write unit tests for age bracket calculation across multiple seasons

### Task 1.4: Member Suggestion Logic
- [ ] Implement `SuggestMembersForKader(db *sql.DB, kader *Kader, season *Season, searchTerm string, filterByAgeBracket bool) []*Member`
  - Query members filtered by:
    - Gender match (or kader gender is "mixed")
    - Birth year within age bracket (if filterByAgeBracket=true)
    - Name LIKE searchTerm
    - Not already in kader
  - Return with "reason" field explaining why suggestion matches
- [ ] Create `internal/kader/suggestions.go` with suggestion logic

### Task 1.5: Copy Workflow — Backend
- [ ] Implement `CopyKaderFromSeason(db *sql.DB, fromSeasonID, toSeasonID int, assignments []CopyAssignment) error`
  - For each assignment in request:
    - Create kader row in target season
    - Based on member_source:
      - `"empty"`: skip member assignment
      - `"same-age-previous"`: copy members from source Kader with same age_class/gender
      - `"age-before-previous"`: suggest and copy from "age class before" (A→B→C→D cycle)
      - `"auto-assign"`: call `SuggestMembersForKader` and auto-insert all
    - Insert rows into `kader_members`
- [ ] Create `internal/kader/copy.go` with copy logic
- [ ] Implement `POST /api/admin/kader/copy-from-season` endpoint
- [ ] Add validation: season exists and is active

### Task 1.6: Content-Assist Endpoint
- [ ] Implement `GET /api/admin/kader/{id}/member-suggestions` endpoint
  - Query params: `search`, `filter_age_bracket` (boolean, default true)
  - Return suggestions from `SuggestMembersForKader`
  - Include "already_in_kader" flag for each suggestion
  - Limit to 20 results

## Phase 2: Frontend — UI Components

### Task 2.1: AdminKaderPage Component
- [ ] Create `web/src/pages/AdminKaderPage.tsx`
- [ ] Show current season in header
- [ ] List all Kader (8 sections max: A-D × m/f + D × mixed)
- [ ] For each Kader:
  - Display age_class, gender, member count
  - "Aus vorheriger Saison kopieren" button (modal trigger)
  - "Mitglied hinzufügen" section with content-assist input
  - List members with remove button [×]
  - Save button
- [ ] Use `api.get('/admin/kader')` to load on mount
- [ ] Load active season from context or endpoint

### Task 2.2: Copy Kader Modal (Multi-Step)
- [ ] Create `web/src/components/CopyKaderModal.tsx`
- [ ] **Step 1 — Source Season Selection**
  - Dropdown of previous seasons (fetch from `/api/admin/seasons`)
  - Next button
- [ ] **Step 2 — Kader Selection**
  - List checkboxes of all Kader from source season
  - Each checkbox shows: `{age_class} {gender} (N members)`
  - All checked by default
  - Next/Back buttons
- [ ] **Step 3 — Member Assignment (per selected Kader)**
  - For each selected Kader, show radio options:
    1. `Nur Struktur` (empty)
    2. `{same age class} {source season} (N members)` (pre-selected)
    3. `{age class before} {source season} (M members)` (if exists)
    4. `Auto-Assign nach Jahrgang+Geschlecht`
  - Back/Confirm buttons
- [ ] **Step 4 — Confirmation**
  - Summary: "Werden {N} Kader mit {M} gesamt Mitgliedern angelegt"
  - Create button triggers `POST /api/admin/kader/copy-from-season`
  - On success: close modal and reload AdminKaderPage
  - Show loading state during request
  - Show error toast on failure

### Task 2.3: Member Content-Assist Component
- [ ] Create `web/src/components/KaderMemberSearch.tsx` reusable component
- [ ] Input field: `"Name eingeben (z.B. Max Müller)"`
- [ ] On change:
  - Call `GET /api/admin/kader/{kader_id}/member-suggestions?search={term}&filter_age_bracket=true`
  - Debounce 300ms
  - Show dropdown with suggestions below input
- [ ] Suggestion display: `{name} ({birth_year}/{gender})`
- [ ] On selection:
  - Call `PUT /api/admin/kader/{kader_id}` with members_add=[id]
  - Clear input (do not auto-clear per spec)
  - Reload member list
  - Show error toast on failure
- [ ] Dropdown shows "Keine Vorschläge" if empty
- [ ] Close dropdown on blur (with delay for click handling)

### Task 2.4: Member List Display
- [ ] Create `web/src/components/KaderMemberList.tsx` component
- [ ] Show member list as compact rows:
  - `☐ {name} ({birth_year}/{gender})` [×] button to remove
- [ ] Remove button calls `PUT /api/admin/kader/{kader_id}` with members_remove=[id]
- [ ] Show loading state while saving
- [ ] Show error toast on failure

### Task 2.5: Save & Loading States
- [ ] Add loading spinner during API calls
- [ ] Show success toast: "Kader gespeichert"
- [ ] Show error toast with API error message on failure
- [ ] Disable buttons while loading

## Phase 3: Integration & Navigation

### Task 3.1: Update Router & Navigation
- [ ] Update `web/src/App.tsx`:
  - Change route from `/admin/teams` to `/admin/kader`
  - Import AdminKaderPage instead of AdminTeamsPage
- [ ] Update `web/src/components/AppShell.tsx`:
  - Change nav label from "Teams" to "Kader"
  - Update link to `/admin/kader`

### Task 3.2: Update Season Selection
- [ ] Ensure `/admin/kader` respects active season
- [ ] If no active season: show message "Bitte aktivieren Sie eine Saison"
- [ ] Consider: should there be season dropdown on page to view past Kader? (Scope question)

### Task 3.3: Keep AdminTeamsPage for Backwards Compatibility
- [ ] Do NOT delete `AdminTeamsPage.tsx` yet (dual-write period)
- [ ] Hide from nav but keep route accessible for testing
- [ ] Document deprecation in code comment

## Phase 4: Testing & Validation

### Task 4.1: Unit Tests — Age Brackets
- [ ] Test age bracket calculation for multiple seasons
- [ ] Test edge cases: leap years, year boundaries
- [ ] Test `MemberBirthYearInBracket` for all age classes

### Task 4.2: Integration Tests — Copy Workflow
- [ ] Test copy empty structure
- [ ] Test copy with same-age members
- [ ] Test copy with age-before members
- [ ] Test auto-assign
- [ ] Verify member duplicates are not created

### Task 4.3: Manual Testing Checklist
- [ ] Create 2025/26 season, then create some Kader manually
- [ ] Verify content-assist filters members by age bracket
- [ ] Copy Kader from 2025/26 to 2026/27 season
  - Test empty copy
  - Test same-age copy
  - Test age-before copy (e.g., A-Jugend 2025/26 → B-Jugend 2026/27)
  - Test auto-assign with mix of genders
- [ ] Add/remove members from Kader
- [ ] Verify no member appears in multiple Kader of same season
- [ ] Test with 0 previous seasons (handle gracefully)

### Task 4.4: UI/UX Testing
- [ ] Modal multi-step flow works smoothly
- [ ] Content-assist dropdown responsive on mobile
- [ ] Accessibility: labels, keyboard navigation, screen reader
- [ ] No broken links or 404s

## Phase 5: Cleanup & Deprecation

### Task 5.1: Documentation
- [ ] Update `CLAUDE.md` with new Kader terminology and API routes
- [ ] Add schema diagram to CLAUDE.md showing kader/kader_members relationship
- [ ] Document age bracket calculation in CLAUDE.md

### Task 5.2: Migration Strategy (Future)
- [ ] Document plan to migrate global `teams` data to per-season `kader`
- [ ] Plan backwards compatibility period (dual-write, gradual deprecation)
- [ ] Write migration script (out of scope for this phase)

### Task 5.3: Remove AdminTeamsPage (Later)
- [ ] After dual-write period, remove `/admin/teams` route
- [ ] Delete `AdminTeamsPage.tsx`
- [ ] Update `CLAUDE.md` to remove Teams API docs
- [ ] Tag as separate commit/PR for cleanup

## Estimation

| Phase | Tasks | Est. Effort |
|-------|-------|------------|
| 1: DB & Core Logic | 1.1–1.6 | ~3 days (dev + tests) |
| 2: Frontend Components | 2.1–2.5 | ~2 days (UI + integration) |
| 3: Integration | 3.1–3.3 | ~0.5 day |
| 4: Testing | 4.1–4.4 | ~1 day |
| 5: Cleanup | 5.1–5.3 | ~0.5 day (5.3 deferred) |
| **Total** | | ~7 days |

## Blocking & Dependencies

- 1.1 (migration) must complete before 1.2 (handler code)
- 1.2–1.6 (backend) can run in parallel
- 2.1 depends on 1.1–1.6 (backend complete)
- 2.2 depends on 1.5 (copy endpoint)
- 2.3 depends on 1.6 (suggestion endpoint)
- 3.1 depends on 2.1–2.5 (frontend complete)
- 4.x (testing) can start once backend is testable

## Success Criteria (from Proposal)

After all tasks complete:
- [ ] `/admin/kader` displays all Kader for selected season
- [ ] Copy workflow creates Kader with member options
- [ ] Auto-assign correctly matches members by birth year + gender
- [ ] Content-assist filters members by age bracket
- [ ] Members can be added/removed at any time
- [ ] Age brackets auto-adjust per season
- [ ] No data loss or schema breaking changes
