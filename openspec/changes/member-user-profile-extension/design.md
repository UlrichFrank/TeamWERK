## Context

TeamWERK verwaltet Mitglieder (club identity) und Nutzer (login identity) in getrennten Tabellen, verbunden über `members.user_id`. Aktuell fehlen beide Datenkategorien, die für einen Sportverein essentiell sind: verwaltungsseitige Daten (Adresse, IBAN, DSGVO, SEPA) und nutzerseitige Kontaktdaten (Telefon, Profilbild).

Für Dateiablage (Fotos) gibt es noch keine Infrastruktur. Der VPS hat begrenzte Ressourcen (1 GB RAM); SQLite ist die einzige Datenbank.

## Goals / Non-Goals

**Goals:**
- Mitglied um Verwaltungsfelder erweitern (Adresse, Eintrittsdatum, IBAN, DSGVO, SEPA, Foto)
- Nutzer um Kontaktfelder erweitern (Telefonnummern, Adresse, Profilbild, Sichtbarkeit)
- Dateiablage-Infrastruktur auf dem Filesystem (Upload + Auslieferung)
- Family-links nur für Nicht-Elternteil-Rollen sichtbar

**Non-Goals:**
- Bildbearbeitung, Thumbnails, Resize
- Verschlüsselung von IBAN in der DB (App-Level-Zugriffsschutz reicht)
- URL-basierter State, externe Dienste

## Decisions

### 1. DB-Struktur: Mitglied inline erweitern
Neue Felder direkt per `ALTER TABLE` auf `members` (SQLite unterstützt ADD COLUMN). Kein separates Tabelle nötig — die Felder gehören organisch zum Mitglied.

Felder: `street TEXT`, `zip TEXT`, `city TEXT`, `join_date DATE`, `iban TEXT`, `photo_path TEXT`, `dsgvo_verarbeitung INTEGER DEFAULT 0`, `dsgvo_verarbeitung_date DATE`, `dsgvo_weitergabe INTEGER DEFAULT 0`, `dsgvo_weitergabe_date DATE`, `sepa_mandat INTEGER DEFAULT 0`, `sepa_mandat_date DATE`, `sepa_mandat_path TEXT`.

### 2. Telefonnummern: eigene Tabelle
Mehrere Nummern pro Nutzer erfordern eine 1:n-Relation. Tabelle `user_phones(id, user_id FK, label TEXT, number TEXT NOT NULL, sort_order INTEGER DEFAULT 0)`. Label ist Freitext — keine CHECK-Constraint, damit eigene Bezeichnungen möglich sind. Empfehlungen (Privat/Mobil/Firma) nur im Frontend als Vorschlagsliste.

### 3. Nutzer-Sichtbarkeit: grobe Sichtbarkeit, eigene Tabelle (1:1)
`user_visibility(user_id PK FK, phones_visible INTEGER DEFAULT 0, address_visible INTEGER DEFAULT 0, photo_visible INTEGER DEFAULT 0)`. Zeile wird beim ersten Profil-Speichern via `INSERT OR REPLACE` angelegt. Default: alles nicht sichtbar (Privacy by Default).

Granularität: **grob** — ein Toggle pro Datentyp, Zielgruppe ist immer "alle Teammitglieder". Keine rollenbasierte Feinsteuerung (kein "nur für Trainer sichtbar"). Kann später verfeinert werden.

### 4. Nutzer-Adresse: inline auf users
Eine Adresse pro Nutzer reicht. Felder `street TEXT`, `zip TEXT`, `city TEXT` direkt auf `users` per `ALTER TABLE`.

### 5. Dateiablage: Filesystem + Pfad in DB
**Entscheidung gegen SQLite-BLOBs**: große Binärdaten blähen DB auf, erschweren Backups, können nicht direkt per nginx ausgeliefert werden.

Dateipfad-Schema: `storage/uploads/{typ}/{uuid}.{ext}` (z.B. `member-photos/a3f2...jpg`). Pfad relativ zu einem konfigurierbaren Upload-Verzeichnis (Standard: `./storage/uploads/`). Ermöglicht spätere Verschiebung ohne DB-Änderung.

Upload-Flow:
```
POST /api/upload/member-photo/{id}    multipart/form-data → jpeg/png/webp ≤ 5 MB → members.photo_path
POST /api/upload/user-photo           multipart/form-data → jpeg/png/webp ≤ 5 MB → users.photo_path
POST /api/upload/sepa-mandat/{id}     multipart/form-data → PDF/jpeg/png/webp ≤ 10 MB → members.sepa_mandat_path
GET  /api/uploads/{filename}          auth required, liefert Datei aus (Content-Type aus Dateiendung)
```

