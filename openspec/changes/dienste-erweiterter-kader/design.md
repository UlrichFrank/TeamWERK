# Design

## Context

Beobachteter Stand (Code, 24.09.2026):

- `duties.Board` (`internal/duties/handler.go`) bildet die Team-Quelle nicht privilegierter
  Nutzer aus `player_memberships` (View über `kader_members`, also nur Stammkader) für den
  Nutzer und seine Kinder sowie aus `trainer_memberships`. Der `'eltern'`-Audience-Match
  nutzt ebenfalls `player_memberships`.
- `duties.Claim` prüft nur `canActForUser` (selbst oder Kind), **kein** Team. Belegen
  scheitert heute allein an der Sichtbarkeit.
- `dashboard` hat zwei eigene SQL-Kopien derselben Frage: `dutyTeamQuery` und
  `audienceMatchClauseSQL`. `queryNextEvents` kennt den erweiterten Kader bereits
  (`extended_teams`/`primary_teams`-CTE, `isExtended`) — Vorlage für die Abgrenzung.
- `dutyfairness.Snapshot` lädt Mitglieder nur aus `kader_members`. Zählungen hängen **am
  Mitglied**, nicht an (Mitglied, Team): ein Kind in zwei Teams erscheint in beiden
  Ranglisten mit denselben Zahlen. Stufe 3 der Zurechnung („eigene Mitglieder unabhängig
  vom Team“) fängt heute jede Zuweisung an einem fremden Team ab.
- `ExtendedBadge` („Erw. Kader“) existiert lokal in `DashboardPage.tsx`.

## Goals / Non-Goals

**Goals:**
- Erweiterter Kader + Eltern sehen und belegen die Dienste des Teams.
- Aushilfe ist an Gruppe, Eingetragenem, Dashboard und Rangliste erkennbar.
- Aushilfe verändert keine Pflicht-Zahl (Soll, Stammteam-Geleistet, Rangfolge).

**Non-Goals:**
- Keine Push für neue Slots an den erweiterten Kader.
- Keine Anrechnung von Aushilfe auf irgendein Soll, keine „Gutschrift“-Logik.
- Kein Eingriff in `Claim` (bleibt teamfrei, wie bisher).
- Keine Änderung an Dienst-Erinnerungen (`duty-reminder-emails`): sie erinnern an
  **offene** Dienste und gehen an nicht eingetragene Stammkader-Mitglieder
  (`scheduler.eligibleUsers`) — der erweiterte Kader bleibt dort bewusst draußen, wie bei
  der Push für neue Slots. Die Absage eines Dienstes geht dagegen an **alle Eingetragenen**
  (`assignedUsers`), eine eingetragene Aushilfe also eingeschlossen; das präzisiert die Spec
  `terminmeldung-empfaenger` (Delta in diesem Change).
- Kein neues Schema.

## Decisions

### 1. Aushilfe wird berechnet, nicht gespeichert

Kein `duty_assignments.is_aushilfe`. Das Kennzeichen folgt aus dem aktuellen Kader der
aktiven Saison — derselbe Grundsatz wie in `dutyfairness` („Existenz + Datum, nie ein
gespeicherter Zustand“). Wird eine Aushilfe in den Stammkader übernommen, zählen ihre
Dienste ab dann als Pflicht. Das ist gewollt: die Bilanz beschreibt den Kader, wie er ist.
Alternative (einfrieren beim Claim) verworfen: bräuchte eine Migration, einen
Backfill für Bestandszuweisungen und erzeugte eine zweite Wahrheit neben dem Kader.

### 2. Zurechnung: zwei neue Stufen zwischen den bisherigen Stufen 2 und 3

`countAssignments` erhält Stufen 3/4 (eigene Mitglieder bzw. Kinder mit
erweitertem Kader in einem Slot-Team). Die bisherige Stufe 3 wird zu Stufe 5 und bleibt
für Fälle ohne jede Kader-Verbindung (z. B. Vorstand belegt vereinsweiten Dienst). Die
Reihenfolge „eigen vor Kind“ spiegelt die bestehenden Stufen 1/2.

