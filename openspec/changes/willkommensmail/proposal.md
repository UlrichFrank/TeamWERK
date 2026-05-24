## Why

Wenn ein neues Mitglied aufgenommen wird, soll es eine persönliche Willkommensmail mit den relevanten Vereinsdokumenten erhalten. Bisher gibt es keinen automatisierten Prozess dafür — die Mail muss manuell aus dem E-Mail-Client verschickt werden. Durch Integration in TeamWERK wird der Prozess standardisiert, der Versand nachvollziehbar protokolliert und Dokumente bleiben zentral gepflegt.

## What Changes

- Neues DB-Flag `welcome_email_sent_at` auf der `members`-Tabelle (nullable timestamp)
- Button „Willkommensmail senden" im Admin-Tab der Mitglieder-Detailseite — sichtbar sobald ein Nutzeraccount verknüpft ist, deaktiviert sobald die Mail bereits verschickt wurde
- Backend-Endpunkt `POST /api/admin/members/{id}/welcome-email` schickt die Mail, speichert Zeitstempel
- Die Mail enthält:
  - Personalisierte Anrede (Lieber/Liebe je nach Geschlecht)
  - Aufnahmedatum und Mitgliedsnummer aus dem Mitglieddatensatz
  - 3 feste PDF-Anhänge (Satzung, Gebührenordnung, Leitbild) — eingebettet ins Binary
  - Vereinssignatur mit Logo-Bild
- Anzeige des Versanddatums im UI nach erfolgreichem Versand

## Capabilities

### New Capabilities

- `welcome-email`: Manuelles Senden einer Willkommensmail an ein Mitglied — personalisiert mit Name/Datum/Mitgliedsnummer, mit drei festen PDF-Anhängen, Versand protokolliert.

### Modified Capabilities

- `members`: Neues Feld `welcome_email_sent_at` auf dem Member-Datensatz; Admin-Tab zeigt neuen Aktionsbereich.

## Impact

- **DB**: 1 neue Spalte auf `members` (Migration 013)
- **Backend**: Neuer Handler in `internal/members/` oder eigenem Package; nutzt bestehenden `mailer.Send`
- **Anhänge**: 3 PDF-Dateien werden eingebettet (embed.FS) — erhöht Binary-Größe um ca. 1–2 MB
- **Frontend**: `MemberAdminTab.tsx` bekommt neuen Button und Status-Anzeige
- **Keine neuen externen Abhängigkeiten**
