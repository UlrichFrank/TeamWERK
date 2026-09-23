# Design — dashboard-offene-gesuche-cross-team

## Kolokations-Regel

Ausgangspunkt sind die nächsten ≤3 künftigen Spiele der eigenen Teams („Anker"). Ein Fremdspiel ist relevant, wenn es mit einem Anker `date` UND `venue_id` teilt und `venue_id IS NOT NULL`.

```sql
WITH my_games AS (
  SELECT g.id, g.date, g.venue_id
  FROM games g JOIN game_teams gt ON gt.game_id = g.id
  WHERE gt.team_id IN ( <user_accessible_teams> )
    AND g.season_id = ? AND DATE(g.date) >= DATE('now')
  ORDER BY g.date, g.time LIMIT 3
),
relevant AS (
  SELECT id FROM my_games
  UNION
  SELECT g2.id FROM games g2
    JOIN my_games mg ON g2.date = mg.date AND g2.venue_id = mg.venue_id
   WHERE g2.venue_id IS NOT NULL
)
SELECT s.id, s.game_id, g.date, g.venue_id, v.name, g.opponent,
       us.first_name||' '||us.last_name, s.plaetze, s.treffpunkt
FROM mitfahrgelegenheiten s
JOIN games g  ON g.id = s.game_id
LEFT JOIN venues v ON v.id = g.venue_id
JOIN users us ON us.id = s.user_id
WHERE s.typ = 'suche'
  AND s.game_id IN (SELECT id FROM relevant)
  AND NOT EXISTS (SELECT 1 FROM mitfahrt_paarungen p
                  WHERE p.suche_id = s.id AND p.status = 'confirmed')
ORDER BY g.date, v.name;
```

## Gruppierungsschlüssel

- `venue_id` vorhanden → Gruppe `(date, venue_id)`, Label = `venue.name`. Pool über alle Spiele dort/dann, teamübergreifend.
- `venue_id IS NULL` → Gruppe `(game_id)`, Label = `opponent`. Kein Cross-Team-Match möglich (Fallback wie Teil 1).

Da ein Pool mehrere Spiele/Teams umfasst, trägt jeder Gesuch-Eintrag seinen Kontext (Gegner/Team), damit der Anlass erkennbar bleibt.

## Bewusste Entscheidungen

- **Tag + Ort, nicht Uhrzeit.** Übereinstimmung auf `date` (nicht `time`) — ein gemeinsames Ziel reicht als Mitfahr-Anlass, auch bei unterschiedlichen Anstoßzeiten in derselben Halle.
- **Datensichtbarkeit.** Heute filtert der `/mitfahrgelegenheiten`-List-Handler auf die eigenen Teams; Gesuche fremder Teams sieht man nirgends. Dieses Feature ist die erste Stelle, an der Namen fremder Teammitglieder erscheinen — eng begrenzt auf Tag+Venue-Übereinstimmung. Da es club-interne Koordination ist und genau dem Wunsch entspricht, bewusst akzeptiert.
- **Venue-Pflege als Voraussetzung.** Ohne `venue_id` kein Cross-Team-Match. Kein Bug, sondern Datenqualitäts-Abhängigkeit; Fallback ist die Teil-1-Anzeige pro Spiel.

## Abgrenzung

`carpoolingConfirmed` / `queryCarpoolingConfirmed` bleiben unverändert (auswärts-only, pro Spiel). Daraus folgt bewusst eine Asymmetrie: Ein Spiel kann offene Gesuche im Pool zeigen, ohne in der Confirmed-Liste zu stehen.
