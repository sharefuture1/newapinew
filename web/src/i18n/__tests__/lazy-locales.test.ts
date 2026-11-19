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
import { describe, expect, it } from 'vitest'

import i18n, {
  i18nReady,
  lazyLocaleBackend,
  lazyLocaleLoaders,
  resources,
} from '../config'
import { INTERFACE_LANGUAGE_OPTIONS } from '../languages'

type ReadCallback = Parameters<NonNullable<typeof lazyLocaleBackend.read>>[2]

function read(language: string) {
  return new Promise<Record<string, unknown>>((resolve, reject) => {
    const callback: ReadCallback = (error, data) =>
      error ? reject(error) : resolve((data ?? {}) as Record<string, unknown>)
    lazyLocaleBackend.read(language, 'translation', callback)
  })
}

describe('lazy interface languages', () => {
  it('bundles only English and Simplified Chinese', () => {
    expect(Object.keys(resources).sort()).toEqual(['en', 'zhCN'])
  })

  it('has a loader for every other interface language', () => {
    const lazy = INTERFACE_LANGUAGE_OPTIONS.map((l) => l.code).filter(
      (code) => !(code in resources)
    )
    expect(Object.keys(lazyLocaleLoaders).sort()).toEqual([...lazy].sort())
  })

  it('loads a lazy language on demand and switches to it', async () => {
    await i18nReady
    const fr = await read('fr')
    expect(Object.keys(fr).length).toBeGreaterThan(1000)
    expect(await read('en')).toEqual({})

    await i18n.changeLanguage('fr')
    expect(i18n.hasResourceBundle('fr', 'translation')).toBe(true)
    expect(i18n.t('Sign in')).not.toBe('Sign in')
    // test-setup re-initialises the shared instance for component tests, so
    // check the bundled Chinese table directly.
    expect(resources.zhCN.translation['Sign in']).toBe('登录')
    await i18n.changeLanguage('en')
  })
})
