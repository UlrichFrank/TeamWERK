# icon-system Specification

## Purpose

Diese Spezifikation beschreibt die Capability `icon-system`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Lucide React als einziges Icon-System
Alle visuellen Symbole in `web/src/` SHALL als Lucide-React-Komponenten gerendert werden. Unicode-Zeichen (☰ ✕ ⋮ ▸ ▾ ✓ ✗ ⚠ « »), Emojis (🗑 📋 ⚽ 🚌 🏠) und der custom Inline-SVG in `BrandCheckbox` sind zu ersetzen.

Das semantische Mapping MUSS folgende Tabelle einhalten:

| Alt | Lucide-Komponente |
|---|---|
| `☰` | `Menu` |
| `✕` / `✗` | `X` |
| `⋮` | `MoreVertical` |
| `▸` | `ChevronRight` |
| `▾` | `ChevronDown` |
| `✓` | `Check` |
| `⚠` | `AlertTriangle` |
| `🗑` | `Trash2` |
| `«` | `ChevronsLeft` |
| `»` | `ChevronsRight` |
| `📋` | `Calendar` |
| `⚽` / `🏠` (Heimspiel) | `Home` |
| `🚌` / `✈` (Auswärtsspiel) | `MapPin` |

#### Scenario: Kein Unicode-Zeichen als Icon
- **WHEN** `web/src/` auf ☰, ✕, ⋮, ▸, ▾, ✓, ✗, ⚠, «, » geprüft wird
- **THEN** gibt es keine Treffer

#### Scenario: Kein Emoji als Icon
- **WHEN** `web/src/` auf 🗑, 📋, ⚽, 🚌, 🏠, ✈ geprüft wird
- **THEN** gibt es keine Treffer (außer ggf. in Text-Strings, nicht in Icon-Funktionen)

---

### Requirement: Einheitliche Icon-Größen
Lucide Icons SHALL exakt eine der drei definierten Größenklassen verwenden:

- `w-4 h-4` — Inline in Text, Tabellen-Aktionen (xs-Button)
- `w-5 h-5` — Standard in Buttons, Nav-Chevrons, Accordion-Header
- `w-6 h-6` — Standalone-Icons, leere Zustände

#### Scenario: Icon in einem Primär-Button hat w-5 h-5
- **WHEN** ein Primär-Button mit Icon gerendert wird
- **THEN** hat das Icon die Klassen `w-5 h-5`

#### Scenario: Icon in einer Tabellen-Aktion hat w-4 h-4
- **WHEN** ein Aktions-Icon in einer Tabellenzeile gerendert wird (Trash2, Edit, etc.)
- **THEN** hat das Icon die Klassen `w-4 h-4`

---

### Requirement: Accessibility für Icon-only-Buttons
Jeder Button, der ausschließlich ein Icon enthält (kein sichtbarer Text), MUSS ein `aria-label`-Attribut tragen.

#### Scenario: Löschen-Button hat aria-label
- **WHEN** ein Button nur `<Trash2 />` enthält
- **THEN** hat der `<button>` das Attribut `aria-label="Eintrag löschen"` oder ähnlich sinnvoll

#### Scenario: Schließen-Button in Modal hat aria-label
- **WHEN** ein Modal-Schließen-Button nur `<X />` enthält
- **THEN** hat der `<button>` das Attribut `aria-label="Schließen"`
