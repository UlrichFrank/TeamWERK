# Tasks

## 1. Migration

- [x] 1.1 `internal/db/migrations/072_chat_conversation_sources.up.sql` + `.down.sql` gemäß design.md §1 anlegen (vorher per `ls internal/db/migrations/` prüfen, dass 072 frei ist). Verifikation: `make migrate-up` und `make migrate-down` laufen lokal durch, `go test ./internal/db/...` grün.

## 2. Gemeinsamer Kachel-Resolver + Ausgetretenen-Filter

- [x] 2.1 `internal/chat/group_sources.go`: `GroupSource`, `validSource`, `canSeeSource`, `resolveSource` (design.md §2). Alle Kachel-Queries (`teamGroupMemberQuery`, `allTrainersMemberQuery`, `practiceGroupMemberQuery`) filtern `members.status <> 'ausgetreten'`, bei `eltern` auf das Kind (§6). Verifikation: neue Tests `TestResolveSource_*` je `groupType`/`kind` sowie `TestResolveSource_AusgetretenerFehlt`, `TestResolveSource_VerletztBleibt`, `TestResolveSource_ElternNurUeberAusgetretenesKindFehlt`.
- [x] 2.2 `ListTeamGroups`/`countTeamGroupMembers`, `ResolveTeamGroup`, `resolveAllTrainers`, `listPracticeGroups`, `ResolvePracticeGroup` auf den Resolver umstellen, ohne die Antwortform zu ändern. Verifikation: bestehende Tests in `team_groups_test.go`, `alle_trainer_test.go`, `practice_groups_test.go` bleiben grün, neu `TestListTeamGroups_CountGleichMembers` (count == len(members) für jede Kachel inkl. ausgetretenem Mitglied).

## 3. Herkunft beim Anlegen

- [x] 3.1 `CreateConversation`/`createGroup` nimmt optional `sources` an: Form prüfen (400), Sichtbarkeit (403) und Existenz in der aktiven Saison (400) prüfen, alles **vor** dem Insert. Konversation, Mitglieder und `conversation_sources` gehen in eine Transaktion. Verifikation: `TestCreateGroup_SpeichertSources` (201 + Zeilen), `TestCreateGroup_FremdeSource403` (keine Konversation entsteht), `TestCreateGroup_UngueltigerKind400`, `TestCreateGroup_OhneSourcesWieBisher`.

## 4. Vorschau-Route

- [x] 4.1 `internal/chat/group_sync.go`: `computeSyncDiff(ctx, q, convID, ownerID, claims, sources)` liefert `sources` (label/total/alreadyIn/problem), `add`, `remove`, `blocked`, `suggestions`. `q` ist `*sql.DB` oder `*sql.Tx`, damit `apply` dieselbe Rechnung in der Transaktion nutzt. Verifikation: Unit-Tests über `SyncPreview` in 4.2.
- [x] 4.2 `POST /api/chat/conversations/{id}/sync/preview` (`SyncPreview`) im Authenticated-Tier von `internal/app/router.go`. Reihenfolge 404 → 403 (nicht Ersteller) → 400 (keine Gruppe / ungültige Kachel). Ohne `sources` im Body gilt die gespeicherte Herkunft. Eintrag in `broadcastAllowlist` (schreibfrei, Begründung). Verifikation: `TestSyncPreview_NeuerSpielerInAdd`, `TestSyncPreview_AusgetretenerInRemove`, `TestSyncPreview_ManuellHinzugefuegterInRemove`, `TestSyncPreview_ErstellerNieInRemove`, `TestSyncPreview_Ueberlappung`, `TestSyncPreview_LeereSourcesAendernNichts`, `TestSyncPreview_LeererKaderSperrt`, `TestSyncPreview_UnsichtbareKachelSperrt` (nach `TransferOwnership`), `TestSyncPreview_NichtErsteller403`, `TestSyncPreview_Unbekannt404`, `TestSyncPreview_Direkt400`, `TestSyncPreview_SchreibtNichts` (Zeilenzahl von `conversation_members`/`messages`/`conversation_sources` unverändert).

