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
import i18next from 'i18next'
import { useEffect, useState } from 'react'

import { isHttpUrl } from '@/lib/content-format'
import { handleServerError } from '@/lib/handle-server-error'

import { getHomePageContent } from '../api'
import type { HomePageContentResult } from '../types'

const STORAGE_KEY = 'home_page_content'

/**
 * The default landing page needs no API data. Stop blocking it when the
 * custom-content endpoint is slow; late responses still replace the page.
 */
const CONTENT_LOAD_TIMEOUT_MS = 1500

/**
 * Hook to load and manage custom home page content
 * Supports both Markdown/HTML content and iframe URLs
 */
export function useHomePageContent(): HomePageContentResult {
  const [content, setContent] = useState<string>('')
  const [isLoaded, setIsLoaded] = useState(false)

  useEffect(() => {
    let mounted = true

    // Render cached content immediately instead of waiting for the API.
    try {
      const cached = localStorage.getItem(STORAGE_KEY)
      if (cached) {
        setContent(cached)
        setIsLoaded(true)
      }
    } catch {
      // Storage can be unavailable in privacy modes; fall back to the API.
    }

    const fallbackTimer = window.setTimeout(() => {
      if (mounted) {
        setIsLoaded(true)
      }
    }, CONTENT_LOAD_TIMEOUT_MS)

    const loadContent = async () => {
      try {
        const response = await getHomePageContent()
        const { success, data } = response

        if (!mounted) return

        if (success && data) {
          setContent(data)
          localStorage.setItem(STORAGE_KEY, data)
        } else {
          // Clear content if API returns empty
          setContent('')
          localStorage.removeItem(STORAGE_KEY)
        }
      } catch (error) {
        if (!mounted) return
        handleServerError(error, i18next.t('Failed to load home page content'))
      } finally {
        if (mounted) {
          window.clearTimeout(fallbackTimer)
          setIsLoaded(true)
        }
      }
    }

    loadContent()

    return () => {
      mounted = false
      window.clearTimeout(fallbackTimer)
    }
  }, [])

  const isUrl = isHttpUrl(content)

  return { content, isLoaded, isUrl }
}
