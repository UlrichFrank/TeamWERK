# Prod-Diagnose: „Push-Benachrichtigungen kommen nicht an"

Schritt-für-Schritt-Runbook, um bei einem konkreten Nutzer (Beispiel: Andrea,
Rolle `standard`) zu klären, **warum** Push-Nachrichten ausbleiben. Alle
DB-Zugriffe sind **read-only** (`SELECT`).

> Kontext: Der Push-Pfad kennt **keine** rollenbasierte Logik — die System-Rolle
> (`standard`/`admin`) beeinflusst die Zustellung nicht. Ausbleibende Pushes
> haben praktisch immer eine von drei Ursachen: (A) kein/kaputtes Abo in der DB,
> (B) Präferenz deaktiviert, (C) client-/geräteseitig (Berechtigung, iOS-PWA).

---

## Voraussetzungen

- SSH-Zugang zum VPS (Alias `vServer`, siehe `.env`).
- `sqlite3` auf dem VPS vorhanden (wird auch vom Backup genutzt).
- DB liegt unter `/var/lib/teamwerk/teamwerk.db`.

`-readonly` stellt sicher, dass nichts verändert wird:

```bash
ssh vServer 'sqlite3 -readonly /var/lib/teamwerk/teamwerk.db "SELECT 1;"'
```

---

## Schritt 1 — user_id über die E-Mail finden

```bash
ssh vServer 'sqlite3 -readonly -box /var/lib/teamwerk/teamwerk.db \
  "SELECT id, email, role, is_active FROM users WHERE email LIKE '\''%andrea%'\'';"'
```

Merke dir die `id` (im Folgenden `<UID>`). Prüfe `is_active=1` — ein inaktives
Konto erklärt fehlende Pushes trivial.

---

## Schritt 2 — Push-Abos prüfen (Hauptverdacht)

```bash
ssh vServer 'sqlite3 -readonly -box /var/lib/teamwerk/teamwerk.db \
  "SELECT id, substr(endpoint,1,45) AS endpoint_prefix, created_at
   FROM push_subscriptions WHERE user_id = <UID>;"'
```

- **0 Zeilen** → Es existiert **kein Abo**. Das ist die häufigste Ursache. Das
  Gerät hat sich nie registriert oder das Abo wurde gelöscht (siehe Kasten unten).
  Weiter bei Schritt 4 (Client).
- **≥1 Zeile** → Ein Abo existiert. Das Problem ist eher client-/geräteseitig
  oder eine deaktivierte Präferenz (Schritt 3). Endpoint-Präfix verrät den
  Push-Dienst (`fcm.googleapis`, `web.push.apple`, `updates.push.services.mozilla`).

> **Warum ein Abo verschwindet:** Bis PR #138 löschte der Server ein Abo bereits
> bei HTTP **400/401** vom Push-Dienst (nicht nur bei 404/410). Ein transienter
> VAPID-/Payload-Fehler konnte so ein **gültiges** Abo dauerhaft entfernen — und
> das Frontend registrierte still nicht neu. Nach dem Deploy von PR #138 wird nur
> noch bei 404/410 gelöscht; 400/401 werden geloggt.

---

## Schritt 3 — Notification-Präferenzen prüfen

```bash
ssh vServer 'sqlite3 -readonly -box /var/lib/teamwerk/teamwerk.db \
  "SELECT category, push_enabled, email_enabled
   FROM notification_preferences WHERE user_id = <UID>;"'
```

- **0 Zeilen** → Alle Kategorien auf Default (`push=an`). Präferenz ist **nicht**
  die Ursache.
- **Zeile mit `push_enabled=0`** → Für diese Kategorie hat der Nutzer Push
  bewusst deaktiviert. Kategorien: `games`, `trainings`, `duties`,
  `duty_reminders`, `carpooling`, `membership`, `chat`, `operativ`, `sonstiges`.
  Für **Chat**-Pushes ist `chat` relevant.

> Hinweis: Vor Migration 026 konnte `chat` gar nicht gespeichert werden (CHECK) —
> eine `chat`-Opt-out-Zeile existiert also nur auf aktuellem Stand.

---

## Schritt 4 — Client-/Geräte-Checks (wenn DB unauffällig)

Mit der betroffenen Person durchgehen:

1. **Browser-Berechtigung:** In den Website-Einstellungen muss „Benachrichtigungen"
   auf *erlaubt* stehen. Bei *blockiert* registriert das Frontend bewusst nichts.
2. **iOS:** Push funktioniert **nur** als installierte Home-Screen-PWA
   (`display-mode: standalone`), nicht im normalen Safari-Tab.
3. **Neu registrieren:** App/PWA einmal schließen und neu öffnen (nach Login läuft
   `usePushSubscription` erneut). Nach PR #138 landet ein fehlgeschlagenes
   Re-Subscribe als `console.warn('[push] subscribe failed', …)` in der
   Browser-Konsole — dort nachsehen (z. B. `InvalidStateError` bei rotierten
   VAPID-Keys, oder Netzwerkfehler beim `POST /api/push/subscribe`).
4. **Gegencheck:** Eine Testnachricht senden und beobachten, ob nach dem
   Neu-Öffnen wieder eine Abo-Zeile in Schritt 2 auftaucht.

---

## Entscheidungsbaum (Kurzfassung)

```
 Abo in push_subscriptions?
   NEIN ─► Warum kein Re-Subscribe? → Schritt 4
           (Berechtigung blockiert · iOS nicht als PWA · VAPID-Mismatch)
   JA   ─► notification_preferences: chat/relevante Kategorie push_enabled=0?
             JA ─► Nutzer hat bewusst deaktiviert (im Profil → Sonstiges wieder an)
             NEIN ─► client-/geräteseitig → Schritt 4, Konsole prüfen
```

---

## Nach dem Deploy von PR #138

- Abos werden nur noch bei **404/410** gelöscht → weniger „stumme" Abo-Verluste.
- Re-Subscribe-Fehler sind in der Konsole **sichtbar** (statt still verschluckt).
- Die Chat-Präferenz ist speicherbar (Migration 026); `PUT
  /api/profile/notification-preferences` liefert bei unbekannter Kategorie 400
  statt 500.
- Neue, abschaltbare Kategorien `operativ` (Vereinsaufgaben) und `sonstiges`
  (Migration 027).

Falls nach dem Deploy weiterhin nichts ankommt und Schritt 2 ein Abo zeigt:
Server-Logs auf `push transient failure` / `push send failed` für die
betreffende `subscription`-ID prüfen (`journalctl -u teamwerk`).
