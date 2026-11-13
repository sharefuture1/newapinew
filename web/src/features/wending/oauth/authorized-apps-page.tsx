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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { formatQuota, formatTimestamp } from '@/lib/format'

import '../i18n'
import { listMyGrants, revokeMyGrant } from './api'
import type { WendingOAuthGrantView } from './types'

export function GrantList({
  grants,
  onRevoke,
  showUser,
}: {
  grants: WendingOAuthGrantView[]
  onRevoke: (grant: WendingOAuthGrantView) => void
  showUser?: boolean
}) {
  const { t } = useTranslation()
  if (grants.length === 0) {
    return (
      <p className='text-muted-foreground py-8 text-center text-sm'>
        {t('wending.grants.empty')}
      </p>
    )
  }
  return (
    <div className='space-y-3'>
      {grants.map((grant) => (
        <Card key={grant.id} size='sm'>
          <CardContent className='flex flex-wrap items-center gap-4'>
            {grant.client_icon_url ? (
              <img
                src={grant.client_icon_url}
                alt=''
                className='size-10 rounded-lg object-cover'
                referrerPolicy='no-referrer'
              />
            ) : (
              <div className='bg-primary text-primary-foreground flex size-10 items-center justify-center rounded-lg font-semibold'>
                {(grant.client_name || '?').slice(0, 1)}
              </div>
            )}
            <div className='min-w-0 flex-1 space-y-1'>
              <div className='flex flex-wrap items-center gap-2 font-medium'>
                {grant.client_name || grant.client_id}
                {grant.device_name ? (
                  <Badge variant='secondary'>{grant.device_name}</Badge>
                ) : null}
                {showUser ? (
                  <Badge variant='outline'>
                    {t('wending.clients.grantsUser', { id: grant.user_id })}
                  </Badge>
                ) : null}
                {grant.revoked_at ? (
                  <Badge variant='destructive'>
                    {t('wending.grants.revoked')}
                  </Badge>
                ) : null}
                {!grant.token_exists && !grant.revoked_at ? (
                  <Badge variant='secondary'>
                    {t('wending.grants.keyMissing')}
                  </Badge>
                ) : null}
              </div>
              <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs'>
                <span>
                  {t('wending.grants.authorizedAt', {
                    time: formatTimestamp(grant.created_at),
                  })}
                </span>
                {grant.last_used_at ? (
                  <span>
                    {t('wending.grants.lastUsed', {
                      time: formatTimestamp(grant.last_used_at),
                    })}
                  </span>
                ) : null}
                {grant.token_exists ? (
                  <>
                    <span>
                      {t('wending.grants.used', {
                        amount: formatQuota(grant.used_quota),
                      })}
                    </span>
                    <span>
                      {grant.unlimited_quota
                        ? t('wending.grants.unlimited')
                        : t('wending.grants.remaining', {
                            amount: formatQuota(grant.remain_quota),
                          })}
                    </span>
                  </>
                ) : null}
              </div>
            </div>
            {!grant.revoked_at ? (
              <Button
                variant='destructive'
                size='sm'
                onClick={() => onRevoke(grant)}
              >
                {t('wending.grants.revoke')}
              </Button>
            ) : null}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export function WendingAuthorizedAppsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [pending, setPending] = useState<WendingOAuthGrantView | null>(null)
  const grants = useQuery({
    queryKey: ['wending-oauth-grants-self'],
    queryFn: listMyGrants,
  })
  const revoke = useMutation({
    mutationFn: (id: number) => revokeMyGrant(id),
    onSuccess: (res) => {
      if (res.success) {
        toast.success(t('wending.grants.revoked'))
      } else {
        toast.error(res.message)
      }
      setPending(null)
      queryClient.invalidateQueries({ queryKey: ['wending-oauth-grants-self'] })
    },
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('wending.grants.title')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <p className='text-muted-foreground text-sm'>
            {t('wending.grants.desc')}
          </p>
          <GrantList grants={grants.data?.data ?? []} onRevoke={setPending} />
        </div>
        <ConfirmDialog
          open={pending !== null}
          onOpenChange={(open) => !open && setPending(null)}
          title={t('wending.grants.revokeTitle', {
            app: pending?.client_name ?? '',
          })}
          desc={t('wending.grants.revokeDesc')}
          destructive
          confirmText={t('wending.grants.revoke')}
          isLoading={revoke.isPending}
          handleConfirm={() => pending && revoke.mutate(pending.id)}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
