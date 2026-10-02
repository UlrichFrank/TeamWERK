# Design

## Context

- Standard-Gruppen-Kacheln werden heute im Dialog „Neues Gespräch" (`ChatPage.tsx`, `addTeamGroup`) über `GET /api/chat/team-groups/{teamId}/{kind}/members` bzw. `/api/chat/practice-groups/{id}/{kind}/members` in User-IDs aufgelöst. An `POST /api/chat/conversations` gehen nur `memberIds`, die Kachel ist danach vergessen.
- Die Auflösungs-SQL liegt in `teamGroupMemberQuery`, `allTrainersMemberQuery` (`team_groups.go`) und `practiceGroupMemberQuery` (`practice_groups.go`). Die Sichtbarkeit prüfen `canSeeTeamGroup`, `callerInTrainerCircle` und `isPracticeGroup` + `canSeePracticeGroup`. Alle drei Handler schreiben die Fallunterscheidung (Platzhalteranzahl je `kind`, Caller-Ausschluss) selbst aus.
- Einzel-Mutationen `AddMember`/`RemoveMember` (`handler.go`): nur der Ersteller, `canContactUser` beim Hinzufügen, Systemnachricht `is_system=1`, Events `chat:new-message:<id>` bzw. `chat:member-left:<id>` an `activeMembers` (+ den Entfernten).
- `members.status` kennt `aktiv|verletzt|pausiert|ausgetreten|foerderkind`. Die Kachel-Queries filtern heute gar nicht auf den Status.
- Letzte Migration: `071_chat_album`.

## Goals / Non-Goals

**Goals:**
- Eine Funktion beantwortet „wer gehört zu Kachel X", für den Anlege-Dialog wie für den Abgleich. Kachel und Abgleich können damit nicht auseinanderlaufen.
- Der Abgleich ändert nie etwas, das der Ersteller nicht gesehen und bestätigt hat.
- Kein Fehlerpfad entfernt still viele Leute (Saisonwechsel, Übergabe der Gruppe).

**Non-Goals:**
- Automatischer oder zeitgesteuerter Abgleich. Er läuft nur auf Knopfdruck.
- Abgleich für Mitteilungen (Broadcasts). Die werden beim Senden aufgelöst und haben das Problem nicht.
- Herkunft für Bestandsgruppen ableiten (siehe Entscheidung 3).
- Ein „Ausnahmen merken" für manuell hinzugefügte Personen. Sie erscheinen bei jedem Abgleich wieder in „Entfernen" und müssen abgewählt werden (siehe Risiken).

## Decisions

### 1. Eigene Tabelle `conversation_sources` statt JSON-Spalte

```sql
CREATE TABLE conversation_sources (
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    group_type      TEXT    NOT NULL CHECK (group_type IN ('team','practice')),
    ref_id          INTEGER NOT NULL,
    kind            TEXT    NOT NULL CHECK (kind IN ('trainer','spieler','eltern','alle_trainer')),
    PRIMARY KEY (conversation_id, group_type, ref_id, kind)
);
```

- Der PK dedupliziert, und `ON DELETE CASCADE` räumt beim Löschen der Gruppe mit auf.
- `ref_id` hat bewusst **keinen** FK. Er zeigt je nach `group_type` auf `teams.id` oder `kader.id` (und bei `alle_trainer` auf 0). Ein gelöschter Kader soll die Herkunft nicht still entfernen, er soll im Modal als `empty` auffallen.
- Verworfen wurde `conversations.sources_json`: Das wäre ohne CHECK, ohne Dedup und schwer abfragbar. Und `conversations` müsste für eine Spalte mit CHECK nicht neu aufgebaut werden, eine eigene Tabelle ist aber ohnehin sauberer.
- `down.sql`: `DROP TABLE conversation_sources`. Die Migration ist additiv, der Rollback per `teamwerk.prev` bleibt damit gefahrlos.

### 2. Ein Resolver für alle Kacheln

Neu in `internal/chat/group_sources.go`:

```go
type GroupSource struct{ GroupType string; RefID int; Kind string }

func validSource(s GroupSource) bool                                  // Form (400)
func (h *Handler) canSeeSource(ctx, claims, s) (bool, error)          // Sichtbarkeit
func (h *Handler) resolveSource(ctx, s, excludeUserID int) ([]TeamGroupMember, error)
```

- `resolveSource` kapselt die Platzhalter-Fallunterscheidung, die heute in `countTeamGroupMembers`, `ResolveTeamGroup`, `resolveAllTrainers` und `listPracticeGroups` dreifach steht.
- Diese Stellen werden auf den Resolver umgestellt (Zählung = `len(resolveSource(...))` bzw. derselbe Query als `COUNT(*)`). `count` in der Kachel-Liste und Abgleich lesen damit dieselbe Menge.
- `canSeeSource` delegiert an die bestehenden Prüfungen. Eine neue Sichtbarkeitsregel kommt nicht hinzu.

### 3. Bestandsgruppen: keine Ableitung

- Die Migration legt nur die Tabelle an.
- Das Modal zeigt bei leerer Herkunft die sichtbaren Kacheln mit `alreadyIn/total` als Erkennungshilfe, wählt aber nichts vor.
- Verworfen wurde, die Kachel mit der größten Überlappung vorzuwählen oder in der Migration zuzuordnen. Ein falscher Treffer sähe plausibel aus („Spieler mC1" bei einer Gruppe, die eigentlich „Spieler mC1 + Eltern mC1" war). Der folgende Abgleich würde dann alle Eltern zum Entfernen anhaken. Diesen Fehler sieht niemand, eine fehlende Vorauswahl dagegen fällt sofort auf.

