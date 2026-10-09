# Proposal: kachel-eckfahne

## Why

Die Kacheln haben 12 px Radius (`rounded-xl`) und oben 4 px Farbe (`border-t-4`). Durch den Radius läuft die Farbe an beiden Seiten in einem Bogen aus — die seitlichen Striche wirken unruhig, und bei fraktionaler Skalierung entsteht zusätzlich die gelbe Haarlinie (Gotcha in `docs/agent/06-gotchas.md`). Der Variantenvergleich `TeamWERK-Kachelvarianten.html` (Übersicht, Kalender, Termine, Dienste; Desktop + iPhone) hat als Richtung **Variante 4 „Eckfahne“** festgelegt.

Die neue Form soll vor dem Wechsel auf Prod mit echten Nutzern ausprobiert werden, ohne Prod zu berühren. Dafür braucht es eine zweite, vollständig getrennte Instanz (`beta.teamwerk.team-stuttgart.org`).

## What Changes

- **Kachel/Modal:** `rounded-xs` (2 px) statt `rounded-xl`, `border-t-2` statt `border-t-4`, dazu das Dreieck oben links als neue Utility `eckfahne` (`@utility` in `web/src/index.css`, `background-image`, 28 px). Durchgängig in allen 199 Kacheln/Modals in `pages/` und `components/`.
- **Statusfarben:** Termin-, Dienst- und Mitfahr-Karten tragen die Fahne in ihrer Statusfarbe (`fahne-*`, gesetzt über `card.border` in `lib/eventColors.ts`; abgesagt/vergangen grau).
- **Inhaltsbereich (Desktop):** `<main>` bekommt `sm:border-l-[3px]` ohne Rundung statt `sm:rounded-tl-3xl sm:rounded-bl-3xl sm:border-l-4`, dazu die größere Fahne (`eckfahne-lg`, 44 px). Mobile unverändert ohne Dekoration.
- Lade-Platzhalter in Kachelform und das Benutzerhandbuch (`public/benutzerhandbuch.html`, `.card`) ziehen nach.
- Neuer Gate-Test `web/src/lib/__tests__/eckfahne.gate.test.ts` (keine `border-t-4`, keine Oberkante ohne Fahne, keine Fahne mit großem Radius; Poison-Tests).
- **Beta-Instanz:** eigener systemd-Dienst `teamwerk-beta` (Port 8081), eigene Env `/etc/teamwerk-beta/env`, eigene DB und Storage unter `/var/lib/teamwerk-beta/`, eigener nginx-Server-Block, eigenes Zertifikat; Make-Targets `deploy-beta` und `setup-beta`. Kein Scheduler-Cron, kein Mailversand, keine Push-Schlüssel, kein BWHV-Abruf, keine Matomo-Erfassung — die Beta erzeugt keine Wirkung außerhalb ihrer selbst.

## Capabilities

- `component-standards` (MODIFIED): Card-, Modal- und Tabellen-Container-String.
- `mobile-navigation` (MODIFIED): Dekoration des Hauptbereichs.
- `beta-umgebung` (ADDED): getrennte Vorschau-Instanz.

## Impact

- Rein visuell im Frontend; keine API-, Schema- oder Rechteänderung.
- Infrastruktur: zusätzlicher Prozess auf dem VPS (~40 MB RAM), zusätzlicher nginx-Server-Block. Der Prod-Block, die Prod-Env, die Prod-DB und die Prod-Crontab werden nicht verändert.
- DNS: A-Record `beta.teamwerk.team-stuttgart.org → 31.70.110.19` muss in der Zone (agenturserver/Mittwald) angelegt werden — manueller Schritt.

## Test-Anforderungen

Keine neuen Routen. Invariante „jede Kachel trägt die Eckfahne“ → `eckfahne.gate.test.ts` (Bestand grün, drei Poison-Fälle rot). Bestehende Tests, die Karten über `div.rounded-xl` fanden, suchen jetzt `div.eckfahne`.
