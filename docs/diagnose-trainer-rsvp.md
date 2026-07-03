# Diagnoseplan: Trainer sehen/können keine RSVP auf `/termine`

**Symptome (Prod):**
- *Matthias Baisch* (Sportlicher Leiter + Trainer mC2): sieht seine mC2-Einträge, kann aber nichts eintragen (keine Buttons).
- *Florian Steinle* (Trainer + Vater von Lias): sieht seine eigenen Termine gar nicht, kann nur für sein Kind ab-/zusagen.
- *Brigitte Bilner* (Trainer): funktioniert.

**Hypothese:** „Trainer sein" wird an zwei unabhängigen Stellen erkannt:
- **Sichtbarkeit** der Termine (`ListSessions`/`ListMyGames`) ist durch `HasFunction("trainer")` gegated — das kommt aus `member_club_functions` (global, teamlos).
- **`my_rsvp`-Default + Buttons** basieren auf `kader_trainers` (die echte Team+Saison-Zuordnung; `trainer_memberships` ist nur eine View darüber).

Laufen die beiden auseinander, bricht `/termine`. Zusätzlich hängt die Button-Sichtbarkeit im Frontend an `my_rsvp != null` → Henne-Ei-Falle für Trainer, die noch nie geantwortet haben.

---

## DB read-only öffnen (kein Schreibrisiko)

```bash
sqlite3 -readonly /var/lib/teamwerk/teamwerk.db
```
```
.headers on
.mode box
```

---

## Q0 — Aktive Saison + mC2-Team-ID

```sql
SELECT id, name, is_active FROM seasons WHERE is_active = 1;

-- mC2-Team-Kandidaten (Name ggf. anpassen):
SELECT DISTINCT t.id AS team_id, t.name, k.season_id, k.age_class, k.gender, k.team_number
FROM teams t
JOIN kader k ON k.team_id = t.id
WHERE t.name LIKE '%C2%';
```
→ mC2 `team_id` und aktive `season_id` notieren (für Q2).

---

## Q1 — Identität der drei Personen (über Login-User)

```sql
SELECT
  u.id  AS user_id, u.email,
  m.id  AS member_id,
  m.user_id AS member_user_id,        -- muss = user_id sein, sonst Verknüpfung kaputt
  m.first_name || ' ' || m.last_name AS member_name,
  (SELECT GROUP_CONCAT(f.function, ',') FROM member_club_functions f WHERE f.member_id = m.id) AS funcs,
  (SELECT COUNT(*) FROM family_links fl WHERE fl.parent_user_id = u.id)                        AS is_parent,
  (SELECT GROUP_CONCAT(k.team_id || '/s' || k.season_id, ', ')
     FROM kader_trainers kt JOIN kader k ON k.id = kt.kader_id
     WHERE kt.member_id = m.id)        AS trainer_kaders
FROM users u
LEFT JOIN members m ON m.user_id = u.id
WHERE u.last_name IN ('Baisch','Steinle','Bilner')
   OR u.email LIKE '%baisch%' OR u.email LIKE '%steinle%' OR u.email LIKE '%bilner%';
```

## Q1b — Member-Sicht (fängt „Member existiert, aber nicht mit Login verknüpft" + Duplikate)

```sql
SELECT
  m.id AS member_id, m.first_name || ' ' || m.last_name AS name,
  m.user_id AS linked_user,           -- NULL = nicht mit Login verknüpft!
  (SELECT GROUP_CONCAT(function, ',') FROM member_club_functions f WHERE f.member_id = m.id) AS funcs,
  (SELECT GROUP_CONCAT(k.team_id || '/s' || k.season_id, ', ')
     FROM kader_trainers kt JOIN kader k ON k.id = kt.kader_id WHERE kt.member_id = m.id)     AS trainer_kaders
FROM members m
WHERE m.last_name IN ('Baisch','Steinle','Bilner');
```

---

## Q2 — ENTSCHEIDEND: `my_rsvp` exakt wie das Backend reproduzieren

Rechnet die drei EXISTS-Zweige und den effektiven `my_rsvp` genau wie `ListMyGames`.
**Zwei Werte oben anpassen** (aus Q0/Q1): `uid` = User-ID der Person, `teamid` = mC2-Team-ID.

