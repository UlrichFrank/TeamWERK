# Proposal: beta-prod-spiegel

## Why

Die Beta-Instanz (`beta.teamwerk.team-stuttgart.org`, Change `kachel-eckfahne`) läuft bisher mit einer anonymisierten Kopie einer lokalen DB. Damit Vorstand, Trainer und Eltern neue Oberflächen mit **ihrem eigenen** Login und ihren echten Terminen, Diensten und Kadern ausprobieren können, braucht die Beta einen aktuellen Prod-Stand. Die harte Randbedingung: die Beta darf Prod **nie** beeinflussen — weder beim Spiegeln noch im Betrieb.

## What Changes

- Neues Skript `deploy/beta-mirror.sh` (läuft auf dem VPS, Daten verlassen den Server nicht) und Make-Target `make mirror-beta FORCE=1`.
- Gespiegelt werden die SQLite-DB, Dokumente (`files`), Beitragslauf-Protokolle und BWHV-Spielbericht-PDFs. **Nicht** gespiegelt werden Videos und alle Bild-Ablagen (`uploads` mit Profilfotos/SEPA-Mandaten, `media` mit Chat-Bildern, `training-diary`, `match-report-images`); deren Beta-Verzeichnisse werden geleert.
- Prod wird ausschließlich gelesen: DB über die SQLite-Online-Backup-API aus einer `-readonly`-Verbindung, alle Lese-/Kopierschritte als `www-data` mit `nice`/`ionice -c3`, jedes Schreibziel wird gegen das Präfix `/var/lib/teamwerk-beta` geprüft, gestoppt/gestartet wird nur `teamwerk-beta`.
- Die Kopie wird vor dem Einsetzen bereinigt: Push-Abos und Refresh-Tokens gelöscht, Wartungsmodus aus. Integritätsprüfung vor dem Austausch — scheitert sie, bleibt die Beta unverändert.
- Weiterhin gilt für die Beta: kein Mail, kein Push, kein BWHV-Abruf, kein Scheduler, keine Matomo-Erfassung — echte Nutzer bekommen aus der Beta keine Nachricht.
- `make seed-beta` (anonymisiert) bleibt als Alternative bestehen.

## Capabilities

- `beta-umgebung` (ADDED): Spiegelung der Prod-Daten.

## Impact

- Keine Änderung an Code, API oder Schema; nur Deploy-Werkzeug und Doku (`docs/agent/10-deployment.md`).
- Die Beta enthält danach echte personenbezogene Daten (inkl. Bank-/SEPA-Ciphertexte, die wie auf Prod nur mit der Tresor-Passphrase lesbar sind). Sie liegen auf demselben Server unter denselben Zugriffsrechten (`www-data`, Verzeichnis `0750`) wie die Prod-Daten; sie sind nicht im Backup.
- Echte Nutzer können sich mit ihrem Prod-Passwort auf der Beta anmelden. Eine Passwortänderung oder jede andere Mutation auf der Beta wirkt nur dort und wird beim nächsten Spiegeln überschrieben.

## Test-Anforderungen

Keine neuen Routen. Invarianten werden beim Lauf geprüft und im Ergebnis berichtet: Prod-Dienst läuft unverändert weiter (gleiche PID), im Prod-Verzeichnis entsteht keine root-eigene Datei, Beta antwortet auf `/api/healthz`, Beta-Login mit Prod-Konto funktioniert, Prod-Zähler (Nutzer, Spiele) unverändert.
