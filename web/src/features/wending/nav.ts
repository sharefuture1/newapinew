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
// Wending fork-local sidebar entries, spread into the upstream sidebar groups.
import { AppWindow, KeyRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { NavItem } from '@/components/layout/types'
import { ROLE } from '@/lib/roles'

import './i18n'

export function useWendingNavItems(): {
  personal: NavItem[]
  admin: NavItem[]
} {
  const { t } = useTranslation()
  return {
    personal: [
      {
        title: t('wending.nav.authorizedApps'),
        url: '/wending/authorized-apps',
        icon: AppWindow,
      },
    ],
    admin: [
      {
        title: t('wending.nav.oauthClients'),
        url: '/wending/oauth-clients',
        icon: KeyRound,
        requiredRole: ROLE.SUPER_ADMIN,
      },
    ],
  }
}