Aushilfe-Werte liegen **nicht** am `Member`, sondern in
`Snapshot.aushilfe map[memberTeamKey]*Counts` — sonst schlüge eine Aushilfe in Team B über
den Member-Zähler in jede Stammteam-Rangliste durch. Der Snapshot lädt dafür zusätzlich
`kader_extended_members` (aktive Saison, `team_id IS NOT NULL`, `status <> 'ausgetreten'`)
als `extTeams map[memberID][]teamID`, dazu eigene Zuordnungen `extOwnByUser`/
`extChildrenByUser` — bewusst getrennt von `ownByUser`/`childrenByUser`, damit Stufen 1, 2
und 5 sich nicht verändern. Mitglieder, die nur im erweiterten Kader stehen, werden
**keinem** `Team.Members` hinzugefügt und zählen nicht in `PlayerCount` — das Soll des
fremden Teams bleibt unberührt. Die Positionen hängen als `Team.Aushilfe` am Team.

Stufen 3/4 greifen **nicht** bei generischen Slots (ohne Team) und **nicht**, wenn der
Account Trainer eines Slot-Teams ist — beides ist in der Dienstbörse keine Aushilfe (dort
zählt Trainer zur Stamm-Menge). Ohne diese Regel gäbe die Bilanz dem Kind eines Trainers
eine Aushilfe, während das Board den Trainer ohne Kennzeichen zeigt.

### 3. Ein Aushilfe-Prädikat für Board, Dashboard und Bilanz

Die Frage „Aushilfe ja/nein“ steht an drei Stellen (Board-Gruppe/Eingetragener in SQL,
Dashboard in SQL, Bilanz in Go). Die SQL-Seite bekommt **einen** Fragment-Baustein in
`internal/db` (Foundation, dort liegt schon `TeamDisplayShort`):
`appdb.UserTeamsSQL(kind, userExpr)` mit `kind ∈ {TeamsStamm, TeamsExtended, TeamsChildren}` —
die Teams eines Accounts (selbst + Kinder, aktive Saison; Stamm inkl. Trainer; Children =
Kinder in Stamm- oder erweitertem Kader, die Team-Menge des `'eltern'`-Matchs). Board und
Dashboard nutzen denselben Baustein.
Die Go-Seite (`dutyfairness`) muss dasselbe Ergebnis liefern; ein Test legt eine
Konstellation an und vergleicht `aushilfe` aus dem Board mit der Stufe aus der Bilanz
(Paritäts-Test statt Doku-Versprechen).

Alternative „gemeinsames Package für alles“ verworfen: Board und Dashboard sind SQL,
die Bilanz rechnet in Go über einen Snapshot — ein Zwang auf eine Form würde entweder N+1
im Board oder SQL in der Bilanz erzeugen.

### 4. Board: Flag auf Gruppe und Eingetragenem

- Gruppe: `aushilfe = NOT EXISTS(Team der Gruppe ∈ stamm(viewer)) AND
  EXISTS(Team der Gruppe ∈ extended(viewer))`. Stamm schlägt erweitert.
- Eingetragener: dieselbe Formel mit dem Account des Eingetragenen. Die Eingetragenen
  werden bereits pro Slot nachgeladen; das Flag wird in dieser Nachlade-Query als
  zusätzliche Spalte berechnet — keine zusätzliche Query pro Slot.

Nur `true`/`false` geht raus, keine Team-Namen des Eingetragenen: das Kennzeichen soll
„hilft aus“ sagen, nicht die Kader-Zugehörigkeit Dritter offenlegen.

### 5. Push bleibt beim Stammkader — Teilmenge statt Gleichheit

`eligibleDutyRecipients` bleibt unverändert; nur der Doc-Kommentar und die Spec
`push-duties` werden von „Empfänger = Sichtbarkeit“ auf „Empfänger ⊆ Sichtbarkeit“
umgestellt, mit Nennung der Ausnahme. Das ist dieselbe Figur wie bei den Video-Trainern
(„Empfänger ⊆ Berechtigte“).

### 6. Dashboard: eigener Block statt Aufweichen des Stamm-Blocks

`queryMeineDienste` bleibt unverändert (der Test
`TestDashboard_MeineDienste_ErweiterterKaderZaehltNicht` bleibt grün und behält seine
Aussage). Neu: `queryMeineDiensteAushilfe` mit denselben Bausteinen, Team-Menge
`extended(user) \ stamm(user)`. `mySlots` des Aushilfe-Blocks filtert eigene Zusagen mit
Aushilfe-Prädikat; `mySlots` des Stamm-Blocks schließt sie aus (sonst stünde dieselbe
Zusage zweimal da, wenn beide Teams ein Spiel teilen — dort greift „Stamm schlägt
erweitert“ und sie steht nur oben).

### 7. Rangliste: Aushilfe-Team im Filter, eigener Abschnitt

