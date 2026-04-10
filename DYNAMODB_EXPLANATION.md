# DynamoDB Architecture - Simple Explanation

## The Problem We're Solving

You have a team of developers. They all need to use OpenAI, Anthropic, Google, etc. 

**Naive approach (INSECURE):**
```
Developer 1: Use sk-proj-my-openai-key-12345
Developer 2: Use sk-proj-my-openai-key-67890
Developer 3: Use sk-proj-my-openai-key-abcde

❌ Problem: Everyone has real keys. If someone leaks it, attacker has real key.
❌ Problem: Can't track who spent what on which provider.
❌ Problem: Can't revoke access without new keys to everyone.
```

**Our approach (SECURE):**
```
Developer 1: Use iw:abc123 (internal key)
Developer 2: Use iw:def456 (internal key)
Developer 3: Use iw:ghi789 (internal key)

(Proxy looks up: iw:abc123 → sk-proj-real-key)
(Proxy looks up: iw:def456 → sk-proj-another-real-key)
(Proxy looks up: iw:ghi789 → sk-proj-third-real-key)

✅ Developers never see real keys
✅ Can track who used what
✅ Can revoke access instantly (disable iw:abc123)
```

---

## DynamoDB Table Structure

Think of DynamoDB as a spreadsheet with rows. The table is: `llm-proxy-api-keys-dev`

### Example 1: Each User Has Their Own Key

```
┌─────────────────────────────────────────────────────────────────┐
│ DynamoDB Table: llm-proxy-api-keys-dev                         │
├─────────────────────────────────────────────────────────────────┤
│ pk (Primary Key)      │ provider  │ actual_key          │ description
├───────────────────────┼───────────┼─────────────────────┼──────────────
│ iw:abc123def456       │ openai    │ sk-proj-user1-key   │ User 1 OpenAI
│ iw:def456ghi789       │ openai    │ sk-proj-user2-key   │ User 2 OpenAI
│ iw:ghi789jkl012       │ anthropic │ sk-ant-user3-key    │ User 3 Anthropic
│ iw:jkl012mno345       │ gemini    │ user4-gemini-key    │ User 4 Gemini
└───────────────────────┴───────────┴─────────────────────┴──────────────

User 1:
  - Gets iw:abc123def456
  - Only they know this internal key
  - Proxy replaces it with: sk-proj-user1-key (OpenAI)
  - Cost tracked to: User 1

User 2:
  - Gets iw:def456ghi789
  - Only they know this internal key
  - Proxy replaces it with: sk-proj-user2-key (OpenAI)
  - Cost tracked to: User 2

(Each user has their own real API key)
```

### Example 2: Multiple Users Share One Key (But Still Tracked)

```
┌─────────────────────────────────────────────────────────────────┐
│ DynamoDB Table: llm-proxy-api-keys-dev                         │
├─────────────────────────────────────────────────────────────────┤
│ pk (Primary Key)      │ provider  │ actual_key              │ description
├───────────────────────┼───────────┼────────────────────────┼──────────────
│ iw:shared-openai-key  │ openai    │ sk-proj-COMPANY-KEY    │ Company OpenAI
│ iw:shared-anthropic   │ anthropic │ sk-ant-COMPANY-KEY     │ Company Anthropic
│ iw:shared-fireworks   │ openai    │ fw_X2nbzVNNw3aM5R14... │ Fireworks
└───────────────────────┴───────────┴────────────────────────┴──────────────

All users:
  - Use iw:shared-openai-key
  - Proxy replaces with: sk-proj-COMPANY-KEY
  
BUT STILL TRACKED PER USER:

Cost Tracking Table:
┌─────────────────────────────────────────────────────────────────┐
│ DynamoDB (or separate file)                                     │
├──────────┬──────────┬──────────┬─────────┬──────────────────────┤
│ userID   │ provider │ model    │ tokens  │ cost
├──────────┼──────────┼──────────┼─────────┼──────────────────────┤
│ user-1   │ openai   │ gpt4o    │ 1000    │ $0.05
│ user-2   │ openai   │ gpt4o    │ 2000    │ $0.10
│ user-3   │ anthropic│ claude   │ 500     │ $0.03
│ user-1   │ openai   │ gpt4o    │ 1500    │ $0.07
└──────────┴──────────┴──────────┴─────────┴──────────────────────┘

Result:
  user-1: $0.12 total (used OpenAI twice)
  user-2: $0.10 total (used OpenAI once)
  user-3: $0.03 total (used Anthropic once)
```

