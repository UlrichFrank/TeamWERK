## Context

Zwei Probleme müssen gemeinsam gelöst werden:

**Problem 1 — Upload-Stub:** `handleSepaUpload` in `MemberDatenschutzTab.tsx` ist ein leerer Stub. Das Backend (`POST /api/upload/sepa-mandat/{id}`) ist fertig.

**Problem 2 — Ungeschützter Download:** SEPA-Dokumente liegen in `uploads/sepa-mandats/` und werden über die **öffentliche** Route `GET /api/uploads/*` ausgeliefert (main.go:120, kein Auth-Middleware). Jeder mit der UUID-URL hat Zugriff. Ausserdem gibt `GET /api/members/{id}` die `sepa_mandat_url` nur Admins zurück — Mitglied, Eltern und Vorstand können ihr eigenes Mandat nicht sehen.

**Problem 3 — Kein Löschen:** Kein Endpoint für Mitglieder, ein Mandat zurückzuziehen.

## Goals / Non-Goals

**Goals:**
- Drei neue Backend-Endpoints für sicheren Download und Löschen
- `sepa_mandat_url` für alle berechtigten Rollen zurückgeben
- Upload-Funktion im Frontend fertigstellen
- Öffnen via Token-Flow (wie DocumentsPage), iOS-kompatibel
- Löschen-Button für Mitglied (isOwn), Vorstand, Admin
- iOS-Bug in `DocumentsPage.openFile` beheben

**Non-Goals:**
- Keine Migration der bestehenden `/api/uploads/*` Route für andere Dateitypen (Fotos bleiben öffentlich)
- Keine Approval-Flow für das Zurückziehen — direkter Delete ohne Bestätigung durch Vorstand
- Kein Vorschau-Modal für PDFs

## Decisions

**1. Neue Endpoints in `internal/upload/handler.go`**  
Die SEPA-Upload-Logik gehört bereits in `upload`. Die Token-Logik für den Download wird als neue Datei `internal/upload/sepa_token.go` implementiert (analog zu `internal/files/token.go`) — gleicher HMAC-Mechanismus, aber mit `memberID` statt `fileID` im Payload.

**2. Token-Payload für SEPA-Download**  
```go
type sepaDownloadTokenPayload struct {
    MemberID  int   `json:"mid"`
    UserID    int   `json:"uid"`
    ExpiresAt int64 `json:"exp"`
}
```
TTL: 5 Minuten (wie files). Der Serve-Endpoint validiert Token und prüft nochmals ob `members.sepa_mandat_path` gesetzt ist.

**3. Neuer Serve-Endpoint statt `/api/uploads/*`**  
`GET /api/members/{id}/sepa-mandat/download?token=...` wird als öffentliche Route registriert (wie `GET /api/files/{id}/download`). Er validiert den Token und serviert die Datei aus `uploadDir/sepa-mandats/...`.  
`/api/uploads/*` bleibt unverändert öffentlich — für Fotos weiterhin benötigt.

**4. Berechtigungsmodell für Token-Endpoint und Löschen**

| Aktion | Wer darf |
|--------|----------|
| Download-Token holen | isOwn, isParent (family_link), vorstand (club_function), admin |
| Dokument löschen | isOwn, isParent (family_link), vorstand (club_function), admin |

`isParent` wird geprüft via `SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?` — die Funktion `isParentOf` existiert bereits in `handler.go:1811`.

**5. `sepa_mandat_url` in `GET /api/members/{id}` erweitern**  
Aktuell: nur `isAdmin`. Neu: `isAdmin || isOwn || isParent || hasFunction("vorstand")`.  
Die URL zeigt künftig auf `/api/members/{id}/sepa-mandat/download` (ohne Token — der Client holt den Token separat via Download-Token-Endpoint).

**6. iOS-Fix für `DocumentsPage.openFile`: Fenster synchron öffnen**  
```tsx
// Synchron im Click-Handler → iOS-Popup-Blocker greift nicht
const tab = window.open('about:blank', '_blank')
try {
  const { data } = await api.get<{ token: string }>(`/files/${file.id}/download-token`)
  if (tab) tab.location.href = `/api/files/${file.id}/download?token=${data.token}`
} catch {
  if (tab) tab.close()
  setFileError('Datei konnte nicht geöffnet werden.')
}
```
Das SEPA-Öffnen in `MemberDatenschutzTab` folgt demselben Muster:
```tsx
const tab = window.open('about:blank', '_blank')
try {
  const { data } = await api.get<{ token: string }>(`/members/${memberId}/sepa-mandat/download-token`)
  if (tab) tab.location.href = `/api/members/${memberId}/sepa-mandat/download?token=${data.token}`
} catch {
  if (tab) tab.close()
  setOpenError('Dokument konnte nicht geöffnet werden.')
}
```

**7. Löschen setzt `sepa_mandat_path = NULL`, lässt `sepa_mandat`-Flag unverändert**  
Das Boolean-Flag `sepa_mandat` und das Datum werden von Admin gesteuert (bestehende Logik). Das Dokument zurückziehen bedeutet nur: Datei löschen + `sepa_mandat_path = NULL`. Admin muss danach bei Bedarf das Flag manuell anpassen.

## Risks / Trade-offs

- [Blank-Tab bei Token-Fehler] Bei Fehler wird der leere Tab sofort geschlossen — kurzes Aufblitzen, akzeptabler Kompromiss.
- [Fotos bleiben öffentlich] `/api/uploads/*` bleibt public für Mitgliedsfotos. Das ist eine separate Sicherheitsfrage und nicht im Scope.
- [Admin kann weiterhin IBAN sehen, Vorstand nicht] Bestehende Trennung zwischen Vorstand und Admin für Bankdaten bleibt unverändert.