Subdirectories: `member-photos/`, `user-photos/`, `sepa-mandats/`.

Fotos: Maximale Dateigröße 5 MB, Typen: image/jpeg, image/png, image/webp.
SEPA-Dokument: Maximale Dateigröße 10 MB, Typen: application/pdf + image/*.

### 5a. Member-Foto Sichtbarkeit: immer für alle eingeloggten Nutzer
Das Mitgliedsfoto (Passfoto) dient der Identifikation — Trainer lernen Namen, Vorstand hat Gesicht zum Namen, Events. Es wird daher **immer** an alle authentifizierten Nutzer ausgeliefert, unabhängig von Sichtbarkeitseinstellungen. Kein Visibility-Toggle für Member-Fotos. User-Profilfotos folgen der `photo_visible`-Einstellung in `user_visibility`.

### 6. Family-Links-Sichtbarkeit: reiner Rollencheck
Keine neue DB-Spalte. Im Handler `GET /api/members/{id}/parents` wird geprüft: wenn `claims.Role == "elternteil"` → nur eigene Links zurückgeben (WHERE parent_user_id = claims.UserID). Alle anderen Rollen bekommen alle Links des Mitglieds.

### 7. IBAN-Zugriffsschutz
`members.iban` wird im `GetMember`-Handler nur eingeschlossen wenn `claims.Role == "admin"`. Im `ListMembers`-Response wird IBAN nie mitgegeben (auch nicht für Admin — nur im Einzelabruf).

### 8. Mitglied-Adresse erbt Nutzer-Adresse als Fallback
Beide Entitäten haben eigene Adressfelder für unterschiedliche Zwecke:
- `users.street/zip/city` = persönliche Adresse (für Fahrdienste, eigene Nutzung)
- `members.street/zip/city` = offizielle Vereinsadresse (für Anschreiben, Rechnungen, SEPA)

**Fallback-Logik in `GetMember`**: Wenn `members.street` leer ist UND das Mitglied einen verknüpften Nutzer hat, werden `users.street/zip/city` als Mitglieds-Adresse zurückgegeben. Das Feld `address_source` im Response zeigt an ob die Adresse vom Mitglied (`"member"`) oder vom Nutzer (`"user"`) kommt — ermöglicht UI-Hinweis "Übernommen vom Nutzerprofil".

Admin kann die Mitglieds-Adresse explizit setzen (überschreibt den Fallback dauerhaft). Löschen der Mitglieds-Adresse (alle drei Felder auf null) reaktiviert den Fallback.

### 9. Mitglied-Daten für verknüpften Nutzer
Wenn ein Nutzer (`spieler` oder `elternteil`) `GET /api/members/{id}` aufruft und `members.user_id == claims.UserID`, bekommt er die eigenen Admin-Felder (effektive Adresse, Eintrittsdatum, DSGVO, SEPA-Status) read-only zurück — aber nie die IBAN und nie `sepa_mandat_path`. Admin bekommt alles inkl. IBAN und SEPA-Dokument-URL.

## Risks / Trade-offs

- **Filesystem-Persistenz**: Dateien in `storage/uploads/` müssen bei Deployments erhalten bleiben. `make deploy` darf dieses Verzeichnis nicht überschreiben. → `rsync` überträgt nur das Binary, nicht das Storage-Verzeichnis; kein Risiko.
- **Datei-Orphans**: Wenn ein Mitglied gelöscht wird, bleibt die Foto-Datei auf dem Filesystem. → Akzeptabel für jetzt; ggf. später Cleanup-Job.
- **Disk-Limit**: Bei vielen Fotos (5 MB × 200 Mitglieder = ~1 GB) könnte Speicher eng werden. → Upload-Limit 5 MB + bevorzugt webp empfehlen. Monitoring per `df -h` manuell.
- **Kein Resize**: Fotos werden in Originalgröße gespeichert und ausgeliefert. Frontend muss mit CSS skalieren. → Scope-Entscheidung; akzeptabel.
- **SQLite ALTER TABLE**: SQLite unterstützt `ADD COLUMN` ohne Table-Rebuild — kein Risiko. Down-Migration erfordert Table-Rebuild (wie 015).

## Migration Plan

1. Migration 017: `ALTER TABLE members ADD COLUMN ...` (alle neuen member-Felder)
2. Migration 018: `ALTER TABLE users ADD COLUMN street/zip/city`; `CREATE TABLE user_phones`; `CREATE TABLE user_visibility`
3. `mkdir -p /var/lib/teamwerk/storage/uploads/{member-photos,user-photos,sepa-mandats}` + `chown www-data` im deploy-Script oder setup-vps.sh ergänzen
4. `make deploy` → Migrationen laufen automatisch