---

## Real World Example: Your Setup

### Current Situation (Your Company)

```
┌─────────────────────────────────────────────────────────────┐
│ Your Company Setup                                          │
├─────────────────────────────────────────────────────────────┤

Fireworks API Key: fw_X2nbzVNNw3aM5R14kK6fsm
  ↓ (stored securely in .env on appmotel)
  ↓ (accessed by proxy only)

DynamoDB (if enabled):
┌───────────────────────────────────────────────────────────┐
│ pk                    │ provider │ actual_key              │
├───────────────────────┼──────────┼────────────────────────┤
│ iw:company-fireworks  │ openai   │ fw_X2nbzVNNw3aM5R14... │
└───────────────────────┴──────────┴────────────────────────┘

Multiple users (You, Bob, Alice, etc.):
  1. User makes request: Authorization: Bearer iw:company-fireworks
  2. Proxy looks up: iw:company-fireworks → fw_X2nbzVNNw3aM5R14...
  3. Proxy forwards request with REAL key
  4. Cost tracked per user (if X-User-ID header sent)

Benefits:
  ✅ Only proxy knows the real key (fw_X2nbzVNNw3aM5R14...)
  ✅ If key leaks, it's on appmotel, not on dev machines
  ✅ Easy to rotate: Update DynamoDB, all users updated instantly
  ✅ Cost tracking per user (if you send X-User-ID)
```

---

## How AWS Services Track Users

### Option 1: Amazon (AWS Services like Bedrock)

```
Uses IAM Roles, NOT API keys:

┌─────────────────────────────────────────────────────────────┐
│ DynamoDB: llm-proxy-api-keys-dev                           │
│                                                             │
│ (NO entries - AWS uses IAM, not API keys)                  │
└─────────────────────────────────────────────────────────────┘

AWS Credentials (on appmotel server):
  ~/.aws/credentials (stored securely)
  ├── AWS_ACCESS_KEY_ID
  └── AWS_SECRET_ACCESS_KEY

How Bedrock knows who made request:
  1. Proxy uses AWS credentials to authenticate
  2. User sends: X-User-ID: user-123 (in header)
  3. Proxy records: userID=user-123, service=bedrock, cost=$X
  4. Cost tracked per userID in separate tracking table

Result:
  - AWS Bedrock doesn't know "user-123"
  - AWS only sees the proxy accessing with IAM role
  - But WE track which user spent what
```

### Option 2: Google (Gemini API)

```
┌─────────────────────────────────────────────────────────────┐
│ DynamoDB: llm-proxy-api-keys-dev                           │
├──────────────────────┬──────────┬─────────────────────────┤
│ pk                   │ provider │ actual_key              │
├──────────────────────┼──────────┼─────────────────────────┤
│ iw:company-gemini    │ gemini   │ AIzaSyDxxxxxxxx...      │
└──────────────────────┴──────────┴─────────────────────────┘

User makes request:
  POST /meta/user-bob/gemini/v1/...
  Authorization: Bearer iw:company-gemini

Proxy:
  1. Extracts userID: "user-bob"
  2. Looks up iw:company-gemini → AIzaSyDxxxxxxxx...
  3. Sends to Google with real key
  4. Records: userID=user-bob, provider=gemini, cost=$0.05

Result:
  - Google sees the proxy's API key (AIzaSyDxxxxxxxx...)
  - Google doesn't know about user-bob
  - We track user-bob's costs internally
```

