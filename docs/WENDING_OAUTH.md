# 问鼎AI账号一键授权 (Wending OAuth)

A first-party OAuth 2.0 **authorization server** inside new-api. Our own products
(first: **WendingAI 创作台**, `https://huabu.wending.ai`; later others and an MCP
server) send a logged-in new-api user to a consent screen. When the user clicks
**授权**, new-api creates a **dedicated API key** (a normal `tokens` row, so
billing, quota, groups, model limits and logs behave exactly like any other key)
and hands it to the product through the **Authorization Code + PKCE (S256)** flow.

- Standards: RFC 6749 (authorization code), RFC 7636 (PKCE), RFC 9700 (OAuth 2.0 Security BCP).
- PKCE `S256` is **mandatory** for every client; `plain` is rejected.
- `redirect_uri` is an **exact string match** against the registered list.
- Codes: 32 random bytes (`wdc_` + base64url), stored only as SHA-256, single-use, 5-minute TTL.
- Client secrets: 32 random bytes (`wds_` + base64url), stored only as SHA-256, shown once.

## Flow

```
 User browser            huabu.wending.ai (server)                   new-api
 ────────────            ─────────────────────────                   ───────
 click "用问鼎AI账号登录/授权"
      ───────────────────► /api/auth/wending/start
                           create state + code_verifier,
                           store both in an httpOnly cookie
      ◄─────────────────── 302 → https://<newapi>/oauth/authorize?
                                 response_type=code&client_id=…&redirect_uri=…
                                 &state=…&code_challenge=…&code_challenge_method=S256&scope=api
 GET /oauth/authorize  ─────────────────────────────────────────────► SPA route
   (not logged in → /sign-in?redirect=… → back to /oauth/authorize)
   SPA: GET /api/wending/oauth/authorize?…   (validates client + redirect_uri + PKCE)
   consent screen: app name/icon, "请求使用你的额度", quota cap / unlimited, group
   click 授权 → POST /api/wending/oauth/authorize {approve:true, remain_quota|unlimited_quota, group}
              ← {redirect_to: redirect_uri?code=wdc_…&state=…}
      ───────────────────► /api/auth/wending/callback?code=…&state=…
                           check state == cookie
                           POST /api/oauth/token  ─────────────────────► verify client secret,
                             grant_type=authorization_code, code,          code (unused, unexpired,
                             redirect_uri, client_id, client_secret,       same client + redirect_uri),
                             code_verifier                                 PKCE S256
                           ◄──── {access_token:"sk-…", token_type:"Bearer", base_url, scope, user_id}
                           GET /api/oauth/userinfo (Bearer sk-…) ──────► who is connected + quota
                           store sk-… encrypted for this huabu user;
                           call https://<newapi>/v1/... with it
```

Deny → `redirect_uri?error=access_denied&state=…`. Request errors after the
client and redirect_uri are verified are returned to `redirect_uri` with
`error=invalid_request|invalid_scope|unsupported_response_type`; an unknown
client or unregistered redirect_uri is **never** redirected (error shown in new-api).

## Endpoints

| Method & path | Auth | Purpose |
| --- | --- | --- |
| `GET /oauth/authorize` | SPA, login required | Consent screen (frontend route) |
| `GET /api/wending/oauth/authorize` | dashboard session | Validate request, return client info, user groups, existing grant |
| `POST /api/wending/oauth/authorize` | dashboard **browser session only** (PATs rejected); `CriticalRateLimit` + per-user limit | Approve/deny; returns `redirect_to` |
| `POST /api/oauth/token` | public, `CriticalRateLimit`; `client_secret_post` or `client_secret_basic` | Exchange code → API key |
| `GET /api/oauth/userinfo` | `Authorization: Bearer sk-…` (OAuth-issued key only) | User id/name, group, user quota, token quota |
| `GET /api/wending/oauth/sidebar-apps` | session | Enabled clients with `show_in_sidebar` + non-empty `launch_url` (console left sidebar) |
| `GET /api/wending/oauth/grants` | session | My authorized apps |
| `DELETE /api/wending/oauth/grants/:id` | session | Revoke (deletes the key immediately) |
| `GET /api/wending/oauth/admin/grants?user_id=&include_revoked=true` | admin | Grants of all / one user |
| `DELETE /api/wending/oauth/admin/grants/:id` | admin | Revoke any grant |
| `GET/POST /api/wending/oauth/clients`, `PUT/DELETE /api/wending/oauth/clients/:id`, `POST /api/wending/oauth/clients/:id/secret` | **root** | Client registry ("接入应用"); create/rotate return the secret once |

CSRF: dashboard APIs authenticate with the `Authorization: Bearer <session access token>`
header (not cookies), so cross-site requests cannot ride the session.

Token endpoint response:

```json
{ "access_token": "sk-…", "token_type": "Bearer", "base_url": "https://<newapi>", "scope": "api", "user_id": 42 }
```