### 4. Vorschau und Anwenden als zwei Routen, Anwenden rechnet neu

- Vorbild sind der H4A-Import und der Massen-Dienstregen: `preview` schreibt nichts (Eintrag in `broadcastAllowlist` mit Begründung), `apply` rechnet den Diff in der Transaktion neu.
- `apply` akzeptiert nur IDs aus dem neuen Diff, sonst `409 sync_stale`. Damit ist `apply` kein verkapptes Massen-`AddMember` für beliebige IDs. Und ein zwischen Vorschau und OK geänderter Kader führt zu einer neuen Vorschau statt zu einer stillen Abweichung.
- Beide Routen sind `POST`, weil die Kachel-Menge als Body übergeben wird. Ein `GET` mit kodierter Kachel-Liste in der Query wäre möglich, aber unhandlich. `POST` für eine schreibfreie Vorschau hat im Projekt Vorbilder.
- Ohne `sources` im Body nimmt `preview` die gespeicherte Herkunft. Das Modal braucht deshalb kein zusätzliches Feld in `GET /api/chat/conversations`.

### 5. Sperre bei leerer oder unsichtbarer Kachel

- Eine Kachel, die zu null Personen auflöst oder für den Ersteller nicht sichtbar ist, setzt `blocked = true` und leert den Diff, `apply` antwortet dann `409 sync_blocked`.
- Die zwei realen Auslöser sind der Saisonwechsel (neuer Kader noch leer → jedes Mitglied stünde in „Entfernen") und `TransferOwnership` an jemanden ohne Zugriff auf das Team.
- Verworfen wurde, eine solche Kachel stillschweigend zu überspringen. Dann stünden ihre Leute ebenfalls in „Entfernen", also genau die Massenentfernung, die die Sperre verhindern soll.

### 6. Ausgetretene filtern im Resolver, nicht im Abgleich

- Den Filter `m.status <> 'ausgetreten'` bekommen alle Kachel-Queries, bei `eltern` auf das Kind (`JOIN members` über `fl.member_id`).
- Ein Filter nur im Abgleich hätte zur Folge: Der Anlege-Dialog nimmt einen Ausgetretenen auf, und der erste Abgleich schlägt sofort vor, ihn zu entfernen.
- `<>` statt `= 'aktiv'` folgt der Statusfilter-Falle aus den Video-Gotchas (`foerderkind`, `verletzt`, `pausiert` gehören dazu).

### 7. Mutation in einer Transaktion, Events nach dem Commit

- `apply` macht alles in einer `tx`: Herkunft löschen und neu schreiben, Mitglieder reaktivieren oder einfügen, `left_at` setzen, Systemnachrichten schreiben.
- Die Events gehen gebündelt nach dem Commit raus, je Empfänger höchstens ein `chat:new-message` und ein `chat:member-left`, kein Event je Person.
- Die Systemnachricht-Texte sind identisch mit `AddMember`/`RemoveMember` (`wurde hinzugefügt`/`wurde entfernt`). Die Spec „Systemnachrichten-Konsistenz" gilt damit ohne Änderung.

### 8. Frontend

- `ConversationSyncModal.tsx` (neu) übernimmt Modal-Kopf und Klassen wie `ConversationParticipantsModal` (`MODAL_TITLE`, `BTN_PRIMARY`, `BTN_SECONDARY`, `useDialogA11y`, `useEscapeKey`).
- Der Knopf „Aktualisieren" (`RefreshCw`) steht im Kopf des Teilnehmer-Modals neben dem Stift, nur für `isOwner`.
- Abgewählte Personen hält das Modal als `Set<number>` je Liste. Nach einem Neuladen der Vorschau werden nur die IDs weiter abgewählt, die noch vorkommen.
- Die Kachel-Bezeichnung (`label`) baut der Server, damit Modal und Anlege-Dialog nicht zwei Formatierungen pflegen. Der Anlege-Dialog behält seine eigene, eine Vereinheitlichung ist nicht Teil des Changes.
- `ChatPage.tsx` hält in „Neues Gespräch" die gewählten Kacheln schon als `pickedTags` und schickt sie zusätzlich als `sources` mit. Wird eine Kachel durch Entfernen einzelner Chips „ausgedünnt", bleibt sie Herkunft. Die entfernten Personen erscheinen dann beim ersten Abgleich in „Hinzufügen", das ist gewollt und abwählbar.
- Live-Updates: `ChatPage` lauscht bereits auf `chat:new-message`/`chat:member-left` und lädt die Konversation nach, eine neue Abonnement-Logik ist nicht nötig.

## Risks / Trade-offs

- **Manuell Hinzugefügte tauchen bei jedem Abgleich in „Entfernen" auf.** Das ist ehrlich (die Person gehört zu keiner Kachel), kostet aber jedes Mal einen Klick. Gegenmittel wäre eine Ausnahmeliste je Gruppe. Bewusst nicht im Change, bei Bedarf als Folge-Change.
- **Verhaltensänderung bei den Kacheln:** Ausgetretene fallen aus „Neues Gespräch" und aus der Kachel-Zahl `count`. Gewollt, aber sichtbar: Eine Kachel kann nach dem Deploy eine kleinere Zahl zeigen.
- **`TransferOwnership`:** Der neue Ersteller erbt die Herkunft. Sieht er sie nicht, ist der Abgleich gesperrt, bis er die Kachel entfernt. Das ist bewusst die sichere Richtung.
- **Wettlauf zweier Sitzungen desselben Erstellers:** `apply` rechnet neu und lehnt veraltete IDs ab. Die schlimmste Folge ist ein 409 mit neuer Vorschau.
