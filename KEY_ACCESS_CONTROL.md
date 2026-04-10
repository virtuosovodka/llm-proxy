# API Key Access Control - Who Uses Which Key?

## The Question You're Asking

"I have a Fireworks key. Should everyone use it? Or specific people? Or different groups?"

**Answer: You decide based on your organization structure and business needs.**

---

## Strategy 1: Everyone Shares ONE Key (Simplest)

```
┌──────────────────────────────────────────────────────┐
│ Your Company                                         │
├──────────────────────────────────────────────────────┤
│                                                      │
│ Fireworks API Key: fw_X2nbz...                      │
│ (stored on proxy, never shared)                     │
│                                                      │
│ Internal Key: iw:company-fireworks                  │
│ (given to everyone)                                 │
│                                                      │
│ Users:                                               │
│ ├─ Alice: uses iw:company-fireworks ✓              │
│ ├─ Bob: uses iw:company-fireworks ✓                │
│ ├─ Charlie: uses iw:company-fireworks ✓            │
│ └─ Diana: uses iw:company-fireworks ✓              │
│                                                      │
│ DynamoDB:                                            │
│ ┌────────────────────────────────────────────────┐ │
│ │ iw:company-fireworks → fw_X2nbz...            │ │
│ └────────────────────────────────────────────────┘ │
│                                                      │
│ Cost Tracking:                                       │
│ ┌────────────────────────────────────────────────┐ │
│ │ Alice: $10.50 (tracked via X-User-ID)        │ │
│ │ Bob: $8.25                                     │ │
│ │ Charlie: $15.00                                │ │
│ │ Diana: $6.75                                   │ │
│ │ Total: $40.50                                  │ │
│ └────────────────────────────────────────────────┘ │
│                                                      │
└──────────────────────────────────────────────────────┘

✅ Pros:
  - Simple, everyone has access
  - Easy to manage (one key)
  - Good for small teams

❌ Cons:
  - No access control
  - Can't revoke access to one person without affecting all
  - Everyone can see/use the same key
```

---

## Strategy 2: Different Keys Per Team/Department

```
┌──────────────────────────────────────────────────────┐
│ Your Company (with departments)                      │
├──────────────────────────────────────────────────────┤
│                                                      │
│ Fireworks Account 1: fw_team-research-xxxx          │
│ Fireworks Account 2: fw_team-engineering-yyyy       │
│ OpenAI Account 1:    sk-proj-team-research-zzz      │
│ OpenAI Account 2:    sk-proj-team-engineering-www   │
│                                                      │
│ DynamoDB (Key Mappings):                             │
│ ┌──────────────────┬──────────────────────────────┐ │
│ │ iw:team-research │ → fw_team-research-xxxx      │ │
│ │ iw:team-research │ → sk-proj-team-research-zzz  │ │
│ │ iw:team-eng      │ → fw_team-engineering-yyyy   │ │
│ │ iw:team-eng      │ → sk-proj-team-engineering.. │ │
│ └──────────────────┴──────────────────────────────┘ │
│                                                      │
│ Access Control:                                      │
│ ├─ Research Team:                                   │
│ │  ├─ Alice: iw:team-research ✓                    │
│ │  └─ Bob: iw:team-research ✓                      │
│ │                                                   │
│ └─ Engineering Team:                                │
│    ├─ Charlie: iw:team-eng ✓                       │
│    └─ Diana: iw:team-eng ✓                         │
│                                                      │
│ Cost Tracking:                                       │
│ ┌────────────────────────────────────────────────┐ │
│ │ Research Team Total: $25.00                    │ │
│ │   ├─ Alice: $10.50                             │ │
│ │   └─ Bob: $14.50                               │ │
│ │ Engineering Team Total: $15.50                 │ │
│ │   ├─ Charlie: $8.00                            │ │
│ │   └─ Diana: $7.50                              │ │
│ └────────────────────────────────────────────────┘ │
│                                                      │
└──────────────────────────────────────────────────────┘

✅ Pros:
  - Team-level isolation
  - Can track costs per team
  - Can rotate keys per team without affecting others
  - Can give different limits per team

❌ Cons:
  - More keys to manage
  - Need separate accounts per provider per team
  - More complex setup
```

---

## Strategy 3: Different Keys Per Individual (Most Control)

