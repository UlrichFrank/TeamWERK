## Why

Die Untersuchung des Andrea-Falls hat gezeigt: Die **gesamte Zustellungs-Hälfte** des Notification-Systems ist heute ungetestet. `testutil.TestConfig()` hat `VAPIDPrivateKey==""`, wodurch `push.SendToUsers`/`SendToUserWithBadge` in jedem Test sofort per Guard zurückkehren — der Subscription-Lookup, der Versand und der Dead-Endpoint-Cleanup laufen nie. Es gibt 42 Notification-Trigger über 7 Domänen + Scheduler; große Teile der Kern-Businesslogik (`FilterByPushPref`, `HasEmailEnabled`, `GetAllPreferences`, `notify.Send`, subscribe/unsubscribe/preferences-Endpoints) haben **keinen einzigen Test**. Genau dort konnten beide Defekte aus `push-notification-reliability` unbemerkt entstehen.

Ziel: so viel wie realistisch sinnvoll abdecken, ohne in teure Randbereiche zu laufen. Das Test-**Fundament** (Send-Seam + Fixtures) entsteht bereits in `push-notification-reliability` (dort zwingend nötig, um Defekt 2 zu testen); dieser Change baut die **Breite** darauf auf.

## What Changes

- **Kern-Unit-Tests (Präferenz-Logik):** `FilterByPushPref`, `HasEmailEnabled`, `GetAllPreferences` — inkl. Default-Verhalten (kein Row ⇒ push=true/email=false) und Kategorie-Isolation.
- **Fassade `notify.Send`:** Fan-out in Push- **und** Email-Zweig, leere Liste kurzschließt, eine Kategorie beeinflusst nicht die andere; `sendCategoryEmail` (Direktlink-Anhang, fehlende E-Mail).
- **HTTP-Endpoints (`internal/notifications`):** `POST/DELETE /api/push/subscribe` (Upsert per Endpoint, Cross-User-Schutz beim Löschen, 400 bei fehlenden Feldern) und `GET/PUT /api/profile/notification-preferences` (Defaults, Auth).
- **Chat-Breite:** Push an eine **Gruppe mit N Empfängern** und **Broadcast**-Push (bislang nur 1-Empfänger-Fall getestet).
- **Kategorie-Korrektheit je Trigger:** pro Domäne (games/trainings/duties/carpooling/membership) ein Test, dass der Trigger die **richtige** Kategorie an `notify.Send` gibt.
- **Preference-Bypass festnageln:** die 6 Call-Sites mit rohem `push.SendToUsers` ohne `FilterByPushPref` (match-report-reminder, attendance-reminder, video-retention, video-ready, carpool-pairing-request, match-report-submitted) — aktuelles Verhalten **pinnen** und im Test kommentieren, damit die Design-Entscheidung „bewusst vs. Bug" sichtbar wird (Folge-Entscheidung außerhalb dieses Change).
- **Fake-Mailer-Seam** in `testutil`, um den Email-Zweig ohne echten SMTP zu beobachten.

## Capabilities

### New Capabilities
- `notification-test-coverage`: Garantierte Test-Abdeckung des Notification-Mechanismus — Präferenz-Filterung, notify-Fassade (Push+Email), Abo-/Präferenz-Endpoints, Chat-Fan-out und Kategorie-Korrektheit je Trigger.

### Modified Capabilities
<!-- keine — reine Test-Coverage, keine Verhaltensänderung an Produktions-Requirements -->

## Impact

- **Tests (neu):** `internal/push/prefs_test.go`, Ergänzungen `internal/push/push_test.go`, `internal/notify/notify_test.go`, `internal/notifications/handler_test.go`, `internal/chat/*_test.go`; je Domäne ein Kategorie-Korrektheits-Test bzw. Ergänzung bestehender Domänen-Tests.
- **Fixtures/Seams:** `testutil` Fake-Mailer (Push-Seam + `CreatePushSubscription`/`CreateNotificationPreference` kommen aus `push-notification-reliability`).
- **Kein** Produktionscode-Verhalten ändert sich (Ausnahme: minimaler Fake-Mailer-Seam analog zum vorhandenen `chat.pushFn`/`videos`-Muster, falls `mailer` eine Injektion braucht).
- **Abhängigkeit:** setzt das Fundament aus `push-notification-reliability` (Send-Seam, Fixtures) voraus → dieser Change **nach** jenem umsetzen.

## Out of Scope (Backlog / Stufe 3)

- **Realer Sendepfad (hohe Treue):** echtes VAPID-Keypair + httptest-Push-Endpoint + ECDH-Fixture für 1–2 „Verschlüsselung funktioniert wirklich"-E2E-Tests. Wertvoll, aber teurer Fixture-Aufbau — bewusst später.
- **Frontend-Tests:** Vitest für `usePushSubscription` (Re-Subscribe, `InvalidStateError`, iOS-Gate) und `ProfileMiscTab` Chat-Toggle-Persistenz.
- **Service-Worker (`sw.ts`) push/`setAppBadge`:** höchster Aufwand (SW-Global-Scope), geringster Grenznutzen — vorerst ausgeschlossen.
- **9 Auth-Email-Flows** (Einladung/Reset/Aktivierung/E-Mail-Wechsel via rohem `mailer.Send`): eigener Test-Bereich, nicht Teil der Notification-Kategorie-Logik.
