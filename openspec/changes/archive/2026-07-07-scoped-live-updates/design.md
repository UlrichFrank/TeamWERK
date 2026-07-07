# Design — scoped-live-updates

## Ausgangslage (verifiziert im Code)

```
/api/events (useLiveUpdates)        /api/chat/events (useChatEvents)
        │                                   │
   Subscribe()                        SubscribeUser(uid)      ← Zielbild
   → h.clients (ALLE)                 → h.userClients[uid]
   Broadcast("games") ×alle           BroadcastToUser(uid, "chat:new-message:123")
   88 globale Aufrufe                  bereits adressiert + Delta
```

Der Umbau bringt den Domänen-Stream auf das bereits produktive Chat-Muster.

## Mechanik

1. **Per-User-Subscription für `/api/events`.** `hub.Handler.Events` ruft `SubscribeUser(claims.UserID)` statt `Subscribe()`. `/api/events` liegt bereits im authentifizierten Tier (Token-basiert), Claims sind verfügbar.
2. **Adressierter Versand.** Neuer Helfer:
   ```go
   func (h *EventHub) BroadcastToUsers(userIDs []int, event string) {
       h.mu.Lock(); defer h.mu.Unlock()
       for _, uid := range userIDs {
           for ch := range h.userClients[uid] {
               select { case ch <- event: default: }
           }
       }
   }
   ```
3. **Audience-Resolver.** Pro Mutation die betroffenen User-IDs bestimmen. Beispiel `games`:
   ```
   audience(gameID) =
       users, die Mitglied eines game_teams-Teams sind (über kader/team-Zugehörigkeit)
     ∪ Trainer/sportliche_leitung der betroffenen Teams
     ∪ alle mit Vereinsfunktion vorstand/vorstand_beisitzer
   ```
   `members`/`users`: `audience = users mit club_function ∈ {vorstand, vorstand_beisitzer, kassierer} ∪ admin (∪ betroffener Nutzer selbst)`.

## Topic-Klassifizierung

| Topic(s) | Scope | Begründung |
|---|---|---|
| `members`, `users` | Vorstand/Kassierer/Admin (+ Betroffener) | nur diese lesen Mitglieder-/Nutzerdaten |
| `games`, `trainings`, `kader`, `duties`, `absences`, `event-note`, `attendance-changed` | Team-Mitglieder + Trainer/sL + Vorstand | team-lokale Daten |
| `mitfahrgelegenheiten`, `video-*` | Team/Kontext, wo ableitbar; sonst global | teils teamgebunden |
| `venues`, `settings`, `beitragssatz-changed`, `stammvereine` | **global** (bleibt `Broadcast`) | echte vereinsweite Referenzdaten, niederfrequent |

## Architektur-Regel (Domain↔Domain)

`internal/arch/arch_test.go` verbietet gegenseitige Domain-Imports. Der Resolver darf also nicht quer durch `members`/`games`/`teams` importieren. Zwei zulässige Optionen:

- **A (bevorzugt):** Der Handler kennt seinen Kontext (z. B. `gameID` → `team_ids` liegen im Handler ohnehin vor) und ruft einen **Foundation**-Query-Helfer `hub/audience` mit `*sql.DB` + bereits bekannten IDs auf, der nur generische Joins (`member_club_functions`, Team-Zugehörigkeit) fährt.
- **B:** Jedes Domänen-Package berechnet seine Audience selbst inline und übergibt fertige `[]int` an `BroadcastToUsers`. Kein neues Package, kein Import-Konflikt.

Entscheidung wird beim ersten Topic (Phase 1) getroffen und für die übrigen konsistent angewandt.

## Migrationsstrategie (topic-weise, priorisiert)

1. Infrastruktur (`BroadcastToUsers`, `/api/events` per-user) + Test.
2. `members`/`users` (23+2 Aufrufe, klarste Audience: Finance-Gruppe).
3. `games`/`trainings` (je 11).
4. `kader`/`duties`.
5. Rest nach Bedarf; `venues`/`settings`/… bleiben global.

Jede Phase ist ein eigener Commit mit Test; nicht migrierte Topics bleiben über `Broadcast` global lauffähig.

## Konservativ scopen

Bei mehrdeutiger Zugehörigkeit lieber **breiter** senden (bis hin zu global) als einen legitimen Reload zu verpassen — ein zu enger Filter macht Clients stumm. Der Keepalive-Ping (30 s) transportiert keine Datenaktualität; die Korrektheit hängt allein am Audience-Resolver, daher Tests pro Topic verpflichtend.
