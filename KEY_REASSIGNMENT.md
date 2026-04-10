# Key Reassignment & Mass Changes Guide

## Your Scenario

```
Current Setup:
  All users → iw:xxx → fw_fireworks (one shared key)

New Setup:
  Department A & B → iw:xxx → fw_fireworks-ab (shared)
  Department C & D → iw:yyy → fw_fireworks-cd (shared)

Question: Can I do this without changing the Fireworks key itself?
Answer: YES! ✅
```

---

## The Solution: Two Approaches

### Approach 1: ONE Backend Key, Multiple Frontend Keys (Easiest) 🎯

Keep the SAME Fireworks key, but reorganize frontend keys:

```
Current DynamoDB:
  iw:alice           → fw_X2nbzVNNw3aM5R14kK6fsm
  iw:bob             → fw_X2nbzVNNw3aM5R14kK6fsm
  iw:charlie         → fw_X2nbzVNNw3aM5R14kK6fsm
  iw:diana           → fw_X2nbzVNNw3aM5R14kK6fsm

New DynamoDB:
  iw:alice           → fw_X2nbzVNNw3aM5R14kK6fsm  (same key!)
  iw:bob             → fw_X2nbzVNNw3aM5R14kK6fsm  (same key!)
  iw:charlie         → fw_X2nbzVNNw3aM5R14kK6fsm  (same key!)
  iw:diana           → fw_X2nbzVNNw3aM5R14kK6fsm  (same key!)

✅ NO CHANGE to actual Fireworks key
✅ Just reorganizing who uses what frontend key
✅ Cost tracking works exactly the same
```

**Bottom line:** Just keep all users on the same Fireworks key. They already ARE. No changes needed!

---

### Approach 2: Department-Level Grouping (If You Want Separate Tracking)

```
Current:
  All → fw_X2nbzVNNw3aM5R14kK6fsm

New (Optional):
  Dept A & B → fw_X2nbzVNNw3aM5R14kK6fsm (same key, different groups)
  Dept C & D → fw_X2nbzVNNw3aM5R14kK6fsm (same key, different groups)

✅ Still same backend key
✅ Just tag different users with different frontend keys
✅ Track costs by department
```

**How this looks:**

```
DynamoDB:
┌──────────────────┬──────────┬────────────────────────────┐
│ iw: Key          │ Provider │ Actual Key                 │
├──────────────────┼──────────┼────────────────────────────┤
│ iw:dept-a-b-key  │ openai   │ fw_X2nbzVNNw3aM5R14kK6fsm │
│ iw:dept-c-d-key  │ openai   │ fw_X2nbzVNNw3aM5R14kK6fsm │
└──────────────────┴──────────┴────────────────────────────┘

Users:
  Alice (Dept A): use iw:dept-a-b-key
  Bob (Dept B):   use iw:dept-a-b-key
  Charlie (Dept C): use iw:dept-c-d-key
  Diana (Dept D): use iw:dept-c-d-key

Cost Tracking:
  Dept A+B Total: Alice $10 + Bob $8 = $18
  Dept C+D Total: Charlie $12 + Diana $5 = $17
  All use SAME backend: fw_X2nbzVNNw3aM5R14kK6fsm
```

---

## How to Do This: Step By Step

### Option A: No Mass Change Needed (Just Give Different Frontend Keys)

```bash
# Create department-level frontend keys (all point to same backend)

# Department A & B share this key
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "actual_key": "fw_X2nbzVNNw3aM5R14kK6fsm",
    "description": "Dept A & B Shared Access",
    "daily_cost_limit": 200000,
    "tags": {"department": "a-b"}
  }'
# Give returned iw:xxxxx to Alice and Bob

# Department C & D share this key
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "actual_key": "fw_X2nbzVNNw3aM5R14kK6fsm",
    "description": "Dept C & D Shared Access",
    "daily_cost_limit": 150000,
    "tags": {"department": "c-d"}
  }'
# Give returned iw:yyyyy to Charlie and Diana
```

**Result:**
- No changes to Fireworks key
- Department A & B use iw:xxxxx
- Department C & D use iw:yyyyy
- Same backend key for everyone
- Easy cost tracking per department

---

### Option B: Update Existing Keys (If Already Set Up)

If you already have individual keys and want to consolidate:

```bash
# Don't delete old keys, just create new department-level keys
# and update who uses what

# BEFORE:
#   iw:alice → fw_X2nbz...
#   iw:bob → fw_X2nbz...
#   iw:charlie → fw_X2nbz...
#   iw:diana → fw_X2nbz...

# AFTER:
#   iw:alice → still works (no change)
#   iw:bob → still works (no change)
#   iw:charlie → still works (no change)
#   iw:diana → still works (no change)
#   
#   PLUS: New department keys
#   iw:dept-a-b → fw_X2nbz...
#   iw:dept-c-d → fw_X2nbz...

# Users switch to department key:
#   Alice: use iw:dept-a-b (instead of iw:alice)
#   Bob: use iw:dept-a-b (instead of iw:bob)
#   ...

# Later, delete old individual keys:
curl -X DELETE 'https://proxyved.../admin/api/keys/iw:alice-key-id?token=admin-prod-2024'
```

---

## Mass Change: Bulk Operations (Best Way)

### Scripted Approach (If Many Users)

