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
  loadTelegramWebAppScript,
  waitForTelegramMiniAppInitData,
} from './telegram-mini-app'

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

function setLocationSearch(search: string) {
  window.history.replaceState(null, '', `/sign-in${search}`)
}

afterEach(() => {
  setTelegramWebApp()
  setLocationSearch('')
  document.head
    .querySelectorAll('script[src*="telegram-web-app.js"]')
    .forEach((script) => script.remove())
})

describe('Telegram Mini App environment detection', () => {
  test('no initData outside Telegram', () => {
    expect(getTelegramMiniAppInitData()).toBe(null)
    expect(isTelegramMiniAppEnvironment()).toBe(false)
  })

  test('initData is read from window.Telegram.WebApp', () => {
    setTelegramWebApp('query_id=1&user=%7B%22id%22%3A42%7D&hash=abc')
    expect(getTelegramMiniAppInitData()).toBe(
      'query_id=1&user=%7B%22id%22%3A42%7D&hash=abc'
    )
    expect(isTelegramMiniAppEnvironment()).toBe(true)
  })

  test('empty initData does not count as a Mini App', () => {
    setTelegramWebApp('')
    expect(getTelegramMiniAppInitData()).toBe(null)
    expect(isTelegramMiniAppEnvironment()).toBe(false)
  })

  test('tgWebAppData in the URL counts before the script loads', () => {
    setLocationSearch('?tgWebAppData=query_id%3D1&tgWebAppPlatform=android')
    expect(isTelegramMiniAppEnvironment()).toBe(true)
    expect(getTelegramMiniAppInitData()).toBe(null)
  })

  test('loadTelegramWebAppScript is a no-op when WebApp is present', async () => {
    setTelegramWebApp('init-data')
    await expect(loadTelegramWebAppScript()).resolves.toBe(true)
    expect(
      document.querySelector('script[src*="telegram-web-app.js"]')
    ).toBe(null)
  })

  test('loadTelegramWebAppScript times out when the script never loads', async () => {
    await expect(loadTelegramWebAppScript(50)).resolves.toBe(false)
  })

  test('waitForTelegramMiniAppInitData picks up a late initData', async () => {
    window.setTimeout(() => setTelegramWebApp('late-init-data'), 150)
    await expect(waitForTelegramMiniAppInitData(1500)).resolves.toBe(
      'late-init-data'
    )
  })

  test('waitForTelegramMiniAppInitData returns null on timeout', async () => {
    await expect(waitForTelegramMiniAppInitData(100)).resolves.toBe(null)
  })
})
