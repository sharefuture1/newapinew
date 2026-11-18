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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { PricingAdvantageBar } from '../components/pricing-advantage-bar'
import type { PricingModel } from '../types'

function pricingModel(overrides: Partial<PricingModel> = {}): PricingModel {
  return {
    id: 1,
    model_name: 'example-model',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 3,
    enable_groups: ['default'],
    group_ratio: { default: 1 },
    ...overrides,
  }
}

describe('pricing advantage bar', () => {
  it('shows the cheapest per-1K input price of the current model list', () => {
    render(
      <PricingAdvantageBar
        models={[
          pricingModel({ id: 1, model_name: 'standard', model_ratio: 1 }),
          pricingModel({ id: 2, model_name: 'cheap', model_ratio: 0.5 }),
        ]}
        groupRatios={{ default: 1 }}
        onSortByLowestPrice={vi.fn()}
      />
    )

    expect(screen.getByText('$0.001')).toBeVisible()
    expect(screen.getByText('/ 1K tokens')).toBeVisible()
  })

  it('advertises a group discount only when the group ratio is below one', () => {
    const { rerender } = render(
      <PricingAdvantageBar
        models={[pricingModel()]}
        groupRatios={{ default: 0.8 }}
        onSortByLowestPrice={vi.fn()}
      />
    )
    expect(screen.getByText('0.8x')).toBeVisible()

    rerender(
      <PricingAdvantageBar
        models={[pricingModel()]}
        groupRatios={{ default: 1.2 }}
        onSortByLowestPrice={vi.fn()}
      />
    )
    expect(screen.queryByText('1.2x')).toBeNull()
  })

  it('requests a lowest-price sort from the visible button', async () => {
    const onSortByLowestPrice = vi.fn()
    render(
      <PricingAdvantageBar
        models={[pricingModel()]}
        onSortByLowestPrice={onSortByLowestPrice}
      />
    )

    await userEvent.click(
      screen.getByRole('button', { name: 'Sort by lowest price' })
    )
    expect(onSortByLowestPrice).toHaveBeenCalledTimes(1)
  })

  it('renders nothing when no model has a token price', () => {
    const { container } = render(
      <PricingAdvantageBar
        models={[pricingModel({ quota_type: 1, model_ratio: 1 })]}
        onSortByLowestPrice={vi.fn()}
      />
    )
    expect(container).toBeEmptyDOMElement()
  })
})
