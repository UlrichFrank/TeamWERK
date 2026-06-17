/**
 * MemberDatenschutzTab inline gate:
 *   canDeleteSepa = isAdmin || isVorstand || user.isParent
 *   isAdmin = user.role === 'admin'
 *   isVorstand = hasFunction(user, 'vorstand')
 *
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 *
 * Designloch (§10): Die Spec listet nur isVorstand ohne admin. Tatsächlich inkludiert
 * canDeleteSepa auch isAdmin — admin kann das Dokument löschen.
 */
import { describe, test, expect } from 'vitest'
import { screen } from '@testing-library/react'
import MemberDatenschutzTab from '../../components/admin/MemberDatenschutzTab'
import { renderAsPersonaNoRouter } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

const FORM_WITH_SEPA = {
  sepa_mandat: true,
  sepa_mandat_date: '2025-01-01',
  sepa_mandat_url: 'https://example.com/sepa.pdf',
  dsgvo_verarbeitung: false,
  dsgvo_weitergabe: false,
}

const noop = async () => {}

// canDeleteSepa = isAdmin || isVorstand || isParent
// isAdmin = role === 'admin', isVorstand = hasFunction('vorstand')
// Personas mit Lösch-Recht: admin + vorstand + vorstand_elternteil + trainer_elternteil + sportliche_leitung_elternteil + elternteil
const CAN_DELETE_SEPA_IDS = [
  'admin',
  'vorstand',
  'vorstand_elternteil',
  'trainer_elternteil',
  'sportliche_leitung_elternteil',
  'elternteil',
]

describe('MemberDatenschutzTab — canDeleteSepa-Gate: "Dokument löschen"-Button', () => {
  test.each(PERSONAS)('Persona $id', (persona) => {
    renderAsPersonaNoRouter(
      <MemberDatenschutzTab
        memberId={1}
        form={FORM_WITH_SEPA}
        isNew={false}
        drafts={[]}
        onFormChange={() => {}}
        onDraftAccept={noop}
        onDraftReject={noop}
        onSave={noop}
        saving={false}
        saved={false}
        error=""
      />,
      persona.id,
    )

    const deleteBtn = screen.queryByText('Dokument löschen')
    if (CAN_DELETE_SEPA_IDS.includes(persona.id)) {
      expect(
        deleteBtn,
        `Persona ${persona.id}: "Dokument löschen" muss sichtbar sein`,
      ).not.toBeNull()
    } else {
      expect(
        deleteBtn,
        `Persona ${persona.id}: "Dokument löschen" darf NICHT sichtbar sein`,
      ).toBeNull()
    }
  })
})
