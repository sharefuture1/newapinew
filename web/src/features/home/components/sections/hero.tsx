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
import { ArrowRight, ArrowUpRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { ConnectionPanel } from '../connection-panel'

interface HeroProps {
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const siteName = useSystemConfigStore((state) => state.config.systemName)

  return (
    <section className='border-border/40 border-b px-5 pt-24 pb-10 sm:px-8 md:pt-28 md:pb-14'>
      <div className='mx-auto w-full max-w-container'>
        <div className='grid items-end gap-8 lg:grid-cols-12 lg:gap-12'>
          <div className='lg:col-span-7'>
            <h1 className='text-[clamp(3rem,11vw,7.5rem)] leading-[0.85] font-black tracking-[-0.04em] break-words'>
              {siteName}
              <span className='bg-primary ml-3 inline-block size-[0.18em] rounded-full align-baseline' />
            </h1>
          </div>

          <div className='lg:col-span-5 lg:pb-3'>
            <h2 className='text-xl leading-snug font-semibold tracking-tight sm:text-2xl'>
              {t('Connect your apps to large language models.')}
            </h2>
            <p className='text-muted-foreground mt-3 text-sm leading-relaxed sm:text-[15px]'>
              {t('Pick a model, create a key, make your first call.')}
            </p>

            <div className='mt-7 flex flex-wrap items-center gap-3'>
              <Button
                className='group h-11 rounded-none px-5 text-sm font-medium'
                render={
                  <Link to={props.isAuthenticated ? '/keys' : '/sign-up'} />
                }
              >
                {t('Create API Key')}
                <ArrowUpRight className='ml-0.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5 group-hover:-translate-y-0.5' />
              </Button>
              <Button
                variant='outline'
                className='group border-foreground/25 hover:border-foreground/50 hover:bg-muted/40 h-11 rounded-none px-5 text-sm font-medium'
                render={<Link to='/pricing' />}
              >
                {t('View Pricing')}
                <ArrowRight className='ml-0.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
              </Button>
            </div>
          </div>
        </div>

        <ConnectionPanel className='mt-10 md:mt-14' />
      </div>
    </section>
  )
}
