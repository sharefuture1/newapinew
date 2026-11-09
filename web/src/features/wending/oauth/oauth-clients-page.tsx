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
import { CopyButton } from '@/components/copy-button'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  formatQuota,
  parseQuotaFromDollars,
  quotaUnitsToEditableAmount,
} from '@/lib/format'

import '../i18n'
import {
  createClient,
  deleteClient,
  listAdminGrants,
  listClients,
  revokeAdminGrant,
  rotateClientSecret,
  updateClient,
} from './api'
import { GrantList } from './authorized-apps-page'
import type {
  WendingOAuthClient,
  WendingOAuthClientInput,
  WendingOAuthGrantView,
} from './types'

const EMPTY_INPUT: WendingOAuthClientInput = {
  name: '',
  description: '',
  icon_url: '',
  homepage_url: '',
  launch_url: '',
  show_in_sidebar: false,
  sidebar_order: 0,
  redirect_uris: [],
  is_public: false,
  enabled: true,
  default_unlimited_quota: false,
  default_remain_quota: 0,
  allow_unlimited_quota: true,
  allowed_group: '',
  model_limits_enabled: false,
  model_limits: '',
  trusted: false,
}

type Editing = { id: number | null; input: WendingOAuthClientInput }

function ClientForm({
  editing,
  onChange,
}: {
  editing: Editing
  onChange: (input: WendingOAuthClientInput) => void
}) {
  const { t } = useTranslation()
  const input = editing.input
  const set = <K extends keyof WendingOAuthClientInput>(
    key: K,
    value: WendingOAuthClientInput[K]
  ) => onChange({ ...input, [key]: value })
  const text = (
    key:
      | 'name'
      | 'description'
      | 'icon_url'
      | 'homepage_url'
      | 'launch_url'
      | 'allowed_group'
      | 'model_limits',
    label: string
  ) => (
    <div className='space-y-1'>
      <Label htmlFor={`wending-${key}`}>{label}</Label>
      <Input
        id={`wending-${key}`}
        value={input[key]}
        onChange={(e) => set(key, e.target.value)}
      />
    </div>
  )
  const toggle = (
    key:
      | 'is_public'
      | 'enabled'
      | 'show_in_sidebar'
      | 'allow_unlimited_quota'
      | 'default_unlimited_quota'
      | 'model_limits_enabled'
      | 'trusted',
    label: string
  ) => (
    <label className='flex items-center justify-between gap-2 text-sm'>
      {label}
      <Switch
        checked={input[key]}
        onCheckedChange={(checked) => set(key, checked)}
      />
    </label>
  )
  return (
    <div className='grid max-h-[65vh] gap-3 overflow-y-auto pr-1'>
      {text('name', t('wending.clients.name'))}
      {text('description', t('wending.clients.description'))}
      {text('icon_url', t('wending.clients.iconUrl'))}
      {text('homepage_url', t('wending.clients.homepageUrl'))}
      {text('launch_url', t('wending.clients.launchUrl'))}
      {toggle('show_in_sidebar', t('wending.clients.showInSidebar'))}
      <div className='space-y-1'>
        <Label htmlFor='wending-sidebar-order'>
          {t('wending.clients.sidebarOrder')}
        </Label>
        <Input
          id='wending-sidebar-order'
          type='number'
          value={input.sidebar_order}
          onChange={(e) => set('sidebar_order', Number(e.target.value) || 0)}
        />
      </div>
      <div className='space-y-1'>
        <Label htmlFor='wending-redirects'>
          {t('wending.clients.redirectUris')}
        </Label>
        <Textarea
          id='wending-redirects'
          rows={3}
          value={input.redirect_uris.join('\n')}
          onChange={(e) => set('redirect_uris', e.target.value.split('\n'))}
          placeholder='https://huabu.wending.ai/api/auth/wending/callback'
        />
      </div>
      {toggle('enabled', t('wending.clients.enabled'))}
      {toggle('is_public', t('wending.clients.isPublic'))}
      {toggle('trusted', t('wending.clients.trusted'))}
      <div className='space-y-1'>
        <Label htmlFor='wending-default-quota'>
          {t('wending.clients.defaultQuota')}
        </Label>
        <Input
          id='wending-default-quota'
          type='number'
          min={0}
          value={quotaUnitsToEditableAmount(input.default_remain_quota)}
          onChange={(e) =>
            set(
              'default_remain_quota',
              parseQuotaFromDollars(Number(e.target.value))
            )
          }
        />
      </div>
      {toggle('allow_unlimited_quota', t('wending.clients.allowUnlimited'))}
      {input.allow_unlimited_quota
        ? toggle(
            'default_unlimited_quota',
            t('wending.clients.defaultUnlimited')
          )
        : null}
      {text('allowed_group', t('wending.clients.allowedGroup'))}
      {toggle('model_limits_enabled', t('wending.clients.modelLimitsEnabled'))}
      {input.model_limits_enabled
        ? text('model_limits', t('wending.clients.modelLimits'))
        : null}
    </div>
  )
}