Errors use RFC 6749 §5.2: `400 {"error":"invalid_grant"|"invalid_request"|"unsupported_grant_type", "error_description":…}`,
`401 {"error":"invalid_client"}`. Responses carry `Cache-Control: no-store`.

`base_url` is the **服务器地址 (ServerAddress)** system setting – make sure it is
set to the public new-api URL in production (系统设置 → 站点/通用), otherwise it
falls back to `http://localhost:3000`.

## Grant & key policy

- One active grant per (user, client). The key is a normal token named `授权:<app name>`
  with the cap the user chose (or unlimited if the client allows it), the chosen/pinned
  group, and the client's model limits. It counts toward the user's max-token limit.
- **Re-authorizing an app that already has an active grant rotates the key**: the same
  `tokens` row is kept (usage history stays together) but gets a **new key** and the newly
  chosen limits; the previous key stops working at once. Rationale: re-auth usually means the
  app lost the key or the user wants different limits, and a stale key must not survive.
  Consequence: a product that runs on several devices should keep one key per user server-side.
- Revoking (user, admin, or deleting the client) marks the grant revoked, invalidates
  outstanding codes and soft-deletes the token (cache invalidated) → the key is rejected by
  `/v1/*` and `/api/oauth/userinfo` immediately.
- If the user deletes the key manually on the API-keys page, the next authorization creates a fresh grant.
- A reused code is rejected with `invalid_grant` and audited. (RFC 6749 suggests also revoking
  tokens issued from a replayed code; we deliberately do not, because the code is bound to the
  PKCE verifier + client secret and auto-revoke would turn a harmless double-submit into an outage.)

## Audit & logging

Audit-log entries (category `security`, visible in 审计日志): `wending_oauth.authorize`,
`wending_oauth.deny`, `wending_oauth.token_exchange` (success and failure with reason),
`wending_oauth.revoke`. Root client CRUD is audited by the existing admin audit middleware.
No key, code, verifier or secret is ever logged (`/api/oauth/token` takes them in the POST body).

## Registering a client (root)

1. Console → 管理员 → **接入应用** → 新建应用.
2. Name `WendingAI 创作台`, icon/homepage URLs, redirect URI(s), one per line, e.g.
   `https://huabu.wending.ai/api/auth/wending/callback` (https required; `http://localhost…`
   allowed for development; reverse-domain custom schemes like `ai.wending.mcp:/cb` for native/MCP).
3. Confidential (default) for server-side apps; tick **公共客户端** only for apps that cannot keep a
   secret (CLI/MCP/desktop) – they still must use PKCE.
4. Default cap, whether unlimited is allowed, optional pinned group, optional model whitelist.
5. Optional: enable **显示在侧边栏**, set **侧边栏启动地址** (RP login-start URL) and sort order.
6. Save → copy **Client ID** and **Client Secret** (shown once; use 重置密钥 to rotate).

## Sidebar launch ("问鼎应用")

Registered clients can appear in the logged-in console **left sidebar** under **问鼎应用 / Wending Apps**.

Admin fields on each client (接入应用):

- `show_in_sidebar` – when true (and `enabled`), list the app for every logged-in user
- `launch_url` – absolute `http(s)` URL opened in a **new tab** on click (required when sidebar is on). This must be the **RP's own login-start URL** (e.g. `https://huabu.wending.ai/api/auth/wending/start` or the app homepage that kicks off login). The authorization server never initiates OAuth itself: the RP must create `state` + PKCE `code_verifier`/`code_challenge` and redirect the browser to `/oauth/authorize`.
- `sidebar_order` – ascending sort (ties broken by id)

`GET /api/wending/oauth/sidebar-apps` returns `{ client_id, name, description, icon_url, homepage_url, launch_url, sidebar_order }` for matching clients. The frontend builds the sidebar group from that list.

## Integration example (huabu, Next.js App Router, server-side exchange)

```ts
// lib/wending-oauth.ts
import crypto from 'node:crypto'

const NEWAPI = process.env.WENDING_NEWAPI_URL!            // e.g. https://<new-api domain>
const CLIENT_ID = process.env.WENDING_CLIENT_ID!
const CLIENT_SECRET = process.env.WENDING_CLIENT_SECRET!  // server only, never shipped to the browser
const REDIRECT_URI = 'https://huabu.wending.ai/api/auth/wending/callback'

const b64url = (buf: Buffer) => buf.toString('base64url')

export function createAuthorizeRequest() {
  const state = b64url(crypto.randomBytes(24))
  const verifier = b64url(crypto.randomBytes(48))           // 64 chars, RFC 7636 charset
  const challenge = b64url(crypto.createHash('sha256').update(verifier).digest())
  const url = new URL('/oauth/authorize', NEWAPI)
  url.search = new URLSearchParams({
    response_type: 'code',
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    state,
    code_challenge: challenge,
    code_challenge_method: 'S256',
    scope: 'api',
  }).toString()
  return { url: url.toString(), state, verifier }
}

export async function exchangeCode(code: string, verifier: string) {
  const res = await fetch(new URL('/api/oauth/token', NEWAPI), {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      redirect_uri: REDIRECT_URI,
      client_id: CLIENT_ID,
      client_secret: CLIENT_SECRET,
      code_verifier: verifier,
    }),
    cache: 'no-store',
  })
  const body = await res.json()
  if (!res.ok) throw new Error(`${body.error}: ${body.error_description}`)
  return body as { access_token: string; token_type: 'Bearer'; base_url: string; scope: string; user_id: number }
}

export async function fetchUserInfo(apiKey: string) {
  const res = await fetch(new URL('/api/oauth/userinfo', NEWAPI), {
    headers: { Authorization: `Bearer ${apiKey}` },
    cache: 'no-store',
  })
  if (res.status === 401) return null                         // revoked → ask the user to re-authorize
  return res.json()
}
```

