## Why

Wer bei einem Termin (`/termine`) mit „Absagen" oder „Vielleicht" antwortet und dabei einen Grund eingibt (Modal aus [[rsvp-reason-modal]]), sieht diesen Grund danach **nirgends mehr**. Auf der Karte in der Terminliste steht nur der Button-Status (rot/gelb hervorgehoben), der eingegebene Text taucht weder für einen selbst noch für Eltern (die für ihre Kinder geantwortet haben) wieder auf.

Auf der Detailseite (`/termine/training/{id}`, `/termine/game/{id}`) existiert die Teilnahme-Tabelle mit Reason-Anzeige per MessageCircle-Popover — aber sie ist inkonsistent gate:
- **Trainings** (`GET /api/training-sessions/{id}/attendances`, `handler.go:1541`): Reason **nur für Trainer** (`isTrainerLike`), d.h. weder das Mitglied selbst noch der Elternteil sieht seinen eigenen bzw. den Grund des Kindes zurück.
- **Games** (`GET /api/games/{id}/attendances`, `handler.go:2938`): **überhaupt kein Gate** — jeder mit Team-Access sieht alle Reasons.

Die andere Route in games (`ListGameResponses` → `/api/games/{id}/participants`, `handler.go:2226`) hat den korrekten 3-Wege-Gate bereits — man weiß also im Codebase schon, wie's aussehen soll. Die anderen zwei Endpoints müssen nachziehen und die Listen brauchen die Info zusätzlich, damit man nicht erst in die Detailseite scrollen muss.

Zielbild in einem Satz: **Trainer sehen alle Reasons, Mitglieder nur ihre eigene, Eltern zusätzlich die ihrer Kinder — konsistent auf Liste und Detail, konsistent zwischen Spiel und Training.**

## What Changes

### Backend

- **`GET /api/games/my` und `GET /api/training-sessions/my`** (Listen-Endpoints, gefüttert von `/termine`):
  - `gameListItem` / `sessionListItem` bekommen ein neues Feld `MyReason *string \`json:"my_reason,omitempty"\``. Wird aus `game_responses.reason` / `training_responses.reason` mitselektiert und nur bei nicht-leerer Antwort auf **explizite** RSVP befüllt (also nicht bei impliziter Default-RSVP).
  - `childRSVP`-Struct in beiden Domänen bekommt `Reason *string \`json:"reason,omitempty"\``. Wird aus `gr.reason`/`tr.reason` in den Children-Aggregations-Queries (`attachChildrenRSVPToGames`, entsprechendes Training-Pendant) mitselektiert. Nur für gelinkte Kinder des anfragenden Elternteils — die Query stellt das über `family_links` bereits sicher.
- **`GET /api/training-sessions/{id}/attendances`** (`internal/trainings/handler.go`, GetAttendances): `canSeeReason` von `isTrainerLike` auf `isTrainerLike || own || parent_of_child` erweitern — spiegelbildlich zum bereits korrekten Muster in `GetSession`/`sessionResponse` (`handler.go:1253-1255`). `memberID` und `childMemberIDs`-Map werden analog zu dort vorab geladen.
- **`GET /api/games/{id}/attendances`** (`internal/games/handler.go`, GetAttendances): heute leaky (kein Gate), zieht dieselbe `trainer || own || parent_of_child`-Regel ein. `memberID` und `childMemberIDs` werden vorab geladen; existierendes Autorisierungs-Gate (`canRecordGameAttendance`) bleibt unverändert — das regelt nur den Zugriff auf den Endpoint an sich, nicht die Reason-Sichtbarkeit pro Zeile.

Kein Schema-Change. Kein neuer Endpoint. Keine Änderung an POST-Semantik (Antworten mit Grund).

### Frontend

- **`web/src/pages/TerminePage.tsx`**:
  - Interfaces `Session` und `Game` bekommen `my_reason?: string`; `children_rsvp[].reason?: string` ergänzen.
  - Rendering: unterhalb des RSVP-Button-Blocks (bei aktivem `my_rsvp` in `declined` oder `maybe`), wenn `my_reason` gesetzt: `<p className="text-xs text-brand-text-muted mt-1"><MessageCircle className="w-3 h-3 inline mr-1"/>{my_reason}</p>`.
  - Analog pro Kind-Zeile: wenn Kind explizit abgesagt/vielleicht **mit** Grund, den Grund darunter.
  - Long-Text: bewusst kein Truncate, weil die Reason-Modale heute schon eine Zeile fordern und die Karten selten mehr als 40–60 Zeichen bekommen; wenn's zu breit wird, ist das ein Follow-up.
