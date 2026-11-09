# HANDOFF – 问鼎AI账号一键授权 (Wending OAuth)

_Last updated: 2026-11-10 (UTC+7) – added the sign-in (`login`) scope for the shop, **not deployed**_

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

## Sign-in scope for the shop (branch `wending-shop-login`) – LIVE 2026-11-10 05:58 UTC+7 (see record below)

Status 2026-11-10 (UTC+7): code + tests done, PR open against `wending`. **Not deployed; nothing
written to the production DB.** Deploying new-api is owned by the bot that runs vultr-sv.

What it adds (details: [WENDING_OAUTH.md § Sign-in scope](WENDING_OAUTH.md)):
- scopes `login` / `email`; login issues a 10-minute `wdat_` token, never an `sk-` key
- userinfo returns id, username, display name, e-mail (`email_verified: false`)
- `trusted` client flag → login-only requests skip the consent screen
- DB (AutoMigrate, additive only): new table `wending_oauth_access_tokens`, new column
  `wending_oauth_clients.trusted` (default false). Existing clients (huabu, lrs) behave exactly as before.

Checks run: `go test ./controller ./model` pass; `bun run typecheck` pass; `bun run test`
199 files / 2368 tests pass; `bun run build` pass. (`bun run lint` fails only on pre-existing
`scripts/sync-i18n.mjs` issues; `src/features/wending` is clean.)

Deploy steps (vultr-sv, same procedure as the records below):
1. Merge the PR into `wending`; on the build box: `cd web && bun install --frozen-lockfile && bun run build`,
   then `go build -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=v1.0.0-rc.45-wending-<sha>" -o new-api-wending-rc45-<sha>`;
   record sha256.
2. On vultr-sv: back up the binary (`cp -a /opt/new-api/new-api /opt/new-api/new-api.bak-$TS`) and the DB
   (`pg_dump "$SQL_DSN" -Fc -f /opt/new-api/db-backup-$TS-pre-login-scope.dump`).
3. `install -m 0755` the new binary, `systemctl restart new-api`, check `journalctl -u new-api` for migration errors.
4. Verify: `/api/status` version; `/oauth/authorize` 200; `POST /api/oauth/token` bogus → 401 `invalid_client`;
   `GET /api/oauth/userinfo` with `Bearer wdat_bogus` → 401; huabu and lrs one-click auth still work.
5. Register the shop client on 接入应用 (root):
   - name `WendingAI 商城`, homepage `https://shop.wending.ai`, confidential (**not** public), enabled
   - redirect URI (exact): `https://shop.wending.ai/auth/wendingai/callback`
   - **trusted = on**; show_in_sidebar optional (launch_url `https://shop.wending.ai/auth/login`)
   - quota / group / model-limit fields are irrelevant for the login scope
   - copy the client secret once into the shop admin (设置 → WendingAI 登录); keep a root-only copy at
     `/opt/new-api/oauth-clients/shop.secret` (600). Never paste it into chat or docs.
6. Rollback: reinstall the `.bak` binary and restart. The extra table/column are ignored by the old binary;
   outstanding `wdat_` tokens simply stop working (shop sessions already issued are unaffected).

## Key paths

- Backend: `model/wending_oauth.go`, `model/wending_oauth_login.go`, `controller/wending_oauth.go`, `controller/wending_oauth_test.go`, `controller/wending_oauth_login_test.go`, `router/wending-oauth-router.go`
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
1b. **lrs (狼人杀) RP integration – LIVE (2026-11-10 04:25 UTC+7).** Repo `sharefuture1/wending-model-arena`
   (`main` @ `53f1ee8`, docs `d5c8b90`), Vercel project `wending-model-arena`, production `dpl_2QPeXP7Pp81pkUMUDX8L9FnqQVMb`
   (rollback `dpl_G6ZJfMjvXT6jxthg15TU1GpUjAFP`). Reused the admin-created client **id 4 `ai狼人杀`**
   `wd_W8JvSLWqrmwBcP2ggPJokbSy`: secret rotated (old one unused), redirect_uris
   `["https://lrs.wending.ai/api/auth/wending/callback","https://lrs.wending.ai/"]`,
   launch_url `https://lrs.wending.ai/?wending_auth=1`, sidebar_order 20. Secret: `/opt/new-api/oauth-clients/lrs.secret` (600).
   Backup/rollback SQL: `/opt/new-api/wending_oauth_clients-20261109-211831.pre-lrs.{backup,rollback}.sql`.
   Details: that repo's `docs/HANDOFF_WENDING_OAUTH.md`.
   2026-11-10 update: client id 4 renamed `ai狼人杀` → `WendingAI 狼人杀` (backup/rollback
   `/opt/new-api/wending_oauth_clients-id4-20261109-213319.pre-rename.{backup.csv,rollback.sql}`, 600). lrs now
   remembers the issued key per device, so `?wending_auth=1` no longer re-authorizes (and rotates the key) when a key is
   saved. Optimization details: that repo's `docs/HANDOFF_OPTIMIZATION.md`.
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

## Production deployment record — 2026-11-09 22:58 UTC (shop sign-in scope)

- PR #1 (`wending-shop-login`) + review fix `e9ed899` (userinfo returns e-mail **only** with the `email` scope)
  merged into `wending` as `44e03dc`. Review: no other blocking issues (see PR #1 review notes in the bot report).
- Binary `v1.0.0-rc.45-wending-44e03dc`, sha256 `eb0c139513e0b58f1eb9880dd919c51e84c4bc752bce04acc16ec722814de6e9`
  (CGO_ENABLED=0, `-s -w`). Restart → healthy `/api/status`: ~5.5 s.
- Previous binary (2c8eae5): `/opt/new-api/new-api.bak-20261109-225755`
- DB backup: `/opt/new-api/db-backup-20261109-225755-pre-login-scope.dump` (600)
- AutoMigrate: table `wending_oauth_access_tokens`, column `wending_oauth_clients.trusted` (nullable, NULL = false,
  same as the other bool columns). Journal: no errors.
- Verified: version; `/oauth/authorize` 200; bogus token request → 401 invalid_client; userinfo `Bearer wdat_bogus`
  and `sk-bogus` → 401; huabu + lrs `/api/auth/wending/start` → 302 → authorize 200; lrs real secret + fake code →
  400 invalid_grant (api-scope exchange path intact).
- **Shop client** registered (id 5): `wd_61eVWNqaUiGtRC8eT3Ss0Gsr`, name `WendingAI 商城`, homepage
  `https://shop.wending.ai`, confidential, enabled, **trusted**, redirect exactly
  `https://shop.wending.ai/auth/wendingai/callback`, sidebar off (launch_url `https://shop.wending.ai/auth/login`).
  Secret: `/opt/new-api/oauth-clients/shop.secret` (root 600; client_id/client_secret lines). Verified real
  secret + fake code → 400 invalid_grant, wrong secret → 401.
  Table backup / rollback: `/opt/new-api/wending_oauth_clients-20261109-225911.pre-shop.{backup.csv,rollback.sql}`.
- Note: an older admin-created sidebar entry id 2 `wendingai shop` (`wd_EL6HVNRwXn7oiEmnLAdoAQcl`, redirect
  `https://shop.wending.ai/`) still exists and is unrelated to sign-in; keep or delete it as the owner prefers.
- Rollback: `install -m 0755 /opt/new-api/new-api.bak-20261109-225755 /opt/new-api/new-api && systemctl restart new-api`
  (extra table/column ignored by the old binary); remove the shop client with the rollback SQL above.

