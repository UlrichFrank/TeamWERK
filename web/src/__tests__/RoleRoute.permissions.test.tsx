/**
 * RoleRoute permission matrix — verifiziert, dass RoleRoute pro Persona
 * korrekt rendert (✅) oder auf "/" umleitet (➜).
 * Quelle: openspec/specs/permissions/spec.md §"Frontend-RoleRoute-Sichtbarkeit"
 * Die roles-Arrays spiegeln web/src/App.tsx — bei Änderungen dort beide Stellen pflegen.
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

const VORSTAND = ['admin', 'vorstand', 'vorstand_elternteil']
const KASSIERER_LIKE = [...VORSTAND, 'kassierer']
const TRAINER_LIKE = ['admin', 'trainer', 'trainer_elternteil', 'sportliche_leitung', 'sportliche_leitung_elternteil']

const GATED_ROUTES: GatedRoute[] = [
  { label: '/mitglieder (vorstand+kassierer)', roles: ['admin', 'vorstand', 'kassierer'], allowedIds: KASSIERER_LIKE },
  { label: '/einstellungen (vorstand+kassierer)', roles: ['admin', 'vorstand', 'kassierer'], allowedIds: KASSIERER_LIKE },
  { label: '/beitragslauf (vorstand+kassierer)', roles: ['admin', 'vorstand', 'kassierer'], allowedIds: KASSIERER_LIKE },
  { label: '/tresor (vorstand+kassierer)', roles: ['admin', 'vorstand', 'kassierer'], allowedIds: KASSIERER_LIKE },
  { label: '/anfragen (vorstand-Gate)', roles: ['admin', 'vorstand'], allowedIds: VORSTAND },
  { label: '/nutzer (vorstand-Gate)', roles: ['admin', 'vorstand'], allowedIds: VORSTAND },
  { label: '/diensttypen (vorstand-Gate)', roles: ['admin', 'vorstand'], allowedIds: VORSTAND },
  { label: '/dienstplan-vorlagen (vorstand-Gate)', roles: ['admin', 'vorstand'], allowedIds: VORSTAND },
  { label: '/veranstaltungsorte (vorstand-Gate)', roles: ['admin', 'vorstand'], allowedIds: VORSTAND },
  {
    label: '/kader (vorstand+trainer+sportliche_leitung-Gate)',
    roles: ['admin', 'vorstand', 'trainer', 'sportliche_leitung'],
    allowedIds: [...VORSTAND, 'trainer', 'trainer_elternteil', 'sportliche_leitung', 'sportliche_leitung_elternteil'],
  },
  {
    label: '/uebungsgruppen (vorstand+trainer+sportliche_leitung-Gate)',
    roles: ['admin', 'vorstand', 'trainer', 'sportliche_leitung'],
    allowedIds: [...VORSTAND, 'trainer', 'trainer_elternteil', 'sportliche_leitung', 'sportliche_leitung_elternteil'],
  },
  { label: '/anwesenheit (trainer+sportliche_leitung-Gate)', roles: ['admin', 'trainer', 'sportliche_leitung'], allowedIds: TRAINER_LIKE },
  {
    label: '/trainingstagebuch (trainer+sportliche_leitung+vorstand)',
    roles: ['admin', 'trainer', 'sportliche_leitung', 'vorstand'],
    allowedIds: [...TRAINER_LIKE, 'vorstand', 'vorstand_elternteil'],
  },
  // Das eigene Tagebuch ist persönlich: kein Admin-Bypass, nur die Funktion spieler.
  { label: '/profil/trainingstagebuch (nur spieler)', roles: ['spieler'], allowedIds: ['spieler'] },
  // Freigeber-Tier: medien ist die einzige Persona außer vorstand/admin, die hier durchkommt.
  { label: '/spielberichte/pruefen (medien+vorstand)', roles: ['admin', 'medien', 'vorstand'], allowedIds: [...VORSTAND, 'medien'] },
  { label: '/wartung (nur admin)', roles: ['admin'], allowedIds: ['admin'] },
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
