## Why

Eine Mitfahrt-Paarung verknüpft im Datenmodell zwingend **beide** Seiten als echte Einträge (`mitfahrt_paarungen.biete_id` und `.suche_id` sind `NOT NULL`). Das Frontend versteckt darum die Buttons »Anfragen«/»Einladen«, solange der Nutzer keinen eigenen Spiegel-Eintrag auf seiner Seite hat (`canRequestAsBiete` setzt `mySucheIds.length > 0`, `canInviteAsSuche` setzt `myBieteIds.length > 0`). Wer auf ein fremdes Angebot reagieren will, muss also erst ein eigenes „Gegenangebot" anlegen — eine vermeidbare Reibung, die das einfache „ich will da mitfahren" zur Zwei-Schritt-Aktion macht.

## What Changes

- `POST /api/mitfahrt-paarungen` (`RequestPairing`) akzeptiert künftig **einseitige** Request-Bodies:
  - `{ "bieteId": N, "forUserId"?: M, "plaetze"?: P }` — ich (oder mein Kind) will bei diesem Angebot mitfahren; der Suche-Spiegel-Eintrag wird automatisch angelegt.
  - `{ "sucheId": N, "plaetze"?: P }` — ich biete diesem Gesuch einen Platz an; der Biete-Spiegel-Eintrag wird automatisch angelegt (immer für den eingeloggten Nutzer).
  - `{ "bieteId": N, "sucheId": M }` (beide) bleibt **abwärtskompatibel** das heutige Verhalten.
- Das Anlegen des fehlenden Spiegel-Eintrags und das Erstellen der Paarung passieren **atomar in einer Transaktion**. Schlägt der Kapazitäts-Check fehl (409), wird nichts persistiert — kein Phantom-Eintrag auf dem Board.
- **get-or-create**: Besitzt der Nutzer bereits einen passenden Eintrag auf seiner Seite für dieses Spiel, wird dieser wiederverwendet statt ein zweiter angelegt (für Biete durch den Unique-Index `(game_id,user_id)` ohnehin nötig).
- Der **Roundtrip bleibt unverändert**: Ergebnis ist eine `pending`-Paarung, die Gegenseite bestätigt wie bisher.
- Frontend: »Mitfahren«/»Platz anbieten« erscheinen auch **ohne eigenen Eintrag**. Der Klick öffnet einen Mini-Dialog, der nur Plätze abfragt — und auf der Mitfahren/Suche-Seite zusätzlich „für wen" (ich / Kind A / Kind B), falls der Nutzer Elternteil ist. Auf der Platz-anbieten/Biete-Seite gibt es keine Für-wen-Auswahl (immer der eingeloggte Nutzer).

### Bewusst nicht im Scope

- **Sofort-bestätigte Paarung** (Wegfall des Roundtrips) — der Bestätigungsschritt der Gegenseite bleibt erhalten.
- **Elternteil bietet Platz für ein Kind an** — ein Kind als Fahrer ergibt fachlich selten Sinn; die Biete-Seite des One-Click bleibt auf den eingeloggten Nutzer beschränkt. Wer das doch braucht, nutzt weiterhin den Formular-Weg (`POST /api/mitfahrgelegenheiten` mit `forUserId`).
- Keine Schema-Änderung an `mitfahrt_paarungen` oder `mitfahrgelegenheiten` (keine Migration).

## Capabilities

### New Capabilities

(keine)

### Modified Capabilities

- `mitfahrt-paarungen`: Die Requirements »Paarungsanfrage stellen (Sucher initiiert)« und »Paarungsanfrage stellen (Bieter initiiert)« werden erweitert, sodass die anfragende Seite ihren Eintrag nicht vorab besitzen muss — er wird beim einseitigen Request implizit (get-or-create) angelegt.

## Impact

- **Backend:** `internal/carpooling/paarungen_handler.go` (`RequestPairing`) — Body-Parsing, einseitiger Pfad, transaktionales get-or-create des Spiegel-Eintrags, Wiederverwendung der bestehenden Kapazitäts-/Berechtigungs-Logik. Tests in `internal/carpooling/handler_test.go`.
- **Frontend:** `web/src/pages/MitfahrgelegenheitenPage.tsx` — Gate-Bedingungen `canRequestAsBiete`/`canInviteAsSuche` entkoppeln vom Vorhandensein eigener Einträge; neuer Mini-Dialog (für wen + Plätze); `onRequest` ruft den einseitigen Endpoint.
- **Keine** DB-Migration, **keine** neuen Routen, **keine** neuen Abhängigkeiten.
- SSE: `RequestPairing` broadcastet bereits `mitfahrgelegenheiten` — bleibt.
