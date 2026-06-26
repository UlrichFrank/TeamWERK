/**
 * MemberKontaktTab (Bankdaten) inline gate für SEPA-Mandat:
 *   canDeleteSepa = isAdmin || isVorstand || user.isParent
 *   isAdmin = user.role === 'admin'
 *   isVorstand = hasFunction(user, 'vorstand')
 *
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 *
 * Designloch (§10): Die Spec listet nur isVorstand ohne admin. Tatsächlich inkludiert
 * canDeleteSepa auch isAdmin — admin kann das Dokument löschen.
 *
 * Hinweis: Die SEPA-Mandat-Sektion wurde von "Datenschutz" zu "Bankdaten" verschoben.
 */
import { describe, test, expect } from 'vitest'
import { screen } from '@testing-library/react'
import MemberKontaktTab from '../../components/admin/MemberKontaktTab'
import { renderAsPersona } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

const FORM_WITH_SEPA = {
  iban: '',
  account_holder: '',
  beitragsfrei: false,
  sepa_mandat: true,
  sepa_mandat_date: '2025-01-01',
  sepa_mandat_url: 'https://example.com/sepa.pdf',
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

describe('MemberKontaktTab — canDeleteSepa-Gate: "Dokument löschen"-Button', () => {
  test.each(PERSONAS)('Persona $id', (persona) => {
    renderAsPersona(
      <MemberKontaktTab
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
