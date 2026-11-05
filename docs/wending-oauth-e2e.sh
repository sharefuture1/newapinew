#!/usr/bin/env bash
# End-to-end check of the Wending OAuth flow against a locally running binary.
# Usage: BASE=http://localhost:PORT ./wending-e2e.sh
set -euo pipefail
BASE=${BASE:?}
REDIRECT="http://localhost:9999/api/auth/wending/callback"
ROOT_PW=${ROOT_PW:-"Root-$(openssl rand -hex 8)"}
ALICE_PW="Alice-$(openssl rand -hex 8)"
step() { echo; echo "=== $*"; }
j() { python3 -c "import sys,json; d=json.load(sys.stdin); print(eval(sys.argv[1]))" "$1"; }

step "1. setup root user (skipped when SKIP_SETUP=1)"
[ "${SKIP_SETUP:-0}" = 1 ] || curl -s -X POST $BASE/api/setup -H 'Content-Type: application/json' \
  -d "{\"username\":\"root\",\"password\":\"$ROOT_PW\",\"confirmPassword\":\"$ROOT_PW\",\"SelfUseModeEnabled\":false,\"DemoSiteEnabled\":false}" | j "d['success']"

step "2. root login"
ROOT_AT=$(curl -s -X POST $BASE/api/user/login -H 'Content-Type: application/json' \
  -d "{\"username\":\"root\",\"password\":\"$ROOT_PW\"}" | j "d['data']['access_token']")
echo "root session token: ${ROOT_AT:0:12}…"

step "3. root registers OAuth client"
CLIENT_JSON=$(curl -s -X POST $BASE/api/wending/oauth/clients -H "Authorization: Bearer $ROOT_AT" -H 'Content-Type: application/json' \
  -d "{\"name\":\"WendingAI 创作台\",\"description\":\"e2e\",\"redirect_uris\":[\"$REDIRECT\"],\"enabled\":true,\"is_public\":false,\"allow_unlimited_quota\":true,\"default_remain_quota\":5000000}")
CLIENT_ID=$(echo "$CLIENT_JSON" | j "d['data']['client']['client_id']")
CLIENT_SECRET=$(echo "$CLIENT_JSON" | j "d['data']['client_secret']")
echo "client_id=$CLIENT_ID secret=${CLIENT_SECRET:0:8}…(hidden)"

step "4. register + login normal user alice"
curl -s -X POST $BASE/api/user/register -H 'Content-Type: application/json' \
  -d "{\"username\":\"alice\",\"password\":\"$ALICE_PW\",\"password2\":\"$ALICE_PW\"}" | j "d['success'], d['message']"
AT=$(curl -s -X POST $BASE/api/user/login -H 'Content-Type: application/json' \
  -d "{\"username\":\"alice\",\"password\":\"$ALICE_PW\"}" | j "d['data']['access_token']")
AUTH="Authorization: Bearer $AT"

VERIFIER=$(openssl rand -base64 48 | tr '+/' '-_' | tr -d '=\n' | cut -c1-64)
CHALLENGE=$(printf '%s' "$VERIFIER" | openssl dgst -sha256 -binary | openssl base64 -A | tr '+/' '-_' | tr -d '=')
STATE="st-$(openssl rand -hex 6)"
Q="response_type=code&client_id=$CLIENT_ID&redirect_uri=$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1],safe=''))" $REDIRECT)&state=$STATE&code_challenge=$CHALLENGE&code_challenge_method=S256&scope=api"

step "5. SPA route /oauth/authorize serves the console"
curl -s -o /dev/null -w "HTTP %{http_code} %{content_type}\n" "$BASE/oauth/authorize?$Q"

step "6. consent info (server-side validation)"
curl -s "$BASE/api/wending/oauth/authorize?$Q" -H "$AUTH" | j "d['success'], d['data']['client']['name'], d['data']['user_group']"
echo "-- unregistered redirect_uri is rejected without redirect:"
curl -s "$BASE/api/wending/oauth/authorize?${Q/localhost%3A9999/evil.example.com}" -H "$AUTH" | j "d['success'], d['oauth_error'], d['redirect_trusted']"

