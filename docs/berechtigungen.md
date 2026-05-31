# TeamWERK — Berechtigungsübersicht

## Zweischichtiges Modell

Das System trennt zwei unabhängige Konzepte:

| Schicht | Datenbankfeld | Werte |
|---|---|---|
| **Systemrolle** | `users.role` | `admin` \| `standard` |
| **Vereinsfunktion** | `member_club_functions.function` (n:m) | `spieler` \| `trainer` \| `vorstand` \| `vorstand_beisitzer` |

**Schlüsselregel:** `admin` umgeht alle Vereinsfunktions-Prüfungen automatisch. Ein Nutzer mit Systemrolle `admin` hat Zugriff auf alle Endpunkte, unabhängig von seiner Vereinsfunktion.

Einem Nutzer können mehrere Vereinsfunktionen gleichzeitig zugewiesen sein (z. B. `trainer` + `vorstand`).

---

## Zugriffsstufen

### Öffentlich (kein Login erforderlich)

| Endpunkt | Beschreibung |
|---|---|
| `POST /api/auth/login` | Login |
| `POST /api/auth/refresh` | Token-Refresh |
| `POST /api/auth/logout` | Logout |
| `POST /api/auth/request-membership` | Beitrittsantrag stellen |
| `POST /api/auth/register` | Registrierung per Einladungs-Token |
| `POST /api/auth/forgot-password` | Passwort-Reset anfordern |
| `POST /api/auth/reset-password` | Passwort zurücksetzen |
| `GET /api/profile/email/confirm` | E-Mail-Änderung bestätigen |
| `GET /api/uploads/*` | Datei-Uploads abrufen (Fotos, PDFs) |

---

### Alle eingeloggten Nutzer

Gilt für alle Nutzer mit Systemrolle `standard` oder `admin`.

| Bereich | Endpunkt | Beschreibung |
|---|---|---|
| **Mitglieder** | `GET /api/members` | Mitgliederliste |
| | `GET /api/members/{id}` | Mitglied-Detailansicht |
| | `GET /api/members/{id}/change-drafts` | Eigene Änderungsanträge lesen |
| | `POST /api/members/{id}/change-request` | Änderungsantrag stellen |
| **Eigenes Profil** | `GET /PUT /api/profile/me` | Profildaten lesen/schreiben |
| | `GET /PUT /api/profile/vehicle` | Fahrzeuginfo |
| | `GET /PUT /api/profile/account` | Kontodaten |
| | `POST /api/profile/password` | Passwort ändern |
| | `POST /api/profile/email` | E-Mail-Änderung anfordern |
| | `POST /PUT /DELETE /api/profile/phones/{id}` | Telefonnummern verwalten |
| | `PUT /api/profile/visibility` | Sichtbarkeitseinstellungen |
| | `POST /api/upload/user-photo` | Profilfoto hochladen |
| **Dashboard** | `GET /api/dashboard` | Dashboard-Daten |
| **Dienstbörse** | `GET /api/duty-board` | Offene Dienst-Slots |
| | `POST /api/duty-board/{slotId}/claim` | Dienst übernehmen |
| | `DELETE /api/duty-board/{slotId}/claim` | Dienst zurückgeben |
| | `GET /api/duty-accounts` | Dienst-Kontostand |
| | `GET /api/duty-slots` | Slot-Liste |
| | `GET /api/duty-slots/{id}/assignments` | Slot-Belegungen |
| **Mitfahrgelegenheiten** | `GET /api/mitfahrgelegenheiten` | Angebote anzeigen |
| | `POST /api/mitfahrgelegenheiten` | Angebot erstellen/aktualisieren |
| | `DELETE /api/mitfahrgelegenheiten/{id}` | Angebot löschen |
| | `POST /api/mitfahrt-paarungen` | Mitfahrt anfragen |
| | `POST /api/mitfahrt-paarungen/{id}/confirm` | Mitfahrt bestätigen |
| | `POST /api/mitfahrt-paarungen/{id}/reject` | Mitfahrt ablehnen |
| **Push-Notifications** | `GET /api/push/vapid-public-key` | VAPID-Key abrufen |
| | `POST /api/push/subscribe` | Push-Abo anlegen |
| | `DELETE /api/push/subscribe` | Push-Abo löschen |
| **Kalender** | `GET /api/kalender` | Spielplan lesen |
| | `GET /api/kalender/{id}` | Spiel-Detailansicht |
| **Teams** | `GET /api/teams` | Team-Liste (gefiltert nach Rolle) |
| **Live-Updates** | `GET /api/events` | Server-Sent Events (SSE) |

