## Context

TeamWERK nutzt ein JWT-basiertes Auth-System: Access Token (15 Min, im Memory) + Refresh Token (7 Tage, HttpOnly Cookie, rolling). Der Auto-Refresh im Axios-Interceptor erneuert den Access Token transparent bei 401. Aktuell gibt es keine Inaktivitäts-Erkennung — ein offener Browser-Tab bleibt bis zu 7 Tage gültig.

## Goals / Non-Goals

**Goals:**
- User wird nach 30 Minuten ohne Interaktion automatisch ausgeloggt
- Vorwarnung bei 25 Minuten mit Option "Angemeldet bleiben"
- Refresh-Token-Lebensdauer auf 2 Tage reduziert (serverseitige Absicherung)

**Non-Goals:**
- Serverseitiges Sliding Window (kein last_used_at in DB)
- Geräteübergreifende Session-Invalidierung
- Konfigurierbare Timeout-Dauer per User/Rolle

## Decisions

### D1: Frontend-Idle-Timer in AuthContext, kein eigener Hook

Der Idle-Timer läuft direkt im `AuthProvider` via `useEffect`. Ein separater `useIdleTimeout`-Hook wäre sauberer, aber für eine einzelne Verwendungsstelle unnötig. Die Events (`mousemove`, `keydown`, `click`, `touchstart`, `scroll`) werden auf `window` gelauscht und bei jedem Event der Timer zurückgesetzt.

**Alternativen:** Eigener Hook (zu viel Indirektion für einen Use Case), externe Library wie `react-idle-timer` (vermieden — keine neue Dependency für 20 Zeilen Logik).

### D2: Warn-Modal inline im AuthProvider, nicht als eigene Komponente

Das Modal wird direkt aus dem `AuthProvider` gerendert (via Portal oder inline im JSX-Baum). Kein eigener `SessionWarningModal`-Component — zu klein, um aufzuteilen.

### D3: Countdown im Modal per `setInterval`

Das Warn-Modal zeigt einen Countdown von 5 Minuten. Ein `setInterval(1000)` aktualisiert die verbleibenden Sekunden. Das Interval wird bei "Angemeldet bleiben" gecleant.

### D4: Refresh-Token-Dauer 2 Tage statt 7

Einzeiliger Change in `tokens.go`: `refreshTokenDuration = 2 * 24 * time.Hour`. Kein Schema-Change, kein Migration nötig. Bestehende 7-Tage-Tokens bleiben bis zu ihrem Ablauf gültig — kein erzwungenes Re-Login bei Deployment.

## Risks / Trade-offs

- **Trainer tippt lange Daten ein ohne Maus:** Nur `keydown`-Events werden erkannt — das reicht, da Tippen = aktiv. → Kein Problem.
- **Tab im Hintergrund:** `mousemove`/`click` feuern nicht in Hintergrund-Tabs. Nach 30 Min im Hintergrund erscheint beim Wechsel zurück das Warn-Modal (oder Logout ist bereits erfolgt). → Akzeptabel, entspricht dem gewünschten Verhalten.
- **Timer läuft weiter bei Server-Down:** Logout schlägt fehl wenn der Server nicht erreichbar ist — `setUser(null)` und Token-Clear passieren trotzdem clientseitig. → Kein Problem.
- **Mehrere Tabs:** Jeder Tab hat eigenen Timer. Tab A kann ausloggen während Tab B noch aktiv ist — nächste API-Anfrage in Tab B schlägt mit 401 fehl, Interceptor leitet auf `/login` um. → Korrekt.
