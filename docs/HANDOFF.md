# HANDOFF – 稳定AI账号一键授权 (Wending OAuth) + standalone frontend

_Last updated: 2026-11-19 16:55 EST – production = `wending` fa32c8e on api.wending.ai (backend restarted 2026-11-19 21:47 UTC) and go.wending.ai (dpl_GCaVwXPVKwp1RQgsaZ42GQRaUU2S)._

## Production deployment record — 2026-11-19 21:28–21:48 UTC (PR #8 remove landing shell, logo/name options, fa32c8e)

- Stone report: "蒙了一层", old style shows then new style flashes on refresh, homepage buttons need two clicks.
- Root cause (headless Chrome, zh-CN, mobile + desktop, cold + warm, both domains): the PR #6 static shell `#wd-boot`
  (position fixed, z-index 40, its own markup + the new /logo.png) stayed on top of the already-rendered React app for
  1.5–6 s (elementFromPoint = shell while the React header already existed), then was removed and the React header
  showed the DB `Logo` option (old R2 wendingai.jpg). Clicks during that window hit shell links (full page loads) or
  landed as the shell was being removed. No service worker involved.
- Fix: PR #8 (`wending-remove-boot-shell` → fa32c8e) restores index.html from 573903d18 (no shell, no R2 preconnect),
  deletes `web/src/lib/boot-shell.ts`; lazy locales / contrast / robots / sitemap / footer sanitising from PR #6 stay.
- Options (DB, live via SyncOptions within SYNC_FREQUENCY, no restart needed): `Logo` R2 wendingai.jpg → `/logo.png`
  (same-origin on api and go, 180×180 14 KB); `SystemName` `稳定AI WendingAI` (typo) → `问鼎AI WendingAI`.
  Backup `/opt/new-api/options-backup-20261119-212839-logo-name.csv`, rollback SQL
  `/opt/new-api/options-rollback-20261119-212839.sql` (both 600).
- go.wending.ai production: **dpl_GCaVwXPVKwp1RQgsaZ42GQRaUU2S** (fa32c8e); previous dpl_6kNrdp6KNX7Nbtmy3nMC3zv5sfqJ.
- Backend: **`v1.0.0-rc.45-wending-fa32c8e`**, sha256 `8de3c8bcf86ad7bded08c32d557508249e257ed795e388aa7d7d80f5ec1c4cb5`,
  200 after 3.5 s. Backups TS 20261119-214712: `db-backup-20261119-214712-pre-fa32c8e.dump` (41 MB, 43 table-data),
  `new-api.bak-20261119-214712` (= f15ca1d, 78c200e3…), `env.conf.bak-20261119-214712`. Caddy untouched, no migration.
- Verified both domains: no shell in HTML/DOM, never topmost element, header logo `/logo.png` on every sample
  (old jpg never seen over 6 reloads), single click works at 200 ms–5 s after navigation; mobile menu taps work;
  控制台/模型广场 for guests open the intended "需要登录 … 5 秒后跳转" dialog (by design, not a dead click).
  api: `/v1/models` 401, chat ok, OAuth fake code 400 / bogus client 401, journal 0 panic/fatal.
- Note: without the shell a cold load from a far network shows a blank page until the bundle runs (~3 s from the
  test box, TTFB ~0.9 s). Visitors with an old `status` in localStorage may see the old logo once until /api/status
  refreshes.
- **Rollback:** backend `install -m 0755 /opt/new-api/new-api.bak-20261119-214712 /opt/new-api/new-api && systemctl restart new-api`;
  go: promote dpl_6kNrdp6KNX7Nbtmy3nMC3zv5sfqJ; options: `psql "$SQL_DSN" -f /opt/new-api/options-rollback-20261119-212839.sql`.


## Production deployment record — 2026-11-19 20:36–20:39 UTC (PR #4 + #5 + #6 + #7, f15ca1d)

- Approved by Stone ("做好后你觉得前端没问题了再重启上线") after the frontend pass.
- PR #6 `wending-fe-perf` (→ 1e5cfaa): lazy locales (only en + zhCN bundled; initial JS ~1.9 → ~1.0 MB gzip),
  static landing shell `#wd-boot` in index.html for zh visitors on "/" (removed by Home when loaded, by any other
  route, or after 15 s; `web/src/lib/boot-shell.ts`), preconnect to the R2 logo host (drop when the logo moves),
  `web/public/robots.txt` + `sitemap.xml`, light `--primary` L 0.692 → 0.53 (WCAG AA), footer legal links full
  contrast, footer HTML via DOMPurify (lazy), WendingAI tagline, notification button named from content.
