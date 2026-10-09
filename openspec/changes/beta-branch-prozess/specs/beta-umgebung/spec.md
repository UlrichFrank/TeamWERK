## ADDED Requirements

### Requirement: Umgebungs-Parameter an allen Server-Targets
Alle Make-Targets, die auf den VPS wirken, SHALL über `ENV=prod|beta` die Ziel-Umgebung wählen; ohne `ENV` gilt `prod` mit unverändertem Verhalten. Ein anderer Wert MUSS sofort abbrechen. Bei `ENV=beta` MUSS jeder verwendete Pfad und Dienstname `teamwerk-beta` enthalten, sonst bricht das Target ab, bevor es den Server berührt. Targets, die es nur für Prod gibt (Server-Umzug), MÜSSEN `ENV=beta` ablehnen.

#### Scenario: Prod ohne Parameter unverändert
- **WHEN** `make deploy` ohne `ENV` aufgerufen wird
- **THEN** führt es dieselben Remote-Befehle aus wie vor der Einführung des Parameters

#### Scenario: Beta-Migration
- **WHEN** `make migrate-remote-up ENV=beta` aufgerufen wird
- **THEN** migriert es ausschließlich `/var/lib/teamwerk-beta/teamwerk.db` mit dem Beta-Binary als `www-data`

#### Scenario: Unbekannte Umgebung
- **WHEN** ein Target mit `ENV=staging` aufgerufen wird
- **THEN** bricht Make vor jedem Remote-Zugriff ab

#### Scenario: Beta-Backup getrennt
- **WHEN** `make backup ENV=beta` läuft
- **THEN** landet die Sicherung unter `backup/beta/<timestamp>/`, und `make restore-local` ohne `ENV` findet sie nicht

### Requirement: Branch-Bindung der Deploys
`make deploy` SHALL Prod nur von `main` oder einem Checkout eines `vX.Y.Z`-Tags deployen und die Beta nur vom Branch `beta`, jeweils nur ohne uncommittete Änderungen an versionierten Dateien. Die Prüfung MUSS vor dem Build laufen. `ALLOW_BRANCH=1` überspringt sie mit sichtbarer Warnung. `deploy-rollback` ist nicht gebunden.

#### Scenario: Beta-Deploy vom Feature-Branch
- **WHEN** `make deploy ENV=beta` auf einem anderen Branch als `beta` aufgerufen wird
- **THEN** bricht es ab, bevor gebaut oder der Server kontaktiert wird

#### Scenario: Prod-Deploy aus deploy.yml
- **WHEN** `deploy.yml` ein Release-Tag detached auscheckt und `make deploy` aufruft
- **THEN** besteht die Branch-Bindung

### Requirement: Automatischer Beta-Deploy
Ein Push auf den Branch `beta` SHALL über `deploy-beta.yml` dasselbe Gate wie der Prod-Deploy durchlaufen und danach `make deploy ENV=beta` ausführen; ein Smoke-Check gegen `https://beta.teamwerk.team-stuttgart.org/api/healthz` schließt ab. Der Workflow erzeugt kein Tag und kein Release und läuft in einer eigenen Concurrency-Gruppe, sodass er keinen Prod-Deploy blockiert.

#### Scenario: Merge nach beta
- **WHEN** ein Feature-Branch nach `beta` gemergt wird
- **THEN** wird nach grünem Gate die Beta-Instanz aktualisiert, Prod bleibt unberührt

#### Scenario: Gate rot
- **WHEN** ein Test im Gate des Beta-Workflows fehlschlägt
- **THEN** wird nicht deployt
