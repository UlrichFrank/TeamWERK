## 1. Foundation — Textbaustein und Capability

- [x] 1.1 `internal/notify`: `CancellationBody(subject, dateStr, actor, reason string) string` — setzt „{subject} am {date} entfällt. Abgesagt von {actor}: {reason}." zusammen, lässt den Grund-Teil bei leerem `reason` weg und fällt bei leerem `actor` auf „Abgesagt von einem Trainer." zurück
- [x] 1.2 `internal/notify`: `ActorName(db, userID) string` — `SELECT first_name, last_name FROM users WHERE id=?`, liefert bei leeren Feldern den leeren String (**nie** die E-Mail)
- [x] 1.3 `internal/notify`: `TrimReason(s string) string` — Trim + Kürzung auf 200 **Runen** (nicht Bytes)
- [x] 1.4 Tests für 1.1–1.3: mit/ohne Grund, ohne Aktor, 500-Zeichen-Grund mit Umlauten (exakt 200 Runen, keine kaputte UTF-8-Sequenz)
- [x] 1.5 `internal/policy/rules.go`: `CapSuppressEventNotification = "suppress_event_notification"`, in `Capabilities()` unter `IsVorstandLike` ergänzen; `CanSuppressEventNotification(p *Principal) bool`
- [x] 1.6 Test: `GET /api/me` liefert die Capability für Vorstand und Admin, nicht für Trainer, sportliche Leitung, Kassierer oder Standard-Nutzer

## 2. Foundation — gemeinsames Request-Parsing

- [x] 2.1 `internal/notify` (oder `internal/httpx`, je nach Arch-Test-Klassifizierung): `DecodeCancellation(r *http.Request) (reason string, silent bool)` — toleranter Decode, `io.EOF` und Syntaxfehler ergeben `("", false)`, `reason` bereits getrimmt/gekürzt
- [x] 2.2 Test: leerer Body, kaputtes JSON, fehlende Felder, überlanger `reason` → jeweils kein Fehler, korrekte Rückgabe
- [x] 2.3 `internal/arch/arch_test.go`: neues Package (falls eines entsteht) klassifizieren, sonst bestätigen dass `notify` Foundation bleibt und keine Domäne importiert

## 3. Spiele — `DELETE /api/games/{id}`

- [x] 3.1 `internal/games/handler.go`: `DecodeCancellation` am Anfang von `DeleteGame`; `silent` gegen `policy.CanSuppressEventNotification` prüfen und bei fehlender Capability auf `false` zurücksetzen (kein 403)
- [x] 3.2 Team-Meldung auf `notify.CancellationBody(opponent, formatDateDMY(eventDate), actor, reason)` umstellen, `url` auf `""`; bei `silent` beide `notify.Send`-Aufrufe überspringen, `broadcastGameTeams` **nicht**
- [x] 3.3 Dienst-Meldung um Aktor + Grund ergänzen, Link `/dienste` beibehalten
- [x] 3.4 Veralteten Doc-Kommentar über `DeleteGame` korrigieren: die Route ist `DELETE /api/games/{id}`, nicht `/api/admin/games/{id}` (`router.go:464`)
- [x] 3.5 Tests: Grund im Text · ohne Grund kein leerer Grund-Satz · `url == ""` · Vorstand+`silent` → 0 Benachrichtigungen (Team **und** Dienste) · Trainer+`silent` → alle Benachrichtigungen · 500-Zeichen-Grund → 200 Zeichen, HTTP-Erfolg · leerer Body → Erfolg · fremdes Team als Trainer → 403 ohne Benachrichtigung · unbekannte ID → 404 ohne Benachrichtigung
- [x] 3.6 Test `TestDeleteGame_GrundWirdNichtPersistiert`: Marker-Grund, vollständiger Tabellen-/Spalten-Scan + Log-Puffer, inkl. Poison-Sanity (Vorbild `TestPreviewH4A_CredentialsWerdenNichtPersistiertOderGeloggt`)

## 4. Trainings — Löschen

- [x] 4.1 `DeleteSession` (`trainings/handler.go:598`): `title` und `date` zusätzlich zum `team_id` vorab laden; `DecodeCancellation` + Capability-Prüfung; Meldung „Training abgesagt" mit `CancellationBody`, `url` auf `""`
- [x] 4.2 `DeleteSeries` (`:534`): `name`, `valid_from`, `valid_until` zusätzlich zum `team_id` vorab laden; Titel „Trainingsserie beendet", Body mit Serienname + Zeitraum + Aktor + Grund, `url` auf `""`
- [x] 4.3 Tests je Route: Grund im Text · `url == ""` · Vorstand+`silent` → 0 Benachrichtigungen · Trainer+`silent` → Benachrichtigung geht raus · unbekannte ID → 404

## 5. Trainings — Absage über `PUT /api/training-sessions/{id}`

