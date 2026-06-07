## Context

Dateien werden aktuell in `DocumentsPage.tsx` via `openFile()` geladen: Der Browser holt die Datei als Blob (mit Authorization-Header), baut eine Blob-URL und ruft `window.open(blobUrl, '_blank')` auf. In iOS-PWA-Standalone-Mode wird `window.open()` mit Blob-URLs systemseitig blockiert — der Klick tut nichts, ohne Fehlermeldung.

Die Lösung: Der Browser navigiert zu einer echten HTTP-URL auf dem eigenen Backend, die ohne Authorization-Header funktioniert, aber kryptographisch abgesichert ist.

## Goals / Non-Goals

**Goals:**
- Dateiklick öffnet die Datei zuverlässig in iOS PWA, Android PWA und Desktop-Browser
- Alle Dateitypen funktionieren (PDF, Bilder, DOCX, …) — der Browser entscheidet je nach MIME-Type
- Kein DB-Schema-Change, keine neue Bibliothek, kein externer Dienst
- Bestehende Bearer-Token-basierte Downloads bleiben funktionsfähig

**Non-Goals:**
- Inline-PDF-Viewer innerhalb der App
- Revocation einzelner Tokens vor Ablauf der TTL
- Download-Statistiken oder Audit-Logging

## Decisions

### D1: HMAC-signiertes Token statt DB-gespeichertem Token

**Entscheidung:** Das Download-Token wird als `base64url(json_payload).base64url(hmac_sha256_sig)` kodiert, signiert mit dem bestehenden `JWT_SECRET`. Kein DB-Eintrag.

**Payload:** `{ "fid": <file_id>, "uid": <user_id>, "exp": <unix_timestamp> }`

**Rationale:** Stateless, kein Lock-Contention auf SQLite, kein Cleanup-Job nötig. Der JWT_SECRET ist bereits im System vorhanden. Token sind nach TTL-Ablauf automatisch ungültig.

**Alternative verworfen:** Zufalls-Token in `download_tokens`-Tabelle — erfordert Migration, Cleanup-Cronjob und zusätzliche DB-Schreibzugriffe für eine sehr kurzlebige Operation.

### D2: TTL von 5 Minuten

**Entscheidung:** 5 Minuten. Das ist lang genug für langsame Verbindungen zwischen Token-Ausstellung und Browser-Request, kurz genug um Missbrauch zu begrenzen.

### D3: Token ist an File-ID und User-ID gebunden

**Entscheidung:** Der Download-Endpoint prüft, ob `fid` im Token mit der URL-Path-ID übereinstimmt. Er prüft außerdem, ob der User (aus `uid`) Leseberechtigung auf den Ordner der Datei hat.

**Rationale:** Token-Weitergabe erlaubt keinen Zugriff auf andere Dateien. Die Berechtigungsprüfung bleibt konsistent mit dem bestehenden ACL-System.

### D4: `window.open()` statt `<a target="_blank">`

**Entscheidung:** Weiterhin `window.open(url, '_blank')` — jetzt aber mit echter Backend-URL statt Blob-URL. Auf iOS PWA wird `window.open` mit echter URL in Safari geöffnet (verlässt die PWA kurz), aber das ist akzeptables Verhalten.

**Alternative erwogen:** Anchor-Element mit `target="_blank"` — gleichwertiges Verhalten, kein nennenswerter Unterschied.

## Risks / Trade-offs

**[Token-Leak im URL-Log]** → Das Token erscheint in Server-Access-Logs und Browser-History. Mitigiert durch 5-Minuten-TTL und User-Bindung. Kein Risiko für andere Dateien/User.

**[PWA verlässt kurz den Fokus]** → Safari öffnet, zeigt das Dokument, User kehrt zur PWA zurück. Unvermeidbar bei Option B — akzeptiert.

**[iOS öffnet kein neues Tab für alle MIME-Types]** → z.B. DOCX triggert Download-Dialog statt Inline-Ansicht. Das ist korrektes Browser-Verhalten und für die Nutzung im Vereinskontext ausreichend.

## Migration Plan

1. Backend deployen (neuer Endpoint + erweiterter Download-Endpoint)
2. Frontend deployen (geänderte `openFile()`-Funktion)
3. Kein Datenbankschritt, kein Rollback-Plan nötig — altes Verhalten ist weiterhin über Bearer-Token erreichbar
