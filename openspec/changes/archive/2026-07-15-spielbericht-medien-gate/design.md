# Design — spielbericht-medien-gate

## Zweck

Fixiert die Kern-Entscheidungen für das Review-Gate zwischen
Presseteam-Autor und öffentlicher TYPO3-Seite.

## Kern-Entscheidungen

### D-1 · Zweischichtige Berechtigung: Rolle (Autor) + Vereinsfunktion (Freigeber)

```
role=presseteam       ──▶ AUTOR    (schreibt, submit-for-review)
Vereinsfkt "medien"   ──▶ FREIGEBER (liest pending, editiert, publisht)
Vereinsfkt "vorstand" ──▶ FREIGEBER (identische Rechte, Fallback)
role=admin            ──▶ überall
```

- **Warum Rolle für Autoren bleibt:** Autor darf ein reiner
  Eltern-Account sein (keine Members-Reihe, keine
  `member_club_functions`); der `spielbericht-typo3-publisher`-Change
  hat das mit `presseteam` als Rolle bewusst so entschieden (D-1
  dort).
- **Warum Vereinsfunktion für Freigeber:** Der Freigeber ist immer
  ein Verantwortlicher im Verein — Members-Rolle Voraussetzung. Die
  Freigabe ist an eine Vereins-Verantwortung geknüpft, nicht an einen
  Content-Autoren-Pool.
- **Warum Vorstand als zweiter Freigeber:** Fallback, falls die
  einzige Medien-Person nicht verfügbar ist. Kein Rangunterschied
  zwischen medien und vorstand in diesem Kontext.

### D-2 · Vier-Augen-Prinzip weich (nicht erzwungen)

Ein Nutzer mit `role=presseteam` UND Vereinsfunktion `medien` (oder
`vorstand`) darf seinen eigenen Bericht submitten UND freigeben.

- **Warum:** In einem kleinen Verein mit 1–2 Medien-Verantwortlichen,
  die auch selbst schreiben, ist eine harte Trennung nicht sinnvoll —
  sie würde in der Praxis nur zu Selbst-Impersonate-Workarounds
  führen. Die soziale Kontrolle bleibt (Medien sieht alle
  pendings von Kollegen), aber der Prozess erzwingt sie nicht.
- **Konsequenz Test:** Kein Test „Autor darf sich nicht selbst
  freigeben"; stattdessen positiver Test „Autor mit Medien-Fkt kann
  publishen".

### D-3 · Kein Rückweg — pending_review → draft existiert nicht

Nach `submit-for-review` verliert der Autor die Edit-Rechte
permanent. Es gibt keinen Reject-Button, keinen Withdraw, kein
Zurücksetzen.

```
       ┌───────┐   submit     ┌───────────────┐    publish    ┌───────────┐
       │ draft │─────────────▶│ pending_review│──────────────▶│ published │
       └───────┘              └───────┬───────┘               └───────────┘
                                      │                             ▲
                                      │ 4xx/5xx                     │
                                      ▼                             │
                              ┌────────────────┐    retry (nur      │
                              │ publish_failed │──── Freigeber)─────┘
                              └────────────────┘
```

- **Warum:** Vereinfacht das Modell erheblich — weniger States,
  keine Race Conditions zwischen Rückwurf und paralleler Publish-Aktion.
  Der Freigeber hat volle Edit-Rechte auf `pending_review`; wenn er
  Änderungen will, macht er sie selbst.
