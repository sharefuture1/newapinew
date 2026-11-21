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
// Modified by WendingAI (2026-11): branded hero with wallet top-up and shop
// entry points (AGPLv3 §7(c) change marking).
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  ArrowUpRight,
  ShieldCheck,
  ShoppingBag,
  Wallet,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { WENDING_BRAND_NAME, WENDING_SHOP_URL } from '@/lib/wending-brand'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { ConnectionPanel } from '../connection-panel'

interface HeroProps {
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const siteName =
    useSystemConfigStore((state) => state.config.systemName) ||
    WENDING_BRAND_NAME

  const highlights = [
    { icon: Zap, label: t('OpenAI-compatible') },
    { icon: ShieldCheck, label: t('Stable & fast') },
    { icon: Wallet, label: t('Pay as you go') },
  ]

  return (
    <section className='border-border/40 relative overflow-hidden border-b px-5 pt-24 pb-12 sm:px-8 md:pt-32 md:pb-16'>
      {/* Soft brand glow; purely decorative. */}
      <div
        aria-hidden='true'
        className='pointer-events-none absolute inset-x-0 -top-40 -z-10 flex justify-center'
      >
        <div className='bg-primary/15 dark:bg-primary/20 h-[28rem] w-[56rem] rounded-full blur-3xl' />
      </div>

      <div className='max-w-container mx-auto w-full'>
        <div className='grid items-end gap-10 lg:grid-cols-12 lg:gap-12'>
          <div className='min-w-0 lg:col-span-7'>
            <span className='border-border/70 bg-background/70 text-muted-foreground inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium backdrop-blur'>
              <span className='bg-primary size-1.5 rounded-full' />
              {t('Enterprise AI API Gateway')}
            </span>
            <h1 className='mt-5 text-[clamp(2.75rem,8.5vw,6.25rem)] leading-[0.92] font-black tracking-[-0.035em] break-words'>
              {siteName}
              <span className='bg-primary ml-2 inline-block size-[0.16em] rounded-full align-baseline' />
            </h1>
            <ul className='text-muted-foreground mt-6 flex flex-wrap gap-x-5 gap-y-2 text-sm'>
              {highlights.map(({ icon: Icon, label }) => (
                <li key={label} className='inline-flex items-center gap-1.5'>
                  <Icon className='text-primary size-4' aria-hidden='true' />
                  {label}
                </li>
              ))}
            </ul>
          </div>

          <div className='min-w-0 lg:col-span-5 lg:pb-2'>
            <h2 className='text-xl leading-snug font-semibold tracking-tight sm:text-2xl'>
              {t('Connect your apps to large language models.')}
            </h2>
            <p className='text-muted-foreground mt-3 text-sm leading-relaxed sm:text-[15px]'>
              {t('Pick a model, create a key, make your first call.')}
            </p>

            <div className='mt-7 grid grid-cols-2 gap-3 sm:flex sm:flex-wrap sm:items-center'>
              <Button
                className='group col-span-2 h-11 px-5 text-sm font-medium sm:col-span-1'
                render={
                  <Link to={props.isAuthenticated ? '/keys' : '/sign-up'} />
                }
              >
                {t('Create API Key')}
                <ArrowUpRight className='ml-0.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5 group-hover:-translate-y-0.5' />
              </Button>
              <Button
                variant='outline'
                className='h-11 px-4 text-sm font-medium'
                render={<Link to='/wallet' />}
              >
                <Wallet className='size-4' aria-hidden='true' />
                {t('Wallet Top-up')}
              </Button>
              <Button
                variant='outline'
                className='h-11 px-4 text-sm font-medium'
                render={
                  <a
                    href={WENDING_SHOP_URL}
                    target='_blank'
                    rel='noopener noreferrer'
                  />
                }
              >
                <ShoppingBag className='size-4' aria-hidden='true' />
                {t('稳定AI Shop')}
              </Button>
            </div>
            <Link
              to='/pricing'
              className='text-muted-foreground hover:text-foreground group mt-4 inline-flex items-center gap-1 text-sm font-medium transition-colors'
            >
              {t('View Pricing')}
              <ArrowRight className='size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
            </Link>
          </div>
        </div>

        <ConnectionPanel className='mt-10 md:mt-14' />
      </div>
    </section>
  )
}