```
┌──────────────────────────────────────────────────────┐
│ Your Company (with individual keys)                  │
├──────────────────────────────────────────────────────┤
│                                                      │
│ DynamoDB (Key Mappings):                             │
│ ┌──────────────────┬──────────────────────────────┐ │
│ │ iw:alice-fwork   │ → fw_alice-key-12345         │ │
│ │ iw:alice-openai  │ → sk-proj-alice-key-67890    │ │
│ │ iw:bob-fwork     │ → fw_bob-key-abcde           │ │
│ │ iw:bob-openai    │ → sk-proj-bob-key-fghij      │ │
│ │ iw:charlie-fwork │ → fw_charlie-key-klmno       │ │
│ │ iw:diana-fwork   │ → fw_diana-key-pqrst         │ │
│ └──────────────────┴──────────────────────────────┘ │
│                                                      │
│ Access Control:                                      │
│ ├─ Alice: iw:alice-fwork, iw:alice-openai ✓        │
│ ├─ Bob: iw:bob-fwork, iw:bob-openai ✓              │
│ ├─ Charlie: iw:charlie-fwork ✓                     │
│ └─ Diana: iw:diana-fwork ✓                         │
│    (Diana doesn't have access to OpenAI)           │
│                                                      │
│ Cost Tracking:                                       │
│ ┌────────────────────────────────────────────────┐ │
│ │ Alice: $18.50 (Fireworks + OpenAI)            │ │
│ │ Bob: $12.25 (Fireworks + OpenAI)              │ │
│ │ Charlie: $5.00 (Fireworks only)               │ │
│ │ Diana: $4.75 (Fireworks only)                 │ │
│ └────────────────────────────────────────────────┘ │
│                                                      │
└──────────────────────────────────────────────────────┘

✅ Pros:
  - Maximum control
  - Fine-grained access (Alice has OpenAI, Diana doesn't)
  - Can disable one person's access instantly
  - Perfect audit trail per individual
  - Can set different daily limits per person

❌ Cons:
  - Lots of keys to manage
  - Need separate API account per person per provider
  - Complex to scale to many people
  - Expensive if many paid APIs
```

---

## Strategy 4: Hybrid (Recommended for Most Companies)

```
┌──────────────────────────────────────────────────────┐
│ Best of Both Worlds                                  │
├──────────────────────────────────────────────────────┤
│                                                      │
│ Shared Company Keys:                                 │
│ ├─ iw:company-openai → sk-proj-company-key         │
│ ├─ iw:company-anthropic → sk-ant-company-key       │
│ └─ iw:company-fireworks → fw_company-key           │
│                                                      │
│ Team-Specific Keys (for high-value teams):          │
│ ├─ iw:research-gemini → specific-research-key      │
│ └─ iw:eng-bedrock → specific-eng-key               │
│                                                      │
│ Individual Keys (for contractors/external):         │
│ ├─ iw:alice-personal → alice-own-key               │
│ └─ iw:contractor-bob → bob-contractor-key          │
│                                                      │
│ DynamoDB:                                            │
│ ┌──────────────────┬──────────────────────────────┐ │
│ │ iw:company-openai    │ → sk-proj-company-key    │ │
│ │ iw:research-gemini   │ → specific-research-key  │ │
│ │ iw:alice-personal    │ → alice-own-key          │ │
│ └──────────────────┴──────────────────────────────┘ │
│                                                      │
│ Access Control:                                      │
│ ├─ Default (everyone):                              │
│ │  Use: iw:company-openai, iw:company-anthropic   │
│ ├─ Research Team:                                   │
│ │  Use: iw:company-openai + iw:research-gemini    │
│ ├─ Alice (contractor):                              │
│ │  Use: iw:alice-personal (only)                  │
│ └─ Bob (contractor):                                │
│    Use: iw:contractor-bob (only)                   │
│                                                      │
└──────────────────────────────────────────────────────┘

✅ Pros:
  - Flexibility for different scenarios
  - Shared keys for most people (simple)
  - Special keys for teams/individuals who need it
  - Easy to add/remove access
  - Scales well

❌ Cons:
  - Some complexity in managing multiple key types
```

---

## How to Decide: Decision Matrix

| Scenario | Strategy | Reason |
|----------|----------|--------|
| **Small team (< 10 people), all trusted** | Strategy 1 (Shared) | Simple, costs tracked per user anyway |
| **Multiple departments with separate budgets** | Strategy 2 (Per Team) | Separate bills, separate rate limits |
| **Contractors/external access** | Strategy 3 (Per Individual) | Tight control, can revoke instantly |
| **Large company, mixed needs** | Strategy 4 (Hybrid) | Most flexibility |
| **High security environment** | Strategy 3 (Per Individual) | Maximum audit trail, granular control |
| **Cost-focused, optimize API usage** | Strategy 2 (Per Team) | Team-level budgets, rate limiting |

---

## Implementation: How to Set It Up (Admin Interface)

### Setting Up Shared Key (Everyone)

```bash
# Create one internal key
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "actual_key": "sk-proj-company-key",
    "description": "Company OpenAI (everyone)",
    "daily_cost_limit": 500000  # $5000/day for whole company
  }'

# Everyone uses the same internal key they're given
# Authorization: Bearer iw:xxxxx (from response)
```

