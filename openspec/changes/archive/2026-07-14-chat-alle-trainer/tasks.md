# Tasks

## 1. Backend: Mengen-Helfer (Inhalt T + Zugriffskreis Z)

- [x] 1.1 In `internal/chat/team_groups.go` `allTrainersMemberQuery()` (= T): SQL-Fragment `DISTINCT (user_id, name)` = `teamGroupMemberQuery("trainer")` **ohne** den `k.team_id = ?`-Filter (alle Teams, aktive Saison).
- [x] 1.2 `trainerCircleMemberQuery()` (= Z, nur für Nutzersuche): UNION aus `allTrainersMemberQuery()` + `member_club_functions`-Query für `vorstand`/`sportliche_leitung`/`vorstand_beisitzer` (`JOIN members JOIN users`, `m.user_id IS NOT NULL`).
- [x] 1.3 `isInTrainerCircle(ctx, userID) (bool, error)`: `SELECT EXISTS(...)` über (a) Kader-Trainer aktive Saison ∨ (b) `vorstand` ∨ (c) `sportliche_leitung` ∨ (d) `vorstand_beisitzer`. Commit: `feat(chat): Mengen-Helfer für Alle-Trainer (Inhalt + Zugriffskreis)`.

## 2. Backend: „Alle Trainer"-Kachel (Listing + Auflösung)

- [x] 2.1 `teamGroupKinds["alle_trainer"] = true`.
- [x] 2.2 `ListTeamGroups`: wenn `claims.Role == "admin"` ODER `isInTrainerCircle(caller)`, `count` via `countTeamGroupMembers`-`alle_trainer`-Zweig ermitteln; bei `count > 0` Eintrag `{teamId:0, displayShort:"Alle Trainer", kind:"alle_trainer", count}` **voranstellen**.
- [x] 2.3 `countTeamGroupMembers`: `alle_trainer`-Zweig — `COUNT(*)` über `allTrainersMemberQuery()` (= T, nur Kader-Trainer) mit `WHERE user_id != ?` (Caller).
- [x] 2.4 `ResolveTeamGroup`: bei `kind == "alle_trainer"` `canSeeTeamGroup` überspringen, stattdessen `isInTrainerCircle(caller)` prüfen (sonst 403); Mitglieder via `allTrainersMemberQuery()` (= T) mit `WHERE user_id != ? ORDER BY name`. Commit: `feat(chat): Standard-Gruppe "Alle Trainer" (teamübergreifend)`.

## 3. Backend: Kontaktierbarkeit erweitern

- [x] 3.1 `canContactUser` (`handler.go`): nach dem admin/vorstand-Bypass Regel ergänzen — `isInTrainerCircle(caller) && isInTrainerCircle(target)` → true; danach unverändert shared-team.
- [x] 3.2 `Users` (`handler.go`): Nicht-admin/vorstand-Zweig um `UNION`-Teil `trainerCircleMemberQuery()` erweitern, nur wenn `isInTrainerCircle(caller)`; Dedup, `q`-Filter, `LIMIT 50` erhalten. Commit: `feat(chat): teamübergreifender Kontakt im Trainerkreis`.

## 4. Frontend

- [x] 4.1 `ChatPage.tsx`: `TeamGroup["kind"]` um `"alle_trainer"` erweitern; `TEAM_GROUP_KIND_LABEL["alle_trainer"] = "Alle Trainer"`.
- [x] 4.2 Tag-Rendering: für `kind === "alle_trainer"` nur das Label rendern (kein `{label} {displayShort}`); `addTeamGroup` bleibt generisch (`/chat/team-groups/0/alle_trainer/members`). Commit: `feat(chat): "Alle Trainer"-Kachel im Gruppen-Picker`.

## 5. Tests (siehe Test-Anforderungen)

- [x] 5.1 `team_groups_test.go`: Listing-Sichtbarkeit (Trainer/Vorstand/sL/`vorstand_beisitzer` sehen Kachel; Spieler nicht) + Auflösung (teamübergreifend, **nur Trainer** — reiner Vorstand nicht im Ergebnis, aber darf auflösen —, Caller gefiltert, 403 für Nicht-Kreis).
- [x] 5.2 `handler_test.go`: `canContactUser`/`createDirect` teamübergreifend im Kreis (201/200), Spieler↔teamfremder Trainer (403); `Users`-Suche findet teamfremden Trainer für Kreis-Caller, nicht für Spieler. Commit: `test(chat): Alle-Trainer-Gruppe + teamübergreifender Kontakt`.

## 6. Abschluss

- [x] 6.1 `/verify-change` (Build/Test/Lint, brand-Tokens, lucide-Icons, `openspec validate`), dann Proposal archivieren. Commit: `docs(openspec): chat-alle-trainer archiviert`.

---

## Test-Anforderungen

Route/Verhalten → Testname → erwarteter Status (garantierte Invariante).

**`GET /api/chat/team-groups` (Listing der Kachel)**
- `TestListTeamGroups_AlleTrainer_TrainerSiehtKachel` → 200, Eintrag `kind=alle_trainer`, `count = |Kreis|−1`. *Invariante: Kachel nur bei Kreis-Zugehörigkeit, Caller nicht mitgezählt.*
- `TestListTeamGroups_AlleTrainer_SpielerSiehtNicht` → 200 ohne `alle_trainer`-Eintrag. *Invariante: keine Kachel außerhalb des Zugriffskreises.*
- `TestListTeamGroups_AlleTrainer_BeisitzerSiehtKachel` → 200 mit `alle_trainer`-Eintrag. *Invariante: `vorstand_beisitzer` ∈ Zugriffskreis.*

**`GET /api/chat/team-groups/0/alle_trainer/members` (Auflösung)**
- `TestResolveTeamGroup_AlleTrainer_Teamuebergreifend` → 200, enthält Trainer aus ≥2 Teams, ohne Caller. *Invariante: teamübergreifende Vereinigung nur der Kader-Trainer, Caller gefiltert.*
- `TestResolveTeamGroup_AlleTrainer_ReinerVorstandNichtImErgebnis` → 200; ein reiner Vorstand/sL/Beisitzer ohne Kader-Trainer-Zuordnung ist NICHT in der Liste. *Invariante: Inhalt = nur Kader-Trainer.*
- `TestResolveTeamGroup_AlleTrainer_NichtKreisForbidden` → 403. *Invariante: nur Zugriffskreis/admin dürfen auflösen.*

**`POST /api/chat/conversations` (Kontakt)**
- `TestCreateDirect_TrainerZuTeamfremdemTrainer_Erlaubt` → 201/200. *Invariante: Kreis↔Kreis ohne gemeinsames Team erlaubt.*
- `TestCreateDirect_SportlicheLeitungZuTrainer_Erlaubt` → 201/200. *Invariante: sL im Kreis.*
- `TestCreateDirect_SpielerZuTeamfremdemTrainer_Forbidden` → 403. *Invariante: keine Kreis-Rechte für Spieler.*

**`GET /api/chat/users` (Suche)**
- `TestChatUsers_TrainerFindetTeamfremdenTrainer` → Treffer enthält teamfremden Trainer. *Invariante: Kreis-Caller findet gesamten Kreis.*
- `TestChatUsers_SpielerFindetTeamfremdenTrainerNicht` → Treffer enthält ihn nicht. *Invariante: unveränderte Beschränkung außerhalb des Kreises.*
