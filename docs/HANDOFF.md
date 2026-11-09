# HANDOFF – 问鼎AI账号一键授权 (Wending OAuth)

_Last updated: 2026-11-05 08:30 EST (America/New_York)_

## What was done

- Branch **`wending`** created from tag **v1.0.0-rc.45** (production version) in
  `sharefuture1/newapinew`. Feature implemented as small `feat(wending): …` commits.
- First-party OAuth 2.0 authorization server (Authorization Code + mandatory PKCE S256):
  client registry (root), consent screen `/oauth/authorize`, approve/deny API,
  `POST /api/oauth/token`, `GET /api/oauth/userinfo`, user "已授权应用" page with revoke,
  admin grant view/revoke, audit-log entries. Details: [WENDING_OAUTH.md](WENDING_OAUTH.md).
- **Sidebar launch ("问鼎应用")**: each client can set `show_in_sidebar`, `launch_url`,
  `sidebar_order`. Logged-in users see enabled apps in the left sidebar; click opens
  `launch_url` in a new tab (RP then initiates OAuth with PKCE/state). Admin toggles
  these on the 接入应用 page. Endpoint: `GET /api/wending/oauth/sidebar-apps`.
- Upstream footprint kept minimal; every touched upstream file is listed in
  [../WENDING_PATCHES.md](../WENDING_PATCHES.md).
- Throwaway branch **`wending-rc48-test`**: same commits rebased onto v1.0.0-rc.48 (one trivial
  import conflict in `web/src/hooks/use-sidebar-data.ts`, resolved). Builds and passes tests/e2e.

## Why huabu was not in the sidebar (2026-11-05)

1. **No clients registered** – prod `wending_oauth_clients` had 0 rows after the first deploy.
2. **No launch mechanism** – sidebar only had static entries ("已授权应用", "接入应用");
   registered OAuth clients were never rendered as launch links.
3. OAuth **must be RP-initiated** (state + PKCE). The AS cannot safely start auth from a
   bare `/oauth/authorize` link without the RP holding verifier/state; hence sidebar
   opens the RP `launch_url` instead of jumping straight to consent.

## Status

| Check | Result |
| --- | --- |
| `go build`, `go vet` (controller/model/router) | pass |
| `go test ./controller ./model ./router` | pass (incl. `TestWendingOAuthSidebarApps`) |
| `cd web && bun run build`, `tsgo -b` | pass |
| `cd web && bun run test --run src/hooks/__tests__/sidebar-config.test.tsx` | 13/13 pass |
| E2E on built binary, SQLite fresh DB | pass – `docs/wending-oauth-e2e.sh` (incl. sidebar-apps step 3b) |
| Production deploy (sidebar launch) | see record below |

## Key paths

- Backend: `model/wending_oauth.go`, `controller/wending_oauth.go`, `controller/wending_oauth_test.go`, `router/wending-oauth-router.go`
- Frontend: `web/src/features/wending/**`, `web/src/routes/oauth/authorize.tsx`, `web/src/routes/_authenticated/wending/*`
- Docs: `docs/WENDING_OAUTH.md`, `WENDING_PATCHES.md`
- E2E script: `docs/wending-oauth-e2e.sh` (`BASE=http://localhost:PORT ./docs/wending-oauth-e2e.sh`)

## Open items

1. **Huabu RP integration – LIVE (2026-11-10 04:05 UTC+7).** Repo `sharefuture1/infinite-canvas-private`,
   branch `codex/wending-brand-and-help` (merged from `wending-oauth`), Vercel project `web`,
   production `dpl_5YBYoUYT7DScWi9tHp2xQs8185F5` (rollback: `dpl_543g8Yg9LuZnfC92fng9bX7KBEgs`).
   `/api/auth/wending/{start,callback,session}` are Vercel functions; env `WENDING_OAUTH_CLIENT_ID`,
   `WENDING_OAUTH_CLIENT_SECRET` (sensitive), `WENDING_OAUTH_BASE_URL`, `WENDING_OAUTH_REDIRECT_URI`.
   Details: that repo's `docs/HANDOFF.md`. Real-user end-to-end login still to be confirmed.
2. Set 系统设置 → **服务器地址** to `https://api.wending.ai` if not already (token `base_url`).
3. MCP server: register as a **public** client with a loopback or reverse-domain redirect URI.
4. When upgrading production to rc.48+: rebase `wending` (see WENDING_PATCHES.md), rebuild web.
5. Optional hardening: per-client consent "remember" UX, refresh/rotation API, more i18n locales.

## Production deployment record — 2026-11-05 (initial OAuth)

- Server: vultr-sv (104.238.180.145), https://api.wending.ai, service `new-api`
- Deployed: 2026-11-05 13:14 UTC (08:14 America/New_York); binary built from `5c0cbe3`, reports `v1.0.0-rc.45-wending-5c0cbe3`
- sha256: `3a19e964f4e1a5d04f3fbab5994890f23f65b93a081e26653396b61d4789e34b`
- Previous binary (v1.0.0-rc.45): `/opt/new-api/new-api.bak-20261105-131408`
- DB backup: `/opt/new-api/db-backup-20261105-131408-pre-wending.dump`
- Auto-migrated tables: wending_oauth_clients, wending_oauth_grants, wending_oauth_codes

## Production deployment record — 2026-11-05 (sidebar launch)

- Server: vultr-sv, https://api.wending.ai, service `new-api`
- Deployed: 2026-11-05 13:25–13:26 UTC (08:25–08:26 America/New_York); ~20s restart
  (stop→healthy). Binary `2c8eae5`, reports `v1.0.0-rc.45-wending-2c8eae5`
- sha256: `36e8de3d7111461f75c69b95038c5a5b0affe03b894cf73c0ad98bc25f6bc4bc`
- Previous binary (5c0cbe3): `/opt/new-api/new-api.bak-20261105-132552`
- DB backup: `/opt/new-api/db-backup-20261105-132552-pre-sidebar.dump`
- AutoMigrate added columns: `launch_url`, `show_in_sidebar`, `sidebar_order`
- Verified: `/api/status` success + version, `/oauth/authorize` 200,
  `/api/oauth/token` bogus → 401 invalid_client, `/api/pricing` anon 401,
  `/api/wending/oauth/sidebar-apps` anon 401, logs clean
- **Huabu client** registered:
  - client_id: `wd_IHtyFqxBDKBrOJIkoS5Gh8FQ`
  - name: `WendingAI 创作台`
  - redirect: `https://huabu.wending.ai/api/auth/wending/callback`
  - launch_url: `https://huabu.wending.ai/?wending_auth=1` (changed 2026-11-09 21:04 UTC; was `https://huabu.wending.ai/`;
    backup/rollback SQL: `/opt/new-api/wending_oauth_clients-huabu-20261109-210422.{backup,rollback}.sql`; no restart needed,
    sidebar-apps reads the DB on every request)
  - show_in_sidebar: true, sidebar_order: 10
  - secret file (root only, chmod 600): `/opt/new-api/oauth-clients/huabu.secret`
  - Sidebar list for a logged-in user was not live-checked (no prod user credentials used);
    DB row matches the list filter (`enabled` + `show_in_sidebar` + non-empty `launch_url`)

Rollback:
```
install -m 0755 /opt/new-api/new-api.bak-20261105-132552 /opt/new-api/new-api && systemctl restart new-api
# additive columns can stay; restore dump only if needed:
#  pg_restore --clean -d "$SQL_DSN" /opt/new-api/db-backup-20261105-132552-pre-sidebar.dump
```

