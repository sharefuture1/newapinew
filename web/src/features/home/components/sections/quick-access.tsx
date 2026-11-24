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
// Modified by WendingAI (2026-11): homepage shortcuts to wallet top-up and the
// WendingAI Shop (AGPLv3 §7(c) change marking).
import { Link } from '@tanstack/react-router'
import { ArrowUpRight, ShoppingBag, Wallet } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { WENDING_SHOP_URL } from '@/lib/wending-brand'

const cardClass =
  'group border-border/70 bg-card hover:border-primary/50 hover:shadow-primary/5 relative flex min-h-36 flex-col justify-between gap-6 overflow-hidden rounded-2xl border p-6 transition-all duration-200 hover:shadow-lg focus-visible:ring-ring focus-visible:ring-2 focus-visible:outline-none sm:p-7'

function CardBody(props: {
  icon: ReactNode
  title: string
  description: string
  action: string
}) {
  return (
    <>
      <div className='flex items-start justify-between gap-4'>
        <span className='bg-primary/10 text-primary inline-flex size-11 items-center justify-center rounded-xl'>
          {props.icon}
        </span>
        <ArrowUpRight className='text-muted-foreground group-hover:text-primary size-5 transition-all duration-200 group-hover:translate-x-0.5 group-hover:-translate-y-0.5' />
      </div>
      <div>
        <h3 className='text-lg font-semibold tracking-tight'>{props.title}</h3>
        <p className='text-muted-foreground mt-1.5 text-sm leading-relaxed'>
          {props.description}
        </p>
        <span className='text-primary mt-4 inline-block text-sm font-medium'>
          {props.action}
        </span>
      </div>
    </>
  )
}

export function QuickAccess() {
  const { t } = useTranslation()

  return (
    <section
      aria-labelledby='quick-access-title'
      className='px-5 py-12 sm:px-8 md:py-16'
    >
      <div className='max-w-container mx-auto w-full'>
        <h2
          id='quick-access-title'
          className='text-muted-foreground mb-5 text-xs font-medium tracking-wider uppercase'
        >
          {t('Quick access')}
        </h2>
        <div className='grid gap-4 md:grid-cols-2'>
          {/* /wallet is login-gated: anonymous visitors are sent to sign-in
              and returned here afterwards by the router guard. */}
          <Link to='/wallet' className={cardClass}>
            <CardBody
              icon={<Wallet className='size-5' />}
              title={t('Wallet Top-up')}
              description={t(
                'Top up your balance instantly and pay only for what you use.'
              )}
              action={t('Go to wallet')}
            />
          </Link>
          <a
            href={WENDING_SHOP_URL}
            target='_blank'
            rel='noopener noreferrer'
            className={cardClass}
          >
            <CardBody
              icon={<ShoppingBag className='size-5' />}
              title={t('稳定点AI Shop')}
              description={t('Plans, credits and AI services in one store.')}
              action={t('Visit shop')}
            />
          </a>
        </div>
      </div>
    </section>
  )
}