---

### Vereinsfunktion `trainer` (und immer: `admin`)

| Bereich | Endpunkt | Beschreibung |
|---|---|---|
| **Dienst-Slots** | `POST /api/duty-slots` | Slot anlegen |
| | `PUT /api/duty-slots/{id}` | Slot bearbeiten |
| | `DELETE /api/duty-slots/{id}` | Slot löschen |
| **Dienst-Assignments** | `POST /api/duty-assignments/{id}/fulfill` | Dienst als erfüllt markieren |
| | `POST /api/duty-assignments/{id}/cash-substitute` | Geldersatz buchen |
| **Beitrittsanträge** | `GET /api/admin/membership-requests` | Anträge auflisten |
| | `POST /api/admin/membership-requests/{id}/approve` | Antrag genehmigen |
| | `POST /api/admin/membership-requests/{id}/reject` | Antrag ablehnen |
| | `DELETE /api/admin/membership-requests/{id}` | Antrag löschen |
| **Einladungen** | `POST /api/auth/invite` | Einladung versenden |

---

### Vereinsfunktion `vorstand` oder `trainer` (und immer: `admin`)

| Bereich | Endpunkt | Beschreibung |
|---|---|---|
| **Spielplan** | `POST /api/admin/kalender` | Spiel anlegen |
| | `PUT /api/admin/kalender/{id}` | Spiel bearbeiten |
| | `DELETE /api/admin/kalender/{id}` | Spiel löschen |
| | `POST /api/admin/kalender/{id}/regenerate` | Dienst-Slots regenerieren |
| | `POST /api/admin/kalender/regenerate-day` | Tages-Slots regenerieren |
| **Änderungsanträge** | `POST /api/members/{id}/change-drafts/{draftId}/accept` | Antrag akzeptieren |
| | `DELETE /api/members/{id}/change-drafts/{draftId}` | Antrag ablehnen |
| **Altersklassen** | `GET /api/admin/age-class-rules` | Regeln lesen |
| **Saisons** | `GET /api/admin/seasons` | Saison-Liste |
| **Kader** | `GET /api/admin/kader` | Kader-Liste |
| | `POST /api/admin/kader` | Kader initialisieren |
| | `GET /api/admin/kader/{id}` | Kader-Detailansicht |
| | `PUT /api/admin/kader/{id}` | Kader bearbeiten |
| | `DELETE /api/admin/kader/{id}` | Kader löschen |
| | `GET /api/admin/kader/{id}/member-suggestions` | Mitglieder-Vorschläge |
| | `PATCH /api/admin/kader/{id}/games-per-season` | Spiele pro Saison setzen |
| | `POST /api/admin/kader/copy-from-season` | Kader aus Saison kopieren |
| | `POST /api/admin/kader/auto-assign` | Kader automatisch zuweisen |

---

### Vereinsfunktion `vorstand` (und immer: `admin`)

