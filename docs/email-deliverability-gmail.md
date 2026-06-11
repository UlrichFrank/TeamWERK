# E-Mail-Zustellung an Gmail — Diagnose & Aktionsplan

**Stand:** 2026-06-11
**Ausgangsfall:** Einladungen aus `/admin/nutzer` an `neomanz0112@gmail.com` werden vom Backend erfolgreich an den SMTP-Relay übergeben (204 No Content, kein SMTP-Fehler), kommen aber bei Gmail nicht im Posteingang an.

---

## Befund

### Was funktioniert
- Backend-Logik in `internal/auth/handler.go:279` (Invite-Handler): sauber, kein Bug.
- SMTP-Auth + TLS gegen `mail.agenturserver.de:587`: ok.
- Mailbox `teamwerk@team-stuttgart.org`: existiert und nimmt Mail an (verifiziert via RCPT TO Probe an `mx1.agenturserver.de`).
- Empfang an web.de und Netcup-MX: DKIM-pass (zwei Signaturen: `s=agenturserver` und `s=agenturserver2048`), SPF-pass, SpamAssassin-Score -0.9 (Ham), Rspamd -3.50 (Ham).

### Was fehlt / schwach ist
- **Kein DMARC-Record auf `team-stuttgart.org`.** Auth-Results melden `DMARC_NA`.
- **Geteilte IP-Reputation** der Sende-IP `185.15.192.32` (mailout01.agenturserver.de): leichter Negativ-Treffer bei Return Paths SenderScore (Header-Hinweis `RBL_SENDERSCORE(2.00)`).
- **Cold-Start-Bias bei Gmail** für die Absenderadresse `teamwerk@team-stuttgart.org` in Verbindung mit Token-Link auf ungewöhnliche Subdomain `internal.team-stuttgart.org`.

### Was kein Problem ist (ausgeschlossen)
- Code-Bug im Mailer: nein, `mailer.go` parst die From-Adresse korrekt und setzt einen sauberen Envelope-Sender.
- Mailbox nicht registriert: nein, RCPT TO liefert `250 OK`.
- DKIM-Signatur fehlt: nein, der Mittwald-Relay signiert mit zwei Selektoren.
- SPF kaputt: nein, `v=spf1 include:agenturserver.de ~all` ist gesetzt und passt.

---

## Aktionsplan

### Kurzfristig — DMARC-Record setzen (im Mittwald mStudio)

DNS für `team-stuttgart.org` läuft über Mittwald (NS = `ns01.agenturserver.{de,co,it}`), trotz Registrar 1API GmbH.

1. **Mailbox `dmarc@team-stuttgart.org`** anlegen (oder als Alias auf `vorstand@`). Ohne empfangsfähige Mailbox bouncen die Aggregate-Reports.
2. **Subdomain `_dmarc` anlegen**: mStudio → Domains → team-stuttgart.org → Subdomains → „Subdomain hinzufügen" → Name `_dmarc`.
   *Mittwald verlangt, dass `_dmarc` als Subdomain existiert, bevor der DNS-Editor TXT-Records dafür akzeptiert.*
3. **TXT-Record im DNS-Editor**: mStudio → Domains → team-stuttgart.org → DNS-Editor → TXT.
   - Host: `_dmarc`
   - Wert: `v=DMARC1; p=none; rua=mailto:dmarc@team-stuttgart.org`
4. **Verifizieren** (5–15 min nach Speichern):
   ```bash
   dig +short TXT _dmarc.team-stuttgart.org
   ```
   Erwartet: die gesetzte Policy.
   Online-Check: <https://dmarcian.com/dmarc-inspector/?domain=team-stuttgart.org>

### Mittelfristig — DMARC-Reports auswerten
- 2–4 Wochen Reports von Gmail/Microsoft/Yahoo sammeln (XML-Anhänge in der `dmarc@`-Mailbox).
- Mit Parser wie <https://dmarc.postmarkapp.com/> oder <https://dmarcian.com/> sichtbar machen.
- Wenn alle eigenen Versandwege sauber laufen, Policy auf `p=quarantine; pct=25` hochstufen, später ggf. `p=reject`.

### Empfänger-seitig (sofort, kostet nichts)
- Empfängerin (`neomanz0112@gmail.com`) bitten zu prüfen:
  - Spam-Ordner
  - Tabs „Werbung" / „Updates"
  - Volltextsuche `from:teamwerk` oder `subject:Einladung` in „Alle Nachrichten"
- Falls gefunden: „Nicht Spam" / „Absender zu Kontakten hinzufügen" klicken → trainiert Gmail dauerhaft positiv für die Domain.

### Strategisch (perspektivisch, wenn DMARC nicht reicht)
- Transaktionale Mails (Invite, Password-Reset, Welcome) über dedizierten Transactional-Provider versenden:
  - **Postmark** (sehr hohe Gmail-Reputation, Free-Tier 100 Mails/Monat)
  - **Mailgun** (Free-Tier 100 Mails/Tag)
  - **AWS SES** (extrem günstig, mehr Setup)
- Übrige Mails (vorstand@, manuelle Korrespondenz) bleiben bei Mittwald.
- Code-Änderung: `internal/mailer/mailer.go` so erweitern, dass je nach Mail-Typ (transactional vs. konversationell) ein anderer SMTP-Backend gewählt wird — oder schlicht die `SMTP_*`-Variablen auf den neuen Provider umstellen, wenn TeamWERK eh nur transactional sendet.

---

## Verifikationsplan nach DMARC-Setup

1. `dig +short TXT _dmarc.team-stuttgart.org` zeigt korrekten Wert.
2. Test-Invite an `neomanz0112@gmail.com` triggern und prüfen, ob ankommt (Posteingang oder Spam).
3. Optional Score-Check über <https://www.mail-tester.com>: Wegwerf-Adresse holen, TeamWERK-Invite dorthin schicken, Score-Page öffnen. Ziel: 9/10 oder besser.
4. Optional: Gmail Postmaster Tools <https://postmaster.google.com> für `team-stuttgart.org` verifizieren — gibt Reputations- und Spam-Komplain-Daten direkt von Gmail.

---

## Quellen

- [Mittwald FAQ — Was ist DKIM und DMARC?](https://www.mittwald.de/faq/e-mail/faq/was-ist-dkim-und-dmarc)
- [Mittwald FAQ — Wie nutze ich den DNS-Editor?](https://www.mittwald.de/faq/domains-ssl/dns/was-ist-der-dns-editor)
- [Mittwald FAQ — SPF-Records](https://www.mittwald.de/faq/e-mail/faq/spf-records)
- [DMARC.org — offizielle Spezifikation](https://dmarc.org/)
- Diagnose-Session vom 2026-06-11 (Headers von `testmail@diefranks.eu`, swaks-Test via mailout01.agenturserver.de, RCPT-Probe an mx1.agenturserver.de)

---

## ⚠ Sicherheits-TODO aus dieser Session

Während der Diagnose wurde versehentlich der **SMTP-Klartext-Passwortwert** aus `/etc/teamwerk/env` im Conversation-Transkript ausgegeben (Redact-Regex griff nicht). **Vor Abschluss dieses Tasks rotieren:**

1. Mittwald mStudio → SMTP-Zugang `p459264p6` → Passwort ändern.
2. Auf VPS: `/etc/teamwerk/env` aktualisieren (`SMTP_PASS=<neu>`).
3. `sudo systemctl restart teamwerk`.
4. Test-Invite verifizieren.
