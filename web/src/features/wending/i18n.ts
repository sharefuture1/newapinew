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
// Wending fork-local translations. Registered at runtime so the upstream
// locale JSON files stay untouched (see WENDING_PATCHES.md).
import i18n from 'i18next'

const en = {
  nav: {
    authorizedApps: 'Authorized Apps',
    oauthClients: 'OAuth Apps',
    apps: 'Wending Apps',
  },
  oauth: {
    brand: 'Wending AI account authorization',
    requestTitle: '{{app}} wants to connect to your account',
    requestDesc:
      '{{app}} is requesting to use your quota. A dedicated API key will be created for it and every call it makes is billed to your balance.',
    signedInAs: 'Signed in as {{name}}',
    permissions: 'This app will be able to:',
    permUseQuota: 'Call models with a dedicated API key billed to your quota',
    permProfile: 'Read your username and balance summary',
    quotaCap: 'Spending limit for this app',
    unlimited: 'Unlimited (uses your whole balance)',
    limited: 'Limit to',
    group: 'Group',
    groupDefault: 'My default group ({{group}})',
    groupFixed: 'This app uses group {{group}}',
    modelLimits: 'Allowed models: {{models}}',
    existingGrant:
      'You already authorized this app. Approving again issues a new key and the previous key stops working.',
    allow: 'Authorize',
    deny: 'Cancel',
    redirecting: 'Redirecting back to the app…',
    invalidRequest: 'Invalid authorization request',
    invalidRequestDesc:
      'This authorization link is invalid or the app is not registered. Please return to the app and try again.',
    redirectTo: 'You will be redirected to {{host}}',
    quotaRequired: 'Please enter a positive spending limit',
  },
  grants: {
    title: 'Authorized Apps',
    desc: 'Apps you have connected with your Wending AI account. Revoking an app deletes its API key immediately.',
    empty: 'You have not authorized any apps yet.',
    authorizedAt: 'Authorized {{time}}',
    lastUsed: 'Last connected {{time}}',
    used: 'Used {{amount}}',
    remaining: 'Remaining limit {{amount}}',
    unlimited: 'Unlimited',
    revoke: 'Revoke',
    revokeTitle: 'Revoke {{app}}?',
    revokeDesc:
      'The app will lose access immediately and its API key will be deleted.',
    revoked: 'Authorization revoked',
    keyMissing: 'Key deleted',
  },
  clients: {
    title: 'OAuth Apps',
    desc: 'Register first-party products that can request a user API key via "Wending AI account authorization".',
    create: 'New app',
    edit: 'Edit app',
    name: 'Name',
    description: 'Description',
    iconUrl: 'Icon URL',
    homepageUrl: 'Homepage URL',
    launchUrl: 'Sidebar launch URL (RP login start)',
    showInSidebar: 'Show in sidebar',
    sidebarOrder: 'Sidebar sort order',
    redirectUris: 'Redirect URIs (one per line, exact match)',
    isPublic: 'Public client (PKCE only, no secret)',
    enabled: 'Enabled',
    defaultQuota: 'Default spending limit',
    allowUnlimited: 'Allow users to choose unlimited quota',
    defaultUnlimited: 'Default to unlimited',
    allowedGroup: 'Pinned group (empty = user choice)',
    modelLimitsEnabled: 'Restrict models',
    modelLimits: 'Allowed models (comma separated)',
    save: 'Save',
    rotateSecret: 'Rotate secret',
    rotateTitle: 'Rotate client secret?',
    rotateDesc:
      'The current secret stops working immediately. Update the app configuration with the new secret.',
    delete: 'Delete',
    deleteTitle: 'Delete {{app}}?',
    deleteDesc:
      'All user authorizations for this app will be revoked and their API keys deleted.',
    secretTitle: 'Client credentials',
    secretDesc:
      'Copy the client secret now. It is stored hashed and will not be shown again.',
    clientId: 'Client ID',
    clientSecret: 'Client secret',
    empty: 'No apps registered yet.',
    public: 'Public',
    confidential: 'Confidential',
    disabled: 'Disabled',
    grantsTitle: 'User authorizations',
    grantsUserId: 'Filter by user ID',
    grantsUser: 'User #{{id}}',
    saved: 'Saved',
  },
}

