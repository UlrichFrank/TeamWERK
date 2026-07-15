## Why

Profilbilder werden aktuell in **zwei Datenfeldern parallel** gehalten:
`users.photo_path` (Nutzer-Strang) und `members.photo_path` (Mitglieder-Strang).
Je nachdem, wer das Foto pflegt, landet es an einer anderen Stelle:

- Kind pflegt eigenes Profil → schreibt `users.photo_path` via
  `POST /api/upload/user-photo`
- Elternteil pflegt Kind-Profil → schreibt `members.photo_path` via
  `POST /api/profile/kind/{memberId}/photo`
- Admin pflegt Mitglied → schreibt `members.photo_path` via
  `POST /api/upload/member-photo/{id}`

Ergebnis: Eltern sehen ein anderes Foto als das Kind selbst, weil beide
in unterschiedliche Slots schreiben. Fachlich soll es **ein Foto pro Person**
geben — und dieses Foto ist immer „das Foto des Nutzers".

## What Changes

**Modell:** `users.photo_path` wird die einzige Foto-Quelle. `members.photo_path`
und `members.photo_visible` entfallen. Ein Mitglied ohne User-Account hat kein
Foto — bewusste Entscheidung, keine Ersatz-Speicherung.

**Endpoint-Semantik (URLs bleiben stabil, Verhalten ändert sich):**

- `POST/DELETE /api/upload/user-photo` — unverändert: schreibt/löscht
  `users.photo_path` des eingeloggten Nutzers.
- `POST/DELETE /api/profile/kind/{memberId}/photo` — schreibt/löscht ab jetzt
  `users.photo_path` **des Kind-Users** (via `members.user_id`-Lookup).
  Bei `members.user_id IS NULL` antwortet der Endpoint mit HTTP 409
  („Kind hat keinen Account — Foto nicht möglich").
- `POST/DELETE /api/upload/member-photo/{id}` (Admin) — schreibt/löscht ebenfalls
  `users.photo_path` des zugehörigen Users. HTTP 409 wenn `members.user_id IS NULL`.

**Lesepfade:** Alle Konsumenten von `members.photo_path` (`GetProfile`,
`GetChildProfile`, `getMember`, `drafts.go`, Members-Liste, MemberDetail,
Carpooling, Roster, Spielberichte etc.) lesen `users.photo_path` via
`members.user_id`-Join. Wo `user_id IS NULL`, kein Foto.

**Response-Konsolidierung:** Die parallelen Felder `photo_url` (member) und
`user_photo_url` (user) in `MemberBase` werden zu einem einzigen `photo_url`
zusammengeführt.

**Migration** (siehe `design.md`):

1. Für jeden Member mit `user_id IS NOT NULL` und `members.photo_path IS NOT NULL`:
   - Wenn `users.photo_path IS NULL` → auf `users.photo_path` übernehmen.
   - Wenn `users.photo_path IS NOT NULL` → `users.photo_path` gewinnt, alte
     Member-Datei aus `uploadDir` löschen.
2. Für jeden Member mit `user_id IS NULL` und `members.photo_path IS NOT NULL`:
   Datei aus `uploadDir` löschen (kein Ziel-User).
3. Spalten `members.photo_path` und `members.photo_visible` entfernen.

**Sichtbarkeit:** `photo_visible` lebt bereits in `user_visibility`
(siehe `kind-profil-user-strang`) — `members.photo_visible` wird mit entfernt.
Die Fallback-Klausel in `kind-profil-user-strang` („bei `user_id IS NULL` in
`members` schreiben") entfällt für `photo_visible` (nicht für die anderen
Visibility-Felder).

## Capabilities

### Modified Capabilities

- **`kind-profil-user-strang`** — ergänzt um Foto-Requirement analog zu Account-
  und Visibility-Requirements: Elternteil-Upload für Kind schreibt Kind-User,
  HTTP 409 wenn Kind keinen Account hat. Fallback für `photo_visible` entfällt.
- **`profilbild-crop-upload`** — die bereits vorhandene Requirement
  „Crop-Modal ist an allen Foto-Upload-Stellen verfügbar" wird um die
  Präzisierung ergänzt, dass alle drei Upload-Stellen serverseitig in
  `users.photo_path` schreiben und HTTP 409 zurückgeben, wenn der Zielperson
  kein User-Account zugeordnet ist.

## Test-Anforderungen

| Route | Testname | Erwartung / Invariante |
|---|---|---|
| `POST /api/profile/kind/{id}/photo` | `UploadChildPhoto_SchreibtUserPhotoPath` | Nach Upload ist `users.photo_path` des Kind-Users gesetzt; `members.photo_path` existiert nicht mehr. |
| `POST /api/profile/kind/{id}/photo` | `UploadChildPhoto_OhneAccount_409` | Kind ohne `user_id` → HTTP 409, kein File-System-Write. |
| `DELETE /api/profile/kind/{id}/photo` | `DeleteChildPhoto_LoeschtUserPhotoPath` | `users.photo_path` des Kind-Users wird `NULL`, Datei entfernt. |
| `POST /api/upload/member-photo/{id}` | `UploadMemberPhoto_SchreibtUserPhotoPath` | Admin-Upload landet in `users.photo_path`. |
| `POST /api/upload/member-photo/{id}` | `UploadMemberPhoto_OhneAccount_409` | Member ohne `user_id` → HTTP 409. |
| `POST /api/upload/user-photo` | `UploadUserPhoto_UnveraendertesVerhalten` | Regression: eingeloggter Nutzer schreibt weiter eigenen `users.photo_path`. |
| `GET /api/profile/kind/{id}` | `GetChildProfile_ZeigtUserPhoto` | `member.photo_url` reflektiert `users.photo_path` (Kind-User), nicht `members.photo_path`. |
| `GET /api/members/{id}` | `GetMember_PhotoUrlAusUserStrang` | `photo_url` kommt aus `users.photo_path` via `user_id`-Join. |

**Garantierte Invariante:** Nach diesem Change gibt es systemweit **genau einen**
Speicherort für Profilbilder (`users.photo_path`) und **genau eine**
Antwort-URL (`photo_url`); `members.photo_path` und der doppelte
`user_photo_url`-Sonderpfad existieren nicht mehr.

## Impact

- **Backend:** Handler in `internal/upload/handler.go` (Upload/Delete-Trio),
  `internal/members/handler.go` (Lese-Queries, `MemberBase`-Struct),
  `internal/members/drafts.go`, `internal/carpooling/handler.go`.
- **Migration:** neue Nummer `029_photo_user_only.up.sql` + `.down.sql`
  — Datenübernahme + Spalten-Drop.
  Down-Migration stellt die Spalten wieder her, kopiert Daten aber nicht zurück
  (Data-Loss auf dem Down-Pfad ist bewusst; Backup vor Deploy zieht die
  Betriebs-Absicherung).
- **Frontend:** `MemberBase`-Konsumenten (`MembersPage.user_photo_url`-Sonderpfad,
  `MemberDetailPage`, `MemberStammdatenTab`) auf einheitliches `photo_url`
  umstellen. `ChildProfilePage`/`ProfileProfilTab` verhalten sich UX-seitig
  unverändert (gleiche Endpoints, gleiches Crop-Modal).
- **Datei-System:** Beim Migrationslauf werden verwaiste Member-Fotos
  physisch aus `uploadDir` gelöscht.
- **Kein Impact auf Endpoint-URLs oder Berechtigungen** — Auth-Tiers und
  Family-Link-Prüfungen bleiben identisch.
