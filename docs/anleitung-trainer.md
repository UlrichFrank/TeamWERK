# TeamWERK — Anleitung für Trainer

Als Trainer hast du erweiterten Zugriff auf Verwaltungsfunktionen für dein Team. Diese Anleitung beschreibt alle Funktionen, die über die Möglichkeiten eines normalen Mitglieds hinausgehen.

Alle Funktionen aus der **[Spieler-Anleitung](anleitung-spieler.md)** stehen dir ebenfalls zur Verfügung.

---

## Überblick deiner Berechtigungen

| Funktion | Spieler | Trainer |
|---|:---:|:---:|
| Eigenes Profil, Dienstbörse, Kalender | ✓ | ✓ |
| Mitgliederliste lesen | ✓ | ✓ |
| Dienst-Slots anlegen und verwalten | — | ✓ |
| Dienste als erfüllt markieren / Geldersatz buchen | — | ✓ |
| Beitrittsanträge bearbeiten | — | ✓ |
| Einladungen versenden | — | ✓ |
| Spielplan verwalten | — | ✓ |
| Kader verwalten | — | ✓ |
| Änderungsanträge genehmigen | — | ✓ |

---

## Mitgliederverwaltung

### Mitgliederliste

Unter **„Mitglieder"** siehst du alle Vereinsmitglieder. Als Trainer hast du zusätzlich Zugriff auf die Kader-Verwaltung (siehe unten).

### Änderungsanträge genehmigen

Mitglieder können Änderungen an ihrem Profil beantragen (z. B. neue Adresse, korrigierte Kontaktdaten). Du siehst offene Anträge in der Mitgliederdetailansicht:

1. Mitglied in der Liste anklicken
2. Bereich „Änderungsanträge" einsehen
3. Antrag **annehmen** (Daten werden übernommen) oder **ablehnen**

---

## Beitrittsanträge

Interessenten können über die öffentliche Seite einen **Beitrittsantrag** stellen. Unter **„Beitrittsanträge"** (im Admin-Bereich) siehst du alle offenen Anträge.

### Antrag bearbeiten

1. Antrag in der Liste anklicken
2. Daten des Interessenten prüfen
3. **„Genehmigen"** → der Antrag wird als genehmigt markiert; der Vorstand legt das Mitglied anschließend an und versendet die Einladung
4. **„Ablehnen"** → mit optionalem Ablehnungsgrund

---

## Einladungen versenden

Neue Mitglieder erhalten Zugang über eine Einladungs-E-Mail.

1. Im Admin-Bereich **„Einladung versenden"** aufrufen
2. E-Mail-Adresse, Team und Rolle (`spieler` oder `trainer`) eintragen
3. Einladung absenden — der Empfänger erhält einen Link, über den er sein Passwort setzt

Offene (noch nicht angenommene) Einladungen kannst du in der Einladungsliste einsehen und bei Bedarf löschen.

---

## Dienst-Slots verwalten

### Slot manuell anlegen

1. Unter **„Dienste"** → **„Neuen Slot anlegen"**
2. Diensttyp auswählen (z. B. Hallendienst, Kassendienst)
3. Datum, Uhrzeit, Bezeichnung und Anzahl der Plätze eintragen
4. Optional: Spiel verknüpfen
5. Speichern — der Slot erscheint sofort in der Dienstbörse

### Slot bearbeiten oder löschen

Klicke in der Slot-Liste auf den jeweiligen Eintrag. Du kannst alle Felder nachträglich ändern oder den Slot löschen (löscht auch alle Anmeldungen).

### Dienst als erfüllt markieren

Nach einem geleisteten Dienst:

1. Slot öffnen → Belegungsliste
2. Neben dem Mitglied **„Als erfüllt markieren"** klicken
3. Das Dienst-Konto des Mitglieds wird automatisch aktualisiert

### Geldersatz buchen

Falls ein Mitglied einen Dienst mit einer Geldleistung ablöst:

1. Slot öffnen → Belegungsliste
2. **„Geldersatz"** klicken und den Betrag eintragen
3. Der Status wechselt auf „Geldersatz geleistet"

---

## Spielplan verwalten

Unter **„Kalender"** kannst du den Spielplan deines Teams pflegen.

### Spiel anlegen

1. **„Neues Spiel"** klicken
2. Gegner, Datum, Uhrzeit, Heim/Auswärts und Team eintragen
3. Optional: Dienst-Template auswählen — dann werden die passenden Dienst-Slots automatisch generiert

### Spiel bearbeiten oder löschen

Klicke auf ein Spiel in der Kalenderansicht. Du kannst alle Felder ändern oder das Spiel löschen.

### Dienst-Slots aus Template neu generieren

Falls sich Anstoßzeit oder andere Parameter geändert haben:

1. Spiel öffnen
2. **„Dienst-Slots neu generieren"** klicken
3. Bestehende Slots werden durch neue ersetzt (Achtung: bestehende Anmeldungen gehen verloren)

---

## Kader verwalten

Unter **„Kader"** (Admin-Bereich) kannst du den Kader für dein Team und die aktuelle Saison pflegen.

### Kader initialisieren

Am Saisonstart einmalig:

1. **„Kader anlegen"** für die aktuelle Saison
2. Mitglieder aus der Gesamtliste zuweisen (Vorschläge basieren auf Altersklasse und bisheriger Zugehörigkeit)
3. Alternativ: **„Aus Vorsaison kopieren"** übernimmt den Kader der letzten Saison

### Kader bearbeiten

- Mitglieder hinzufügen oder entfernen
- **Spiele pro Saison** eintragen (für statistische Auswertungen)
- Automatische Zuweisung nutzen: **„Auto-Assign"** schlägt Mitglieder basierend auf Altersklassenregeln vor

---

## Häufige Fragen

**Ich sehe den Admin-Bereich nicht.**
Prüfe, ob dein Account die Vereinsfunktion `trainer` zugewiesen bekommen hat. Falls nicht, wende dich an den Admin.

**Ein Mitglied hat sich für einen Dienst angemeldet, ist aber nicht erschienen.**
Markiere den Dienst nicht als erfüllt. Du kannst die Anmeldung nicht selbst löschen — wende dich an den Admin, falls der Slot bereinigt werden muss.

**Ich möchte einen Dienst-Slot für ein einzelnes Spiel anlegen, ohne das ganze Template zu nutzen.**
Lege den Slot manuell an (siehe „Dienst-Slots verwalten") und verknüpfe ihn optional mit dem Spiel.

**Das Dienst-Konto eines Mitglieds zeigt falsche Werte.**
Prüfe die Belegungsliste aller Slots und ob die Dienste korrekt als erfüllt markiert wurden. Bei Datenbankfehlern wende dich an den Admin.

**Ich möchte ein Mitglied vollständig anlegen oder löschen.**
Das ist eine Vorstand-Funktion. Als Trainer kannst du nur Einladungen versenden und Beitrittsanträge bearbeiten. Sprich den Vorstand an.
