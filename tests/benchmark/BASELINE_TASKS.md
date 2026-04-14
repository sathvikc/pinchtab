# PinchTab Benchmark Tasks (Baseline)

Reproducible benchmark using explicit curl commands against controlled fixture pages.
Each task maps 1:1 to an agent task in AGENT_TASKS.md for direct comparison.

## Environment

- **PinchTab Server**: `http://localhost:9867`
- **Fixtures Server**: `http://fixtures/` (inside Docker network)
- **Token**: `benchmark-token`
- **Auth Header**: `Authorization: Bearer benchmark-token`

## MANDATORY: Docker

```bash
cd ~/dev/pinchtab/tests/benchmark
docker compose up -d --build
# Wait for healthy: curl -sf -H "Authorization: Bearer benchmark-token" http://localhost:9867/health
```

## Recording

```bash
# Baseline (no tokens, optional response bytes):
./scripts/record-step.sh --type baseline <group> <step> <pass|fail> "notes"

# Minimal (just pass/fail):
./scripts/record-step.sh <group> <step> <pass|fail> "notes"
```

**On failure, include in notes:**
- What was expected
- What was actually returned
- HTTP status code / error message

## Tab Reuse

`POST /navigate` creates a new tab by default. To avoid multi-tab issues, the
runner must:

1. Capture `tabId` from the first navigate response (step 0.2).
2. Pass `"tabId":"TAB_ID"` in every subsequent navigate request body.
3. Use tab-scoped endpoints for actions and snapshots:
   - `POST /tabs/TAB_ID/action` instead of `POST /action`
   - `GET /tabs/TAB_ID/snapshot?...` instead of `GET /snapshot?...`
   - `GET /tabs/TAB_ID/text` instead of `GET /text`
   - `POST /tabs/TAB_ID/back` instead of `POST /back`
   - `GET /tabs/TAB_ID/screenshot` instead of `GET /screenshot`
   - `GET /tabs/TAB_ID/pdf` instead of `GET /pdf`

All curl examples below use `TAB_ID` as a placeholder. Replace with the actual
tab ID captured in step 0.2.

---

## Group 0: Setup & Diagnosis

### 0.1 Server reachable
```bash
curl -sf http://localhost:9867/health \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: `status == "ok"` in response body.

### 0.2 Auth required
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:9867/health
```
**Pass if**: HTTP status is `401` (auth rejected without token).

### 0.3 Auth works with token
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:9867/health \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: HTTP status is `200`.

### 0.4 Instance available
```bash
curl -sf http://localhost:9867/health \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Response contains `defaultInstance.status == "running"`. If not, run:
```bash
curl -X POST http://localhost:9867/instances/start \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{}'
```
And re-verify the new instance is running.

### 0.5 List existing tabs
```bash
curl -sf http://localhost:9867/tabs \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Returns a JSON array (possibly empty) without error.

### 0.6 Clean stale tabs
For each tab returned by step 0.5, close it:
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/close \
  -H "Authorization: Bearer benchmark-token"
```
Then verify cleanup:
```bash
curl -sf http://localhost:9867/tabs \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: After cleanup, the tab list is empty or contains only an about:blank tab.

### 0.7 Network reach to target
```bash
curl -sf -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"url":"http://fixtures/"}'
```
**Capture**: Save the `tabId` from the JSON response. All subsequent commands use this tab ID.
```bash
curl -sf "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Navigate returns HTTP 200 AND snapshot contains `VERIFY_HOME_LOADED_12345`.

### 0.8 Capture initial tab ID
```bash
curl -sf http://localhost:9867/tabs \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: The captured `TAB_ID` from step 0.7 appears in the tabs list.

---

## Group 1: Reading & Extracting

### 1.1 Wiki categories
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_WIKI_INDEX_55555` AND `COUNT_LANGUAGES_12` AND `COUNT_TOOLS_15`.

### 1.2 Click a link
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#link-go","waitNav":true}'
```
**Pass if**: HTTP 200 with `{"success":true}`.

### 1.3 Table extraction
```bash
curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_WIKI_GO_LANG_88888` AND `Robert Griesemer` AND `2009`.

