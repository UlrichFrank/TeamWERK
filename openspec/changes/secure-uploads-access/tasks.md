## 1. Backend: Token-Ausgabe & geschützte Auslieferung

- [ ] 1.1 Token-Ausgabe für Uploads analog `SepaDownloadToken` implementieren: authentifiziert, Sichtbarkeitsprüfung (`policy.MemberCan`-analog) für das angeforderte Foto, kurzlebiges HMAC-Token gebunden an die konkrete Datei
- [ ] 1.2 `ServeUpload` (`internal/upload/handler.go`): Token validieren (Signatur, Ablauf, Datei-Bindung) statt offener Auslieferung; bei fehlendem/ungültigem Token 401/403
- [ ] 1.3 `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` auf der Auslieferung setzen; UUID-/`..`-Abwehr beibehalten
- [ ] 1.4 `/api/uploads/*` in `internal/app/router.go` aus dem Public-Tier nehmen; irreführenden Doc-Kommentar korrigieren

## 2. Frontend: Token-URLs

- [ ] 2.1 Alle `photoURL`-Verwender in `web/src/` ermitteln
- [ ] 2.2 Foto-URLs auf das Token-Muster umstellen (Token aus der jeweiligen API-Antwort mitführen oder via Token-Endpoint holen), `<img src=...>` entsprechend setzen
- [ ] 2.3 Betroffene Seiten visuell prüfen (Mitgliederliste, Profil, Kind-Profil, ggf. Kader)

## 3. Tests & Verifikation

- [ ] 3.1 Tokenloser `GET /api/uploads/<datei>` → 401/403
- [ ] 3.2 Berechtigter Aufrufer: Token wird ausgestellt, Auslieferung 200 + `Referrer-Policy`/`Cache-Control`
- [ ] 3.3 Token für nicht sichtbares Foto wird nicht ausgestellt → 403
- [ ] 3.4 Abgelaufenes/ungültiges Token → 401/403
- [ ] 3.5 `/verify-change` + `openspec validate secure-uploads-access --strict`
