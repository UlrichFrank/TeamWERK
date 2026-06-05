# Dienst-Zielgruppe

## Warum

Dienste auf der Dienstbörse sollen nur für die Personengruppen sichtbar sein, die sie tatsächlich betreffen. Ein Kassierer-Dienst ist irrelevant für Eltern; ein Fahrdienst irrelevant für Spieler ohne Fahrpflicht. Aktuell sehen alle Nutzer alle Dienste eines Teams — ohne Möglichkeit, zu filtern.

## Was ändert sich

Jeder Dienst bekommt eine optionale **Zielgruppe** (audience). Ist keine gesetzt, gibt es keine Einschränkung. Ist eine gesetzt, sehen nur Nutzer mit der passenden Vereinsfunktion (bzw. Eltern-Link) den Dienst.

Die Zielgruppe wird am **Diensttyp** definiert und kaskadiert automatisch über **Vorlage → Slot**. An jedem Level kann sie überschrieben oder auf „keine Einschränkung" gesetzt werden.

Privilegierte Nutzer (Admin, Vorstand, Vorstands-Beisitzer, Trainer) sehen alle Dienste ihres Teams unabhängig von der Zielgruppe.

## Zielgruppen

| Wert | Bedeutung |
|------|-----------|
| NULL | Keine Einschränkung (Standard) |
| `spieler` | Nutzer mit Vereinsfunktion Spieler |
| `trainer` | Nutzer mit Vereinsfunktion Trainer |
| `vorstand` | Nutzer mit Vereinsfunktion Vorstand |
| `vorstand_beisitzer` | Nutzer mit Vereinsfunktion Vorstands-Beisitzer |
| `eltern` | Nutzer mit mindestens einem family_links-Eintrag |

## Betroffene Seiten

- `/admin/diensttypen` — Diensttyp anlegen/bearbeiten
- `/admin/dienstplan-vorlagen/:id` — Vorlage bearbeiten (Template-Items)
- `/kalender/:id` — Dienst anlegen/bearbeiten am Spieltag
- `/dienste` und `/kalender/:id` — Dienstbörse-Ansicht (Zielgruppe als Badge)

## Constraints

- Bestehende Datensätze bleiben unverändert (NULL = keine Einschränkung)
- Kein RAM-overhead: audience wird als TEXT-Spalte gespeichert, kein Join-Overhead
- Kompatibel mit dem bestehenden Rollen-Modell — Vereinsfunktionen kommen aus `member_club_functions`
