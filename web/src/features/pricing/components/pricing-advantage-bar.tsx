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
import { ArrowDownWideNarrow } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { GroupMultiplierBadge } from '@/components/group-badge'
import { Button } from '@/components/ui/button'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { TOKEN_UNIT_DIVISORS } from '../constants'
import { calculateTokenPrice } from '../lib/price'
import { getDisplayGroupRatio } from '../lib/model-helpers'
import type { PricingModel } from '../types'

interface PricingAdvantageBarProps {
  models: PricingModel[]
  groupRatios?: Record<string, number>
  selectedGroup?: string
  onSortByLowestPrice: () => void
}

function getLowestInputPricePerK(
  models: PricingModel[],
  selectedGroup?: string
): number | null {
  let lowest = Number.POSITIVE_INFINITY
  for (const model of models) {
    if (model.quota_type !== 0) continue
    const price = calculateTokenPrice(
      model,
      'input',
      getDisplayGroupRatio(model, selectedGroup)
    )
    if (!Number.isFinite(price) || price <= 0) continue
    lowest = Math.min(lowest, price)
  }
  return Number.isFinite(lowest)
    ? lowest / TOKEN_UNIT_DIVISORS.K
    : null
}

function getLowestGroupRatio(
  groupRatios: Record<string, number>
): number | null {
  let lowest = Number.POSITIVE_INFINITY
  for (const ratio of Object.values(groupRatios)) {
    if (!Number.isFinite(ratio) || ratio <= 0) continue
    lowest = Math.min(lowest, ratio)
  }
  return Number.isFinite(lowest) ? lowest : null
}

export function PricingAdvantageBar(props: PricingAdvantageBarProps) {
  const { t } = useTranslation()
  const currency = useSystemConfigStore((state) => state.config.currency)

  const lowestPrice = useMemo(
    () => getLowestInputPricePerK(props.models, props.selectedGroup),
    // Currency is read indirectly by the price formatter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [props.models, props.selectedGroup, currency]
  )
  const lowestGroupRatio = useMemo(
    () => getLowestGroupRatio(props.groupRatios ?? {}),
    [props.groupRatios]
  )

  if (lowestPrice === null) {
    return null
  }

  return (
    <div className='bg-card flex flex-wrap items-center gap-x-5 gap-y-2 rounded-xl border px-4 py-3'>
      <div className='flex items-baseline gap-1.5'>
        <span className='text-muted-foreground text-xs'>
          {t('Lowest input price')}
        </span>
        <span className='font-mono text-base font-semibold tabular-nums'>
          {formatBillingCurrencyFromUSD(lowestPrice, {
            digitsLarge: 4,
            digitsSmall: 6,
            abbreviate: false,
          })}
        </span>
        <span className='text-muted-foreground text-xs'>/ 1K tokens</span>
      </div>

      {lowestGroupRatio !== null && lowestGroupRatio < 1 && (
        <div className='flex items-center gap-1.5'>
          <span className='text-muted-foreground text-xs'>
            {t('Group price as low as')}
          </span>
          <GroupMultiplierBadge ratio={lowestGroupRatio} />
        </div>
      )}

      <Button
        type='button'
        variant='outline'
        size='sm'
        className='ml-auto h-8 gap-1.5 px-3 text-xs'
        onClick={props.onSortByLowestPrice}
      >
        <ArrowDownWideNarrow className='size-3.5' />
        {t('Sort by lowest price')}
      </Button>
    </div>
  )
}
