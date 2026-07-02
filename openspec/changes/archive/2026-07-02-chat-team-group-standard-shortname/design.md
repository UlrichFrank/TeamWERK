## Context

Die kanonische Team-Kurzform (`mA`, `mA1`, `wB2`, `gE`) folgt einer Regel: Geschlecht (`m`/`w`/`g`) + erster Buchstabe der Altersklasse + Team-Nummer **genau dann, wenn in der aktiven Saison mehrere Teams dieselbe Altersklasse+Geschlecht teilen**. Diese Regel ist zweimal implementiert:

- **Server (kanonisch):** `internal/db/team_display_short.go` → `TeamDisplayShort(alias)`, ein SQL-Subausdruck, dessen `COUNT(*)` **saisonweit** über `kader` zählt. Genutzt von Kalender, Termine, Dienstbörse, Mitfahrten.
- **Client (Renderer):** `web/src/lib/teamName.ts` → `buildTeamShortNames(teams)`, hängt die Nummer an, wenn `group_count > 1`. Der `group_count` **muss** vom Server saisonweit geliefert werden.

Der Endpoint `GET /api/chat/team-groups` (`internal/chat/team_groups.go`, `ListTeamGroups`) bricht diese Kette: Er lädt für Nicht-Global-Caller nur die **sichtbaren** Teams (`user_accessible_teams`) und berechnet `groupCount` anschließend **in Go über genau diese Teilmenge** (Zeilen ~115–119). Sieht ein Trainer nur `mB2`, ergibt die Zählung `1` → der Client lässt die Nummer weg → `mB`. Der Schwester-Endpoint `GET /api/teams` (`internal/games/handler.go`, `groupCountSub`) macht es richtig: `group_count` ist dort ein saisonweiter SQL-Subquery — deshalb zeigt das Broadcast-Fenster korrekt `mB2`.

## Goals / Non-Goals

**Goals:**
- Der Chat-Picker zeigt dieselbe kanonische Kurzform wie der Rest der App.
- Eine einzige Quelle der Wahrheit für die Chat-Gruppen-Kurzform: der Server.
- Keine Verhaltensänderung an Sichtbarkeit und `count`.

**Non-Goals:**
- `buildTeamShortNames` global entfernen (bleibt für das Broadcast-Fenster, das bereits einen korrekten saisonweiten `group_count` erhält).
- Änderungen am `/api/teams`-Endpoint oder anderen Kurzform-Konsumenten.
- Schema-/Migrations-Änderungen.

## Decisions

### D1: Server liefert `displayShort` via `TeamDisplayShort` (Approach B)

`ListTeamGroups` selektiert die Kurzform direkt mit dem geteilten Helper und gibt sie als `displayShort` zurück; der Client übernimmt sie unverändert.

```go
// im SELECT der Team-Query (beide Zweige):
SELECT DISTINCT t.id, t.age_class, t.gender, k.team_number,
       COALESCE(` + db.TeamDisplayShort("t") + `, t.name) AS display_short
FROM ...
```

`COALESCE(..., t.name)` schützt den (hier praktisch unmöglichen) NULL-Fall, wenn ein Team kein Kader in der aktiven Saison hätte — konsistent mit der Doku des Helpers. Da `internal/chat` bislang `internal/db` nicht importiert: neuer Import; Domain→Foundation ist laut Architektur-Test erlaubt.

**Alternative (verworfen):** Nur `groupCount` saisonweit korrigieren (denselben `groupCountSub` wie `/api/teams` benutzen) und die Client-Berechnung behalten. Kleiner, aber lässt die Kurzform-Logik an drei Stellen leben und den Chat-Picker weiter selbst rechnen. Approach B zentralisiert stattdessen auf den bereits kanonischen Server-Helper.

### D2: `teamName` und `groupCount` aus der Response entfernen

`teamName` wurde nie befüllt und von keinem Client/Test gelesen; `groupCount` diente ausschließlich der jetzt entfallenden Client-Berechnung. Beide Felder werden aus dem `TeamGroup`-Struct und dem Frontend-Interface entfernt, um keine toten/irreführenden Felder zu hinterlassen.

### D3: `count` und Sichtbarkeit bleiben caller-scoped

Die Mitgliederzahl (`count`, ohne Caller) und die Zeilenauswahl (global vs. `user_accessible_teams`) bleiben unverändert. Nur die **Disambiguierung des Labels** wird saisonweit. Das ist die Invariante, die ein Test festnageln muss: gleicher sichtbarer Datenbestand, aber `displayShort` reflektiert die saisonweite Team-Anzahl.

## Risks / Trade-offs

- **Divergenz der beiden Kurzform-Pfade bleibt teilweise bestehen** (Chat liefert `displayShort`, `/api/teams` liefert `group_count`) → akzeptiert: beide sind jetzt saisonweit korrekt; eine vollständige Vereinheitlichung ist außerhalb des Scopes dieses Bugfixes.
- **Feldentfernung ist API-brechend** für hypothetische externe Konsumenten → Mitigation: Route ist rein intern, nur `ChatPage.tsx` konsumiert sie; kein Test liest `teamName`/`groupCount`.

## Migration Plan

Kein DB-/Datenmigrationsschritt. Backend und Frontend werden zusammen deployt (`make build`), da das Response-Format sich ändert. Rollback = Revert des Commits.

## Open Questions

Keine.