step "7. approve -> redirect with code"
APPROVE=$(curl -s -X POST $BASE/api/wending/oauth/authorize -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"response_type\":\"code\",\"client_id\":\"$CLIENT_ID\",\"redirect_uri\":\"$REDIRECT\",\"state\":\"$STATE\",\"code_challenge\":\"$CHALLENGE\",\"code_challenge_method\":\"S256\",\"scope\":\"api\",\"approve\":true,\"unlimited_quota\":false,\"remain_quota\":2500000,\"group\":\"\"}")
REDIRECT_TO=$(echo "$APPROVE" | j "d['data']['redirect_to']")
echo "redirect_to=$(echo "$REDIRECT_TO" | sed -E 's/code=([^&]{10})[^&]*/code=\1…/')"
CODE=$(python3 -c "import urllib.parse,sys;q=urllib.parse.parse_qs(urllib.parse.urlparse(sys.argv[1]).query);assert q['state'][0]==sys.argv[2];print(q['code'][0])" "$REDIRECT_TO" "$STATE")
echo "state round-trip OK"

step "8. token exchange with WRONG verifier"
curl -s -X POST $BASE/api/oauth/token -d grant_type=authorization_code --data-urlencode "code=$CODE" \
  --data-urlencode "redirect_uri=$REDIRECT" -d client_id=$CLIENT_ID --data-urlencode "client_secret=$CLIENT_SECRET" \
  -d code_verifier=$(printf 'x%.0s' {1..50}) ; echo

step "9. token exchange (correct)"
TOKEN_JSON=$(curl -s -X POST $BASE/api/oauth/token -d grant_type=authorization_code --data-urlencode "code=$CODE" \
  --data-urlencode "redirect_uri=$REDIRECT" -d client_id=$CLIENT_ID --data-urlencode "client_secret=$CLIENT_SECRET" \
  --data-urlencode "code_verifier=$VERIFIER")
KEY=$(echo "$TOKEN_JSON" | j "d['access_token']")
echo "$TOKEN_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); d['access_token']=d['access_token'][:7]+'…'; print(d)"

step "10. code reuse"
curl -s -X POST $BASE/api/oauth/token -d grant_type=authorization_code --data-urlencode "code=$CODE" \
  --data-urlencode "redirect_uri=$REDIRECT" -d client_id=$CLIENT_ID --data-urlencode "client_secret=$CLIENT_SECRET" \
  --data-urlencode "code_verifier=$VERIFIER"; echo

step "11. call /v1/models with issued key"
curl -s -w "\nHTTP %{http_code}\n" $BASE/v1/models -H "Authorization: Bearer $KEY"

step "12. userinfo"
curl -s $BASE/api/oauth/userinfo -H "Authorization: Bearer $KEY"; echo

step "13. key is visible as a normal token in the user's key list"
curl -s "$BASE/api/token/?p=1&size=10" -H "$AUTH" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; items=d['items'] if isinstance(d,dict) else d; print([(t['id'],t['name'],t['remain_quota'],t['unlimited_quota']) for t in items])"

step "14. list grants and revoke"
GRANT_ID=$(curl -s $BASE/api/wending/oauth/grants -H "$AUTH" | j "d['data'][0]['id']")
echo "grant_id=$GRANT_ID"
curl -s -X DELETE $BASE/api/wending/oauth/grants/$GRANT_ID -H "$AUTH" | j "d['success']"

step "15. revoked key is rejected"
curl -s -w "\nHTTP %{http_code}\n" $BASE/v1/models -H "Authorization: Bearer $KEY"
curl -s -w "\nHTTP %{http_code}\n" $BASE/api/oauth/userinfo -H "Authorization: Bearer $KEY"

step "16. deny path"
curl -s -X POST $BASE/api/wending/oauth/authorize -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"response_type\":\"code\",\"client_id\":\"$CLIENT_ID\",\"redirect_uri\":\"$REDIRECT\",\"state\":\"$STATE\",\"code_challenge\":\"$CHALLENGE\",\"code_challenge_method\":\"S256\",\"approve\":false}" | j "d['data']['redirect_to']"

step "17. admin grant view (root)"
curl -s "$BASE/api/wending/oauth/admin/grants?include_revoked=true" -H "Authorization: Bearer $ROOT_AT" | j "[(g['id'],g['user_id'],g['client_name'],g['revoked_at']>0) for g in d['data']]"

step "18. audit log entries (security category)"
curl -s "$BASE/api/audit/self?p=1&page_size=50" -H "$AUTH" | python3 -c "
import sys,json
d=json.load(sys.stdin)
data=d.get('data') or {}
items=data.get('items',data) if isinstance(data,dict) else data
print([ (i.get('action'), i.get('success')) for i in items if 'wending' in str(i.get('action'))] if isinstance(items,list) else d)"
echo; echo "E2E DONE"
