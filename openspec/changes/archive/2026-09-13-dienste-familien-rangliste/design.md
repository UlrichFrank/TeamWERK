## Context

Siehe proposal.md - Why. Relevante Bestandsteile, auf denen dieser Change aufbaut:

- `internal/dashboard/handler.go:queryDutyAccount` berechnet den Ist-Wert eines
  einzelnen Nutzers heute schon live per `COUNT(*)` über `duty_assignments` —
  bewusst unabhängig von der gespeicherten, bekanntermaßen fehlerhaften Tabelle
  `duty_accounts` (offener Change `dienstkonto-ist-buchung`, dessen `ist`-Wert nur
  beim Löschen eines Termins nachgezogen wird).
- `family_links (parent_user_id, member_id)` ist eine reine Kanten-Tabelle ohne
  eigene Familien-Entität. „Familie bezogen auf ein Kind" ist damit immer:
  `SELECT parent_user_id FROM family_links WHERE member_id = ?`.
- `kader.team_id → teams.id`; `duty_slots` hängen entweder über `game_id` an ein
  Spiel (Team via `game_teams`), direkt über `team_id` (game-lose Slots), oder
  weder noch (generische Slots, z.B. Vereinsfest).
- `web/src/components/TeamFilter.tsx` + `web/src/lib/teamFilter.ts` sind der
  frisch vereinheitlichte Mehrfachauswahl-Baustein (Change
  `team-mehrfachfilter-alle-listen`, archiviert) — Kalender, Termine, Dienste-Board
  und Mitfahrgelegenheiten nutzen ihn bereits.

## Goals / Non-Goals

**Goals:**
- Alle Zahlen (Geleistet, Vorhersage, Gesamtsumme, Fair-Anteil) live berechnen,
  ohne `duty_accounts` zu berühren oder eine neue Tabelle einzuführen.
- Dieselbe Berechnung für drei Sichten wiederverwenden: Dashboard-Kachel,
  anonymisierte Rangliste, Vorstands-Rangliste — ein Codepfad, unterschiedliche
  Sichtbarkeits-Filterung.
- Die Rangliste reagiert auf dasselbe SSE-Event wie das bestehende Dienste-Board
  (`duties`-Broadcast), damit ein frisch übernommener Dienst dort ohne manuellen
  Reload auftaucht.

**Non-Goals:**
- Keine Korrektur von `duty_accounts.ist` — bleibt Gegenstand von
  `dienstkonto-ist-buchung`.
- Kein Umbau der Fulfill/CashSubstitute-Workflows.
- Keine historische Auswertung über vergangene Saisons hinweg — nur die aktive
  Saison.
- Keine Push-Benachrichtigung bei Rückstand („ihr liegt zurück") — reine
  Pull-Ansicht in diesem Change.

## Decisions

**1. Zählung ignoriert `status`, prüft nur Existenz + Datum.**
Alternative verworfen: nur `status='fulfilled'` zählen. `Fulfill` wird laut
`dienstkonto-ist-buchung` nicht zuverlässig geklickt — eine statusbasierte
Zählung hätte dieselbe Verzerrung geerbt, die dort bereits als Bug dokumentiert
ist. Die Existenz-plus-Datum-Regel ist robust gegen diesen Bug, weil sie ihn gar
nicht erst befragt.

**2. Kein Geschwister-Dedup beim Fair-Anteil.**
Einfachere, vorhersehbare Regel (`Gesamtsumme / Spieleranzahl`, kein
Gruppierungsschritt über `family_links`). Nutzerentscheidung explizit: Anteile
werden pro Kind gedacht, nicht pro Haushalt.

**3. Generische Slots proportional zur Spieleranzahl je Kader verteilt, Bruchzahlen erlaubt.**
Alternative verworfen: Gleichverteilung über alle Kader unabhängig von deren
Größe — hätte einem 5-Spieler-Kader denselben Anteil an einem
Vereinsfest-Dienst zugerechnet wie einem 25-Spieler-Kader.

**4. Neues, eigenständiges Package statt Erweiterung von `internal/duties` oder `internal/dashboard`.**
Die Berechnung braucht Daten aus `kader`, `kader_members`, `family_links`,
`games`, `game_teams`, `duty_slots`, `duty_assignments`, `teams` und `seasons` —
mehr Domänen, als eine einzelne bestehende Handler-Struct sauber importieren
sollte (Architektur-Test verbietet Domain-zu-Domain-Importe zwischen
`internal/`-Packages). Wie `internal/dashboard` es bereits vormacht, fragt das
neue Package diese Tabellen direkt per `database/sql` ab, statt Go-Funktionen
anderer Domänen aufzurufen — das bleibt architektonisch zulässig, weil es kein
Package-Import ist. Vorschlag: `internal/dutyfairness`, mit `Handler` im
Standard-Muster (`type Handler struct{ db *sql.DB }`).

**5. Ein Endpoint für alle drei Sichten, serverseitig gefiltert statt zwei Codepfaden.**
`GET /api/duty-fairness/rangliste?team=<id,id>` liefert für Standard-Nutzer nur
Teams, zu denen `family_links` oder eigene `kader_members`-Mitgliedschaft eine
Verbindung herstellt (403 bei anderer `team`-ID), und maskiert fremde Namen zu
Platzierungen. Für `admin`/`vorstand` liefert derselbe Endpoint alle Teams und
alle Namen unmaskiert. Alternative verworfen: zwei getrennte Endpoints
(`/rangliste` und `/admin/rangliste`) — hätte dieselbe Berechnung zweimal
implementiert und divergieren lassen können, genau das Muster, das
`team-mehrfachfilter-alle-listen` gerade erst für vier Listen aufgeräumt hat.

**6. „Eigene Teams"-Scope ist eine neue Query, nicht `GET /teams`.**
`GET /teams` liefert heute ungefiltert alle Teams (siehe Nutzung in
`DutyPage.tsx`, wo das Dienste-Board für jeden Nutzer dieselbe volle Liste im
Filter zeigt — dort gewollt, weil offene Slots aller Teams für alle claimbar
sind). Für die Rangliste braucht es stattdessen:
```sql
SELECT DISTINCT t.id FROM teams t
JOIN kader k ON k.team_id = t.id AND k.season_id = ?
JOIN kader_members km ON km.kader_id = k.id
WHERE km.member_id IN (
    SELECT member_id FROM family_links WHERE parent_user_id = ?
    UNION
    SELECT id FROM members WHERE user_id = ?
)
```
Für `admin`/`vorstand`: alle Teams mit aktivem Kader der Saison, ohne
Einschränkung.

**7. Stabile Sekundärsortierung bei Gleichstand.**
`geleistet + vorhersage` absteigend, bei Gleichstand sekundär nach `member_id`
aufsteigend — jede Zeile bekommt eine eindeutige, reproduzierbare Position statt
geteilter Plätze.

**8. Zurechnung einer Zuweisung zum Kind: Team-Match, geteilt (Entscheidung beim Apply).**
`duty_assignments` trägt nur eine `user_id` — der Dienst liegt entweder auf dem
Account des Kindes (eigener Login oder Proxy-Account, `users.can_login = 0`) oder
auf dem eines Elternteils. Zurechnung in Stufen, die erste nicht-leere gewinnt und
wird gleichmäßig geteilt: (1) eigene Mitglieder des Accounts, deren Team zum Slot
passt; (2) Kinder via `family_links`, deren Team zum Slot passt; (3) eigene
Mitglieder unabhängig vom Team. Ein generischer Slot passt zu jedem Team. Eine
Eltern-Zuweisung ohne passendes Kind zählt für niemanden.
Alternativen verworfen: „voll je Geschwisterkind" (ein Dienst zählte bei zwei
Geschwistern doppelt) und „nur Kind-Account" (die meisten Eltern tragen sich
unter dem eigenen Account ein — die Zahlen wären systematisch zu niedrig). Mit
der Teilung bleibt die Summe einer Familie gleich der Zahl ihrer tatsächlichen
Dienste, während `soll` bewusst je Kind voll zählt (Entscheidung 2).

