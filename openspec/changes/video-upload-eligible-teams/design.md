## Context

Siehe proposal.md — Why. Stand heute:

- `GET /api/videos/upload-eligible-games` (`internal/videos/eligible_games.go`) liefert nur `{game_ids: [...]}`; die Vereinigung Rollen-Pfad ∪ Dienst-Pfad ist dort schon korrekt.
- Das Tool (`tools/video-encoder/ui.go`) ruft nach dem Login `ActiveSeasonID`, `Teams()` (= `GET /api/teams`, gefiltert auf `is_active`) und `EligibleGameIDs()`; bei Mannschaftswahl `Games()` (= `GET /api/games?season_id=…`, clientseitig auf Team, Datum ≤ heute und `eligibleGames` gefiltert) sowie `VideosByGame()` (= `GET /api/videos?team_id=…`, reine Zugabe für die Ersetzen-Auswahl).
- `GET /api/teams` liefert seit `teamfilter-trainer-elternteil` die Vereinigung aus `user_accessible_teams` (Trainer/Spieler/Eltern, Stamm- und erweiterter Kader); `GET /api/games` filtert über `auth.GameVisibilityClause` mit denselben Kader-Wegen. Beide kennen den Dienst-Pfad nicht.
- `CreateUpload` prüft `CanUploadToTeam(team_id)` und fällt auf `CanUploadForGameViaDuty(game_id)` zurück, ohne `team_id` gegen `game_teams` zu prüfen.
- Die Fyne-Oberfläche ist nicht unit-testbar; der CI-Job `video-encoder` testet `tools/video-encoder/internal/...`.

## Goals / Non-Goals

**Goals:**

- Ein Endpoint beantwortet vollständig, was der Nutzer wo hochladen darf; das Tool braucht keine zweite Quelle, die eine andere Frage beantwortet.
- Der Server bleibt die einzige Autorität; die Tool-Liste ist nie großzügiger als `CreateUpload`.
- Rückwärtskompatibel: ausgerollte Tool-Versionen lesen weiter `game_ids`.
- Auswahl-Logik des Tools unit-getestet.

**Non-Goals:**

- Keine Änderung an `GameVisibilityClause`, `GET /api/teams` oder der Termin-Sichtbarkeit: ob ein Dienst-Inhaber den Termin in `/termine` sehen soll, ist eine Kalender-Frage (Dienste sind über die Dienstbörse sichtbar).
- Keine Änderung an der Video-Sichtbarkeit (`CanViewVideo`/`visibilityFilter`/`pushRecipients`).
- Kein Web-Upload (entfallen seit `video-offline-encoding-tool`).
- Keine Ersetzen-Auswahl für Dienst-Inhaber: `VideosByGame` folgt `visibilityFilter`, und Löschen verlangt `CanManageTeamVideos` — für Dienst-only-Teams bleibt die Auswahl wie bisher verborgen, das ist konsistent.

## Decisions

**D1 — Ein Endpoint mit drei Mengen statt Nachbesserung von `/api/teams` und `/api/games`.**
Alternativen: (a) `GET /api/teams` um „Teams meiner Video-Dienste" erweitern und `GameVisibilityClause` um Dienst-Zuweisungen — das kippt zwei allgemeine Sichtbarkeitsregeln für einen Upload-Sonderfall und macht Dienst-Inhabern Termine sichtbar, die sie fachlich nichts angehen (Kalender, Mitfahrgelegenheiten, Filter); (b) Tool ruft `GET /api/games/{id}` je eligible Spiel — scheitert an derselben Sichtbarkeitsklausel. Gewählt: der bestehende Berechtigungs-Endpoint liefert `teams` und `games` mit. Response:

```json
{
  "game_ids": [12, 14],
  "teams": [{"id": 3, "name": "mB2", "upload_without_game": true},
            {"id": 5, "name": "mA2", "upload_without_game": false}],
  "games":  [{"id": 12, "date": "2026-09-12T00:00:00Z", "opponent": "TSV X",
              "event_type": "heim", "season_id": 4, "team_ids": [3]},
             {"id": 14, "date": "2026-09-13T00:00:00Z", "opponent": "HSG Y",
              "event_type": "auswaerts", "season_id": 4, "team_ids": [5]}]
}
```

