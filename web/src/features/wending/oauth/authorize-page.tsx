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
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { getCurrencyLabel } from '@/lib/currency'
import {
  formatQuota,
  getEditableQuotaStep,
  parseQuotaFromDollars,
  quotaUnitsToEditableAmount,
} from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import '../i18n'
import { getAuthorizeInfo, submitAuthorize } from './api'
import type { WendingAuthorizeInfo, WendingAuthorizeParams } from './types'

function redirectHost(uri: string): string {
  try {
    return new URL(uri).host || uri
  } catch {
    return uri
  }
}

export function WendingAuthorizePage({
  params,
}: {
  params: WendingAuthorizeParams
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['wending-oauth-authorize', params],
    queryFn: () => getAuthorizeInfo(params),
    retry: false,
    staleTime: Infinity,
  })

  if (query.isLoading) {
    return (
      <Shell>
        <p className='text-muted-foreground text-center text-sm'>…</p>
      </Shell>
    )
  }
  const res = query.data
  if (!res?.success || !res.data) {
    const errorRedirect = res?.redirect_to
    return (
      <Shell>
        <Card>
          <CardHeader>
            <CardTitle>{t('wending.oauth.invalidRequest')}</CardTitle>
            <CardDescription>
              {t('wending.oauth.invalidRequestDesc')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className='text-destructive text-sm break-all'>
              {res?.message || String(query.error ?? '')}
            </p>
          </CardContent>
          {res?.redirect_trusted && errorRedirect ? (
            <CardFooter>
              <Button
                variant='outline'
                onClick={() => window.location.assign(errorRedirect)}
              >
                {t('wending.oauth.deny')}
              </Button>
            </CardFooter>
          ) : null}
        </Card>
      </Shell>
    )
  }
  return <ConsentCard params={params} info={res.data} />
}

