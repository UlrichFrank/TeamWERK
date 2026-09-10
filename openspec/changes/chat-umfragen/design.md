## Context

Anforderungen: `specs/chat-umfragen/spec.md`, Motivation: `proposal.md`.

Relevanter Ist-Zustand in `internal/chat`:
- Eine Nachricht ist eine `messages`-Zeile (`body` 1–2000 Zeichen oder Bild, Soft-Delete
  über `deleted_at`, `is_system` für Systemhinweise).
- **Reaktionen** sind das nächste Vorbild für „Auswahl pro Nutzer an einer Nachricht":
  eigene Tabelle `message_reactions`, in `ListMessages` per Batch-Query über die IDs der
  ausgelieferten Seite angehängt, Mutation fächert `chat:new-message:<convId>` an alle
  aktiven Mitglieder auf.
- Der Chat hat einen eigenen SSE-Kanal (`/api/chat/events`, plain colon-strings,
  `hub.BroadcastToUser`). `chat:new-message` führt im Client zu `appendNewMessages`
  (`?after=`-Delta) und bei leerem Delta zum **Voll-Reload** der Seite — so kommen
  Reaktionsänderungen heute an.
- Das Broadcast-Gate (`internal/arch/broadcast_test.go`) akzeptiert jeden Aufruf, dessen
  Bezeichner „broadcast" enthält.

## Goals / Non-Goals

**Goals:**
- Umfrage fügt sich in alle bestehenden Nachrichten-Pfade ein (Ungelesen, Konversations-
  liste, Push, Antworten, Reaktionen, Löschen), ohne diese Pfade umzubauen.
- Eine Stimme in einer großen Team-Gruppe (40+ Mitglieder) löst keinen Voll-Reload bei allen
  offenen Clients aus.

**Non-Goals:**
- Kein generisches „Nachrichtentyp"-System (`messages.kind`). Umfrage ist der zweite
  Sondertyp nach Bild; ein Typsystem lohnt sich erst beim dritten.
- Keine Umfragen in Direktchats und Broadcasts, keine anonymen Umfragen, kein Ablaufdatum
  (siehe Proposal).

## Decisions

### 1. Umfrage = `messages`-Zeile + Seitentabellen (statt eigener Entität im Verlauf)

Die Frage steht im bestehenden `messages.body`. Dazu drei neue Tabellen (Migration `059`):

```sql
CREATE TABLE chat_polls (
    message_id     INTEGER PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    allow_multiple INTEGER NOT NULL DEFAULT 0 CHECK (allow_multiple IN (0,1)),
    closed_at      DATETIME
);
CREATE TABLE chat_poll_options (
    id         INTEGER PRIMARY KEY,
    message_id INTEGER NOT NULL REFERENCES chat_polls(message_id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    label      TEXT    NOT NULL CHECK (length(label) BETWEEN 1 AND 100),
    UNIQUE (message_id, position)
);
CREATE TABLE chat_poll_votes (
    option_id  INTEGER NOT NULL REFERENCES chat_poll_options(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (option_id, user_id)
);
CREATE INDEX idx_chat_poll_votes_user ON chat_poll_votes(user_id);
```

**Warum:** Ungelesen-Zähler, `lastMessage`, Sortierung der Konversationsliste, Push-Titel,
Antwort-Zitat (`reply_to_body`), Reaktionen und Soft-Delete arbeiten alle auf `messages` und
funktionieren damit ohne Änderung. „Ist diese Nachricht eine Umfrage?" = existiert eine
`chat_polls`-Zeile. Frage ≤ 200 Zeichen ⇒ nie über `messagePreviewLen` (280), der
Volltext-Nachladepfad greift also nie.

**Kein `closed_by`:** nur der Ersteller (= `messages.sender_id`) darf beenden, die Spalte
wäre redundant.

**Alternative verworfen — Umfrage als JSON in `messages.body`:** Stimmen müssten per
Read-Modify-Write in JSON geschrieben werden (Race bei gleichzeitigen Stimmen, keine
FK-Kaskade auf Nutzer, Preview/Push würden JSON zeigen).

**Alternative verworfen — `messages.kind`-Spalte:** erfordert einen Table-Rebuild von
`messages` (SQLite, siehe Migration 028) für eine Information, die die Existenz der
`chat_polls`-Zeile schon trägt.

### 2. Stimme = vollständige Auswahl setzen (`PUT …/poll/vote`), nicht Toggle

`PUT /api/chat/messages/{id}/poll/vote {optionIds: []}` ersetzt in einer Transaktion alle
Stimmen des Nutzers an dieser Umfrage (`DELETE … WHERE user_id=? AND option_id IN (Optionen
der Umfrage)`, dann `INSERT`). Idempotent, leere Liste = zurückziehen.

