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

// Helpers for running inside a Telegram Mini App (WebApp). The Mini App
// script is only loaded in that environment; a normal browser never fetches
// it and keeps the existing OIDC login button.

const TELEGRAM_WEB_APP_SCRIPT_URL = 'https://telegram.org/js/telegram-web-app.js'

type TelegramWebApp = {
  initData: string
  ready: () => void
  expand: () => void
}

function getTelegramWebApp(): TelegramWebApp | null {
  const browserWindow = window as unknown as Record<string, unknown>
  const telegram = browserWindow.Telegram
  if (!telegram || typeof telegram !== 'object') return null
  const webApp = (telegram as Record<string, unknown>).WebApp
  if (!webApp || typeof webApp !== 'object') return null
  const app = webApp as Record<string, unknown>
  if (typeof app.initData !== 'string') return null
  return {
    initData: app.initData,
    ready: () => {
      if (typeof app.ready === 'function') {
        ;(app.ready as () => void)()
      }
    },
    expand: () => {
      if (typeof app.expand === 'function') {
        ;(app.expand as () => void)()
      }
    },
  }
}

/** The signed initData payload, or null outside a Telegram Mini App. */
export function getTelegramMiniAppInitData(): string | null {
  const webApp = getTelegramWebApp()
  if (!webApp || !webApp.initData) return null
  return webApp.initData
}

/**
 * True when the page runs inside a Telegram Mini App. Telegram appends
 * tgWebAppData to the URL, so this is also true before the WebApp script
 * has loaded and initData is available.
 */
export function isTelegramMiniAppEnvironment(): boolean {
  if (getTelegramMiniAppInitData()) return true
  const params = new URLSearchParams(window.location.search)
  return params.has('tgWebAppData') || params.has('tgWebAppPlatform')
}

/** Tell Telegram the Mini App is ready and expand it to full height. */
export function readyTelegramMiniApp(): void {
  const webApp = getTelegramWebApp()
  if (!webApp) return
  webApp.ready()
  webApp.expand()
}

/**
 * Load the Telegram WebApp script. Returns true when
 * window.Telegram.WebApp is available afterwards. A no-op when the script
 * is already present.
 */
export function loadTelegramWebAppScript(timeoutMs = 3000): Promise<boolean> {
  if (getTelegramWebApp()) return Promise.resolve(true)
  const existing = document.querySelector(
    `script[src="${TELEGRAM_WEB_APP_SCRIPT_URL}"]`
  )
  if (existing) {
    return new Promise((resolve) => {
      const timer = window.setTimeout(() => resolve(Boolean(getTelegramWebApp())), timeoutMs)
      existing.addEventListener('load', () => {
        window.clearTimeout(timer)
        resolve(Boolean(getTelegramWebApp()))
      }, { once: true })
    })
  }
  return new Promise((resolve) => {
    const script = document.createElement('script')
    script.async = true
    script.src = TELEGRAM_WEB_APP_SCRIPT_URL
    const timer = window.setTimeout(() => {
      script.remove()
      resolve(false)
    }, timeoutMs)
    script.addEventListener('load', () => {
      window.clearTimeout(timer)
      resolve(Boolean(getTelegramWebApp()))
    }, { once: true })
    script.addEventListener('error', () => {
      window.clearTimeout(timer)
      script.remove()
      resolve(false)
    }, { once: true })
    document.head.appendChild(script)
  })
}

/**
 * Wait for initData to appear (Telegram may deliver it slightly after the
 * script loads). Returns null on timeout.
 */
export async function waitForTelegramMiniAppInitData(
  timeoutMs = 1500
): Promise<string | null> {
  const deadline = Date.now() + timeoutMs
  for (;;) {
    const initData = getTelegramMiniAppInitData()
    if (initData) return initData
    if (Date.now() >= deadline) return null
    await new Promise((resolve) => window.setTimeout(resolve, 100))
  }
}
