## 1. Spiegelung

- [x] 1.1 `deploy/beta-mirror.sh`: readonly Online-Backup als www-data, Integritätsprüfung, Bereinigung der Kopie, Pfad-Guards, nur `teamwerk-beta` stoppen/starten
- [x] 1.2 Dateiablagen `files`, `beitragslauf-protokolle`, `bwhv-reports` spiegeln; Bild-Ablagen der Beta leeren
- [x] 1.3 Make-Target `mirror-beta` (mit `FORCE=1`)
- [x] 1.4 Doku `docs/agent/10-deployment.md`
- [x] 1.5 Erster Lauf auf dem VPS, Prod-Invarianten geprüft
