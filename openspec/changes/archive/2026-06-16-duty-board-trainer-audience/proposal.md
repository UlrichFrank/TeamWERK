## Why

Trainer und sportliche Leitung sehen auf der Dienstbörse aktuell **keine** Dienst-Slots der Teams, die sie trainieren — sondern nur Slots der Teams, in denen sie selbst Spieler oder Eltern eines Spielers sind. Wer ein Spiel anlegt, sieht zwar die generierten Slots im Wizard, danach aber nie wieder. Das macht das Verwalten und Nachhalten von Helfern auf eigenen Veranstaltungen praktisch unmöglich.

Zusätzlich gilt heute für die Funktionen `vorstand`, `vorstand_beisitzer` und `trainer` ein pauschaler **Audience-Bypass**: sie sehen alle Audiences (`eltern`, `spieler`, `vorstand`, …) ohne Filtermöglichkeit. Im Alltag wollen sie aber primär nur „Dienste, die mich angehen" sehen und nur bei Bedarf gezielt fremde Audiences einsehen.

## What Changes

- **Backend `GET /api/duty-board`** erweitert die Team-Quelle der Sichtbarkeit um `trainer_memberships` (View über `kader_trainers`). Trainer/sportliche Leitung sehen damit automatisch alle Slots der Teams, die sie trainieren — analog zu Spielern/Eltern.
- **Backend** ersetzt den bisherigen Audience-**Bypass** für die Funktionen `vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung` durch einen Audience-**Self-Filter**, der standardmäßig nur Slots zeigt, deren `audiences`-Array eine der Funktionen des Nutzers enthält (oder null = öffentlich).
- **Backend** akzeptiert einen neuen Query-Parameter `?audience_all=1` (alternativ benannt analog zu `?view=mine`), der den Self-Filter deaktiviert und die alte Bypass-Sicht (alle Audiences) wiederherstellt.
- **Frontend `DutyPage`** erhält eine neue Filter-Pille „Nur meine Audience" (Icon `Filter`, default aktiv, persistiert in URL via `?audience=mine`/`?audience=all`). Die Pille ist **nur** für Nutzer sichtbar, die mindestens eine der Funktionen `vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung` haben.
- **Tests**: Neue Test-Szenarien für Trainer-Team-Sichtbarkeit, Audience-Self-Filter und Audience-All-Override.

Nicht-Ziele: kein neuer Toggle für andere Nutzer, keine Änderung am `?view=mine`-Verhalten, keine Änderung der Slot-Erstellung oder -Claim-Logik.

## Capabilities

### New Capabilities

(keine)

### Modified Capabilities

- `duties`: Sichtbarkeitsregel der Dienstbörse wird erweitert (Trainer-Team-Quelle ergänzt) und der Audience-Bypass für privilegierte Funktionen durch einen umschaltbaren Audience-Self-Filter ersetzt.

## Impact

- **Code**: `internal/duties/handler.go` `Board()` — Erweiterung der `whereParts`-Konstruktion um `trainer_memberships`; Umbau des `audienceBypass`-Blocks zu drei Modi (default-self / all / non-privileged).
- **Code**: `web/src/pages/DutyPage.tsx` — neue Pille mit URL-Param-Persistenz; Sichtbarkeit nur für die vier Funktionen.
- **Datenbank**: keine Migration nötig — `trainer_memberships`-View existiert seit Migration 039.
- **API**: kein Breaking Change. Neuer optionaler Query-Param. Default-Verhalten für nicht-privilegierte Rollen bleibt identisch.
- **Sicherheit**: Trainer dürfen Slots fremder Teams nicht sehen — die Erweiterung beschränkt sich strikt auf Teams, in denen der Nutzer als Trainer in einem Kader eingetragen ist.