```sql
WITH p(uid, teamid) AS (VALUES (999, 999)),     -- << ANPASSEN: (user_id, mC2 team_id)
     me(member_id) AS (SELECT id FROM members WHERE user_id = (SELECT uid FROM p))
SELECT
  g.id AS game_id, g.date, g.season_id,
  (SELECT member_id FROM me) AS my_member_id,   -- NULL => Login nicht mit Member verknüpft (Ursache 1)
  EXISTS(SELECT 1 FROM game_teams gt JOIN kader k ON k.team_id=gt.team_id AND k.season_id=g.season_id
         JOIN kader_members km ON km.kader_id=k.id AND km.member_id=(SELECT member_id FROM me)
         WHERE gt.game_id=g.id) AS in_regular,
  EXISTS(SELECT 1 FROM game_teams gt JOIN kader k ON k.team_id=gt.team_id AND k.season_id=g.season_id
         JOIN kader_extended_members kem ON kem.kader_id=k.id AND kem.member_id=(SELECT member_id FROM me)
         WHERE gt.game_id=g.id) AS in_extended,
  EXISTS(SELECT 1 FROM game_teams gt JOIN kader k ON k.team_id=gt.team_id AND k.season_id=g.season_id
         JOIN kader_trainers ktr ON ktr.kader_id=k.id AND ktr.member_id=(SELECT member_id FROM me)
         WHERE gt.game_id=g.id) AS in_trainer,   -- 1 erwartet, wenn er das Team trainiert
  (SELECT status FROM game_responses gr WHERE gr.game_id=g.id AND gr.member_id=(SELECT member_id FROM me)) AS explicit_response
FROM games g
WHERE g.id IN (SELECT game_id FROM game_teams WHERE team_id = (SELECT teamid FROM p))
ORDER BY g.date DESC
LIMIT 10;
```

**Für alle drei wiederholen** (jeweils `uid` tauschen; `teamid` = das Team, das die Person trainiert).

---

## Q3 — Brigitte-Maskierung prüfen (hat sie schon aktiv geantwortet?)

```sql
SELECT gr.game_id, gr.status, m.first_name || ' ' || m.last_name AS name
FROM game_responses gr JOIN members m ON m.id = gr.member_id
WHERE m.last_name = 'Bilner';
```
→ Zeilen vorhanden ⇒ Brigitte funktioniert **nur** über echte Responses; der Trainer-Default-Zweig ist damit ungetestet/verdächtig.

---

## Q4 — Systemischer Scan: wie viele Trainer sind betroffen?

```sql
-- (a) Florian-Typ: im Kader-Trainerstab, aber OHNE Club-Funktion 'trainer'
--     => Termine unsichtbar (Sicht-Gate greift nicht)
SELECT DISTINCT m.id, m.first_name || ' ' || m.last_name AS name
FROM kader_trainers kt JOIN members m ON m.id = kt.member_id
WHERE NOT EXISTS (SELECT 1 FROM member_club_functions f
                  WHERE f.member_id = m.id AND f.function = 'trainer');

-- (b) Matthias-Typ (Verknüpfung): Kader-Trainer ohne Login-Verknüpfung
--     => memberID=0 => keine Buttons, kein Response möglich
SELECT DISTINCT m.id, m.first_name || ' ' || m.last_name AS name, m.user_id
FROM kader_trainers kt JOIN members m ON m.id = kt.member_id
WHERE m.user_id IS NULL;

-- (c) Duplikate: gleiche Person mehrfach als Member (Login zeigt evtl. auf den falschen Record)
SELECT first_name, last_name, COUNT(*) AS n, GROUP_CONCAT(id) AS member_ids
FROM members GROUP BY first_name, last_name HAVING COUNT(*) > 1;
```

---

## Interpretations-Leitfaden

| Q2-Ergebnis für die Person | Ursache | Fix |
|---|---|---|
| `my_member_id` ist NULL | Login nicht mit Member verknüpft | **DATEN** (verknüpfen) |
| `in_trainer=0`, `my_member_id` gesetzt, obwohl Person das Team trainiert | nicht im Kader-Trainerstab (oder Kader-/Spiel-Saison ≠) | **DATEN** (`kader_trainers` / Saison) |
| `in_trainer=1`, `explicit_response=NULL`, aber keine Buttons in der App | Backend-`my_rsvp` müsste `confirmed` sein → Fehler woanders | **CODE** (Frontend/JWT-Pfad) |
| `explicit_response` gesetzt (Brigitte) | funktioniert nur via Alt-Response | **CODE** (Default-Zweig robust — Fix D) |

### Kandidaten-Fixes (nach Diagnose zu schneiden)

- **A — Backend:** `HasFunction("trainer")`-Gate an der Trainer-Sicht in `ListSessions` + `ListMyGames` entfernen → Sichtbarkeit rein über `kader_trainers`. (fixt Florians Unsichtbarkeit)
- **B — Frontend:** In `TerminePage` bei Elternteil **und** eigener Teilnahme (`my_rsvp != null`) beide Button-Zeilen rendern statt entweder/oder. (fixt „nur fürs Kind")
- **D — Robustheit:** Button-Sichtbarkeit nicht an `my_rsvp != null` hängen, sondern an einem expliziten Backend-Signal „du bist Teilnehmer dieses Termins" (`kader_members ∪ kader_extended ∪ kader_trainers`, mit aufgelöster memberID). (behebt Henne-Ei; deckt Matthias, sobald Identität sauber)
- **C — Daten:** je nach Q2/Q4-Ergebnis Verknüpfung/Trainer-Eintrag korrigieren (kein Code).

---

## Auszuwertende Ausgaben zurückschicken

Q1, Q1b, Q2 (alle drei Personen), Q3, Q4 — daraus wird der Change-Proposal exakt zugeschnitten (Code A/B/D vs. Datenkorrektur C, inkl. konkreter `UPDATE`/`INSERT`-Statements für die Datenfälle).