```ts
// app/api/auth/wending/start/route.ts
import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'
import { createAuthorizeRequest } from '@/lib/wending-oauth'

export async function GET() {
  const { url, state, verifier } = createAuthorizeRequest()
  ;(await cookies()).set('wd_oauth', JSON.stringify({ state, verifier }), {
    httpOnly: true, secure: true, sameSite: 'lax', path: '/api/auth/wending', maxAge: 600,
  })
  return NextResponse.redirect(url)
}

// app/api/auth/wending/callback/route.ts
import { cookies } from 'next/headers'
import { NextResponse, type NextRequest } from 'next/server'
import { exchangeCode, fetchUserInfo } from '@/lib/wending-oauth'

export async function GET(req: NextRequest) {
  const jar = await cookies()
  const saved = JSON.parse(jar.get('wd_oauth')?.value ?? '{}')
  jar.delete('wd_oauth')
  const params = req.nextUrl.searchParams
  if (params.get('error')) return NextResponse.redirect(new URL('/?wending=denied', req.url))
  if (!saved.state || params.get('state') !== saved.state) {
    return new NextResponse('state mismatch', { status: 400 })
  }
  const token = await exchangeCode(params.get('code')!, saved.verifier)
  const who = await fetchUserInfo(token.access_token)
  // Persist token.access_token ENCRYPTED, linked to the current huabu user, together with
  // who.user_id / who.username. Call `${token.base_url}/v1/...` with it from the server.
  // await db.wendingLink.upsert({ ... })
  return NextResponse.redirect(new URL('/?wending=connected', req.url))
}
```

Treat a `401` from `/v1/*` or `/api/oauth/userinfo` as "authorization revoked" and offer re-authorization.

## Production deploy (manual – not performed by the agent)

Production runs the stock v1.0.0-rc.45 binary at `/opt/new-api/new-api` under systemd unit `new-api`
with PostgreSQL. The `wending` branch only **adds** three tables; nothing existing is altered.

```bash
# 0) copy the artifact to the server and verify it
scp new-api-wending-rc45-<sha> server:/tmp/
ssh server
sha256sum /tmp/new-api-wending-rc45-<sha>          # must match the .sha256 file
/tmp/new-api-wending-rc45-<sha> --version

# 1) backups
TS=$(date +%Y%m%d-%H%M%S)
sudo cp -a /opt/new-api/new-api /opt/new-api/new-api.bak-$TS
pg_dump "$SQL_DSN" -Fc -f /root/newapi-$TS.dump      # use the DSN from the service env file
ls -la /root/newapi-$TS.dump

# 2) replace + restart (brief downtime)
sudo install -m 0755 /tmp/new-api-wending-rc45-<sha> /opt/new-api/new-api
sudo systemctl restart new-api
sudo systemctl status new-api --no-pager
sudo journalctl -u new-api -n 100 --no-pager           # look for migration errors

# 3) verify
curl -fsS https://<newapi-domain>/api/status | head -c 200; echo
curl -s -o /dev/null -w '%{http_code}\n' https://<newapi-domain>/oauth/authorize   # 200 (SPA)
curl -s -X POST https://<newapi-domain>/api/oauth/token -d grant_type=authorization_code   # 401 invalid_client (expected)
# existing keys still work:
curl -s https://<newapi-domain>/v1/models -H "Authorization: Bearer <an existing key>" | head -c 200
# then: set 系统设置 → 服务器地址 to the public URL, register the huabu client in 接入应用,
# and run one real authorization from huabu.

# Rollback
sudo install -m 0755 /opt/new-api/new-api.bak-$TS /opt/new-api/new-api
sudo systemctl restart new-api
# The wending_oauth_* tables are ignored by the stock binary (verified locally on PostgreSQL 16);
# issued OAuth keys keep working as normal keys – delete them on the keys page if needed.
# Restore the DB dump only if something else went wrong: pg_restore --clean -d "$SQL_DSN" /root/newapi-$TS.dump
```
