## Why

Spieler trainieren zu einem erheblichen Teil außerhalb der Mannschaftseinheiten — Kraftraum,
Laufen, Athletik-Programm, Reha nach Verletzung, und vor allem in der Sommerpause, wenn gar kein
Teamtraining stattfindet. Von all dem sieht TeamWERK heute nichts: `internal/trainings` kennt
ausschließlich terminierte Mannschaftseinheiten mit RSVP und Trainer-erfasster Anwesenheit.

Daraus folgen zwei Lücken. Der **Trainer** hat keinerlei Bild davon, wer in der trainingsfreien
Zeit etwas tut — genau die Information, die über die Vorbereitung auf die neue Saison entscheidet.
Der **Spieler** hat keinen Ort, an dem sein eigener Aufwand sichtbar wird und sich über eine
Saison zu etwas summiert; ohne Sichtbarkeit fehlt der Antrieb, dranzubleiben.

Ein Trainingstagebuch schließt beides mit einem Objekt: der Spieler erfasst selbst, was er getan
hat, hängt optional einen Nachweis an, und der Trainer bekommt eine Übersicht über seine
Mannschaft. Der Nachweis ist bewusst **weich** — fehlt er, ist das kein Fehlerzustand, sondern nur
ein weniger belastbarer Eintrag.

## What Changes

- **Neues Domain-Package `internal/trainingdiary`.** Ein Spieler erfasst eigene Trainingseinheiten
  mit Datum, Art, Dauer und Intensität. Die Art kommt aus einer festen Liste (`kraft`, `ausdauer`,
  `athletik`, `technik`, `beweglichkeit`, `reha`) oder ist Freitext (`sonstiges` + `kind_custom`).
  Intensität ist RPE 1–10 mit ausklappbarer Erklärung im UI.
- **Nachweis je Eintrag, auch nachträglich.** Eine optionale Datei pro Eintrag, über eigene Routen
  anfüg-, ersetz- und entfernbar. Bilder werden clientseitig auf **150 KB / längste Kante 1280 px**
  gedrückt (deutlich enger als die 1 MB / 1920 px im Chat), Server-Backstop bei 1 MB.
- **Eigener Store, getrennt von `internal/media`.** `media.Serve` prüft nur „ist eingeloggt" — jeder
  Nutzer kann Medien-IDs durchzählen. Für Trainingsnachweise gilt eine echte Zugriffsprüfung pro
  Objekt; Dateien liegen unter einem eigenen `TRAINING_DIARY_DIR`, nicht in der `media`-Tabelle.
- **Sichtbarkeit spiegelt `attendance.canSeeMemberStats`.** Zugriff auf fremde Einträge haben der
  Spieler selbst, seine Eltern über `family_links`, Trainer des Kaders in der **aktiven Saison**,
  `sportliche_leitung` und `admin`. Spieler sehen einander ausdrücklich **nicht**.
- **Trainer-Übersicht nach Muster Anwesenheit.** Team-Liste mit Kennzahlen je Spieler (Einheiten,
  Minuten, ⌀ RPE, Balken) und Drill-down in dessen Einzeleinträge inklusive Nachweisen — analog
  `AttendanceStatsView` mit seinen zwei Endpunkten.
- **Retention: das Bild verfällt, der Eintrag bleibt.** Ein täglicher Scheduler-Job löscht
  Nachweisdateien, deren Saison vor mehr als 90 Tagen endete, und markiert `proof_purged_at`.
  Datum, Art, Dauer und RPE bleiben dauerhaft als Historie erhalten.
- **Saison-Anker über die aktive Saison, nicht über das Datum.** `season_id` wird bei der Erfassung
  aus `seasons.is_active` gesetzt. Begründung siehe `design.md` — die datumsbasierte Zuordnung
  verliert ausgerechnet die Sommerpause.

**Explizite Nicht-Ziele:** Keine Verknüpfung mit `training_sessions` und keine Einrechnung in
`attendance-statistics` — das Tagebuch steht zunächst für sich (spätere Einblendung in die
Team-Termine ist ein eigener Change). Keine Trainingslast-Kennzahl (`Dauer × RPE`), solange
niemand weiß, wie sie zu lesen ist. Keine Prüf-/Freigabe-Workflows auf Nachweisen. Keine
serverseitige Bildkonvertierung — die bestehende clientseitige Kompression trägt in der Praxis,
wie der Chat zeigt.

## Capabilities

### New Capabilities

- `trainingstagebuch`: Erfassung eigener Trainingseinheiten — Datenmodell, Pflichtfelder,
  Artenkatalog mit Freitext-Zweig, RPE-Skala, CRUD auf eigenen Einträgen.
- `trainingstagebuch-sichtbarkeit`: Wer welche Einträge lesen darf, und die aggregierte
  Trainer-Sicht auf eine Mannschaft.
- `trainingstagebuch-nachweis`: Datei-Anhang je Eintrag — Upload (auch nachträglich), Kompression,
  MIME-Whitelist, ausgelieferte Bytes und deren Zugriffsschutz.
