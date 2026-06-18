## 1. Backend: Neue Endpoints

- [x] 1.1 `RemoveMember` in `internal/chat/handler.go`: `DELETE /api/chat/conversations/{id}/members/{uid}` — Creator-Check, `type='group'`-Check, Selbst-Removal verbieten (HTTP 400 falls `uid == claims.UserID`), `UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP WHERE conversation_id=? AND user_id=? AND left_at IS NULL`, Systemnachricht „wurde entfernt" mit `sender_id = uid`, SSE `chat:member-left:<id>` an alle aktiven Mitglieder + entfernten User
- [x] 1.2 `UpdateConversation` in `internal/chat/handler.go`: `PUT /api/chat/conversations/{id}` — Creator-Check, `type='group'`-Check, `{ name }`-Body (1..100 Zeichen, getrimmt), `UPDATE conversations SET name = ? WHERE id = ?`, Systemnachricht „hat die Gruppe in 'Y' umbenannt" mit `sender_id = creator`, SSE `chat:conv-updated:<id>` an alle aktiven Mitglieder
- [x] 1.3 `TransferOwnership` in `internal/chat/handler.go`: `POST /api/chat/conversations/{id}/transfer-ownership` — Creator-Check, `{ newOwnerId }`-Body, neuer Owner muss aktives Mitglied sein (`isMember + left_at IS NULL`), kein Self-Transfer (HTTP 400), `UPDATE conversations SET created_by = ? WHERE id = ?`, Systemnachricht „hat die Verwaltung an Y übergeben" mit `sender_id = alter Owner` und neuem Owner-Namen interpoliert, SSE `chat:conv-updated:<id>`
- [x] 1.4 `DeleteConversationForEveryone` in `internal/chat/handler.go`: `DELETE /api/chat/conversations/{id}/everyone` — Creator-Check, `type='group'`-Check, vor Hard-Delete Liste aktiver Member-IDs einsammeln, `DELETE FROM conversations WHERE id = ?` (Cascade), SSE neues Event `chat:conv-deleted:<id>` an alle eingesammelten User-IDs
- [x] 1.5 Konsistenz-Patch: bestehender `AddMember`-Handler bekommt nach erfolgreichem INSERT/UPDATE eine Systemnachricht „wurde hinzugefügt" mit `sender_id = body.UserID`
- [x] 1.6 Routes in `cmd/teamwerk/main.go` registrieren (vier neue Zeilen unter den bestehenden `chat`-Routes in der Authenticated-Gruppe)

## 2. Backend: Tests

- [x] 2.1 `internal/chat/handler_test.go` — Test `TestRemoveMember_CreatorRemovesMember`: Gruppe mit 3 Mitgliedern, Owner entfernt User B → `left_at` ist gesetzt, System-Message vorhanden, Antwort 204
- [x] 2.2 `internal/chat/handler_test.go` — Test `TestRemoveMember_NonCreatorForbidden`: Nicht-Owner versucht Remove → 403
- [x] 2.3 `internal/chat/handler_test.go` — Test `TestRemoveMember_SelfRejected`: Owner versucht sich selbst zu entfernen → 400
- [x] 2.4 `internal/chat/handler_test.go` — Test `TestUpdateConversation_RenameSuccess`: Owner setzt neuen Namen → DB-Wert geändert, Systemnachricht „in 'Y' umbenannt"
- [x] 2.5 `internal/chat/handler_test.go` — Test `TestUpdateConversation_EmptyNameRejected`: leerer Name → 400, kein Update
- [x] 2.6 `internal/chat/handler_test.go` — Test `TestTransferOwnership_HappyPath`: Owner überträgt an Mitglied B → `created_by` ist nun B, Systemnachricht vorhanden, alter Owner ist noch Mitglied
- [x] 2.7 `internal/chat/handler_test.go` — Test `TestTransferOwnership_RecipientNotMember`: Übertragung an Außenstehenden → 400
- [x] 2.8 `internal/chat/handler_test.go` — Test `TestTransferOwnership_NonCreatorForbidden`: Mitglied versucht Übertragung → 403
- [x] 2.9 `internal/chat/handler_test.go` — Test `TestDeleteForEveryone_HardDelete`: Owner ruft `everyone` → `conversations`-Zeile weg, Cascade leert `messages`, `conversation_members`
- [x] 2.10 `internal/chat/handler_test.go` — Test `TestDeleteForEveryone_NonCreatorForbidden`: Mitglied versucht für-alle-Delete → 403
- [x] 2.11 `internal/chat/handler_test.go` — Test `TestAddMember_EmitsSystemMessage`: bestehender AddMember-Test um Assertion erweitern, dass „wurde hinzugefügt"-Systemnachricht existiert

## 3. Frontend: ConversationParticipantsModal

