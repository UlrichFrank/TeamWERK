## Context

Der Spielbericht-Publisher (PR #133) warnt vor Team-Mitgliedern „ohne Foto-Freigabe" und stützt
sich dafür auf `members.photo_visible`. Dieses Flag steuert aber nur die **interne**
Profilbild-Sichtbarkeit im Portal — nicht die Einwilligung zur öffentlichen Veröffentlichung. Die
bestehenden DSGVO-Felder (`dsgvo_verarbeitung`, `dsgvo_weitergabe`, je mit `_date`) sind das
Vorbild für Struktur, API-Transport, UI und den `dsgvo`-Draft-Workflow. Constraints: SQLite ohne
`RETURNING`, keine ORM, `_date`-Logik liegt heute teils clientseitig (Feld wird als String
mitgeschickt).

## Goals / Non-Goals

**Goals:**
- Semantisch korrektes, dokumentiertes Einwilligungsfeld für öffentliche Foto-Veröffentlichung.
- Publisher entkoppelt von `photo_visible`.
- Erklärtexte zu allen drei DSGVO-Schaltern in Profil und Mitglieder-Verwaltung.
- Bestandsmitglieder migrieren auf „an", Neuanlage auf „aus" (opt-in).

**Non-Goals:**
- Keine Änderung an `photo_visible` selbst oder an dessen interner Sichtbarkeitslogik.
- Keine kanalspezifische Feingranularität (getrennt Homepage vs. Spielbericht) — ein Flag.
- Kein Rückwirken auf bereits veröffentlichte Spielberichte.

## Decisions

**1. Zwei Spalten statt JSON.** `foto_veroeffentlichung` + `foto_veroeffentlichung_date` als eigene
Spalten, exakt analog zu `dsgvo_verarbeitung`/`_date`. Alternative (in ein DSGVO-JSON packen)
verworfen: bricht das bestehende Schema-Muster und die Scan-Pfade.

**2. `_date` beim Aktivieren setzen (aus→an).** Konsistent mit den bestehenden DSGVO-Feldern. Da
das Datum heute vom Client als String mitkommt, bleibt dieses Muster erhalten; der Server setzt es
zusätzlich defensiv, wenn der Wechsel aus→an ohne Datum eintrifft. Beim Deaktivieren bleibt das
alte Datum stehen (Nachweis der einstigen Einwilligung).

**3. Migration: Bestand = 1, Default = 0.** Spaltendefault `DEFAULT 0` (opt-in für Neuanlagen);
ein `UPDATE members SET foto_veroeffentlichung=1, foto_veroeffentlichung_date=<Migrationsdatum>`
im selben `.up.sql` hebt den **Bestand** auf „an". Bewusste Entscheidung des Betreibers
(Bestand hatte de facto bereits über `photo_visible` publiziert; Neu-Beitritte müssen aktiv
zustimmen). SQLite: Spaltenzugabe via `ALTER TABLE ADD COLUMN`; Down-Migration entfernt beide
Spalten (SQLite 3.35+ unterstützt `DROP COLUMN`; sonst Tabellen-Rebuild wie in bestehenden
Down-Migrationen).

**4. `dsgvo`-Draft erweitern.** Der bestehende `field_name='dsgvo'`-Draft trägt heute
`{verarbeitung, weitergabe}`. Er wird um `foto_veroeffentlichung` erweitert
(`extractFieldValue` + Apply-`switch`-Zweig `case "dsgvo"`). Alternative (eigener Draft-Typ
`foto`) verworfen: unnötige Fragmentierung, ein DSGVO-Draft deckt alle Einwilligungen ab.

**5. Publisher-Query.** In `consentMissing` nur die `WHERE`-Bedingung
`COALESCE(m.photo_visible,0)=0` → `COALESCE(m.foto_veroeffentlichung,0)=0` ändern; der
Notlösungs-Kommentar entfällt.

## Risks / Trade-offs

- **DSGVO-Bedenken bei Bestand=„an"** → Einwilligung-per-Default ist rechtlich heikel. Mitigation:
  bewusste, dokumentierte Betreiber-Entscheidung; die Einwilligung ist jederzeit im Profil
  einsehbar und über den Draft-Workflow widerrufbar; `_date` dokumentiert den Migrationszeitpunkt.
- **Doppelte `_date`-Setzung (Client + Server)** → Server-Logik ist rein additiv/defensiv und darf
  einen vom Client gelieferten Wert nicht überschreiben. Mitigation: Server setzt `_date` nur, wenn
  aus→an **und** kein Datum geliefert wurde.
- **Frontend-Typ-Drift** → das `Member`-Interface ist an mehreren Stellen dupliziert
  (`ProfilePage`, `MemberDetailPage`, Tab-Komponenten). Mitigation: alle Fundstellen aus dem
  Impact-Abschnitt der Proposal in den Tasks abarbeiten; `pnpm -C web build` (tsc) fängt Lücken.

## Migration Plan

1. Migration `022_press_photo_consent.up.sql` (ADD COLUMN + Bestands-UPDATE) / `.down.sql`.
2. Backend: Struct-Felder, Scan-Pfade, Create/Update, Draft extract/apply, Publisher-Query.
3. Frontend: beide Tabs + Erklärtexte + `Member`-Typen + `dsgvo`-Draft-Payload.
4. Tests (Backend Happy-Path/Fehlerfall, Draft-Apply, Publisher; Frontend-Tab-Test aktualisieren).
5. Deploy via `make deploy` (führt `migrate up` aus). Rollback: `make migrate-down` entfernt Spalten.
