# WENDING_PATCHES

Fork-local customizations on top of upstream `QuantumNous/new-api`.
Branch `wending` is based on tag **v1.0.0-rc.45** (the version running in production).
All commits are prefixed `feat(wending): ...`.

## Feature: 稳定点AI账号一键授权 (first-party OAuth 2.0 authorization server)

See [docs/WENDING_OAUTH.md](docs/WENDING_OAUTH.md).

### New files (fork-only, no upstream conflicts expected)

| Path | Purpose |
| --- | --- |
| `model/wending_oauth.go` | Tables `wending_oauth_clients`, `wending_oauth_grants`, `wending_oauth_codes`; secret hashing, PKCE, grant/code lifecycle; sidebar launch fields (`show_in_sidebar`, `launch_url`, `sidebar_order`); `trusted` flag |
| `controller/wending_oauth.go` | Consent API, token endpoint, userinfo, user grant management, root client CRUD, admin grant view, `GET /sidebar-apps` |
| `controller/wending_oauth_test.go` | Token-endpoint tests (PKCE, redirect mismatch, code reuse, revoked grant, client auth, rotation, expiry) |
| `model/wending_oauth_login.go` | `login`/`email` scopes, table `wending_oauth_access_tokens` (short-lived `wdat_` sign-in tokens, SHA-256 only), login grant without API key |
| `controller/wending_oauth_login_test.go` | Sign-in scope tests (exchange → `wdat_`, userinfo, scope rules, trusted skip-consent, revoke) |
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
| `web/src/hooks/use-sidebar-data.ts` | import `useWendingNavItems` / `useWendingAppsNavGroup`, spread personal/admin items, inject dynamic "稳定点应用" group | +8 |
| `web/src/hooks/__tests__/sidebar-config.test.tsx` | expected Personal sidebar items now include `wending.nav.authorizedApps` | +6/-1 |
| `web/src/components/layout/components/nav-group.tsx` | `SidebarMenuLink` opens absolute `http(s)://` URLs in a new tab (sidebar launch apps) | +18 |
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

## Branch `wending-integration` (2026-11-18): non-OAuth customizations

`wending-integration` = `wending` + `wending-desktop-auth` (merged) + fixes. Besides the OAuth
feature above it carries Stone's site/pricing/landing work, which **does touch upstream
files** and will conflict on upstream merges (tested 2026-11-18 against upstream `main`
669d77497: Go merges cleanly; conflicts only in `web/src/features/pricing/components/model-card.tsx`
and `web/src/hooks/use-sidebar-data.ts`, plus locale JSON when merging tag rc.51).

| Area | Upstream files |
| --- | --- |
| Admin-forced appearance (`appearance_setting`, exposed via `/api/status`) | `controller/misc.go`, `controller/option.go`, `setting/appearance_setting/appearance.go` (new), `web/src/context/theme-*.tsx`, `web/src/components/{theme-switch,config-drawer,command-menu}.tsx`, `web/src/features/system-settings/site/*`, `web/src/stores/system-config-store.ts`, `web/src/lib/status-query.ts`, `web/src/main.tsx` |
| Per-1K public pricing, advantage bar | `web/src/features/pricing/**` (`DEFAULT_TOKEN_UNIT = 'K'`; raw task-expression prices in logs/admin editors keep the 1M label) |
| Editorial landing page | `web/src/features/home/**` (old sections removed); price preview only fetches `/api/pricing` when pricing is visible to the visitor |
| Standalone frontend on Vercel | `web/vercel.json` (fork-only) – see docs/HANDOFF.md § go.wending.ai |

## Branch `wending-ui` (2026-11-18): 稳定点AI branding / UI (frontend only)

New files: `web/src/lib/wending-brand.ts`, `web/src/features/home/components/sections/quick-access.tsx`,
`web/src/features/wallet/components/shop-cta.tsx`, brand assets in `web/public/`
(`favicon.svg`, `apple-touch-icon.png`, `icon-192.png`, `icon-512.png`, `og-image.jpg`,
`wendingai-mark.svg`; `favicon.ico` and `logo.png` replaced), tests
`features/about/__tests__/open-source-notice.test.tsx`, `components/layout/components/__tests__/footer.test.tsx`.

| Upstream file | Change |
|---|---|
| `web/index.html` | 稳定点AI title, zh/en description, OG/Twitter meta, svg/touch icons, `lang=zh-CN` |
| `web/src/components/layout/components/footer.tsx` | removed New API/GitHub credit (`ProjectAttribution`); About link |
| `web/src/features/about/index.tsx` | AGPLv3 §7(b) open-source notice on every variant; empty state without upstream links |
| `web/src/features/home/**` | hero CTAs (wallet top-up, shop), quick-access section, connection panel without Cherry Studio / CC Switch |
| `web/src/features/keys/**` | CC Switch row action, dialog and dialog type removed |
| `web/src/features/pricing/lib/mock-stats.ts` | Cherry Studio removed from mock app list |
| `web/src/features/auth/auth-layout.tsx` | branded split layout |
| `web/src/features/wallet/index.tsx` | 稳定点AI Shop card |
| `web/src/features/errors/general-error.tsx` | feedback link → 稳定点AI support |
| `web/src/features/dashboard/components/overview/overview-dashboard.tsx` | sample model fallback `deepseek-chat` |
| `web/src/lib/constants.ts`, `system-brand.tsx`, `assets/logo.tsx` | default name 稳定点AI |
| `web/src/i18n/locales/*.json` | new keys; zh/zh-TW vendor-neutral wording |
