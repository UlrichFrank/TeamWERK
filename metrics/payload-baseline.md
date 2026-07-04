# Payload-Baseline (main)

> Deterministischer Seed (measureRefTime=2026-01-15): 200 Members / 100 Games / 500 Duty-Slots / 100 Training-Sessions / 20 Duty-Types (10× 3072-Byte instruction_md) / 100 Chat-Nachrichten.
> Erhoben über den vollen Produktions-Router (testutil/prodserver) mit Admin-Bearer. Erzeugt via `make measure` (nicht Teil des blockierenden Gates).

## Payload pro Route

| Route | Pfad | Status | Bytes |
|---|---|---:|---:|
| kader | `/api/kader` | 200 | 19449 |
| duty-slots | `/api/duty-slots` | 200 | 89169 |
| duty-board | `/api/duty-board` | 200 | 101334 |
| duty-types | `/api/duty-types` | 200 | 34764 |
| games | `/api/games` | 200 | 60394 |
| game-participants | `/api/games/61/participants` | 200 | 6512 |
| training-sessions | `/api/training-sessions?from=2025-11-16&to=2026-03-16` | 200 | 39634 |
| chat-messages | `/api/chat/conversations/1/messages` | 200 | 48444 |
| teams | `/api/teams` | 200 | 466 |
| team-names | `/api/teams/names` | 200 | 334 |
| seasons | `/api/seasons` | 200 | 406 |
| venues | `/api/venues` | 200 | 526 |
| age-class-rules | `/api/age-class-rules` | 200 | 3 |
| encryption-pubkey | `/api/encryption-pubkey` | 200 | 43 |
| vapid-public-key | `/api/push/vapid-public-key` | 200 | 17 |

## Referenzdaten-Revalidierung (If-None-Match)

| Route | 1. Call | 2. Call (If-None-Match) |
|---|---|---|
| seasons | 200 / 406 B | 200 / 406 B _(kein ETag)_ |
| teams | 200 / 466 B | 200 / 466 B _(kein ETag)_ |
| venues | 200 / 526 B | 200 / 526 B _(kein ETag)_ |
| age-class-rules | 200 / 3 B | 200 / 3 B _(kein ETag)_ |
| duty-types | 200 / 34764 B | 200 / 34764 B _(kein ETag)_ |
| encryption-pubkey | 200 / 43 B | 200 / 43 B _(kein ETag)_ |
| vapid-public-key | 200 / 17 B | 200 / 17 B _(kein ETag)_ |

## SSE-Fan-out pro Mutation (8 Clients C1..C8)

| Mutation | Clients erreicht | Verteilung |
|---|---:|---|
| members (PUT /api/members/{C5}/status) | 8/8 | C1:1 C2:1 C3:1 C4:1 C5:1 C6:1 C7:1 C8:1 |
| games(T1) (PUT /api/games/{T1}) | 8/8 | C1:1 C2:1 C3:1 C4:1 C5:1 C6:1 C7:1 C8:1 |
| settings (PUT /api/club) | 8/8 | C1:1 C2:1 C3:1 C4:1 C5:1 C6:1 C7:1 C8:1 |
