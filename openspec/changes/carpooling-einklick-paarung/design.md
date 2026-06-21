## Context

Mitfahrt-Paarungen (`mitfahrt_paarungen`) verknüpfen zwingend einen Biete- mit einem Suche-Eintrag (`biete_id`/`suche_id` beide `NOT NULL`, `UNIQUE(biete_id, suche_id)`). Die Erstellung läuft heute über `RequestPairing` (`POST /api/mitfahrt-paarungen`), das einen Body `{bieteId, sucheId}` mit **beiden** IDs erwartet und prüft, dass der aufrufende Nutzer (oder eines seiner Kinder) eine der beiden Seiten besitzt. Das Frontend blendet die Aktionsbuttons nur ein, wenn der Nutzer bereits einen Eintrag auf seiner Seite hat (`mySucheIds`/`myBieteIds`). Wer auf ein fremdes Angebot reagieren will, muss daher erst ein „Gegenangebot" anlegen.

Relevante Constraints:
- `mitfahrgelegenheiten`: Unique-Index `idx_mitfahr_biete_unique ON (game_id, user_id) WHERE typ = 'biete'` — pro Nutzer/Spiel höchstens ein Biete-Eintrag; für `suche` kein Unique-Constraint.
- `mitfahrgelegenheiten.plaetze` ist nullable (`suche`: Personenzahl, Default-Annahme 1; `biete`: freie Plätze, NULL = unbegrenzt).
- SQLite ohne `RETURNING` → `LastInsertId()`.
- Bestehende Kapazitäts- und Berechtigungslogik in `RequestPairing` soll wiederverwendet, nicht dupliziert werden.

## Goals / Non-Goals

**Goals:**
- Reaktion auf einen fremden Biete-/Suche-Eintrag mit einem Klick, ohne vorher manuell einen Spiegel-Eintrag anzulegen.
- Atomares Anlegen (Spiegel-Eintrag + Paarung) ohne Phantom-Einträge bei Fehlern.
- Volle Abwärtskompatibilität des bestehenden `{bieteId, sucheId}`-Pfads.

**Non-Goals:**
- Wegfall des Bestätigungs-Roundtrips (Ergebnis bleibt `pending`).
- Elternteil bietet Platz für ein Kind im einseitigen Biete-Pfad an.
- Schema-Änderungen / Migration.

## Decisions

### Entscheidung 1: Atomar im Backend statt zwei Frontend-Calls
Der Spiegel-Eintrag wird im Backend innerhalb einer DB-Transaktion zusammen mit der Paarung angelegt. **Alternative** (zwei Frontend-Calls: erst `POST /mitfahrgelegenheiten`, dann `POST /mitfahrt-paarungen`) wurde verworfen: Schlägt der zweite Call fehl (z.B. 409 Kapazität), bliebe ein ungewolltes „Geister-Gesuch" auf dem Board stehen; ein Rollback per zusätzlichem DELETE wäre racy. Die Transaktion macht den Vorgang sauber atomar.

### Entscheidung 2: `RequestPairing` erweitern statt neue Route
`RequestPairing` akzeptiert weiterhin `{bieteId, sucheId}` (beide gesetzt) und zusätzlich einseitige Bodies:
- `{bieteId, forUserId?, plaetze?}` → `initiiert_von='suche'`, Suche-Spiegel für `forUserId` (Default: eingeloggter Nutzer).
- `{sucheId, plaetze?}` → `initiiert_von='biete'`, Biete-Spiegel für den eingeloggten Nutzer.

Genau eine der beiden IDs gesetzt ⇒ einseitiger Pfad; beide gesetzt ⇒ heutiger Pfad; keine ⇒ 400. **Alternative** (separate Route `/api/mitfahrt-paarungen/quick`) verworfen: dieselbe Kapazitäts-/Berechtigungslogik, künstliche Aufspaltung, mehr Router-Fläche.

### Entscheidung 3: get-or-create des Spiegel-Eintrags
Vor dem Anlegen wird geprüft, ob die Zielperson für dieses Spiel bereits einen passenden Eintrag auf der Spiegel-Seite hat:
- **Biete:** durch Unique-Index ohnehin höchstens einer → vorhandenen wiederverwenden.
- **Suche:** vorhandenen Suche-Eintrag *ohne aktive Paarung* wiederverwenden, sonst neu anlegen (verhindert Duplikat-Gesuche auf dem Board).

Nach Auflösung des Spiegel-Eintrags läuft der bestehende Code-Pfad (Berechtigung, Kapazität, Insert der Paarung) unverändert weiter.

### Entscheidung 4: Reihenfolge — Berechtigung & Kapazität vor Insert
Innerhalb der Transaktion: (1) Spiegel-Eintrag-ID ermitteln/anlegen, (2) Berechtigung (`forUserId`-Bezug bzw. Ownership) prüfen, (3) Kapazitäts-Check, (4) Paarung-Insert. Bei (2)/(3)-Fehler → `tx.Rollback()`, kein Eintrag persistiert. Berechtigung des `forUserId`-Bezugs wird vor dem Insert des Spiegel-Eintrags geprüft, damit ein fremder `forUserId` gar keinen Eintrag erzeugt.

### Entscheidung 5: Frontend — Mini-Dialog statt sofortigem POST
Die Gate-Bedingungen `canRequestAsBiete`/`canInviteAsSuche` werden vom Vorhandensein eigener Einträge entkoppelt (nur noch: fremder Eintrag, freie Kapazität, keine bestehende aktive Paarung mit mir). Der Klick öffnet einen Mini-Dialog:
- **Mitfahren (Suche-Seite):** Plätze + (falls Elternteil) Auswahl ich / Kind A / Kind B → `forUserId`.
- **Platz anbieten (Biete-Seite):** nur Plätze (kein `forUserId`).

Hat der Nutzer bereits einen passenden Eintrag, bleibt der heutige Direkt-Pfad möglich; der Dialog kann den vorhandenen Eintrag vorbelegen.

## Risks / Trade-offs

- **Doppelte Suche-Einträge** bei wiederholtem One-Click auf verschiedene Angebote → Mitigation: get-or-create verwendet vorhandenes Gesuch ohne aktive Paarung wieder; ein Nutzer kann ohnehin nur eine aktive (pending/confirmed) Paarung pro Gesuch haben.
- **Transaktion + Kapazitäts-Race** (zwei gleichzeitige One-Clicks auf dasselbe knappe Angebot) → Mitigation: Kapazitäts-Check und Insert in derselben Transaktion; SQLite serialisiert Schreibzugriffe (WAL, ein Writer). Bei Verlust gewinnt der erste, der zweite erhält 409.
- **Default-Plätze unscharf** (Mitfahrer bringt mehr Personen mit als angenommen) → Mitigation: Mini-Dialog fragt Plätze explizit ab; Default 1 nur als Fallback.

## Migration Plan

Keine DB-Migration. Rein additive Backend-Logik + Frontend-UI. Deploy via `make deploy`. Rollback: vorheriges Binary/Frontend — neue einseitige Requests gäben dann 400 zurück, bestehende Paarungen bleiben unberührt.

## Open Questions

(keine offen — Für-wen-Auswahl auf Biete-Seite bewusst ausgeschlossen, siehe Non-Goals)
