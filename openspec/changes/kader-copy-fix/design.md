## Context

Der Kader-Kopier-Workflow in `internal/kader/copy.go` enthält zwei Bugs:

1. `ageClassBefore` bildet die Klassen-Progression falsch ab. Die Funktion gibt für „B-Jugend" → „A-Jugend" zurück, obwohl der ältere Jahrgang von B-Jugend *nach oben* zu A-Jugend wechselt — d.h. A-Jugend in der neuen Saison sollte den alten Jahrgang aus B-Jugend übernehmen. Die korrekte Richtung ist A←B, B←C, C←D.

2. `copyMembersFromKader` kopiert alle Mitglieder aus dem Quell-Kader ohne Jahrgangsfilter. Dadurch werden Spieler übernommen, die in der neuen Saison nicht mehr zur Altersklasse gehören (der ältere Jahrgang des oberen Kaders scheidet aus oder rückt weiter hoch).

Darüber hinaus vermischt das aktuelle Copy-Modal zwei konzeptuell verschiedene Aktionen: „aus Vorjahr übernehmen" (Kontinuität) und „Auto-Assign nach Jahrgang" (Neubelegung). Das erzeugt Verwirrung bei der Bedienung.

## Goals / Non-Goals

**Goals:**
- Korrekte Jahrgangs-Progression beim Kopieren: älterer Jahrgang rückt eine Klasse hoch, jüngerer bleibt
- Jahrgangsfilter beim Kopieren: nur Mitglieder übernehmen, deren Geburtsjahr im Bracket des Ziel-Kaders liegt
- Copy-Modal vereinfachen: eine Smart-Copy-Aktion statt drei wählbare Optionen
- Auto-Assign als eigenständige Aktion mit Kader-Auswahl anbieten

**Non-Goals:**
- Keine DB-Schema-Änderungen
- Keine neuen API-Endpunkte (bestehende Endpunkte werden angepasst)
- Kein Rückwärts-Kompatibilitätsmodus für den alten `age-before-previous` Parameter

## Decisions

### Smart Copy Algorithmus

**Entscheidung**: Für jeden Ziel-Kader werden Mitglieder aus zwei Quellen kopiert — dem gleichnamigen Kader der Vorsaison (jüngerer Jahrgang bleibt) und dem eine Klasse *tieferen* Kader der Vorsaison (älterer Jahrgang rückt hoch). Der Filter in beiden Fällen ist das Bracket des Ziel-Kaders in der neuen Saison.

```
Für A-Jugend 2026/27 (Bracket [2008, 2009]):
  Quelle 1: A-Jugend 2025/26 → kopiere Mitglieder mit Jg. 2008 oder 2009
            (Jg. 2007 wird gefiltert — scheidet aus)
  Quelle 2: B-Jugend 2025/26 → kopiere Mitglieder mit Jg. 2008 oder 2009
            (Jg. 2010 wird gefiltert — bleibt in B-Jugend)
```

Der Filter `birth_year BETWEEN bracket[0] AND bracket[1]` verwendet die bestehende `ComputeAgeBrackets`-Logik, die bereits korrekt funktioniert.

**Warum nicht zwei separate Optionen beibehalten?** Die zwei Quellen (gleiche Klasse + tiefere Klasse) sind immer beide sinnvoll und ergänzen sich. Eine Auswahl würde den Nutzer zwingen, die interne Logik zu verstehen — das ist unnötige Komplexität.

**Neue Funktion `ageClassBelow`** ersetzt `ageClassBefore`:
```
A-Jugend → B-Jugend
B-Jugend → C-Jugend
C-Jugend → D-Jugend
D-Jugend → "" (kein tieferer Kader)
```

### Auto-Assign als separate Aktion

**Entscheidung**: Auto-Assign verlässt das Copy-Modal und wird ein eigenständiger Button auf der Kader-Seite. Ein neues `AutoAssignModal` zeigt alle Kader der aktiven Saison mit Checkboxen und ruft für jeden ausgewählten Kader den bestehenden `autoAssignMembers`-Endpunkt auf.

**Warum nicht per-Kader-Button?** Globale Selektion ermöglicht, mehrere Kader auf einmal zu befüllen — typischer Anwendungsfall am Saisonbeginn ohne Vorjahres-Daten. Per-Kader-Buttons bleiben weiterhin über den bestehenden Kader-Bearbeitungsbereich möglich.

### API-Änderungen

Der bestehende `POST /api/admin/kader/copy-from-season` Endpunkt wird geändert: Das `assignments`-Array verliert die `member_source`-Werte `same-age-previous`, `age-before-previous` und `auto-assign`. Der einzige gültige Wert ist `smart-copy` (Standard wenn leer). `empty` bleibt als Opt-out.

Der bestehende `POST /api/admin/kader/auto-assign` (oder Erweiterung des InitializeKader-Endpunkts) wird für das neue Modal genutzt — tatsächlich wird `copyKader` mit `auto-assign` direkt via CopyFromSeason aufgerufen, aber das Frontend ruft es nicht mehr aus dem Copy-Modal auf.

**Einfachste Lösung**: `CopyFromSeason` Handler unterstützt weiterhin alle `member_source`-Werte intern, aber das Frontend sendet nur noch `smart-copy` oder `empty`. Kein Breaking Change am API-Level nötig.

## Risks / Trade-offs

- **[Risiko] Bestehende Kader-Daten**: Bereits kopierte Kader (mit falscher Logik) bleiben unverändert. → Kein automatisches Fix nötig, manuelle Korrektur über bestehenden Kader-Editor.
- **[Trade-off] Smart Copy ersetzt Einzeloptionen**: Nutzer, die bewusst nur eine Quelle wollen (z.B. nur gleiche Klasse ohne Aufsteiger), verlieren diese Option. → Akzeptabel: manuelle Nachbearbeitung über Kader-Editor möglich.
- **[Risiko] D-Jugend hat keine tiefere Klasse**: Der neue Jahrgang kann nicht automatisch befüllt werden. → Smart Copy kopiert nur den verbleibenden Jahrgang aus D-Jugend Vorsaison. Auto-Assign ist der empfohlene Weg für den neuen Jahrgang.