- PR #7 `wending-model-alias-quota` (→ f15ca1d): live check after the first restart showed `usage.quota.model`
  (a wb2 gateway extension) still said deepseek-v4.1-flash; now aliased too, non-empty `usage.quota.shared_models`
  dropped.
- go.wending.ai production: **dpl_6kNrdp6KNX7Nbtmy3nMC3zv5sfqJ** (1e5cfaa); previous dpl_HRiJJ3s3HToyWmMb4DN9r6EG5MnN.
- Lighthouse go.wending.ai (median of 3, zh-CN): mobile perf 38 → 54, FCP 5.2 → 3.7 s, LCP 16.8 → 5.0 s,
  TBT 782 → 492 ms; desktop perf 53 → 95, FCP 3.9 → 0.6 s, LCP 6.3 → 0.7 s; a11y 96 → 100, SEO 92 → 100, BP 100.
- Backups (TS 20261119-203622 UTC): `/opt/new-api/db-backup-20261119-203622-pre-1e5cfaa.dump` (pg_dump -Fc, 600,
  41 MB, 43 table-data entries), `/opt/new-api/new-api.bak-20261119-203622` (= 2fb5dac, sha256 e1800a48…),
  `/opt/new-api/env.conf.bak-20261119-203622` (env unchanged).
- Restart 1: `v1.0.0-rc.45-wending-1e5cfaa` (sha256 19f5cf70…, kept as `new-api.staged-1e5cfaa`) → 200 after 3.7 s.
  Restart 2: **`v1.0.0-rc.45-wending-f15ca1d`**, sha256 `78c200e3f7eb3b074475427a6281d91f37df2c51d6d0fd80573b2e1974170ffc`
  → 200 after 3.5 s. Caddy not touched. No DB migration.
- Verified: `/` 200 (shell present), `/robots.txt` text/plain, `/v1/models` 401, OAuth token fake code 400 /
  bogus client 401, authorize 200, refresh foreign origin 403 / go origin 401; Stone's key: kimi-k3 non-stream,
  stream (no "deepseek" anywhere), Claude /v1/messages all return `kimi-k3`; `/api/log/token` (user view) 0 rows
  with model trace while DB rows still store it (root view); journal clean (0 panic, relay 5xx 0).
- **Rollback:** `install -m 0755 /opt/new-api/new-api.bak-20261119-203622 /opt/new-api/new-api && systemctl restart new-api`
  (2fb5dac); go.wending.ai: promote dpl_HRiJJ3s3HToyWmMb4DN9r6EG5MnN. DB restore only if really needed.

## 2026-11-19: PR #5 – relay responses report the requested model (STAGED, not live)

