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
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { handleServerError } from '@/lib/handle-server-error'
import { AuthOperationError } from '@/lib/secure-verification'
import { createServerError } from '@/lib/server-error-message'

import { telegramMiniAppLogin } from '../api'
import {
  getTelegramMiniAppInitData,
  isTelegramMiniAppEnvironment,
  loadTelegramWebAppScript,
  readyTelegramMiniApp,
  waitForTelegramMiniAppInitData,
} from '../lib/telegram-mini-app'
import { useAuthRedirect } from './use-auth-redirect'

/**
 * Attempts a Telegram Mini App login exactly once when the page runs
 * inside Telegram. Outside Telegram this hook does nothing and the normal
 * sign-in form (including the Telegram OIDC button) is unchanged. A failed
 * attempt is never retried automatically; the user can still sign in with
 * any other method on the page.
 */
export function useTelegramMiniAppLogin(redirectTo?: string) {
  const { t } = useTranslation()
  const { handleLoginResult } = useAuthRedirect()
  const attempted = useRef(false)
  const [isMiniApp, setIsMiniApp] = useState(() =>
    isTelegramMiniAppEnvironment()
  )
  const [isAttempting, setIsAttempting] = useState(false)

  useEffect(() => {
    if (attempted.current) return
    if (!isTelegramMiniAppEnvironment()) return
    attempted.current = true
    setIsMiniApp(true)
    let cancelled = false

    const attempt = async (initData: string) => {
      setIsAttempting(true)
      try {
        const res = await telegramMiniAppLogin(initData)
        if (cancelled) return
        if (res?.success) {
          if (await handleLoginResult(res.data, redirectTo)) {
            toast.success(t('Signed in with Telegram'))
          }
        } else {
          handleServerError(
            createServerError(res, t('Login failed'))
          )
        }
      } catch (error: unknown) {
        if (!cancelled) {
          handleServerError(AuthOperationError.from(error, t('Login failed')))
        }
      } finally {
        if (!cancelled) setIsAttempting(false)
      }
    }

    const start = async () => {
      const direct = getTelegramMiniAppInitData()
      if (direct) {
        readyTelegramMiniApp()
        await attempt(direct)
        return
      }
      const loaded = await loadTelegramWebAppScript()
      if (cancelled || !loaded) return
      readyTelegramMiniApp()
      const initData = await waitForTelegramMiniAppInitData()
      if (initData && !cancelled) await attempt(initData)
    }
    void start()
    return () => {
      cancelled = true
    }
  }, [handleLoginResult, redirectTo, t])

  return { isMiniApp, isAttemptingMiniAppLogin: isAttempting }
}