**9. `internal/dutyfairness` ist Foundation, nicht Domain.**
Präzisiert Entscheidung 4: `internal/dashboard` (Domain) nutzt dieselbe
Berechnung (Task 3.2, kein Duplikat), Domains dürfen sich aber nicht gegenseitig
importieren. Das Package importiert selbst nur Foundation (`auth`, `db`, `timez`)
und steht deshalb — wie `settings`, das ebenfalls einen Handler trägt — in der
Foundation-Liste des Architektur-Tests. Die Berechnung lädt pro Request die
Saison in fünf gebatchten Queries (Teams, Kader-Mitglieder, `family_links`,
Slots samt `game_teams`, Zuweisungen) und rechnet in Go — kein N+1.

## Risks / Trade-offs

- **[Risiko]** Live-Berechnung über alle Kinder eines Kaders pro Request kann bei
  naivem Pro-Kind-Loop (wie `computeSollForElternteil` es heute für eigene Kinder
  tut) zu N+1-Queries führen, jetzt aber über eine ganze Kader-Größe statt nur
  eigene Kinder.
  → **Mitigation**: eine gebatchte Query pro Kader (ein `JOIN` über alle
  `kader_members`, ein `GROUP BY member_id`) statt Pro-Kind-Einzelabfragen.

- **[Risiko]** Anonymisierung ist bei sehr kleinen Kadern (2-3 Kinder) faktisch
  durchschaubar — wer die anderen Familien der eigenen (kleinen) Jugendmannschaft
  kennt, kann anhand der Balkenlänge oft erraten, wer hinter „Platz 2" steckt.
  → **Mitigation**: keine technische; bewusste Produktentscheidung des Vereins.
  Festgehalten als Grenze des Datenschutz-Anspruchs, nicht als offener Punkt.

- **[Risiko]** `/api/dashboard` ändert die Form von `dutyAccount` (Objekt → Liste)
  — **BREAKING** für jeden Client, der die alte Form erwartet.
  → **Mitigation**: Backend und Frontend werden gemeinsam über `make deploy`
  ausgerollt (embed.FS-Modell, keine unabhängigen Client-Versionen im Feld);
  kein Versionierungsbedarf.

## Migration Plan

Keine Datenbank-Migration, keine neue Tabelle. Deploy ist ein regulärer
`make deploy`-Lauf (Build + Restart). Rollback: vorherige Binary erneut
deployen — da kein Schema betroffen ist, ist das gefahrlos in beide Richtungen.

## Open Questions

- ~~Soll die Rangliste-Seite per SSE live aktualisieren?~~ Entschieden: ja, sie
  abonniert `duties` (neue Zuweisungen) und `games` (neue/gelöschte Termine
  ändern die Gesamtsumme).
- Braucht es einen Mindest-Kadergröße-Schwellwert, unter dem die Anonymisierung
  ausgesetzt oder anders kommuniziert wird? Kann nach erster Nutzung im echten
  Vereinsbetrieb entschieden werden.
