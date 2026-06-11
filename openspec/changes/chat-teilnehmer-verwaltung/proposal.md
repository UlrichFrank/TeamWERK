## Why

Gruppen-Chats haben heute fragmentierte Verwaltung:

1. **Kein Einblick in die Teilnehmerliste.** Das `Users`-Icon im Chat-Header zeigt nur eine Zahl, ist nicht klickbar. Wer in der Gruppe ist, erfährt man nur indirekt über Nachrichten.
2. **Kein Entfernen, kein Umbenennen.** Der Ersteller kann zwar via separatem `UserPlus`-Icon Mitglieder hinzufügen — aber niemand kann ein Mitglied wieder entfernen oder den Gruppennamen ändern. Beides nur über DB-Eingriff möglich.
3. **`AddMember` ist still.** Wer hinzugefügt wird, taucht plötzlich ohne Systemnachricht in der Gruppe auf, während Self-Leave eine Systemnachricht erzeugt — inkonsistent.
4. **Creator-Exit ist ein Sackgassen-Szenario.** Verlässt der Ersteller die Gruppe, bleibt `conversations.created_by` auf ihm — niemand kann danach mehr Teilnehmer verwalten oder umbenennen. Heute lediglich theoretisch unschön; mit den neuen Verwaltungsfunktionen wird das spürbar.

## What Changes

- **NEU: Teilnehmer-Modal** (`ConversationParticipantsModal`) öffnet sich beim Klick auf das `Users`-Icon im Chat-Header. View-Modus für alle Mitglieder zeigt die Teilnehmerliste inkl. Ersteller-Pill. Ersteller sieht zusätzlich einen Pencil-Button neben dem `X`, der in den Edit-Modus „Teilnehmer bearbeiten" wechselt — analog `EventInfoModal`. Im Edit-Modus: Gruppenname ändern, Teilnehmer entfernen (`✕` je Eintrag), Teilnehmer hinzufügen (eingebettete Suche, ersetzt das eigenständige `AddMemberModal`).
- **NEU: `DELETE /api/chat/conversations/{id}/members/{userId}`** — Creator-only, setzt `left_at` (soft), erzeugt Systemnachricht „X wurde entfernt", broadcastet `chat:member-left:<id>` an alle aktiven Mitglieder plus den entfernten User. Self-Removal über diesen Endpoint ist verboten — Ersteller nutzen den Creator-Exit-Flow.
- **NEU: `PUT /api/chat/conversations/{id}`** — Creator-only, akzeptiert `{ name }`, persistiert in `conversations.name`, Systemnachricht „… hat die Gruppe in 'Y' umbenannt", broadcastet `chat:conv-updated:<id>`.
- **NEU: `POST /api/chat/conversations/{id}/transfer-ownership`** — Creator-only, akzeptiert `{ newOwnerId }`. Empfänger muss aktives Mitglied sein, kein Self-Transfer. Setzt `conversations.created_by = newOwnerId`, Systemnachricht „… hat die Verwaltung an Y übergeben", broadcastet `chat:conv-updated:<id>`.
- **NEU: `DELETE /api/chat/conversations/{id}/everyone`** — Creator-only, hart löschen für die ganze Gruppe. Setzt alle `left_at`, löscht `conversations`-Zeile (Cascade auf `messages`, `message_reactions`), broadcastet **neues** Event `chat:conv-deleted:<id>` an alle ehemals aktiven Mitglieder.
- **NEU: Creator-Exit-Modal** (`CreatorExitChoiceModal`) ersetzt den direkten `LogOut`-Confirm, wenn der Klickende der Ersteller ist. Bietet „Verwaltung übergeben an…" (Mitglied-Picker) oder „Gruppe für alle löschen" (mit zweitem Confirm-Step).
- **Konsistenz: `AddMember` ergänzt Systemnachricht** „X wurde hinzugefügt", damit alle vier Mutations-Aktionen (add, remove, leave, rename, transfer, delete-for-all) eine sichtbare Historie hinterlassen.
- **Header-Aufräumung**: Eigenständiges `UserPlus`-Icon im Chat-Header entfällt — die Funktion lebt im Edit-Modus des Teilnehmer-Modals. `LogOut` bleibt im Header (Self-Aktion, gilt für alle Mitglieder).
- **Frontend reagiert auf neue SSE-Events**: `chat:conv-updated:<id>` lädt die Conversation neu (Name, Ersteller, Teilnehmer); `chat:conv-deleted:<id>` entfernt sie aus der Liste und zeigt einen Toast, falls gerade aktiv geöffnet.

## Capabilities

### Modified Capabilities

- `chat-konversationen` — Erweitert um Member-Verwaltung (Add/Remove/Rename), Ownership-Transfer, Hard-Delete-für-alle, sowie Systemnachrichten-Konsistenz für alle Gruppen-Mutationen.

## Impact

- `internal/chat/handler.go` (~180 Zeilen): vier neue Handler (`RemoveMember`, `UpdateConversation`, `TransferOwnership`, `DeleteConversationForEveryone`), `AddMember` um Systemnachricht ergänzen
- `cmd/teamwerk/main.go` (~5 Zeilen): vier neue Routes registrieren
- `web/src/pages/ChatPage.tsx` (~150 Zeilen): `UserPlus`-Header-Icon entfernen, `Users`-Icon klickbar machen, `AddMemberModal` durch `ConversationParticipantsModal` ersetzen, `CreatorExitChoiceModal` einbinden, `useChatEvents` um neue Event-Typen erweitern
- `web/src/components/ConversationParticipantsModal.tsx` (NEU, ~200 Zeilen): View/Edit-Modal nach Vorbild `EventInfoModal`
- `web/src/components/CreatorExitChoiceModal.tsx` (NEU, ~120 Zeilen): Übergeben-vs-Löschen-Auswahl mit Confirm-Step
- `web/src/hooks/useChatEvents.ts`: neue Event-Typen `chat:conv-updated` und `chat:conv-deleted` propagieren
- **Keine DB-Migration** — Schema bleibt unverändert. Soft-Delete nutzt bestehende `conversation_members.left_at`-Spalte; Cascade auf `messages` greift über bestehende FK.
- **Direct-Konversationen unverändert** — alle neuen Endpoints lehnen `type='direct'` mit HTTP 400 ab.
