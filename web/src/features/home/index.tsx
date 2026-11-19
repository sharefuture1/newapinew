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
import { lazy, Suspense, useCallback, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Footer } from '@/components/layout/components/footer'
import { LoadingState } from '@/components/loading-state'
import { useTheme } from '@/context/theme-provider'
import { removeBootShell } from '@/lib/boot-shell'
import { isLikelyHtml } from '@/lib/content-format'
import { useAuthStore } from '@/stores/auth-store'

import { Hero, HowItWorks, QuickAccess } from './components'
import { useHomePageContent } from './hooks'

// Custom home content may be Markdown/HTML, which pulls in marked, dompurify
// and KaTeX. Keep that weight out of the default landing-page chunk.
const RichContent = lazy(() =>
  import('@/components/rich-content').then((module) => ({
    default: module.RichContent,
  }))
)

// Pricing preview is below the fold and pulls the pricing data layer.
const PricePreview = lazy(() =>
  import('./components/sections/price-preview').then((module) => ({
    default: module.PricePreview,
  }))
)

export function Home() {
  const { i18n, t } = useTranslation()
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const { resolvedTheme } = useTheme()
  const { auth } = useAuthStore()
  const isAuthenticated = !!auth.user
  const { content, isLoaded, isUrl } = useHomePageContent()

  // WendingAI: hand over from the static index.html shell once the real home
  // page (default hero or custom content) can render.
  useEffect(() => {
    if (isLoaded) removeBootShell()
  }, [isLoaded])

  const syncIframePreferences = useCallback(() => {
    try {
      iframeRef.current?.contentWindow?.postMessage(
        { themeMode: resolvedTheme },
        '*'
      )
      iframeRef.current?.contentWindow?.postMessage(
        { lang: i18n.language },
        '*'
      )
    } catch {
      // Cross-origin frames may reject access while navigating.
    }
  }, [i18n.language, resolvedTheme])

  useEffect(() => {
    if (isUrl) {
      syncIframePreferences()
    }
  }, [isUrl, syncIframePreferences])

  if (!isLoaded) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='flex min-h-screen items-center justify-center'>
          <LoadingState message={t('Loading...')} />
        </main>
      </PublicLayout>
    )
  }

  if (content) {
    if (isUrl) {
      return (
        <PublicLayout showMainContainer={false}>
          {/*
            allow-top-navigation-by-user-activation: the custom home page URL is
            admin-configured (trusted); this lets its target="_top" nav/menu links
            navigate the top-level window on user click. The default sandbox blocks
            this on desktop, while some mobile browsers allow it via allow-popups,
            causing inconsistent behavior. This token only permits user-activated
            top-level navigation and does NOT grant same-origin access.
          */}
          <iframe
            ref={iframeRef}
            src={content}
            className='h-screen w-full border-none'
            title={t('Custom Home Page')}
            sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-scripts allow-top-navigation-by-user-activation'
            onLoad={syncIframePreferences}
          />
        </PublicLayout>
      )
    }

    const contentIsHtml = isLikelyHtml(content)

    if (contentIsHtml) {
      return (
        <PublicLayout showMainContainer={false}>
          <Suspense fallback={<LoadingState className='min-h-screen' />}>
            <RichContent
              mode='html'
              htmlVariant='isolated'
              content={content}
              className='custom-home-content'
            />
          </Suspense>
          <Footer />
        </PublicLayout>
      )
    }

    return (
      <PublicLayout>
        <div className='mx-auto max-w-6xl px-4 py-8'>
          <Suspense fallback={<LoadingState />}>
            <RichContent
              mode='markdown'
              content={content}
              className='custom-home-content'
            />
          </Suspense>
        </div>
        <Footer />
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <main>
        <Hero isAuthenticated={isAuthenticated} />
        <QuickAccess />
        <Suspense fallback={null}>
          <PricePreview />
        </Suspense>
        <HowItWorks isAuthenticated={isAuthenticated} />
      </main>
      <Footer />
    </PublicLayout>
  )
}