- **Kein Change an TermineDetailPage** nötig — die `ParticipantRow`-Komponente rendert `row.reason` bereits ohne eigenes Gate; die Server-Änderung schaltet dort automatisch die richtigen Zeilen frei.

### Nicht enthalten (bewusst)

- Reason nachträglich editieren, ohne den Status zu ändern → separater Change.
- Reason-Länge-Warnung / Truncate-mit-Expand → optischer Feinschliff, kein Feature-Gap.
- Sichtbarkeit von Reasons in E-Mail-/Push-Benachrichtigungen → nicht adressiert; hier geht's nur um UI-Views.

## Capabilities

### New Capabilities

- `rsvp-reason-visibility`: Regelt, welche Nutzer den freiwilligen/pflichtweisen RSVP-Grund eines Termins wieder sehen. Trainer (Rolle `trainer` global oder `admin`) sehen alle Reasons in Liste und Detail; das antwortende Mitglied sieht seinen eigenen Grund; Elternteile sehen den Grund für die per `family_links` verknüpften Kinder. Für Fremd-Reasons bleibt das Feld `null`. Gilt in `/api/games/my`, `/api/training-sessions/my`, `/api/games/{id}/attendances`, `/api/training-sessions/{id}/attendances`.

### Modified Capabilities

(keine im Sinn von OpenSpec-Diffs — die verwandten Capabilities [[rsvp-reason-modal]], [[game-attendance]], [[training-attendance]], [[game-rsvp]], [[training-rsvp]] behalten ihre bestehenden Requirements; die Sichtbarkeitsregel ist ein orthogonaler Cross-Cut.)

## Impact

- **`internal/games/handler.go`**:
  - `gameListItem`-Struct: neues Feld `MyReason`.
  - `childRSVP`-Struct: neues Feld `Reason`.
  - `ListMyGames`-SQL (Zeile 1943): zusätzlicher `SELECT reason FROM game_responses …` als Scan-Ziel für explicit-only.
  - `attachChildrenRSVPToGames`-SQL (Zeile 2551, 2564): `gr.reason` in die SELECT-Liste + Scan.
  - `GetAttendances` (Zeile 2793): `memberID` und `childMemberIDs` vorab laden; im Scan-Loop den 3-Wege-Gate anwenden.
- **`internal/trainings/handler.go`**:
  - `sessionListItem`-Struct: neues Feld `MyReason`.
  - `childRSVP`-Struct: neues Feld `Reason`.
  - `ListSessions`-SQL (Zeile 1011, 1167): zusätzlicher `reason`-Scan für explicit-only.
  - Children-Aggregation (in der Nähe von 1090): `tr.reason` mitziehen.
  - `GetAttendances` (Zeile 1406): 3-Wege-Gate wie in `GetSession`.
- **`web/src/pages/TerminePage.tsx`**: Interface-Erweiterungen + zwei Render-Blöcke (eigene Karte, Kind-Zeile). Icon aus `lucide-react` (MessageCircle) — bereits an anderer Stelle importiert.
- **Tests**: siehe Test-Anforderungen unten. Coverage der bestehenden RSVP-Tests wird gestreift (Struct-Änderungen), aber die Verträge bleiben rückwärtskompatibel (nur additive Felder mit `omitempty`).
- **Migration/Live-Updates**: keine Datenbank-Migration. SSE-Broadcast bleibt unverändert (`rsvp-changed` triggert Reload; `my_reason` fließt automatisch mit).
- **Performance**: `ListMyGames`/`ListSessions` bekommen einen zusätzlichen Column-Scan pro Row (`reason` ist im Response schon per member_id gefiltert, keine neue Subquery). `GetAttendances` bekommt eine zusätzliche Vorab-Query für `family_links` (nur wenn `IsParent`) — analog zu den bereits existierenden `GetSession`/`ListGameResponses`-Handlern, dort seit langem produktiv.
- **Sicherheit**: die Änderung ist eine **Verschärfung** im games/attendances-Fall (heute leakt der Endpoint alle Reasons an jeden mit Team-Access) und eine **Lockerung** im trainings/attendances-Fall (heute sieht nur Trainer, künftig auch eigene und Eltern-Kind-Zeilen). Die Regel gleicht sich damit an die bestehende Praxis in `ListGameResponses` und `sessionResponse` an — Konsistenz nach innen.
- **Rollout-Reihenfolge**: Backend zuerst (additive Felder, brechen nichts). Frontend danach (nutzt die neuen Felder). Wenn ein einzelner Deploy alles auf einmal ausrollt (üblich für TeamWERK per `make deploy`), ist die Reihenfolge egal — das Frontend fällt auf `undefined` zurück, wenn das Backend die Felder noch nicht liefert.

