## Context

TeamWERK hat keine Kalender-Export-Funktion. Mitglieder müssen Spieltermine und Dienste manuell im Blick behalten. Das iCal-Format (RFC 5545) ist der universelle Standard für Kalender-Abonnements — unterstützt von Google Calendar, Apple Calendar und Outlook ohne weitere Integration.

Das bestehende `games`-Datenmodell kennt nur `event_type IN ('heim','auswärts','generisch')`. Training und sonstige Termine sind nicht unterscheidbar, was konfigurierbare Feeds erschwert.

## Goals / Non-Goals

**Goals:**
- Persönlicher iCal-Feed pro User, konfigurierbar über 5 Toggles
- `event_type='training'` als neuer, eigenständiger Spielplan-Typ
- Kein externer Kalender-Dienst, kein externes Go-Package
- Feed-URL ist statisch abonnierbar (kein Login, kein Refresh)

**Non-Goals:**
- Fahrgemeinschaften im Feed
- Token-Regenerierung (DELETE + POST reicht)
- Eltern sehen Kinder-Termine im eigenen Feed
- Push-Benachrichtigung bei Spielplan-Änderungen im externen Kalender

## Decisions

### 1. Reines Go für iCal-Generierung — keine externe Library

RFC 5545 ist ein Textformat. Die kritischen Regeln sind: CRLF-Zeilenenden, Line-Folding bei 75 Oktetten, Escaping von `\`, `,`, `;`, `\n` in Text-Werten. Alle drei Punkte sind mit `strings.Builder` und einer Hilfsfunktion handhabbar. Eine Dependency (`github.com/arran4/golang-ical` o.ä.) wäre overhead für ~100 Zeilen Logik.

### 2. Token im URL-Pfad (`/api/calendar/feed/{token}.ics`)

Alternativen: Query-Parameter (`?token=…`) oder HTTP Basic Auth. Pfad-Token ist der de-facto-Standard (SpielerPlus, Nextcloud, Fastmail) — Calendar-Apps cachen die URL verlässlich, und der `.ics`-Suffix signalisiert Content-Type korrekt. Query-Parameter werden von einigen Calendar-Apps beim Caching ignoriert oder abgeschnitten.

### 3. Ein Token pro User (UNIQUE user_id in calendar_tokens)

Kein Bedarf für mehrere Tokens pro User (kein Multi-Device-Splitting, kein Team-granularer Feed). UNIQUE(user_id) hält das Schema einfach. `POST /api/calendar/token` ist idempotent — existierendes Token wird gepatcht, kein neues angelegt.

### 4. Einstellungen im Token-Record statt separater Tabelle

Die 5 Toggles (include_heim, include_auswaerts, include_training, include_generisch, include_duty) gehören zum Token. Vorteil: ein Query liest Token + Einstellungen in einem. Nachteil: `DELETE /api/calendar/token` löscht auch die Einstellungen. Für diesen Use-Case akzeptabel — beim nächsten `POST` gelten Defaults (alles aktiviert).

### 5. Zeitzonen: Europe/Berlin via Go-Stdlib

DB speichert Datum als `DATE` (ISO-String) und Uhrzeit als `TEXT` ("HH:MM"). Für iCal: Kombination via `time.LoadLocation("Europe/Berlin")` und Ausgabe als `DTSTART;TZID=Europe/Berlin:YYYYMMDDTHHmmss`. Das ist RFC 5545 konform und erzwingt keine UTC-Konversion mit Sommerzeit-Logik. Go's stdlib hat `time/tzdata` als optionales embed; auf dem VPS sind System-Timezone-Daten verfügbar.

### 6. Trainings aus `training_sessions`, nicht aus `games`

Ursprünglich war geplant, `event_type='training'` zu `games` hinzuzufügen. Beim Frontend-Anpassen wurde aber sichtbar: Trainings haben in TeamWERK schon eine dedizierte Tabelle (`training_sessions`) mit eigener API, eigenem Wizard-Button und eigener RSVP-Logik. Eine doppelte Quelle hätte das Datenmodell aufgeweicht. Stattdessen liest der `include_training`-Toggle direkt aus `training_sessions` (status='active'). Migration 045 beschränkt sich auf `calendar_tokens`; das `games`-Schema bleibt unverändert.

## Risks / Trade-offs

**[Risk] Calendar-App cached die Feed-URL und zeigt veraltete Events** → Mitigation: `X-WR-CALNAME` und `LAST-MODIFIED`/`SEQUENCE` in VEVENTs ermöglichen Kalender-Apps einen Diff. Polling-Intervall liegt bei 15–60 Minuten — für einen Vereinskalender akzeptabel.

**[Risk] Migration 045 ist trivial (nur CREATE TABLE) — kein Risiko durch table-recreate, da `games`-Schema unverändert bleibt.**

**[Risk] Long token (UUID v4) in URL erscheint in Server-Logs** → Mitigation: Nginx-Log-Format auf VPS loggt keine Query-Strings und Path-Parameter standardmäßig. Falls nötig, kann der Token im Log-Format maskiert werden. Keine personenbezogenen Daten im iCal-Feed.

**[Trade-off] `DELETE` löscht Einstellungen** → Akzeptiert. Alternativ wäre ein separates PATCH ohne DELETE, aber das erhöht die API-Komplexität ohne erkennbaren Nutzen für den Use-Case.

## Migration Plan

1. **Migration 045 up**: `CREATE TABLE calendar_tokens` — einfacher reiner Insert ohne Schema-Änderungen an Bestandstabellen.
2. **Migration 045 down**: `DROP TABLE calendar_tokens` — keine Datenrückführung nötig, Inhalte sind nur Konfiguration.
3. **Deployment**: `make deploy` führt `migrate up` automatisch aus. Kein Daten-Risiko.

## Open Questions

_Keine offenen Fragen — Scope ist klar vereinbart._
