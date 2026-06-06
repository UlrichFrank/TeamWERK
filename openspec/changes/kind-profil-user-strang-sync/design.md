## Context

Das System trennt Benutzerdaten in zwei Stränge:

- **User-Strang** (`users`-Tabelle + `user_phones` + `user_visibility`): Sofort editierbar, der User kontrolliert diese Daten selbst
- **Member-Strang** (`members`-Tabelle + `member_phones`): Offizielle Vereinsdaten, Änderungen erfordern Vorstand-Freigabe via Change-Draft

Beim eigenen Profil (`/profil`) wird dieses Modell korrekt umgesetzt: Name/Adresse werden sofort in `users` gespeichert, und zusätzlich ein Change-Draft für `members` erstellt. Telefonnummern und Sichtbarkeit schreiben direkt in `user_phones`/`user_visibility`.

Beim Kind-Profil (`/profil/kind/{memberId}`) fehlt dieser User-Strang-Zugriff. Alle Endpunkte lesen und schreiben ausschließlich die `members`-Tabelle, auch wenn das Kind einen eigenen User-Account hat. Das Elternteil agiert dabei als Proxy für das Kind.

## Goals / Non-Goals

**Goals:**
- Wenn Kind `user_id IS NOT NULL`: `GET /profile/kind/{id}` liefert `users`-Daten des Kindes zurück
- Neuer Endpoint `PUT /profile/kind/{id}/account` aktualisiert `users`-Datensatz des Kindes (sofort, kein Draft)
- Phones- und Visibility-Endpunkte zielen auf `user_phones`/`user_visibility` des Kindes
- Verhalten identisch zu `/profil` — Elternteil als Proxy für das Kind

**Non-Goals:**
- Kinder ohne User-Account: Phones und Visibility-Bearbeitung werden bewusst nicht unterstützt (kein User-Strang vorhanden)
- Keine Änderung am Change-Draft-Mechanismus für Name/Adresse
- Keine neuen Rollen oder Berechtigungsänderungen
- **Kein UI-Redesign**: Layout, Tabs, Komponenten und visueller Aufbau von `ChildProfilePage` bleiben unverändert. Nur die Datenquelle und die Save-Logik werden korrigiert. Neue Felder oder Abschnitte werden nicht eingeführt.

## Decisions

### Keine direkten members-Writes durch User

Die Grundregel ist absolut: kein User schreibt direkt in `members`-Daten. Nur Admins dürfen das. Deshalb:
- `user_id IS NOT NULL` → `user_phones`/`user_visibility` des Kindes (User-Strang)
- `user_id IS NULL` → HTTP 403 für Phones/Visibility-Endpunkte; UI blendet diese Abschnitte aus

**Alternative betrachtet:** Für Kinder ohne Account eine Art "Proxy-Draft" für Phones einführen. Abgelehnt — unnötige Komplexität; Eltern von Kindern ohne Account müssen den Admin kontaktieren.

### Neuer Endpoint `PUT /profile/kind/{memberId}/account`

Statt den bestehenden `PUT /profile/kind/{memberId}/member` zu erweitern, wird ein separater `/account`-Endpoint eingeführt — analog zu `PUT /profile/me` für eigene Profile. Das macht die Trennung User-Strang vs. Member-Strang explizit.

**Alternative betrachtet:** `/member`-Endpoint schreibt beides (users + draft). Abgelehnt — verletzt Single-Responsibility und erschwert das Verständnis des Workflows.

### Frontend ruft beide Endpoints auf

`ProfileProfilTab` im `mode="child"` ruft beim Speichern auf:
1. `PUT /profile/kind/{id}/account` (wenn `member.user_id` vorhanden) — sofort
2. `POST /members/{id}/change-request` mit `field_name: 'profil'` — immer

Das Frontend erhält `user_id` des Kindes über das `member`-Objekt (bereits im API-Response vorhanden).

### `GET /profile/kind/{memberId}` gibt `users`-Daten zurück

Der Endpoint gibt zusätzlich `user_first_name`, `user_last_name`, `user_street`, `user_zip`, `user_city`, `user_phones`, `user_visibility` zurück (nur wenn `user_id` vorhanden). Das Frontend priorisiert diese Felder für die Anzeige.

## Risks / Trade-offs

- **Datendopplung users vs. members bleibt bestehen** → Akzeptiert — Change-Draft-Prozess synchronisiert `members` bei Genehmigung
- **user_visibility-Tabelle muss für Kind-User existieren** → `GetChildProfile` erstellt beim Lesen keinen Eintrag, wenn noch keiner vorhanden ist. Fallback auf `false` für alle Visibility-Felder ist sicher.
- **Parallele Writes (account + change-request) können partiell fehlschlagen** → Geringes Risiko; kein Rollback nötig, da beide Operationen unabhängig sind. Retry durch erneutes Speichern idempotent.

## Migration Plan

- Keine DB-Migration erforderlich (alle Tabellen existieren)
- Deployment: normaler Build + Deploy-Prozess
- Rollback: vorheriger Binary-Stand, kein Datenverlust