### 1.4 Count list items
```bash
curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `FEATURE_COUNT_6`.

### 1.5 Article headlines
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/articles.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `The Future of Artificial Intelligence` AND `Climate Action in 2026` AND `Mars Colony`.

### 1.6 Dashboard metrics
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/dashboard.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `24,582` AND `$1,284,930` AND `4.28%`.

---

## Group 2: Search & Dynamic

### 2.1 Wiki search
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#wiki-search-input","text":"golang"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#wiki-search-btn","waitNav":true}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `VERIFY_WIKI_GO_LANG_88888` (search redirected to Go page).

### 2.2 No results search
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/search.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#search-input","text":"xyznonexistent"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#search-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: All curls return HTTP 200 AND snapshot shows no-results message.

### 2.3 AI content search
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/search.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#search-input","text":"artificial intelligence"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#search-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `The Future of Artificial Intelligence`.

---

## Group 3: Form

### 3.1 Complete form
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/form.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#fullname","text":"John Benchmark"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#email","text":"john@benchmark.test"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#phone","text":"+44 20 1234 5678"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"select","selector":"#country","value":"uk"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"select","selector":"#subject","value":"support"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#message","text":"This is a benchmark test message for PinchTab automation."}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#newsletter"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"input[name=priority][value=high]"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#submit-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_FORM_SUBMITTED_SUCCESS` AND `SUBMISSION_DATA_NAME_JOHN_BENCHMARK`.

### 3.2 Reset/refill
```bash
curl "http://localhost:9867/tabs/TAB_ID/snapshot?filter=interactive&format=compact" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains reset button or form element (after submission, page still has interactive elements).

---

## Group 4: SPA

### 4.1 Read app state
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/spa.html?reset=1"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_SPA_PAGE_99999` AND `TASK_STATS_TOTAL_3_ACTIVE_2_DONE_1`.

### 4.2 Add task
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#new-task-input","text":"Deploy to production"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"select","selector":"#priority-select","value":"high"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#add-task-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `TASK_ADDED_DEPLOY_TO_PRODUCTION_PRIORITY_HIGH`.

### 4.3 Delete task
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#task-1 .delete-task"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Total count is `3` (started with 3, added 1, deleted 1 = 3).

---

## Group 5: Login

### 5.1 Invalid login
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/login.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#username","text":"baduser"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#password","text":"wrongpassword"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#login-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `INVALID_CREDENTIALS_ERROR`.

### 5.2 Valid login
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#username","text":"benchmark"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#password","text":"test456"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#login-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_LOGIN_SUCCESS_DASHBOARD` AND `SESSION_TOKEN_ACTIVE_TRUE`.

---

## Group 6: E-commerce

### 6.1 Research products
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/ecommerce.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_SHOP_PAGE_44444` AND `$149.99` AND `$299.99` AND `$49.99` AND `Out of Stock`.

### 6.2 Add to cart
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#product-1 .add-to-cart"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#product-2 .add-to-cart"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `CART_ITEM_WIRELESS_HEADPHONES` AND `CART_ITEM_SMART_WATCH_PRO` AND `449.98`.

### 6.3 Checkout
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#checkout-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_CHECKOUT_SUCCESS_ORDER` AND `ORDER_TOTAL_449_98`.

---

## Group 7: Content + Interaction

### 7.1 Read & comment
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki-go.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#comment-text","text":"Great article on Go! Very comprehensive."}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"select","selector":"#comment-rating","value":"5"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#submit-comment"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: First snapshot contains `VERIFY_WIKI_GO_LANG_88888` AND final snapshot contains `COMMENT_POSTED_RATING_5_TEXT_RECEIVED`.

### 7.2 Cross-page research
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#link-go","waitNav":true}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: First snapshot contains `COUNT_LANGUAGES_12` AND second snapshot contains `VERIFY_WIKI_GO_LANG_88888`.

---

## Group 8: Error Handling

### 8.1 404 handling
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/nonexistent-page-xyz.html"}'
```
**Pass if**: Returns response without crash (HTTP 200 with error page, or structured error).

### 8.2 Missing element
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#element-that-does-not-exist"}'
```
**Pass if**: Error response with clear message (not crash).

