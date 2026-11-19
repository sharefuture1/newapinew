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
// Modified by WendingAI (2026-11): only English (fallback) and Simplified
// Chinese (default market) are bundled into the entry chunk; the other
// interface languages are fetched as separate chunks the first time they are
// used, which keeps ~3 MB of JSON out of the initial download.
import i18n, { type BackendModule, type ResourceKey } from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import { convertDetectedLanguage } from './languages'
import en from './locales/en.json'
import zhCN from './locales/zh.json'

export const resources = {
  en,
  zhCN,
} as const

type LocaleModule = { default: Record<string, ResourceKey> }

export const lazyLocaleLoaders: Record<string, () => Promise<LocaleModule>> = {
  fr: () => import('./locales/fr.json'),
  ru: () => import('./locales/ru.json'),
  ja: () => import('./locales/ja.json'),
  vi: () => import('./locales/vi.json'),
  zhTW: () => import('./locales/zh-TW.json'),
}

export const lazyLocaleBackend: BackendModule = {
  type: 'backend',
  init() {},
  read(language, namespace, callback) {
    const load = lazyLocaleLoaders[language]
    if (!load) {
      // Bundled languages (or unknown codes) have nothing to fetch.
      callback(null, {})
      return
    }
    void (async () => {
      let data: ResourceKey
      try {
        data = (await load()).default[namespace] ?? {}
      } catch (error) {
        callback(error as Error, null)
        return
      }
      callback(null, data)
    })()
  },
}

// Resolves once the detected language (bundled or lazily fetched) is ready;
// main.tsx waits for it before the first render so no raw keys flash.
export const i18nReady = i18n
  .use(lazyLocaleBackend)
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    partialBundledLanguages: true,
    fallbackLng: 'en',
    supportedLngs: ['en', 'zhCN', 'fr', 'ru', 'ja', 'vi', 'zhTW'],
    load: 'currentOnly',
    nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false, // not needed for react as it escapes by default
    },
    react: {
      // Lazily loaded languages resolve after the first render; re-render
      // instead of suspending (the app has no Suspense boundary at the root).
      useSuspense: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      // Browsers report `zh-CN`/`zh-TW`/`zh`; map them onto our `zhCN`/`zhTW`
      // codes (non-Chinese codes pass through for normal supportedLngs matching).
      convertDetectedLanguage,
    },
  })

export default i18n
