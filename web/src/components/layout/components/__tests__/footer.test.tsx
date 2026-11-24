/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { Footer } from '../footer'

const config = vi.hoisted(() => ({
  footerHtml: '' as string,
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: {} }),
}))
vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({
    systemName: '稳定点AI',
    logo: '/logo.png',
    footerHtml: config.footerHtml,
    demoSiteEnabled: false,
  }),
}))

async function renderFooter() {
  const root = createRootRoute()
  const routes = ['/', '/about'].map((path) =>
    createRoute({
      getParentRoute: () => root,
      path,
      component: () => <Footer />,
    })
  )
  const router = createRouter({
    routeTree: root.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(<RouterProvider router={router} />)
}

describe('site footer', () => {
  for (const footerHtml of ['', 'Telegram @wendingai']) {
    it(`has no upstream GitHub credit (custom html: ${!!footerHtml})`, async () => {
      config.footerHtml = footerHtml
      await renderFooter()

      expect(document.querySelector('a[href*="github.com"]')).toBeNull()
      expect(screen.getByRole('link', { name: 'About' })).toHaveAttribute(
        'href',
        '/about'
      )
    })
  }

  it('sanitizes the admin footer HTML before rendering it', async () => {
    config.footerHtml =
      '<a href="https://t.me/wendingai">Telegram</a><img src="x" onerror="window.__footerXss=1"><script>window.__footerXss=2</script>'
    await renderFooter()

    await waitFor(() =>
      expect(screen.getByRole('link', { name: 'Telegram' })).toBeInTheDocument()
    )
    const footer = document.querySelector('.custom-footer')
    expect(footer?.querySelector('script')).toBeNull()
    expect(footer?.querySelector('img')?.getAttribute('onerror')).toBeNull()
  })

  it('uses the 稳定点AI tagline instead of the generic upstream one', async () => {
    config.footerHtml = ''
    await renderFooter()

    expect(screen.getByText('Enterprise AI API Gateway')).toBeInTheDocument()
    expect(screen.queryByText('Powerful API Management Platform')).toBeNull()
  })
})