### Option 3: OpenAI / Anthropic / Fireworks

```
Same as Google (API key based):

┌─────────────────────────────────────────────────────────────┐
│ DynamoDB: llm-proxy-api-keys-dev                           │
├──────────────────────┬──────────┬─────────────────────────┤
│ pk                   │ provider │ actual_key              │
├──────────────────────┼──────────┼─────────────────────────┤
│ iw:company-openai    │ openai   │ sk-proj-xxxxx...        │
│ iw:company-anthropic │ anthropic│ sk-ant-xxxxx...         │
│ iw:company-fireworks │ openai   │ fw_X2nbzVNNw3aM5R14... │
└──────────────────────┴──────────┴─────────────────────────┘

All external services:
  - See the proxy's key (sk-proj-*, sk-ant-*, fw_*)
  - Don't know who the actual user is
  - Charge the proxy/company for all usage

We track:
  - Who (userID) used what (provider/model)
  - When (timestamp)
  - How much (tokens, cost)
  - Stored in cost-tracking.jsonl or DynamoDB
```

---

## The Complete Picture: Where Each Key/Credential Lives

```
┌────────────────────────────────────────────────────────────────────┐
│ DEVELOPER'S LAPTOP                                                 │
├────────────────────────────────────────────────────────────────────┤
│ curl -H "Authorization: Bearer iw:company-openai" \               │
│   https://proxyved.../openai/v1/chat/completions                  │
│                                                                    │
│ ❌ Developer doesn't know: sk-proj-real-key                       │
│ ✅ Developer only knows: iw:company-openai (internal key)         │
└────────────────────────────────────────────────────────────────────┘
                          │
                          │ HTTPS (encrypted)
                          ▼
┌────────────────────────────────────────────────────────────────────┐
│ PROXY SERVER (appmotel)                                            │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│ 1. Receives: Authorization: Bearer iw:company-openai              │
│                                                                    │
│ 2. Looks up in DynamoDB:                                          │
│    iw:company-openai → sk-proj-real-key-xyz                      │
│                                                                    │
│ 3. Replaces header: Authorization: Bearer sk-proj-real-key-xyz    │
│                                                                    │
│ 4. Other secrets on disk:                                         │
│    ~/.aws/credentials (AWS keys)                                  │
│    .env (FIREWORKS_API_KEY=fw_X2nbz...)                          │
│                                                                    │
│ 5. Records in DynamoDB cost tracking:                             │
│    {userID, provider, model, tokens, cost}                        │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
                          │
                          │ HTTPS (encrypted)
                          ▼
┌────────────────────────────────────────────────────────────────────┐
│ EXTERNAL SERVICES (OpenAI, Google, AWS, Anthropic, Fireworks)    │
├────────────────────────────────────────────────────────────────────┤
│ Receives: Authorization: Bearer sk-proj-real-key-xyz              │
│                                                                    │
│ ❌ These services don't know: user details, company structure     │
│ ✅ These services know: proxy made a request, charged $X          │
│                                                                    │
│ They see request as coming from: single entity (the proxy)       │
│ They bill: the proxy/company (not individual users)              │
└────────────────────────────────────────────────────────────────────┘
```

---

## FAQ: Understanding This Architecture

### Q: "Do multiple users share ONE API key to OpenAI?"

**Answer: Yes, typically.**

Most companies:
- Buy ONE OpenAI API key
- Store it securely on proxy server
- All developers use `iw:company-openai`
- Proxy translates to real key
- OpenAI only sees: company-level usage
- We internally track: per-developer usage

Benefit: If one developer leaks `iw:company-openai`, attacker can't use it (proxy intercepts). The real key stays on appmotel.

---

### Q: "How does Amazon know which user made the request?"

**Answer: Amazon doesn't.**

AWS Bedrock:
- Uses IAM role to authenticate (no per-user key)
- Sees: "appmotel proxy" made the request
- Charges: the AWS account (company-level)
- Doesn't know: which user on the company made it