export function WendingOAuthClientsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState<Editing | null>(null)
  const [credentials, setCredentials] = useState<{
    clientId: string
    secret: string
  } | null>(null)
  const [confirm, setConfirm] = useState<{
    kind: 'delete' | 'rotate'
    client: WendingOAuthClient
  } | null>(null)
  const [grantUserId, setGrantUserId] = useState('')
  const [pendingGrant, setPendingGrant] =
    useState<WendingOAuthGrantView | null>(null)

  const clients = useQuery({
    queryKey: ['wending-oauth-clients'],
    queryFn: listClients,
  })
  const grants = useQuery({
    queryKey: ['wending-oauth-admin-grants', grantUserId],
    queryFn: () => listAdminGrants(Number(grantUserId) || undefined),
  })
  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['wending-oauth-clients'] })
    queryClient.invalidateQueries({ queryKey: ['wending-oauth-admin-grants'] })
  }

  const save = useMutation({
    mutationFn: async (value: Editing) => {
      const input = {
        ...value.input,
        redirect_uris: value.input.redirect_uris
          .map((uri) => uri.trim())
          .filter(Boolean),
      }
      if (value.id === null) {
        const res = await createClient(input)
        if (res.success && res.data) {
          setCredentials({
            clientId: res.data.client.client_id,
            secret: res.data.client_secret,
          })
        }
        return res
      }
      return updateClient(value.id, input)
    },
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message)
        return
      }
      toast.success(t('wending.clients.saved'))
      setEditing(null)
      refresh()
    },
  })

  const act = useMutation({
    mutationFn: async (value: {
      kind: 'delete' | 'rotate'
      client: WendingOAuthClient
    }) => {
      if (value.kind === 'delete') return deleteClient(value.client.id)
      const res = await rotateClientSecret(value.client.id)
      if (res.success && res.data) {
        setCredentials({
          clientId: value.client.client_id,
          secret: res.data.client_secret,
        })
      }
      return res
    },
    onSuccess: (res) => {
      if (!res.success) toast.error(res.message)
      setConfirm(null)
      refresh()
    },
  })

  const revokeGrant = useMutation({
    mutationFn: (id: number) => revokeAdminGrant(id),
    onSuccess: (res) => {
      if (!res.success) toast.error(res.message)
      setPendingGrant(null)
      refresh()
    },
  })

  const list = clients.data?.data ?? []
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <span className='inline-flex items-center gap-2'>
          {t('wending.clients.title')}
          <Badge variant='outline'>Root</Badge>
        </span>
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button onClick={() => setEditing({ id: null, input: EMPTY_INPUT })}>
          {t('wending.clients.create')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <p className='text-muted-foreground text-sm'>
            {t('wending.clients.desc')}
          </p>
          {list.length === 0 ? (
            <p className='text-muted-foreground py-6 text-center text-sm'>
              {t('wending.clients.empty')}
            </p>
          ) : null}
          {list.map((client) => (
            <Card key={client.id} size='sm'>
              <CardHeader>
                <CardTitle className='flex flex-wrap items-center gap-2'>
                  {client.name}
                  <Badge variant='outline'>
                    {client.is_public
                      ? t('wending.clients.public')
                      : t('wending.clients.confidential')}
                  </Badge>
                  {!client.enabled ? (
                    <Badge variant='destructive'>
                      {t('wending.clients.disabled')}
                    </Badge>
                  ) : null}
                  {client.trusted ? (
                    <Badge variant='secondary'>
                      {t('wending.clients.trustedBadge')}
                    </Badge>
                  ) : null}
                  {client.show_in_sidebar ? (
                    <Badge variant='secondary'>
                      {t('wending.clients.showInSidebar')}
                    </Badge>
                  ) : null}
                </CardTitle>
                <CardDescription>{client.description}</CardDescription>
              </CardHeader>
              <CardContent className='space-y-2 text-sm'>
                <div className='flex items-center gap-2'>
                  <span className='text-muted-foreground'>
                    {t('wending.clients.clientId')}:
                  </span>
                  <code className='break-all'>{client.client_id}</code>
                  <CopyButton value={client.client_id} />
                </div>
                <ul className='text-muted-foreground list-disc pl-5 text-xs break-all'>
                  {client.redirect_uris.map((uri) => (
                    <li key={uri}>{uri}</li>
                  ))}
                </ul>
                <p className='text-muted-foreground text-xs'>
                  {t('wending.clients.defaultQuota')}:{' '}
                  {client.default_unlimited_quota
                    ? t('wending.grants.unlimited')
                    : formatQuota(client.default_remain_quota)}
                  {client.allowed_group ? ` · ${client.allowed_group}` : ''}
                </p>
                <div className='flex flex-wrap gap-2 pt-1'>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() =>
                      setEditing({
                        id: client.id,
                        input: {
                          name: client.name,
                          description: client.description,
                          icon_url: client.icon_url,
                          homepage_url: client.homepage_url,
                          launch_url: client.launch_url,
                          show_in_sidebar: client.show_in_sidebar,
                          sidebar_order: client.sidebar_order,
                          redirect_uris: client.redirect_uris,
                          is_public: client.is_public,
                          enabled: client.enabled,
                          default_unlimited_quota:
                            client.default_unlimited_quota,
                          default_remain_quota: client.default_remain_quota,
                          allow_unlimited_quota: client.allow_unlimited_quota,
                          allowed_group: client.allowed_group,
                          model_limits_enabled: client.model_limits_enabled,
                          model_limits: client.model_limits,
                          trusted: client.trusted,
                        },
                      })
                    }
                  >
                    {t('wending.clients.edit')}
                  </Button>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => setConfirm({ kind: 'rotate', client })}
                  >
                    {t('wending.clients.rotateSecret')}
                  </Button>
                  <Button
                    size='sm'
                    variant='destructive'
                    onClick={() => setConfirm({ kind: 'delete', client })}
                  >
                    {t('wending.clients.delete')}
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}

          <div className='space-y-3 pt-4'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <h3 className='font-medium'>
                {t('wending.clients.grantsTitle')}
              </h3>
              <Input
                className='w-48'
                inputMode='numeric'
                placeholder={t('wending.clients.grantsUserId')}
                value={grantUserId}
                onChange={(e) =>
                  setGrantUserId(e.target.value.replaceAll(/\D/g, ''))
                }
              />
            </div>
            <GrantList
              grants={grants.data?.data ?? []}
              onRevoke={setPendingGrant}
              showUser
            />
          </div>
        </div>

        <Dialog
          open={editing !== null}
          onOpenChange={(open) => !open && setEditing(null)}
        >
          <DialogContent className='sm:max-w-lg'>
            <DialogHeader>
              <DialogTitle>
                {editing?.id === null
                  ? t('wending.clients.create')
                  : t('wending.clients.edit')}
              </DialogTitle>
            </DialogHeader>
            {editing ? (
              <ClientForm
                editing={editing}
                onChange={(input) => setEditing({ ...editing, input })}
              />
            ) : null}
            <DialogFooter>
              <Button
                disabled={save.isPending}
                onClick={() => editing && save.mutate(editing)}
              >
                {t('wending.clients.save')}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog
          open={credentials !== null}
          onOpenChange={(open) => !open && setCredentials(null)}
        >
          <DialogContent className='sm:max-w-lg'>
            <DialogHeader>
              <DialogTitle>{t('wending.clients.secretTitle')}</DialogTitle>
              <DialogDescription>
                {t('wending.clients.secretDesc')}
              </DialogDescription>
            </DialogHeader>
            {credentials ? (
              <div className='space-y-2 text-sm'>
                <div className='flex items-center gap-2'>
                  <span className='text-muted-foreground shrink-0'>
                    {t('wending.clients.clientId')}
                  </span>
                  <code className='break-all'>{credentials.clientId}</code>
                  <CopyButton value={credentials.clientId} />
                </div>
                <div className='flex items-center gap-2'>
                  <span className='text-muted-foreground shrink-0'>
                    {t('wending.clients.clientSecret')}
                  </span>
                  <code className='break-all'>{credentials.secret}</code>
                  <CopyButton value={credentials.secret} />
                </div>
              </div>
            ) : null}
          </DialogContent>
        </Dialog>

        <ConfirmDialog
          open={confirm !== null}
          onOpenChange={(open) => !open && setConfirm(null)}
          title={
            confirm?.kind === 'delete'
              ? t('wending.clients.deleteTitle', { app: confirm.client.name })
              : t('wending.clients.rotateTitle')
          }
          desc={
            confirm?.kind === 'delete'
              ? t('wending.clients.deleteDesc')
              : t('wending.clients.rotateDesc')
          }
          destructive
          isLoading={act.isPending}
          handleConfirm={() => confirm && act.mutate(confirm)}
        />
        <ConfirmDialog
          open={pendingGrant !== null}
          onOpenChange={(open) => !open && setPendingGrant(null)}
          title={t('wending.grants.revokeTitle', {
            app: pendingGrant?.client_name ?? '',
          })}
          desc={t('wending.grants.revokeDesc')}
          destructive
          confirmText={t('wending.grants.revoke')}
          isLoading={revokeGrant.isPending}
          handleConfirm={() =>
            pendingGrant && revokeGrant.mutate(pendingGrant.id)
          }
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