### Setting Up Team Keys

```bash
# Research team gets their own key
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "gemini",
    "actual_key": "AIzaSyD-research-specific-key",
    "description": "Gemini for Research Team",
    "daily_cost_limit": 100000,  # $1000/day for research
    "tags": {"team": "research"}
  }'

# Engineering team gets their own key
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "bedrock",
    "actual_key": "aws-bedrock-engineering-key",
    "description": "Bedrock for Engineering Team",
    "daily_cost_limit": 200000,  # $2000/day for engineering
    "tags": {"team": "engineering"}
  }'
```

### Setting Up Individual Keys

```bash
# Alice gets her own contractor key
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "actual_key": "sk-proj-alice-contractor-key",
    "description": "Alice (Contractor)",
    "daily_cost_limit": 10000,  # $100/day limit for alice
    "tags": {"type": "contractor", "name": "alice"}
  }'

# Later, when contract ends - just disable
curl -X PUT 'https://proxyved.../admin/api/keys/iw:alice-key-id?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Alice can no longer use it!
```

---

## Real-World Example: Your Company

### Your Current Situation

```
You have:
  - Fireworks API Key (fw_X2nbz...)
  - No other external services set up yet

Question: Who should use this key?

Recommendation:
  Start with Strategy 1 (Shared):
  1. Create iw:company-fireworks → fw_X2nbz...
  2. Give to: yourself (vedikasheth) + team members
  3. Track costs per person using X-User-ID
  
  Later, as you grow:
  1. Add OpenAI key for everyone (Strategy 1)
  2. If research team needs Gemini, add iw:research-gemini (Strategy 2)
  3. If you hire contractors, add individual keys (Strategy 3)
```

---

## How Users Request Access

### Option A: Admin Grants Keys (Centralized)

```
1. Team member asks: "I need access to OpenAI"
2. Admin goes to: https://proxyved.../admin
3. Admin creates: iw:team-member-name-openai
4. Admin gives them: iw:team-member-name-openai
5. User starts using it

Setup in code:
  Authorization: Bearer iw:team-member-name-openai
```

### Option B: Self-Service Portal (Future Enhancement)

```
Future feature:
  1. User visits: https://proxyved.../request-key
  2. User fills form: "Need access to OpenAI"
  3. System auto-creates: iw:username-openai
  4. User gets key immediately
  5. Uses it right away

This requires building a request/approval workflow
(not currently built, but possible)
```

---

## Security Considerations

### What Users See vs What They Don't

| What User Sees | What User Doesn't See |
|----------------|----------------------|
| Their internal key: `iw:xxxxx` | Real API key: `sk-proj-...` |
| Their usage: costs tracked to them | Other users' real keys |
| Their daily limit: $100 | Company's total spending |
| | Proxy's AWS credentials |

### Revoking Access

```
User leaves company?

Option 1: Disable their key
  curl -X PUT '.../admin/api/keys/iw:alice-key?token=admin...' \
    -d '{"enabled": false}'
  → Alice's key stops working immediately

Option 2: Delete their key
  curl -X DELETE '.../admin/api/keys/iw:alice-key?token=admin...'
  → Their key is gone, can't be recovered

Option 3: Rotate company key
  (if everyone used shared key)
  1. Create new: iw:company-openai-v2 → sk-proj-new-key
  2. Tell everyone to use new key
  3. Delete old: iw:company-openai-v1
  → Old key stops working
```

---

## Summary Table: Quick Reference

| Aspect | Strategy 1 | Strategy 2 | Strategy 3 | Strategy 4 |
|--------|-----------|-----------|-----------|-----------|
| **Shared Key** | Yes (one) | Yes (one per team) | No | Partial |
| **Team Access** | Everyone | Group | Individual | Mixed |
| **Cost per** | User | Team | User | Both |
| **Revoke Access** | Hard | Medium | Easy | Easy |
| **Complexity** | Low | Medium | High | Medium |
| **Best For** | Small teams | Departments | Contractors | Growing companies |
| **Keys to Manage** | 1 | 3-5 | 10+ | 5-8 |

---

## What You Should Do Now (For Your Setup)

```
Current: You have Fireworks key

Next Steps:
1. Create one internal key: iw:company-fireworks
   - Give to everyone who should use it
   
2. When you add more providers (OpenAI, Google, etc.):
   - Create iw:company-openai
   - Create iw:company-gemini
   - Give to team
   
3. If someone needs special access:
   - Create iw:team-research-gemini (for research)
   - Give only to research team
   
4. If contractor joins:
   - Create iw:contractor-name-openai
   - Set daily limit
   - Can disable when they leave

This is Strategy 4 (Hybrid) - simple to start, grows with you!
```
