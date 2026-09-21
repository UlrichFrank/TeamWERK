import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import StaffelPicker from '../StaffelPicker'

let mock: MockAdapter

const katalog = [
  { code: 'mB-RL-BW', name: 'männliche B-Jugend Regionalliga', org: 'BWHV' },
  { code: 'mC-BOL-SRM', name: 'männliche C-Jugend Bezirksoberliga', org: 'Stuttgart-Rems-Murr' },
]

beforeEach(() => { mock = new MockAdapter(api, { onNoMatch: 'passthrough' }) })
afterEach(() => { mock.restore(); vi.clearAllMocks() })

describe('StaffelPicker', () => {
  test('zeigt "nicht zugeordnet" ohne Staffel', () => {
    render(<StaffelPicker value="" onSave={async () => {}} />)
    expect(screen.getByText('nicht zugeordnet')).toBeInTheDocument()
  })

  test('zeigt den Code, wenn einer gesetzt ist', () => {
    render(<StaffelPicker value="mB-RL-BW" onSave={async () => {}} />)
    expect(screen.getByText('mB-RL-BW')).toBeInTheDocument()
  })

  // Der Katalog geht nach außen und wird deshalb erst beim Öffnen geholt —
  // sonst wären es bei neun Kadern neun überflüssige Abrufe.
  test('lädt den Katalog erst beim Öffnen', async () => {
    mock.onGet('/bwhv/staffel-katalog').reply(200, katalog)
    render(<StaffelPicker value="" onSave={async () => {}} />)
    expect(mock.history.get.length).toBe(0)

    fireEvent.click(screen.getByLabelText('Staffel zuordnen'))
    await waitFor(() => expect(mock.history.get.length).toBe(1))
  })

  test('bietet die Staffeln aus dem Katalog an', async () => {
    mock.onGet('/bwhv/staffel-katalog').reply(200, katalog)
    render(<StaffelPicker value="" onSave={async () => {}} />)
    fireEvent.click(screen.getByLabelText('Staffel zuordnen'))

    await waitFor(() => expect(screen.getByLabelText('Staffel aus Katalog wählen')).toBeInTheDocument())
    const select = screen.getByLabelText('Staffel aus Katalog wählen') as HTMLSelectElement
    expect(select.options).toHaveLength(3) // Platzhalter + zwei Staffeln
  })

  test('speichert die Auswahl aus dem Katalog', async () => {
    mock.onGet('/bwhv/staffel-katalog').reply(200, katalog)
    const onSave = vi.fn(async () => {})
    render(<StaffelPicker value="" onSave={onSave} />)
    fireEvent.click(screen.getByLabelText('Staffel zuordnen'))
    await waitFor(() => expect(screen.getByLabelText('Staffel aus Katalog wählen')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText('Staffel aus Katalog wählen'), { target: { value: 'mC-BOL-SRM' } })
    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(onSave).toHaveBeenCalledWith('mC-BOL-SRM'))
  })

  test('Freitext funktioniert auch ohne Katalog', async () => {
    mock.onGet('/bwhv/staffel-katalog').reply(502, { error: 'bwhv_unreachable' })
    const onSave = vi.fn(async () => {})
    render(<StaffelPicker value="" onSave={onSave} />)
    fireEvent.click(screen.getByLabelText('Staffel zuordnen'))

    await waitFor(() => expect(screen.getByText(/nicht erreichbar/)).toBeInTheDocument())
    fireEvent.change(screen.getByLabelText('Staffelcode eingeben'), { target: { value: 'gD-BOL-SRM' } })
    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(onSave).toHaveBeenCalledWith('gD-BOL-SRM'))
  })

  test('Entfernen speichert einen leeren Code', async () => {
    mock.onGet('/bwhv/staffel-katalog').reply(200, katalog)
    const onSave = vi.fn(async () => {})
    render(<StaffelPicker value="mB-RL-BW" onSave={onSave} />)
    fireEvent.click(screen.getByLabelText('Staffel zuordnen'))

    await waitFor(() => expect(screen.getByText('Entfernen')).toBeInTheDocument())
    fireEvent.click(screen.getByText('Entfernen'))
    await waitFor(() => expect(onSave).toHaveBeenCalledWith(''))
  })

  // Die Validierung liefert sprechende Meldungen (unpassendes Geschlecht,
  // falsche Altersklasse, Übungsgruppe) — die gehören ans Eingabefeld, nicht
  // in einen Toast.
  test('zeigt die Fehlermeldung des Servers am Feld', async () => {
    mock.onGet('/bwhv/staffel-katalog').reply(200, katalog)
    const onSave = vi.fn(async () => {
      throw { response: { data: { error: 'Staffelcode passt nicht zum Geschlecht des Kaders' } } }
    })
    render(<StaffelPicker value="" onSave={onSave} />)
    fireEvent.click(screen.getByLabelText('Staffel zuordnen'))
    await waitFor(() => expect(screen.getByLabelText('Staffelcode eingeben')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText('Staffelcode eingeben'), { target: { value: 'mB-RL-BW' } })
    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(screen.getByText(/passt nicht zum Geschlecht/)).toBeInTheDocument())
  })

  test('bleibt nach einem Fehler geöffnet', async () => {
    mock.onGet('/bwhv/staffel-katalog').reply(200, katalog)
    const onSave = vi.fn(async () => { throw { response: { data: { error: 'kaputt' } } } })
    render(<StaffelPicker value="" onSave={onSave} />)
    fireEvent.click(screen.getByLabelText('Staffel zuordnen'))
    await waitFor(() => expect(screen.getByLabelText('Staffelcode eingeben')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText('Staffelcode eingeben'), { target: { value: 'x' } })
    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(screen.getByText('kaputt')).toBeInTheDocument())
    expect(screen.getByLabelText('Staffelcode eingeben')).toBeInTheDocument()
  })
})
