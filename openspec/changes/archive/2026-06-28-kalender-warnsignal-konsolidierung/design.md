## Context

KalenderPage rendert Spiel-Tiles mit drei unabhängigen Signalquellen:
1. `AlertTriangle` (Zeile 1, nach letztem Fix) — `slot_count > 0 && filled_count < total_count`
2. `EventNoteIndicator` Icon (Zeile 3) — `note.trim() !== ''`
3. Duty-Dot (farbiger Kreis, Zeile 1) — Slot-Füllgrad in Nuancen (grün/gelb/rot)

Das Detail-Modal (`EventInfoModal`) kennt `slot_count`/`filled_count`/`total_count` nicht — die fehlten in der lokalen `Game`-Interface.

## Goals / Non-Goals

**Goals:**
- Ein einziges AlertTriangle pro Spiel-Tile, das beide Signalquellen abdeckt
- Duty-Dot aus Tiles entfernen
- Im EventInfoModal: offene Slots als Text unterhalb des Hinweistexts anzeigen
- Training-Tiles bleiben unberührt

**Non-Goals:**
- API-Änderungen oder neue Datenbankfelder
- Änderungen am `EventNoteIndicator`-Basiskomponent
- Änderungen am `SpieltagDetailModal` (separate Ansicht mit eigenem Slot-Management)

## Decisions

**1. Kombinierter Tooltip statt separater Icons**

Das konsolidierte AlertTriangle bekommt einen `title`-Attribut, der beide Gründe auflistet:
```
"Hinweis: <note>\n<n> offene Dienst-Slots"
```
Nur vorhandene Teile werden eingefügt. Das `aria-label` erhält denselben Text.

Alternativen verworfen:
- Neues `GameWarningIndicator`-Komponente: unnötige Abstraktion für zwei Dateien
- Getrenntes Icon für Note vs. Slots belassen: widerspricht dem „ein Signal"-Ziel

**2. Logik direkt in KalenderPage inline (kein Helper)**

Die Tile-Rendering-Logik ist bereits eng mit dem Game-Objekt verzahnt. Eine lokale Berechnung (`const warningTitle = [...]`) ist hier lesbarer als ein ausgelagerter Helper, der nur an einer Stelle genutzt würde.

**3. EventInfoModal: Text-only, kein Icon**

Die generierte Slot-Info erscheint als `<p className="text-sm text-brand-text-muted mt-1">` — kein AlertTriangle, keine weitere visuelle Komponente. Der Abstand (`mt-1`) schafft die gewünschte visuelle Trennung vom Hinweistext darüber.

**4. Game-Interface in EventInfoModal erweitern**

`slot_count`, `filled_count`, `total_count` werden der lokalen `Game`-Interface hinzugefügt (optional, `?`), da KalenderPage sie immer mitschickt (`setInfoItem({ type: 'game', game: { ...g } })`).

## Risks / Trade-offs

- [Duty-Dot Entfernung] Nuancierter Füllgrad (grün/gelb/rot) geht verloren → akzeptiert, AlertTriangle deckt den kritischen Fall (unvollständig) ab; der Füllgrad ist im SpieltagDetailModal einsehbar.
- [Optional-Felder] Falls EventInfoModal künftig aus einer anderen Quelle ohne Slot-Felder aufgerufen wird, erscheint kein generierter Text (graceful degradation, kein Fehler).
