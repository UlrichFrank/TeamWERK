# Profile & Member Management Restructuring

## Ist-Stand (bereits implementiert)

### Datenbank
- Tabelle `member_change_drafts` (Migration 020): `id, member_id, field_name, old_value JSON, new_value JSON, created_at, created_by_user_id`
- UNIQUE(member_id, field_name)

### Backend-Handler
- `GET /api/members/{id}/change-drafts` — Draft-Liste abrufen ✓
- `POST /api/members/{id}/change-request` — Draft erstellen/updaten (UPSERT) ✓
- `POST /api/members/{id}/change-drafts/{draftId}/accept` — Draft akzeptieren ✓
- `DELETE /api/members/{id}/change-drafts/{draftId}` — Draft ablehnen/löschen ✓

### Frontend
- ProfilePage hat Tabs: Konto, Profil, Mitgliedsdaten, Sonstiges ✓
- MemberDetailPage hat Tabs: Stammdaten, Kontakt, Datenschutz, Familie, Admin ✓
- ProfileAccountTab: Name, Passwort (Modal), Email (Modal) ✓
- ProfileProfilTab: Adresse, Telefon, Foto, Sichtbarkeit, IBAN, Familie ✓
- ProfileMiscTab: Fahrzeug ✓

---

## Was noch fehlt (Scope dieser Änderung)

### A. ProfileMemberTab — Vollständige Neuimplementierung

Der Tab zeigt aktuell nur read-only Daten mit Dummy-Texten wie "Adresse (read-only)". Es fehlen:
- Echte Daten aus `ownMember` anzeigen
- Editierbare Felder zum Erstellen von Change-Requests
- Korrekte API-Calls zu `POST /members/{id}/change-request`

**Neue Struktur:**

```
┌──────────────────────────────────────────────────────┐
│ [Stammdaten] — read-only                            │
│   Vorname, Nachname, Geb.-Datum, Passnummer,        │
│   Rückennummer, Position, Status                    │
│   Falls Name-Draft: → Angefordert: [Wert] [Abbrechen]│
├──────────────────────────────────────────────────────┤
│ [Name ändern] — editable → Draft                   │
│   Vorname + Nachname (2 Felder, 1 Draft "name")     │
│   [Änderung anfordern] Button                       │
├──────────────────────────────────────────────────────┤
│ [IBAN] — editable → Draft                          │
│   IBAN-Feld, [Änderung anfordern] Button            │
│   Falls Draft: → Angefordert: DE... [Abbrechen]    │
└──────────────────────────────────────────────────────┘
```

**Datenquelle**: `ownMember` kommt als Prop aus ProfilePage (bereits via `/profile/me` geladen)

**Gespeichert via**:
- `POST /members/{id}/change-request` mit `{ field_name: "name", new_value: { first_name, last_name } }`
- `POST /members/{id}/change-request` mit `{ field_name: "iban", new_value: "DE..." }`
- `DELETE /members/{id}/change-drafts/{draftId}` — Abbrechen

---

### B. Backend — applyDraftToMember erweitern

`applyDraftToMember` in `internal/members/drafts.go` fehlen Felder:
- `email`: UPDATE members SET email = ?
- `dsgvo`: UPDATE members SET dsgvo_verarbeitung = ?, dsgvo_weitergabe = ?
- `sepa_mandat`: UPDATE members SET sepa_mandat = ?

`extractFieldValue` fehlen:
- `email`: members.email
- `dsgvo`: members.dsgvo_verarbeitung, members.dsgvo_weitergabe
- `sepa_mandat`: members.sepa_mandat

---

### C. MemberDetailPage — Drafts laden und weitergeben

MemberDetailPage lädt aktuell keine Drafts. Es fehlen:
- `GET /members/{id}/change-drafts` beim Laden aufrufen
- Drafts-State an alle Tab-Komponenten weitergeben
- Handler `handleDraftAccept(draftId)` → `POST .../accept`
- Handler `handleDraftReject(draftId)` → `DELETE .../change-drafts/{id}`

---

### D. MemberStammdatenTab — Draft-Anzeige für Name

Aktuell: `onDraftAccept={() => {}}` und `onDraftReject={() => {}}` sind no-ops.

Gewünschtes Verhalten: Wenn ein Name-Draft vorhanden, zeige unter Vorname/Nachname:
```
↓ Angefordert: Maximilian Muster  [✓ Annehmen] [✗ Ablehnen]
```

---

### E. MemberKontaktTab — Draft-Anzeige für Adresse + IBAN

Wenn Adresse-Draft oder IBAN-Draft vorhanden:
```
Adresse:
  Neue Straße 5, 71000 Ludwigsburg
  ↓ Angefordert: [Straße], [PLZ], [Ort]  [✓] [✗]

IBAN:
  ↓ Angefordert: DE89 370400440532013000  [✓] [✗]
```

---

### F. MemberDatenschutzTab — Draft-Anzeige für DSGVO + SEPA

Wenn DSGVO-Draft oder SEPA-Draft vorhanden:
```
Datenverarbeitung: ✓ (aktuell)
↓ Angefordert: ✗ (neu)  [✓ Annehmen] [✗ Ablehnen]
```

---

### G. MembersPage — ⏳-Indikator

Der `/api/members` Endpunkt muss einen `has_pending_drafts`-Boolean zurückgeben.
In der Mitgliederliste erscheint ein ⏳-Icon in der Zeile, wenn Drafts vorhanden.
Klick auf ⏳ → navigiert zu `/mitglieder/{id}` (Tab Stammdaten).

---

## Change-Request-Workflow (Zusammenfassung)

```
Nutzer (ProfileMemberTab):
  1. Feld ändern → Formular mit neuem Wert ausfüllen
  2. "Änderung anfordern" klicken
  3. POST /members/{id}/change-request → Draft gespeichert
  4. ⏳ Symbol erscheint mit "Angefordert: [Wert]" + [Abbrechen]-Button

Admin (MemberDetailPage):
  1. Sieht Draft-Box: "Angefordert: [Wert]"
  2. [✓ Annehmen] → POST .../accept → Original überschrieben, Draft gelöscht
  3. [✗ Ablehnen] → DELETE .../change-drafts/{id} → Draft gelöscht
```

## Felder die Drafts unterstützen

| Feld       | field_name   | new_value Format                                        |
|------------|--------------|----------------------------------------------------------|
| Name       | `name`       | `{ first_name: "...", last_name: "..." }`               |
| IBAN       | `iban`       | `"DE89 370400..."`                                      |
| Email      | `email`      | `"user@example.com"`                                    |
| DSGVO      | `dsgvo`      | `{ verarbeitung: true, weitergabe: false }`             |
| SEPA       | `sepa_mandat`| `true`                                                  |

*Adresse, Telefon, Foto werden direkt im Nutzerprofil (ProfileProfilTab) verwaltet — kein Draft nötig.*
