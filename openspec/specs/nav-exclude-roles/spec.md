# nav-exclude-roles Specification

## Purpose

Diese Spezifikation beschreibt die Capability `nav-exclude-roles`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: NavItem unterstützt excludeRoles
Das NavItem-Typ-Interface SHALL eine optionale Eigenschaft `excludeRoles?: string[]` besitzen.

#### Scenario: Item mit excludeRoles wird für ausgeschlossene Rolle nicht angezeigt
- **WHEN** ein NavItem `excludeRoles: ['admin']` hat und der eingeloggte User die Rolle `admin` hat
- **THEN** wird das Item in der Navigation nicht angezeigt

#### Scenario: Item mit excludeRoles wird für nicht ausgeschlossene Rolle angezeigt
- **WHEN** ein NavItem `excludeRoles: ['admin']` hat und der eingeloggte User die Rolle `trainer` hat
- **THEN** wird das Item in der Navigation angezeigt (sofern `roles` passt oder leer ist)

#### Scenario: excludeRoles undefined verhält sich wie leeres Array
- **WHEN** ein NavItem kein `excludeRoles` definiert
- **THEN** wird kein Ausschluss angewendet — Verhalten unverändert gegenüber heute
