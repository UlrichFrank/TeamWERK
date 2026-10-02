# Proposal

## Why

Chat-Gruppen werden meist aus Standard-Gruppen-Kacheln angelegt („Spieler mC1", „Eltern mC1", „Alle Trainer", Übungsgruppen). Die Kachel wird dabei aber nur einmal im Browser in User-IDs aufgelöst. Ändert sich danach der Kader (neue Spieler, Wechsel, Austritte), weicht die Gruppe still von der Mannschaft ab. Heute bleibt dem Ersteller dann nur, jede Person einzeln zu suchen und hinzuzufügen oder zu entfernen. Und welche Kachel die Gruppe einmal war, weiß der Server gar nicht, `conversations` speichert keine Herkunft.

## What Changes

- **Neue Tabelle `conversation_sources`** (additive Migration `072`): je Gruppe 0–n Herkunfts-Kacheln `(group_type, ref_id, kind)`. Die Migration befüllt nichts und rät nichts. Bestandsgruppen haben keine Herkunft, bis der Ersteller sie einmal festlegt.
- **Anlegen speichert die Herkunft:** `POST /api/chat/conversations` mit `type=group` nimmt optional `sources: [{groupType, refId, kind}]` an. Das Frontend schickt die Kacheln mit, die im Dialog „Neues Gespräch" gewählt wurden.
- **Neuer Abgleich in zwei Phasen**, nur für den Ersteller (`conversations.created_by`):
  - `POST /api/chat/conversations/{id}/sync/preview` mit `{sources}`: liefert die Soll-Menge als Diff, `add` (in der Soll-Menge, aber kein aktives Mitglied) und `remove` (aktives Mitglied, aber nicht mehr in der Soll-Menge, nie der Ersteller). Dazu je Kachel die Überlappung („11 von 12 schon drin"). Schreibt nichts.
  - `POST /api/chat/conversations/{id}/sync/apply` mit `{sources, addUserIds, removeUserIds}`: rechnet den Diff neu, akzeptiert nur IDs, die darin vorkommen, speichert die Herkunft und wendet die Auswahl in einer Transaktion an. Systemnachrichten und SSE-Events verhalten sich wie beim Hinzufügen und Entfernen einzelner Personen.
- **Ausgetretene fallen aus den Standard-Gruppen:** Die Auflösung der Kacheln (Anlegen und Abgleich gleichermaßen) filtert Mitglieder mit `members.status = 'ausgetreten'`. Bei `eltern` gilt der Status des Kindes. Bisher blieb ein ausgetretenes Mitglied, das noch im Kader der aktiven Saison steht, in der Kachel.
- **UI im Teilnehmer-Modal** (`ConversationParticipantsModal`): Der Ersteller sieht bei Gruppen den Knopf „Aktualisieren". Er öffnet ein eigenes Modal. Dort steht ein kurzer Erklärtext, darunter die editierbaren Herkunfts-Kacheln (bei Bestandsgruppen leer, mit Überlappungsangabe und ohne Vorauswahl) und zwei Checkbox-Listen „Hinzufügen" und „Entfernen" (alles vorausgewählt). „OK" wendet die Auswahl an, „Abbrechen" verwirft sie.

## Capabilities

### New Capabilities
- `chat-gruppe-aktualisieren`: Herkunft einer Chat-Gruppe speichern, Abgleich gegen die Standard-Gruppen (Vorschau + Anwenden), Berechtigung, Aktualisieren-Modal.

### Modified Capabilities
- `chat-team-groups`: Die Auflösung der Mannschafts-, „Alle Trainer"- und Übungsgruppen-Kacheln schließt ausgetretene Mitglieder (bei `eltern`: ausgetretene Kinder) aus. Die Zahl `count` der Kachel folgt derselben Menge.

## Impact

- **Backend:** `internal/chat` (neue Datei `group_sync.go`, `createGroup` nimmt `sources`, gemeinsamer Resolver für die Kacheln aus `team_groups.go`/`practice_groups.go`), `internal/app/router.go` (zwei Routen im Authenticated-Tier), Migration `072_chat_conversation_sources`.
- **Gates:** Beide neuen Routen sind `POST`. `apply` broadcastet über die Chat-Events. `preview` ist schreibfrei und kommt mit Begründung in die `broadcastAllowlist`. Die Objektrechte-Matrix braucht Fixtures für die neuen `{id}`-Routen (Gruppe von Nutzer B, Aufruf durch Nutzer A → 403/404).
- **Frontend:** `web/src/components/ConversationParticipantsModal.tsx`, neues `ConversationSyncModal.tsx`, Dialog „Neues Gespräch" in `ChatPage.tsx` (schickt `sources` mit).
- **Berechtigungsmodell:** Kein neues Tier. Recht = Ersteller der Konversation (wie `RemoveMember`/`AddMember`). Die Kacheln selbst werden mit den bestehenden Sichtbarkeitsregeln des Erstellers aufgelöst (`canSeeTeamGroup`, `canSeePracticeGroup`, Zugriffskreis). Hinzugefügte Personen durchlaufen `canContactUser`.
- **RAM/Last:** Vernachlässigbar, wenige Abfragen je Abgleich und nur auf Knopfdruck.
