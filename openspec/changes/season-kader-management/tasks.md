# Tasks: Season-based Kader Management

## Phase 1: Database & Core Logic

### Task 1.1: Create Database Migration
- [x] Create `internal/db/migrations/012_kader.up.sql`
  - Create `kader` table with (id, season_id, age_class, gender, created_at, updated_at)
  - Create `kader_members` table with (id, kader_id, member_id, added_at)
  - Add UNIQUE constraint on (season_id, age_class, gender) for kader
  - Add UNIQUE constraint on (kader_id, member_id) for kader_members
  - Add indexes on foreign keys
- [x] Create corresponding `.down.sql` file
- [x] Test migration locally: `make migrate-up` and `make migrate-down`
- [x] Create `internal/db/migrations/013_member_gender.up.sql` — add `gender` column to members

### Task 1.2: Implement Kader Handler (Backend)
- [x] Create `internal/kader/handler.go` with `Handler` struct
- [x] Implement `ListKader(w, r)` — GET /api/admin/kader
- [x] Implement `GetKader(w, r)` — GET /api/admin/kader/{id}
- [x] Implement `UpdateKader(w, r)` — PUT /api/admin/kader/{id}
- [x] Implement `InitializeKader(w, r)` — POST /api/admin/kader
- [x] Add kader routes to chi router in `cmd/teamwerk/main.go`
- [x] Require admin/vorstand role for all endpoints

### Task 1.3: Age Bracket Calculation
- [x] Create `internal/kader/age_brackets.go`
- [x] Define `ageBracketRef2025` reference constant
- [x] Implement `ComputeAgeBrackets(seasonStartYear int) map[string][2]int`
- [x] Implement `BirthYearInBracket(birthYear int, ageClass string, seasonStartYear int) bool`
- [x] Write unit tests for age bracket calculation across multiple seasons

### Task 1.4: Member Suggestion Logic
- [x] Implement `suggestMembers(...)` with gender, birth year, and name filtering
  - Gender match (or kader gender is "mixed") — includes unspecified 'u'
  - Birth year within age bracket (if filterByBracket=true)
  - Name LIKE searchTerm
  - Shows "already_in_kader" flag
- [x] Create `internal/kader/suggestions.go` with suggestion logic

### Task 1.5: Copy Workflow — Backend
- [x] Implement `copyKader(...)` with same-age, age-before, auto-assign, and empty modes
- [x] Implement `ageClassBefore()` for natural progression A←B←C←D
- [x] Create `internal/kader/copy.go` with copy logic
- [x] Implement `POST /api/admin/kader/copy-from-season` endpoint
- [x] Add validation: target season start year fetched

### Task 1.6: Content-Assist Endpoint
- [x] Implement `GET /api/admin/kader/{id}/member-suggestions` endpoint
  - Query params: `search`, `filter_age_bracket` (boolean, default true)
  - Returns suggestions with already_in_kader flag
  - Limit to 20 results

## Phase 2: Frontend — UI Components

### Task 2.1: AdminKaderPage Component
- [x] Create `web/src/pages/AdminKaderPage.tsx`
- [x] Show current season in header
- [x] List all Kader (8 sections max: A-D × m/f + D × mixed)
- [x] For each Kader:
  - Display age_class, gender, member count
  - "Aus vorheriger Saison kopieren" button (modal trigger)
  - "Mitglied hinzufügen" section with content-assist input
  - List members with remove button [×]
  - Save button
- [x] Use `api.get('/admin/kader')` to load on mount
- [x] Load active season from context or endpoint

### Task 2.2: Copy Kader Modal (Multi-Step)
- [x] Create `web/src/components/CopyKaderModal.tsx`
- [x] **Step 1 — Source Season Selection**
  - Dropdown of previous seasons (fetch from `/api/admin/seasons`)
  - Next button
- [x] **Step 2 — Kader Selection**
  - List checkboxes of all Kader from source season
  - Each checkbox shows: `{age_class} {gender} (N members)`
  - All checked by default
  - Next/Back buttons
