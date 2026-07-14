# Design

## Zwei Mengen sauber trennen (Zugriff ≠ Inhalt)

Der entscheidende Punkt: **wer die Kachel sehen/nutzen darf** ist ein anderer Kreis als **wer in der Gruppe landet**.

- **Zugriffskreis Z** (Sichtbarkeit der Kachel · Auflöse-Berechtigung · 1:1-Kontakt · Nutzersuche): User, die **(a)** Kader-Trainer der aktiven Saison sind ODER **(b)** Vereinsfunktion `vorstand` ODER **(c)** `sportliche_leitung` ODER **(d)** `vorstand_beisitzer` haben. `admin` stets berechtigt.
- **Mitgliedermenge T** (Inhalt der „Alle Trainer"-Gruppe): **NUR** Kader-Trainer der aktiven Saison — also Bedingung (a) allein. Vorstand, sportliche Leitung und vorstand_beisitzer sind **nicht** enthalten, es sei denn, sie sind selbst Kader-Trainer dieser Saison.

Es gilt **T ⊆ Z**. Beide werden **DB-abgeleitet** bestimmt (nicht rein aus JWT-Claims), weil ein Kader-Trainer nicht zwingend die Vereinsfunktion `trainer` trägt — die bestehende `trainer`-Auflösung in `team_groups.go` geht ebenfalls über `kader_trainers`, nicht über die Claim.

Empfohlene Helfer in `internal/chat`:

- `allTrainersMemberQuery()` — SQL-Fragment `DISTINCT (user_id, name)` für **T**: identisch zu `teamGroupMemberQuery("trainer")`, nur **ohne** den `k.team_id = ?`-Filter (alle Teams). Genutzt in `ResolveTeamGroup` und `countTeamGroupMembers` für `kind="alle_trainer"`.
- `isInTrainerCircle(ctx, userID) (bool, error)` — `SELECT EXISTS(...)` über (a)∨(b)∨(c)∨(d) = Zugehörigkeit zu **Z**. Für den Caller darf (b)/(c)/(d) aus `claims.HasFunction(...)` kurzgeschlossen werden.
- `trainerCircleMemberQuery()` — SQL-Fragment `DISTINCT (user_id, name)` für **Z**: UNION aus `allTrainersMemberQuery()` + `member_club_functions`-Query für `vorstand`/`sportliche_leitung`/`vorstand_beisitzer`. Genutzt **nur** für die Nutzersuche-Erweiterung (`GET /chat/users`), nicht für die Gruppenauflösung.

**Ersteller-Caveat:** Die „Alle Trainer"-Gruppe entsteht wie jede andere über `createGroup`, das den Ersteller (`claims.UserID`) stets als Teilnehmer einfügt. Legt ein Vorstand/sL (∈ Z, ∉ T) die Gruppe an, ist er als **Ersteller** drin — die aufgelöste Mitgliederliste (T) enthält ihn aber nicht. Das ist inhärent und gewollt (der Ersteller ist Teilnehmer seiner Gruppe).

## „Alle Trainer" als synthetische Kachel (kein neues Primitiv)

Die Kachel reiht sich in den bestehenden Team-Gruppen-Mechanismus ein:

```
   ListTeamGroups  ── stellt (wenn Caller ∈ Z ∨ admin) EINEN Eintrag voran:
                      { teamId: 0, displayShort: "Alle Trainer",
                        kind: "alle_trainer", count: |T| − (Caller∈T?1:0) }

   addTeamGroup    ── Frontend baut wie gehabt:
                      GET /chat/team-groups/0/alle_trainer/members
                      → Mitgliederliste (T ohne Caller = nur Kader-Trainer)
                      → in Auswahl übernehmen

   createGroup     ── unverändert: normale group-Conversation (Momentaufnahme)
```

- **Sentinel `teamId = 0`.** Der bestehende Routenparameter `{teamId}/{kind}/members` matcht `/0/alle_trainer/members` ohne neue Route. `ResolveTeamGroup` verzweigt bei `kind == "alle_trainer"`: **kein** `canSeeTeamGroup`-Check (der ist teambezogen), stattdessen `isInTrainerCircle(caller)` (= Z) → sonst 403. Die Mitglieder kommen aus `allTrainersMemberQuery()` (= T, nur Kader-Trainer).
- `teamGroupKinds["alle_trainer"] = true`, sonst schlägt die `kind`-Validierung mit 400 fehl.
- `countTeamGroupMembers` bekommt einen `alle_trainer`-Zweig (COUNT über `allTrainersMemberQuery()` minus Caller). Bei `count == 0` wird die Kachel — wie bei den Team-Gruppen — weggelassen.

**Warum synthetisch statt echter Dauer-Gruppe:** bewusst gewählt (Nutzerentscheidung). Kein `reconcile`, keine berechnete Live-Mitgliedschaft, keine Sonderfälle in Unread/Push/SSE. Die erzeugte Gruppe ist eine gewöhnliche `group`-Conversation; alle bestehenden Chat-Mechaniken greifen unverändert.

## Kontaktierbarkeit erweitern

`canContactUser(caller, target)` — neue Reihenfolge:

```
   1. caller ist admin ODER vorstand            → true   (bestehend)
   2. caller ∈ Z UND target ∈ Z                 → true   (NEU: Zugriffskreis)
   3. gemeinsames Team (user_accessible_teams)  → true   (bestehend)
   4. sonst                                       false
```

- Deckt **alle** geforderten Fälle ab: Trainer↔Trainer (2), Trainer↔sL (2), Trainer↔Vorstand (Regel 1 greift, wenn Vorstand *Caller* ist; ist der Trainer Caller, greift 2, da Vorstand ∈ Z). sportliche Leitung als Caller erreicht jeden Trainer über 2.
- `createGroup` prüft `canContactUser` je Mitglied — mit Regel 2 passieren die aufgelösten „Alle Trainer"-Mitglieder (alle ∈ T ⊆ Z) die Prüfung, wenn der Ersteller ∈ Z ist.

`GET /chat/users` (Handler `Users`) — Suchmenge für Nicht-admin/vorstand:

```
   bisher:  User mit gemeinsamem Team
   neu:     ∪  (falls Caller ∈ M)  alle Mitglieder von M
```

Umsetzung: zweiter Zweig als `UNION` der bestehenden shared-team-Query mit `trainerCircleMemberQuery()`, nur wenn `isInTrainerCircle(caller)`; Dedup nach `user_id`, `q`-Filter und `LIMIT 50` bleiben.

## Frontend

- `TeamGroup["kind"]` → `"trainer" | "spieler" | "eltern" | "alle_trainer"`.
- `TEAM_GROUP_KIND_LABEL["alle_trainer"] = "Alle Trainer"`.
- Tag-Rendering: für `alle_trainer` nur das Label anzeigen (kein `{label} {displayShort}`-Suffix), da `displayShort` selbst „Alle Trainer" bzw. leer ist. `tagKey` = `"0:alle_trainer"` ist eindeutig.
- Keine weitere Änderung am „Neue Gruppe"-Flow.

## Nicht im Scope

- Mitwachsende/berechnete Live-Mitgliedschaft, Reconcile-Trigger.
- Dauerhafter, geseedeter Vereinskanal.
- Eigene Kachel für „alle Spieler"/„alle Eltern" (nur Trainer angefragt).
- `vorstand_beisitzer`.