**Warum nicht Toggle wie bei Reaktionen:** der Wechsel einer Einfachauswahl wären sonst zwei
Requests (ab- und anwählen) mit einem sichtbaren Zwischenzustand „keine Stimme" bei den
anderen Clients; und ein doppelt gesendeter Toggle kippt die Auswahl zurück. Mit „setzen"
entscheidet der Client die Toggle-Semantik (erneutes Antippen der eigenen Option sendet die
Auswahl ohne sie), der Server prüft nur Invarianten.

Der Status-Check (`closed_at IS NULL`) läuft **innerhalb** derselben Transaktion wie das
Schreiben, damit eine Stimme nicht nach dem Beenden durchrutscht.

### 3. Eigenes Live-Event `chat:poll-updated:<convId>:<messageId>` + Einzelabruf

Stimmen und Beenden fächern **nicht** `chat:new-message` auf, sondern
`chat:poll-updated:<convId>:<messageId>`. Der Client lädt dann nur
`GET /api/chat/messages/{id}/poll` und ersetzt `poll` an dieser einen Nachricht im State.

**Warum:** `chat:new-message` ohne neue Nachricht endet im Client im Voll-Reload (leeres
Delta ⇒ Fallback). Bei einer Umfrage in einer 40er-Team-Gruppe, in der binnen Minuten 30 Leute
abstimmen, wären das 30 × 40 Seiten-Reloads à 100 Nachrichten — und jeder davon riskiert die
Scroll-Position (die Klasse Bug, die `chat-open-at-unread`/`chat-scroll-anchor` mühsam gefixt
haben). Das Event trägt die `messageId`, damit der Client den Einzelabruf ohne Suche machen
kann; ist die Konversation nicht geöffnet, ignoriert er es (Ungelesen ändert sich nicht).

Neue Umfrage und Löschen einer Umfrage bleiben bei `chat:new-message` — das sind echte
Verlaufsänderungen.

### 4. Auslieferung in `ListMessages`: Batch-Anhang nach Reaktions-Muster

`ListMessages` bekommt nach dem Reaktions-Anhang einen Poll-Anhang über die IDs der Seite
(zwei Queries: Umfragen+Optionen, dann Stimmen mit Namen), nur für nicht gelöschte
Nachrichten. Pro Umfrage-Nachricht ein Feld:

```json
"poll": {
  "allowMultiple": false,
  "closedAt": null,
  "voterCount": 4,
  "options": [
    { "id": 11, "label": "Pizza", "count": 3, "voted": true,
      "voters": [{ "id": 7, "name": "Anna Beispiel" }] }
  ]
}
```

`voters` wird wie `reactions[].userNames` immer mitgeliefert (nicht anonym, Größenordnung
≤ 10 Optionen × Gruppengröße ist unkritisch). `voterCount` = Anzahl **verschiedener** Nutzer
(bei Mehrfachauswahl ≠ Summe der `count`). Der Balken rechnet `count / voterCount`.
`GET /api/chat/messages/{id}/poll` liefert dasselbe Objekt über dieselbe Aufbau-Funktion
(`loadPolls(ctx, userID, msgIDs)`), damit Liste und Einzelabruf nicht auseinanderlaufen.

Konversationsliste: `LastMessage` bekommt `isPoll bool` (per `EXISTS` auf `chat_polls` in
der `last_body`-Subquery-Nachbarschaft von `ListConversations` und `getConversation`), das
Frontend zeigt dann ein `BarChart3`-Icon vor der Frage.

### 5. Anlegen über eigene Route, Fan-out geteilt mit `SendMessage`

`POST /api/chat/conversations/{id}/polls` statt eines `poll`-Felds in `SendMessage`: die
Validierung (2–10 Optionen, Duplikate, nur Gruppen) und die Mehr-Tabellen-Transaktion
bleiben aus dem ohnehin langen `SendMessage` heraus. Den Teil nach dem Insert (SSE
`chat:new-message` + Push mit Unread-Badge) ziehen beide aus einem gemeinsamen Helfer
`broadcastNewMessage(r, convID, senderID, convType, convName, preview)`; der Name enthält
„broadcast" und erfüllt damit das Broadcast-Gate für beide Routen. Die Push-Vorschau lautet
`Umfrage: <Frage>` (auf 80 Zeichen gekürzt wie bisher).

### 6. Berechtigungen

| Aktion | Wer | sonst |
|---|---|---|
| Anlegen | aktives Mitglied, nur `type='group'` | 403 / 400 |
| Abstimmen | aktives Mitglied (`isActiveMember`) | 403 |
| Lesen (Liste/Einzel) | Mitglied inkl. ausgetretener (`isMember`, wie `ListMessages`) | 403 |
| Beenden | nur `messages.sender_id` — **kein** Admin-Bypass | 403 |
| Löschen | unverändert (`chat-message-delete`: Absender oder Admin) | 403 |

