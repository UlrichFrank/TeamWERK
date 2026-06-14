## Why

Beim lokalen Testen auf localhost sollen keine echten E-Mails versendet werden. Bisher gibt es keinen Dev-Schalter — der Mailer versucht immer, per SMTP zu senden.

## What Changes

- Neues Env-Flag `MAILER_DISABLED=true` deaktiviert den SMTP-Versand vollständig
- `Mailer.Send()` schreibt stattdessen einen Info-Logeintrag und gibt `nil` zurück
- `.env.example` dokumentiert das Flag

## Capabilities

### New Capabilities

- `mailer-disabled-flag`: Opt-out-Schalter für SMTP-Versand via Umgebungsvariable — für lokale Entwicklung und Tests

### Modified Capabilities

_(keine)_

## Impact

- `internal/config/config.go`: neues Feld `MailerDisabled bool`
- `internal/mailer/mailer.go`: `disabled bool` im Mailer-Struct, Early-Return in `Send()`
- `.env.example`: `MAILER_DISABLED=` dokumentiert
- Kein Interface-Umbau, keine neuen Dependencies
