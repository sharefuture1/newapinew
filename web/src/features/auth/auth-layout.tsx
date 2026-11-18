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
// Modified by WendingAI (2026-11): branded split layout for sign-in/sign-up
// (AGPLv3 §7(c) change marking).
import { Link } from '@tanstack/react-router'
import { ShieldCheck, Wallet, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

function BrandPanel() {
  const { t } = useTranslation()
  const items = [
    { icon: Zap, label: t('OpenAI-compatible') },
    { icon: ShieldCheck, label: t('Stable & fast') },
    { icon: Wallet, label: t('Pay as you go') },
  ]
  return (
    <aside className='relative hidden overflow-hidden bg-[#0b1220] text-white lg:flex lg:flex-col lg:justify-end lg:p-12 xl:p-16'>
      <div
        aria-hidden='true'
        className='absolute inset-0 bg-[radial-gradient(900px_520px_at_85%_-10%,rgba(37,99,235,0.55),transparent_60%),radial-gradient(700px_420px_at_0%_110%,rgba(56,189,248,0.25),transparent_60%)]'
      />
      <div className='relative'>
        <img
          src='/favicon.svg'
          alt=''
          aria-hidden='true'
          className='size-12 rounded-xl shadow-lg shadow-blue-900/40'
        />
        <p className='mt-8 text-3xl leading-tight font-semibold tracking-tight xl:text-4xl'>
          {t('Connect your apps to large language models.')}
        </p>
        <p className='mt-3 max-w-md text-sm leading-relaxed text-white/70'>
          {t('Enterprise AI API Gateway')}
        </p>
        <ul className='mt-8 flex flex-wrap gap-x-6 gap-y-3 text-sm text-white/80'>
          {items.map(({ icon: Icon, label }) => (
            <li key={label} className='inline-flex items-center gap-2'>
              <Icon className='size-4 text-sky-400' aria-hidden='true' />
              {label}
            </li>
          ))}
        </ul>
      </div>
    </aside>
  )
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()

  return (
    <div className='relative grid min-h-svh max-w-none lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]'>
      <BrandPanel />
      <div className='relative flex min-h-svh flex-col'>
        <Link
          to='/'
          className='z-10 flex w-fit items-center gap-2 px-4 pt-4 transition-opacity hover:opacity-80 sm:px-8 sm:pt-8'
        >
          <div className='relative h-8 w-8'>
            {loading ? (
              <Skeleton className='absolute inset-0 rounded-lg' />
            ) : (
              <img
                src={logo}
                alt={t('Logo')}
                className='h-8 w-8 rounded-lg object-cover'
              />
            )}
          </div>
          {loading ? (
            <Skeleton className='h-6 w-24' />
          ) : (
            <h1 className='text-lg font-semibold tracking-tight'>
              {systemName}
            </h1>
          )}
        </Link>
        <div className='flex flex-1 items-center'>
          <div className='mx-auto flex w-full flex-col justify-center space-y-2 px-5 py-10 sm:w-[460px] sm:px-8'>
            {children}
          </div>
        </div>
      </div>
    </div>
  )
}
