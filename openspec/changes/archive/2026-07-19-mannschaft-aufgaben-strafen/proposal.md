## Why

Mannschaften organisieren sich abseits des Spielbetriebs über zwei informelle Mechanismen, die heute nur auf Zetteln/WhatsApp existieren: **Aufgaben** (wer ist für Mannschaftskasse, Leibchen, Harz … zuständig) und die **Strichliste/Mannschaftskasse** (Strafen fürs vergessene Trikot, den vergessenen Harztopf, den Aufsetzer übers Tor). Beide gehören sichtbar in die App — die Aufgaben offen im Team, die Strafen bewusst nur teamintern (Spieler + Trainer, ausdrücklich **nicht** Eltern).

## What Changes

- **Aufgaben (Responsibilities):** Der Trainer pflegt pro Kader einen anpass-/erweiterbaren Catalog von Aufgaben-Labels und weist sie Spielern zu (Dropdown + Freitext, keine weitere Semantik). Sie erscheinen als Chips neben dem Spielernamen im Team-Tab der Mein-Team-Seite und reiten auf der bestehenden Roster-Response mit.
- **Strafen (Penalties):** Eine pro Kader ernannte Rolle **Strafenwart** vergibt Strafen mit Betrag an Teammitglieder. Beträge sind unterschiedlich hoch (Default aus einem trainer-gepflegten Catalog, pro Fall editierbar). Strafen werden über einen **eigenen Endpoint mit strengerem Read-Gate** ausgeliefert (Spieler + Trainer des Teams, Stamm- und Erweiterter Kader — **keine** Eltern, niemand außerhalb).
- **Strafenwart als per-Kader-Appointment** (Table `kader_strafenwarte`, Sibling von `kader_trainers`), vom Trainer ernannt — **kein** neuer globaler `member_club_functions`-Wert, kein neuer JWT-Claim.
- **Zwei Lösch-Operationen bei Strafen**, beide echt löschend (kein Status/Strikethrough): **Storno** (eine einzelne Strafe, Fehleingabe/erlassen) und **Zurücksetzen je Spieler** (alle Strafen eines Members, wenn abgegolten).
- Alles ist **per Kader / aktive Saison** scoped, konsistent mit dem bestehenden `kader_members`/`kader_trainers`-Modell.

## Capabilities

### New Capabilities
- `spieler-aufgaben`: Trainer-gepflegter Aufgaben-Catalog pro Kader + Zuweisung an Spieler (Snapshot-Label), Anzeige auf der Roster-Response (Roster-Sichtbarkeit inkl. Eltern).
- `mannschaftsstrafen`: Strafenwart-Appointment pro Kader, Strafen-Catalog, Vergeben/Storno/Zurücksetzen von Strafen, sowie das teaminterne Read-Gate (Spieler + Trainer, ohne Eltern).

### Modified Capabilities
<!-- Keine bestehende Capability ändert ihre Requirements. member_club_functions bleibt unverändert (bewusst kein neuer Funktionswert). Die Roster-Response wird additiv erweitert, ohne bestehende Requirements der teams-Routen zu brechen. -->

## Impact

- **Neue Migration** (`internal/db/migrations/`, nächste freie Nummer, `.up.sql` + `.down.sql`): `responsibility_types`, `penalty_types`, `kader_strafenwarte`, `member_responsibilities`, `team_penalties` (alle kader-scoped; Geldbeträge in Cent).
- **Backend:** `internal/teams/handler.go` (`GetRoster` liefert Aufgaben additiv mit); neuer Handler/Package für Strafen + Aufgaben-Mutationen; neue Routen in `internal/app/router.go` (`BuildRouter`) unter dem Authenticated- bzw. Trainer-Tier; neues Read-Gate (Spieler-oder-Trainer-des-Teams), das es so noch nicht gibt — Erweiterung im Stil des Trainer-Scopings in `internal/policy/rules.go`.
- **SSE (Hard Rule):** jede Mutations-Route ruft `h.hub.Broadcast(...)` (Events `responsibilities`, `penalties`); vom `internal/arch/broadcast_test.go`-Gate erfasst.
- **Frontend:** `web/src/pages/MeinTeamPage.tsx` — Aufgaben-Chips im Team-Tab, Strafen als eigener Bereich/Tab nur für Berechtigte (Liste + Kassenstand-Summe pro Spieler), Trainer-Verwaltung (Catalog, Ernennung, Zuweisung) und Strafenwart-Aktionen inline; `useLiveUpdates`, `brand-*`-Tokens, `lucide-react`.
- **Tests (Hard Rule):** Happy-Path + Fehlerfall je neuer Route; Fixtures in `internal/testutil/`.