`TeamsFor` bezieht die `extTeams` der verbundenen Mitglieder ein. Der Block bekommt
`aushilfen []row` aus `Snapshot.aushilfe`, gleich anonymisiert wie `rows`. Ungerankt, nach
`aushilfe_geleistet + aushilfe_vorhersage` absteigend sortiert (nur zur Lesbarkeit).

### 8. Frontend

- `ExtendedBadge` aus `DashboardPage.tsx` wird zu `components/AushilfeBadge.tsx`
  verallgemeinert (Text als Prop; „Erw. Kader“ bleibt am Termin, „Aushilfe“ an Diensten).
  Keine neuen Farben: gleiche `brand-blue`-Tokens.
- `DutySlotList`: Chip an der Gruppenzeile, Badge neben dem Namen des Eingetragenen.
- `DashboardPage`: Aushilfe-Abschnitt unter „Meine Dienste“ und unter der Bilanz, Icon
  `Handshake` (lucide).
- `DienstRanglistePage`: Abschnitt „Aushilfen“ unter der Tabelle bzw. unter den
  Mobile-Cards.

## Risks / Trade-offs

- **Zielgruppe `spieler` und Förderkinder:** der Audience-Match über Vereinsfunktionen
  bleibt unverändert. Ein Förderkind ohne Funktion `spieler` sieht Slots mit
  `audiences=["spieler"]` auch im erweiterten Team nicht. Das ist konsistent mit dem
  Stammkader (gleiche Regel), wird aber auffallen. Kein Teil dieses Changes.
- **Dienst-Bilanz verschiebt sich rückwirkend:** Zuweisungen, die heute über die alte
  Stufe 3 einem Stammteam gutgeschrieben werden, wandern nach dem Deploy in den
  Aushilfe-Abschnitt — Geleistet im Stammteam sinkt. Vorab-Zählung auf Prod (Task 1.1),
  um die Größenordnung zu kennen und ggf. den Vorstand zu informieren.
  Query (read-only, vor dem Deploy von Hand gegen die Prod-DB ausführen):

  ```sql
  WITH act AS (SELECT id FROM seasons WHERE is_active = 1),
  slot_teams AS (
    SELECT ds.id AS slot_id, gt.team_id FROM duty_slots ds
    JOIN game_teams gt ON gt.game_id = ds.game_id
    WHERE ds.season_id = (SELECT id FROM act)
    UNION
    SELECT ds.id, ds.team_id FROM duty_slots ds
    WHERE ds.game_id IS NULL AND ds.team_id IS NOT NULL AND ds.season_id = (SELECT id FROM act)
  ),
  acc AS (  -- Account → Mitglied (eigenes + Kinder)
    SELECT user_id, id AS member_id FROM members WHERE user_id IS NOT NULL
    UNION SELECT parent_user_id, member_id FROM family_links
  ),
  stamm AS (
    SELECT acc.user_id, k.team_id FROM acc
    JOIN kader_members km ON km.member_id = acc.member_id
    JOIN kader k ON k.id = km.kader_id AND k.season_id = (SELECT id FROM act)
    UNION
    SELECT m.user_id, k.team_id FROM members m
    JOIN kader_trainers kt ON kt.member_id = m.id
    JOIN kader k ON k.id = kt.kader_id AND k.season_id = (SELECT id FROM act)
  ),
  ext AS (
    SELECT acc.user_id, k.team_id FROM acc
    JOIN kader_extended_members kem ON kem.member_id = acc.member_id
    JOIN members m ON m.id = kem.member_id AND m.status <> 'ausgetreten'
    JOIN kader k ON k.id = kem.kader_id AND k.season_id = (SELECT id FROM act)
  )
  SELECT COUNT(*) AS aushilfe_zuweisungen, COUNT(DISTINCT da.user_id) AS accounts
  FROM duty_assignments da
  WHERE EXISTS (SELECT 1 FROM slot_teams st JOIN ext e ON e.team_id = st.team_id
                WHERE st.slot_id = da.duty_slot_id AND e.user_id = da.user_id)
    AND NOT EXISTS (SELECT 1 FROM slot_teams st JOIN stamm s ON s.team_id = st.team_id
                    WHERE st.slot_id = da.duty_slot_id AND s.user_id = da.user_id);
  ```
- **Mehr Sichtbarkeit = mehr Claims auf fremden Teams:** Trainer könnten Slots „ihres“
  Teams von Aushilfen belegt finden, die sie erst später bemerken. Mitigiert durch das
  Kennzeichen am Eingetragenen.
- **Query-Last im Board:** zwei zusätzliche `EXISTS` je Gruppe und je Eingetragenem;
  bei <1000 Slots pro Saison vernachlässigbar (VPS 1 GB).