function Shell({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation()
  return (
    <div className='bg-muted/30 flex min-h-svh items-center justify-center p-4'>
      <div className='w-full max-w-md space-y-4'>
        <p className='text-muted-foreground text-center text-sm font-medium'>
          {t('wending.oauth.brand')}
        </p>
        {children}
      </div>
    </div>
  )
}

function ConsentCard({
  params,
  info,
}: {
  params: WendingAuthorizeParams
  info: WendingAuthorizeInfo
}) {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.auth.user)
  const client = info.client
  const [unlimited, setUnlimited] = useState(
    client.allow_unlimited_quota && client.default_unlimited_quota
  )
  const [amount, setAmount] = useState(() =>
    String(
      quotaUnitsToEditableAmount(
        client.default_remain_quota > 0
          ? client.default_remain_quota
          : info.quota_per_unit * 10
      )
    )
  )
  const [group, setGroup] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function decide(approve: boolean) {
    setError('')
    const remainQuota = parseQuotaFromDollars(Number(amount))
    if (approve && !unlimited && !(remainQuota > 0)) {
      setError(t('wending.oauth.quotaRequired'))
      return
    }
    setSubmitting(true)
    try {
      const res = await submitAuthorize({
        ...params,
        approve,
        unlimited_quota: unlimited,
        remain_quota: unlimited ? 0 : remainQuota,
        group,
      })
      const target = res.data?.redirect_to ?? res.redirect_to
      if (target) {
        window.location.assign(target)
        return
      }
      setError(res.message || t('wending.oauth.invalidRequest'))
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
    setSubmitting(false)
  }

  const userName = user?.display_name || user?.username || ''
  return (
    <Shell>
      <Card>
        <CardHeader className='items-center text-center'>
          {client.icon_url ? (
            <img
              src={client.icon_url}
              alt=''
              className='mx-auto mb-2 size-14 rounded-xl object-cover'
              referrerPolicy='no-referrer'
            />
          ) : (
            <div className='bg-primary text-primary-foreground mx-auto mb-2 flex size-14 items-center justify-center rounded-xl text-2xl font-semibold'>
              {client.name.slice(0, 1)}
            </div>
          )}
          <CardTitle>
            {t('wending.oauth.requestTitle', { app: client.name })}
          </CardTitle>
          <CardDescription>
            {t('wending.oauth.requestDesc', { app: client.name })}
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4 text-sm'>
          {userName ? (
            <p className='text-muted-foreground'>
              {t('wending.oauth.signedInAs', { name: userName })}
            </p>
          ) : null}
          <div>
            <p className='mb-1 font-medium'>{t('wending.oauth.permissions')}</p>
            <ul className='text-muted-foreground list-disc space-y-1 pl-5'>
              <li>{t('wending.oauth.permUseQuota')}</li>
              <li>{t('wending.oauth.permProfile')}</li>
              {client.model_limits_enabled && client.model_limits ? (
                <li>
                  {t('wending.oauth.modelLimits', {
                    models: client.model_limits,
                  })}
                </li>
              ) : null}
            </ul>
          </div>

          <fieldset className='space-y-2'>
            <legend className='mb-1 font-medium'>
              {t('wending.oauth.quotaCap')}
            </legend>
            {client.allow_unlimited_quota ? (
              <label className='flex items-center gap-2'>
                <input
                  type='radio'
                  name='quota-mode'
                  checked={unlimited}
                  onChange={() => setUnlimited(true)}
                />
                {t('wending.oauth.unlimited')}
              </label>
            ) : null}
            <label className='flex items-center gap-2'>
              <input
                type='radio'
                name='quota-mode'
                checked={!unlimited}
                onChange={() => setUnlimited(false)}
              />
              <span className='shrink-0'>{t('wending.oauth.limited')}</span>
              <Input
                type='number'
                min={0}
                step={getEditableQuotaStep()}
                value={amount}
                disabled={unlimited}
                onChange={(e) => setAmount(e.target.value)}
                className='h-8 w-32'
              />
              <span className='text-muted-foreground'>
                {getCurrencyLabel()}
              </span>
            </label>
            {!unlimited && Number(amount) > 0 ? (
              <p className='text-muted-foreground text-xs'>
                ≈ {formatQuota(parseQuotaFromDollars(Number(amount)))}
              </p>
            ) : null}
          </fieldset>

          <div className='space-y-1'>
            <p className='font-medium'>{t('wending.oauth.group')}</p>
            {client.allowed_group ? (
              <p className='text-muted-foreground'>
                {t('wending.oauth.groupFixed', { group: client.allowed_group })}
              </p>
            ) : (
              <NativeSelect
                className='w-full'
                value={group}
                onChange={(e) => setGroup(e.target.value)}
              >
                <NativeSelectOption value=''>
                  {t('wending.oauth.groupDefault', { group: info.user_group })}
                </NativeSelectOption>
                {info.groups
                  .filter((g) => g.name !== info.user_group)
                  .map((g) => (
                    <NativeSelectOption key={g.name} value={g.name}>
                      {g.name}
                      {g.desc ? ` — ${g.desc}` : ''}
                    </NativeSelectOption>
                  ))}
              </NativeSelect>
            )}
          </div>

          {info.existing_grant ? (
            <p className='rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-xs'>
              {t('wending.oauth.existingGrant')}
            </p>
          ) : null}
          {error ? <p className='text-destructive text-sm'>{error}</p> : null}
          <p className='text-muted-foreground text-xs break-all'>
            {t('wending.oauth.redirectTo', {
              host: redirectHost(params.redirect_uri),
            })}
          </p>
        </CardContent>
        <CardFooter className='flex justify-end gap-2'>
          <Button
            variant='outline'
            disabled={submitting}
            onClick={() => decide(false)}
          >
            {t('wending.oauth.deny')}
          </Button>
          <Button disabled={submitting} onClick={() => decide(true)}>
            {submitting
              ? t('wending.oauth.redirecting')
              : t('wending.oauth.allow')}
          </Button>
        </CardFooter>
      </Card>
    </Shell>
  )
}