`game_ids` bleibt unverändert (Kompatibilität). `date` bleibt der rohe SQLite-Wert; das Tool truncated wie bisher (`dateOnly`, Gotcha „SQLite DATE-Felder").

**D2 — `teams` = Rollen-Teams (Kader in aktiver Saison) ∪ Teams der Dienst-Spiele; `upload_without_game` = Rollen-Pfad.**
Rollen-Teams folgen genau `CanUploadToTeam`: admin/vorstand/sportliche_leitung → alle Teams mit Kader in der aktiven Saison (Entsprechung zur bisherigen `/api/teams`-Liste, kein Team ohne Kader), trainer → `kader_trainers` der aktiven Saison. Für Dienst-Teams ist das Kennzeichen `false`, außer das Team ist zugleich Rollen-Team (dann gewinnt `true`). Das Kennzeichen ersetzt die Regel, die das Tool sonst raten müsste: ohne `game_id` autorisiert `CreateUpload` nur über `CanUploadToTeam`. Alternative „Tool leitet es aus den Claims ab" verworfen — das Tool kennt die Claims nicht und müsste die Rollenlogik duplizieren.

**D3 — Rollen-Spiele bleiben saisonübergreifend, das Tool filtert auf aktive Saison und Datum ≤ heute.**
Der Endpoint ändert die Spielmenge nicht (heute: admin alle Spiele, trainer alle Spiele seiner Teams). Das Tool filtert wie bisher clientseitig auf `season_id == aktive Saison` und Datum ≤ heute — dafür trägt jeder `games`-Datensatz `season_id`. Ein serverseitiger Saisonfilter wäre eine Vertragsänderung für `game_ids`, die ältere Tools träfe.

**D4 — Auswahl-Logik in `tools/video-encoder/internal/pick`.**
Reine Funktionen über der Endpoint-Antwort: `Teams(resp) []Team` (Reihenfolge wie Server), `Games(resp, teamID, seasonID, today) []Game` (Filter + Sortierung jüngstes zuerst) und `AllowFreeTitle(resp, teamID) bool`. `ui.go` ruft sie nur noch auf und baut Labels. Damit sind alle vier Spec-Szenarien des Tools ohne Fyne testbar. `client.Teams()`/`client.Games()` und der `Team`-/`Game`-Typ des alten Vertrags entfallen (kein zweiter Pfad, der wieder auseinanderlaufen kann); `client.EligibleGameIDs` wird durch `client.Eligible(ctx) (Eligible, error)` ersetzt.

**D5 — Härtung `team_id ∈ game_teams` nur im Dienst-Pfad, HTTP 400.**
Für den Rollen-Pfad bleibt `team_id` frei (Vorstand darf ein Video bewusst einer anderen Mannschaft zuordnen, z. B. bei Doppelspieltagen). Im Dienst-Pfad ist die Berechtigung an das Spiel gebunden, also muss auch das Ziel-Team eines seiner Teams sein — sonst wird das Video über `visibilityFilter` bei einer unbeteiligten Mannschaft sichtbar. 400 statt 403, weil es eine Inkonsistenz der Anfrage ist (Spiel und Team passen nicht zusammen), keine fehlende Berechtigung; die Prüfung läuft **nach** der Berechtigungsprüfung, damit die Objektrechte-Matrix keinen Befund „Validierung vor Autorisierung" bekommt.

**D6 — „Freier Titel" für Dienst-only-Teams gar nicht anbieten statt beim Start ablehnen.**
Die Option erst im Klick mit „nicht erlaubt" zu quittieren, wäre eine Sackgasse. Für Dienst-only-Teams enthält das Spiel-Dropdown nur die Spiele; ist es leer (nur Zukunftsspiele), zeigt der Status den Grund. `onGameChanged` und `start()` bleiben unverändert, weil `gameID == 0` dort weiterhin „kein Spiel" heißt.

## Risks / Trade-offs

- [Response wächst mit Spielanzahl für admin/vorstand: alle Spiele aller Saisons] → pro Zeile ~100 Byte, bei einigen tausend Spielen wenige hundert KB, einmal pro Login; kein Hot-Path. Kein Paging, weil das Tool die volle Menge zum Filtern braucht; ein späterer `?season_id=`-Parameter ist additiv möglich.
- [Zwei Wahrheiten für „Rollen-Teams": `/api/teams` (Kader-Zugehörigkeit) und `teams` hier (Upload-Recht)] → bewusst verschieden, weil sie verschiedene Fragen beantworten; der Handler-Kommentar benennt den Unterschied, und der Test „Spieler ohne Funktion → leere Listen" hält fest, dass Kader-Zugehörigkeit allein hier nichts freischaltet.
- [Alte Tool-Version bei neuem Server] → liest nur `game_ids`, Verhalten unverändert (inkl. des bekannten Lochs). [Neue Tool-Version bei altem Server] → `teams`/`games` fehlen, Mannschaftsliste leer, Hinweistext greift; ein Versionsgate ist dafür nicht nötig, weil das Release erst nach dem Server-Deploy gebaut wird (siehe Migration Plan).
- [400-Härtung könnte einen legitimen Dienst-Upload treffen, wenn `game_teams` unvollständig ist] → jedes Spiel trägt mindestens ein Team (`CreateGame`/H4A-Import schreiben `game_teams`); das Tool sendet genau das gewählte Team des Spiels.

## Migration Plan

1. Server deployen (`make deploy`, keine Migration). Alte Tools laufen weiter.
2. Release mit dem Tool-Build erzeugen (Merge nach `main` → `release.yml` hängt die Binaries an). Filmende laden die neue Version über `/videos`.
3. Rollback: `make deploy-rollback` ist gefahrlos, die neuen Felder sind additiv. Ein bereits verteiltes neues Tool sähe gegen den alten Server eine leere Mannschaftsliste — dann Release vorziehen statt Server zurückrollen.
