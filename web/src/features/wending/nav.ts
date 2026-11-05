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
import { useQuery } from '@tanstack/react-query'
import { AppWindow, ExternalLink, KeyRound } from 'lucide-react'
import { createElement, type ElementType, type SVGProps } from 'react'
import { useTranslation } from 'react-i18next'

import type { NavGroup, NavItem } from '@/components/layout/types'
import { ROLE } from '@/lib/roles'

import './i18n'
import { listSidebarApps } from './oauth/api'

function makeAppIcon(iconUrl: string): ElementType {
  if (!iconUrl) return ExternalLink
  return function AppIcon(props: SVGProps<SVGSVGElement>) {
    void props
    return createElement('img', {
      src: iconUrl,
      alt: '',
      className: 'size-4 shrink-0 rounded object-cover',
      loading: 'lazy',
      referrerPolicy: 'no-referrer',
    })
  }
}

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

/** Dynamic "问鼎应用" group: opens each client's launch_url in a new tab. */
export function useWendingAppsNavGroup(): NavGroup | null {
  const { t } = useTranslation()
  const { data } = useQuery({
    queryKey: ['wending', 'sidebar-apps'],
    queryFn: async () => {
      const res = await listSidebarApps()
      if (!res.success) return []
      return res.data ?? []
    },
    staleTime: 60_000,
    retry: false,
  })
  if (!data?.length) return null
  return {
    id: 'wending-apps',
    title: t('wending.nav.apps'),
    items: data.map((app) => ({
      title: app.name,
      url: app.launch_url,
      icon: makeAppIcon(app.icon_url),
    })),
  }
}