| Bereich | Endpunkt | Beschreibung |
|---|---|---|
| **Mitglieder** | `POST /api/members` | Mitglied anlegen |
| | `PUT /api/members/{id}` | Mitglied bearbeiten |
| | `PUT /api/members/{id}/status` | Status ändern |
| | `GET /api/members/export` | Mitgliederliste exportieren |
| | `POST /api/members/import` | Mitglieder importieren |
| | `DELETE /api/admin/members/{id}` | Mitglied löschen |
| | `PUT /api/admin/members/{id}/user` | User-Verknüpfung setzen |
| | `POST /api/admin/members/{id}/welcome-email` | Willkommensmail senden |
| | `GET /api/admin/members/{id}/parents` | Elternteile abrufen |
| | `POST /api/admin/users/{id}/create-member` | Mitglied aus User anlegen |
| **Familien-Links** | `POST /api/admin/family-links` | Eltern-Kind-Verknüpfung anlegen |
| | `DELETE /api/admin/family-links` | Verknüpfung löschen |
| **Vereinskonfiguration** | `GET /api/admin/club` | Vereinsdaten lesen |
| | `PUT /api/admin/club` | Vereinsdaten bearbeiten |
| **Saisonverwaltung** | `POST /api/admin/seasons` | Saison anlegen |
| | `PUT /api/admin/seasons/{id}` | Saison bearbeiten |
| | `PUT /api/admin/seasons/{id}/activate` | Saison aktivieren |
| | `DELETE /api/admin/seasons/{id}` | Saison löschen |
| | `PUT /api/admin/seasons/{id}/duty-targets` | Dienst-Ziele setzen |
| **Team-Verwaltung** | `GET /api/admin/teams` | Team-Liste |
| | `POST /api/admin/teams` | Team anlegen |
| | `PUT /api/admin/teams/{id}` | Team bearbeiten |
| | `POST /api/admin/teams/{id}/assign-trainer` | Trainer zuweisen |
| **Nutzerverwaltung** | `GET /api/admin/users` | Nutzer-Liste |
| | `PUT /api/admin/users/{id}/role` | Systemrolle ändern |
| | `DELETE /api/admin/users/{id}` | Nutzer löschen |
| **Einladungsverwaltung** | `GET /api/admin/invitations` | Einladungen auflisten |
| | `DELETE /api/admin/invitations/{id}` | Einladung löschen |
| **Diensttypen** | `GET /api/admin/duty-types` | Diensttypen auflisten |
| | `POST /api/admin/duty-types` | Diensttyp anlegen |
| | `PUT /api/admin/duty-types/{id}` | Diensttyp bearbeiten |
| | `DELETE /api/admin/duty-types/{id}` | Diensttyp löschen |
| **Dienst-Konten** | `GET /api/admin/duty-accounts/export` | Konten exportieren |
| **Dienst-Templates** | `GET /api/admin/duty-templates` | Templates auflisten |
| | `POST /api/admin/duty-templates` | Template anlegen |
| | `GET /api/admin/duty-templates/{id}` | Template lesen |
| | `PUT /api/admin/duty-templates/{id}` | Template bearbeiten |
| | `DELETE /api/admin/duty-templates/{id}` | Template löschen |
| | `GET /api/admin/duty-templates/{id}/preview` | Slot-Vorschau |
| **Datei-Uploads** | `POST /api/upload/member-photo/{id}` | Mitgliederfoto hochladen |
| | `POST /api/upload/sepa-mandat/{id}` | SEPA-Mandat hochladen |
| **Altersklassen** | `PUT /api/admin/age-class-rules/{ageClass}` | Regelwerk bearbeiten |

---

## Übersichtsmatrix

| Funktion | eingeloggt | trainer | vorstand\|trainer | vorstand | admin |
|---|:---:|:---:|:---:|:---:|:---:|
| Login / Registrierung | ✓ | ✓ | ✓ | ✓ | ✓ |
| Eigenes Profil verwalten | ✓ | ✓ | ✓ | ✓ | ✓ |
| Mitgliederliste lesen | ✓ | ✓ | ✓ | ✓ | ✓ |
| Dienstbörse nutzen | ✓ | ✓ | ✓ | ✓ | ✓ |
| Kalender lesen | ✓ | ✓ | ✓ | ✓ | ✓ |
| Mitfahrgelegenheiten | ✓ | ✓ | ✓ | ✓ | ✓ |
| Dienst-Slots verwalten | — | ✓ | ✓ | ✓ | ✓ |
| Beitrittsanträge verwalten | — | ✓ | ✓ | ✓ | ✓ |
| Einladungen versenden | — | ✓ | ✓ | ✓ | ✓ |
| Spielplan verwalten | — | — | ✓ | ✓ | ✓ |
| Kader verwalten | — | — | ✓ | ✓ | ✓ |
| Mitglieder anlegen / löschen | — | — | — | ✓ | ✓ |
| Nutzerverwaltung | — | — | — | ✓ | ✓ |
| Vereinskonfiguration | — | — | — | ✓ | ✓ |
| Saisonverwaltung | — | — | — | ✓ | ✓ |
| Diensttypen & Templates | — | — | — | ✓ | ✓ |
| Datenexport / -import | — | — | — | ✓ | ✓ |
| Alle Rechte | — | — | — | — | ✓ |
