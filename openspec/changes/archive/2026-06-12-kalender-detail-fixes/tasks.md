## 1. EventInfoModal — Interface-Erweiterungen

- [x] 1.1 `Game`-Interface in `EventInfoModal.tsx` um `end_date?: string | null` und `teams?: Array<{ id: number; name: string }>` erweitern
- [x] 1.2 `Training`-Interface in `EventInfoModal.tsx` um `team_name?: string` erweitern

## 2. EventInfoModal — Rendering-Fixes

- [x] 2.1 Datumsanzeige für Spiele: bei `end_date` (das vom `date` abweicht) eine Datumsspanne rendern (z.B. "7. September – 10. September 2026")
- [x] 2.2 Label "Gegner" → "Event-Name" für `event_type === 'generisch'`
- [x] 2.3 Team-Zeile in der Spieldetail-Ansicht einfügen: `game.teams` als kommagetrennte Kurznamen, nur wenn Array nicht leer
- [x] 2.4 Team-Zeile in der Trainingsdetail-Ansicht einfügen: `training.team_name`, nur wenn vorhanden

## 3. KalenderPage — Props-Übergabe

- [x] 3.1 Beim Öffnen des `EventInfoModal` für ein Spiel: `game.teams` mit vorberechneten Kurznamen aus `shortNames`-Map übergeben
- [x] 3.2 Beim Öffnen des `EventInfoModal` für ein Training: Kurzname aus `shortNames.get(training.team_id)` als `team_name` in das Training-Objekt einfügen
- [x] 3.3 `end_date` ist im `Game`-Interface von `KalenderPage` bereits vorhanden und wird via `infoItem` automatisch weitergereicht — sicherstellen dass das `Game`-Interface in `EventInfoModal` die beiden neuen Felder trägt
