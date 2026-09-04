## Why

Mitteilungen sind der einzige Kanal, in dem eine Information stehen bleibt: eigener Tab,
eigener Ungelesen-Zähler, keine nachrückenden Nachrichten. Genau das brauchen Trainer für
Ansagen an ihr Team — im Gruppenchat scrollt die Hallenänderung nach zwei Wortmeldungen
aus dem Blick.

`mitteilung-zielgruppen` (archiviert 2026-08-15) hat Trainern das Senderecht entzogen, mit
der Begründung, die Team-Standardgruppen des Chats erreichten denselben Kreis mit Rückkanal
(`design.md §2`). Das Argument betrachtet nur den **Empfängerkreis** und übersieht die
**Haltbarkeit**: eine Gruppennachricht ist ein Beitrag im Verlauf, eine Mitteilung ist eine
Ansage, die oben stehen bleibt. Beide Kanäle erreichen dieselben Leute — sie halten die
Information unterschiedlich lange sichtbar.

## What Changes

- **Trainer dürfen wieder Mitteilungen senden**, aber nur an die Standardgruppen der Kader,
  die sie in der **aktiven Saison** trainieren (`kader_trainers`). Vereinsweite Zielgruppen
  bleiben ihnen verschlossen.
- **Zielgruppen-Auswahl wird zur Mehrfachauswahl** und übernimmt das Vokabular des
  Gruppen-Pickers aus dem Chat: Team-Standardgruppen (`spieler` / `eltern` / `trainer` je
  Team, plus „Alle Trainer"), **ohne** Einzelpersonen. Die vier vereinsweiten Ziele
  (`users`, `members`, `spieler`, `eltern`) bleiben unverändert bestehen und stehen weiter
  nur Vorstand / sportlicher Leitung / Admin offen.
- **BREAKING (API):** `POST /api/chat/broadcasts` nimmt statt `targetType: string` ein
  `targets: [...]`-Array entgegen. Die Empfängermenge ist die **Vereinigung** der gewählten
  Ziele, dedupliziert; `recipients` zählt weiterhin distinkte User ohne den Absender.
- `broadcasts.target_type` (eine Spalte, ein Wert) weicht der Zeilentabelle
  `broadcast_targets` — genau der additive Weg, den `mitteilung-zielgruppen` `design.md §4`
  für diesen Fall vorgezeichnet hat. Bestandszeilen werden als Ein-Zeilen-Fall migriert,
  `legacy` bleibt lesbar und nicht schreibbar.
- Die Capability `broadcast_messages` bekommt damit wieder zwei Stufen: wer sie hat, darf
  senden — *woran*, entscheidet die serverseitig geprüfte Ziel-Allowlist des Absenders.

Nicht Teil dieses Changes: Einzelpersonen als Ziel (dafür gibt es den Direktchat),
Lesebestätigungen pro Ziel, und ein Rückkanal auf Mitteilungen.

## Capabilities

### New Capabilities

Keine.

### Modified Capabilities

- `permissions`: die neue Route `GET /api/chat/broadcast-targets` hängt wie
  `POST /api/chat/broadcasts` an der Ziel-Allowlist des Absenders und gehört damit nicht
  zu den Endpoints, die jede Persona erreicht.
- `chat-broadcasts`: Senderecht (Trainer kommen hinzu, begrenzt auf ihre Kader),
  Ziel-Vokabular (Team-Standardgruppen zusätzlich zu den vereinsweiten Zielen),
  Request-Format (`targets`-Array statt `targetType`-String), Empfänger-Auflösung als
  deduplizierte Vereinigung, und die Ablösung von `broadcasts.target_type` durch
  `broadcast_targets`.

## Impact

**Datenbank** — neue Migration (nächste freie Nummer):
- Neue Tabelle `broadcast_targets (broadcast_id, kind, team_id)`, PK über alle drei Spalten.
- `broadcasts.target_type` entfällt; Bestandswerte wandern als eine `broadcast_targets`-Zeile
  mit `kind = <alter Wert>` und `team_id IS NULL` mit. Tabellen-Rebuild nach dem Muster aus
  `049` (SQLite kann CHECK nicht in-place ändern).
- `broadcast_reads` wird **nicht** angefasst — daran hängt die gesamte Zustellung.

**Backend:**
- `internal/chat/audiences.go` — Ziel-Vokabular um die Team-Gruppen erweitern; Auflösung
  einer Ziel-Liste zur deduplizierten User-Menge.
- `internal/chat/handler.go` — `SendBroadcast` (Autorisierung pro Ziel statt pauschal,
  neues Request-Format), `resolveBroadcastRecipients`.
- `internal/chat/team_groups.go` — die vorhandenen Auflösungs-Queries
  (`teamGroupMemberQuery`, `allTrainersMemberQuery`) werden wiederverwendet, nicht kopiert.
- `internal/policy/rules.go` — `CanBroadcast` schließt Trainer ein; die Zielmenge wird
  nicht in `policy` entschieden, sondern in `chat` gegen die Kader des Absenders geprüft.
- `internal/permissions/matrix_test.go` — Erwartung für `POST /api/chat/broadcasts`.

**Frontend:**
- `web/src/pages/ChatPage.tsx` — Composer bekommt die Mehrfachauswahl (Vereinsweit-Block +
  Gruppen-Block), analog zum bestehenden Gruppen-Picker; `canBroadcast`-Gating bleibt.
- Neue Route für die erlaubten Ziele des Absenders (der Composer kann sie nicht raten:
  `GET /api/chat/team-groups` zeigt bewusst mehr, nämlich alles Lesbare).

**Betrieb:** Nach dem Deploy können Trainer senden, ohne dass jemand etwas einstellt. Die
Ankündigung an die Trainer gehört zu den Tasks — `mitteilung-zielgruppen` hatte ihnen die
Fähigkeit ausdrücklich weggenommen.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `GET /api/chat/broadcast-targets` | `TestBroadcastTargets_Vorstand` | 200, enthält die vier vereinsweiten Ziele **und** je Team der aktiven Saison drei `team_*`-Ziele |
| `GET /api/chat/broadcast-targets` | `TestBroadcastTargets_TrainerNurEigeneKader` | 200, nur `team_*` seiner Kader + `alle_trainer`; kein vereinsweites Ziel, kein fremdes Team |
| `GET /api/chat/broadcast-targets` | `TestBroadcastTargets_OhneSenderecht` | 403 |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_TrainerAnEigenesTeam` | 201; genau Spieler (inkl. erweitertem Kader) + Eltern des Teams haben je **eine** `broadcast_reads`-Zeile |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_TrainerFremdesTeam` | 403, kein `broadcasts`-Insert |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_EinUnerlaubtesZielKipptAlles` | 403; auch für das *erlaubte* Ziel entsteht keine `broadcast_reads`-Zeile |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_TrainerNichtVereinsweit` | 403 bei `kind: "spieler"` |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_TrainerErbtKeinRechtAlsElternteil` | 403 für das Team des eigenen Kindes |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_VereinigungDedupliziert` | 201; Elternteil zweier Kinder im selben Team bekommt eine Zeile, `recipients` zählt es einmal |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_LeereZiele` / `_TeamZielOhneTeamID` / `_LegacyNichtSetzbar` | je 400 |
| `POST /api/chat/broadcasts` | `TestSendBroadcast_VorstandUnverändert` | 201 für alle vier vereinsweiten Ziele (Regression gegen die Bestandsfunktion) |
| Migration `055` | `TestMigration055_BestandszielUndReadsUeberleben` | Bestands-`broadcast_reads` unverändert; je Bestandszeile genau eine `broadcast_targets`-Zeile mit altem Wert, `team_id IS NULL` |

**Garantierte Invarianten:**
1. Kein Ziel ohne Allowlist-Treffer wird zugestellt — und ein einziges unerlaubtes Ziel
   verhindert die gesamte Zustellung (keine Teilzustellung).
2. Ein Empfänger bekommt für eine Mitteilung genau eine `broadcast_reads`-Zeile, unabhängig
   davon, über wie viele Ziele er getroffen wird.
3. `team_spieler` löst denselben Kreis auf wie die Chat-Standardgruppe `spieler` desselben
   Teams — mechanisch abgesichert dadurch, dass beide dieselbe Query benutzen.
4. Die Zustellung hängt ausschließlich an `broadcast_reads`; die Migration lässt die Tabelle
   unangetastet.