- `trainingstagebuch-retention`: Saison-Zuordnung der Einträge und die automatische Löschung der
  Nachweisdateien nach Saisonende.

### Modified Capabilities

- `permissions`: Die neun Tagebuch-Routen werden in die Persona-Matrix aufgenommen. Zwei neue
  Erwartungsklassen, weil beide Gates im Handler sitzen und nicht im Router-Tier: Schreiben ist
  reine Eigentümerschaft (keine Rolle kommt durch, auch `admin` nicht), Fremdlesen erlaubt
  zusätzlich Eltern, Kader-Trainer, `sportliche_leitung` und `admin` — **nicht** aber `vorstand`
  oder `kassierer`.

## Impact

**Datenbank**
- Migration `040_training_diary.up.sql` / `.down.sql`: Tabelle `training_diary_entries` mit
  `CHECK`-Constraints auf `kind`, `duration_min`, `rpe`; Indizes auf `(member_id, trained_on)` und
  `season_id`.

**Backend**
- Neu: `internal/trainingdiary/` (Handler, CRUD, Nachweis-Upload/-Serve, Team-Aggregation).
  Muss im Architektur-Test `internal/arch/arch_test.go` als **Domain**-Package klassifiziert werden.
- `internal/app/router.go`: neun Routen im Tier „Authenticated" (ACL im Handler, wie bei
  `attendance-stats`).
- `internal/config/`: `TRAINING_DIARY_DIR` (Default `./storage/training-diary`).
- `internal/scheduler/scheduler.go`: `runTrainingDiaryRetention()` im Daily-Block. Inline-SQL, da
  der Scheduler als Foundation kein Domain-Package importieren darf — Muster
  `deleteRetainedVideos`.

**Frontend**
- Neu: `pages/ProfilTrainingstagebuchPage.tsx`, `pages/TeamTrainingstagebuchPage.tsx`,
  `components/TrainingDiaryStatsView.tsx`, `components/RpeScaleInfo.tsx`,
  `components/TrainingDiaryEntryForm.tsx`.
- `App.tsx`: `profil/trainingstagebuch` (alle), `trainingstagebuch` + `team/:id/trainingstagebuch`
  (`RoleRoute roles={['admin','trainer','sportliche_leitung']}`).
- `AppShell.tsx`: Nav-Einträge.
- Tagebuch-Tab in `ProfilePage` und `ChildProfilePage` (Muster: bestehender Anwesenheits-Tab).
- `lib/imageCompress.ts` wird **unverändert** wiederverwendet, nur mit engeren `opts`.

**Betrieb**
- Neues Storage-Verzeichnis gehört ins Backup — wie `BEITRAGSLAUF_DIR`. Nachweise sind nach
  Saisonende + 90 Tage weg; ein älteres Backup zurückzuspielen holt gelöschte Bilder zurück.
- Speicherbedarf grob: 500 Einträge/Saison × 150 KB ≈ **75 MB**, die rollierend wieder freiwerden.

**Risiko / Regression**
- Kein Bestandsverhalten wird verändert: neue Tabelle, neue Routen, neue Seiten. Der einzige
  Eingriff in geteilten Code ist ein zusätzlicher Aufruf von `compressImage` mit eigenen Optionen —
  die Funktion selbst bleibt unangetastet, Chat/Mitteilungen/Spielberichte sind nicht betroffen.
- HEIC-Dateien, die eine Browser-Engine nicht dekodieren kann (Chrome/Firefox), rutschen
  unkomprimiert bis zur Server-Whitelist durch und werden dort mit HTTP 400 abgelehnt. Das ist
  identisch zum heutigen Chat-Verhalten und bewusst akzeptiert: iOS wandelt Fotos beim Datei-Picker
  praktisch immer nach JPEG.

## Test-Anforderungen

