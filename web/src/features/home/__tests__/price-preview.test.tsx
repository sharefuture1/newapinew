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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import { PricePreview } from '../components/sections/price-preview'

const getPricing = vi.fn()
let headerNav: Record<string, unknown> = {}

vi.mock('@/features/pricing/api', () => ({
  getPricing: () => getPricing(),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({
    status: { HeaderNavModules: JSON.stringify(headerNav) },
  }),
}))

function renderPreview() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <PricePreview />
    </QueryClientProvider>
  )
}

describe('PricePreview', () => {
  beforeEach(() => {
    getPricing.mockReset()
    getPricing.mockResolvedValue({ success: true, data: [], vendors: [] })
  })

  it('does not request login-only pricing for anonymous visitors', () => {
    headerNav = { pricing: { enabled: true, requireAuth: true } }
    useAuthStore.setState((state) => ({
      auth: { ...state.auth, user: null },
    }))
    renderPreview()
    expect(getPricing).not.toHaveBeenCalled()
  })

  it('requests public pricing for anonymous visitors', async () => {
    headerNav = { pricing: { enabled: true, requireAuth: false } }
    useAuthStore.setState((state) => ({
      auth: { ...state.auth, user: null },
    }))
    renderPreview()
    await vi.waitFor(() => expect(getPricing).toHaveBeenCalledTimes(1))
  })
})
