## 1. Backend: Refresh-Token-Laufzeit kürzen

- [x] 1.1 In `internal/auth/tokens.go` Konstante `refreshTokenDuration` von `7 * 24 * time.Hour` auf `2 * 24 * time.Hour` ändern

## 2. Frontend: Idle-Timer-Logik in AuthContext

- [x] 2.1 In `web/src/contexts/AuthContext.tsx` State `showWarning` (boolean) und `countdown` (number, Sekunden) hinzufügen
- [x] 2.2 `useEffect` mit `window`-Event-Listenern für `mousemove`, `keydown`, `click`, `touchstart`, `scroll` ergänzen — jeder Event ruft `resetTimer()` auf
- [x] 2.3 `resetTimer()`-Funktion implementieren: löscht bestehende Timeouts/Intervals, setzt neuen 25-Min-Timeout für Vorwarnung und 30-Min-Timeout für Auto-Logout
- [x] 2.4 Bei 25-Min-Timeout: `showWarning = true` und `setInterval(1s)` für Countdown-Dekrement starten
- [x] 2.5 Bei 30-Min-Timeout: `logout()` aufrufen
- [x] 2.6 Timer nur starten wenn `user !== null`; bei Logout alle Timeouts/Intervals cleanen
- [x] 2.7 Cleanup beim Unmount: alle `removeEventListener`-Aufrufe und `clearTimeout`/`clearInterval`

## 3. Frontend: Warn-Modal rendern

- [x] 3.1 Im Return-JSX des `AuthProvider` einen modalen Dialog rendern wenn `showWarning === true`
- [x] 3.2 Modal zeigt Text "Sie werden in X Minuten automatisch abgemeldet" mit Countdown in Sekunden
- [x] 3.3 Button "Angemeldet bleiben": ruft `resetTimer()` auf, setzt `showWarning = false`
- [x] 3.4 Button "Jetzt abmelden": ruft `logout()` auf
- [x] 3.5 Modal-Styling konsistent mit bestehendem Design (Tailwind, Markenfarben)

## 4. Build & Deploy

- [ ] 4.1 `make deploy` ausführen und Funktion im Browser prüfen  ← manuell