- Request (Stone): "api 返回的也要改名成用户请求的模型".
- PR #5 `wending-response-model-alias` → `wending` **509e8b4**. New `middleware/response_model_alias.go`
  (`ResponseModelAlias()`, installed before `Distribute()` on /v1 HTTP relay, /v1beta Gemini, /pg, /v1/responses,
  /v1/images/*). It wraps the response writer and rewrites `model`, `message.model`, `response.model`,
  `modelVersion` to `original_model` (the requested name): buffered JSON bodies (Content-Length recomputed) and every
  SSE / NDJSON `data:` line. Non-JSON (audio etc.), compressed bodies, unknown requested model → untouched.
  Billing/logs/diagnostics are unaffected (they read the upstream body first). Toggle: env
  `RESPONSE_MODEL_ALIAS_ENABLED=false` restores passthrough (default on).
  Not covered: WebSocket realtime / responses-over-WS, video / MJ task routes.
- Tests: `middleware/response_model_alias_wending_test.go`; full Go suites pass except the pre-existing, env-related
  `relay/channel/advancedcustom` TestAdaptorMasksDialedHostsInDialFailure (fails on origin/wending too).
  Local smoke (throwaway SQLite instance + fake upstream returning deepseek-v4.1-flash): chat non-stream/stream,
  Claude /v1/messages non-stream/stream, embeddings all returned `kimi-k3`; quota/token counts unchanged; log
  diagnostics still record the returned upstream name (root-only).
- Binary `v1.0.0-rc.45-wending-509e8b4`, sha256 `b03374b7c9e01c12207e595fabfa1727f9ae1a4f4403bafbbe3ba69afee3540b`,
  staged at `/opt/new-api/new-api.staged-509e8b4` (supersedes and replaced `new-api.staged-1a74589`). Web unchanged
  vs 1a74589, so no Vercel deploy. Go-live after approval: back up, `install -m 0755 /opt/new-api/new-api.staged-509e8b4
  /opt/new-api/new-api && systemctl restart new-api`; no DB migration; rollback = reinstall backup (2fb5dac, e1800a48…).
- Frontend review (2026-11-19, Lighthouse on go.wending.ai: mobile perf 30 / a11y 96 / BP 100 / SEO 92; desktop
  perf 67): top items — 7 locale JSONs bundled into the entry chunk (entry 5.0 MB, 1.4 MB gz; initial JS ~1.9 MB gz);
  system Logo is a 742 KB JPEG on r2.dev without cache headers; robots.txt / sitemap.xml return the SPA HTML;
  gpt-* model names enabled for users (compliance); sign-up CTA contrast; footerHtml rendered unsanitized.

## 2026-11-18 (evening): PR #4 – model trace hidden from non-root (backend STAGED, not live)

- Request (Stone): hide "响应模型 / 上游请求模型 / 上游返回的模型名…不一致" from all users, keep for root.
- PR #4 `wending-hide-model-trace` → merged into `wending` as **1a74589**.
  - Backend: `model/log_other.go` strips `response_model`, `upstream_model_name`, `is_model_mapped` from
    `other` unless visibility is root (affects self logs, search, token logs, export, and admin (role 10) list);
    `controller/task.go` blanks `properties.upstream_model_name` for non-root. Tests:
    `model/log_model_trace_wending_test.go`, `controller/task_model_trace_wending_test.go`.
  - Frontend: usage-logs table column, mobile card, details dialog (Response Model + Model Mapping), task details
    show them only for root view (role 100 + "all" scope). Web 204 files / 2386 tests, `tsgo -b`, build OK.
  - Relay response bodies NOT changed: OpenAI-compatible channels pass the upstream `model` field through, so API
    clients still see e.g. `"model":"deepseek-v4.1-flash"` in the JSON. Changing that needs a separate decision.
- go.wending.ai production: **dpl_HRiJJ3s3HToyWmMb4DN9r6EG5MnN** (1a74589); previous dpl_9YMu3WZUJsJWzn2H3mcrMqfynNhZ.
  Frontend gating is live there; the backend still returns the fields until the restart.
- Staged binary on vultr-sv: `/opt/new-api/new-api.staged-1a74589` (`v1.0.0-rc.45-wending-1a74589`, 0755),
  sha256 `a364d555a072e155d692cc543e536efe80574d9b9cd5da2cc3784cc06c861ab4`. **Not swapped, not restarted.**
  To go live (after approval): back up current binary (`cp -a /opt/new-api/new-api /opt/new-api/new-api.bak-<TS>`),
  `install -m 0755 /opt/new-api/new-api.staged-1a74589 /opt/new-api/new-api && systemctl restart new-api`.
  No DB migration in this change. Rollback: reinstall the backup (= 2fb5dac, sha256 e1800a48…) and restart.
- Why kimi-k3 answered as deepseek-v4.1-flash: channel 35 "国模wendingai" (type OpenAI, https://wb2.wending.ai) has
  no mapping for kimi-k3 (its mapping only rewrites gpt-5.5/gpt-5.4), so new-api forwarded `kimi-k3` unchanged.
  The upstream WorkBuddy manager (host "vultr", `/opt/workbuddy-manager`, settings key `model_map` in
  `data/manager.db`) maps `kimi-k3`, `kimi-k3.1`, `kimi-k3.1-pro`, `deepseek-v4-pro` → `deepseek-v4.1-flash`.
  Channel 24 "kimi" is disabled. Nothing was changed (read-only inspection).
- Vercel Git: project `wending-go` still has no Git link (no `link` in project, pushes create no deployments) —
  deploys keep going through the API.

## Production deployment record — 2026-11-18 20:00 UTC (2fb5dac: PR #2 + PR #3)

- PR #3 (`wending-ui`) merged into `wending` as **2fb5dac** (sharefuture1). Tree identical to the tested
  `wending-ui` head 054e88333.
- Binary `v1.0.0-rc.45-wending-2fb5dac`, sha256 `e1800a48a4160d170471f1a4df65b9710c62f1a1aad4c923e6ebef7a907ebc99`
  (CGO_ENABLED=0, `-s -w`, go1.27.3, web built with `bun run build`). Checks: `go vet`, `go test ./controller
  ./model ./middleware ./router ./service ./common`, web 204 files / 2383 tests, `tsgo -b`, build.
- Backups (TS 20261118-195912, UTC): `/opt/new-api/db-backup-20261118-195912-pre-2fb5dac.dump` (pg_dump -Fc,
  600), `/opt/new-api/new-api.bak-20261118-195912` (= 44e03dc, sha256 eb0c1395…), `/opt/new-api/env.conf.bak-20261118-195912`,
  `/etc/caddy/Caddyfile.bak-20261118-195912`, `/etc/caddy/wending-edge.caddy.bak-20261118-195912` (old secret),
  `/etc/caddy/Caddyfile.go-staged.bak-20261118-195912`. Old edge secret also kept on the ops box
  (`/workspace/.secrets/*.old-20261118`).
- env.conf: `SESSION_COOKIE_TRUSTED_URL=https://api.wending.ai,https://go.wending.ai` (was api only).
- new-api restart → `/api/status` 200 after **3.4 s**. AutoMigrate added `wending_oauth_grants.device_id` /
  `device_name` (varchar). Journal: no migration errors.
- Caddy: staged edge config applied (`Caddyfile` = `Caddyfile.go-staged`), `X-Wending-Edge` secret rotated in
  `/etc/caddy/wending-edge.caddy` (root:caddy 0640) and in Vercel env `WENDING_EDGE_SECRET` (prod+preview).
  `caddy validate` OK; restart → 200 after **4.8 s**. The header is now stripped before request-capture (52029).
- go.wending.ai production: **dpl_9YMu3WZUJsJWzn2H3mcrMqfynNhZ** (`wending` 2fb5dac, READY, new edge secret);
  previous dpl_FgwV7kZ8tm7uogFscXCH2Fdssx8b. Via go.wending.ai new-api logs the real visitor IP; forged
  `X-Real-IP` / wrong `X-Wending-Edge` sent directly to api.wending.ai is ignored (logs the TCP peer).
- Verified: version; `/` 200 (new WendingAI title); `/v1/models` 401; `/oauth/authorize` 200; bogus token → 401
  invalid_client; userinfo bogus → 401; real secrets (huabu, lrs, shop, casino/21, laotalk) + fake code → 400
  invalid_grant; huabu/lrs/21/laos `/api/auth/wending/start` → 302 → authorize 200; refresh with
  `Origin: https://go.wending.ai` (direct and via go.wending.ai) → 401 AUTH_UNAUTHORIZED (no longer 403), foreign
  origin → 403; webhooks reachable: waffo-pancake 401 invalid signature, epay notify "fail", legacy waffo 403
  (disabled, unchanged).
- Pre-existing, unrelated: `mcpwd1.wending.ai` → 502 (nothing listens on 127.0.0.1:3087 since ~2026-09).
- **Rollback:**
  `install -m 0755 /opt/new-api/new-api.bak-20261118-195912 /opt/new-api/new-api && cp -a /opt/new-api/env.conf.bak-20261118-195912 /opt/new-api/env.conf && systemctl restart new-api`
  (extra columns are ignored by the old binary);
  Caddy: `cp -a /etc/caddy/Caddyfile.bak-20261118-195912 /etc/caddy/Caddyfile && systemctl restart caddy`;
  go.wending.ai: promote dpl_FgwV7kZ8tm7uogFscXCH2Fdssx8b (needs the old secret only if the edge Caddyfile stays).
  DB restore only if really needed: `pg_restore --clean -d "$SQL_DSN" db-backup-20261118-195912-pre-2fb5dac.dump`.

## 2026-11-18 (afternoon): PR #2 merged, Vercel Git, `wending-ui`

- **PR #2 merged** into `wending` as **cf0d5b4ff** (sharefuture1). (Deployed later with PR #3 as 2fb5dac.)
  Note: the embedded UI on api.wending.ai only changes when the backend binary is redeployed.
- go.wending.ai production at that time = dpl_FgwV7kZ8tm7uogFscXCH2Fdssx8b (`wending` cf0d5b4ff); superseded by dpl_9YMu3WZUJsJWzn2H3mcrMqfynNhZ.
- **Vercel Git (project `wending-go`)**: the API/MCP cannot connect a repo to an existing project or
  set the production branch. Already set via API: Ignored Build Step
  `if [ "$VERCEL_GIT_COMMIT_REF" = "main" ]; then exit 0; fi; git diff --quiet HEAD^ HEAD -- .`
  (root is `web/`, so `.` = web/; `main` never builds). **Still manual (dashboard):** Settings → Git →
  Connect `sharefuture1/newapinew`; Settings → Environments → Production → Branch Tracking = `wending`;
  Root Directory stays `web`. Until then deploy with `create_deployment` + `gitSource`
  (`ref`/`sha`), no `target` for previews, `target: production` only for `wending`.
- **Branch `wending-ui`** (PR #3 → `wending`), frontend only:
  - Cherry Studio / CC Switch removed (home connection panel, key row action + `cc-switch-dialog`,
    pricing mock app list). Chat presets stay data-driven (production `Chats=[]`).
  - Footer GitHub/New API credit removed; footer gets an "About" link.
  - **License:** upstream NOTICE (AGPLv3 §7(b)) requires the text
    "Frontend design and development by New API contributors." and a visible link to
    https://github.com/QuantumNous/new-api in a prominent about/legal/footer location. Both are kept in
    an "开源声明" block at the bottom of **every** About variant (markdown/html/url/empty), plus the
    modified source link https://github.com/sharefuture1/newapinew (AGPLv3 §13 – the repo must stay
    public, or another source offer is needed). Changed files carry "Modified by WendingAI (2026-11)"
    comments (§7(c)). Do not remove the About notice; tests `open-source-notice.test.tsx` and
    `footer.test.tsx` guard it.
  - Homepage: hero buttons **钱包充值** (`/wallet`, router guard sends anonymous users to sign-in and
    back) and **WendingAI 商城** (https://shop.wending.ai/, new tab), plus a "快捷入口" card section.
    Wallet page: WendingAI Shop card. Constants in `web/src/lib/wending-brand.ts`.
  - Branding: vector mark (`public/wendingai-mark.svg`, traced from the R2 logo), `favicon.svg/.ico`,
    `apple-touch-icon.png`, `icon-192/512.png`, `og-image.jpg`, zh/en SEO + OG/Twitter meta in
    `index.html`, default system name WendingAI. Auth pages: branded split layout on desktop.
    The DB options `Logo` (R2 jpg) and `SystemName` still override header logo/favicon/title at runtime.
  - Compliance: zh copy says "兼容主流接口协议" instead of naming a vendor; dashboard sample-curl fallback
    model `deepseek-chat` (real first model is used when available). Error page "联系客服" → t.me/wendingai.
  - Checks: `bun run test` 204 files / 2383 tests, `tsgo -b`, `bun run build` pass.
  - Screenshots (local instance with prod-like options): `/workspace/newapi-shots/{before,after}/`
    on the ops box, desktop 1440 + mobile 390, light + dark.

## 2026-11-18: `wending-integration` + go.wending.ai (Vercel)

**Branch `wending-integration`** (PR → `wending`), based on `wending` 45b8cf4 (= production 44e03dc + docs):
- merges `wending-desktop-auth` (desktop/loopback one-click auth with per-device grants, fixed
  client ids, `GET /api/oauth/clients/:id`; per-1K pricing; admin-forced appearance; editorial
  landing). Note: `wending-desktop-auth` alone was branched **before** the login scope and would
  drop shop/21 sign-in if deployed by itself – always deploy from `wending-integration`/`wending`.
  Only conflict: `web/src/features/wending/i18n.ts` (kept both key sets).
- **fix**: user with a login-only grant (token_id 0) asking for the `api` scope failed with
  "Token ID or user ID is empty". A key is now attached to that grant (transaction, guarded by
  `token_id <= 0`); rotations merge scopes so `login`/`email` survive; the code carries only the
  requested scope. Regression test `TestWendingAPIScopeAfterLoginOnlyGrant`.
- **fix**: usage-log details / admin task pricing editors said "/1K token" for raw per-1M
  expression prices (1000x overstated); they keep "/1M token". Public pricing stays per-1K.
- **fix**: new landing page fetched `/api/pricing` for anonymous visitors although pricing is
  login-only in production (`HeaderNavModules.pricing.requireAuth=true`) → 401 → refresh →
  redirect to /sign-in. Now only fetched when visible (test `price-preview.test.tsx`).
- `web/vercel.json` for the standalone frontend.
- DB impact when this binary is eventually deployed (AutoMigrate, additive): columns
  `wending_oauth_grants.device_id` / `device_name`. Not deployed yet.
- **huabu-nav**: the "server-only huabu-nav modification" does not exist in the running binary.
  `/opt/new-api/upgrades/20261020-230130-huabu-nav/` shows a custom rc45 build was prepared on
  2026-10-20, but the binary that ran afterwards (`new-api.bak-20261105-131408`) has the same
  sha256 as the stock rc45 binary (rolled back); since 2026-11-05 production runs builds from this
  repo. The huabu entry today comes from the DB: the sidebar "问鼎应用" group (OAuth client
  `launch_url`) and the `About` option text. Nothing to recover; no code re-implemented.

Checks: `go build`, `go vet ./controller ./model ./router`, `go test ./controller ./model
./middleware ./router ./service ./common` pass; `cd web && bun run test` 203 files pass,
`bunx tsgo -b` pass, `bun run build` pass.

### go.wending.ai (Vercel project `wending-go`)

- Vercel team `sharefuture1s-projects`, project `wending-go` (prj_0XHZGNfKYZP8zUc1KM6wMBeUrN5t),
  root `web/`, no Git auto-deploy (deployed explicitly from a branch/sha via the API).
  Current production: dpl_EM758hkun9hC33QErmPzCCxJ3VnP (wending-integration 5f891662a);
  previous: dpl_8zEgfBbyV7nL1nBecyEdY6KaUwH6 (0be55aff1, landing bounced anonymous visitors).
  Env `WENDING_EDGE_SECRET` (sensitive; copy kept on the ops box only).
- `web/vercel.json`: `/api`, `/pg`, `/v1`, `/mj` are proxied to `https://api.wending.ai`
  (same-origin for the browser, so the refresh cookie lands on go.wending.ai), rewrite caching
  disabled, header `x-wending-edge: $WENDING_EDGE_SECRET` added; `/static/*` immutable; SPA
  fallback. `/dashboard` is a SPA route and is not proxied.
- DNS (Cloudflare zone wending.ai): `go.wending.ai CNAME cname.vercel-dns.com`, DNS-only
  (record id 422ad44c08802e8ba35353fa80c9dcdd). Overrides the proxied `*.wending.ai` wildcard.
- **Backend items below were applied on 2026-11-18 20:00 UTC (see the deployment record above):**
  1. `SESSION_COOKIE_TRUSTED_URL=https://api.wending.ai,https://go.wending.ai` in
     `/opt/new-api/env.conf`. It is read from env once at startup (`common.InitSessionCookieSettings`),
     not from DB options. Until then `POST /api/user/auth/refresh|logout` from go.wending.ai return
     403 `AUTH_ORIGIN_FORBIDDEN`: sign-in works, but the session is lost on reload / after the
     15-minute access token.
  2. Caddy: staged `/etc/caddy/Caddyfile.go-staged` (validated) trusts `X-Real-IP` only when
     `X-Wending-Edge` matches (`/etc/caddy/wending-edge.caddy`, root:caddy 0640) and strips the
     header. Caddy runs with `admin off`, so `caddy reload` is impossible; applying needs
     `systemctl restart caddy`. Until then all go.wending.ai visitors share Vercel egress IPs for
     per-IP rate limits and logs.
  3. Turnstile: the widget renders on go.wending.ai/sign-in without a domain error (checked in
     headless Chrome), so no hostname change is needed.
- Rollback: remove the domain from the Vercel project (or delete project `wending-go`) and delete
  DNS record 422ad44c…; Caddy: `/etc/caddy/Caddyfile` is unchanged (backup
  `Caddyfile.bak-go-vercel-20261118-011136`).


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

