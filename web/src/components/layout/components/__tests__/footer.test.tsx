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
import { render, screen } from '@testing-library/react'
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
    systemName: 'WendingAI',
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
})