## Test-Anforderungen

| Route / Ort | Testname | Erwarteter Status / Invariante |
|---|---|---|
| `GET /api/games/my` | `TestListMyGames_MyReason_Populated_When_RespondedWithReason` | User hat abgesagt mit Reason „Arbeit" → `my_reason = "Arbeit"` im Response. |
| `GET /api/games/my` | `TestListMyGames_MyReason_Absent_When_DefaultRsvp` | User hat nicht explizit geantwortet, `rsvp_default_players=confirmed` → `my_rsvp="confirmed"`, `my_reason` **nicht** im JSON (omitempty). |
| `GET /api/games/my` | `TestListMyGames_ChildrenReason_ForParent` | Elternteil, Kind hat mit Reason abgesagt → `children_rsvp[i].reason` = Kind-Reason. |
| `GET /api/games/my` | `TestListMyGames_ChildrenReason_OmittedWhenEmpty` | Kind ohne explizite Antwort → `reason` nicht im JSON. |
| `GET /api/training-sessions/my` | `TestListSessions_MyReason_Populated_When_RespondedWithReason` | wie oben, für Training. |
| `GET /api/training-sessions/my` | `TestListSessions_MyReason_Absent_When_DefaultRsvp` | wie oben. |
| `GET /api/training-sessions/my` | `TestListSessions_ChildrenReason_ForParent` | wie oben. |
| `GET /api/games/{id}/attendances` | `TestGetGameAttendances_Reason_Trainer_SeesAll` | User mit `HasFunction("trainer")`: alle Zeilen mit `reason`-Text bekommen `reason` befüllt. |
| `GET /api/games/{id}/attendances` | `TestGetGameAttendances_Reason_Member_SeesOwn` | Regulärer Kader-Spieler ohne Trainer-Funktion: eigene Zeile hat `reason`, alle anderen `reason=null`. |
| `GET /api/games/{id}/attendances` | `TestGetGameAttendances_Reason_Parent_SeesChild` | Elternteil (ohne eigene Team-Membership): Zeile des verknüpften Kindes hat `reason`, alle anderen `null`. |
| `GET /api/games/{id}/attendances` | `TestGetGameAttendances_Reason_Foreigner_Hidden` | User mit Team-Access aber kein Trainer, kein Eltern-Link, keine eigene Zeile → alle `reason=null`. Regressionstest gegen das heutige Leak. |
| `GET /api/training-sessions/{id}/attendances` | `TestGetTrainingAttendances_Reason_Trainer_SeesAll` | wie oben, für Training. |
| `GET /api/training-sessions/{id}/attendances` | `TestGetTrainingAttendances_Reason_Member_SeesOwn` | wie oben. |
| `GET /api/training-sessions/{id}/attendances` | `TestGetTrainingAttendances_Reason_Parent_SeesChild` | wie oben. |
| `GET /api/training-sessions/{id}/attendances` | `TestGetTrainingAttendances_Reason_Foreigner_Hidden` | User ohne Trainer-Funktion, ohne Kind-Link, ohne eigene Zeile → alle `reason=null`. |
| Frontend `TerminePage` | Component-Test | Card zeigt `my_reason` unterhalb der RSVP-Buttons, wenn Feld gesetzt; kein Element wenn `my_reason` nicht im Payload. |
| Frontend `TerminePage` | Component-Test | Kind-Zeile zeigt Kind-Reason, wenn `children_rsvp[i].reason` gesetzt. |

Alle Backend-Tests laufen gegen die Fixtures aus `internal/testutil/` (`CreateUser`, `CreateMember`, `CreateGame`, `CreateKader`, `CreateSeason`, plus `INSERT INTO family_links` inline). Kein Mock der DB — direkte SQLite-Fixture-DB.

## Migration-Kontrakt

Additive JSON-Felder mit `omitempty`. Bestehende Frontend-Clients (Web/PWA), die die neuen Felder nicht kennen, verhalten sich unverändert. Umgekehrt zeigt der neue Frontend-Code die Felder nur an, wenn sie im Payload sind — d.h. ein Backend, das die Felder noch nicht ausrollt, verursacht keinen Fehler.

## Rollback

Reines Code-Revert. Keine Datenbank-Rückabwicklung, keine Config-Änderung, keine Session-Invalidation. `git revert` reicht.
