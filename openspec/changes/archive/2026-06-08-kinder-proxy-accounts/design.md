## Context

Das `users`-Datenmodell setzt aktuell eine gültige, eindeutige E-Mail-Adresse und `can_login = 1` für jeden Datensatz voraus. Kinder im Verein haben häufig keine eigene E-Mail-Adresse, erhalten daher keinen Account und fehlen im Dienstsystem (`duty_accounts`, `duty_assignments`) vollständig. Der bestehende `family_links`-Mechanismus (Elternteil → Mitglied) setzt bereits `members.user_id` voraus, um die RSVP-Logik für Kinder zu ermöglichen — ohne Proxy-Account fehlt diese Verknüpfung.

## Goals / Non-Goals

**Goals:**
- Kinder können einen `users`-Eintrag mit `can_login = 0` erhalten (Proxy-Account), ohne E-Mail-Adresse
- Der Proxy-Account dient als Ankerpunkt für Dienstkonten und RSVP-Logik
- Elternteile können auf der Dienstbörse Dienste stellvertretend für verknüpfte Kinder beanspruchen
- Admin kann für ein Mitglied einen Proxy-Account anlegen und diesen später zu einem vollständigen Login-Account aktivieren
- E-Mail-Eindeutigkeit gilt weiterhin für alle login-fähigen Accounts (`can_login = 1`)

**Non-Goals:**
- Kein eigenständiger Login für Proxy-Accounts (kein Session-Management, kein Passwort-Hash erforderlich)
- Kein automatisches Anlegen von Proxy-Accounts (immer manuell durch Admin)
- Keine Änderung der RSVP-Logik für Trainings und Spiele — die läuft weiterhin über `family_links` + `member_id`
- Kein Self-Service-Aktivierungsflow für Kinder (Admin-only)

## Decisions

### D1 — Partieller Unique-Index statt globales UNIQUE auf `email`

**Entscheidung:** `email` auf `users` wird nullable, der bisherige `UNIQUE(email)`-Constraint entfällt. Stattdessen ein partieller Index:
```sql
CREATE UNIQUE INDEX users_email_login_unique ON users(email)
WHERE can_login = 1 AND email IS NOT NULL;
```

**Rationale:** SQLite unterstützt partielle Unique-Indizes seit Version 3.8.9 (2015). Damit können mehrere Proxy-Accounts ohne E-Mail existieren, während Login-Accounts weiterhin eindeutige Adressen haben. Alternativen:
- *Sentinel-E-Mail (z.B. `proxy+123@intern`)*: Würde E-Mail-Felder mit Fake-Daten befüllen und in Passwort-Reset- und Einladungsflows lautlos fehlschlagen.
- *Separate `proxy_users`-Tabelle*: Würde alle FK-Referenzen auf `users.id` verdoppeln und die Abfragelogik stark verkomplizieren.

### D2 — SQLite Tabellen-Rebuild für Schema-Änderung

**Entscheidung:** Da SQLite kein `ALTER COLUMN` und kein `DROP CONSTRAINT` kennt, muss die Migration die `users`-Tabelle via Rename → Create → Insert → Drop neu aufbauen.

**Rationale:** Standard-Vorgehen bei SQLite-Schema-Änderungen. `PRAGMA foreign_keys = OFF` für die Dauer der Migration, danach wieder ON. `PRAGMA integrity_check` nach der Migration als Absicherung. Die Migration ist einmalig und läuft automatisch beim Binary-Start.

### D3 — Proxy-Account-Erstellung als eigener Admin-Endpoint

**Entscheidung:** Neuer Endpoint `POST /api/members/{id}/proxy-account`. Gibt den neu angelegten `user_id` zurück.

**Rationale:** Vermeidet die Komplexität, den bestehenden `POST /api/users`-Invite-Flow um `can_login = 0`-Logik zu erweitern. Der neue Endpoint ist minimal: Name des Mitglieds übernehmen, optionale E-Mail setzen, `can_login = 0`, `members.user_id` verknüpfen. Keine Einladungsmail.

