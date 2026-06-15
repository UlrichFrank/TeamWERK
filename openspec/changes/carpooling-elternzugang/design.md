## Context

Das Carpooling-Modul (`internal/carpooling/handler.go`) kennt nur `claims.UserID` als Identität. Das Proxy-Account-Pattern ist im System bereits für Games (`RespondToGame`) und Trainings (`RespondToTraining`) umgesetzt: Eltern reichen `member_id` mit, Backend prüft `parentHasChild()`.

Im Carpooling-Modul existiert dieses Pattern bislang nicht. Einträge in `mitfahrgelegenheiten` haben ein `user_id`-Feld — bei Kind-Einträgen steht dort die `users.id` des Kindes, nicht des Elternteils.

## Goals / Non-Goals

**Goals:**
- Elternteile können Einträge für ihre Kinder anlegen, bearbeiten, löschen
- Elternteile können Paarungsanfragen für Kind-Einträge stellen, bestätigen und ablehnen
- `isOwn` / `bieteIsOwn` / `sucheIsOwn` sind für Kind-Einträge ebenfalls `true`
- `GET /api/mitfahrgelegenheiten` liefert `childUserIds` damit das Frontend selbst urteilen kann
- `GET /api/dashboard` liefert Kind-Paarungen in `carpoolingConfirmed`
- Gilt unabhängig von `can_login` des Kindes

**Non-Goals:**
- Kein neues DB-Schema / keine Migration
- Elternteile sehen NICHT die Einträge anderer Kinder (nur ihre eigenen)
- Keine Push-Notification-Änderungen (Benachrichtigungen laufen weiter über den Eintrag-Owner)

## Decisions

### 1. Hilfsquery: childUserIDs statt childMemberIDs

Games/Trainings prüfen `parentHasChild(parentUserID, memberID)`. Carpooling speichert `user_id`, nicht `member_id`. Daher brauchen wir eine neue Hilfsfunktion:

```go
// childUserIDs lädt alle user_ids der Kinder eines Elternteils.
func (h *Handler) childUserIDs(ctx context.Context, parentUserID int) []int {
    rows, _ := h.db.QueryContext(ctx, `
        SELECT m.user_id FROM family_links fl
        JOIN members m ON m.id = fl.member_id
        WHERE fl.parent_user_id = ? AND m.user_id IS NOT NULL`, parentUserID)
    // ... scan rows
}

// isChildOf prüft ob targetUserID ein Kind von parentUserID ist.
func (h *Handler) isChildOf(ctx context.Context, parentUserID, targetUserID int) bool {
    var count int
    h.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM family_links fl
        JOIN members m ON m.id = fl.member_id
        WHERE fl.parent_user_id = ? AND m.user_id = ?`,
        parentUserID, targetUserID).Scan(&count)
    return count > 0
}
```

**Warum nicht claims.IsParent prüfen?** Weil auch Elternteile mit `role != "elternteil"` (z.B. ein Vorstandsmitglied das auch Elternteil ist) in `family_links` eingetragen sein können. `isChildOf()` ist die sicherere, explizite Prüfung.

### 2. childUserIds in ListResponse

```go
type ListResponse struct {
    Games        []CarpoolResponse `json:"games"`
    VehicleSeats *int              `json:"vehicleSeats"`
    ChildUserIDs []int             `json:"childUserIds"`
}
```

Das Frontend erhält damit eine vollständige Liste und kann:
- `mineMatches()` auf Kind-user_id prüfen
- Aktionsbuttons (Löschen, Anfragen, Bestätigen) für Kind-Einträge zeigen

Alternative wäre ein serverseitiges Flag `isFamilyOwn` pro Eintrag — aber `childUserIds` ist einfacher und ermöglicht dem Frontend mehr Flexibilität (z.B. Kind-Name anzeigen).

### 3. isOwn-Erweiterung in queryEntries / queryPaarungen

```go
// Einmalig vor queryEntries/queryPaarungen laden:
childIDs := h.childUserIDs(r.Context(), userID)
childIDSet := makeSet(childIDs)

