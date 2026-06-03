## Why

Team Stuttgart verwaltet Trainings bisher außerhalb von TeamWERK (WhatsApp, Zettel). Trainer haben keine strukturierte Möglichkeit, Trainingstermine anzulegen, Zu-/Absagen zu sammeln und Anwesenheiten festzuhalten. Mit dem bestehenden Spielplan als Vorbild ist die Plattform bereit für dieses Modul.

## What Changes

- Trainer können **Trainingsserien** anlegen (fester Wochentag, Uhrzeit, Ort) — das Backend generiert daraus alle Einzeltermine bis Saisonende
- Trainer können **Einzeltermine** außerhalb einer Serie anlegen sowie bestehende Sessions absagen oder individuell anpassen
- **Spieler** können für jede Session zu-/absagen (confirmed/declined/maybe) mit optionaler Begründung
- **Elternteile** können über die bestehende `family_links`-Beziehung für ihr Kind antworten
- Trainer sehen in Echtzeit, wer kommt — andere Spieler sehen Namen + Status, aber Absage-Begründungen bleiben privat (nur Trainer/Admin + der Spieler/seine Eltern)
- Nach dem Training kann der Trainer die **tatsächliche Anwesenheit** pro Mitglied erfassen
- Der bestehende **Kalender** wird um Trainingstermine erweitert

## Capabilities

### New Capabilities

- `training-series`: Anlegen, Bearbeiten und Löschen von Trainingsserien mit Wochentag-basierter Session-Generierung
- `training-sessions`: Verwaltung einzelner Trainingstermine (aus Serie oder standalone), inkl. Absagen
- `training-rsvp`: Zu-/Absage-System für Spieler und Eltern mit Privacy-Modell für Begründungen
- `training-attendance`: Nachträgliche Anwesenheitserfassung durch den Trainer nach dem Training

### Modified Capabilities

- `games`: Kalender-Integration — Trainings erscheinen neben Spielen in der Kalenderansicht (keine Anforderungsänderung an games selbst, aber KalenderPage holt zusätzliche Daten)

## Impact

- **Neues Package**: `internal/trainings/` (Handler + DB-Queries)
- **Neue Migration**: `009_trainings.up.sql` / `.down.sql`
- **Neue Frontend-Seiten**: `TrainingsPage.tsx`, `TrainingsDetailPage.tsx`, `AdminTrainingsPage.tsx`
- **Geänderte Frontend-Seiten**: `KalenderPage.tsx` (zweiter API-Call), `AppShell.tsx` (Nav-Eintrag)
- **Neue API-Routen**: unter `/api/training-sessions` und `/api/training-series`
- **Keine neuen externen Dependencies**