## Test-Anforderungen

Jede neue Route bekommt Happy-Path + Fehlerfall. Route → Testname → erwarteter Status → garantierte Invariante:

**Aufgaben (spieler-aufgaben)**
- `POST /api/teams/{id}/responsibility-types` → `TestResponsibilityCatalog_TrainerCreates_200` → 200/201 → Trainer des Kaders darf Catalog erweitern.
- `POST /api/teams/{id}/responsibility-types` → `TestResponsibilityCatalog_NonTrainer_403` → 403 → nur Trainer/admin darf pflegen.
- `POST /api/teams/{id}/responsibilities` → `TestResponsibilityAssign_Trainer_200` → 200/201 → Zuweisung erscheint auf Roster-Response.
- `GET /api/teams/{id}/roster` → `TestRoster_IncludesResponsibilities` → 200 → Aufgaben-Labels je Spieler enthalten (auch für Eltern-Sicht).
- Invariante `TestResponsibility_CatalogEditKeepsSnapshot` → Catalog-Edit ändert bereits zugewiesenes Label **nicht**.

**Strafen (mannschaftsstrafen)**
- `GET /api/teams/{id}/penalties` → `TestPenalties_Player_200` / `TestPenalties_ExtendedMember_200` → 200 → Spieler + Erweiterter Kader dürfen lesen.
- `GET /api/teams/{id}/penalties` → `TestPenalties_Parent_403` → 403 → **Eltern sehen keine Strafen** (Kern-Invariante).
- `GET /api/teams/{id}/penalties` → `TestPenalties_Outsider_403` → 403 → Außenstehende gesperrt.
- `GET /api/teams/{id}/roster` → `TestRoster_ExcludesPenalties` → 200 → Roster-Response enthält **keine** Strafendaten.
- `POST /api/teams/{id}/penalties` → `TestPenaltyCreate_Strafenwart_200` → 200/201 → Strafenwart darf vergeben; Betrag als Snapshot.
- `POST /api/teams/{id}/penalties` → `TestPenaltyCreate_NonStrafenwart_403` → 403 → nur Strafenwart des Kaders.
- `POST /api/teams/{id}/penalties` → `TestPenaltyCreate_ForeignTeamStrafenwart_403` → 403 → **Strafenwart Team A kann Team B nicht bestrafen** (keine Row angelegt).
- `DELETE /api/teams/{id}/penalties/{pid}` → `TestPenaltyStorno_Strafenwart_200` → 200 → einzelne Strafe hart gelöscht.
- `DELETE /api/teams/{id}/penalties?member={mid}` → `TestPenaltyReset_PerMember_200` → 200 → alle Strafen des Members gelöscht, Kassenstand 0.
- `POST /api/teams/{id}/strafenwarte` → `TestStrafenwartAppoint_Trainer_200` / `_NonTrainer_403` → 200/403 → nur Trainer ernennt.
- Invariante `TestPenalty_CatalogEditKeepsSnapshot` → Catalog-Betrag-Edit ändert vergebene Strafe **nicht**.
- Invariante `TestClubFunctions_NoStrafenwartValue` → CHECK-Constraint von `member_club_functions` enthält keinen `strafenwart`-Wert.
- Broadcast-Gate (`internal/arch/broadcast_test.go`) erfasst alle Mutations-Routen automatisch (`responsibilities`/`penalties`).

## Non-Goals

- **Keine Zahlungs-Historie / kein Audit-Trail:** Storno und Zurücksetzen löschen echt; die App hält nur den aktuell offenen Kassenstand. Ein Nachweis „wer hat wann wie viel bezahlt" wäre ein separater Follow-up-Change.
- **Keine Semantik für Aufgaben** (keine Erinnerungen, kein Workflow) — reine Darstellung.
- **Kein neuer globaler Vereinsfunktions-Wert** und keine Änderung am zweidimensionalen Berechtigungsmodell.