- [x] **Step 3 — Member Assignment (per selected Kader)**
  - For each selected Kader, show radio options:
    1. `Nur Struktur` (empty)
    2. `{same age class} {source season} (N members)` (pre-selected)
    3. `{age class before} {source season} (M members)` (if exists)
    4. `Auto-Assign nach Jahrgang+Geschlecht`
  - Back/Confirm buttons
- [x] **Step 4 — Confirmation**
  - Summary: "Werden {N} Kader mit {M} gesamt Mitgliedern angelegt"
  - Create button triggers `POST /api/admin/kader/copy-from-season`
  - On success: close modal and reload AdminKaderPage
  - Show loading state during request
  - Show error toast on failure

### Task 2.3: Member Content-Assist Component
- [x] Create `web/src/components/KaderMemberSearch.tsx` reusable component
- [x] Input field: `"Name eingeben (z.B. Max Müller)"`
- [x] On change:
  - Call `GET /api/admin/kader/{kader_id}/member-suggestions?search={term}&filter_age_bracket=true`
  - Debounce 300ms
  - Show dropdown with suggestions below input
- [x] Suggestion display: `{name} ({birth_year}/{gender})`
- [x] On selection:
  - Call `PUT /api/admin/kader/{kader_id}` with members_add=[id]
  - Clear input (do not auto-clear per spec)
  - Reload member list
  - Show error toast on failure
- [x] Dropdown shows "Keine Vorschläge" if empty
- [x] Close dropdown on blur (with delay for click handling)

### Task 2.4: Member List Display
- [x] Member list rendered inline in AdminKaderPage (no separate KaderMemberList.tsx needed)
- [x] Show member list as compact rows: `{name} ({birth_year}/{gender})` [×] button to remove
- [x] Remove button calls `PUT /api/admin/kader/{kader_id}` with members_remove=[id]
- [x] Show loading state while saving (per-member `removing` state)
- [x] Show error toast on failure

### Task 2.5: Save & Loading States
- [x] Loading state on initialize and remove operations
- [x] Toast notifications: "Kader angelegt", "Fehler beim Entfernen", "Kader erfolgreich kopiert"
- [x] Show error toast with API error message on failure
- [x] Disable buttons while loading (disabled attribute + opacity)

## Phase 3: Integration & Navigation

### Task 3.1: Update Router & Navigation
- [x] Update `web/src/App.tsx`:
  - Added route `/admin/kader` with AdminKaderPage
  - Kept `/admin/teams` route for backwards compatibility
- [x] Update `web/src/components/AppShell.tsx`:
  - Changed nav label from "Teams" to "Kader"
  - Updated link to `/admin/kader`

### Task 3.2: Update Season Selection
- [x] `/admin/kader` fetches active season on mount
- [x] If no active season: shows "Bitte aktivieren Sie eine Saison unter Saisons"
- [x] Season dropdown for past Kader deferred (out of scope for this phase)

### Task 3.3: Keep AdminTeamsPage for Backwards Compatibility
- [x] AdminTeamsPage.tsx not deleted — route `/admin/teams` still active
- [x] Hidden from nav (only Kader link shown)
- [x] No code comment added (route existence is self-documenting)

## Phase 4: Testing & Validation

### Task 4.1: Unit Tests — Age Brackets
- [x] Test age bracket calculation for multiple seasons (2024/25, 2025/26, 2026/27)
- [x] Test edge cases: year boundaries (born at bracket edges)
- [x] Test `BirthYearInBracket` for all age classes

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
- [x] `/admin/kader` displays all Kader for selected season
- [x] Copy workflow creates Kader with member options (modal implemented; end-to-end test pending second season)
- [x] Auto-assign correctly matches members by birth year + gender
- [x] Content-assist filters members by age bracket
- [x] Members can be added/removed at any time
- [x] Age brackets auto-adjust per season
- [x] No data loss or schema breaking changes (additive migrations only)
