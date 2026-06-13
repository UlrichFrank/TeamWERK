## ADDED Requirements

### Requirement: Nearest-Ancestor-Wins Berechtigungsauflösung
Das System SHALL die Zugriffsrechte auf einen Ordner ausschließlich anhand der Berechtigungen des nächsten Vorfahren (inkl. des Ordners selbst) bestimmen, der eigene Berechtigungseinträge besitzt. Berechtigungen weiter entfernter Vorfahren MÜSSEN ignoriert werden, sobald ein näherer Vorfahre eigene Einträge hat. Ein Ordner ohne Berechtigungseinträge an keiner Stelle seiner Vorfahren-Kette gilt als nicht zugänglich.

#### Scenario: Unterordner mit eigenen Regeln schlägt Elternregel
- **WHEN** ein Elternordner `everyone: can_read=true` hat und ein Unterordner `role=vorstand: can_read=true` hat
- **THEN** darf ein Standard-Nutzer (kein Vorstand) den Unterordner NICHT lesen (403)

#### Scenario: Unterordner ohne eigene Regeln erbt vom Elternordner
- **WHEN** ein Elternordner `everyone: can_read=true` hat und ein Unterordner keine eigenen Berechtigungen hat
- **THEN** darf ein beliebiger eingeloggter Nutzer den Unterordner lesen (200)

#### Scenario: Unterordner ohne Regeln erbt vom nächsten Vorfahren mit Regeln
- **WHEN** Root `everyone: can_read=true` hat, Kind keine Regeln, Enkel keine Regeln
- **THEN** darf ein beliebiger eingeloggter Nutzer den Enkel lesen (200)

#### Scenario: Restriktiver Unterordner, sein Kind erbt Restriktion
- **WHEN** Elternordner `club_function=vorstand: can_read=true`, Unterordner A hat eigene Regeln für Vorstand, Unterordner A/B hat keine eigenen Regeln
- **THEN** erbt A/B von A (nur Vorstand darf lesen); Elternregel spielt keine Rolle mehr

#### Scenario: Ordner ohne jegliche Vorfahren-Regeln ist nicht zugänglich
- **WHEN** weder der Ordner noch eines seiner Vorfahren Berechtigungseinträge hat
- **THEN** erhält jeder Nicht-Admin-Nutzer 403

#### Scenario: Admin hat immer Vollzugriff
- **WHEN** ein Nutzer `role=admin` hat
- **THEN** darf er jeden Ordner lesen und schreiben, unabhängig von Berechtigungseinträgen

### Requirement: Family-Context für club_function-Berechtigungen
Das System SHALL einem Nutzer Lesezugriff auf einen Ordner gewähren, wenn der Ordner `principal_type=club_function` mit einem bestimmten Wert hat und der Nutzer über `family_links` mit einem Mitglied verknüpft ist, dessen Nutzerkonto diese Vereinsfunktion trägt — auch wenn der anfragende Nutzer selbst diese Funktion nicht hat.

#### Scenario: Elternteil liest Spieler-Ordner über family_links
- **WHEN** Ordner hat `club_function=spieler: can_read=true` und Nutzer U ist via `family_links` mit Mitglied M verknüpft, dessen User-Account `club_function=spieler` hat
- **THEN** darf Nutzer U den Ordner lesen (200)

#### Scenario: Nutzer ohne family_links zur gesuchten Funktion hat keinen Zugriff
- **WHEN** Ordner hat `club_function=spieler: can_read=true` und Nutzer U hat selbst nicht `club_function=spieler` und hat keine family_links zu einem Spieler
- **THEN** erhält Nutzer U 403

### Requirement: Family-Context für user-ID-Berechtigungen
Das System SHALL einem Nutzer Lesezugriff auf einen Ordner gewähren, wenn der Ordner `principal_type=user` mit einer bestimmten User-ID hat und der anfragende Nutzer über `family_links` mit einem Mitglied verknüpft ist, dessen `user_id` dieser ID entspricht.

#### Scenario: Elternteil liest Ordner der explizit für sein Kind freigegeben ist
- **WHEN** Ordner hat `user=42: can_read=true` und Nutzer P ist via `family_links` mit Mitglied M verknüpft, dessen `user_id=42`
- **THEN** darf Nutzer P den Ordner lesen (200)

#### Scenario: Elternteil ohne Verknüpfung zu User 42 hat keinen Zugriff
- **WHEN** Ordner hat `user=42: can_read=true` und Nutzer P hat keine family_links zu User 42
- **THEN** erhält Nutzer P 403

## Test-Anforderungen

- `resolveAccess`: TestResolveAccess_NearestAncestorWins (Unterordner-Regel schlägt Eltern-Regel)
- `resolveAccess`: TestResolveAccess_InheritFromParent (Unterordner ohne Regeln erbt)
- `resolveAccess`: TestResolveAccess_NoRulesAnywhere (kein Zugriff ohne Einträge)
- `resolveAccess`: TestResolveAccess_FamilyContext_ClubFunction (Elternteil via club_function)
- `resolveAccess`: TestResolveAccess_FamilyContext_UserID (Elternteil via user-ID)
- Route `GET /api/folders/{id}/contents`: TestFolderContents_RestrictedSubfolder (403 für Standard-User)