- [x] 5.1 `UpdateSession` (`:786`): alten `status` **vor** dem `UPDATE` lesen (der `team_id`-Query dort erweitern, keine zweite Runde)
- [x] 5.2 Bei `alt != neu && neu == "cancelled"`: Titel „Training abgesagt", Body `CancellationBody(title, date, actor, req.CancelReason)`, Link `/termine?focus=training-{id}` beibehalten; sonst unverändert „Training geändert"
- [x] 5.3 Test: `active → cancelled` mit Grund → „Training abgesagt", Grund im Body, Link mit `focus=training-{id}`
- [x] 5.4 Test: `cancelled → cancelled` → genau **eine** „Training geändert"-Meldung, keine zweite Absage-Meldung
- [x] 5.5 Test: `cancelled → active` → „Training geändert"
- [x] 5.6 Test: ungültiger `status` → weiterhin 400

## 6. Dienste — `DELETE /api/duty-slots/{id}`

- [x] 6.1 `DeleteSlot` (`duties/handler.go:514`): `event_name`, `event_date` und den Namen der Dienstart (`JOIN duty_types`) vor dem `DELETE` laden — heute wird nichts davon gelesen
- [x] 6.2 `DecodeCancellation` + Capability-Prüfung; Body mit Dienstart + Event + Datum + Aktor + Grund, Link `/dienste` **beibehalten**
- [x] 6.3 Tests: Dienstart und Event-Name im Text · Link bleibt `/dienste` · Vorstand+`silent` → 0 Benachrichtigungen · unbekannte ID → 404

## 7. Service Worker — leeres Linkziel

- [x] 7.1 `web/src/sw.ts`: exportierte reine Funktion `resolveClickTarget(data): { navigate: boolean; url: string }` — leerer/fehlender `url` ⇒ `{ navigate: false, url: '/' }`, sonst `{ navigate: true, url }`
- [x] 7.2 `notificationclick`-Handler auf die Funktion umstellen: bei `navigate === false` nur `existing.focus()` bzw. `openWindow('/')`, **kein** `existing.navigate()`
- [x] 7.3 Vitest auf `resolveClickTarget`: `""`, `undefined`, `null`, `"/dienste"`
- [x] 7.4 Vitest auf den Handler mit Fake-Clients: bei `url: ""` wird `focus()` aufgerufen und `navigate()` nicht

## 8. Frontend — Grundfeld und Stummschalt-Häkchen

- [x] 8.1 `web/src/components/GameEditModal.tsx`: Textfeld „Grund (optional)" im Lösch-Bestätigungsblock; `api.delete('/games/'+id, { data: { reason, silent } })`
- [x] 8.2 `web/src/components/SpieltagDetailModal.tsx`: gleiches Feld im „Spiel löschen?"-Block (`:396`) und im „Dienst löschen?"-Block (`:379`)
- [x] 8.3 `web/src/components/TrainingEditModal.tsx`: gleiches Feld im Lösch-Block (`:245-256`), für Einzeltermin **und** Serie
- [x] 8.4 `web/src/pages/AdminTrainingsPage.tsx`: gleiches Feld im Lösch-Dialog (`:375`)
- [x] 8.5 Häkchen „Ohne Benachrichtigung löschen" in allen vier Dialogen, sichtbar **nur** bei Capability `suppress_event_notification` (nicht an `manage_games`/`manage_trainings` hängen — Begründung in `design.md` §4)
- [x] 8.6 Styling strikt nach `docs/agent/05-frontend.md`: Input-Klassenstring, `brand-*`-Tokens, keine Unicode-Icons
- [x] 8.7 Vitest: Häkchen fehlt ohne Capability · `reason` und `silent` landen im Request-Body · leeres Feld sendet keinen `reason`-Schlüssel bzw. einen leeren String, den der Server toleriert

- [x] 8.8 **Nachgezogen:** `web/src/components/DutySlotList.tsx` ist der tatsächliche Löschpfad für Dienst-Slots (`api.delete('/duty-slots/'+slotId)`), nicht der in 8.2 genannte `SpieltagDetailModal` — dessen beide Lösch-Blöcke sind toter Code (`setShowDeleteGame(true)`/`setDeleteSlotId(<id>)` werden nirgends aufgerufen). Ohne diesen Task hätte die Backend-Arbeit aus Abschnitt 6 keine erreichbare Oberfläche. `DeleteReasonFields` im Bestätigungsdialog ergänzt; der Direktlöschpfad für **unbesetzte** Slots bleibt bewusst ohne Body, weil dort niemand benachrichtigt wird

## 9. Abschluss

- [x] 9.1 `docs/agent/06-gotchas.md`: kurzer Eintrag „Absage-Benachrichtigungen" — leeres `url` braucht die SW-Behandlung, `silent` ist eine eigene Capability enger als das Löschrecht, der Grund wird bewusst nicht persistiert
- [x] 9.2 Prüfen, dass die drei MODIFIED-Requirements den Wortlaut der Bestands-Specs vollständig ersetzen (nicht ergänzen) — insbesondere die `/termine`-Zusage in `push-games` und `push-trainings`, die dieser Change aufhebt
- [x] 9.3 `/verify-change` ausführen (Build/Test/Lint + Route→Tests, Mutation→Broadcast, brand-Tokens, lucide-Icons, `openspec validate`)
- [x] 9.4 `openspec validate absage-benachrichtigung --strict`
- [ ] 9.5 Proposal archivieren
