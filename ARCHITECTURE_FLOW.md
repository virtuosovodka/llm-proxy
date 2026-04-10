# LLM Proxy - Complete Architecture Flow

## 1. How Users Access the Proxy

```
┌─────────────────┐
│   End User      │
│   (Developer)   │
└────────┬────────┘
         │ Makes request with internal key
         │ Authorization: Bearer iw:xxxxx
         │ X-User-ID: user123 (or in URL path)
         ▼
┌──────────────────────────────────────┐
│  LLM Proxy (proxyved)                │
│  https://proxyved.hc-students.../    │
│  openai/v1/chat/completions          │
└──────────────────────────────────────┘
```

## 2. Request Processing Pipeline

```
Request comes in
    ↓
[1] URL Rewriting Middleware
    - Normalizes /meta/:userID/provider/ paths
    - Extracts userID from URL
    ↓
[2] API Key Validation Middleware
    - Reads Authorization: Bearer iw:xxxxx
    - Calls ValidateAndGetActualKey(iw:xxxxx)
    - REPLACES header with: Bearer sk-proj-real-key
    - Key never exposed to user
    ↓
[3] Logging Middleware
    - Logs request details
    - Records userID, provider, model
    ↓
[4] Rate Limiting Middleware
    - Checks user's rate limit
    - Enforces rpm/tpm/rpd/tpd limits
    ↓
[5] Token Parsing Middleware
    - Estimates tokens from request
    - Calls cost tracking callback
    - Collects stats for dashboard
    ↓
[6] Streaming Middleware
    - Detects streaming mode
    ↓
[7] Provider Handler
    - Routes to OpenAI/Anthropic/Gemini/Bedrock
    - Sends REAL key (not iw:)
    ↓
[8] Response Processing
    - Parses response metadata (tokens, costs)
    - Tracks in DynamoDB & Datadog
    - Updates dashboard stats
    ↓
Response sent back to user
```

## 3. The Key Mapping System (No credentials exposed)

```
┌─────────────────────────────────┐
│ Admin Interface                 │
│ Create key mapping:             │
│                                 │
│ iw:93619c03a2c89b01aea... ──┐   │
│       (Internal Key)         │   │
│                              ├──→ DynamoDB Table
│ sk-proj-actual-key-xyz ──────┘   │ (llm-proxy-api-keys-dev)
│       (Actual API Key)            │
│                                 │
│ Provider: openai              │
│ Daily Limit: $100             │
│ Status: Enabled               │
└─────────────────────────────────┘

User sends: Authorization: Bearer iw:93619c03a2c89b01aea...
Proxy looks up: iw:93619c03a2c89b01aea... → sk-proj-actual-key-xyz
Proxy sends to OpenAI: Authorization: Bearer sk-proj-actual-key-xyz

✅ User only knows: iw:93619c03a2c89b01aea...
✅ User never sees: sk-proj-actual-key-xyz
✅ DynamoDB stores both (backend only)
```

## 4. User Tracking & Cost Attribution

```
Request arrives:
    ↓
Extract userID from (in priority order):
  1. Context value (set by middleware)
  2. URL path: /meta/{userID}/openai/v1/...
  3. Header: X-User-ID: user123
  4. Query param: ?llm_user_id=user123
  5. Provider-specific extraction
  6. Default: "unknown"
    ↓
Record in cost tracking:
{
  userID: "user123",          ← Who made the request
  provider: "openai",         ← Which provider
  model: "gpt-4o-mini",       ← Which model
  inputTokens: 150,
  outputTokens: 450,
  totalTokens: 600,
  costCents: 45,              ← $0.45
  timestamp: 2026-04-10T12:34:56Z,
  ipAddress: "203.0.113.42",  ← Request source
  path: "/meta/user123/openai/v1/chat/completions"
}
    ↓
Stored in:
  - File: ./logs/cost-tracking.jsonl
  - DynamoDB: (if configured)
  - Datadog: (if configured)
```

## 5. How Each User Gets Their Own Key

**Option A: Dedicated key per user (Recommended)**
```
User A:
  - Gets internal key: iw:aaaaa...
  - Maps to OpenAI key: sk-proj-user-a-key
  - Can only use their own key
  - Cost tracked to user A

User B:
  - Gets internal key: iw:bbbbb...
  - Maps to OpenAI key: sk-proj-user-b-key
  - Can only use their own key
  - Cost tracked to user B

Admin Interface:
  - Create iw:aaaaa → sk-proj-user-a-key
  - Create iw:bbbbb → sk-proj-user-b-key
  - Both stored in DynamoDB
  - Each user only authorized to use their own iw: key
```

**Option B: Shared key with user tracking (Current)**
```
All users share one Fireworks key:
  - Internal key: iw:shared-key...
  - Maps to: fw_X2nbzVNNw3aM5R14kK6fsm
  - Multiple users use same key
  - Cost still tracked per user via X-User-ID / URL path

Requests:
  User A: /meta/user-a/openai/v1/... → userID = user-a
  User B: /meta/user-b/openai/v1/... → userID = user-b
  
Cost tracking:
  user-a: $10.50 total
  user-b: $8.25 total
  (Even though they share the same key)
```

