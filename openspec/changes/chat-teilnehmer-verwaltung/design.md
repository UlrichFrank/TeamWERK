## Context

Heute ist die Gruppen-Verwaltung im Chat fragmentiert: ein `UserPlus`-Icon für „Hinzufügen" (Creator-only), ein `LogOut`-Icon für „Verlassen" (alle Mitglieder), ein `Users`-Icon, das nur eine Zahl anzeigt. Es gibt kein Entfernen, kein Umbenennen, keinen Einblick in die Teilnehmerliste, und der Creator-Exit hinterlässt eine unwartbare Geistergruppe.

Der Kalender löst das gleiche „View ↔ Edit"-Muster bereits mit `EventInfoModal`: ein Modal, zwei States, Pencil-Button neben dem Schließen-`X` wechselt um, der Titel ändert sich mit. Dieses Muster übernehmen wir.

## Goals / Non-Goals

**Goals**
- Jedes Mitglied sieht jederzeit, wer in der Gruppe ist.
- Ersteller hat eine einzige zentrale Anlaufstelle für alle Verwaltungsaufgaben (Name, Add, Remove).
- Verlässt der Ersteller die Gruppe, bleibt sie verwaltbar — entweder durch Ownership-Transfer oder durch explizites Löschen-für-alle.
- Alle Mutations-Aktionen hinterlassen eine sichtbare System-Historie.

**Non-Goals**
- Mehrere Co-Creator / „Admins" pro Gruppe — der Owner bleibt singular.
- Audit-Log über die System-Nachrichten hinaus.
- Re-Invite eines explizit entfernten Users durch sich selbst (Einladung muss vom Owner kommen).
- Gruppen-Avatare oder beschreibende Felder über `name` hinaus.

## Decisions

### Decision 1: View/Edit in einem Modal nach Kalender-Vorbild

Ein einziges `ConversationParticipantsModal` mit internem `editing`-State. Pencil-Button neben `X` schaltet um, Titel wechselt von „Teilnehmer" zu „Teilnehmer bearbeiten". Pencil nur sichtbar wenn `activeConv.createdBy === user.id`.

**Alternative verworfen**: Zwei separate Modals (`ParticipantsViewModal` + `ParticipantsEditModal`). Hätte den `EventInfoModal`-Präzedenzfall ignoriert, hätte mehr Code für die Open/Close-Logik benötigt, und der Übergang vom „Ich schau mal rein" zu „Ah, eigentlich will ich was ändern" wäre durch einen Modal-Wechsel verzögert worden.

### Decision 2: Mutations sind atomar — kein „Speichern"-Button im Edit-Modus

Jede Aktion im Edit-Modus ist ein einzelner API-Call:
- Klick auf `✕` neben einem Mitglied → sofort `DELETE /members/{uid}`
- Klick auf User in Suchergebnissen → sofort `POST /members`
- Gruppenname-Feld → onBlur oder ↵ → `PUT /conversations/{id}`

So wie `AddMemberModal` heute schon funktioniert. Reduziert State-Management drastisch und macht jede Aktion einzeln revidierbar (Mitglied versehentlich entfernt? — wieder hinzufügen).

**Konsequenz für Rename**: Wir wollen kein PUT bei jedem Tastendruck. Lösung: lokaler `draftName`-State, gesendet bei `onBlur` oder Enter, mit optimistischem Update + Rollback bei Fehler. Wenn `draftName === conv.name` → kein Call.

### Decision 3: Soft-Delete via bestehende `left_at`-Spalte

`RemoveMember` setzt `conversation_members.left_at = CURRENT_TIMESTAMP` — identisches Verhalten wie `LeaveConversation`. Vorteil: vollständige Symmetrie mit dem bestehenden `chat-verlassen-rejoin`-Mechanismus. Wenn der Owner einen entfernten User später wieder hinzufügt, setzt `AddMember` `left_at = NULL` (existierende Logik) und der User sieht den Verlauf seit seinem Wiedereintritt — exakt wie nach freiwilligem Verlassen.

**Hard-Delete für „Für alle löschen"** ist eine bewusste Ausnahme: dort entfernt der Owner die Gruppe als Ganzes, niemand soll später noch hineinkommen.

### Decision 4: Vier separate Endpoints statt einem Bulk-Endpoint