---

## Group 9: Export

### 9.1 Screenshot
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/dashboard.html"}'

curl http://localhost:9867/tabs/TAB_ID/screenshot \
  -H "Authorization: Bearer benchmark-token" \
  --output /tmp/benchmark-screenshot.png

ls -la /tmp/benchmark-screenshot.png
```
**Pass if**: File exists and size > 10240 bytes.

### 9.2 PDF export
```bash
curl http://localhost:9867/tabs/TAB_ID/pdf \
  -H "Authorization: Bearer benchmark-token" \
  --output /tmp/benchmark-dashboard.pdf

ls -la /tmp/benchmark-dashboard.pdf
```
**Pass if**: File exists and size > 10240 bytes.

---

## Group 10: Modals

### 10.1 Open modal
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/dashboard.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#settings-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `Dashboard Settings`.

### 10.2 Modal interaction
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"select","selector":"#theme-select","value":"dark"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#modal-save"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `THEME_DARK_APPLIED`.

---

## Group 11: Persistence

### 11.1 State after reload
```bash
# Start fresh (reset param clears localStorage on this load only)
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/spa.html?reset=1"}'

# Add the persistent task
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#new-task-input","text":"Persistent Task Test"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#add-task-btn"}'

# Navigate away then back WITHOUT reset param — state should persist in localStorage
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/"}'

curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/spa.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `TASK_PERSISTENT_TEST_FOUND_AFTER_RELOAD`.

### 11.2 Logout/re-login
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/login.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#username","text":"benchmark"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#password","text":"test456"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#login-btn"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#logout-btn"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#username","text":"benchmark"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#password","text":"test456"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#login-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_LOGIN_SUCCESS_DASHBOARD` AND `SESSION_RENEWED`.

---

## Group 12: Multi-page Nav

### 12.1 Navigate & return
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/"}'

curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#link-go","waitNav":true}'

curl -X POST http://localhost:9867/tabs/TAB_ID/back \
  -H "Authorization: Bearer benchmark-token"

curl -X POST http://localhost:9867/tabs/TAB_ID/back \
  -H "Authorization: Bearer benchmark-token"

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Final snapshot contains `VERIFY_HOME_LOADED_12345` (returned to home).

### 12.2 Cross-page compare
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"

curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/articles.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Wiki snapshot contains `COUNT_LANGUAGES_12` AND articles snapshot contains article titles.

---

## Group 13: Form Validation

### 13.1 Required field
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/form.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#fullname","text":"Validator Test"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#submit-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot does NOT contain `VERIFY_FORM_SUBMITTED_SUCCESS` (submission blocked by validation).

### 13.2 Optional field
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/form.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#fullname","text":"No Phone User"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"fill","selector":"#email","text":"nophone@test.com"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"select","selector":"#country","value":"de"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"select","selector":"#subject","value":"feedback"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#submit-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `VERIFY_FORM_SUBMITTED_SUCCESS` AND `OPTIONAL_FIELD_SKIPPED_SUCCESS`.

---

## Group 14: Dynamic Content

### 14.1 Load more
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/ecommerce.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#load-more-btn"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `ADDITIONAL_PRODUCTS_LOADED`.

### 14.2 Lazy-loaded item
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#product-5 .add-to-cart"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `CART_UPDATED_WITH_LAZY_PRODUCT`.

---

## Group 15: Data Aggregation

### 15.1 Financial calc
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/dashboard.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Contains `$1,284,930` (revenue) AND `$384,930` (profit) AND `PROFIT_MARGIN_CALCULATED`.

### 15.2 Multi-page comparison
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki-go.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"

curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki-python.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"

curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/wiki-rust.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Go snapshot contains `FEATURE_COUNT_6` AND Python snapshot contains `FEATURE_COUNT_7` AND `COMPARISON_TABLE_BUILT` AND Rust snapshot contains `FEATURE_COUNT_5`.

---

## Group 16: Hover & Tooltips

