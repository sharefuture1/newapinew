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

import { Hero } from '../components/sections/hero'

vi.mock('@lobehub/icons', () => ({
  CherryStudio: { Color: () => null },
}))
vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: undefined }),
}))
vi.mock('../components/hero-terminal-demo', () => ({
  HeroTerminalDemo: () => null,
}))

async function renderHero(isAuthenticated: boolean) {
  const root = createRootRoute()
  const routes = [
    createRoute({
      getParentRoute: () => root,
      path: '/',
      component: () => <Hero isAuthenticated={isAuthenticated} />,
    }),
    ...['/sign-up', '/sign-in', '/pricing', '/keys', '/dashboard'].map((path) =>
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
  it('offers sign-up and pricing to signed-out visitors', async () => {
    await renderHero(false)

    expect(screen.getByRole('button', { name: 'Get Started' })).toHaveAttribute(
      'href',
      '/sign-up'
    )
    expect(
      screen.getByRole('button', { name: 'View Pricing' })
    ).toHaveAttribute('href', '/pricing')
  })

  it('offers console, key creation and pricing to signed-in users', async () => {
    await renderHero(true)

    expect(
      screen.getByRole('button', { name: 'Go to Dashboard' })
    ).toHaveAttribute('href', '/dashboard')
    expect(
      screen.getByRole('button', { name: 'Create API Key' })
    ).toHaveAttribute('href', '/keys')
    expect(
      screen.getByRole('button', { name: 'Model Pricing' })
    ).toHaveAttribute('href', '/pricing')
  })
})
