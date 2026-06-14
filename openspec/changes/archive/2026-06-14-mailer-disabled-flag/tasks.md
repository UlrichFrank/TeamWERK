## 1. Config

- [x] 1.1 `internal/config/config.go`: Feld `MailerDisabled bool` zur `Config`-Struct hinzufügen, aus Env `MAILER_DISABLED` lesen (`os.Getenv("MAILER_DISABLED") == "true"`)

## 2. Mailer

- [x] 2.1 `internal/mailer/mailer.go`: Feld `disabled bool` zum `Mailer`-Struct hinzufügen
- [x] 2.2 `mailer.New()`: Signatur auf `New(cfg config.SMTPConfig, baseURL string, disabled bool)` erweitern und `disabled` setzen
- [x] 2.3 `mailer.Send()`: am Anfang prüfen — wenn `m.disabled`, `log.Printf("[mailer] disabled — an: %s, Betreff: %s", to, subject)` und `return nil`
- [x] 2.4 `cmd/teamwerk/main.go`: `mailer.New()`-Aufruf um `cfg.MailerDisabled` ergänzen (beide Aufrufstellen: HTTP-Handler und Scheduler)

## 3. Dokumentation

- [x] 3.1 `.env.example`: Zeile `MAILER_DISABLED=` mit Kommentar `# Auf true setzen um E-Mail-Versand lokal zu deaktivieren` einfügen