## 6. AWS Bedrock Integration

```
┌─────────────────────────────────────────────┐
│ Bedrock Access                              │
├─────────────────────────────────────────────┤
│ Uses AWS IAM (not API keys)                 │
│ AWS Credentials loaded from:                │
│   ~/.aws/credentials                        │
│   ~/.aws/config                             │
│                                             │
│ Profile precedence:                         │
│   1. AWS_PROFILE env var                    │
│   2. [bedrock] profile                      │
│   3. [default] profile                      │
│                                             │
│ User tracking still works:                  │
│   - X-User-ID header                        │
│   - /meta/{userID}/bedrock/...              │
│   - Cost attributed to userID               │
└─────────────────────────────────────────────┘

Example:
  Request: POST /meta/user123/bedrock/v1/messages
           X-User-ID: user123
  
  Proxy:
    1. Identifies provider: bedrock
    2. Extracts userID: user123
    3. Uses AWS IAM credentials (no API key)
    4. Sends to AWS Bedrock
    5. Records cost to user123 in DynamoDB
```

## 7. Complete Data Flow Example

```
┌─────────────────────────────────────────────────────────────┐
│ USER MAKES REQUEST                                          │
└─────────────────────────────────────────────────────────────┘

curl -X POST \
  https://proxyved.../meta/team-research/openai/v1/chat/completions \
  -H "Authorization: Bearer iw:93619c03a2c89b01aea495cc3413e3f4" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ PROXY PROCESSING                                            │
├─────────────────────────────────────────────────────────────┤
│ 1. Extract userID from URL: "team-research"               │
│ 2. Extract key from header: "iw:93619c03a2c89b01aea..."  │
│ 3. Look up in DynamoDB:                                    │
│    iw:93619c03a2c89b01aea... → sk-proj-actual-key         │
│ 4. Verify key is enabled and not expired                   │
│ 5. Replace header: Authorization: Bearer sk-proj-...      │
│ 6. Forward to OpenAI with REAL key                         │
│ 7. Receive response: 300 tokens                            │
│ 8. Record to cost tracking:                                │
│    {                                                        │
│      userID: "team-research",                              │
│      provider: "openai",                                    │
│      model: "gpt-4o-mini",                                 │
│      tokens: 300,                                           │
│      cost: $0.15,                                           │
│      timestamp: now                                         │
│    }                                                        │
│ 9. Update dashboard stats                                  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ STORED RECORDS                                              │
├─────────────────────────────────────────────────────────────┤
│ DynamoDB - llm-proxy-api-keys-dev:                          │
│   Key: iw:93619c03a2c89b01aea495cc3413e3f4                │
│   ActualKey: sk-proj-actual-key                            │
│   Provider: openai                                          │
│   Enabled: true                                             │
│   DailyCostLimit: $100                                      │
│                                                             │
│ DynamoDB - cost tracking (if configured):                   │
│   userID: team-research                                     │
│   provider: openai                                          │
│   model: gpt-4o-mini                                        │
│   tokens: 300                                               │
│   cost: 15 (cents)                                          │
│   timestamp: 2026-04-10T12:34:56Z                           │
│                                                             │
│ Dashboard (in-memory):                                      │
│   Request added to stats                                    │
│   Graph updated                                             │
│   Recent requests table updated                             │
│                                                             │
│ Datadog (if configured):                                    │
│   Metric: llm.requests.total                                │
│   Tag: userID=team-research                                 │
│   Tag: provider=openai                                      │
│   Tag: model=gpt-4o-mini                                    │
└─────────────────────────────────────────────────────────────┘
```

## 8. Summary: Security & Tracking

| Aspect | Implementation |
|--------|-----------------|
| **Credentials** | Stored in DynamoDB, never exposed to users |
| **User Identity** | Extracted from URL path or headers per request |
| **Cost Tracking** | Per-user attribution (shared keys still tracked individually) |
| **Rate Limiting** | Per-user rate limits enforced |
| **Audit Trail** | All requests logged with userID, provider, model, tokens, cost |
| **AWS Bedrock** | Uses IAM credentials, not API keys; same user tracking applies |
| **Isolation** | Each user gets own `iw:` key OR shared key with per-user cost tracking |

## 9. FAQ

**Q: Can two users share one API key?**
A: Yes! Use the same `iw:xxxxx` key, but send different `X-User-ID` headers. Costs tracked separately.

**Q: How does Bedrock know which user made the request?**
A: Via `X-User-ID` header or `/meta/{userID}/bedrock/...` URL path. IAM credentials are shared, but user tracking is separate.

**Q: What if a user knows someone else's `iw:` key?**
A: They can use it. This is intentional - if you want user isolation, give each user their own `iw:` key in the admin UI.

**Q: Are credentials ever logged?**
A: No. Only the `iw:` prefixed keys appear in logs. Real keys (sk-proj-*, fw_*, etc.) are replaced before forwarding.

**Q: Can I track per-team costs?**
A: Yes! Use userID like `team-research`, `team-engineering`, etc. Costs aggregated per userID.