const zhCN: typeof en = {
  nav: {
    authorizedApps: '已授权应用',
    oauthClients: '接入应用',
    apps: '问鼎应用',
  },
  oauth: {
    brand: '问鼎AI账号一键授权',
    requestTitle: '{{app}} 请求连接你的账号',
    requestDesc:
      '{{app}} 请求使用你的额度。授权后将为其创建一个专用 API 密钥，其所有调用均从你的余额中扣费。',
    signedInAs: '当前登录：{{name}}',
    permissions: '该应用将可以：',
    permUseQuota: '使用专用 API 密钥调用模型，消耗你的额度',
    permProfile: '读取你的用户名与余额概要',
    quotaCap: '该应用的额度上限',
    unlimited: '不限额度（可使用全部余额）',
    limited: '限额',
    group: '分组',
    groupDefault: '我的默认分组（{{group}}）',
    groupFixed: '该应用使用分组 {{group}}',
    modelLimits: '可用模型：{{models}}',
    existingGrant:
      '你已授权过此应用。再次授权将签发新密钥，之前的密钥会立即失效。',
    allow: '授权',
    deny: '取消',
    redirecting: '正在返回应用…',
    invalidRequest: '授权请求无效',
    invalidRequestDesc: '授权链接无效或应用未注册，请返回应用重试。',
    redirectTo: '授权后将跳转到 {{host}}',
    quotaRequired: '请输入大于 0 的额度上限',
  },
  grants: {
    title: '已授权应用',
    desc: '你使用问鼎AI账号授权过的应用。撤销授权会立即删除对应的 API 密钥。',
    empty: '你还没有授权任何应用。',
    authorizedAt: '授权于 {{time}}',
    lastUsed: '最近连接 {{time}}',
    used: '已用 {{amount}}',
    remaining: '剩余额度 {{amount}}',
    unlimited: '不限额度',
    revoke: '撤销授权',
    revokeTitle: '撤销 {{app}} 的授权？',
    revokeDesc: '该应用将立即失去访问权限，其 API 密钥会被删除。',
    revoked: '已撤销授权',
    keyMissing: '密钥已删除',
  },
  clients: {
    title: '接入应用',
    desc: '注册可通过“问鼎AI账号一键授权”获取用户 API 密钥的自有产品。',
    create: '新建应用',
    edit: '编辑应用',
    name: '名称',
    description: '描述',
    iconUrl: '图标 URL',
    homepageUrl: '主页 URL',
    launchUrl: '侧边栏启动地址（应用发起登录的 URL）',
    showInSidebar: '显示在侧边栏',
    sidebarOrder: '侧边栏排序',
    redirectUris: '回调地址（每行一个，精确匹配）',
    isPublic: '公共客户端（仅 PKCE，无密钥）',
    enabled: '启用',
    defaultQuota: '默认额度上限',
    allowUnlimited: '允许用户选择不限额度',
    defaultUnlimited: '默认不限额度',
    allowedGroup: '固定分组（留空则由用户选择）',
    modelLimitsEnabled: '限制模型',
    modelLimits: '可用模型（逗号分隔）',
    save: '保存',
    rotateSecret: '重置密钥',
    rotateTitle: '重置客户端密钥？',
    rotateDesc: '当前密钥将立即失效，请用新密钥更新应用配置。',
    delete: '删除',
    deleteTitle: '删除 {{app}}？',
    deleteDesc: '该应用的所有用户授权都将被撤销，对应 API 密钥会被删除。',
    secretTitle: '客户端凭据',
    secretDesc: '请立即复制客户端密钥。密钥以哈希形式存储，之后不会再次显示。',
    clientId: 'Client ID',
    clientSecret: 'Client Secret',
    empty: '尚未注册任何应用。',
    public: '公共',
    confidential: '机密',
    disabled: '已禁用',
    grantsTitle: '用户授权记录',
    grantsUserId: '按用户 ID 筛选',
    grantsUser: '用户 #{{id}}',
    saved: '已保存',
  },
}

const zhTW: typeof en = {
  ...zhCN,
  nav: { authorizedApps: '已授權應用', oauthClients: '接入應用', apps: '問鼎應用' },
  oauth: {
    ...zhCN.oauth,
    brand: '問鼎AI帳號一鍵授權',
    requestTitle: '{{app}} 請求連接你的帳號',
    allow: '授權',
  },
  grants: { ...zhCN.grants, title: '已授權應用', revoke: '撤銷授權' },
}

const bundles: Record<string, typeof en> = { en, zhCN, zhTW }

// Unit tests mock i18next with a partial object; skip registration there.
if (typeof i18n.addResourceBundle === 'function') {
  for (const [lng, wending] of Object.entries(bundles)) {
    i18n.addResourceBundle(lng, 'translation', { wending }, true, false)
  }
}

export {}
