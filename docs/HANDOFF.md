# HANDOFF – 问鼎AI账号一键授权 (Wending OAuth)

_Last updated: 2026-11-05 07:20 EST (America/New_York)_

## What was done

- Branch **`wending`** created from tag **v1.0.0-rc.45** (production version) in
  `sharefuture1/newapinew`. Feature implemented as small `feat(wending): …` commits.
- First-party OAuth 2.0 authorization server (Authorization Code + mandatory PKCE S256):
  client registry (root), consent screen `/oauth/authorize`, approve/deny API,
  `POST /api/oauth/token`, `GET /api/oauth/userinfo`, user "已授权应用" page with revoke,
  admin grant view/revoke, audit-log entries. Details: [WENDING_OAUTH.md](WENDING_OAUTH.md).
- Upstream footprint kept minimal; every touched upstream file is listed in
  [../WENDING_PATCHES.md](../WENDING_PATCHES.md).
- Throwaway branch **`wending-rc48-test`**: same commits rebased onto v1.0.0-rc.48 (one trivial
  import conflict in `web/src/hooks/use-sidebar-data.ts`, resolved). Builds and passes tests/e2e.

## Status

| Check | Result |
| --- | --- |
| `go build`, `go vet` (controller/model/router) | pass |
| `go test ./controller ./model ./router ./middleware` | pass (incl. 10 new `TestWendingOAuth*` tests) |
| `cd web && bun run build`, `tsgo -b`, oxlint/oxfmt on new files | pass |
| `cd web && bun run test` (vitest) | full run: 190/199 files passed; the 9 failing files (i18n mock lacking `addResourceBundle`, sidebar expectation) all pass after commit 5c0cbe3 (re-run of those 9 files) |
| E2E on built binary, SQLite fresh DB | pass – `docs/wending-oauth-e2e.sh` |
| E2E on PostgreSQL 16: DB created by **stock rc.45**, then wending binary started twice (migration idempotent), then **rollback to stock rc.45** starts fine | pass |
| UI smoke (headless Chrome): login redirect → consent → approve → 已授权应用; root 接入应用 page + edit dialog | pass (screenshots on the agent box `/workspace/build/screens/`) |
| Production deploy | **NOT done** (by instruction) |

Not verified: MySQL (production uses PostgreSQL; no MySQL available on the build box),
Redis-enabled mode (token cache invalidation goes through upstream helpers), real upstream
channel calls (e2e uses `/v1/models` with no channels configured), mobile layout of the new pages.

## Key paths

- Backend: `model/wending_oauth.go`, `controller/wending_oauth.go`, `controller/wending_oauth_test.go`, `router/wending-oauth-router.go`
- Frontend: `web/src/features/wending/**`, `web/src/routes/oauth/authorize.tsx`, `web/src/routes/_authenticated/wending/*`
- Docs: `docs/WENDING_OAUTH.md` (flow, endpoints, integration snippet, deploy/rollback), `WENDING_PATCHES.md`
- E2E script: `docs/wending-oauth-e2e.sh` (`BASE=http://localhost:PORT ./docs/wending-oauth-e2e.sh`)
- Build artifact (agent box): `/workspace/build/new-api-wending-rc45-5c0cbe3` + `.sha256`
  (sha256 `3a19e964f4e1a5d04f3fbab5994890f23f65b93a081e26653396b61d4789e34b`, linux/amd64, CGO disabled, version string `v1.0.0-rc.45-wending-5c0cbe3`)

## Open items

1. Deploy to production (steps in WENDING_OAUTH.md → "Production deploy"); requires a maintenance window of a few seconds.
2. In production set 系统设置 → **服务器地址** to the public new-api URL (used as `base_url` in the token response).
3. Register the huabu client in 接入应用 with `https://huabu.wending.ai/api/auth/wending/callback` (or the real callback path) and put client id/secret into huabu's server env.
4. Implement the huabu side (`/api/auth/wending/start` + `/callback`, encrypted key storage) – example in WENDING_OAUTH.md.
5. MCP server: register as a **public** client with a loopback or reverse-domain redirect URI.
6. When upgrading production to rc.48+: rebase `wending` (see WENDING_PATCHES.md), rebuild web so `routeTree.gen.ts` regenerates.
7. Optional hardening: per-client consent "remember" UX, refresh/rotation API for apps, i18n for fr/ru/ja/vi (falls back to English).

## Production deployment record — 2026-11-05

- Server: vultr-sv (104.238.180.145), https://api.wending.ai, service `new-api`
- Deployed: 2026-11-05 13:14 UTC (08:14 America/New_York); binary built from `5c0cbe3`, reports `v1.0.0-rc.45-wending-5c0cbe3`
- sha256: `3a19e964f4e1a5d04f3fbab5994890f23f65b93a081e26653396b61d4789e34b`
- Previous binary (v1.0.0-rc.45, sha256 `0eb9bcda72b25f840226d2493f2ef1074471fa511298213bca89672add7ceee9`): `/opt/new-api/new-api.bak-20261105-131408`
- DB backup (pg_dump -Fc, 4.5 MB): `/opt/new-api/db-backup-20261105-131408-pre-wending.dump`
- Restart: stop took ~44s (graceful shutdown), ~47s total until healthy.
- Auto-migrated tables: wending_oauth_clients, wending_oauth_grants, wending_oauth_codes
- Verified: /api/status success, /oauth/authorize 200, /api/oauth/token bogus → 401 invalid_client, /api/pricing anon 401, /v1/models no key 401.

Rollback:
```
install -m 0755 /opt/new-api/new-api.bak-20261105-131408 /opt/new-api/new-api && systemctl restart new-api
# (new wending_oauth_* tables are additive and can stay; DB restore only if needed:
#  pg_restore --clean -d "$SQL_DSN" /opt/new-api/db-backup-20261105-131408-pre-wending.dump)
```