Statt eines monolithischen `PATCH /conversations/{id}` mit `{ name?, addUsers?, removeUsers?, newOwner? }` haben wir vier separate Endpoints. Begründung:
- Kleinere Surface pro Endpoint = leichteres Mocking + Testen
- Atomare Mutations (siehe Decision 2) brauchen ohnehin keine Bulk-Form
- `transfer-ownership` ist später isoliert wiederverwendbar (z.B. Settings-Seite „Verwaltung übergeben ohne zu verlassen")

### Decision 5: Creator-Exit ist Frontend-Logik, nicht Backend-Zwang

Der `LogOut`-Klick prüft `activeConv.createdBy === user.id`. Falls ja → `CreatorExitChoiceModal`. Falls nein → bestehender `window.confirm`-Pfad.

Das Backend zwingt **nichts** auf: Theoretisch kann ein API-Client den Owner-Exit auch via direktem `DELETE /members/me` durchführen. Das verlassen wir bewusst, weil:
- Die Spec sagt „Ersteller bekommt eine Wahl", nicht „der Server lehnt sonst ab"
- Verlangsamung des Backends durch zusätzliche Owner-Checks wäre unverhältnismäßig
- Falls ein User per cURL den Owner-Exit erzwingt, ist die Gruppe in dem alten unschönen Zustand — kein neuer Schaden, nur kein Fortschritt

Der `CreatorExitChoiceModal` ruft intern zwei separate Endpoints in Sequenz:
```
1. POST /chat/conversations/{id}/transfer-ownership { newOwnerId }
2. DELETE /chat/conversations/{id}/members/me   (bestehend)
```

Bei „Für alle löschen": `DELETE /chat/conversations/{id}/everyone` plus zweiter `window.confirm`-Step im Modal.

### Decision 6: `chat:conv-updated` und `chat:conv-deleted` als neue SSE-Event-Typen

Bestehend: `chat:new-message:<id>`, `chat:member-left:<id>`.

Neu:
- `chat:conv-updated:<id>` — nach Rename und Transfer-Ownership. Frontend re-fetcht die einzelne Conversation (`GET /chat/conversations` filtern auf id, oder neuer Endpoint später).
- `chat:conv-deleted:<id>` — nach `DELETE …/everyone`. Frontend entfernt die Conversation aus `conversations[]`; wenn `activeConv?.id === id` → Toast „Die Gruppe wurde gelöscht" + zur Listenansicht.

**Alternative verworfen**: `chat:member-left` für RemoveMember wiederverwenden ist OK (semantisch korrekt — ein Member ist nicht mehr da). `chat:conv-updated` für RemoveMember wäre redundant. Daher: Remove löst `chat:member-left:<id>` aus (wie Self-Leave), Rename/Transfer lösen `chat:conv-updated:<id>` aus.

### Decision 7: Systemnachrichten-Sprache und -Form

Konsistente Formulierungen, immer aus Sicht des Auslösers, deutsche Verb-Endung passt zur bestehenden „hat die Gruppe verlassen":

| Aktion | Body | sender_id |
|---|---|---|
| Add | `wurde hinzugefügt` | hinzugefügter User |
| Remove | `wurde entfernt` | entfernter User |
| Rename | `hat die Gruppe in "Y" umbenannt` | Owner |
| Transfer | `hat die Verwaltung an Y übergeben` | alter Owner |
| Leave (existiert) | `hat die Gruppe verlassen` | leaving User |

`sender_id` ist die ID des Subjekts, das Frontend rendert sie als `{senderName} {body}` — bei Add/Remove wirkt das natürlich („Anna Müller wurde hinzugefügt"), bei Rename/Transfer steht der Owner-Name vorne („Ben Klein hat die Gruppe in 'Taktik' umbenannt").

### Decision 8: „Für alle löschen" bekommt Hard-Delete

Die Gruppe wird als ganzes entfernt: `DELETE FROM conversations WHERE id = ?` — der bestehende FK-Cascade auf `messages`, `message_reactions`, `message_reads`, `conversation_members` entfernt den Rest.

**Alternative verworfen**: Soft-Delete via neue Spalte `conversations.deleted_at`. Mehr Komplexität (Migration, alle Queries müssen filtern), kein erkennbarer Nutzen — wenn der Owner „für alle löschen" wählt, ist das die endgültige Aktion. Eine Wiederherstellung wäre ein separates Feature.

### Decision 9: AddMember erhält Systemnachricht nachträglich

Dies ist ein Bugfix, kein neues Verhalten — die Inkonsistenz zwischen Self-Leave (mit Systemnachricht) und Add (ohne) wird im selben Change behoben. Andernfalls würden wir den Spalt aktiv festschreiben.

## Risks / Trade-offs

- **Owner kann Gruppe still „kapern".** Wenn der Owner sich an einen Konföderierten überträgt und alle anderen entfernt, könnte das eine 1-zu-1-Gruppe werden — aber jeder entfernte User sieht „X wurde entfernt" als Systemnachricht und kann die Conv noch lesen (left_at-Filter zeigt sie zwar nicht mehr aktiv, alte Nachrichten werden trotzdem aus der DB nicht gelöscht). Akzeptiert: Owner hatte die Macht schon vorher implizit, die Sichtbarkeit verbessert sich.
- **Rename per onBlur kann mit Fokus-Hin-und-Her zu mehrfachen PUTs führen.** Mit `if (draftName === conv.name) return` als Guard entschärft.
- **Race: zwei Owner-Aktionen gleichzeitig.** Owner ändert Name auf Gerät A, überträgt auf Gerät B → Reihenfolge bestimmt Endzustand, beide Calls sind unabhängig sicher. Kein Lock nötig.
- **Member, der gerade entfernt wird, sendet zeitgleich eine Nachricht.** `messages.sender_id`-Check nutzt `isMember` (linke Spalte `left_at IS NULL`). Race möglich: Nachricht geht durch wenige ms vor dem Soft-Delete. Akzeptiert — kein Datenintegritätsproblem.

## Migration Plan

Keine DB-Migration. Reine Code-Änderung. Rollout-Reihenfolge:
1. Backend-Endpoints + Tests deployen
2. Frontend-Komponenten deployen
3. Alte `AddMemberModal`-Komponente entfernen, sobald `ConversationParticipantsModal` produktiv ist

## Open Questions

- **Push Notification für entfernten User?** Heute kriegt der Self-Leaver keine Push (logisch, war sein Akt). Beim Remove durch Owner wäre eine Push „Du wurdest aus 'Gruppe X' entfernt" denkbar. **Vorschlag**: vorerst keine Push, nur SSE — der User merkt es beim nächsten App-Öffnen. Falls Feedback kommt, separater Mini-Change.
- **Self-Removal via `/members/{uid}` mit `uid === self`?** Verboten (HTTP 400). Self-Aktion läuft über bestehenden `/members/me`-Endpoint, der konsistent für alle Mitglieder funktioniert.
