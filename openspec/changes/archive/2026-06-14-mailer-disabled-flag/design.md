## Context

Der Mailer ist ein einfaches Struct (`internal/mailer/Mailer`) ohne Interface. `Send()` ruft direkt `smtp.SendMail()` auf. Beim lokalen Entwickeln sollen keine echten Mails versandt werden.

## Goals / Non-Goals

**Goals:**
- `MAILER_DISABLED=true` in `.env` schaltet SMTP-Versand stumm
- Stattdessen erscheint ein `log.Printf`-Eintrag im Serverlog mit Empfänger und Subject
- Kein Interface-Umbau, keine neuen Dependencies

**Non-Goals:**
- Sichtbarmachen des Mail-Inhalts im Log (nur Empfänger + Subject)
- Mailpit / lokaler SMTP-Catcher
- Testbarkeit via Dependency Injection (separate Entscheidung)

## Decisions

**Env-Flag statt Interface-Umbau**  
`Mailer.disabled bool` wird in `New()` aus `cfg.MailerDisabled` gesetzt. `Send()` prüft das Flag als erste Aktion. Alternativ wäre ein `Mailer`-Interface mit `NoopMailer` sauberer, aber unverhältnismäßig für diesen Use-Case.

**Config-Feld in `SMTPConfig` vs. eigenständig**  
Das Flag gehört konzeptuell nicht zur SMTP-Verbindung, daher eigenes Feld `MailerDisabled bool` direkt in `Config` (nicht in `SMTPConfig`). `New()` erhält die Information über den vorhandenen `config.Config`-Parameter — kein Signatur-Change nötig, da `mailer.New` ohnehin `cfg.SMTP` und `cfg.BaseURL` bekommt; wir ergänzen `cfg.MailerDisabled`.

## Risks / Trade-offs

- [Risiko] Flag wird vergessen und läuft auf Prod → Mitigation: `.env.example` zeigt `MAILER_DISABLED=` (leer = disabled false), Prod-Env enthält das Flag nicht
- [Trade-off] Mail-Inhalt erscheint nicht im Log → bewusste Entscheidung, reicht für lokales Testen
