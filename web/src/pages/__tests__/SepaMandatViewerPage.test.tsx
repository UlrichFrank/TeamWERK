import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { render } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'

// Vault-State pro Test über Module-Mock steuern.
const vaultState: { privateKey: CryptoKey | null } = { privateKey: null }
vi.mock('../../contexts/VaultContext', () => ({
  useVault: () => ({
    privateKey: vaultState.privateKey,
    isUnlocked: !!vaultState.privateKey,
    unlock: vi.fn(),
    lock: vi.fn(),
  }),
}))

// decryptFile entkoppeln — wir prüfen den Daten-Pfad, nicht die Krypto-Primitiven.
const decryptMock = vi.fn()
vi.mock('../../lib/bankCrypto', () => ({
  decryptFile: (...args: unknown[]) => decryptMock(...args),
}))

// PdfRenderer stubben (kein pdfjs in jsdom).
vi.mock('../../components/PdfRenderer', () => ({
  default: ({ blob }: { blob: Blob }) => <div data-testid="pdf-renderer-stub">PDF[{blob.size}]</div>,
}))

import SepaMandatViewerPage from '../SepaMandatViewerPage'

let mock: MockAdapter
const FAKE_KEY = {} as CryptoKey

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  decryptMock.mockReset()
  vaultState.privateKey = null
  URL.createObjectURL = vi.fn(() => 'blob:mock')
  URL.revokeObjectURL = vi.fn()
})

afterEach(() => {
  mock.restore()
})

function renderPage(memberId = '7') {
  return render(
    <MemoryRouter initialEntries={[`/mitglieder/${memberId}/sepa-mandat/anzeigen`]}>
      <Routes>
        <Route
          path="/mitglieder/:memberId/sepa-mandat/anzeigen"
          element={<SepaMandatViewerPage />}
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('SepaMandatViewerPage', () => {
  test('Vault gesperrt → Hinweis statt Decrypt-Versuch', () => {
    vaultState.privateKey = null
    renderPage()
    expect(screen.getByText(/Tresor gesperrt/i)).toBeInTheDocument()
    expect(mock.history.get.length).toBe(0)
    expect(decryptMock).not.toHaveBeenCalled()
  })

  test('Vault offen → Token holen, Datei laden, entschlüsseln, FileViewer rendern', async () => {
    vaultState.privateKey = FAKE_KEY
    mock.onGet('/members/7/sepa-mandat/download-token').reply(200, {
      token: 'tok-xyz',
      dek_enc: 'enc-dek',
    })
    mock.onGet('/members/7/sepa-mandat/download?token=tok-xyz').reply(
      200,
      new ArrayBuffer(8),
    )
    decryptMock.mockResolvedValue(new Uint8Array([1, 2, 3, 4]))

    renderPage()
    expect(await screen.findByTestId('pdf-renderer-stub')).toBeInTheDocument()
    expect(decryptMock).toHaveBeenCalledWith(expect.any(Uint8Array), 'enc-dek', FAKE_KEY)
  })

  test('Decrypt-Fehler → Fehler-UI', async () => {
    vaultState.privateKey = FAKE_KEY
    mock.onGet('/members/7/sepa-mandat/download-token').reply(200, { token: 't', dek_enc: 'e' })
    mock.onGet('/members/7/sepa-mandat/download?token=t').reply(200, new ArrayBuffer(4))
    decryptMock.mockRejectedValue(new Error('bad key'))

    renderPage()
    expect(await screen.findByText(/Entschlüsselung fehlgeschlagen/i)).toBeInTheDocument()
  })

  test('Backend 404 → „Kein Mandat hinterlegt"', async () => {
    vaultState.privateKey = FAKE_KEY
    mock.onGet('/members/7/sepa-mandat/download-token').reply(404)
    renderPage()
    expect(await screen.findByText(/Kein Mandat hinterlegt/i)).toBeInTheDocument()
  })
})