### 16.1 Hover reveals info
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/hovers.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"hover","selector":"#avatar-1"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `HOVER_REVEALED_USER_1`.

### 16.2 Hover swap
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"hover","selector":"#avatar-2"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `HOVER_REVEALED_USER_2`.

---

## Group 17: Scrolling

### 17.1 Scroll by pixels
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/scroll.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"scroll","scrollY":1500}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `SCROLL_MIDDLE_MARKER`.

### 17.2 Scroll to footer
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"scroll","selector":"#footer"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=1500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `SCROLL_REACHED_FOOTER`.

---

## Group 18: File Download

### 18.1 Download a file
```bash
curl "http://localhost:9867/tabs/TAB_ID/download?url=http://fixtures/download-sample.txt" \
  -H "Authorization: Bearer benchmark-token" | \
  jq -r .data | base64 -d > /tmp/benchmark-download.txt

grep "DOWNLOAD_FILE_CONTENT_VERIFIED" /tmp/benchmark-download.txt
```
**Pass if**: File exists and contains `DOWNLOAD_FILE_CONTENT_VERIFIED`.

**Note**: The download endpoint returns JSON with `{contentType, data (base64), size, url}`, not the raw file. Decode `.data` to get the file contents. For PinchTab to download from internal hosts (like `fixtures`), the domain must be in `security.downloadAllowedDomains` config.

---

## Group 19: iFrame

### 19.1 Read iframe content
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/iframe.html"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=2000" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `IFRAME_INNER_CONTENT_LOADED`.

### 19.2 Type into iframe input
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/evaluate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"expression":"const f=document.getElementById(\"content-frame\").contentDocument; f.getElementById(\"iframe-input\").value=\"Hello World\"; f.getElementById(\"iframe-submit\").click(); f.getElementById(\"iframe-result\").textContent;"}'
```
**Pass if**: Response value contains `IFRAME_INPUT_RECEIVED_HELLO_WORLD`.

**Note**: PinchTab's `/action` selectors don't reach inside iframes. Use `/evaluate` with `contentDocument` to interact with iframe content.

---

## Group 20: Dialogs

### 20.1 Accept alert
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/alerts.html"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#alert-btn","dialogAction":"accept"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `DIALOG_ALERT_DISMISSED`.

### 20.2 Cancel confirm
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","selector":"#confirm-btn","dialogAction":"dismiss"}'

curl "http://localhost:9867/tabs/TAB_ID/snapshot?format=compact&maxTokens=500" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Snapshot contains `DIALOG_CONFIRM_CANCELLED`.

**Note**: Use `dialogAction` on the click action to auto-accept or auto-dismiss a JS dialog that the click opens. Without it, the click would hang until `/dialog` is called from a separate request.

---

## Group 21: Async / awaitPromise

### 21.1 Await a promise-returning function
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/async.html"}'

# Without awaitPromise: returns {} (unresolved Promise representation)
curl -X POST http://localhost:9867/tabs/TAB_ID/evaluate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.fetchPayload()","awaitPromise":true}'
```
**Pass if**: Response `.result` equals `"ASYNC_PAYLOAD_READY_42"`. Also verify that repeating the same call with `awaitPromise` omitted returns `{}` (proves `awaitPromise` is actually doing work).

### 21.2 Await a promise resolving to an object
```bash
curl -X POST http://localhost:9867/tabs/TAB_ID/evaluate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.fetchUser()","awaitPromise":true}'
```
**Pass if**: Response `.result.name` equals `"ASYNC_USER_NAME_ADA"` (structured value survives the round-trip).

**Note**: See SKILL.md and `references/api.md` — `awaitPromise:true` on `/evaluate` makes the server wait for the returned Promise to settle before responding. Omit the flag when you want the raw Promise reference.

---

## Group 22: Mouse Drag & Drop

### 22.1 Drag piece into Zone A (high-level `drag` action)
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"tabId":"TAB_ID","url":"http://fixtures/drag.html"}'

# Piece center starts at (92, 358); Zone A center is at (104, 200).
# Drag offset: (+12, -158).
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"drag","selector":"#piece","dragX":12,"dragY":-158}'

