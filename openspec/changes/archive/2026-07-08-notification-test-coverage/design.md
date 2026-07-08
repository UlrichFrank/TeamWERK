## Context

Reine Test-Coverage-Erweiterung. Das Fundament (Send-Seam `var sendNotification`, `testutil.CreatePushSubscription`/`CreateNotificationPreference`) entsteht in `push-notification-reliability` (dort zwingend, um Defekt 2 zu testen). Dieser Change baut ausschließlich Breite darauf auf und ändert **kein** Produktions-Verhalten (Ausnahme: minimaler Fake-Mailer-Seam, falls der Email-Zweig sonst nicht beobachtbar ist).

Vollständige Trigger-Landkarte (aus der Exploration): 42 Trigger, davon 17 über `notify.Send` (Push+Email), 8 push-only, 9 email-only (Auth), 2 chat-badge. 6 Call-Sites umgehen `FilterByPushPref`.

## Goals / Non-Goals

**Goals:**
- Kern-Präferenzlogik, notify-Fassade, Abo-/Präferenz-Endpoints und Chat-Fan-out unter Test.
- Je Domäne ein Kategorie-Korrektheits-Test (games/trainings/duties/carpooling/membership).
- Die 6 Preference-Bypass-Sites als bewusste Entscheidung sichtbar pinnen.

**Non-Goals (Backlog / Stufe 3):**
- Realer Sendepfad mit echter webpush-Verschlüsselung (VAPID-Keypair + httptest + ECDH-Fixture).
- Frontend-Tests (`usePushSubscription`, `ProfileMiscTab`) und `sw.ts`.
- Die 9 Auth-Email-Flows (Einladung/Reset/Aktivierung/E-Mail-Wechsel).

## Decisions

**D1 — Auf dem Seam aus `push-notification-reliability` aufbauen, nicht duplizieren.**
Dieser Change setzt `sendNotification`-Seam + Fixtures voraus und wird **nach** jenem umgesetzt. Vermeidet doppelten Naht-Aufbau und Merge-Konflikte.

**D2 — Fake-Mailer-Seam analog zu vorhandenem Muster.**
Der Email-Zweig (`notify.Send`/`sendCategoryEmail`) braucht Beobachtbarkeit ohne echten SMTP. Idiomatisch wie `chat.pushFn`/`videos.Worker.pushSend`: eine überschreibbare Funktion bzw. `MailerDisabled`-Pfad nutzen und den Versand über einen Capture-Seam prüfen. Kein neues Framework.
*Alternative verworfen:* echten SMTP mit lokalem Mailhog — zu schwer für Unit-Tests.

**D3 — Kategorie-Korrektheit über den Seam, nicht über End-to-End.**
Statt echten Versand zu prüfen, wird der `sendNotification`/`notify`-Seam abgegriffen und die **übergebene Kategorie** assertiert. Billig, deterministisch, deckt die häufigste Regressionsklasse (falsche Kategorie ⇒ falsche Präferenz greift).

**D4 — Bypass-Sites pinnen statt fixen.**
Die 6 rohen `push.SendToUsers`-Sites werden mit ihrem **aktuellen** Verhalten getestet und im Kommentar als „bewusst? → offene Design-Frage" markiert. Ob sie `FilterByPushPref` bekommen sollen, ist eine Produkt-Entscheidung außerhalb dieses reinen Coverage-Change.

## Risks / Trade-offs

- **Seam-Abhängigkeit zu `push-notification-reliability`** → klare Reihenfolge (dieser Change danach); falls parallel, Fundament zuerst mergen.
- **Pinning zementiert evtl. einen Bug (Bypass-Sites)** → durch expliziten Kommentar + Verweis auf offene Design-Frage entschärft; der Test schützt vor *unbeabsichtigter* Änderung, nicht vor bewusster Korrektur.
- **Kategorie-Tests über Seam statt E2E** → testet nicht die echte Verschlüsselung/HTTP; bewusst als Non-Goal (Stufe 3) ausgelagert.
- **Umfang** → nach Stufe 2 fällt der Grenznutzen steil; Stufe 3 bleibt dokumentierter Backlog, kein Scope-Creep.

## Migration Plan

Keine DB-Migration, kein Deploy-Risiko (Tests + evtl. Test-Seam). `make test`/`/verify-change` müssen grün bleiben; Coverage steigt (kein Gate, nur Indikator).

## Open Questions

- Sollen die 6 Bypass-Sites `FilterByPushPref` erhalten? → **außerhalb** dieses Change; hier nur sichtbar gemacht.
