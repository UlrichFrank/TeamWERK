## 1. Migration

- [x] 1.1 `internal/db/migrations/022_press_photo_consent.up.sql`: `ALTER TABLE members ADD COLUMN foto_veroeffentlichung INTEGER NOT NULL DEFAULT 0` + `ADD COLUMN foto_veroeffentlichung_date DATE` + `UPDATE members SET foto_veroeffentlichung=1, foto_veroeffentlichung_date=<Migrationsdatum>` (Bestand auf „an").
- [x] 1.2 `022_press_photo_consent.down.sql`: beide Spalten entfernen (DROP COLUMN bzw. Tabellen-Rebuild-Muster wie in bestehenden Down-Migrationen).
- [x] 1.3 `make migrate-up` lokal, Spalten + Bestands-Update verifizieren; `make migrate-down` rundreise-testen.

## 2. Backend — Member-API

- [ ] 2.1 `internal/members/handler.go`: `Member`-Struct um `FotoVeroeffentlichung bool` + `FotoVeroeffentlichungDate *string` erweitern (JSON `foto_veroeffentlichung`/`_date`); Request-Struct entsprechend.
- [ ] 2.2 Alle Scan-Pfade (Get, List, Create-Reload) um die zwei neuen Spalten ergänzen.
- [ ] 2.3 Create-INSERT und Update-UPDATE (Vorstand-Zweig) schreiben `foto_veroeffentlichung` + `_date`; Server setzt `_date` defensiv beim Wechsel aus→an ohne geliefertes Datum.

## 3. Backend — Draft-Workflow

- [ ] 3.1 `internal/members/drafts.go`: `case "dsgvo"` in `extractFieldValue` um `foto_veroeffentlichung` erweitern.
- [ ] 3.2 Apply-Zweig `case "dsgvo"` schreibt `foto_veroeffentlichung` (mit `_date`-Logik) auf das Mitglied.

## 4. Backend — Spielbericht-Publisher

- [ ] 4.1 `internal/matchreports/photo_consent.go`: `consentMissing`-Query von `photo_visible` auf `foto_veroeffentlichung` umstellen; Notlösungs-Kommentar entfernen.

## 5. Backend — Tests

- [ ] 5.1 Members: Happy-Path (Vorstand setzt `foto_veroeffentlichung`, `_date` wird gesetzt) + Fehlerfall (non-privileged darf nicht direkt schreiben).
- [ ] 5.2 Draft-Apply: `dsgvo`-Draft mit `foto_veroeffentlichung` wird korrekt übernommen.
- [ ] 5.3 `matchreports`: Mitglied mit `foto_veroeffentlichung=0` erscheint in Consent-Warnliste, mit `=1` nicht (unabhängig von `photo_visible`).

## 6. Frontend

- [ ] 6.1 `Member`-Typen (`ProfilePage.tsx`, `MemberDetailPage.tsx`, Tab-Interfaces) um `foto_veroeffentlichung` + `_date` erweitern.
- [ ] 6.2 `MemberDatenschutzTab.tsx`: editierbarer Schalter + Erklärtexte zu allen drei DSGVO-Schaltern; `dsgvo`-Draft-Payload und Draft-Anzeige um `foto_veroeffentlichung` erweitern.
- [ ] 6.3 `ProfileDatenschutzTab.tsx`: read-only-Anzeige der dritten Einwilligung + Erklärtexte zu allen drei Schaltern.
- [ ] 6.4 „Kontakt"/Profil-Draft-Erzeugung: `foto_veroeffentlichung` in den `dsgvo`-Draft aufnehmen.
- [ ] 6.5 `ProfileDatenschutzTab.test.tsx` aktualisieren (dritte Einwilligung + Erklärtexte).

## 7. Abschluss

- [ ] 7.1 `/verify-change` (Build/Test/Lint + Invarianten), `openspec validate press-photo-consent-field --strict`.
- [ ] 7.2 Change archivieren.