## 5. Anwenden-Route

- [x] 5.1 `POST /api/chat/conversations/{id}/sync/apply` (`SyncApply`): gleiche Prüf-Reihenfolge, Diff in der `tx` neu rechnen, `409 sync_blocked` / `409 sync_stale`, `canContactUser` je Hinzugefügtem (403), Herkunft ersetzen, Mitglieder reaktivieren/einfügen/`left_at` setzen, Systemnachrichten wie `AddMember`/`RemoveMember`, Events gebündelt nach dem Commit (design.md §7). Verifikation: `TestSyncApply_WendetAuswahlAn` (Mitglieder, Systemnachrichten, Herkunft, `{added:1, removed:1}`), `TestSyncApply_AbgewaehlteBleibt`, `TestSyncApply_FremdeID409OhneAenderung`, `TestSyncApply_ErstellerEntfernen409`, `TestSyncApply_Gesperrt409`, `TestSyncApply_NichtErsteller403`, `TestSyncApply_ReaktiviertAusgetretenesMitglied` (`left_at` → NULL), `TestSyncApply_NurHerkunftFestlegen`, `TestSyncApply_Events` (Hub-Mitschnitt: je Empfänger höchstens ein Event je Typ, der Entfernte bekommt `chat:member-left`).
- [x] 5.2 Objektrechte-Matrix: Fixture-Erzeuger für beide `{id}`-Routen in `internal/permissions/object_matrix_test.go` (Gruppe von Nutzer B, Aufruf durch Mitglied A → 403). Verifikation: `go test ./internal/permissions/... ./internal/arch/...` grün (Broadcast-Gate inklusive).

## 6. Frontend: Anlegen schickt die Herkunft

- [x] 6.1 `ChatPage.tsx` „Neues Gespräch": die gewählten Kacheln (`pickedTags` → `{groupType, refId: teamId, kind}`) bei `type=group` als `sources` mitschicken. Verifikation: Vitest prüft den Payload von `POST /chat/conversations` nach Auswahl zweier Kacheln.

## 7. Frontend: Aktualisieren-Modal

- [ ] 7.1 `web/src/components/ConversationSyncModal.tsx` nach design.md §8 und dem Requirement „Aktualisieren-Modal": Erklärtext, Kacheln (Chips + Auswahl mit „x von y schon drin"), Listen „Hinzufügen"/„Entfernen" mit Checkboxen, Sperr-Hinweis, Abbrechen/OK, 409-Handling. Ausschließlich `brand-*`-Tokens, `buttonStyles`/`typography`-Konstanten, lucide-Icons. Verifikation: `ConversationSyncModal.test.tsx` mit „Abbrechen ruft apply nicht", „OK schickt nur angehakte", „Bestandsgruppe ohne Vorauswahl", „Kachel entfernen lädt Vorschau neu, Abwahl bleibt erhalten", „blocked deaktiviert OK", „409 sync_stale lädt neu und zeigt Hinweis".
- [ ] 7.2 Knopf „Aktualisieren" (`RefreshCw`, `aria-label`) in `ConversationParticipantsModal` nur für den Ersteller, öffnet das Sync-Modal, nach Erfolg `onChanged()`. Verifikation: Vitest „Nicht-Ersteller sieht keinen Knopf" / „Ersteller öffnet Sync-Modal". `pnpm -C web test` und `pnpm -C web lint` grün (Token-, Typografie- und Button-Gate).

## 8. Dokumentation und Abschluss

- [ ] 8.1 Gotcha-Absatz „Chat-Gruppen-Herkunft (`conversation_sources`)" in `docs/agent/06-gotchas.md`: ein Resolver für Kachel und Abgleich, keine Ableitung für Bestandsgruppen, Sperre bei leerer oder unsichtbarer Kachel, `<>`-Statusfilter. Verifikation: Review des Absatzes.
- [ ] 8.2 `/verify-change` durchlaufen (Build/Test/Lint, Route→Tests, Broadcast, Tokens, Migrationsnummer, `openspec validate chat-gruppe-aktualisieren --strict`). Verifikation: alle Punkte grün.
