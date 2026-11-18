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
// Modified by WendingAI (2026-11): WendingAI Shop shortcut on the wallet page
// (AGPLv3 §7(c) change marking).
import { ArrowUpRight, ShoppingBag } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { WENDING_SHOP_URL } from '@/lib/wending-brand'

export function ShopCta() {
  const { t } = useTranslation()
  return (
    <div className='border-border/70 bg-card flex flex-col gap-4 rounded-xl border p-4 sm:flex-row sm:items-center sm:justify-between sm:p-5'>
      <div className='flex min-w-0 items-start gap-3'>
        <span className='bg-primary/10 text-primary inline-flex size-10 shrink-0 items-center justify-center rounded-lg'>
          <ShoppingBag className='size-5' aria-hidden='true' />
        </span>
        <div className='min-w-0'>
          <p className='text-sm font-semibold'>{t('WendingAI Shop')}</p>
          <p className='text-muted-foreground mt-0.5 text-sm leading-relaxed'>
            {t('Plans, credits and AI services in one store.')}
          </p>
        </div>
      </div>
      <Button
        variant='outline'
        className='h-10 w-full shrink-0 sm:w-auto'
        render={
          <a
            href={WENDING_SHOP_URL}
            target='_blank'
            rel='noopener noreferrer'
          />
        }
      >
        {t('Visit shop')}
        <ArrowUpRight className='size-4' aria-hidden='true' />
      </Button>
    </div>
  )
}
