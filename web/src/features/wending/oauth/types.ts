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
export interface WendingOAuthClientPublic {
  client_id: string
  name: string
  description: string
  icon_url: string
  homepage_url: string
  default_unlimited_quota: boolean
  default_remain_quota: number
  allow_unlimited_quota: boolean
  allowed_group: string
  model_limits_enabled: boolean
  model_limits: string
}

export interface WendingAuthorizeInfo {
  client: WendingOAuthClientPublic
  scope: string
  user_group: string
  groups: { name: string; desc: string }[]
  existing_grant: { id: number; created_at: number; scope: string } | null
  quota_per_unit: number
  max_quota: number
}

export interface WendingAuthorizeParams {
  response_type: string
  client_id: string
  redirect_uri: string
  state: string
  scope: string
  code_challenge: string
  code_challenge_method: string
}

export interface WendingApiResponse<T> {
  success: boolean
  message?: string
  data?: T
  oauth_error?: string
  redirect_trusted?: boolean
  redirect_to?: string
}

export interface WendingOAuthGrantView {
  id: number
  user_id: number
  client_id: string
  token_id: number
  scope: string
  created_at: number
  updated_at: number
  last_used_at: number
  revoked_at: number
  client_name: string
  client_icon_url: string
  client_homepage_url: string
  token_name: string
  token_status: number
  token_exists: boolean
  unlimited_quota: boolean
  remain_quota: number
  used_quota: number
  group: string
}

export interface WendingOAuthSidebarApp {
  client_id: string
  name: string
  description: string
  icon_url: string
  homepage_url: string
  launch_url: string
  sidebar_order: number
}

export interface WendingOAuthClient {
  id: number
  client_id: string
  name: string
  description: string
  icon_url: string
  homepage_url: string
  launch_url: string
  show_in_sidebar: boolean
  sidebar_order: number
  redirect_uris: string[]
  is_public: boolean
  enabled: boolean
  default_unlimited_quota: boolean
  default_remain_quota: number
  allow_unlimited_quota: boolean
  allowed_group: string
  model_limits_enabled: boolean
  model_limits: string
  has_secret: boolean
  created_at: number
  updated_at: number
}

export type WendingOAuthClientInput = Omit<
  WendingOAuthClient,
  'id' | 'client_id' | 'has_secret' | 'created_at' | 'updated_at'
>
