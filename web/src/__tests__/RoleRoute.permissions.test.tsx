/**
 * RoleRoute permission matrix — verifiziert, dass RoleRoute pro Persona
 * korrekt rendert (✅) oder auf "/" umleitet (➜).
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Frontend-RoleRoute-Sichtbarkeit"
 *
 * Spiegelbild: web/src/__tests__/RoleRoute.permissions.test.tsx
 * Backend-Äquivalent: internal/permissions/matrix_test.go
 */
import { describe, test, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { RoleRoute } from '../App'
import { renderAsPersona } from '../test/renderAsPersona'
import { PERSONAS } from '../test/personas'

const PageStub = ({ name }: { name: string }) => (
  <div data-testid={name}>{name}</div>
)

interface GatedRoute {
  label: string
  roles: string[]
  allowedIds: string[]
}

const GATED_ROUTES: GatedRoute[] = [
  {
    label: '/mitglieder (vorstand-Gate)',
    roles: ['admin', 'vorstand'],
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: '/anfragen (vorstand-Gate)',
    roles: ['admin', 'vorstand'],
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: '/nutzer (vorstand-Gate)',
    roles: ['admin', 'vorstand'],
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: '/einstellungen (vorstand-Gate)',
    roles: ['admin', 'vorstand'],
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: '/diensttypen (vorstand-Gate)',
    roles: ['admin', 'vorstand'],
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: '/dienstplan-vorlagen (vorstand-Gate)',
    roles: ['admin', 'vorstand'],
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: '/veranstaltungsorte (vorstand-Gate)',
    roles: ['admin', 'vorstand'],
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: '/kader (vorstand+trainer+sportliche_leitung-Gate)',
    roles: ['admin', 'vorstand', 'trainer', 'sportliche_leitung'],
    allowedIds: [
      'admin',
      'vorstand',
      'vorstand_elternteil',
      'trainer',
      'trainer_elternteil',
      'sportliche_leitung',
      'sportliche_leitung_elternteil',
    ],
  },
]

describe('RoleRoute — Permission-Matrix', () => {
  for (const route of GATED_ROUTES) {
    describe(route.label, () => {
      test.each(PERSONAS)('Persona $id', (persona) => {
        const testId = `page-${route.label.replace(/[^a-z0-9]/gi, '_')}`
        renderAsPersona(
          <RoleRoute roles={route.roles}>
            <PageStub name={testId} />
          </RoleRoute>,
          persona.id,
          { route: '/test' },
        )

        const isAllowed = route.allowedIds.includes(persona.id)
        if (isAllowed) {
          expect(screen.getByTestId(testId)).toBeInTheDocument()
        } else {
          expect(screen.queryByTestId(testId)).not.toBeInTheDocument()
        }
      })
    })
  }
})
