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
import { afterEach, describe, expect, test, vi } from 'vitest'

import {
  getTelegramMiniAppInitData,
  isTelegramMiniAppEnvironment,
} from '../telegram-mini-app'

const INIT_DATA = 'query_id=1&user=%7B%22id%22%3A42%7D&hash=abc'

function setTelegramWebApp(initData?: string) {
  const browserWindow = window as unknown as Record<string, unknown>
  if (initData === undefined) {
    delete browserWindow.Telegram
    return
  }
  browserWindow.Telegram = {
    WebApp: { initData, ready: vi.fn(), expand: vi.fn() },
  }
}

function setLocation(path: string) {
  window.history.replaceState(null, '', path)
}

afterEach(() => {
  setTelegramWebApp()
  setLocation('/sign-in')
})

describe('Telegram Mini App initData survives SPA navigation', () => {
  test('initData still holds after tgWebAppData is stripped from the URL', () => {
    // Landing in Telegram: the static telegram-web-app.js tag in index.html
    // parses tgWebAppData from the URL at page load and stores initData on
    // window.Telegram.WebApp.
    setLocation('/?tgWebAppData=query_id%3D1&tgWebAppPlatform=android')
    setTelegramWebApp(INIT_DATA)

    // The SPA then navigates Home -> /sign-in, which drops the
    // tgWebAppData/tgWebAppPlatform params from the URL.
    setLocation('/sign-in')

    expect(getTelegramMiniAppInitData()).toBe(INIT_DATA)
    expect(isTelegramMiniAppEnvironment()).toBe(true)
  })

  test('without a pre-loaded WebApp object, stripping the params loses the Mini App', () => {
    // This is the bug the static script tag fixes: if the script was never
    // loaded against the original URL, nothing holds initData once the
    // params are gone.
    setLocation('/?tgWebAppData=query_id%3D1&tgWebAppPlatform=android')
    expect(isTelegramMiniAppEnvironment()).toBe(true)

    setLocation('/sign-in')

    expect(getTelegramMiniAppInitData()).toBe(null)
    expect(isTelegramMiniAppEnvironment()).toBe(false)
  })
})
