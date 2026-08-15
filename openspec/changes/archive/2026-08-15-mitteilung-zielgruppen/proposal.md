## Why

Die Zielgruppen-Auswahl der Mitteilungen ist zur Hälfte tot und zur anderen Hälfte
redundant.

**Tot:** Der Composer bietet Vorstand und Admin die Zielgruppe „Rolle" mit den Werten
`spieler` / `elternteil` / `trainer` an (`web/src/pages/ChatPage.tsx:2711`). Der Resolver
löst sie so auf (`internal/chat/handler.go:1755`):

```go
case "role":
    rows, err = h.db.QueryContext(r.Context(),
        `SELECT id FROM users WHERE role = ?`, targetRole)
```

`users.role` kennt genau zwei Werte: `admin` und `standard` (`001_initial.up.sql:264`).
Die drei Optionen im Dropdown sind **Vereinsfunktionen** und stehen in
`member_club_functions` — die Query trifft die falsche Tabelle und liefert **immer null
Zeilen**. Der `broadcasts.target_role`-CHECK (`001_initial.up.sql:582`) listet korrekt
`spieler|elternteil|trainer|vorstand|…`; das Schema war für Vereinsfunktionen gedacht, der
Resolver liest woanders.

Der Fehler ist still. Der Absender bekommt seine eigene `broadcast_reads`-Zeile
(`handler.go:1244`) und sieht die Mitteilung anschließend in seinem Tab — sie wirkt
zugestellt. `ListBroadcasts` joint über `broadcast_reads`, also sieht sonst **niemand**
etwas. Kein Test deckt `targetType=role` ab; die vorhandenen Broadcast-Tests senden
ausschließlich `"all"`.

Ein zweiter, verwandter Bruch daneben: eine reine **sportliche Leitung** hat die Capability
`broadcast_messages`, sieht den Button, kann laut Dropdown aber nur „Team" wählen — und
scheitert dann an der Trainer-Klausel (`handler.go:1187`), die `kader_trainers`-Mitgliedschaft
verlangt. Sie steht dort typischerweise nicht drin. Ergebnis: Button sichtbar, jede
Zielgruppe endet in HTTP 403.

**Redundant:** Die Zielgruppe „Team" dupliziert, was der Chat besser kann.
`internal/chat/team_groups.go` liefert pro Team bereits drei Standardgruppen — **Trainer /
Spieler / Eltern** — mit Rückkanal. Eine Mitteilung an ein Team ist derselbe Empfängerkreis
ohne Antwortmöglichkeit. Zwei Wege zum selben Publikum, von denen einer schlechter ist.

Was fehlt, ist die Gegenrichtung: **vereinsweite** Zielgruppen. Wer alle Spieler oder alle
Eltern des Vereins erreichen will, hat heute nur „Alle Mitglieder" (= wörtlich alle User,
inklusive Vorstand und Kassierer) oder das kaputte Rollen-Dropdown.

## What Changes

- **Vier Zielgruppen ersetzen die alten drei.** `target_type` wird zu
  `users` | `members` | `spieler` | `eltern`. `target_id` und `target_role` entfallen
  ersatzlos:

  | Zielgruppe | Label | Auflösung |
  |---|---|---|
  | `users` | Alle Nutzer | `SELECT id FROM users` |
  | `members` | Alle Mitglieder | `users` ⋈ `members.user_id` |
  | `spieler` | Alle Spieler | `users` ⋈ `members` ⋈ `member_club_functions.function='spieler'` |
  | `eltern` | Alle Eltern | `SELECT DISTINCT parent_user_id FROM family_links` |

- **Keine Eltern-Erbung bei `spieler`.** Anders als die Ordner-ACL
  (`policy/folders.go`, `case "club_function"`, wo Eltern über `ctx.family()` die
  Vereinsfunktionen ihrer Kinder erben) sind die Mengen hier **disjunkt gedacht**: „Alle
  Spieler" erreicht Spieler, „Alle Eltern" erreicht Eltern. Wer beide will, sendet zweimal
  oder wählt „Alle Mitglieder".

- **Zielgruppe „Team" entfällt.** Ersatz ist die bestehende Team-Standardgruppe im Chat
  (`GET /api/chat/team-groups`), die denselben Kreis mit Rückkanal erreicht.

- **Trainer verlieren das Mitteilungsrecht.** `CanBroadcast` wird zu
  `admin | vorstand | sportliche_leitung`. Der Trainer-Sonderpfad in `SendBroadcast`
  (`handler.go:1187-1206`) entfällt vollständig — er existierte nur für `target_type: team`.

- **`broadcast_all` wird ersatzlos gestrichen.** Sportliche Leitung darf dieselben vier
  Zielgruppen wie Vorstand und Admin; die Capability unterscheidet damit keine zwei Mengen
  mehr. Eine Capability (`broadcast_messages`) genügt.

- **Der Fan-out wird sichtbar.** `POST /api/chat/broadcasts` antwortet künftig
  `201 {"id": <id>, "recipients": <n>}`; das Frontend meldet „An 183 Empfänger gesendet".
  Ein Fan-out ins Leere ist danach nicht mehr unbemerkbar — genau die Klasse Fehler, die
  diesen Change ausgelöst hat.

- **Migration `049`.** `broadcasts` wird neu aufgebaut: neuer CHECK, `target_id` und
  `target_role` fallen weg. Bestandswerte werden abgebildet: `'all' → 'users'`,
  `'team'`/`'role'` → `'legacy'`. `broadcast_reads` bleibt unangetastet, alte Mitteilungen
  bleiben für ihre Empfänger unverändert lesbar.

