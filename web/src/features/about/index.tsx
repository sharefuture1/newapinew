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
import { useQuery } from '@tanstack/react-query'
import { Construction } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { RichContent } from '@/components/rich-content'
import { Skeleton } from '@/components/ui/skeleton'
import { isHttpUrl, isLikelyHtml } from '@/lib/content-format'
import { requireServerSuccess } from '@/lib/server-error-message'
import {
  MODIFIED_SOURCE_URL,
  UPSTREAM_ATTRIBUTION_TEXT,
  UPSTREAM_PROJECT_URL,
} from '@/lib/wending-brand'

import { getAboutContent } from './api'

// Modified by WendingAI (2026-11): the upstream attribution required by the
// New API NOTICE (AGPLv3 §7(b)) is shown here on every About variant instead
// of in the site footer.
function OpenSourceNotice() {
  const { t } = useTranslation()
  return (
    <section
      aria-label={t('Open-source notice')}
      className='border-border/60 text-muted-foreground mx-auto w-full max-w-4xl border-t px-4 py-6 text-xs leading-relaxed'
    >
      <p className='text-foreground/80 mb-1 font-medium'>
        {t('Open-source notice')}
      </p>
      <p>{UPSTREAM_ATTRIBUTION_TEXT}</p>
      <p>
        {t('Original project')}:{' '}
        <a
          href={UPSTREAM_PROJECT_URL}
          target='_blank'
          rel='noopener noreferrer'
          className='text-primary hover:underline'
        >
          {UPSTREAM_PROJECT_URL}
        </a>{' '}
        · {t('License')}:{' '}
        <a
          href={`${UPSTREAM_PROJECT_URL}/blob/main/LICENSE`}
          target='_blank'
          rel='noopener noreferrer'
          className='text-primary hover:underline'
        >
          AGPL-3.0
        </a>
      </p>
      <p>
        {t('This service runs a modified version. Source code')}:{' '}
        <a
          href={MODIFIED_SOURCE_URL}
          target='_blank'
          rel='noopener noreferrer'
          className='text-primary hover:underline'
        >
          {MODIFIED_SOURCE_URL}
        </a>
      </p>
    </section>
  )
}

function EmptyAboutState() {
  const { t } = useTranslation()

  return (
    <div className='flex min-h-[60vh] items-center justify-center p-8'>
      <div className='max-w-2xl space-y-6 text-center'>
        <div className='flex justify-center'>
          <Construction className='text-muted-foreground h-24 w-24' />
        </div>
        <div className='space-y-2'>
          <h2 className='text-2xl font-bold'>{t('No About Content Set')}</h2>
          <p className='text-muted-foreground'>
            {t(
              'The administrator has not configured any about content yet. You can set it in the settings page, supporting HTML or URL.'
            )}
          </p>
        </div>
      </div>
    </div>
  )
}

export function About() {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['about-content'],
    queryFn: async () => requireServerSuccess(await getAboutContent()),
  })

  const rawContent = data?.data?.trim() ?? ''
  const hasContent = rawContent.length > 0
  const isUrl = hasContent && isHttpUrl(rawContent)
  const contentIsHtml = hasContent && isLikelyHtml(rawContent)

  if (isLoading) {
    return (
      <PublicLayout>
        <div className='mx-auto flex max-w-4xl flex-col gap-4 py-12'>
          <Skeleton className='h-8 w-[45%]' />
          <Skeleton className='h-4 w-full' />
          <Skeleton className='h-4 w-[90%]' />
          <Skeleton className='h-4 w-[80%]' />
        </div>
      </PublicLayout>
    )
  }

  if (!hasContent) {
    return (
      <PublicLayout>
        <EmptyAboutState />
        <OpenSourceNotice />
      </PublicLayout>
    )
  }

  if (isUrl) {
    return (
      <PublicLayout showMainContainer={false}>
        <iframe
          src={rawContent}
          className='h-[calc(100vh-3.5rem)] w-full border-0'
          title={t('About')}
          sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-scripts'
        />
        <OpenSourceNotice />
      </PublicLayout>
    )
  }

  if (contentIsHtml) {
    return (
      <PublicLayout showMainContainer={false}>
        <RichContent
          mode='html'
          htmlVariant='isolated'
          content={rawContent}
          className='prose-neutral dark:prose-invert max-w-none'
        />
        <OpenSourceNotice />
      </PublicLayout>
    )
  }

  return (
    <PublicLayout>
      <div className='mx-auto max-w-6xl px-4 py-8'>
        <RichContent
          mode='markdown'
          content={rawContent}
          className='prose-neutral dark:prose-invert max-w-none'
        />
      </div>
      <OpenSourceNotice />
    </PublicLayout>
  )
}
