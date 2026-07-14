## MODIFIED Requirements

### Requirement: Crop-Modal ist an allen Foto-Upload-Stellen verfügbar

Das Crop-Modal MUST einheitlich eingebunden sein bei: eigenem Profilbild (`ProfileProfilTab`), Kinderfoto durch Elternteil (`ChildProfilePage`), und Mitgliedsfoto durch Admin (`MemberStammdatenTab`). Alle drei Upload-Pfade MUST serverseitig `users.photo_path` als einzige Foto-Quelle schreiben (Kind-Upload und Admin-Upload via `members.user_id`-Lookup). Sind die Zielperson und kein User-Account verknüpft (Member ohne `user_id`), MUST der Server mit HTTP 409 antworten; die UI MUST diesen Fehler in eine klare Meldung „Foto benötigt einen Account" übersetzen.

#### Scenario: Eigenes Profilbild

- **WHEN** ein eingeloggter Nutzer auf der Profil-Seite ein Foto auswählt
- **THEN** öffnet sich das Crop-Modal
- **AND** nach Bestätigung landet die Datei über `POST /api/upload/user-photo` in `users.photo_path` des Nutzers

#### Scenario: Kinderfoto durch Elternteil

- **WHEN** ein Elternteil auf der Kind-Profil-Seite ein Foto für ein Kind auswählt
- **THEN** öffnet sich das Crop-Modal
- **AND** nach Bestätigung landet die Datei über `POST /api/profile/kind/{memberId}/photo` in `users.photo_path` des Kind-Users
- **AND** hat das Kind keinen User-Account, antwortet der Server mit HTTP 409 und die UI zeigt „Foto benötigt einen Account"

#### Scenario: Mitgliedsfoto durch Admin

- **WHEN** ein Admin in den Stammdaten eines Mitglieds ein Foto auswählt
- **THEN** öffnet sich das Crop-Modal
- **AND** nach Bestätigung landet die Datei über `POST /api/upload/member-photo/{id}` in `users.photo_path` des mit dem Member verknüpften Users
- **AND** hat das Mitglied keinen User-Account, antwortet der Server mit HTTP 409 und die UI zeigt „Foto benötigt einen Account"
