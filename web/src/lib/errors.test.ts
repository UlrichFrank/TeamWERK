import { describe, it, expect } from 'vitest'
import { AxiosError, AxiosHeaders } from 'axios'
import { errorMessage, errorCodeMessage, errorStatus, errorData } from './errors'

/** Baut einen axios-Fehler mit dem gegebenen Response-Body. */
function axiosErr(status: number, data: unknown): AxiosError {
  const config = { headers: new AxiosHeaders() }
  const err = new AxiosError('Request failed', 'ERR_BAD_REQUEST', config as never)
  err.response = {
    status,
    statusText: '',
    data,
    headers: {},
    config: config as never,
  }
  return err
}

describe('errorMessage', () => {
  it('übersetzt den generischen Code internal', () => {
    expect(errorMessage(axiosErr(500, { error: 'internal' })))
      .toBe('Interner Fehler, bitte später erneut versuchen')
  })

  it('übersetzt die übrigen generischen Codes', () => {
    const cases: Array<[string, string]> = [
      ['invalid_id', 'Ungültige ID'],
      ['invalid_body', 'Ungültige Anfrage'],
      ['not_found', 'Nicht gefunden'],
      ['forbidden', 'Keine Berechtigung'],
      ['conflict', 'Der Datensatz existiert bereits'],
      ['validation', 'Eingabe ungültig'],
    ]
    for (const [code, text] of cases) {
      expect(errorMessage(axiosErr(400, { error: code }))).toBe(text)
    }
  })

  it('reicht unbekannte Codes unverändert durch', () => {
    expect(errorMessage(axiosErr(400, { error: 'h4a_login_failed' }))).toBe('h4a_login_failed')
  })

  it('bevorzugt data.error vor data.message', () => {
    expect(errorMessage(axiosErr(409, { error: 'conflict', message: 'egal' })))
      .toBe('Der Datensatz existiert bereits')
  })

  it('nutzt data.message, wenn kein Code vorhanden ist', () => {
    expect(errorMessage(axiosErr(422, { message: 'Bis 2 Stunden vorher' })))
      .toBe('Bis 2 Stunden vorher')
  })

  it('reicht einen Plain-Text-Body durch (noch nicht migrierte Domänen)', () => {
    expect(errorMessage(axiosErr(400, 'invalid id\n'))).toBe('invalid id\n')
  })

  it('fällt auf den Fallback zurück, wenn keine Response vorliegt', () => {
    const err = new AxiosError('', 'ERR_NETWORK', { headers: new AxiosHeaders() } as never)
    expect(errorMessage(err, 'Netzfehler')).toBe('Netzfehler')
  })

  it('behandelt generische Error-Werte und Nicht-Fehler', () => {
    expect(errorMessage(new Error('kaputt'))).toBe('kaputt')
    expect(errorMessage('irgendwas', 'Fallback')).toBe('Fallback')
  })
})

describe('errorCodeMessage', () => {
  it('übersetzt bekannte Codes und liefert sonst undefined', () => {
    expect(errorCodeMessage('forbidden')).toBe('Keine Berechtigung')
    expect(errorCodeMessage('unbekannt')).toBeUndefined()
    expect(errorCodeMessage(undefined)).toBeUndefined()
  })
})

describe('errorStatus / errorData', () => {
  it('liest Status und Body eines axios-Fehlers', () => {
    const err = axiosErr(403, { error: 'forbidden' })
    expect(errorStatus(err)).toBe(403)
    expect(errorData<{ error?: string }>(err)?.error).toBe('forbidden')
  })

  it('liefert undefined für Nicht-axios-Fehler', () => {
    expect(errorStatus(new Error('x'))).toBeUndefined()
    expect(errorData(new Error('x'))).toBeUndefined()
  })
})
