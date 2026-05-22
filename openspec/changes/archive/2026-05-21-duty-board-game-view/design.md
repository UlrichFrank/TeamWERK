## Context

`duty_slots` hat ein optionales `game_id`-Feld. Slots mit `game_id` gehören zu einem Heimspiel; Slots ohne gehören zu keinem konkreten Spiel. Die Verbindung User → Team läuft über `members.user_id` (Spieler) und `family_links` (Elternteil → Kind → `team_memberships`). Heute ignoriert der Board-Handler beides.

## Goals / Non-Goals

**Goals:**
- Team-gefilterte, spielgruppierte Ansicht in einem einzigen API-Call
- `claimed_by_me` pro Slot (kein separater Endpunkt nötig)
- Austragen per DELETE
- Vergangene Gruppen clientseitig steuerbar

**Non-Goals:**
- Keine Pagination (Saison hat überschaubar viele Heimspiele)
- Kein Eintragen für andere User (Claim bleibt immer für den eingeloggten User)
- Kein Ändern der Claim-Logik selbst

## Decisions

### 1. Gruppierung im Backend, nicht im Frontend

Das Backend liefert `[]Group` mit `slots []Slot` darin. Alternativ könnte das Frontend eine flache Liste groupBy aufteilen — aber dann müsste der Frontend-Code Spielmetadaten (opponent, team_name) mitliefern. Eine strukturierte Antwort ist klarer und einfacher zu rendern.

### 2. "Meine Teams"-Query

```sql
SELECT DISTINCT tm.team_id
FROM team_memberships tm
JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
WHERE tm.member_id IN (
    SELECT id FROM members WHERE user_id = ?          -- ich als Spieler
    UNION
    SELECT member_id FROM family_links WHERE parent_user_id = ?  -- meine Kinder
)
```

### 3. Response-Shape

```json
[
  {
    "game_id": 5,
    "date": "2026-06-07",
    "event_time": "11:00",
    "opponent": "TSG Söflingen",
    "team_name": "A-Jugend",
    "past": false,
    "slots": [
      {
        "id": 12,
        "duty_type": "Kassendienst",
        "event_time": "10:30",
        "slots_total": 2,
        "vacancies": 1,
        "claimed_by_me": true,
        "role_desc": ""
      }
    ]
  },
  {
    "game_id": null,
    "date": null,
    "event_time": null,
    "opponent": null,
    "team_name": "A-Jugend",
    "label": "Sonstige Dienste",
    "past": false,
    "slots": [...]
  }
]
```

`past` = `date < today` (bzw. bei game_id null: `event_date < today`).

### 4. Unclaim-Endpunkt

`DELETE /api/duty-board/{slotId}/claim` — löscht `duty_assignments WHERE duty_slot_id=? AND user_id=?`, dekrementiert `slots_filled`, aktualisiert `duty_accounts`. Gibt 404 wenn keine Assignment existiert.

### 5. Vergangene Spieltage: clientseitig filtern

Das Backend liefert immer alle Gruppen (auch vergangene). Das Frontend blendet `past=true`-Gruppen standardmäßig aus. Ein State `showPast` schaltet um. Kein separater API-Parameter nötig — die Datenmenge ist klein.

### 6. Slot-Zustände im Frontend

```
claimed_by_me = true  →  [Austragen]-Button (immer, auch wenn past)
                          aber nur wenn !past: Button aktiv
vacancies > 0          →  [Eintragen]-Button
vacancies == 0         →  "Besetzt" (kein Button)
```

## Risks / Trade-offs

- [Leere Dienstbörse] Wenn User keiner Mannschaft zugeordnet ist, gibt das Backend eine leere Liste zurück. Frontend zeigt Hinweistext.
- [Unclaim bei bereits erfülltem Dienst] Wenn `status='fulfilled'`, sollte Austragen blockiert werden (400). Der Handler prüft das.
