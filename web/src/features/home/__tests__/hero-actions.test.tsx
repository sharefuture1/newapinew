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

import { useSystemConfigStore } from '@/stores/system-config-store'

import { Hero } from '../components/sections/hero'

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({
    status: { server_address: 'https://gateway.example.com/' },
  }),
}))

async function renderHero(isAuthenticated: boolean) {
  const root = createRootRoute()
  const routes = [
    createRoute({
      getParentRoute: () => root,
      path: '/',
      component: () => <Hero isAuthenticated={isAuthenticated} />,
    }),
    ...[
      '/sign-up',
      '/sign-in',
      '/pricing',
      '/keys',
      '/dashboard',
      '/wallet',
    ].map((path) =>
      createRoute({
        getParentRoute: () => root,
        path,
        component: () => null,
      })
    ),
  ]
  const router = createRouter({
    routeTree: root.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(<RouterProvider router={router} />)
}

describe('landing hero actions', () => {
  it('shows the site name as the wordmark', async () => {
    useSystemConfigStore.setState((state) => ({
      config: { ...state.config, systemName: 'WendingAI' },
    }))
    await renderHero(false)

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'WendingAI'
    )
    useSystemConfigStore.setState((state) => ({
      config: { ...state.config, systemName: 'WendingAI' },
    }))
  })

  it('offers sign-up and pricing to signed-out visitors', async () => {
    await renderHero(false)

    expect(
      screen.getByRole('button', { name: 'Create API Key' })
    ).toHaveAttribute('href', '/sign-up')
    expect(screen.getByRole('link', { name: 'View Pricing' })).toHaveAttribute(
      'href',
      '/pricing'
    )
  })

  it('sends signed-in users to key creation', async () => {
    await renderHero(true)

    expect(
      screen.getByRole('button', { name: 'Create API Key' })
    ).toHaveAttribute('href', '/keys')
    expect(screen.getByRole('link', { name: 'View Pricing' })).toHaveAttribute(
      'href',
      '/pricing'
    )
  })

  it('links to wallet top-up and the WendingAI Shop', async () => {
    await renderHero(false)

    expect(
      screen.getByRole('button', { name: 'Wallet Top-up' })
    ).toHaveAttribute('href', '/wallet')
    const shop = screen.getByRole('button', { name: 'WendingAI Shop' })
    expect(shop).toHaveAttribute('href', 'https://shop.wending.ai/')
    expect(shop).toHaveAttribute('target', '_blank')
  })

  it('does not link to third-party desktop clients', async () => {
    await renderHero(false)

    expect(screen.queryByText(/Cherry Studio/i)).toBeNull()
    expect(screen.queryByText(/CC Switch/i)).toBeNull()
    expect(document.querySelector('a[href*="cherry-ai.com"]')).toBeNull()
    expect(document.querySelector('a[href*="ccswitch.io"]')).toBeNull()
  })

  it('advertises the configured base URL and primary endpoint', async () => {
    await renderHero(false)

    expect(screen.getByText('https://gateway.example.com/v1')).toBeVisible()
    expect(screen.getByText('/chat/completions')).toBeVisible()
    expect(screen.getByText('POST')).toBeVisible()
  })
})