- **Prop-Umbenennung.** `BroadcastComposer` bekommt seine Zielgruppen künftig aus
  `broadcast_messages`; der irreführende Prop-Name `isAdmin` (der in Wahrheit
  `broadcast_all` transportierte, also *vorstand-like*, nicht Admin) verschwindet mit der
  Capability.

## Nicht Teil dieses Changes

- **`/dokumente` bleibt unverändert.** Kein neuer Principal, keine Änderung an
  `folder_permissions` oder `policy.FolderAccess`. Die beiden Vokabulare sind nach diesem
  Change bewusst getrennt — Begründung in `design.md` §1.
- **Keine Mehrfachauswahl.** Genau eine Zielgruppe pro Mitteilung. „Spieler **und** Eltern"
  heißt zwei Mitteilungen oder „Alle Mitglieder". Siehe `design.md` §4.
- **Keine Empfänger-Vorschau vor dem Senden.** Der Zähler kommt in der Antwort, nicht als
  eigener Preview-Endpoint. Siehe `design.md` §5.
- **Lesebestätigungen** (`47 / 183 gelesen`) sind ein eigener Folge-Change
  (`mitteilung-lesebestaetigung`). Er setzt auf der hier entstehenden, sauberen
  Empfängermenge auf.

## Capabilities

### Modified Capabilities

- **`chat-broadcasts`** — Zielgruppen-Vokabular, Absenderkreis und die Antwort auf
  `POST /api/chat/broadcasts`. Die heutige Spec schreibt „admin, vorstand oder trainer …
  `target_type: all`, `team` oder `role`" wörtlich fest und steht nach diesem Change
  vollständig gegen den Code.
- **`me-capabilities`** — `broadcast_all` fällt aus dem Capability-Vokabular; die Personas
  von `broadcast_messages` ändern sich (`trainer` raus). Zwei bestehende Szenarien der Spec
  referenzieren `broadcast_all` namentlich und müssen mitziehen.

## Test-Anforderungen

**Routen:**

| Route | Fall | Erwartung |
|---|---|---|
| `POST /api/chat/broadcasts` | Vorstand, `targetType: "spieler"` | 201, `recipients` == Anzahl User mit Vereinsfunktion `spieler`; genau diese haben eine `broadcast_reads`-Zeile |
| | Vorstand, `targetType: "eltern"` | 201, nur `family_links.parent_user_id` erhalten Zeilen |
| | Vorstand, `targetType: "members"` | 201, User ohne `members`-Zeile erhalten **keine** Zeile |
| | Vorstand, `targetType: "users"` | 201, alle User erhalten Zeilen |
| | sportliche Leitung, `targetType: "users"` | 201 (nicht 403 — heutiges Verhalten wäre 403) |
| | Trainer (ohne vorstand/sL), beliebige Zielgruppe | 403 |
| | Spieler / Kassierer / Eltern | 403 |
| | `targetType: "team"` | 400 (Wert existiert nicht mehr) |
| | `targetType: "role"` | 400 |
| | `targetType` fehlt oder unbekannt | 400 |
| | leerer `body` ohne `mediaId` | 400 (unverändert) |
| | `mediaId` auf nicht existierende Zeile | 400 (unverändert) |
| `GET /api/me` | sportliche Leitung | `capabilities` enthält `broadcast_messages` |
| | Trainer (ohne vorstand/sL) | `capabilities` enthält **nicht** `broadcast_messages` |
| | beliebige Persona | `capabilities` enthält **nie** `broadcast_all` |

**Garantierte Invarianten** (je ein eigener Test):

1. **Keine Zielgruppe fällt still ins Leere.** Für jede der vier Zielgruppen gilt in einer
   Fixture mit passenden Nutzern: `recipients > 0` **und** die Menge der erzeugten
   `broadcast_reads.user_id` ist exakt die erwartete Menge — nicht nur „nicht leer". Genau
   der Test, der den Bug hier verhindert hätte.
2. **`spieler` und `eltern` sind disjunkt aufgelöst.** Ein Elternteil, dessen Kind die
   Vereinsfunktion `spieler` trägt und das selbst keine hat, erhält bei `spieler`
   **keine** Zeile und bei `eltern` eine. Der Test unterscheidet diesen Change explizit
   von der Ordner-ACL.
3. **Mitglieder ohne Zugang verschwinden nicht unbemerkt.** Ein Mitglied mit
   `members.user_id IS NULL` und Vereinsfunktion `spieler` taucht in keiner Empfängermenge
   auf, und `recipients` zählt es nicht mit.
4. **Der Absender ist immer im Bestand, aber nie im Fan-out-Zähler.** Sendet ein Vorstand
   an `users`, erhält er eine `broadcast_reads`-Zeile mit gesetztem `read_at`, bekommt
   weder SSE noch Push, und `recipients` zählt ihn nicht doppelt.
5. **Kein Doppelversand bei Mehrfachzugehörigkeit.** Ein Elternteil mit zwei Kindern
   erhält bei `eltern` genau **eine** Zeile (`DISTINCT`), ein Mitglied mit mehreren
   `member_club_functions`-Zeilen bei `members` genau eine.
6. **Legacy-Werte sind schreibseitig tot, leseseitig harmlos.** Nach der Migration
   antwortet jeder `POST` mit `targetType` aus `{all, team, role}` mit 400; ein per
   Migration auf `'legacy'` gesetzter Bestands-Broadcast bleibt über
   `GET /api/chat/broadcasts` für seine Empfänger unverändert sichtbar.
7. **`broadcast_all` existiert nirgends mehr.** Repo-weiter Grep-Test (Vorbild:
   `internal/arch`): der String `broadcast_all` kommt in `internal/` und `web/src/` nicht
   mehr vor. Verhindert, dass eine Frontend-Abfrage auf eine nie gelieferte Capability
   zurückbleibt und einen Button dauerhaft versteckt.
