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
import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { GroupMultiplierBadge } from '@/components/group-badge'
import { Badge } from '@/components/ui/badge'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { TOKEN_UNIT_DIVISORS } from '@/features/pricing/constants'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { getDisplayGroupRatio } from '@/features/pricing/lib/model-helpers'
import { calculateTokenPrice } from '@/features/pricing/lib/price'

const PREVIEW_MODEL_COUNT = 4

export function PricePreview() {
  const { t } = useTranslation()
  const { models, isLoading } = usePricingData()
  const currency = useSystemConfigStore((state) => state.config.currency)

  const cheapest = useMemo(
    () =>
      (models || [])
        .filter((model) => model.quota_type === 0)
        .map((model) => {
          const ratio = getDisplayGroupRatio(model)
          const priceUSD = calculateTokenPrice(model, 'input', ratio)
          return { model, ratio, priceUSD }
        })
        .filter((entry) => Number.isFinite(entry.priceUSD) && entry.priceUSD > 0)
        .sort((a, b) => a.priceUSD - b.priceUSD)
        .slice(0, PREVIEW_MODEL_COUNT),
    // Currency is read indirectly by the price formatter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [models, currency]
  )

  if (isLoading || cheapest.length === 0) {
    return null
  }

  return (
    <section className='border-border/40 relative z-10 border-t px-6 py-20 md:py-24'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-10 text-center'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Model Pricing')}
          </p>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('Low-cost models')}
          </h2>
        </AnimateInView>

        <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
          {cheapest.map((entry, index) => (
            <AnimateInView
              key={entry.model.model_name}
              delay={index * 80}
              animation='fade-up'
            >
              <div className='bg-card hover:ring-foreground/20 flex h-full flex-col gap-3 rounded-xl border p-4 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-sm'>
                <div className='min-w-0'>
                  <p
                    className='truncate font-mono text-sm font-semibold'
                    title={entry.model.model_name}
                  >
                    {entry.model.model_name}
                  </p>
                  {entry.model.vendor_name && (
                    <Badge
                      variant='secondary'
                      className='mt-1 max-w-full font-normal'
                    >
                      <span className='truncate'>
                        {entry.model.vendor_name}
                      </span>
                    </Badge>
                  )}
                </div>
                <div className='mt-auto'>
                  <p className='text-muted-foreground text-[11px] font-medium tracking-wide uppercase'>
                    {t('Input')}
                  </p>
                  <p className='flex flex-wrap items-baseline gap-x-1 font-mono text-lg font-semibold tabular-nums'>
                    <span>
                      {formatBillingCurrencyFromUSD(
                        entry.priceUSD / TOKEN_UNIT_DIVISORS.K,
                        {
                          digitsLarge: 4,
                          digitsSmall: 6,
                          abbreviate: false,
                        }
                      )}
                    </span>
                    <span className='text-muted-foreground text-[11px] font-normal'>
                      / 1K tokens
                    </span>
                  </p>
                  {entry.ratio < 1 && (
                    <GroupMultiplierBadge
                      ratio={entry.ratio}
                      className='mt-2'
                    />
                  )}
                </div>
              </div>
            </AnimateInView>
          ))}
        </div>

        <div className='mt-8 flex justify-center'>
          <Link
            to='/pricing'
            className='text-primary group inline-flex items-center gap-1.5 text-sm font-medium'
          >
            {t('View all {{count}} models', { count: models.length })}
            <ArrowRight
              aria-hidden
              className='size-3.5 transition-transform duration-200 group-hover:translate-x-0.5'
            />
          </Link>
        </div>
      </div>
    </section>
  )
}