- **Konsequenz Autor-UX:** Der „Zur Prüfung senden"-Button muss vorher
  eine Bestätigung erfragen („Nach dem Absenden kannst du den Bericht
  nicht mehr bearbeiten"). Reine Client-Sache, kein Backend-Zwang.

### D-4 · Notification an alle Freigeber, kein Round-Robin

`submit-for-review` löst genau einen Push-Broadcast aus — an alle
User mit Vereinsfunktion `medien` ODER `vorstand`. Kein Round-Robin,
keine Zuweisung, keine „claim"-Semantik.

- **Warum:** Klein-Verein-Realität — 1 bis 3 Medien-Personen. Wer
  zuerst da ist, publisht.
- **Race-Guard:** State-Machine im Server. Wenn zwei Freigeber
  gleichzeitig auf „Veröffentlichen" klicken, gewinnt genau einer den
  atomaren `pending_review → publishing`-Übergang; der zweite bekommt
  HTTP 409.

### D-5 · Reminder-Job nach 5 Tagen, idempotent

Ein Scheduled Job (Interval z. B. hourly, im bestehenden
`internal/scheduler/`) sucht nach:

```sql
SELECT id FROM match_reports
WHERE state = 'pending_review'
  AND submitted_at IS NOT NULL
  AND submitted_at < datetime('now', '-5 days')
  AND id NOT IN (
    SELECT context_id FROM notification_log
    WHERE context_type = 'match_report_review_reminder'
  )
```

und sendet an alle aktuellen Freigeber-User einen Push:

> „Spielbericht wartet seit 5 Tagen auf Freigabe: {Spielpaar}"

Danach `INSERT INTO notification_log (context_type, context_id, ...)`
— das verhindert Doppel-Reminder.

- **Warum idempotent:** Wenn der Bericht 30 Tage im Review liegt,
  soll er keine 25 Reminder auslösen. Einmal pinnen reicht.
- **Warum nicht „jeden Tag einer"?:** Ergibt bei 1–3 Medien-Personen
  Notification-Ermüdung. Ein Reminder = ein sozialer Nudge, mehr nicht.
- **Wenn niemand reagiert:** manueller Eingriff (Admin, Vorstand
  direkt ansprechen). Wir bauen keinen Auto-Escalate.

### D-6 · Reviewer_user_id als Audit-Feld, kein Ownership

`match_reports.reviewer_user_id` wird beim ersten erfolgreichen Publish
gesetzt. Bei Retry nach `publish_failed` überschrieben (letzter
Publisher gewinnt).

- **Warum nur Audit, kein Ownership:** Ein zweiter Freigeber darf
  einspringen, wenn der erste unterwegs abbricht (z. B. Publisher-
  Fehler + verlässt den Browser). Kein „claim"-Modell nötig.
- **Konsequenz:** Anzeige „Freigegeben von X" ist immer der letzte
  Publish-Ereignis-Ausführer, nicht der historisch erste.

### D-7 · Edit-Regel ist State- + Rollen-Matrix, nicht per-Feld

`PUT /api/match-reports/{id}` erlaubt oder verweigert komplett, kein
Feld-selektives Autor-vs-Freigeber-Update. Grund: kein Feld ist so
kritisch, dass ein Freigeber es nicht anfassen dürfen sollte — der
Freigeber ist im Zweifelsfall inhaltlich verantwortlich.

- **Konsequenz:** UI kann in `pending_review` das ganze Formular
  editierbar zeigen (für Freigeber). Für den Autor: read-only mit
  Hinweis „Zur Prüfung eingereicht — nur Medien/Vorstand können jetzt
  bearbeiten".

## Offene Punkte (bewusst nicht in Version 1)

1. **Kein Reviewer-Assignment.** Wenn wir später merken, dass es
   wildwuchs gibt, könnte ein „claim"-Mechanismus dazu — jetzt nicht.
2. **Kein Escalation-Job.** Wenn der 5-Tage-Reminder erfolglos bleibt,
   passiert nichts weiter. Kein 10-Tage-Ping, kein Admin-Alarm.
3. **Keine Reviewer-Rückmeldung an Autor.** Der Autor sieht nur den
   Zustand des Berichts (im UI erkennbar); er bekommt keine
   „freigegeben durch X"-Nachricht. Kann Follow-up sein, wenn sich
   Bedarf zeigt.
4. **Kein Delete für Freigeber.** Ein Bericht im `pending_review` kann
   nicht gelöscht werden — nur publisht (oder ewig hängen). Das ist
   OK, weil der Autor den Draft nur einmal absendet; das Modell ist
   nicht auf Massen-Absagen ausgelegt.

## Verhältnis zum archivierten Change

Der Change `spielbericht-typo3-publisher` (archiviert 2026-07-07)
bleibt die Basis: State-Machine, TYPO3-Publisher, Bilder-Pfad,
photo-consent-Warnung, Sanitizer — alles unverändert. Dieser Change
schiebt nur einen State dazwischen und trennt Autor/Freigeber.
