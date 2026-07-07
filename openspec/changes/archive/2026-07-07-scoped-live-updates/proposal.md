## Why

Der Domänen-SSE-Stream ist heute ein **globaler Fan-out**: `EventHub.Broadcast(event)` (`internal/hub/hub.go:32`) sendet jedes Event an **alle** verbundenen Clients, und der `/api/events`-Handler abonniert über das globale `Subscribe()` (`internal/hub/handler.go:30`). Es gibt 88 `Broadcast`-Aufrufe mit bloßen Topic-Strings — `"members"` allein an 23 Stellen, `"games"`/`"trainings"` je 11×, `"kader"` 7×.

Konsequenz: **eine** Mutation eines einzelnen Nutzers erzeugt bei jedem eingeloggten Client einen vollen Refetch — unabhängig davon, ob dieser Client die Daten überhaupt sehen darf oder betrifft. Bei `"members"` hören auch Nutzer mit, die Mitgliederdaten gar nicht sehen (nur Vorstand/Kassierer/Admin haben Zugriff). Der Datenverkehr skaliert mit `O(aktive Nutzer × Listengröße)` pro einzelner Änderung.

Der Chat-Teil des Systems zeigt bereits das Zielbild: `SubscribeUser`/`BroadcastToUser` (`internal/hub/hub.go:43,65`) mit adressierten Payloads (`chat:new-message:123`) — er sendet nur an betroffene Nutzer. Dieser Change überträgt dieses erprobte Muster auf die Domänen-Events.

**Nicht-Ziel:** Echte Delta-Events (Payload mit geänderter ID, Client patcht lokal) sind bewusst **ausgeklammert**. Der globale Channel verwirft bei Bursts (Buffer 1, `hub.go:36`); ohne Resync-/Sequence-Mechanismus wäre ein verworfenes Delta ein dauerhaft veralteter Client. Dieser Change behält die „Reload-bei-Event"-Semantik und reduziert nur den **Empfängerkreis**. Delta-Events können ein späterer, separater Change sein.

## What Changes

- **`/api/events` abonniert pro Nutzer.** Der Handler nutzt `SubscribeUser(claims.UserID)` statt des globalen `Subscribe()`, analog zum Chat-Stream. Damit werden Domänen-Events adressierbar.
- **Audience-Auflösung pro Domäne.** Mutations-Handler berechnen die betroffenen/berechtigten Nutzer-IDs und senden gezielt (neuer Helfer `BroadcastToUsers([]int, event)` bzw. `BroadcastToAudience`). Regeln pro Topic:
  - `games`/`trainings`/`kader`/`duties`/`absences`/`event-note`/`attendance-changed`: Mitglieder der betroffenen Teams + zuständige Trainer/sportliche Leitung + Vorstand.
  - `members`/`users`: nur Vorstand/Kassierer/Admin (plus der betroffene Nutzer selbst, wenn es sein eigenes Profil ist).
  - `venues`/`settings`/`beitragssatz-changed`/`stammvereine`: bleiben **global** (echte vereinsweite Referenzdaten), aber niedrigfrequent.
  - `video-*`, `mitfahrgelegenheiten`: Team-/Kontext-bezogen scopen, wo Team ableitbar; sonst global.
- **Fallback bleibt erhalten.** `Broadcast(event)` (global) bleibt für die bewusst globalen Topics bestehen. Kein Zwang, alle 88 Stellen sofort umzustellen — die Umstellung erfolgt topic-weise, priorisiert nach Frequenz (`members`, `games`, `trainings`, `kader` zuerst).

## Topic-Abdeckung (vollständig, verifiziert)

Alle 18 im Code beobachteten `Broadcast`-Topics sind klassifiziert — kein Topic bleibt unadressiert:

| Scope | Topics | Audience |
|---|---|---|
| **Rolle** (Finance) | `members`, `users` | vorstand/vorstand_beisitzer/kassierer + admin (+ Betroffener) |
| **Team** | `games`, `trainings`, `kader`, `duties`, `absences`, `event-note`, `attendance-changed`, `mitfahrgelegenheiten`, `video-queued/-ready/-updated/-deleted` | Team-Mitglieder + Trainer/sL + Vorstand (Team aus Kontext ableitbar; sonst global) |
| **Global** (bewusst) | `venues`, `settings`, `beitragssatz-changed`, `stammvereine` | alle — echte vereinsweite Referenzdaten, niederfrequent |

