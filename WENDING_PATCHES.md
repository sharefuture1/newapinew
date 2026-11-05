# WENDING_PATCHES

Fork-local customizations on top of upstream `QuantumNous/new-api`.
Branch `wending` is based on tag **v1.0.0-rc.45** (the version running in production).
All commits are prefixed `feat(wending): ...`.

## Feature: 问鼎AI账号一键授权 (first-party OAuth 2.0 authorization server)

See [docs/WENDING_OAUTH.md](docs/WENDING_OAUTH.md).

### New files (fork-only, no upstream conflicts expected)

| Path | Purpose |
| --- | --- |
| `model/wending_oauth.go` | Tables `wending_oauth_clients`, `wending_oauth_grants`, `wending_oauth_codes`; secret hashing, PKCE, grant/code lifecycle |
| `controller/wending_oauth.go` | Consent API, token endpoint, userinfo, user grant management, root client CRUD, admin grant view |
| `controller/wending_oauth_test.go` | Token-endpoint tests (PKCE, redirect mismatch, code reuse, revoked grant, client auth, rotation, expiry) |
| `router/wending-oauth-router.go` | `SetWendingOAuthRouter` – every route of the feature |
| `web/src/features/wending/**` | Consent page, "已授权应用" page, root "接入应用" page, API client, runtime i18n bundle (`wending.*` keys), sidebar items |
| `web/src/routes/oauth/authorize.tsx` | SPA route `/oauth/authorize` (requires login, returns after sign-in) |
| `web/src/routes/_authenticated/wending/*.tsx` | `/wending/authorized-apps`, `/wending/oauth-clients` |
| `docs/WENDING_OAUTH.md`, `docs/HANDOFF.md`, `WENDING_PATCHES.md` | Documentation |

### Upstream files touched (keep this list exact)

| File | Change | Lines |
| --- | --- | --- |
| `model/main.go` | `migrateDB()` calls `migrateWendingOAuth()` right after the main `AutoMigrate` | +3 |
| `router/main.go` | `SetRouter()` calls `SetWendingOAuthRouter(router)` | +1 |
| `web/src/hooks/use-sidebar-data.ts` | import `useWendingNavItems`, spread `personal`/`admin` items | +4 |
| `web/src/hooks/__tests__/sidebar-config.test.tsx` | expected Personal sidebar items now include `wending.nav.authorizedApps` | +6/-1 |
| `web/src/routeTree.gen.ts` | **generated** by the TanStack Router plugin during `bun run build`; never edit by hand – regenerate after a rebase | generated |

Upstream locale JSON files are **not** touched: translations are registered at
runtime by `web/src/features/wending/i18n.ts` (`i18n.addResourceBundle`).

### Rebase notes

- Rebase onto **v1.0.0-rc.48** was tested (branch `wending-rc48-test`):
  one trivial conflict in `web/src/hooks/use-sidebar-data.ts` (upstream changed
  the `SidebarData` import line to also import `SidebarBadges`; keep both imports).
  Everything else applied cleanly; Go build, web build, OAuth tests and the e2e
  script pass on the rebased branch.
- After any rebase: run `cd web && bun run build` so `routeTree.gen.ts` is regenerated, and commit it if it changed.
