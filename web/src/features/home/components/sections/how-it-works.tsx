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
import { ArrowUpRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface HowItWorksProps {
  isAuthenticated?: boolean
}

export function HowItWorks(props: HowItWorksProps) {
  const { t } = useTranslation()

  const steps = [
    {
      num: '01',
      title: t('Pick a model'),
      desc: t('Browse the model square and compare price and endpoints.'),
      href: '/pricing' as const,
    },
    {
      num: '02',
      title: t('Create a key'),
      desc: t('Generate an API key in the console.'),
      href: props.isAuthenticated ? ('/keys' as const) : ('/sign-up' as const),
    },
    {
      num: '03',
      title: t('Connect your app'),
      desc: t('Fill in the base URL, key and model name.'),
      href: props.isAuthenticated
        ? ('/dashboard' as const)
        : ('/sign-up' as const),
    },
  ]

  return (
    <section className='px-5 py-14 sm:px-8 md:py-20'>
      <div className='mx-auto w-full max-w-container'>
        <div className='border-border/60 border-t'>
          {steps.map((step) => (
            <Link
              key={step.num}
              to={step.href}
              className='group border-border/60 hover:bg-muted/30 grid grid-cols-[2.5rem_1fr_auto] items-center gap-4 border-b py-6 transition-colors sm:grid-cols-[4rem_12rem_1fr_auto] sm:gap-6 md:py-7'
            >
              <span className='text-muted-foreground font-mono text-xs tabular-nums'>
                {step.num}
              </span>
              <span className='text-base font-semibold tracking-tight sm:text-lg'>
                {step.title}
              </span>
              <span className='text-muted-foreground col-start-2 text-sm leading-relaxed sm:col-start-3'>
                {step.desc}
              </span>
              <ArrowUpRight
                aria-hidden
                className='text-muted-foreground group-hover:text-foreground col-start-3 row-start-1 size-4 justify-self-end transition-all duration-200 group-hover:-translate-y-0.5 group-hover:translate-x-0.5 sm:col-start-4'
              />
            </Link>
          ))}
        </div>
      </div>
    </section>
  )
}
