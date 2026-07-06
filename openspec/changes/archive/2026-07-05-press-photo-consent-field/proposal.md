## Why

Der Spielbericht-Publisher (`internal/matchreports/photo_consent.go`) nutzt `members.photo_visible`
als Proxy für die Frage „darf auf öffentlichen Kanälen abgebildet werden". Das ist semantisch
falsch: `photo_visible` steuert die **interne** Sichtbarkeit des Profilbilds im Portal, nicht die
Einwilligung zur **öffentlichen** Veröffentlichung von Fotos (Homepage `team-stuttgart.org`,
Spielberichte). Es fehlt eine dedizierte, dokumentierte DSGVO-Einwilligung für Foto-Veröffentlichung.

## What Changes

- **Neues DSGVO-Feld** `members.foto_veroeffentlichung` (INTEGER 0/1) + `foto_veroeffentlichung_date`
  (DATE), analog zu `dsgvo_verarbeitung`/`dsgvo_weitergabe`.
- **Publisher-Umstellung:** `consentMissing` prüft künftig `foto_veroeffentlichung` statt
  `photo_visible`. Der Notlösungs-Kommentar entfällt.
- **UI Profil** (`ProfileDatenschutzTab`): neuer Schalter im DSGVO-Block (read-only, Änderung via
  „Kontakt"-Draft wie die anderen DSGVO-Felder). Zu **jedem** der drei Schalter
  (`dsgvo_verarbeitung`, `dsgvo_weitergabe`, `foto_veroeffentlichung`) eine erklärende Beschreibung,
  was er bedeutet.
- **UI Mitglieder-Verwaltung** (`MemberDatenschutzTab`): editierbarer Schalter für Vorstand +
  gleiche Erklärtexte; der `dsgvo`-Draft-Payload wird um `foto_veroeffentlichung` erweitert.
- **Backend:** `Member`-Response und Create/Update-Handler tragen das neue Feld; das `_date` wird
  gesetzt, wenn die Einwilligung von aus→an wechselt. Draft-Extraktion/-Apply für `field_name='dsgvo'`
  bezieht das Feld mit ein.
- **Migration** (`022`): Spalten anlegen; **Bestandsmitglieder** bekommen
  `foto_veroeffentlichung=1` + `foto_veroeffentlichung_date` = Migrationsdatum; **Neuanlage-Default
  = 0** (opt-in).

## Capabilities

### New Capabilities
- `press-photo-consent`: dediziertes DSGVO-Einwilligungsfeld für die öffentliche
  Foto-Veröffentlichung — Datenmodell, Create/Update/Draft-Verhalten, UI-Schalter mit Erklärtexten
  in Profil und Mitglieder-Verwaltung, und die Nutzung durch den Spielbericht-Publisher.

### Modified Capabilities
- `profile-datenschutz-tab`: der DSGVO-Block zeigt drei statt zwei Einwilligungen und ergänzt zu
  jedem Schalter einen Erklärtext.

## Impact

- **Migration:** `internal/db/migrations/022_press_photo_consent.up.sql` / `.down.sql`.
- **Backend:** `internal/members/handler.go` (Member-Struct, Create-INSERT, Update-UPDATE,
  Scan-Pfade), `internal/members/drafts.go` (`dsgvo`-Draft extract/apply),
  `internal/matchreports/photo_consent.go` (Consent-Query).
- **Frontend:** `web/src/components/profile/ProfileDatenschutzTab.tsx`,
  `web/src/components/admin/MemberDatenschutzTab.tsx`, ggf. `Member`-Typen in
  `pages/ProfilePage.tsx` / `MemberDetailPage.tsx` und der „Kontakt"-Tab, der den `dsgvo`-Draft baut.
- **Keine** neuen externen Dienste, kein nennenswerter RAM-Footprint.