```bash
#!/bin/bash
# bulk-update-keys.sh

TOKEN="admin-prod-2024"
API="https://proxyved.hc-students.osu.internetchen.de"
BACKEND_KEY="fw_X2nbzVNNw3aM5R14kK6fsm"

# Department A & B group
curl -s -X POST "$API/admin/api/keys?token=$TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"provider\": \"openai\",
    \"actual_key\": \"$BACKEND_KEY\",
    \"description\": \"Dept A & B Group\",
    \"daily_cost_limit\": 200000,
    \"tags\": {\"group\": \"a-b\"}
  }" | jq '.pk' # Returns iw:xxxxx

# Department C & D group
curl -s -X POST "$API/admin/api/keys?token=$TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"provider\": \"openai\",
    \"actual_key\": \"$BACKEND_KEY\",
    \"description\": \"Dept C & D Group\",
    \"daily_cost_limit\": 150000,
    \"tags\": {\"group\": \"c-d\"}
  }" | jq '.pk' # Returns iw:yyyyy

echo "Done! All departments use same backend key: $BACKEND_KEY"
```

**Run it:**
```bash
bash bulk-update-keys.sh

# Output:
# "iw:93619c03a2c89b01aea495cc3413e3f4"
# "iw:def456ghi789jkl012mno345"
# Done! All departments use same backend key: fw_X2nbzVNNw3aM5R14kK6fsm
```

---

## Web UI Alternative (One-by-One, But Visual)

If you prefer the web interface for fewer keys:

```
1. Go to: https://proxyved.hc-students.osu.internetchen.de/admin?token=admin-prod-2024

2. Click "Create New Key"

3. Fill in:
   Provider: openai
   API Key: fw_X2nbzVNNw3aM5R14kK6fsm
   Description: Dept A & B Shared
   Daily Limit: $2000
   Tags: department=a-b

4. Click Create

5. Repeat for Dept C & D

6. Share the iw: keys with respective departments
```

---

## Complete Example: Your Setup

### Current State
```
You have: fw_X2nbzVNNw3aM5R14kK6fsm (Fireworks)

Users:
  Alice (Dept A)
  Bob (Dept B)
  Charlie (Dept C)
  Diana (Dept D)
```

### Mass Change Setup (2 Keys Instead of 4)

```bash
# Step 1: Create Dept A & B key
KEY_AB=$(curl -s -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "actual_key": "fw_X2nbzVNNw3aM5R14kK6fsm",
    "description": "Department A & B",
    "daily_cost_limit": 200000
  }' | jq -r '.pk')

echo "Department A & B Key: $KEY_AB"
# Output: Department A & B Key: iw:93619c03a2c89b01aea495cc3413e3f4

# Step 2: Create Dept C & D key
KEY_CD=$(curl -s -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "actual_key": "fw_X2nbzVNNw3aM5R14kK6fsm",
    "description": "Department C & D",
    "daily_cost_limit": 150000
  }' | jq -r '.pk')

echo "Department C & D Key: $KEY_CD"
# Output: Department C & D Key: iw:def456ghi789jkl012mno345

# Done! Now distribute keys:
echo ""
echo "=== Distribution ==="
echo "Tell Alice & Bob to use: $KEY_AB"
echo "Tell Charlie & Diana to use: $KEY_CD"
echo ""
echo "All using backend: fw_X2nbzVNNw3aM5R14kK6fsm"
```

### Result
```
DynamoDB after changes:
┌──────────────────────────────────┬──────────┬────────────────────────────┐
│ iw: Key                          │ Provider │ Actual Key                 │
├──────────────────────────────────┼──────────┼────────────────────────────┤
│ iw:93619c03a2c89b01aea495...     │ openai   │ fw_X2nbzVNNw3aM5R14kK6fsm │
│ iw:def456ghi789jkl012mno345      │ openai   │ fw_X2nbzVNNw3aM5R14kK6fsm │
└──────────────────────────────────┴──────────┴────────────────────────────┘

Cost Tracking:
  Alice (uses iw:93619c03a...):    $10.50
  Bob (uses iw:93619c03a...):      $8.25
  Subtotal A+B:                    $18.75

  Charlie (uses iw:def456gh...):   $12.00
  Diana (uses iw:def456gh...):     $5.50
  Subtotal C+D:                    $17.50

  TOTAL:                           $36.25
  All from backend:                fw_X2nbzVNNw3aM5R14kK6fsm (ONE key!)
```

---

## Summary

| Question | Answer |
|----------|--------|
| **Can I keep same Fireworks key?** | ✅ Yes, no changes needed |
| **Can I reorganize departments?** | ✅ Yes, just create new iw: keys |
| **Do I need to change the backend key?** | ❌ No, never touch fw_X2nbz... |
| **Can I do this in bulk?** | ✅ Yes, with script or CLI |
| **Can I do it one-by-one?** | ✅ Yes, via web UI |
| **Will it break anything?** | ❌ No, completely safe |
| **Can I undo it?** | ✅ Yes, just delete/disable old keys |

**Best Practice:**
1. Create new department-level iw: keys (all pointing to same fw_)
2. Tell departments to switch
3. Delete old individual iw: keys
4. Done! Same backend, reorganized frontend, easy tracking

---

## One-Line Recommendation

You already have what you need. Just create 2 new keys (one per department pair) pointing to the same Fireworks key, and you're done!

```bash
# Dept A & B
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' -H "Content-Type: application/json" -d '{"provider":"openai","actual_key":"fw_X2nbzVNNw3aM5R14kK6fsm","description":"Dept A & B","daily_cost_limit":200000}'

# Dept C & D
curl -X POST 'https://proxyved.../admin/api/keys?token=admin-prod-2024' -H "Content-Type: application/json" -d '{"provider":"openai","actual_key":"fw_X2nbzVNNw3aM5R14kK6fsm","description":"Dept C & D","daily_cost_limit":150000}'
```

That's it! 🎯
