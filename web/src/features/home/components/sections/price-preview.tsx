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

import { GroupMultiplierBadge } from '@/components/group-badge'
import { useStatus } from '@/hooks/use-status'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { parseHeaderNavModulesFromStatus } from '@/lib/nav-modules'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { TOKEN_UNIT_DIVISORS } from '@/features/pricing/constants'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { getDisplayGroupRatio } from '@/features/pricing/lib/model-helpers'
import { calculateTokenPrice } from '@/features/pricing/lib/price'

const PREVIEW_MODEL_COUNT = 4

export function PricePreview() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const isAuthed = useAuthStore((state) => !!state.auth.user)
  // Only fetch /api/pricing when visitors may see it: a 401 from a
  // login-only pricing page would bounce anonymous visitors to /sign-in.
  const pricing = parseHeaderNavModulesFromStatus(
    status as Record<string, unknown> | null
  )?.pricing
  const canSeePricing =
    !!pricing &&
    typeof pricing === 'object' &&
    pricing.enabled &&
    (!pricing.requireAuth || isAuthed)
  const { models, isLoading } = usePricingData(canSeePricing)
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

  if (!canSeePricing || isLoading || cheapest.length === 0) {
    return null
  }

  return (
    <section className='border-border/40 border-t px-5 py-14 sm:px-8 md:py-20'>
      <div className='mx-auto w-full max-w-container'>
        <div className='mb-8 flex flex-wrap items-end justify-between gap-4'>
          <div>
            <p className='text-muted-foreground mb-2 text-xs font-medium tracking-wide uppercase'>
              {t('Model Pricing')}
            </p>
            <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
              {t('Low-cost models')}
            </h2>
          </div>
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

        <div className='border-border/60 grid border-t sm:grid-cols-2 lg:grid-cols-4'>
          {cheapest.map((entry) => (
            <div
              key={entry.model.model_name}
              className='border-border/60 hover:bg-muted/30 flex min-w-0 flex-col gap-3 border-b p-5 transition-colors sm:[&:nth-child(2n)]:border-l lg:[&:not(:first-child)]:border-l'
            >
              <div className='min-w-0'>
                <p
                  className='truncate font-mono text-sm font-semibold'
                  title={entry.model.model_name}
                >
                  {entry.model.model_name}
                </p>
                {entry.model.vendor_name && (
                  <p className='text-muted-foreground mt-1 truncate text-xs'>
                    {entry.model.vendor_name}
                  </p>
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
                  <GroupMultiplierBadge ratio={entry.ratio} className='mt-2' />
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