Admin darf nicht beenden, weil Beenden das Ergebnis im Namen des Erstellers festschreibt;
für Moderation reicht Löschen. Stimmen ausgetretener Mitglieder bleiben (die Stimme war
zum Zeitpunkt der Abgabe legitim; Nachzählen beim Austritt hieße, Ergebnisse still zu
ändern). Löscht ein Nutzer seinen Account, kaskadiert `ON DELETE CASCADE` seine Stimmen weg
— konsistent mit `message_reactions`.

### 7. Bearbeiten gesperrt

`EditMessage` prüft vor dem `UPDATE`, ob eine `chat_polls`-Zeile existiert → HTTP 409.
Im Frontend blendet das Kontextmenü (Desktop + `MobileMessageActionOverlay`) „Bearbeiten"
für `msg.poll` aus und zeigt stattdessen — für den Ersteller bei offener Umfrage —
„Umfrage beenden".

### 8. Frontend-Aufteilung

`ChatPage.tsx` ist mit ~2960 Zeilen bereits sehr groß; die Umfrage kommt deshalb in eigene
Komponenten unter `web/src/components/`:
- `ChatPollCreateModal` — Frage, dynamische Optionsliste (startet mit 2 Feldern, „Option
  hinzufügen" bis 10, leere Felder werden beim Senden verworfen), Schalter
  „Mehrfachantworten erlauben"; Senden erst aktiv, wenn Frage + ≥ 2 verschiedene Optionen.
  Modal-/Input-/Button-Klassen aus `buttonStyles.ts`.
- `ChatPollCard` — ersetzt in `MessageBubble` den Text-Body, wenn `msg.poll` gesetzt ist:
  Frage, Hinweis „Eine Antwort wählen"/„Mehrere Antworten möglich" bzw. „Beendet", je Option
  Radio/Checkbox-Optik (`Circle`/`CircleCheck`/`Square`/`SquareCheck` aus lucide), Label,
  Zahl, Balken (`bg-brand-yellow` auf `bg-brand-border-subtle`), Fußzeile „N Stimmen ·
  Stimmen anzeigen". Optimistisches Update beim Tippen, Rollback + Toast bei Fehler.
- `ChatPollVotesModal` — Namen je Option.
- Composer: `BarChart3`-Button neben `Paperclip`, nur wenn `activeConv.type === "group"` und
  kein Edit-Modus.
- `useChatEvents`-Handler: `chat:poll-updated` → bei geöffneter Konversation
  `GET /chat/messages/{id}/poll` und `setMessages(prev => prev.map(...))`.

## Risks / Trade-offs

- [Stimme kommt an, Live-Event geht verloren (SSE-Reconnect)] → Beim nächsten Öffnen/Laden
  der Konversation liefert `ListMessages` den aktuellen Stand; der eigene Client aktualisiert
  optimistisch und korrigiert über die `PUT`-Antwort/den Einzelabruf.
- [Sehr alte Umfrage nicht auf der geladenen Seite] → Das Event wird ignoriert, wenn die
  Nachricht nicht im State ist; beim Hochscrollen (`?before=`) kommt sie mit aktuellem Stand.
- [Namen aller Abstimmenden für alle sichtbar] → gewollt (Signal-Verhalten, im Proposal
  festgehalten); Nutzer sehen den Hinweis „Stimmen sind für alle sichtbar" im Erstell-Modal.
- [Doppeltes Antippen vor Server-Antwort] → `PUT` mit vollständiger Auswahl ist idempotent;
  der letzte Request gewinnt, kein Kippen wie bei einem Toggle.
- [Migrationsnummer-Kollision mit parallelen Changes] → Nummer beim Implementieren gegen
  `ls internal/db/migrations/` prüfen; golang-migrate überspringt Nummern ≤ DB-Version still.

## Migration Plan

1. Migration `059_chat_polls.up.sql` (drei Tabellen + Index) / `.down.sql` (DROP in
   umgekehrter Reihenfolge). Rein additiv, keine Bestandsdaten betroffen.
2. `make deploy` führt `migrate up` automatisch aus.
3. Rollback: `make migrate-down` droppt die Tabellen; bestehende Umfrage-Nachrichten bleiben
   dann als reine Textnachricht (die Frage) im Verlauf stehen — kein Datenverlust außerhalb
   der Umfrage selbst.

## Open Questions

- Soll beim Beenden zusätzlich ein Systemhinweis („Anna hat die Umfrage beendet") in den
  Verlauf? Aktuell nein (würde Ungelesen-Zähler erhöhen); lässt sich später additiv ergänzen,
  ohne Specs oder Datenmodell dieses Changes zu ändern.
