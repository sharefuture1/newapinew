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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

/** The OpenAI-compatible route every client dials first. */
const PRIMARY_ENDPOINT = '/chat/completions'

interface ConnectionPanelProps {
  className?: string
}

export function ConnectionPanel(props: ConnectionPanelProps) {
  const { t } = useTranslation()
  const { status } = useStatus()

  // Admins set `server_address`; fall back to the address the visitor is on.
  const baseUrl = useMemo(() => {
    const configured = status?.server_address
    if (typeof configured === 'string' && configured.trim()) {
      return configured.trim().replace(/\/+$/, '')
    }
    if (typeof window !== 'undefined') {
      return window.location.origin
    }
    return ''
  }, [status])

  const fullUrl = `${baseUrl}/v1`
  const docsUrl =
    typeof status?.docs_link === 'string' && status.docs_link
      ? status.docs_link
      : undefined

  return (
    <div
      className={cn(
        'bg-muted/40 grid overflow-hidden border md:grid-cols-2',
        props.className
      )}
    >
      <div className='flex min-w-0 flex-col justify-between gap-6 p-5 sm:p-7'>
        <div className='min-w-0'>
          <p className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
            {t('Base URL')}
          </p>
          <div className='mt-2 flex min-w-0 items-center gap-2'>
            <p className='min-w-0 truncate font-mono text-lg leading-tight font-semibold sm:text-2xl'>
              {fullUrl}
            </p>
            {baseUrl && (
              <CopyButton
                value={fullUrl}
                tooltip={t('Copy base URL')}
                className='shrink-0'
              />
            )}
          </div>
        </div>

        <div className='border-border/60 flex flex-wrap items-center gap-2 border-t pt-4'>
          <span className='bg-primary/10 text-primary rounded px-1.5 py-0.5 font-mono text-[11px] font-bold tracking-wide'>
            POST
          </span>
          <span className='text-muted-foreground min-w-0 truncate font-mono text-sm'>
            {PRIMARY_ENDPOINT}
          </span>
        </div>
      </div>

      <div className='border-border/60 flex min-w-0 flex-col justify-between gap-6 border-t p-5 sm:p-7 md:border-t-0 md:border-l'>
        <div className='min-w-0'>
          <p className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
            {t('Supported Applications')}
          </p>
          <p className='mt-2 text-sm leading-relaxed'>
            {t(
              'Any OpenAI-compatible client works: fill in the base URL above with an API key from the console.'
            )}
          </p>
        </div>

        <div className='flex flex-wrap gap-2'>
          <Button
            variant='secondary'
            size='sm'
            className='h-9 rounded-none px-4'
            render={<a href='https://cherry-ai.com' target='_blank' rel='noopener noreferrer' />}
          >
            Cherry Studio
          </Button>
          <Button
            variant='secondary'
            size='sm'
            className='h-9 rounded-none px-4'
            render={<a href='https://ccswitch.io' target='_blank' rel='noopener noreferrer' />}
          >
            CC Switch
          </Button>
          {docsUrl && (
            <Button
              variant='outline'
              size='sm'
              className='h-9 rounded-none px-4'
              render={
                <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
              }
            >
              {t('Docs')}
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