- [x] 3.1 `web/src/components/ConversationParticipantsModal.tsx` anlegen: View-State zeigt Teilnehmerliste mit Ersteller-Pill, Edit-State nur für Owner (`createdBy === user.id`)
- [x] 3.2 Pencil-Button neben `X` im Header (nur Owner sichtbar), Klick toggelt `editing`, Titel wechselt zwischen „Teilnehmer" und „Teilnehmer bearbeiten" — Vorbild `EventInfoModal.tsx`
- [x] 3.3 Edit-Modus: Sektion „Gruppenname" mit Input, lokaler `draftName`-State, onBlur/Enter → `PUT /chat/conversations/{id}` (nur falls verändert), optimistisches Update + Rollback bei Fehler
- [x] 3.4 Edit-Modus: Sektion „Aktuelle Teilnehmer" — Liste mit `✕`-Button pro Eintrag (außer Owner-Selbst), Klick → `DELETE /chat/conversations/{id}/members/{uid}`, lokale Liste sofort aktualisieren
- [x] 3.5 Edit-Modus: Sektion „Hinzufügen" — Suchfeld + Ergebnisliste aus `GET /chat/users?q=…` gefiltert auf Nicht-Mitglieder, Klick → `POST /chat/conversations/{id}/members`, lokale Liste sofort aktualisieren
- [x] 3.6 Modal nutzt `useEscapeKey` und reagiert auf SSE `chat:conv-updated` und `chat:member-left` für Live-Sync mit anderem Owner-Gerät
- [x] 3.7 In `ChatPage.tsx`: `Users`-Icon im Chat-Header umbauen zu `<button onClick={() => setShowParticipants(true)}>` mit `aria-label="Teilnehmer anzeigen"`
- [x] 3.8 In `ChatPage.tsx`: eigenständigen `UserPlus`-Button und `showAddMember`-State + `AddMemberModal`-Render entfernen
- [x] 3.9 In `ChatPage.tsx`: `<ConversationParticipantsModal>` rendern wenn `showParticipants && activeConv?.type === 'group'`

## 4. Frontend: CreatorExitChoiceModal

- [x] 4.1 `web/src/components/CreatorExitChoiceModal.tsx` anlegen: Radio-Auswahl zwischen „Verwaltung übergeben an…" (mit Mitglied-Dropdown gefüllt aus `activeConv.members` ohne Owner-Selbst) und „Gruppe für alle löschen"
- [x] 4.2 „Bestätigen"-Button bei „Übergeben": ruft sequenziell `POST /chat/conversations/{id}/transfer-ownership` und `DELETE /chat/conversations/{id}/members/me`
- [x] 4.3 „Bestätigen"-Button bei „Für alle löschen": zweiter `window.confirm`-Step („Diese Aktion löscht alle Nachrichten endgültig. Fortfahren?") → `DELETE /chat/conversations/{id}/everyone`
- [x] 4.4 In `ChatPage.tsx`: `leaveGroup`-Handler verzweigt — `if (activeConv.createdBy === user.id) setShowCreatorExit(true)` sonst bestehender `window.confirm`-Pfad
- [x] 4.5 Nach Bestätigung Modal schließen, `activeConv` auf null, `loadConversations()` neu laden

## 5. Frontend: SSE-Events

- [x] 5.1 `web/src/hooks/useChatEvents.ts`: neue Event-Typen `chat:conv-updated:<id>` und `chat:conv-deleted:<id>` durchreichen (Pattern-Match bei Doppelpunkt + id)
- [x] 5.2 In `ChatPage.tsx`: bei `chat:conv-updated:<id>` die einzelne Conversation aus `/chat/conversations` neu fetchen und im `conversations`-State + ggf. `activeConv` ersetzen
- [x] 5.3 In `ChatPage.tsx`: bei `chat:conv-deleted:<id>` Conversation aus `conversations[]` entfernen, falls `activeConv?.id === id` → Toast „Die Gruppe wurde gelöscht", `setActiveConv(null)`

## 6. Verifikation

- [ ] 6.1 Manuell: Als Mitglied (Nicht-Owner) auf `Users`-Icon klicken → Modal öffnet sich im View-Modus, kein Pencil-Button sichtbar
- [ ] 6.2 Manuell: Als Owner auf `Users`-Icon klicken → Pencil-Button sichtbar, Klick wechselt in Edit-Modus, Titel „Teilnehmer bearbeiten"
- [ ] 6.3 Manuell: Mitglied im Edit-Modus entfernen → Liste aktualisiert sich, im Nachrichten-Bereich erscheint Systemnachricht „X wurde entfernt", entfernter User (zweite Browser-Session) sieht die Gruppe verschwinden
- [ ] 6.4 Manuell: Gruppenname ändern → Systemnachricht „in 'Y' umbenannt", andere Mitglieder (zweite Browser-Session) sehen neuen Namen sofort
- [ ] 6.5 Manuell: Owner klickt `LogOut` → `CreatorExitChoiceModal` öffnet sich (statt einfachem Confirm)
- [ ] 6.6 Manuell: Verwaltung an Mitglied B übergeben → Owner verlässt Gruppe, B sieht beim nächsten Modal-Öffnen den Pencil-Button
- [ ] 6.7 Manuell: „Für alle löschen" → Confirm-Step erscheint, nach Bestätigung verschwindet Gruppe bei allen Mitgliedern (SSE), Toast bei aktivem Tab
- [ ] 6.8 Manuell: Mobile-Layout — Teilnehmer-Modal ist scrollbar, Touch-Targets ≥44px (Buttons in der Liste haben `py-2.5`)
