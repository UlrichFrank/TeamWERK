import { describe, test, expect, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import TransitionalHostnameBanner from './TransitionalHostnameBanner'

// window.location ist in jsdom read-only; wir ersetzen es pro Test über
// Object.defineProperty und stellen es danach wieder her.
const realLocation = window.location

function setLocation(host: string, pathname = '/', search = '') {
  Object.defineProperty(window, 'location', {
    value: { host, pathname, search },
    writable: true,
    configurable: true,
  })
}

afterEach(() => {
  cleanup()
  Object.defineProperty(window, 'location', {
    value: realLocation,
    writable: true,
    configurable: true,
  })
})

describe('TransitionalHostnameBanner', () => {
  test('rendert Banner mit CTA auf den Primärhost, wenn Origin der Alias ist', () => {
    setLocation('internal.team-stuttgart.org', '/dashboard', '?tab=x')
    render(<TransitionalHostnameBanner />)

    // Hinweistext sichtbar
    expect(screen.getByText(/Wir sind umgezogen/i)).toBeTruthy()

    // CTA-Link zeigt auf teamwerk.* und bewahrt pathname + search
    const cta = screen.getByRole('link') as HTMLAnchorElement
    expect(cta.getAttribute('href')).toBe(
      'https://teamwerk.team-stuttgart.org/dashboard?tab=x',
    )
  })

  test('rendert null auf dem Primärhost', () => {
    setLocation('teamwerk.team-stuttgart.org', '/dashboard')
    const { container } = render(<TransitionalHostnameBanner />)
    expect(container.firstChild).toBeNull()
  })

  test('rendert null in der lokalen Entwicklung (localhost)', () => {
    setLocation('localhost', '/')
    const { container } = render(<TransitionalHostnameBanner />)
    expect(container.firstChild).toBeNull()
  })
})
