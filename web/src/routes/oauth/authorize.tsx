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
// Wending fork-local route: OAuth consent screen ("问鼎AI账号一键授权").
import { createFileRoute, redirect } from '@tanstack/react-router'

import { WendingAuthorizePage } from '@/features/wending/oauth/authorize-page'
import { resolveAuthentication } from '@/lib/auth-session'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/oauth/authorize')({
  beforeLoad: async ({ location }) => {
    await resolveAuthentication()
    const { auth } = useAuthStore.getState()
    if (!auth.user || !auth.accessToken) {
      throw redirect({
        to: '/sign-in',
        search: { redirect: location.pathname + location.searchStr },
      })
    }
  },
  component: AuthorizeRoute,
})

function AuthorizeRoute() {
  // Read the raw query string: the router's JSON search parsing would turn
  // numeric-looking values (e.g. state=123) into numbers.
  const search = new URLSearchParams(window.location.search)
  const get = (key: string) => search.get(key) ?? ''
  return (
    <WendingAuthorizePage
      params={{
        response_type: get('response_type'),
        client_id: get('client_id'),
        redirect_uri: get('redirect_uri'),
        state: get('state'),
        scope: get('scope'),
        code_challenge: get('code_challenge'),
        code_challenge_method: get('code_challenge_method'),
      }}
    />
  )
}
