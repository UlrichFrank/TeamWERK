## 1. KalenderPage — Spiel-Tile konsolidieren

- [x] 1.1 Duty-Dot (`hidden @tile-sm:block w-1.5 h-1.5 rounded-full`) aus Zeile 1 des Spiel-Tiles entfernen
- [x] 1.2 `EventNoteIndicator` aus Zeile 3 des Spiel-Tiles entfernen
- [x] 1.3 AlertTriangle-Bedingung auf `note.trim() !== '' || (slot_count > 0 && filled_count < total_count)` erweitern; `title`-Attribut mit kombiniertem Text befüllen (Note + Slot-Anzahl, nur vorhandene Teile)

## 2. EventInfoModal — Slot-Info ergänzen

- [x] 2.1 `Game`-Interface in `EventInfoModal.tsx` um optionale Felder `slot_count?: number`, `filled_count?: number`, `total_count?: number` erweitern
- [x] 2.2 Unterhalb von `<EventNoteIndicator variant="inline" .../>` eine generierte Zeile `<p className="text-sm text-brand-text-muted mt-1">X offene Dienst-Slots</p>` einfügen, die nur erscheint wenn `(game.slot_count ?? 0) > 0 && (game.filled_count ?? 0) < (game.total_count ?? 0)`
