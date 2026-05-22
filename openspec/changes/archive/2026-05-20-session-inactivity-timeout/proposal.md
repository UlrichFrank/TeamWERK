## Why

Aktuell bleiben Nutzer unbegrenzt eingeloggt, solange sie den Browser nicht schließen — der Refresh Token läuft erst nach 7 Tagen ab. Wird ein Gerät unbeaufsichtigt gelassen oder ein Browser-Tab offen gelassen, hat jeder physische Zugang zur Plattform uneingeschränkten Zugriff.

## What Changes

- Idle-Timer im Frontend: Nach 30 Minuten ohne Nutzerinteraktion wird der User automatisch ausgeloggt
- Vorwarnung bei 25 Minuten: Modal-Dialog mit Countdown und Optionen "Angemeldet bleiben" / "Jetzt abmelden"
- Refresh-Token-Laufzeit: von 7 Tagen auf 2 Tage gekürzt — wer die App länger nicht nutzt, muss sich neu einloggen

## Capabilities

### New Capabilities

- `session-inactivity-timeout`: Automatischer Logout nach Inaktivität mit Vorwarn-Dialog im Frontend; kürzere Refresh-Token-Lebensdauer im Backend

### Modified Capabilities

*(keine bestehenden Spec-Anforderungen ändern sich)*

## Impact

- `web/src/contexts/AuthContext.tsx`: Idle-Event-Listener + Timer-Logik + Warn-Modal
- `internal/auth/tokens.go`: Konstante `refreshTokenDuration` von 7 auf 2 Tage
- Keine DB-Migration nötig
- Keine API-Änderungen
- Bestehende Sessions werden beim nächsten Refresh nach Ablauf der 2 Tage invalidiert