- **Migrationsreihenfolge nach Frequenz** (aus der Broadcast-Zählung): `members` (23×) → `games`/`trainings` (je 11×) → `kader` (7×)/`duties` (6×) → Rest. Jede Phase ein Commit mit Test; nicht migrierte Topics bleiben über das globale `Broadcast` lauffähig.
- **`video-*`/`mitfahrgelegenheiten`:** team-scopebar, wo das Team aus dem Kontext ableitbar ist — sonst konservativ global (siehe „Konservativ scopen" im Design).

## Capabilities

### Added Capabilities

- `scoped-live-updates`: Domänen-SSE-Events werden nur an Clients zugestellt, die die betroffenen Daten sehen dürfen bzw. von der Änderung betroffen sind; global bleiben nur explizit vereinsweite Topics.

## Test-Anforderungen

| Route/Mechanismus | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `EventHub` | `TestHub_BroadcastToUsers_OnlyTargets` | `BroadcastToUsers([a,b], e)` erreicht die Kanäle von a und b, nicht die von c. |
| `/api/events` | `TestEvents_SubscribesPerUser` | Ein per `BroadcastToUser` an Nutzer A gesendetes Event erreicht A's `/api/events`-Stream, nicht den eines fremden Nutzers. |
| `POST /api/members` (o.ä.) | `TestMembersMutation_ScopedToVorstand` | Mitglieder-Event erreicht Vorstand/Kassierer/Admin-Streams, NICHT den eines reinen Spielers. |
| `PUT /api/games/{id}` | `TestGamesMutation_ScopedToTeamAndStaff` | Spiel-Event erreicht Team-Mitglieder + Trainer/sL + Vorstand, nicht teamfremde Spieler. |
| `PUT /api/club` (settings) | `TestSettingsMutation_StaysGlobal` | Vereinsweites Topic (`settings`) erreicht weiterhin alle Streams. |

**Garantierte Invariante:** Ein Client empfängt ein Domänen-Event **nur dann**, wenn er die zugehörige Ressource unter den bestehenden Auth-/Sichtbarkeitsregeln lesen dürfte, oder das Topic explizit als vereinsweit klassifiziert ist. Scoping verändert **nie** die Datensichtbarkeit selbst (die Lese-Routen bleiben autoritativ) — es reduziert nur, wer zum Nachladen aufgefordert wird.

## Mess-Anforderungen

Dies ist der Change mit dem größten erwarteten Effekt — und der in Einzel-Request-Payload **unsichtbar** ist. Er wird über die **SSE-Fan-out-Messung** aus `payload-measurement-harness` (Voraussetzung) belegt.

Das Werkzeug nutzt den festen 8-Client-Roster aus `payload-measurement-harness` (C1..C8, bekannte Funktion/Team). Baseline auf `main` = `8` für jede Mutation (globaler Broadcast).

| Mutation | Baseline (`main`) | Erwartung nach diesem Change | ausschlaggebende „darf-nicht-mehr"-Fälle |
|---|---|---|---|
| `members` (`PUT /api/members/{C5}`) | 8 | **3** (C1 admin, C2 vorstand, C3 kassierer) | Spieler C5/C6/C7, Trainer C4, Elternteil C8 empfangen nicht mehr |
| `games(T1)` (`PUT /api/games/{T1-Spiel}`) | 8 | **5** (C1, C2, C4 trainer T1, C5 spieler T1, C8 elternteil T1) | kassierer C3 und teamfremde Spieler C6/C7 empfangen nicht mehr |
| `settings` (`PUT /api/club`, Kontrolle) | 8 | **8** (bleibt global) | — (Regressionsschutz: global bleibt global) |

**Baseline-Regel:** Der Rückgang `8 → 3` bzw. `8 → 5` bei gleichbleibendem `8` für `settings` ist der Wirkungsnachweis; die Fan-out-Tabelle in `metrics/payload-baseline.md` wird mit den Nachher-Zahlen fortgeschrieben.

## Impact

- **Backend:**
  - `internal/hub/hub.go` — `BroadcastToUsers([]int, event)`-Helfer; ggf. `SubscribeUser`-Buffer prüfen.
  - `internal/hub/handler.go` — `/api/events` auf `SubscribeUser(claims.UserID)` umstellen; setzt voraus, dass der Handler die Claims sieht (Auth-Tier prüfen; `/api/events` ist bereits authentifiziert).
  - Audience-Resolver (neu, z. B. `internal/hub/audience.go` **oder** je Domänen-Package eine kleine Helper-Funktion, um die Architektur-Regel „Domain importiert nicht Domain" nicht zu verletzen) — betroffene User-IDs aus Team-/Rollen-Zugehörigkeit ableiten.
  - Mutations-Handler in `members`, `games`, `trainings`, `kader`, `duties` (Phase 1) — `Broadcast` → gezielter Versand.
- **Frontend:** keine Vertragsänderung nötig — `useLiveUpdates` empfängt dieselben Topic-Strings, nur seltener. (Profitiert zusätzlich vom Coalescing aus `efficient-data-loading-quickwins`.)
- **Architektur-Test:** Audience-Resolver so platzieren, dass `internal/arch/arch_test.go` grün bleibt (kein Domain↔Domain-Import). Bevorzugt: Resolver bekommt bereits aufgelöste IDs vom Handler, oder liegt als Foundation-Query-Helfer vor.
- **Reihenfolge/Abhängigkeit:** unabhängig lauffähig; entfaltet die volle Wirkung zusammen mit `live-update-coalescing` (Quick-Wins). Topic-weise Migration hält jeden Commit klein und testbar.
- **Risiko:** Ein zu eng gefasster Audience-Resolver könnte ein legitimes „Reload" verpassen (Client bleibt stumm). Mitigation: im Zweifel breiter scopen (z. B. ganzer Verein bei mehrdeutiger Zugehörigkeit) und pro Topic testen; der 30-Sekunden-Keepalive-Ping deckt keine Datenaktualität ab, daher konservativ scopen.