We track it:
- User sends: X-User-ID: alice header
- Proxy records: alice used Bedrock, cost $0.50
- We track: alice=$0.50, bob=$0.75, etc.
- AWS only sees: total company spending

---

### Q: "Where does each credential live?"

**Location of secrets:**

| Credential | Where Stored | Who Can Access | Security |
|-----------|-------------|----------------|----------|
| `iw:company-openai` | In your request | Developers/Users | Not secret, just internal routing key |
| `sk-proj-real-openai-key` | DynamoDB + .env | Proxy only | Secret, encrypted in transit & at rest |
| `fw_X2nbzVNNw3aM5R14...` | .env on appmotel | Proxy only | Secret, filesystem-only |
| `AWS_ACCESS_KEY_ID` | ~/.aws/credentials | Proxy only | Secret, IAM-protected |
| `userID` | Request header/URL | Everywhere (logged) | Not secret, just identifier |

---

### Q: "Why not just give each user their own OpenAI key?"

**Reasons to use shared key + proxy:**
1. **Cost aggregation** - See total spending per user
2. **Security** - Real keys never leak to client machines
3. **Rotation** - Change key once, all users updated instantly
4. **Audit** - Full record of who used what
5. **Rate limiting** - Enforce limits per-user, not per-key
6. **Monetization** - Charge users for API usage internally

**When to give each user their own key:**
- They're a contractor who should be independent
- They're outside the organization
- You want them managing their own cost

---

## Your Current Setup (Specifically)

```
┌───────────────────────────────────────────────────────────────┐
│ What You Have Right Now                                       │
├───────────────────────────────────────────────────────────────┤

1. Fireworks API Key:
   Location: ~/.env on appmotel
   Value: fw_X2nbzVNNw3aM5R14kK6fsm
   Access: Proxy only

2. Admin Interface (what we built):
   - Create internal keys: iw:xxxxx
   - Each maps to: fw_X2nbzVNNw3aM5R14...
   - Or map to other providers

3. Users:
   - You (vedikasheth) - using iw:xxxxx
   - Could add Bob, Alice, etc. - each gets own iw:yyyyy
   - They all use: same Fireworks key (fw_X2nbz...)
   - But costs tracked separately per user

4. DynamoDB:
   - Stores: iw:xxxxx ↔ fw_X2nbz... mappings
   - Stores: cost tracking (userID → cost)
   - Access: Proxy only

Example Scenario:
  You send:     Authorization: Bearer iw:93619c03a...
  Proxy looks:  iw:93619c03a... → fw_X2nbz...
  Proxy sends:  Authorization: Bearer fw_X2nbz... to Fireworks
  Fireworks sends response
  Proxy records: userID=vedikasheth, tokens=300, cost=$0.15
  
Result:
  ✅ You never see fw_X2nbz... (it's on appmotel)
  ✅ If your iw: key leaks, it's useless without proxy
  ✅ Costs tracked to you
  ✅ Can revoke your access instantly
```

---

## Summary

| Aspect | Answer |
|--------|--------|
| **Do users share API keys?** | Usually yes (one key per provider, shared by team) |
| **Are users tracked separately?** | Yes (via X-User-ID or URL path) |
| **Where do real credentials live?** | On proxy server only (DynamoDB, .env, ~/.aws/) |
| **What do external services see?** | Only the proxy's identity, not user details |
| **How do we track per-user costs?** | Cost tracking table (separate from key storage) |
| **Is DynamoDB for storing keys or tracking?** | Both: key mappings (iw: ↔ real key) + cost tracking |
| **How does AWS Bedrock work?** | IAM auth (no key), but still tracked per userID internally |

**Key Takeaway:** 
- Users see internal keys (`iw:`)
- Proxy knows real keys (stored securely)
- External services only see proxy's keys
- We track everything per-user internally
- Everyone benefits: security + cost transparency + easy management
