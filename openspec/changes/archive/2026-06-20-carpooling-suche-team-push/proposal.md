## Why

Heute löst eine neue Mitfahr-Suche nur einen Push an die wenigen User aus, die bereits ein `biete` zum selben Spiel angelegt haben (`notifyOpposite` in `internal/carpooling/handler.go`). In der Praxis fahren aber oft Eltern, die noch gar kein Angebot eingestellt haben. Damit eine offene Suche überhaupt eine Chance hat, einen Fahrer zu finden, müssen die Personen erreicht werden, die typischerweise zum Spiel fahren — Eltern der Kaderspieler (regulär + erweitert) und Trainer des Kaders.

Um Spam zu vermeiden, wird der Team-Push auf das **nächste anstehende Spiel des Teams** begrenzt. Suchen für weiter entfernte Spiele lösen weiterhin nur den existierenden `notifyOpposite`-Push aus.

## What Changes

- `POST /api/mitfahrgelegenheiten` mit `typ='suche'` und `isNewEntry=true` (Insert, kein Update) löst zusätzlich zum bestehenden `notifyOpposite` einen **Team-Push** aus, sofern das betroffene Spiel das **nächste anstehende Spiel seines Teams** ist.
- Empfängerkreis pro betroffenem Team:
  - Eltern (`family_links.parent_user_id`) der Mitglieder in `kader_members` ∪ `kader_extended_members`
  - Trainer (`kader_trainers`) — aufgelöst zu deren `user_id`
  - minus der Suche-Steller selbst
- Spiele über `game_teams` (m:n): pro assoziiertem Team einzeln prüfen, ob es das nächste anstehende Spiel **dieses** Teams ist. Empfänger der qualifizierenden Teams werden vereinigt.
- Wenn für ein Team keine `kader`-Zeile zur passenden `season_id` existiert: **stiller Skip** (kein Fallback auf alle Team-Member).
- `UPDATE` einer bestehenden Suche triggert **nicht** (verhindert Wiederholungs-Spam beim Bearbeiten).
- Pref-Kategorie bleibt `"carpooling"` (kein neuer Switch).
- Versand asynchron in Goroutine, wie bisher.
- `notifyOpposite` für `typ='suche'` bleibt unverändert (keine Dedup-Logik); Empfänger können in seltenen Fällen zwei Pushes erhalten — bewusst akzeptiert.

## Capabilities

### Modified Capabilities

- `carpooling-notifications`: Notification bei neuer Suchanfrage wird um einen Team-Push erweitert, der das gefährdete Erreichen potenzieller Fahrer adressiert.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheInsert_NextGame_TeamPushFanOut` | Suche zum nächsten Spiel des Teams → Push an Eltern (regulär+erweitert) **und** Trainer des Kaders, **ohne** Steller. |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheInsert_NotNextGame_NoTeamPush` | Suche zu einem späteren Spiel des Teams → **kein** Team-Push (nur bestehender `notifyOpposite`). |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheUpdate_NoTeamPush` | Update einer existierenden Suche (auch zum nächsten Spiel) → **kein** Team-Push. |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheInsert_NoKaderSilent` | Nächstes Spiel hat keinen Kader → **kein** Team-Push, keine Fehlerantwort. |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheInsert_MultiTeamGame` | Spiel in `game_teams` mit zwei Teams; nur Team A's Kader-Empfänger erhalten Push, wenn das Spiel nur für A das nächste ist. |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheInsert_BieteTyp_NoTeamPush` | `typ='biete'` löst **keinen** Team-Push aus (Verhalten unverändert). |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheInsert_SelfExcluded` | Der Suche-Steller bekommt keinen Push, auch wenn er als Trainer oder Elternteil im Empfängerkreis wäre. |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_SucheInsert_PrefRespected` | User mit `notification_preferences.push_enabled=0` für `"carpooling"` erhält keinen Team-Push. |

**Garantierte Invariante:** Ein Team-Push wird genau dann ausgelöst, wenn (a) `typ='suche'`, (b) der Eintrag neu ist (Insert), und (c) das Spiel für mindestens ein assoziiertes Team das nächste Spiel mit `date >= date('now')` ist. Empfänger sind ausschließlich Eltern der Kaderspieler (regulär + erweitert) und Trainer der qualifizierenden Kader-Zeile(n), abzüglich des Stellers.

## Impact

- **Datei:** `internal/carpooling/handler.go` — `Upsert` um Team-Push-Fan-out nach `notifyOpposite` ergänzen; neue private Helfer `nextGameTeams(gameID)` und `kaderRecipients(teamIDs, seasonID, excludeUserID)`.
- **Datei:** `internal/carpooling/handler_test.go` — neue Tests (siehe Tabelle).
- **Lese-Pattern (Eltern via `family_links` über regulär+erweitert)** existiert in `internal/teams/handler.go:168–190` — daran orientieren.
- **Kein** Schema-Change, **keine** neue Route, **kein** Frontend-Change (existierende Push-Subscription reicht).
- **Kategorie `"carpooling"`** unverändert; bestehende User-Prefs (`push_enabled`) gelten automatisch.
- **Doppel-Push** in Randfällen (z.B. ein Elternteil hat selbst ein `biete` zum gleichen Spiel) bewusst akzeptiert — siehe `design.md`.