curl "http://localhost:9867/tabs/TAB_ID/text" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Page text contains `LAST_DROP=DROP_ZONE_A_OK`.

### 22.2 Low-level sequence: drag into Zone B, then Zone C
```bash
# Zone centers in viewport coords (for the default window size):
#   piece starts: (92, 358)
#   Zone A:       (104, 200)
#   Zone B:       (344, 200)
#   Zone C:       (584, 400)
# If the viewport differs, query these with /evaluate +
#   Array.from(['piece','zone-a','zone-b','zone-c']).map(id => ...getBoundingClientRect()).

# Drag piece (now at Zone A) to Zone B via explicit mouse-down → mouse-move → mouse-up.
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"mouse-move","x":104,"y":200}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"mouse-down","x":104,"y":200,"button":"left"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"mouse-move","x":344,"y":200}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"mouse-up","x":344,"y":200,"button":"left"}'

# Now drag Zone B → Zone C
curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"mouse-down","x":344,"y":200,"button":"left"}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"mouse-move","x":584,"y":400}'

curl -X POST http://localhost:9867/tabs/TAB_ID/action \
  -H "Authorization: Bearer benchmark-token" \
  -H "Content-Type: application/json" \
  -d '{"kind":"mouse-up","x":584,"y":400,"button":"left"}'

curl "http://localhost:9867/tabs/TAB_ID/text" \
  -H "Authorization: Bearer benchmark-token"
```
**Pass if**: Page text contains `DROP_SEQUENCE=DROP_ZONE_A_OK,DROP_ZONE_B_OK,DROP_ZONE_C_OK`.

**Note**: Either path (the high-level `drag` action with `dragX`/`dragY` offsets, or the explicit `mouse-down → mouse-move → mouse-up` sequence at absolute viewport coordinates) works. Use `drag` for simple point-to-point drags, and the low-level trio when the target site depends on intermediate `mousemove` events or when you need precise pacing.

---

## Summary

| Group | Tasks | Description |
|-------|-------|-------------|
| 0 | 8 | Setup & Diagnosis |
| 1 | 6 | Reading & Extracting |
| 2 | 3 | Search & Dynamic |
| 3 | 2 | Form |
| 4 | 3 | SPA |
| 5 | 2 | Login |
| 6 | 3 | E-commerce |
| 7 | 2 | Content + Interaction |
| 8 | 2 | Error Handling |
| 9 | 2 | Export |
| 10 | 2 | Modals |
| 11 | 2 | Persistence |
| 12 | 2 | Multi-page Nav |
| 13 | 2 | Form Validation |
| 14 | 2 | Dynamic Content |
| 15 | 2 | Data Aggregation |
| 16 | 2 | Hover & Tooltips |
| 17 | 2 | Scrolling |
| 18 | 1 | File Download |
| 19 | 2 | iFrame |
| 20 | 2 | Dialogs |
| 21 | 2 | Async / awaitPromise |
| 22 | 2 | Mouse Drag & Drop |

**Total: 58 tasks**

## Verification Strings

| Page | String |
|------|--------|
| Home | `VERIFY_HOME_LOADED_12345` |
| Articles | `VERIFY_ARTICLES_PAGE_67890` |
| Search | `VERIFY_SEARCH_PAGE_11111` |
| Form | `VERIFY_FORM_PAGE_22222` |
| Dashboard | `VERIFY_DASHBOARD_PAGE_33333` |
| Shop | `VERIFY_SHOP_PAGE_44444` |
| Wiki Index | `VERIFY_WIKI_INDEX_55555` |
| Login | `VERIFY_LOGIN_PAGE_77777` |
| Go Article | `VERIFY_WIKI_GO_LANG_88888` |
| SPA | `VERIFY_SPA_PAGE_99999` |
| Python Article | `VERIFY_WIKI_PYTHON_LANG` |
| Rust Article | `VERIFY_WIKI_RUST_LANG` |
| Async | `VERIFY_ASYNC_PAGE_77777` |
| Drag | `VERIFY_DRAG_PAGE_33333` |