| Route | Testname | Erwartung | Garantierte Invariante |
|---|---|---|---|
| `POST /api/training-diary` | `TestCreateEntry_Success` | 201 | Happy-Path: Eintrag liegt mit `member_id` des Aufrufers und aktiver `season_id` in der DB |
| `POST /api/training-diary` | `TestCreateEntry_CustomKindRequiresText` | 400 | `kind='sonstiges'` ohne `kind_custom` wird abgelehnt |
| `POST /api/training-diary` | `TestCreateEntry_InvalidRPE` | 400 | RPE außerhalb 1–10 wird abgelehnt |
| `POST /api/training-diary` | `TestCreateEntry_FutureDate` | 400 | `trained_on` in der Zukunft wird abgelehnt |
| `POST /api/training-diary` | `TestCreateEntry_NoMemberForUser` | 403 | Nutzer ohne Mitglieds-Datensatz kann nicht erfassen |
| `PUT /api/training-diary/{id}` | `TestUpdateEntry_ForeignEntry` | 403 | Fremde Einträge sind nicht änderbar, auch nicht durch Trainer |
| `DELETE /api/training-diary/{id}` | `TestDeleteEntry_Success` | 204 | Eigener Eintrag samt Nachweisdatei entfernt |
| `DELETE /api/training-diary/{id}` | `TestDeleteEntry_ForeignEntry` | 403 | — |
| `GET /api/training-diary` | `TestListOwn_OnlyOwnEntries` | 200 | Antwort enthält ausschließlich Einträge des Aufrufers |
| `POST /api/training-diary/{id}/proof` | `TestUploadProof_Success` | 201 | Nachträglicher Upload setzt `proof_disk_name`, Datei liegt in `TRAINING_DIARY_DIR` |
| `POST /api/training-diary/{id}/proof` | `TestUploadProof_UnsupportedType` | 400 | MIME-Whitelist greift (u. a. `image/heic`), keine Datei geschrieben |
| `POST /api/training-diary/{id}/proof` | `TestUploadProof_TooLarge` | 413 | 1-MB-Backstop greift |
| `POST /api/training-diary/{id}/proof` | `TestUploadProof_ForeignEntry` | 403 | Kein Fremd-Upload, auch nicht durch den Trainer |
| `POST /api/training-diary/{id}/proof` | `TestUploadProof_ReplacesOld` | 201 | Alte Datei wird von der Platte entfernt, kein Waisen-Blob |
| `GET /api/training-diary/{id}/proof` | `TestServeProof_Owner` | 200 | Eigentümer bekommt die Bytes mit korrektem Content-Type |
| `GET /api/training-diary/{id}/proof` | `TestServeProof_TrainerOfKader` | 200 | Trainer des Kaders der aktiven Saison darf lesen |
| `GET /api/training-diary/{id}/proof` | `TestServeProof_OtherPlayer` | 403 | **Kern-Invariante:** Spieler sehen einander nicht |
| `GET /api/training-diary/{id}/proof` | `TestServeProof_TrainerOtherTeam` | 403 | Trainer fremder Mannschaften sind ausgeschlossen |
| `GET /api/training-diary/{id}/proof` | `TestServeProof_Purged` | 410 | Nach der Retention liefert die Route „Gone", keine 500 |
| `GET /api/members/{id}/training-diary` | `TestMemberDiary_ParentAccess` | 200 | Elternteil über `family_links` liest das Kind |
| `GET /api/members/{id}/training-diary` | `TestMemberDiary_OtherPlayer` | 403 | — |
| `GET /api/members/{id}/training-diary` | `TestMemberDiary_SportlicheLeitung` | 200 | sL liest vereinsweit |
| `GET /api/teams/{id}/training-diary-stats` | `TestTeamStats_TrainerOwnTeam` | 200 | Aggregat je Kadermitglied (Einheiten, Minuten, ⌀ RPE) |
| `GET /api/teams/{id}/training-diary-stats` | `TestTeamStats_PlayerForbidden` | 403 | Spieler bekommen keine Team-Übersicht |
| `GET /api/teams/{id}/training-diary-stats` | `TestTeamStats_TrainerForeignTeam` | 403 | — |
| Scheduler (Go) | `TestTrainingDiaryRetention_PurgesAfterSeasonEnd` | — | Saisonende älter als 90 Tage → Datei weg, `proof_purged_at` gesetzt, Eintragsfelder unverändert |
| Scheduler (Go) | `TestTrainingDiaryRetention_KeepsWithinWindow` | — | Saisonende vor 89 Tagen → Datei bleibt |
| Scheduler (Go) | `TestTrainingDiaryRetention_NullSeasonNeverPurged` | — | `season_id IS NULL` wird nie gelöscht |
| Scheduler (Go) | `TestTrainingDiaryRetention_Idempotent` | — | Zweiter Lauf ändert nichts und wirft nicht |
| Arch (Go) | `arch_test.go` | — | `internal/trainingdiary` ist als Domain klassifiziert und importiert kein anderes Domain-Package |
| Broadcast-Gate (Go) | `broadcast_test.go` | — | Alle sechs Mutations-Routen broadcasten, ohne Allowlist-Eintrag |
| Frontend (Vitest) | `TrainingDiaryEntryForm.test.tsx` | — | `sonstiges` blendet das Freitextfeld ein; Speichern ohne Freitext ist blockiert |
| Frontend (Vitest) | `TrainingDiaryEntryForm.compress.test.tsx` | — | `compressImage` wird mit `targetBytes: 153600`, `maxEdge: 1280` aufgerufen |
| Frontend (Vitest) | `RpeScaleInfo.test.tsx` | — | Info-Box ist eingeklappt und per Klick ausklappbar |
| Frontend (Vitest) | `TrainingDiaryStatsView.test.tsx` | — | Team-Liste rendert Kennzahlen; Klick auf eine Zeile lädt die Detailansicht nach |
| Frontend (Vitest) | `ProfilTrainingstagebuchPage.test.tsx` | — | Abgelaufener Nachweis rendert den Hinweis „Nachweis gelöscht", kein defektes `<img>` |