### D4 — „Für wen?"-Selektor im Claim-Flow, nicht als globale Delegation

**Entscheidung:** Der Selektor erscheint nur, wenn das Elternteil mindestens ein verknüpftes Kind mit aktivem Proxy-Account hat. Der Dienst wird direkt dem gewählten `user_id` zugebucht — keine Indirektionsebene (kein `claimed_by` zusätzlich zu `user_id`).

**Rationale:** Einfachstes Datenmodell. Alternativ wäre ein `claimed_by_user_id`-Feld denkbar (wer hat geklickt vs. wer trägt den Dienst). Das würde die Abfragelogik in Dienstkonten und -börse verkomplizieren und wird erst bei späterem Auditbedarf relevant.

### D5 — Proxy-Account-Aktivierung Admin-only

**Entscheidung:** Ein Admin setzt `can_login = 1`, trägt eine gültige E-Mail ein und versendet optional eine Einladungsmail via bestehendem Invite-Flow.

**Rationale:** Kein separater „Aktivierungstoken"-Flow nötig. Der bestehende `invitation_tokens`-Mechanismus kann direkt genutzt werden, sobald der Account login-fähig ist.

## Risks / Trade-offs

**[Risiko] SQLite partial index — Verhalten bei NULL:** SQLite schließt NULL-Werte aus partiellen Indizes aus. Zwei Proxy-Accounts ohne E-Mail belegen denselben Index nicht — korrekt für unseren Fall.
→ Mitigation: Durch explizite `WHERE email IS NOT NULL`-Klausel im Index dokumentiert.

**[Risiko] Bestehende Queries mit `WHERE email = ?` ohne `can_login`-Filter:** Login, Passwort-Reset und E-Mail-Eindeutigkeitsprüfungen müssen explizit auf `can_login = 1` filtern, sonst blockieren Proxy-Accounts mit gleicher E-Mail den Login.
→ Mitigation: Alle betroffenen Queries in `internal/auth/handler.go` und `internal/members/handler.go` werden im Rahmen der Tasks geprüft und angepasst.

**[Risiko] Dienstbörse — Elternteil bucht Dienst für Kind, das das Soll nie erfüllen kann:** Ist das Kind noch zu jung für eigenständige Dienste, baut es formal ein Dienstkonto auf. Das ist gewollt, aber kommunikationsbedürftig.
→ Mitigation: Kein technisches Problem; organisatorisch durch Vereinsleitung zu regeln.

**[Risiko] Migration auf Production — Tabellen-Rebuild sperrt `users` kurz:** Bei < 200 Users auf einem SQLite-WAL-System vernachlässigbar (< 100 ms).
→ Mitigation: Migration läuft beim Binary-Start vor dem ersten Request-Handling; kein separater Downtime-Plan nötig.

## Migration Plan

1. `make migrate-up` lokal testen (SQLite Dev-DB)
2. `make build` + `make deploy` — der neue Binary führt `migrate up` automatisch aus
3. Nach Deploy: `PRAGMA integrity_check` remote via `sqlite3 /var/lib/teamwerk/teamwerk.db "PRAGMA integrity_check;"` prüfen
4. Rollback: `make migrate-down` (entfernt `can_login`, stellt UNIQUE-Constraint wieder her — Proxy-Accounts werden beim Down gelöscht, sofern keine FK-Konflikte bestehen)

## Open Questions

- **Soll die Proxy-Account-Erstellung direkt aus dem Mitglieds-Admin oder nur aus dem Familien-Tab zugänglich sein?** Aktuell: beide Stellen vorgesehen. Falls zu viel UI-Aufwand, nur Familien-Tab als MVP.
- **Proxy-Account-Name:** Vollname aus `members` übernehmen oder eigenes Namensfeld? Aktuell: `users.name = members.first_name + ' ' + members.last_name` bei Erstellung.