// In queryEntries:
e.IsOwn = ownerID == currentUserID || childIDSet[ownerID]

// In queryPaarungen:
p.BieteIsOwn = bieteUserID == currentUserID || childIDSet[bieteUserID]
p.SucheIsOwn = sucheUserID == currentUserID || childIDSet[sucheUserID]
```

### 4. Upsert: forUserId Parameter

```go
var body struct {
    GameID     int    `json:"gameId"`
    Typ        string `json:"typ"`
    ForUserID  *int   `json:"forUserId,omitempty"`
    // ...
}
```

Wenn `ForUserID` gesetzt:
1. `isChildOf(ctx, claims.UserID, *ForUserID)` — 403 wenn false
2. `userID = *ForUserID` für alle DB-Operationen

Biete-Upsert bleibt idempotent (UPDATE wenn existiert, INSERT wenn nicht).  
Suche-Upsert: bisherige Logik (ein neuer Eintrag), mit Kind-user_id als owner.

### 5. Delete, RequestPairing, ConfirmPairing, RejectPairing

Autorisierungscheck wird überall erweitert:

```
darf handeln, wenn:
  requestingUser == entryOwner
  OR isChildOf(requestingUser, entryOwner)
```

Für Paarungen (RequestPairing): der User darf initiieren wenn er Bieter ODER Sucher ist — oder wenn einer davon sein Kind ist.

### 6. Dashboard queryCarpoolingConfirmed

```sql
WHERE p.status = 'confirmed'
  AND (mb.user_id = ? OR ms.user_id = ?
       OR mb.user_id IN (SELECT m.user_id FROM family_links fl JOIN members m ON m.id = fl.member_id WHERE fl.parent_user_id = ?)
       OR ms.user_id IN (SELECT m.user_id FROM family_links fl JOIN members m ON m.id = fl.member_id WHERE fl.parent_user_id = ?))
  AND mb.game_id = ?
```

`PartnerName` bleibt die Gegenseite aus Sicht des Elternteils/Kindes. Für Kind-Paarungen wird der Name des Fahrtpartners angezeigt (nicht der Name des Kindes) — gleiche Logik wie bisher.

### 7. FormModal: Für-wen-Selektor

Neues optionales Dropdown, nur sichtbar wenn `childUserIds.length > 0`. Optionen: `[{ userId: currentUserId, name: "Ich" }, ...children]`. Beim Speichern wird `forUserId` mitgeschickt (nur wenn Kind ausgewählt).

Kind-Namen kommen aus dem `ListResponse.childUserIds` + einer separaten Ladung der Kind-Nutzerprofile, oder einfacher: das Backend liefert `children: [{userId, name}]` statt nur IDs.

**Entscheidung:** `ChildUsers []ChildUser` statt `ChildUserIDs []int` — spart einen zusätzlichen API-Call im Frontend.

```go
type ChildUser struct {
    UserID int    `json:"userId"`
    Name   string `json:"name"`
}

type ListResponse struct {
    Games        []CarpoolResponse `json:"games"`
    VehicleSeats *int              `json:"vehicleSeats"`
    Children     []ChildUser       `json:"children"`
}
```

Frontend leitet `childUserIds` ab als `children.map(c => c.userId)`.

## Risks / Trade-offs

**[Race Condition bei gleichzeitiger Eltern-/Kind-Bearbeitung]** → Beide sehen `isOwn: true` und können gleichzeitig handeln. Der Upsert ist idempotent, Deletes und Paarungsänderungen sind Einzeloperationen — SQLite serialisiert WAL-Writes. Kein signifikantes Risiko.

**[Kind mit can_login=1 sieht elterliche Änderungen]** → Das Kind sieht sofort die Änderungen seines Elternteils (SSE via `hub.Broadcast`). Kein separater Notification-Flow nötig.

**[isChildOf-Query pro Request]** → Wird einmalig pro `List()`-Call ausgeführt und cached in `childIDSet`. Bei Einzel-Mutations (Delete, Pairing) ist es ein einzelner Count-Query — vernachlässigbar für SQLite.
