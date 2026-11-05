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
import { api } from '@/lib/api'

import type {
  WendingApiResponse,
  WendingAuthorizeInfo,
  WendingAuthorizeParams,
  WendingOAuthClient,
  WendingOAuthClientInput,
  WendingOAuthGrantView,
} from './types'

const BASE = '/api/wending/oauth'
const quiet = { skipErrorHandler: true, skipBusinessError: true }

export async function getAuthorizeInfo(params: WendingAuthorizeParams) {
  const res = await api.get<WendingApiResponse<WendingAuthorizeInfo>>(
    `${BASE}/authorize`,
    { params, ...quiet }
  )
  return res.data
}

export async function submitAuthorize(
  body: WendingAuthorizeParams & {
    approve: boolean
    unlimited_quota: boolean
    remain_quota: number
    group: string
  }
) {
  const res = await api.post<
    WendingApiResponse<{ redirect_to: string; rotated?: boolean }>
  >(`${BASE}/authorize`, body, quiet)
  return res.data
}

export async function listMyGrants() {
  const res = await api.get<WendingApiResponse<WendingOAuthGrantView[]>>(
    `${BASE}/grants`
  )
  return res.data
}

export async function revokeMyGrant(id: number) {
  const res = await api.delete<WendingApiResponse<null>>(`${BASE}/grants/${id}`)
  return res.data
}

export async function listAdminGrants(userId?: number) {
  const res = await api.get<WendingApiResponse<WendingOAuthGrantView[]>>(
    `${BASE}/admin/grants`,
    { params: userId ? { user_id: userId } : {} }
  )
  return res.data
}

export async function revokeAdminGrant(id: number) {
  const res = await api.delete<WendingApiResponse<null>>(
    `${BASE}/admin/grants/${id}`
  )
  return res.data
}

export async function listClients() {
  const res = await api.get<WendingApiResponse<WendingOAuthClient[]>>(
    `${BASE}/clients`
  )
  return res.data
}

export async function createClient(input: WendingOAuthClientInput) {
  const res = await api.post<
    WendingApiResponse<{ client: WendingOAuthClient; client_secret: string }>
  >(`${BASE}/clients`, input)
  return res.data
}

export async function updateClient(id: number, input: WendingOAuthClientInput) {
  const res = await api.put<WendingApiResponse<WendingOAuthClient>>(
    `${BASE}/clients/${id}`,
    input
  )
  return res.data
}

export async function rotateClientSecret(id: number) {
  const res = await api.post<WendingApiResponse<{ client_secret: string }>>(
    `${BASE}/clients/${id}/secret`
  )
  return res.data
}

export async function deleteClient(id: number) {
  const res = await api.delete<WendingApiResponse<null>>(
    `${BASE}/clients/${id}`
  )
  return res.data
}
