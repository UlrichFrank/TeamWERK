import { describe, test, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import LinkifiedText from '../LinkifiedText'

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="path">{loc.pathname}</div>
}

function setViewport(mobile: boolean) {
  vi.stubGlobal('matchMedia', (q: string) => ({
    matches: mobile && q.includes('max-width'),
    media: q,
    addListener: () => {},
    removeListener: () => {},
  }))
}

function renderAt(text: string, onCardClick = vi.fn()) {
  render(
    <MemoryRouter initialEntries={['/termine/spiel/5']}>
      <Routes>
        <Route
          path="/termine/spiel/5"
          element={
            <div onClick={onCardClick}>
              <LinkifiedText text={text} />
            </div>
          }
        />
        <Route path="*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  )
  return onCardClick
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('LinkifiedText', () => {
  test('fremde URL öffnet in neuem Tab', () => {
    renderAt('Infos unter https://team-stuttgart.org/info')
    const link = screen.getByRole('link', { name: 'https://team-stuttgart.org/info' })
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  test('App-Link navigiert innerhalb der SPA (kein neuer Tab, kein Seitenwechsel)', () => {
    setViewport(false)
    const url = `${window.location.origin}/termine/spiel/9`
    const onCard = renderAt(`Rückspiel: ${url}`)
    const link = screen.getByRole('link', { name: url })
    expect(link).not.toHaveAttribute('target')
    fireEvent.click(link)
    expect(screen.getByTestId('path').textContent).toBe('/termine/spiel/9')
    expect(onCard).not.toHaveBeenCalled()
  })

  test('Dokument-Link mobil → In-App-Route (Viewer mit Zurück-Knopf)', () => {
    setViewport(true)
    const open = vi.spyOn(window, 'open').mockReturnValue(null)
    const url = `${window.location.origin}/dokumente/datei/12`
    renderAt(`Turnierplan: ${url}`)
    fireEvent.click(screen.getByRole('link', { name: url }))
    expect(open).not.toHaveBeenCalled()
    expect(screen.getByTestId('path').textContent).toBe('/dokumente/datei/12')
  })

  test('Dokument-Link am Desktop → neuer Tab, App bleibt im Ursprungstab', () => {
    setViewport(false)
    const open = vi.spyOn(window, 'open').mockReturnValue(null)
    const url = `${window.location.origin}/dokumente/datei/12`
    const onCard = renderAt(`Turnierplan: ${url}`)
    fireEvent.click(screen.getByRole('link', { name: url }))
    expect(open).toHaveBeenCalledWith('/dokumente/datei/12', '_blank', 'noopener')
    expect(screen.queryByTestId('path')).toBeNull()
    expect(onCard).not.toHaveBeenCalled()
  })

  test('Cmd-Klick überlässt dem Browser den neuen Tab', () => {
    setViewport(false)
    const url = `${window.location.origin}/termine/spiel/9`
    renderAt(url)
    fireEvent.click(screen.getByRole('link', { name: url }), { metaKey: true })
    expect(screen.queryByTestId('path')).toBeNull()
  })
})
